package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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
