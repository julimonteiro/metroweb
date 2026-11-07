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
