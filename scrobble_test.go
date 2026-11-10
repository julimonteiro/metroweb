package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestListenBrainzSubmit(t *testing.T) {
	var got map[string]any
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		if r.URL.Path != "/1/submit-listens" || r.Method != "POST" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&got)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()
	s := NewScrobbler("tok123", nil, "", true)
	if s == nil {
		t.Fatal("scrobbler should exist with a token")
	}
	s.lbBase = srv.URL
	at := time.Unix(1_700_000_000, 0)
	s.handle(scrobbleJob{Track{ID: "abcdefghijk", Title: "One More Time", Artist: "Daft Punk, Romanthony", Album: "Discovery"}, at})
	if auth != "Token tok123" {
		t.Fatalf("auth = %q", auth)
	}
	p := got["payload"].([]any)[0].(map[string]any)
	md := p["track_metadata"].(map[string]any)
	if got["listen_type"] != "single" || p["listened_at"].(float64) != 1_700_000_000 || md["artist_name"] != "Daft Punk" || md["track_name"] != "One More Time" || md["release_name"] != "Discovery" {
		t.Fatalf("payload = %v", got)
	}
	if st := s.Status(); st.Sent != 1 || st.Failed != 0 || !st.ListenBrainz {
		t.Fatalf("status = %+v", st)
	}
}

func TestListenBrainzRejectedTokenIsNotRetried(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.Error(w, "invalid token", 401)
	}))
	defer srv.Close()
	s := NewScrobbler("bad", nil, "", true)
	s.lbBase = srv.URL
	s.handle(scrobbleJob{Track{ID: "abcdefghijk", Title: "T", Artist: "A"}, time.Now()})
	if calls != 1 {
		t.Fatalf("a 401 must not be retried, got %d calls", calls)
	}
	if st := s.Status(); st.Failed != 1 || st.Error == "" {
		t.Fatalf("status = %+v", st)
	}
}

func TestNoScrobblerWithoutConfig(t *testing.T) {
	if NewScrobbler("", nil, "", true) != nil {
		t.Fatal("nothing configured: no scrobbler")
	}
	var s *Scrobbler
	s.Play(Track{}, time.Now()) // must be safe on nil
	if s.Status().ListenBrainz {
		t.Fatal("nil status")
	}
}

func TestJellyfinMarkPlayed(t *testing.T) {
	var marked, key string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key = r.Header.Get("Authorization")
		switch {
		case r.URL.Path == "/Items":
			json.NewEncoder(w).Encode(map[string]any{"TotalRecordCount": 1, "Items": []map[string]any{
				{"Id": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "Name": "One More Time", "Artists": []string{"Daft Punk"}}}})
		case r.URL.Path == "/Users":
			json.NewEncoder(w).Encode([]map[string]any{
				{"Id": "u1", "Name": "off", "Policy": map[string]any{"IsDisabled": true}},
				{"Id": "u2", "Name": "ju", "Policy": map[string]any{"IsAdministrator": true}}})
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/Users/u2/PlayedItems/"):
			marked = strings.TrimPrefix(r.URL.Path, "/Users/u2/PlayedItems/")
			if r.URL.Query().Get("datePlayed") == "" {
				t.Error("datePlayed missing")
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	j := NewJellyfinIndex(srv.URL, "k")
	if err := j.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	ok, err := j.MarkPlayed(context.Background(), "One More Time", "Daft Punk", "", time.Now())
	if err != nil || !ok {
		t.Fatalf("mark: %v %v", ok, err)
	}
	if marked != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" || key != `MediaBrowser Token="k"` {
		t.Fatalf("marked %q with %q", marked, key)
	}
	if ok, err := j.MarkPlayed(context.Background(), "Not In Jellyfin", "Nobody", "", time.Now()); ok || err != nil {
		t.Fatalf("a song you do not own is skipped silently: %v %v", ok, err)
	}
}
