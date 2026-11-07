package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newTestAuth(t *testing.T) *Auth {
	a, err := NewAuth(t.TempDir(), "", "", false, []byte("login"))
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestAuthFlow(t *testing.T) {
	a := newTestAuth(t)
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("secret")) })
	h := a.Middleware(ok)

	// anonymous: redirect for pages, 401 for the API
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("page without session: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/search?q=a", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("api without session: %d", rec.Code)
	}

	login := func(user, pass string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/login", strings.NewReader(url.Values{"user": {user}, "password": {pass}}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.RemoteAddr = "127.0.0.1:4000"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	if r := login("admin", "wrong"); r.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password: %d", r.Code)
	}
	r := login("admin", "admin")
	if r.Code != 200 || len(r.Result().Cookies()) == 0 {
		t.Fatalf("default login failed: %d", r.Code)
	}
	cookie := r.Result().Cookies()[0]
	if !cookie.HttpOnly {
		t.Fatal("session cookie must be HttpOnly")
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Body.String() != "secret" {
		t.Fatalf("valid session rejected: %d", rec.Code)
	}

	// changing the password invalidates the old session
	if err := a.setCreds("julio", "a-better-password"); err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("old session must be invalid after password change: %d", rec.Code)
	}
	if r := login("admin", "admin"); r.Code != http.StatusUnauthorized {
		t.Fatal("old credentials must stop working")
	}
	if r := login("julio", "a-better-password"); r.Code != 200 {
		t.Fatalf("new credentials failed: %d", r.Code)
	}
}

func TestAuthLockout(t *testing.T) {
	a := newTestAuth(t)
	for i := 0; i < maxFailures; i++ {
		a.fail("1.2.3.4")
	}
	if !a.locked("1.2.3.4") || a.locked("5.6.7.8") {
		t.Fatal("lockout should apply per IP after repeated failures")
	}
}

func TestRenameKeepsPasswordAndSession(t *testing.T) {
	a := newTestAuth(t)
	if err := a.setUser("julio"); err != nil {
		t.Fatal(err)
	}
	if !a.check("julio", "admin") || a.check("admin", "admin") {
		t.Fatal("rename must change the username and keep the password")
	}
	if !a.snapshot().Default {
		t.Fatal("password is still the default, so the warning must stay")
	}
	b, err := NewAuth(a.dir, "", "", false, nil) // reload from disk
	if err != nil || !b.check("julio", "admin") {
		t.Fatalf("rename not persisted: %v", err)
	}
}

func postLogin(h http.Handler, headers map[string]string) int {
	req := httptest.NewRequest("POST", "/login", strings.NewReader("user=admin&password=admin"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = "192.168.1.20:5555"
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestDefaultPasswordNeverAcceptedFromRemote(t *testing.T) {
	a := newTestAuth(t)
	h := a.Middleware(http.NotFoundHandler())
	if c := postLogin(h, nil); c != 200 {
		t.Fatalf("LAN client with default password should work: %d", c)
	}
	// Forwarding headers we don't trust make the client look remote, never local.
	for _, hdr := range []map[string]string{
		{"X-Forwarded-For": "8.8.8.8"}, {"X-Forwarded-For": "192.168.1.5"}, {"CF-Connecting-IP": "1.2.3.4"},
	} {
		if c := postLogin(h, hdr); c != http.StatusForbidden {
			t.Fatalf("default password via %v must be refused, got %d", hdr, c)
		}
	}
	// Behind a trusted proxy the forwarded client address decides.
	a.trustProxy = true
	if c := postLogin(h, map[string]string{"X-Forwarded-For": "8.8.8.8"}); c != http.StatusForbidden {
		t.Fatalf("public client behind trusted proxy must be refused: %d", c)
	}
	if c := postLogin(h, map[string]string{"X-Forwarded-For": "100.64.1.2"}); c != 200 {
		t.Fatalf("Tailscale client behind trusted proxy should work: %d", c)
	}
	// Once the password is changed, remote logins are allowed again.
	a.setCreds("julio", "a-better-password")
	req := httptest.NewRequest("POST", "/login", strings.NewReader("user=julio&password=a-better-password"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Forwarded-For", "8.8.8.8")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("remote login with a real password: %d", rec.Code)
	}
}

func TestCrossSiteWritesBlocked(t *testing.T) {
	a := newTestAuth(t)
	h := a.Middleware(http.NotFoundHandler())
	for name, hdr := range map[string]map[string]string{
		"foreign origin":   {"Origin": "https://evil.example"},
		"cross-site fetch": {"Sec-Fetch-Site": "cross-site"},
	} {
		req := httptest.NewRequest("POST", "/api/playlists", nil)
		req.Host = "music.example.com"
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s: want 403, got %d", name, rec.Code)
		}
	}
	req := httptest.NewRequest("POST", "/login", strings.NewReader("user=x&password=y"))
	req.Host = "music.example.com"
	req.Header.Set("Origin", "https://music.example.com")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = "127.0.0.1:4000"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized { // reached the login handler (wrong password), not blocked as cross-site
		t.Fatalf("same-origin request must reach the handler, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Security-Policy") == "" || rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatal("security headers missing")
	}
}

func TestPasswordPolicy(t *testing.T) {
	for _, bad := range []string{"short", "admin", "adminadmin", "password", "julio-julio"[:0] + "12345678"} {
		if checkNewPassword("julio", bad) == "" {
			t.Fatalf("%q should be rejected", bad)
		}
	}
	if checkNewPassword("julio", "JULIO") == "" || checkNewPassword("a-very-long-name", "a-very-long-name") == "" {
		t.Fatal("password equal to username must be rejected")
	}
	if checkNewPassword("julio", "correct horse battery") != "" {
		t.Fatal("a long passphrase should be accepted")
	}
}
