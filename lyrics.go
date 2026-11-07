package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LyricLine is one line of time-synced lyrics (t in seconds).
type LyricLine struct {
	T    float64 `json:"t"`
	Text string  `json:"text"`
}

type Lyrics struct {
	Enabled      bool        `json:"enabled"`
	Found        bool        `json:"found"`
	Synced       []LyricLine `json:"synced,omitempty"`
	Plain        string      `json:"plain,omitempty"`
	Instrumental bool        `json:"instrumental,omitempty"`
}

// LyricsService looks lyrics up on LRCLIB (https://lrclib.net), a free, open
// lyrics database. Only the song title, artist, album and duration are sent.
type LyricsService struct {
	base   string
	client *http.Client
	mu     sync.Mutex
	cache  map[string]lyricsEntry
}

type lyricsEntry struct {
	v  Lyrics
	at time.Time
}

func NewLyricsService(enabled bool) *LyricsService {
	if !enabled {
		return nil
	}
	return &LyricsService{base: "https://lrclib.net", client: &http.Client{Timeout: 12 * time.Second}, cache: map[string]lyricsEntry{}}
}

type lrclibTrack struct {
	TrackName    string  `json:"trackName"`
	ArtistName   string  `json:"artistName"`
	Duration     float64 `json:"duration"`
	Instrumental bool    `json:"instrumental"`
	PlainLyrics  string  `json:"plainLyrics"`
	SyncedLyrics string  `json:"syncedLyrics"`
}

func (s *LyricsService) fetch(ctx context.Context, path string, q url.Values, out any) (int, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", s.base+path+"?"+q.Encode(), nil)
	req.Header.Set("User-Agent", "metroweb/1.0 (https://github.com/julimonteiro/metroweb)")
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
		return resp.StatusCode, nil
	}
	return 200, json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(out)
}

// Get returns lyrics for a song, preferring time-synced ones. Results (and
// misses) are cached in memory.
func (s *LyricsService) Get(ctx context.Context, title, artist, album string, durSec int) (Lyrics, error) {
	if s == nil {
		return Lyrics{}, nil
	}
	artist = strings.TrimSpace(strings.Split(artist, ",")[0])
	key := fold(title) + "|" + fold(artist) + "|" + strconv.Itoa(durSec/3)
	s.mu.Lock()
	if e, ok := s.cache[key]; ok && time.Since(e.at) < map[bool]time.Duration{true: 24 * time.Hour, false: time.Hour}[e.v.Found] {
		s.mu.Unlock()
		return e.v, nil
	}
	s.mu.Unlock()

	best, err := s.find(ctx, title, artist, album, durSec)
	if err != nil {
		return Lyrics{}, err
	}
	res := Lyrics{Enabled: true}
	if best != nil {
		res.Instrumental = best.Instrumental
		res.Synced = parseLRC(best.SyncedLyrics)
		res.Plain = strings.TrimSpace(best.PlainLyrics)
		res.Found = len(res.Synced) > 0 || res.Plain != "" || best.Instrumental
	}
	s.mu.Lock()
	if len(s.cache) > 500 {
		s.cache = map[string]lyricsEntry{}
	}
	s.cache[key] = lyricsEntry{res, time.Now()}
	s.mu.Unlock()
	return res, nil
}

func (s *LyricsService) find(ctx context.Context, title, artist, album string, durSec int) (*lrclibTrack, error) {
	// 1) exact lookup (best quality when title/album/duration match)
	if durSec > 0 && album != "" {
		var t lrclibTrack
		code, err := s.fetch(ctx, "/api/get", url.Values{"artist_name": {artist}, "track_name": {title}, "album_name": {album}, "duration": {strconv.Itoa(durSec)}}, &t)
		if err != nil {
			return nil, err
		}
		if code == 200 && (t.SyncedLyrics != "" || t.PlainLyrics != "" || t.Instrumental) {
			return &t, nil
		}
	}
	// 2) search by cleaned title ("One More Time (Radio Edit)" -> "One More Time")
	for _, name := range uniqueStrings(title, strings.TrimSpace(baseTitle(title))) {
		var list []lrclibTrack
		code, err := s.fetch(ctx, "/api/search", url.Values{"track_name": {name}, "artist_name": {artist}}, &list)
		if err != nil {
			return nil, err
		}
		if code != 200 {
			continue
		}
		if t := pickLyrics(list, artist, durSec); t != nil {
			return t, nil
		}
	}
	return nil, nil
}

func uniqueStrings(v ...string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range v {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// pickLyrics chooses the result by the same artist and the closest duration,
// preferring synced lyrics.
func pickLyrics(list []lrclibTrack, artist string, durSec int) *lrclibTrack {
	type scored struct {
		t     *lrclibTrack
		score float64
	}
	var cands []scored
	for i := range list {
		t := &list[i]
		if fold(t.ArtistName) != fold(artist) && !strings.Contains(fold(t.ArtistName), fold(artist)) {
			continue
		}
		if t.SyncedLyrics == "" && t.PlainLyrics == "" && !t.Instrumental {
			continue
		}
		diff := 0.0
		if durSec > 0 && t.Duration > 0 {
			diff = math.Abs(t.Duration - float64(durSec))
			if diff > 6 {
				continue // a different recording/version
			}
		}
		score := diff
		if t.SyncedLyrics == "" {
			score += 3
		}
		cands = append(cands, scored{t, score})
	}
	if len(cands) == 0 {
		return nil
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].score < cands[j].score })
	return cands[0].t
}

var lrcTime = regexp.MustCompile(`\[(\d{1,3}):(\d{2})(?:[.:](\d{1,3}))?\]`)

// parseLRC turns "[01:23.45] text" lines into sorted LyricLines. Blank lines
// are kept (instrumental breaks); metadata tags like [ar:..] are ignored.
func parseLRC(s string) []LyricLine {
	var out []LyricLine
	for _, raw := range strings.Split(strings.ReplaceAll(s, "\r", ""), "\n") {
		ms := lrcTime.FindAllStringSubmatchIndex(raw, -1)
		if len(ms) == 0 {
			continue
		}
		text := strings.TrimSpace(raw[ms[len(ms)-1][1]:])
		for _, m := range ms {
			min, _ := strconv.Atoi(raw[m[2]:m[3]])
			sec, _ := strconv.Atoi(raw[m[4]:m[5]])
			frac := 0.0
			if m[6] >= 0 {
				f := raw[m[6]:m[7]]
				v, _ := strconv.Atoi(f)
				frac = float64(v) / math.Pow(10, float64(len(f)))
			}
			out = append(out, LyricLine{T: float64(min*60+sec) + frac, Text: text})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].T < out[j].T })
	return out
}

func durationFromParam(s string) int {
	if strings.Contains(s, ":") {
		return durationSeconds(s)
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 4*3600 {
		return 0
	}
	return n
}

func lyricsHandler(svc *LyricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svc == nil {
			writeJSON(w, 200, Lyrics{})
			return
		}
		q := r.URL.Query()
		title, artist := strings.TrimSpace(q.Get("title")), strings.TrimSpace(q.Get("artist"))
		if title == "" || artist == "" || len(title) > 200 || len(artist) > 200 || len(q.Get("album")) > 200 {
			apiErr(w, 400, "title and artist are required")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		res, err := svc.Get(ctx, title, artist, strings.TrimSpace(q.Get("album")), durationFromParam(q.Get("duration")))
		if err != nil {
			apiErr(w, 502, fmt.Sprintf("lyrics lookup failed: %v", err))
			return
		}
		res.Enabled = true
		writeJSON(w, 200, res)
	}
}
