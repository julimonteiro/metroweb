package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const likedID = "liked"

type Playlist struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Tracks  []Track   `json:"tracks"`
	Ranked  bool      `json:"ranked,omitempty"` // order matters (a generated chart): the cover is the first song
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

// PlaylistSummary is the light version used in lists.
type PlaylistSummary struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
	Thumb string `json:"thumb,omitempty"`
}

// PlaylistStore keeps playlists (and the special "Liked songs" one) in a JSON
// file inside the data directory so they follow you across devices.
type PlaylistStore struct {
	mu         sync.Mutex
	path       string
	lists      []*Playlist
	supPath    string
	suppressed map[string]bool // songs the user un-liked: automatic liking must not bring them back
}

func NewPlaylistStore(dir string) (*PlaylistStore, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	s := &PlaylistStore{path: filepath.Join(dir, "playlists.json"), supPath: filepath.Join(dir, "liked-suppressed.json"), suppressed: map[string]bool{}}
	if b, err := os.ReadFile(s.supPath); err == nil {
		var ids []string
		if json.Unmarshal(b, &ids) == nil {
			for _, id := range ids {
				s.suppressed[id] = true
			}
		}
	}
	if b, err := os.ReadFile(s.path); err == nil {
		if err := json.Unmarshal(b, &s.lists); err != nil {
			return nil, err
		}
	}
	if s.find(likedID) == nil {
		now := time.Now()
		s.lists = append([]*Playlist{{ID: likedID, Name: "Liked songs", Tracks: []Track{}, Created: now, Updated: now}}, s.lists...)
	}
	return s, nil
}

var (
	videoIDRe   = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
	browseIDRe2 = regexp.MustCompile(`^[A-Za-z0-9_-]{4,80}$`)
)

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

// sanitizeTrack accepts only well-formed tracks from clients: a real video id,
// bounded text fields, and https thumbnails.
func sanitizeTrack(t Track) (Track, bool) {
	if !videoIDRe.MatchString(t.ID) || strings.TrimSpace(t.Title) == "" {
		return Track{}, false
	}
	out := Track{ID: t.ID, Title: clip(t.Title, 300), Artist: clip(t.Artist, 300), Album: clip(t.Album, 300), Duration: clip(t.Duration, 12)}
	if browseIDRe2.MatchString(t.ArtistID) {
		out.ArtistID = t.ArtistID
	}
	if browseIDRe2.MatchString(t.AlbumID) {
		out.AlbumID = t.AlbumID
	}
	if strings.HasPrefix(t.Thumb, "https://") && len(t.Thumb) <= 500 {
		out.Thumb = t.Thumb
	}
	return out, true
}

