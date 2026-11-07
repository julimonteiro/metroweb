package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

type jfItem struct {
	Name        string   `json:"Name"`
	Artists     []string `json:"Artists"`
	AlbumArtist string   `json:"AlbumArtist"`
}

func fakeJellyfin(t *testing.T, items []jfItem, wantToken string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != `MediaBrowser Token="`+wantToken+`"` {
			http.Error(w, "nope", http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/Items" || r.URL.Query().Get("IncludeItemTypes") != "Audio" {
			t.Errorf("unexpected request %s", r.URL)
		}
		start, _ := strconv.Atoi(r.URL.Query().Get("StartIndex"))
		limit := 2 // force pagination
		end := start + limit
		if end > len(items) {
			end = len(items)
		}
		json.NewEncoder(w).Encode(map[string]any{"Items": items[start:end], "TotalRecordCount": len(items)})
	}))
}

func TestJellyfinIndexMatching(t *testing.T) {
	srv := fakeJellyfin(t, []jfItem{
		{Name: "One More Time", Artists: []string{"Daft Punk"}, AlbumArtist: "Daft Punk"},
		{Name: "Não Existe Amor em SP", Artists: []string{"Criolo"}, AlbumArtist: "Criolo"},
		{Name: "Get Lucky", Artists: []string{"Daft Punk", "Pharrell Williams"}},
		{Name: "Intro", Artists: []string{"Someone Else"}},
		{Name: "Cold Heart", Artists: []string{"Elton John"}, AlbumArtist: "Elton John"},
	}, "k3y")
	defer srv.Close()

	j := NewJellyfinIndex(srv.URL+"/", "k3y")
	if err := j.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if st := j.Status(); !st.Enabled || st.Songs != 5 {
		t.Fatalf("status: %+v", st)
	}
	for _, c := range []struct {
		title, artist string
		want          bool
	}{
		{"One More Time", "Daft Punk", true},
		{"One More Time (Radio Edit)", "Daft Punk", true},      // bracketed edit suffix
		{"ONE MORE TIME - Remastered 2011", "daft punk", true}, // case + dash suffix
		{"Nao Existe Amor em SP", "Criolo", true},              // accents
		{"Get Lucky (feat. Pharrell Williams and Nile Rodgers)", "Daft Punk, Pharrell Williams, Nile Rodgers", true},
		{"Get Lucky", "Pharrell Williams", true}, // any listed artist
		{"Cold Heart (PNAU Remix)", "Elton John, Dua Lipa", true},
		{"Intro", "Daft Punk", false}, // same title, different artist
		{"Instant Crush", "Daft Punk", false},
		{"One More Time", "", false}, // no artist: never guess
		{"", "Daft Punk", false},
	} {
		if got := j.Has(c.title, c.artist); got != c.want {
			t.Errorf("Has(%q, %q) = %v, want %v", c.title, c.artist, got, c.want)
		}
	}
}

func TestJellyfinWrongKey(t *testing.T) {
	srv := fakeJellyfin(t, nil, "right")
	defer srv.Close()
	j := NewJellyfinIndex(srv.URL, "wrong")
	if err := j.Refresh(context.Background()); err == nil {
		t.Fatal("a rejected API key must surface as an error")
	}
}

func TestJellyfinDisabledWithoutConfig(t *testing.T) {
	var j *JellyfinIndex = NewJellyfinIndex("", "")
	if j != nil || j.Has("a", "b") || j.Status().Enabled {
		t.Fatal("no URL/key must mean disabled, and never claim ownership")
	}
}

func TestJellyfinFindAndStream(t *testing.T) {
	const fileBody = "0123456789abcdefghij"
	var gotAuth, gotRange string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/Items":
			json.NewEncoder(w).Encode(map[string]any{"TotalRecordCount": 2, "Items": []map[string]any{
				{"Id": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "Name": "One More Time", "Artists": []string{"Daft Punk"}},
				{"Id": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "Name": "Starboy", "Artists": []string{"The Weeknd", "Daft Punk"}},
			}})
		case r.URL.Path == "/Audio/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/stream" && r.URL.Query().Get("static") == "true":
			gotAuth, gotRange = r.Header.Get("Authorization"), r.Header.Get("Range")
			w.Header().Set("Content-Type", "audio/flac")
			w.Header().Set("Accept-Ranges", "bytes")
			if gotRange == "bytes=5-9" {
				w.Header().Set("Content-Range", "bytes 5-9/20")
				w.WriteHeader(http.StatusPartialContent)
				w.Write([]byte(fileBody[5:10]))
				return
			}
			w.Write([]byte(fileBody))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	j := NewJellyfinIndex(srv.URL, "secret-key")
	if err := j.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if id, ok := j.Find("One More Time (Radio Edit)", "Daft Punk, Romanthony"); !ok || id != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("Find: %q %v", id, ok)
	}
	if _, ok := j.Find("One More Time", "Someone Else"); ok {
		t.Fatal("a different artist must not match")
	}
	if _, ok := j.Find("", "Daft Punk"); ok {
		t.Fatal("empty title must not match")
	}

	// whole file
	rec := httptest.NewRecorder()
	if !j.Stream(rec, httptest.NewRequest("GET", "/x", nil), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa") || rec.Body.String() != fileBody || rec.Header().Get("Content-Type") != "audio/flac" {
		t.Fatalf("stream: %d %q", rec.Code, rec.Body.String())
	}
	if gotAuth != `MediaBrowser Token="secret-key"` {
		t.Fatalf("the API key must be sent to Jellyfin from the server: %q", gotAuth)
	}
	if rec.Header().Get("Authorization") != "" || strings.Contains(rec.Body.String(), "secret-key") {
		t.Fatal("the API key must never reach the client")
	}
	// Range (seek)
	rec = httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Range", "bytes=5-9")
	if !j.Stream(rec, req, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa") || rec.Code != 206 || rec.Body.String() != "56789" || rec.Header().Get("Content-Range") != "bytes 5-9/20" {
		t.Fatalf("range: %d %q %q", rec.Code, rec.Body.String(), rec.Header().Get("Content-Range"))
	}
	// missing item: nothing written so the caller can fall back to YouTube
	rec = httptest.NewRecorder()
	if j.Stream(rec, httptest.NewRequest("GET", "/x", nil), "cccccccccccccccccccccccccccccccc") || rec.Body.Len() != 0 {
		t.Fatal("an unavailable item must report false and write nothing")
	}

	// end to end through the stream handler: no YouTube involved
	streamJF = j
	defer func() { streamJF = nil }()
	rec = httptest.NewRecorder()
	handleStream(rec, httptest.NewRequest("GET", "/api/stream/wU26xVT_vBU?fmt=opus&t=One+More+Time+%28Radio+Edit%29&a=Daft+Punk", nil))
	if rec.Code != 200 || rec.Body.String() != fileBody {
		t.Fatalf("handleStream should play the Jellyfin file: %d %q", rec.Code, rec.Body.String())
	}
}
