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
