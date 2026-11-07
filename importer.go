package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Importer matches the songs of the Jellyfin library to their YouTube Music
// counterparts (so they can be played here) and adds them to "Liked songs". It
// runs by itself about once a week, in the background, one job at a time, and is
// deliberately strict: a song is only imported when title, artist and duration
// agree. It remembers what it already resolved, so later runs only look up new
// songs, and it never re-likes something you un-liked.
type Importer struct {
	jf     *JellyfinIndex
	pls    *PlaylistStore
	search func(ctx context.Context, q string) ([]Item, error)
	dir    string

	mu      sync.Mutex
	st      ImportStatus
	seen    map[string]seenEntry
	lastRun int64
}

type seenEntry struct {
	ID string `json:"id,omitempty"` // YouTube video id; empty = no confident match
	At int64  `json:"at"`
}

const (
	importEvery     = 7 * 24 * time.Hour
	importRetryMiss = 30 * 24 * time.Hour // look again for unmatched songs after a month
)

type ImportStatus struct {
	Running   bool     `json:"running"`
	Total     int      `json:"total"`
	Done      int      `json:"done"`
	Added     int      `json:"added"`   // newly liked
	Already   int      `json:"already"` // matched but already liked
	Unmatched int      `json:"unmatched"`
	Missing   []string `json:"missing,omitempty"` // first few that could not be matched
	Skipped   int      `json:"skipped"`           // already handled by an earlier run
	Started   int64    `json:"started,omitempty"`
	Finished  int64    `json:"finished,omitempty"`
	LastRun   int64    `json:"lastRun,omitempty"`
	Next      int64    `json:"next,omitempty"`
	Error     string   `json:"error,omitempty"`
}

func NewImporter(jf *JellyfinIndex, pls *PlaylistStore, dir string) *Importer {
	im := &Importer{jf: jf, pls: pls, dir: dir, seen: map[string]seenEntry{}, search: func(ctx context.Context, q string) ([]Item, error) {
		return Search(ctx, q, "songs")
	}}
	if b, err := os.ReadFile(filepath.Join(dir, "import-state.json")); err == nil {
		var st struct {
			LastRun int64                `json:"lastRun"`
			Seen    map[string]seenEntry `json:"seen"`
		}
		if json.Unmarshal(b, &st) == nil {
			im.lastRun = st.LastRun
			if st.Seen != nil {
				im.seen = st.Seen
			}
		}
	}
	return im
}

// saveLocked persists the state; the caller holds im.mu.
func (im *Importer) saveLocked() {
	if im.dir == "" {
		return
	}
	b, _ := json.Marshal(map[string]any{"lastRun": im.lastRun, "seen": im.seen})
	os.WriteFile(filepath.Join(im.dir, "import-state.json"), b, 0o600)
}

func songKey(s JfSong) string { return fold(s.Artists[0]) + "|" + fold(s.Title) }

// Schedule runs the import shortly after start-up when it is due, then checks
// once an hour (so it syncs about once a week).
func (im *Importer) Schedule() {
	if im.jf == nil {
		return
	}
	go func() {
		time.Sleep(90 * time.Second) // let the Jellyfin index load first
		for {
			if im.Due() {
				if err := im.Start(); err != nil && !strings.Contains(err.Error(), "already running") {
					log.Printf("jellyfin sync: %v", err)
				}
			}
			time.Sleep(time.Hour)
		}
	}()
}

func (im *Importer) Due() bool {
	im.mu.Lock()
	defer im.mu.Unlock()
	return !im.st.Running && (im.lastRun == 0 || time.Since(time.Unix(im.lastRun, 0)) >= importEvery)
}

func (im *Importer) Status() ImportStatus {
	im.mu.Lock()
	defer im.mu.Unlock()
	st := im.st
	st.Missing = append([]string(nil), st.Missing...)
	st.LastRun = im.lastRun
	if im.lastRun > 0 {
		st.Next = im.lastRun + int64(importEvery/time.Second)
	}
	return st
}

// Start launches the import unless one is already running.
func (im *Importer) Start() error {
	if im.jf == nil {
		return errors.New("Jellyfin is not configured on this server")
	}
	songs := im.jf.Songs()
	if len(songs) == 0 {
		return errors.New("the Jellyfin library has no songs yet (it may still be loading, try again in a minute)")
	}
	im.mu.Lock()
	if im.st.Running {
		im.mu.Unlock()
		return errors.New("an import is already running")
	}
	var todo []JfSong
	skipped := 0
	for _, s := range songs {
		if e, ok := im.seen[songKey(s)]; ok && (e.ID != "" || time.Since(time.Unix(e.At, 0)) < importRetryMiss) {
			skipped++
			continue
		}
		todo = append(todo, s)
	}
	im.st = ImportStatus{Running: true, Total: len(todo), Skipped: skipped, Started: time.Now().Unix()}
	im.mu.Unlock()
	go im.run(todo)
	return nil
}

