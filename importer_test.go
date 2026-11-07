package main

import (
	"context"
	"testing"
	"time"
)

func song(id, title, artist, dur string) Item {
	return Item{Type: "song", Track: Track{ID: id, Title: title, Artist: artist, Duration: dur}}
}

func TestMatchSong(t *testing.T) {
	s := JfSong{Title: "One More Time", Artists: []string{"Daft Punk"}, Album: "Discovery", Seconds: 320}
	for _, tc := range []struct {
		name  string
		cands []Item
		want  string
	}{
		{"exact", []Item{song("aaaaaaaaaaa", "One More Time", "Daft Punk", "5:20")}, "aaaaaaaaaaa"},
		{"radio edit suffix is fine only if it is the best we have", []Item{song("bbbbbbbbbbb", "One More Time (Radio Edit)", "Daft Punk", "5:21")}, "bbbbbbbbbbb"},
		{"prefers the exact title", []Item{song("ccccccccccc", "One More Time (Radio Edit)", "Daft Punk", "5:20"), song("ddddddddddd", "One More Time", "Daft Punk", "5:21")}, "ddddddddddd"},
		{"other artist", []Item{song("eeeeeeeeeee", "One More Time", "Super Junior", "5:20")}, ""},
		{"different version by length", []Item{song("fffffffffff", "One More Time", "Daft Punk", "4:00")}, ""},
		{"live version is not the studio song", []Item{song("ggggggggggg", "One More Time (Live)", "Daft Punk", "5:20")}, ""},
		{"cards are ignored", []Item{{Type: "album", Track: Track{ID: "MPREb_x", Title: "One More Time"}}}, ""},
		{"feat. artists still match", []Item{song("hhhhhhhhhhh", "One More Time", "Daft Punk, Romanthony", "5:20")}, "hhhhhhhhhhh"},
	} {
		got := matchSong(s, tc.cands)
		switch {
		case tc.want == "" && got != nil:
			t.Errorf("%s: expected no match, got %s", tc.name, got.ID)
		case tc.want != "" && (got == nil || got.ID != tc.want):
			t.Errorf("%s: want %s, got %+v", tc.name, tc.want, got)
		}
	}
	// A live title in Jellyfin matches the live recording.
	live := JfSong{Title: "One More Time (Live)", Artists: []string{"Daft Punk"}, Seconds: 330}
	if m := matchSong(live, []Item{song("iiiiiiiiiii", "One More Time (Live)", "Daft Punk", "5:30")}); m == nil || m.ID != "iiiiiiiiiii" {
		t.Errorf("live song should match the live recording: %+v", m)
	}
	// accents and ampersands in names
	br := JfSong{Title: "Não Existe Amor em SP", Artists: []string{"Criolo"}, Seconds: 270}
	if m := matchSong(br, []Item{song("jjjjjjjjjjj", "Nao Existe Amor Em SP", "Criolo", "4:30")}); m == nil {
		t.Error("accent-insensitive match expected")
	}
}

