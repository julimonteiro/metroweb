package main

import (
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	cookieName  = "mw_session"
	sessionTTL  = 30 * 24 * time.Hour
	maxFailures = 5
	lockoutFor  = 5 * time.Minute
	pbkdfIters  = 150000
)

// Auth is a single-user login. Credentials live in <data>/credentials.json
// (PBKDF2-hashed) and can be changed from the web UI. Sessions are signed,
// expiring cookies whose key depends on the password hash, so changing the
// password logs every other device out.
type Auth struct {
	dir        string
	trustProxy bool
	loginPage  []byte

	mu     sync.Mutex
	c      creds
	secret []byte
	fails  map[string]*failState
}

type creds struct {
	User    string `json:"user"`
	Salt    string `json:"salt"`
	Hash    string `json:"hash"`
	Default bool   `json:"default"` // still admin/admin
}

type failState struct {
	n     int
	until time.Time
}

func NewAuth(dataDir, seedUser, seedPass string, trustProxy bool, loginPage []byte) (*Auth, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	a := &Auth{dir: dataDir, trustProxy: trustProxy, loginPage: loginPage, fails: map[string]*failState{}}
	var err error
	if a.secret, err = loadSecret(dataDir); err != nil {
		return nil, err
	}
	if b, err := os.ReadFile(filepath.Join(dataDir, "credentials.json")); err == nil {
		if json.Unmarshal(b, &a.c) == nil && a.c.Hash != "" {
			return a, nil
		}
	}
	if seedUser == "" {
		seedUser = "admin"
	}
	if seedPass == "" {
		seedPass = "admin"
	}
	if err := a.setCreds(seedUser, seedPass); err != nil {
		return nil, err
	}
	return a, nil
}

func hashPass(pass string, salt []byte) (string, error) {
	k, err := pbkdf2.Key(sha256.New, pass, salt, pbkdfIters, 32)
	return hex.EncodeToString(k), err
}

// setCreds must be called with a.mu unlocked (or during construction).
func (a *Auth) setCreds(user, pass string) error {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	h, err := hashPass(pass, salt)
	if err != nil {
		return err
	}
	c := creds{User: user, Salt: hex.EncodeToString(salt), Hash: h, Default: user == "admin" && pass == "admin"}
	b, _ := json.Marshal(c)
	if err := os.WriteFile(filepath.Join(a.dir, "credentials.json"), b, 0o600); err != nil {
		return err
	}
	a.mu.Lock()
	a.c = c
	a.mu.Unlock()
	return nil
}

func loadSecret(dir string) ([]byte, error) {
	path := filepath.Join(dir, "session-secret")
	if b, err := os.ReadFile(path); err == nil {
		if raw, err := hex.DecodeString(strings.TrimSpace(string(b))); err == nil && len(raw) >= 32 {
			return raw, nil
		}
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	return raw, os.WriteFile(path, []byte(hex.EncodeToString(raw)), 0o600)
}

func (a *Auth) snapshot() creds {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.c
}

func (a *Auth) sign(expiry int64) string {
	c := a.snapshot()
	msg := strconv.FormatInt(expiry, 10)
	m := hmac.New(sha256.New, append(append([]byte{}, a.secret...), c.Hash...))
	m.Write([]byte(msg))
	return msg + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func (a *Auth) valid(r *http.Request) bool {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return false
	}
	exp, _, ok := strings.Cut(c.Value, ".")
	n, err := strconv.ParseInt(exp, 10, 64)
	if !ok || err != nil || time.Now().Unix() > n {
		return false
	}
	return hmac.Equal([]byte(c.Value), []byte(a.sign(n)))
}

func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func (a *Auth) issue(w http.ResponseWriter, r *http.Request) {
	exp := time.Now().Add(sessionTTL)
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: a.sign(exp.Unix()), Path: "/", Expires: exp,
		HttpOnly: true, Secure: isHTTPS(r), SameSite: http.SameSiteLaxMode,
	})
}

