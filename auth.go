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
	"net/url"
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

	mu          sync.Mutex
	recentFails []time.Time
	c           creds
	secret      []byte
	fails       map[string]*failState
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

// setUser renames the account without touching the password hash.
func (a *Auth) setUser(user string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	c := a.c
	c.User = user
	b, _ := json.Marshal(c)
	if err := os.WriteFile(filepath.Join(a.dir, "credentials.json"), b, 0o600); err != nil {
		return err
	}
	a.c = c
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

// isLocalClient reports whether the request clearly comes from this machine or
// a private network (LAN, Docker, Tailscale). Anything that went through a
// proxy is judged by the forwarded client address, and forwarding headers we are
// not configured to trust make the client count as remote: a spoofed header can
// only make a request look *less* local, never more.
func (a *Auth) isLocalClient(r *http.Request) bool {
	forwarded := r.Header.Get("X-Forwarded-For") != "" || r.Header.Get("Forwarded") != "" ||
		r.Header.Get("CF-Connecting-IP") != "" || r.Header.Get("X-Real-IP") != ""
	if forwarded && !a.trustProxy {
		return false
	}
	return isPrivateIP(net.ParseIP(a.clientIP(r)))
}

var cgnat = &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)} // Tailscale

func isPrivateIP(ip net.IP) bool {
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || cgnat.Contains(ip))
}

// globalFailures counts failed logins from everyone, to slow down distributed
// guessing that rotates IPs.
func (a *Auth) noteGlobalFailure() (delay time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	kept := a.recentFails[:0]
	for _, t := range a.recentFails {
		if now.Sub(t) < 10*time.Minute {
			kept = append(kept, t)
		}
	}
	a.recentFails = append(kept, now)
	if n := len(a.recentFails); n > 20 {
		return 3 * time.Second
	}
	return 400 * time.Millisecond
}

var weakPasswords = map[string]bool{"password": true, "12345678": true, "123456789": true, "1234567890": true,
	"qwertyuiop": true, "adminadmin": true, "metroweb": true, "password1": true, "letmein123": true}

func checkNewPassword(user, pass string) string {
	switch {
	case len(pass) < 10:
		return "The new password must be at least 10 characters."
	case len(pass) > 200:
		return "The new password is too long."
	case strings.EqualFold(pass, user) || strings.EqualFold(pass, "admin") || weakPasswords[strings.ToLower(pass)]:
		return "That password is too easy to guess. Pick something harder."
	}
	return ""
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
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "Too many attempts. Try again in a few minutes."})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		if err := r.ParseForm(); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request."})
			return
		}
		if a.snapshot().Default && !a.isLocalClient(r) {
			// The password is still admin/admin: never accept it from the internet.
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "This server still uses the default password. Sign in from your home network (or localhost) and change it first."})
			return
		}
		if !a.check(r.PostForm.Get("user"), r.PostForm.Get("password")) {
			a.fail(ip)
			time.Sleep(a.noteGlobalFailure())
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Incorrect username or password."})
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
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "Too many attempts. Try again in a few minutes."})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		var in struct{ Current, User, Password string }
		if json.NewDecoder(r.Body).Decode(&in) != nil {
			writeJSON(w, 400, map[string]string{"error": "Invalid request."})
			return
		}
		if !a.check(a.snapshot().User, in.Current) {
			a.fail(ip)
			time.Sleep(a.noteGlobalFailure())
			writeJSON(w, 401, map[string]string{"error": "Current password is incorrect."})
			return
		}
		user := strings.TrimSpace(in.User)
		if user == "" {
			user = a.snapshot().User
		}
		if len(user) > 64 {
			writeJSON(w, 400, map[string]string{"error": "The username must be at most 64 characters."})
			return
		}
		if in.Password == "" {
			// Username-only change: keep the current password (and sessions).
			if err := a.setUser(user); err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"user": user, "defaultCreds": a.snapshot().Default})
			return
		}
		if msg := checkNewPassword(user, in.Password); msg != "" {
			writeJSON(w, 400, map[string]string{"error": msg})
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
	return p == "/healthz" || p == "/manifest.webmanifest" || p == "/icon.svg" || p == "/login.css" || p == "/login.js"
}

func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setSecurityHeaders(w, r)
		if !a.sameOrigin(r) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "Cross-site request blocked."})
			return
		}
		switch {
		case r.URL.Path == "/login":
			a.handleLogin(w, r)
		case r.URL.Path == "/logout":
			a.handleLogout(w, r)
		case publicPath(r.URL.Path):
			next.ServeHTTP(w, r)
		case !a.valid(r):
			if strings.HasPrefix(r.URL.Path, "/api/") {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login required"})
			} else {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
			}
		case r.URL.Path == "/api/account":
			a.handleAccount(w, r)
		default:
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				w.Header().Set("Cache-Control", "no-cache") // always revalidate the app files
			}
			next.ServeHTTP(w, r)
		}
	})
}

// contentSecurityPolicy: only our own files; no inline scripts, no framing.
const contentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; media-src 'self' blob:; connect-src 'self'; manifest-src 'self'; " +
	"object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"

func setSecurityHeaders(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Content-Security-Policy", contentSecurityPolicy)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
	h.Set("Cross-Origin-Opener-Policy", "same-origin")
	h.Set("Cross-Origin-Resource-Policy", "same-origin")
	if isHTTPS(r) {
		h.Set("Strict-Transport-Security", "max-age=31536000")
	}
}

// sameOrigin rejects state-changing requests that a browser marks (or whose
// Origin shows) as coming from another site. Requests without either header
// (curl, scripts) pass; they still need a valid session.
func (a *Auth) sameOrigin(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	if s := r.Header.Get("Sec-Fetch-Site"); s != "" && s != "same-origin" && s != "none" {
		return false
	}
	if o := r.Header.Get("Origin"); o != "" {
		u, err := url.Parse(o)
		if err != nil {
			return false
		}
		host := r.Host
		if a.trustProxy {
			if xh := r.Header.Get("X-Forwarded-Host"); xh != "" {
				host = strings.TrimSpace(strings.Split(xh, ",")[0])
			}
		}
		return strings.EqualFold(u.Host, host)
	}
	return true
}
