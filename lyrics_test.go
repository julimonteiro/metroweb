package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseLRC(t *testing.T) {
	got := parseLRC("[ar:Daft Punk]\n[00:12.50] Around the world\r\n[00:20.0]\n[01:05.123][02:00.5] Chorus line\n[length: 5:21]\nnot a lyric line")
	want := []LyricLine{{12.5, "Around the world"}, {20, ""}, {65.123, "Chorus line"}, {120.5, "Chorus line"}}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i].Text != want[i].Text || got[i].T < want[i].T-0.001 || got[i].T > want[i].T+0.001 {
			t.Fatalf("line %d: got %+v want %+v", i, got[i], want[i])
		}
	}
}

func TestPickLyricsPrefersSyncedAndMatchingDuration(t *testing.T) {
	list := []lrclibTrack{
		{ArtistName: "Someone Else", Duration: 321, SyncedLyrics: "[00:01.00] x"},
		{ArtistName: "Daft Punk", Duration: 400, SyncedLyrics: "[00:01.00] live version"},   // far duration
		{ArtistName: "Daft Punk", Duration: 320, PlainLyrics: "plain only"},                 // plain
		{ArtistName: "Daft Punk", Duration: 322, SyncedLyrics: "[00:01.00] studio version"}, // best
	}
	if got := pickLyrics(list, "Daft Punk", 321); got == nil || got.SyncedLyrics != "[00:01.00] studio version" {
		t.Fatalf("picked %+v", got)
	}
	if pickLyrics(list, "Nobody", 321) != nil {
		t.Fatal("a different artist must never match")
	}
}

func TestLyricsServiceLookupAndCache(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch r.URL.Path {
		case "/api/get":
			http.NotFound(w, r) // exact lookup misses -> falls back to search
		case "/api/search":
			if r.URL.Query().Get("track_name") == "One More Time" {
				json.NewEncoder(w).Encode([]lrclibTrack{{TrackName: "One More Time", ArtistName: "Daft Punk", Duration: 321,
					SyncedLyrics: "[00:10.00] One more time\n[00:14.20] We're gonna celebrate"}})
				return
			}
			json.NewEncoder(w).Encode([]lrclibTrack{})
		}
	}))
	defer srv.Close()
	s := NewLyricsService(true)
	s.base = srv.URL
	res, err := s.Get(context.Background(), "One More Time (Radio Edit)", "Daft Punk, Romanthony", "Discovery", 321)
	if err != nil || !res.Found || len(res.Synced) != 2 || res.Synced[1].T != 14.2 {
		t.Fatalf("lookup: %+v %v", res, err)
	}
	before := calls
	if _, err := s.Get(context.Background(), "One More Time (Radio Edit)", "Daft Punk", "Discovery", 321); err != nil || calls != before {
		t.Fatalf("second lookup must come from the cache (calls %d -> %d)", before, calls)
	}
	if miss, _ := s.Get(context.Background(), "Unknown Song", "Daft Punk", "", 200); miss.Found {
		t.Fatal("nothing should be found for an unknown song")
	}
}

func TestLyricsDisabled(t *testing.T) {
	if NewLyricsService(false) != nil {
		t.Fatal("disabled service must be nil")
	}
	var s *LyricsService
	if r, err := s.Get(context.Background(), "a", "b", "", 1); err != nil || r.Found {
		t.Fatal("nil service must answer 'nothing found'")
	}
}

func TestClipAllowList(t *testing.T) {
	if clipAllowed("abcdefghijk") {
		t.Fatal("unknown clip must not be allowed")
	}
	allowClip("abcdefghijk")
	if !clipAllowed("abcdefghijk") {
		t.Fatal("vouched clip must be allowed")
	}
}