func (s *PlaylistStore) find(id string) *Playlist {
	for _, p := range s.lists {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (s *PlaylistStore) saveSuppressedLocked() {
	ids := make([]string, 0, len(s.suppressed))
	for id := range s.suppressed {
		ids = append(ids, id)
	}
	b, _ := json.Marshal(ids)
	os.WriteFile(s.supPath, b, 0o600)
}

// Suppress remembers that the user un-liked a song, so it is never liked again
// automatically (liking it by hand clears this).
func (s *PlaylistStore) Suppress(id string) {
	if !videoIDRe.MatchString(id) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.suppressed[id] {
		s.suppressed[id] = true
		s.saveSuppressedLocked()
	}
}

func (s *PlaylistStore) IsSuppressed(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.suppressed[id]
}

// AutoLike adds songs to "Liked songs" on the system's own initiative (they are
// in the Jellyfin library). Songs the user un-liked are skipped. It returns the
// ids that were newly liked.
func (s *PlaylistStore) AutoLike(tracks []Track) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.find(likedID)
	if p == nil {
		return nil
	}
	have := map[string]bool{}
	for _, t := range p.Tracks {
		have[t.ID] = true
	}
	var added []string
	for _, raw := range tracks {
		t, ok := sanitizeTrack(raw)
		if !ok || have[t.ID] || s.suppressed[t.ID] || len(p.Tracks) >= 5000 {
			continue
		}
		have[t.ID] = true
		p.Tracks = append(p.Tracks, t)
		added = append(added, t.ID)
	}
	if len(added) > 0 {
		p.Updated = time.Now()
		s.save()
	}
	return added
}

func (s *PlaylistStore) save() error {
	b, err := json.MarshalIndent(s.lists, "", " ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *PlaylistStore) List() []PlaylistSummary {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]PlaylistSummary, 0, len(s.lists))
	for _, p := range s.lists {
		sum := PlaylistSummary{ID: p.ID, Name: p.Name, Count: len(p.Tracks)}
		if len(p.Tracks) > 0 {
			sum.Thumb = p.Tracks[len(p.Tracks)-1].Thumb
			if p.Ranked {
				sum.Thumb = p.Tracks[0].Thumb
			}
		}
		out = append(out, sum)
	}
	return out
}

func (s *PlaylistStore) Get(id string) (Playlist, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.find(id)
	if p == nil {
		return Playlist{}, false
	}
	cp := *p
	cp.Tracks = append([]Track{}, p.Tracks...)
	return cp, true
}

func (s *PlaylistStore) Create(name string) (PlaylistSummary, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 {
		return PlaylistSummary{}, errors.New("playlist name must be 1-100 characters")
	}
	b := make([]byte, 6)
	rand.Read(b)
	now := time.Now()
	s.mu.Lock()
	n := len(s.lists)
	s.mu.Unlock()
	if n >= 200 {
		return PlaylistSummary{}, errors.New("too many playlists")
	}
	p := &Playlist{ID: hex.EncodeToString(b), Name: name, Tracks: []Track{}, Created: now, Updated: now}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lists = append(s.lists, p)
	return PlaylistSummary{ID: p.ID, Name: p.Name}, s.save()
}

// ReplaceRanked creates a generated, ordered playlist, replacing an earlier one with the same name.
func (s *PlaylistStore) ReplaceRanked(name string, tracks []Track) (PlaylistSummary, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 {
		return PlaylistSummary{}, errors.New("playlist name must be 1-100 characters")
	}
	b := make([]byte, 6)
	rand.Read(b)
	now := time.Now()
	p := &Playlist{ID: hex.EncodeToString(b), Name: name, Tracks: []Track{}, Ranked: true, Created: now, Updated: now}
	seen := map[string]bool{}
	for _, raw := range tracks {
		if t, ok := sanitizeTrack(raw); ok && !seen[t.ID] {
			seen[t.ID] = true
			p.Tracks = append(p.Tracks, t)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.lists[:0:0]
	for _, l := range s.lists {
		if l.ID != likedID && l.Name == name {
			continue
		}
		kept = append(kept, l)
	}
	s.lists = append(kept, p)
	sum := PlaylistSummary{ID: p.ID, Name: p.Name, Count: len(p.Tracks)}
	if len(p.Tracks) > 0 {
		sum.Thumb = p.Tracks[0].Thumb
	}
	return sum, s.save()
}

func (s *PlaylistStore) Rename(id, name string) error {
	name = strings.TrimSpace(name)
	if id == likedID {
		return errors.New("Liked songs can't be renamed")
	}
	if name == "" || len(name) > 100 {
		return errors.New("playlist name must be 1-100 characters")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.find(id)
	if p == nil {
		return errors.New("playlist not found")
	}
	p.Name, p.Updated = name, time.Now()
	return s.save()
}

func (s *PlaylistStore) Delete(id string) error {
	if id == likedID {
		return errors.New("Liked songs can't be deleted")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, p := range s.lists {
		if p.ID == id {
			s.lists = append(s.lists[:i], s.lists[i+1:]...)
			return s.save()
		}
	}
	return errors.New("playlist not found")
}

// AddTracks appends tracks, skipping ones already present. For "Liked songs"
// new tracks go to the end too; the UI shows the list newest-first.
func (s *PlaylistStore) AddTracks(id string, tracks []Track) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.find(id)
	if p == nil {
		return 0, errors.New("playlist not found")
	}
	have := map[string]bool{}
	for _, t := range p.Tracks {
		have[t.ID] = true
	}
	added := 0
	for _, raw := range tracks {
		t, ok := sanitizeTrack(raw)
		if ok && id == likedID && s.suppressed[t.ID] { // a deliberate like overrides an earlier un-like
			delete(s.suppressed, t.ID)
			s.saveSuppressedLocked()
		}
		if !ok || have[t.ID] || len(p.Tracks) >= 5000 {
			continue
		}
		have[t.ID] = true
		p.Tracks = append(p.Tracks, t)
		added++
	}
	if added > 0 {
		p.Updated = time.Now()
		return added, s.save()
	}
	return 0, nil
}

func (s *PlaylistStore) RemoveTrack(id, trackID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.find(id)
	if p == nil {
		return errors.New("playlist not found")
	}
	for i, t := range p.Tracks {
		if t.ID == trackID {
			p.Tracks = append(p.Tracks[:i], p.Tracks[i+1:]...)
			p.Updated = time.Now()
			return s.save()
		}
	}
	return nil
}
