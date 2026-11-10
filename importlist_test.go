package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestParseTrackList(t *testing.T) {
	lines := ParseTrackList("# my list\nDaft Punk - One More Time\n\nRadiohead – Creep\nQueen;Don't Stop Me Now\nno separator here\n - missing artist\n")
	want := []ListEntry{{"One More Time", "Daft Punk"}, {"Creep", "Radiohead"}, {"Don't Stop Me Now", "Queen"}}
	if len(lines) != len(want) {
		t.Fatalf("got %v", lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("line %d: %v want %v", i, lines[i], want[i])
		}
	}
	csvText := "\xef\xbb\xbf\"Track URI\",\"Track Name\",\"Album Name\",\"Artist Name(s)\",\"Duration (ms)\"\n\"spotify:track:1\",\"Starboy\",\"Starboy\",\"The Weeknd;Daft Punk\",230000\n\"spotify:track:2\",\"Hello, World\",\"X\",\"Some Band\",1\n"
	es := ParseTrackList(csvText)
	if len(es) != 2 || es[0].Title != "Starboy" || es[0].Artist != "The Weeknd, Daft Punk" || es[1].Title != "Hello, World" {
		t.Fatalf("csv = %v", es)
	}
	if ParseTrackList("   ") != nil {
		t.Fatal("blank text has no songs")
	}
}

func TestListImportBuildsPlaylist(t *testing.T) {
	pls, err := NewPlaylistStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	li := NewListImporter(pls)
	li.search = func(ctx context.Context, q string) ([]Item, error) {
		if strings.Contains(q, "Creep") {
			return []Item{{Type: "song", Track: Track{ID: "abcdefghijk", Title: "Creep", Artist: "Radiohead"}}}, nil
		}
		if strings.Contains(q, "Nothing") {
			return []Item{{Type: "song", Track: Track{ID: "zzzzzzzzzzz", Title: "Something Else", Artist: "Other"}}}, nil
		}
		return nil, nil
	}
	id, err := li.Start("Imported", "Radiohead - Creep\nRadiohead - Creep\nNobody - Nothing Here")
	if err != nil {
		t.Fatal(err)
	}
	var st ImportListStatus
	for i := 0; i < 100; i++ {
		st, _ = li.Status(id)
		if !st.Running {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if st.Running || st.PlaylistID == "" || st.Error != "" {
		t.Fatalf("status = %+v", st)
	}
	if st.Total != 3 || st.Found != 2 || len(st.Missing) != 1 {
		t.Fatalf("counts = %+v", st)
	}
	p, ok := pls.Get(st.PlaylistID)
	if !ok || len(p.Tracks) != 1 || p.Tracks[0].ID != "abcdefghijk" {
		t.Fatalf("playlist = %+v %v (duplicates must collapse)", p, ok)
	}
	if _, err := li.Start("", "A - B"); err == nil {
		t.Fatal("name required")
	}
	if _, err := li.Start("x", "no songs"); err == nil {
		t.Fatal("empty list must be rejected")
	}
}
