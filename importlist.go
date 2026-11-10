package main

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"sync"
	"time"
)

// Playlist import from text: a list of "Artist - Title" lines, or a CSV exported
// from Spotify (Exportify and similar), is matched song by song against YouTube
// Music and saved as a new playlist. Matching is strict: a song is only added
// when title and artist agree, and the rest is reported back.

type ListEntry struct{ Title, Artist string }

const maxImportLines = 1500

// ParseTrackList understands CSV with a header (Track Name / Artist Name(s) or
// title / artist columns) and plain lines: "Artist - Title", "Title - Artist"
// is NOT guessed. Tabs and semicolons also separate the two.
func ParseTrackList(text string) []ListEntry {
	text = strings.TrimPrefix(strings.TrimSpace(text), "\xef\xbb\xbf")
	if text == "" {
		return nil
	}
	if first := strings.SplitN(text, "\n", 2)[0]; strings.Contains(first, ",") {
		if es := parseCSVTracks(text); len(es) > 0 {
			return es
		}
	}
	var out []ListEntry
	for _, ln := range strings.Split(text, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		var a, t string
		for _, sep := range []string{"\t", ";", " - ", " – ", " — "} {
			if i := strings.Index(ln, sep); i > 0 {
				a, t = ln[:i], ln[i+len(sep):]
				break
			}
		}
		if a = strings.TrimSpace(a); a == "" || strings.TrimSpace(t) == "" {
			continue
		}
		out = append(out, ListEntry{Title: strings.TrimSpace(t), Artist: a})
		if len(out) >= maxImportLines {
			break
		}
	}
	return out
}

func parseCSVTracks(text string) []ListEntry {
	r := csv.NewReader(strings.NewReader(text))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	head, err := r.Read()
	if err != nil {
		return nil
	}
	ti, ai := -1, -1
	for i, h := range head {
		switch h = strings.ToLower(strings.TrimSpace(h)); {
		case h == "track name" || h == "title" || h == "name" || h == "track" || h == "song":
			if ti < 0 {
				ti = i
			}
		case h == "artist name(s)" || h == "artist name" || h == "artist" || h == "artists":
			if ai < 0 {
				ai = i
			}
		}
	}
	if ti < 0 || ai < 0 {
		return nil
	}
	var out []ListEntry
	for len(out) < maxImportLines {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || ti >= len(rec) || ai >= len(rec) {
			continue
		}
		t, a := strings.TrimSpace(rec[ti]), strings.TrimSpace(strings.ReplaceAll(rec[ai], ";", ", "))
		if t != "" && a != "" {
			out = append(out, ListEntry{Title: t, Artist: a})
		}
	}
	return out
}

type ImportListStatus struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Running    bool     `json:"running"`
	Total      int      `json:"total"`
	Done       int      `json:"done"`
	Found      int      `json:"found"`
	Missing    []string `json:"missing,omitempty"`
	PlaylistID string   `json:"playlistId,omitempty"`
	Error      string   `json:"error,omitempty"`
}

type ListImporter struct {
	pls    *PlaylistStore
	search func(ctx context.Context, q string) ([]Item, error)

	mu   sync.Mutex
	jobs map[string]*ImportListStatus
	busy bool
}

func NewListImporter(pls *PlaylistStore) *ListImporter {
	return &ListImporter{pls: pls, jobs: map[string]*ImportListStatus{}, search: func(ctx context.Context, q string) ([]Item, error) {
		return Search(ctx, q, "songs")
	}}
}

func (li *ListImporter) Start(name, text string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("give the playlist a name")
	}
	entries := ParseTrackList(text)
	if len(entries) == 0 {
		return "", errors.New("no songs found. Use one \"Artist - Title\" per line, or a CSV with Track Name and Artist Name(s) columns")
	}
	li.mu.Lock()
	if li.busy {
		li.mu.Unlock()
		return "", errors.New("another import is still running")
	}
	b := make([]byte, 6)
	rand.Read(b)
	id := hex.EncodeToString(b)
	li.jobs[id] = &ImportListStatus{ID: id, Name: name, Running: true, Total: len(entries)}
	li.busy = true
	if len(li.jobs) > 20 {
		for k, j := range li.jobs {
			if !j.Running && k != id {
				delete(li.jobs, k)
				break
			}
		}
	}
	li.mu.Unlock()
	go li.run(id, name, entries)
	return id, nil
}

func (li *ListImporter) Status(id string) (ImportListStatus, bool) {
	li.mu.Lock()
	defer li.mu.Unlock()
	j, ok := li.jobs[id]
	if !ok {
		return ImportListStatus{}, false
	}
	c := *j
	c.Missing = append([]string(nil), j.Missing...)
	return c, true
}

func (li *ListImporter) run(id, name string, entries []ListEntry) {
	found := make([]*Track, len(entries))
	var wg sync.WaitGroup
	jobs := make(chan int)
	for w := 0; w < 3; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				e := entries[i]
				ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
				cands, err := li.search(ctx, e.Artist+" "+baseTitle(e.Title))
				cancel()
				var m *Item
				if err == nil {
					m = matchSong(JfSong{Title: e.Title, Artists: []string{e.Artist}}, cands)
				}
				li.mu.Lock()
				j := li.jobs[id]
				j.Done++
				if m != nil {
					t := m.Track
					found[i] = &t
					j.Found++
				} else if len(j.Missing) < 100 {
					j.Missing = append(j.Missing, e.Artist+" - "+e.Title)
				}
				li.mu.Unlock()
				time.Sleep(250 * time.Millisecond)
			}
		}()
	}
	for i := range entries {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	var tracks []Track
	seen := map[string]bool{}
	for _, t := range found {
		if t != nil && !seen[t.ID] {
			seen[t.ID] = true
			tracks = append(tracks, *t)
		}
	}
	li.mu.Lock()
	defer li.mu.Unlock()
	j := li.jobs[id]
	j.Running, li.busy = false, false
	if len(tracks) == 0 {
		j.Error = "None of the songs could be matched on YouTube Music"
		return
	}
	pl, err := li.pls.Create(name)
	if err != nil {
		j.Error = err.Error()
		return
	}
	if _, err := li.pls.AddTracks(pl.ID, tracks); err != nil {
		j.Error = err.Error()
		return
	}
	j.PlaylistID = pl.ID
}