func TestImporterAddsMatchesToLiked(t *testing.T) {
	jf := NewJellyfinIndex("http://x", "k")
	jf.songs = []JfSong{
		{Title: "One More Time", Artists: []string{"Daft Punk"}, Seconds: 320},
		{Title: "Starboy", Artists: []string{"The Weeknd"}, Seconds: 230},
		{Title: "Obscure Local Song", Artists: []string{"Nobody"}, Seconds: 100},
		{Title: "Already Liked", Artists: []string{"Someone"}, Seconds: 200},
	}
	pls, _ := NewPlaylistStore(t.TempDir())
	pls.AddTracks(likedID, []Track{{ID: "zzzzzzzzzzz", Title: "Already Liked"}})
	dir := t.TempDir()
	im := NewImporter(jf, pls, dir)
	im.search = func(ctx context.Context, q string) ([]Item, error) {
		switch {
		case contains(q, "One More Time"):
			return []Item{song("aaaaaaaaaaa", "One More Time", "Daft Punk", "5:20")}, nil
		case contains(q, "Starboy"):
			return []Item{song("bbbbbbbbbbb", "Starboy", "The Weeknd", "3:50")}, nil
		case contains(q, "Already Liked"):
			return []Item{song("zzzzzzzzzzz", "Already Liked", "Someone", "3:20")}, nil
		}
		return nil, nil
	}
	if err := im.Start(); err != nil {
		t.Fatal(err)
	}
	if err := im.Start(); err == nil {
		t.Fatal("a second import must be refused while one is running")
	}
	for i := 0; i < 200 && im.Status().Running; i++ {
		time.Sleep(50 * time.Millisecond)
	}
	st := im.Status()
	if st.Running || st.Total != 4 || st.Done != 4 || st.Added != 2 || st.Already != 1 || st.Unmatched != 1 {
		t.Fatalf("status: %+v", st)
	}
	if len(st.Missing) != 1 || st.Missing[0] != "Nobody - Obscure Local Song" {
		t.Fatalf("missing: %v", st.Missing)
	}
	liked, _ := pls.Get(likedID)
	if len(liked.Tracks) != 3 {
		t.Fatalf("liked has %d tracks, want 3", len(liked.Tracks))
	}
}

func TestImporterNeedsJellyfin(t *testing.T) {
	pls, _ := NewPlaylistStore(t.TempDir())
	if err := NewImporter(nil, pls, t.TempDir()).Start(); err == nil {
		t.Fatal("import must fail without Jellyfin")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool { return indexOf(s, sub) >= 0 })()
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestImporterIsIncrementalAndRespectsUnlikes(t *testing.T) {
	jf := NewJellyfinIndex("http://x", "k")
	jf.songs = []JfSong{
		{Title: "One More Time", Artists: []string{"Daft Punk"}, Seconds: 320},
		{Title: "Obscure Local Song", Artists: []string{"Nobody"}, Seconds: 100},
	}
	pls, _ := NewPlaylistStore(t.TempDir())
	dir := t.TempDir()
	searches := 0
	newIm := func() *Importer {
		im := NewImporter(jf, pls, dir)
		im.search = func(ctx context.Context, q string) ([]Item, error) {
			searches++
			if contains(q, "One More Time") {
				return []Item{song("aaaaaaaaaaa", "One More Time", "Daft Punk", "5:20")}, nil
			}
			if contains(q, "Brand New") {
				return []Item{song("bbbbbbbbbbb", "Brand New Song", "Fresh Artist", "3:00")}, nil
			}
			return nil, nil
		}
		return im
	}
	wait := func(im *Importer) {
		for i := 0; i < 200 && im.Status().Running; i++ {
			time.Sleep(50 * time.Millisecond)
		}
	}
	im := newIm()
	if !im.Due() {
		t.Fatal("a never-run importer must be due")
	}
	im.Start()
	wait(im)
	if searches != 2 {
		t.Fatalf("first run searches = %d", searches)
	}
	if im.Due() {
		t.Fatal("must not be due again right after a run")
	}
	// the user un-likes the imported song
	pls.RemoveTrack(likedID, "aaaaaaaaaaa")

	// a later run (new process, same data dir) with one new song in Jellyfin
	jf.songs = append(jf.songs, JfSong{Title: "Brand New Song", Artists: []string{"Fresh Artist"}, Seconds: 180})
	searches = 0
	im2 := newIm()
	if im2.Due() {
		t.Fatal("state must survive a restart")
	}
	im2.lastRun = time.Now().Add(-8 * 24 * time.Hour).Unix() // a week passed
	if !im2.Due() {
		t.Fatal("must be due after a week")
	}
	im2.Start()
	wait(im2)
	st := im2.Status()
	if searches != 1 || st.Skipped != 2 || st.Added != 1 {
		t.Fatalf("incremental run: searches=%d status=%+v", searches, st)
	}
	liked, _ := pls.Get(likedID)
	for _, tr := range liked.Tracks {
		if tr.ID == "aaaaaaaaaaa" {
			t.Fatal("a song the user un-liked must not come back")
		}
	}
}