func (im *Importer) run(songs []JfSong) {
	liked, _ := im.pls.Get(likedID)
	have := map[string]bool{}
	for _, t := range liked.Tracks {
		have[t.ID] = true
	}
	var pending []Track
	flush := func() {
		if len(pending) > 0 {
			im.pls.AutoLike(pending)
			pending = nil
		}
	}
	jobs := make(chan JfSong)
	var wg sync.WaitGroup
	var haveMu sync.Mutex
	for w := 0; w < 3; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for s := range jobs {
				ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
				found, err := im.resolve(ctx, s)
				cancel()
				haveMu.Lock()
				im.mu.Lock()
				im.st.Done++
				switch {
				case err != nil:
					im.st.Unmatched++
					im.st.Error = err.Error() // not remembered: retried on the next run
				case found == nil:
					im.seen[songKey(s)] = seenEntry{At: time.Now().Unix()}
					im.st.Unmatched++
					if len(im.st.Missing) < 100 {
						im.st.Missing = append(im.st.Missing, s.Artists[0]+" - "+s.Title)
					}
				case have[found.ID] || im.pls.IsSuppressed(found.ID):
					im.seen[songKey(s)] = seenEntry{ID: found.ID, At: time.Now().Unix()}
					im.st.Already++
				default:
					im.seen[songKey(s)] = seenEntry{ID: found.ID, At: time.Now().Unix()}
					have[found.ID] = true
					im.st.Added++
					pending = append(pending, found.Track)
				}
				if len(pending) >= 25 {
					flush()
				}
				im.mu.Unlock()
				haveMu.Unlock()
				time.Sleep(300 * time.Millisecond) // be gentle with YouTube Music
			}
		}()
	}
	for _, s := range songs {
		jobs <- s
	}
	close(jobs)
	wg.Wait()
	haveMu.Lock()
	flush()
	haveMu.Unlock()
	im.mu.Lock()
	im.st.Running, im.st.Finished = false, time.Now().Unix()
	if im.st.Error == "" || im.st.Done > im.st.Unmatched { // a fully failed run (network down) is retried soon
		im.lastRun = im.st.Finished
	}
	im.saveLocked()
	im.mu.Unlock()
}

func (im *Importer) resolve(ctx context.Context, s JfSong) (*Item, error) {
	q := strings.TrimSpace(s.Artists[0] + " " + baseTitle(s.Title))
	cands, err := im.search(ctx, q)
	if err != nil {
		return nil, err
	}
	return matchSong(s, cands), nil
}

var versionWords = []string{"live", "remix", "acoustic", "instrumental", "karaoke", "cover", "sped up", "slowed", "reverb", "nightcore", "8d"}

// matchSong picks the YouTube Music result that is the same recording as the
// Jellyfin song: same title (exactly, or without bracketed extras), a shared
// artist, and a similar length. It returns nil rather than guess.
func matchSong(s JfSong, cands []Item) *Item {
	wantFull, wantBase := fold(s.Title), fold(baseTitle(s.Title))
	var wantArtists []string
	for _, a := range s.Artists {
		for _, p := range splitArtists(a) {
			if f := fold(p); f != "" {
				wantArtists = append(wantArtists, f)
			}
		}
	}
	lowerTitle := strings.ToLower(s.Title)
	var best *Item
	bestScore := 1e9
	for i := range cands {
		c := &cands[i]
		if c.Type != "song" {
			continue
		}
		score := 0.0
		switch cf := fold(c.Title); {
		case cf == wantFull:
		case wantBase != "" && fold(baseTitle(c.Title)) == wantBase:
			score += 1
		default:
			continue
		}
		artistOK := false
		for _, a := range splitArtists(c.Artist) {
			ca := fold(a)
			for _, w := range wantArtists {
				if ca != "" && (ca == w || strings.Contains(ca, w) || strings.Contains(w, ca)) {
					artistOK = true
				}
			}
		}
		if !artistOK {
			continue
		}
		if s.Seconds > 0 && c.Duration != "" {
			diff := absInt(durationSeconds(c.Duration) - s.Seconds)
			if diff > 12 {
				continue // a different version
			}
			score += float64(diff) / 12
		}
		lc := strings.ToLower(c.Title)
		for _, w := range versionWords {
			if strings.Contains(lc, w) && !strings.Contains(lowerTitle, w) {
				score += 2
			}
		}
		if score < bestScore {
			best, bestScore = c, score
		}
	}
	if bestScore >= 2 { // only a version marker or a weak match left: don't guess
		return nil
	}
	return best
}
