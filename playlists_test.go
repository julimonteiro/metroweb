package main

import "testing"

func TestPlaylistStore(t *testing.T) {
	dir := t.TempDir()
	s, err := NewPlaylistStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if l := s.List(); len(l) != 1 || l[0].ID != likedID {
		t.Fatalf("expected only Liked songs, got %+v", l)
	}
	p, err := s.Create("  Road trip ")
	if err != nil || p.Name != "Road trip" {
		t.Fatalf("create: %+v %v", p, err)
	}
	tr := Track{ID: "abcdefghijk", Title: "Song"}
	if n, _ := s.AddTracks(p.ID, []Track{tr, tr, {ID: "", Title: "x"}}); n != 1 {
		t.Fatalf("expected 1 added (dedupe + invalid skipped), got %d", n)
	}
	// persisted across reloads
	s2, _ := NewPlaylistStore(dir)
	got, ok := s2.Get(p.ID)
	if !ok || len(got.Tracks) != 1 {
		t.Fatalf("reload: %+v %v", got, ok)
	}
	if err := s2.RemoveTrack(p.ID, tr.ID); err != nil {
		t.Fatal(err)
	}
	if err := s2.Rename(likedID, "x"); err == nil {
		t.Fatal("Liked songs must not be renamable")
	}
	if err := s2.Delete(likedID); err == nil {
		t.Fatal("Liked songs must not be deletable")
	}
	if err := s2.Delete(p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s2.Create(""); err == nil {
		t.Fatal("empty name must fail")
	}
}

func TestSanitizeTrackRejectsHostileInput(t *testing.T) {
	for _, bad := range []Track{
		{ID: "../../etc/passwd", Title: "x"}, {ID: "abc;rm -rf", Title: "x"}, {ID: "short", Title: "x"},
		{ID: "abcdefghijk", Title: "   "},
	} {
		if _, ok := sanitizeTrack(bad); ok {
			t.Fatalf("%+v must be rejected", bad)
		}
	}
	got, ok := sanitizeTrack(Track{ID: "abcdefghijk", Title: "T", Thumb: "javascript:alert(1)", ArtistID: "../x", AlbumID: "MPREb_ok"})
	if !ok || got.Thumb != "" || got.ArtistID != "" || got.AlbumID != "MPREb_ok" {
		t.Fatalf("unsafe fields must be dropped: %+v", got)
	}
	long := make([]byte, 5000)
	for i := range long {
		long[i] = 'a'
	}
	got, _ = sanitizeTrack(Track{ID: "abcdefghijk", Title: string(long)})
	if len([]rune(got.Title)) != 300 {
		t.Fatalf("title must be clipped, got %d", len(got.Title))
	}
}

func TestAutoLikeRespectsUnlikes(t *testing.T) {
	s, _ := NewPlaylistStore(t.TempDir())
	a := Track{ID: "aaaaaaaaaaa", Title: "A", Artist: "X"}
	b := Track{ID: "bbbbbbbbbbb", Title: "B", Artist: "X"}
	if got := s.AutoLike([]Track{a, b, {ID: "bad", Title: "no"}}); len(got) != 2 {
		t.Fatalf("auto-liked %v", got)
	}
	if got := s.AutoLike([]Track{a}); len(got) != 0 {
		t.Fatalf("already liked songs must not be added twice: %v", got)
	}
	// the user un-likes A: it must stay un-liked, even across a restart
	s.RemoveTrack(likedID, a.ID)
	s.Suppress(a.ID)
	if got := s.AutoLike([]Track{a}); len(got) != 0 {
		t.Fatalf("an un-liked song came back: %v", got)
	}
	s2, _ := NewPlaylistStore(s.path[:len(s.path)-len("playlists.json")])
	if !s2.IsSuppressed(a.ID) || len(s2.AutoLike([]Track{a})) != 0 {
		t.Fatal("suppression must survive a restart")
	}
	// liking it again by hand clears the suppression
	if n, _ := s2.AddTracks(likedID, []Track{a}); n != 1 || s2.IsSuppressed(a.ID) {
		t.Fatal("a deliberate like must override the earlier un-like")
	}
}
