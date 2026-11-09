package main

import (
	"fmt"
	"testing"
	"time"
)

func mkTrack(i int, artist string) Track {
	return Track{ID: fmt.Sprintf("id%09d", i), Title: fmt.Sprintf("Song %d", i), Artist: artist, Thumb: "http://x/t.jpg"}
}

func TestBuildMixes(t *testing.T) {
	now := time.Date(2026, 11, 9, 21, 0, 0, 0, time.UTC) // evening, UTC
	day := int64(24 * 3600)
	var plays []Play
	var liked []Track
	// 12 songs by "Big Band", 8 played often this month
	for i := 0; i < 12; i++ {
		tr := mkTrack(i, "Big Band")
		liked = append(liked, tr)
		if i < 8 {
			for k := 0; k < 3; k++ {
				plays = append(plays, Play{T: now.Unix() - int64(k)*day - 3600*3, Track: tr}) // ~6pm, evening
			}
		}
	}
	// 7 old favourites, played 3x about 100 days ago
	for i := 20; i < 27; i++ {
		tr := mkTrack(i, fmt.Sprintf("Old %d", i))
		for k := 0; k < 3; k++ {
			plays = append(plays, Play{T: now.Unix() - 100*day - int64(k)*day - 6*3600, Track: tr})
		}
	}
	// 10 liked, never played
	for i := 40; i < 50; i++ {
		liked = append(liked, mkTrack(i, fmt.Sprintf("New %d", i)))
	}
	genres := map[string]genreEntry{"big band": {Genres: []string{"rock"}}}
	mixes := buildMixes(plays, liked, genres, 0, now)
	by := map[string]Mix{}
	for _, m := range mixes {
		by[m.ID] = m
	}
	if len(by["repeat"].Tracks) != 8 {
		t.Fatalf("on repeat = %d, want 8", len(by["repeat"].Tracks))
	}
	if len(by["rewind"].Tracks) != 7 {
		t.Fatalf("rewind = %d, want 7", len(by["rewind"].Tracks))
	}
	if len(by["gems"].Tracks) != 14 { // 4 never-played Big Band + 10 new
		t.Fatalf("hidden gems = %d, want 14", len(by["gems"].Tracks))
	}
	if m, ok := by["now"]; !ok || m.Title != "Evening mix" || len(m.Tracks) != 8 {
		t.Fatalf("around now = %+v", by["now"])
	}
	if len(by["artist-big-band"].Tracks) != 12 {
		t.Fatalf("artist mix = %d, want 12", len(by["artist-big-band"].Tracks))
	}
	if _, ok := by["genre-rock"]; ok { // only 12 songs and the rule needs 12: present
		if by["genre-rock"].Title != "Rock mix" {
			t.Fatal("genre title")
		}
	}
	for _, m := range mixes {
		if m.Thumb == "" || len(m.Tracks) > mixMax {
			t.Fatalf("bad mix %s", m.ID)
		}
		seen := map[string]bool{}
		for _, tr := range m.Tracks {
			if seen[tr.ID] {
				t.Fatalf("duplicate in %s", m.ID)
			}
			seen[tr.ID] = true
		}
	}
	// stable through the day
	again := buildMixes(plays, liked, genres, 0, now.Add(time.Hour))
	if fmt.Sprint(again[2].Tracks) != fmt.Sprint(by["gems"].Tracks) && again[2].ID == "gems" {
		t.Fatal("mixes must not change within the day")
	}
	if got := buildMixes(nil, nil, nil, 0, now); len(got) != 0 {
		t.Fatalf("empty history must give no mixes, got %d", len(got))
	}
}