func (a *Auth) clientIP(r *http.Request) string {
	if a.trustProxy {
		if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
			return strings.TrimSpace(strings.Split(xf, ",")[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (a *Auth) locked(ip string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	f := a.fails[ip]
	return f != nil && time.Now().Before(f.until)
}

func (a *Auth) fail(ip string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	f := a.fails[ip]
	if f == nil {
		f = &failState{}
		a.fails[ip] = f
	}
	f.n++
	if f.n >= maxFailures {
		f.n, f.until = 0, time.Now().Add(lockoutFor)
	}
}

// check compares user and password against the stored credentials without
// leaking which of the two was wrong.
func (a *Auth) check(user, pass string) bool {
	c := a.snapshot()
	salt, _ := hex.DecodeString(c.Salt)
	h, err := hashPass(pass, salt)
	if err != nil {
		return false
	}
	okH := subtle.ConstantTimeCompare([]byte(h), []byte(c.Hash))
	okU := subtle.ConstantTimeCompare([]byte(user), []byte(c.User))
	return okH&okU == 1
}

func (a *Auth) handleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if a.valid(r) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(a.loginPage)
	case http.MethodPost:
		ip := a.clientIP(r)
		if a.locked(ip) {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "Muitas tentativas. Tente de novo em alguns minutos."})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		if err := r.ParseForm(); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Requisição inválida."})
			return
		}
		if !a.check(r.PostForm.Get("user"), r.PostForm.Get("password")) {
			a.fail(ip)
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Usuário ou senha incorretos."})
			return
		}
		a.issue(w, r)
		writeJSON(w, http.StatusOK, map[string]string{"ok": "1"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *Auth) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: isHTTPS(r), SameSite: http.SameSiteLaxMode})
	writeJSON(w, http.StatusOK, map[string]string{"ok": "1"})
}

// handleAccount: GET reports the user and whether the default password is
// still in use; POST changes user/password (needs the current password).
func (a *Auth) handleAccount(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		c := a.snapshot()
		writeJSON(w, 200, map[string]any{"user": c.User, "defaultCreds": c.Default})
	case http.MethodPost:
		ip := a.clientIP(r)
		if a.locked(ip) {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "Muitas tentativas. Tente de novo em alguns minutos."})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		var in struct{ Current, User, Password string }
		if json.NewDecoder(r.Body).Decode(&in) != nil {
			writeJSON(w, 400, map[string]string{"error": "Requisição inválida."})
			return
		}
		if !a.check(a.snapshot().User, in.Current) {
			a.fail(ip)
			writeJSON(w, 401, map[string]string{"error": "Senha atual incorreta."})
			return
		}
		user := strings.TrimSpace(in.User)
		if user == "" {
			user = a.snapshot().User
		}
		if len(in.Password) < 8 {
			writeJSON(w, 400, map[string]string{"error": "A nova senha precisa ter pelo menos 8 caracteres."})
			return
		}
		if err := a.setCreds(user, in.Password); err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		a.issue(w, r) // the old cookie is now invalid everywhere; keep this device logged in
		writeJSON(w, 200, map[string]any{"user": user, "defaultCreds": false})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// publicPath: the login page itself and the PWA files (browsers fetch the
// manifest without credentials).
func publicPath(p string) bool {
	return p == "/manifest.webmanifest" || p == "/icon.svg"
}

func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		switch {
		case r.URL.Path == "/login":
			a.handleLogin(w, r)
		case r.URL.Path == "/logout":
			a.handleLogout(w, r)
		case publicPath(r.URL.Path):
			next.ServeHTTP(w, r)
		case !a.valid(r):
			if strings.HasPrefix(r.URL.Path, "/api/") {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login necessário"})
			} else {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
			}
		case r.URL.Path == "/api/account":
			a.handleAccount(w, r)
		default:
			next.ServeHTTP(w, r)
		}
	})
}
