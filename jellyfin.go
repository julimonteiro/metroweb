package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode"
)

// JellyfinIndex keeps an in-memory list of the songs already in a Jellyfin
// library (title + artists only) so metroweb can say "you already have this"
// instead of downloading duplicates. It only ever reads item names.
type JellyfinIndex struct {
	base, key string
	client    *http.Client

	mu      sync.RWMutex
	byTitle map[string]map[string]bool   // folded title -> set of folded artists
	idByKey map[string]map[string]string // folded title -> folded artist -> Jellyfin item id
	songs   []JfSong
	count   int
	updated time.Time
	lastErr string
	kick    chan struct{}
	userID  string // Jellyfin user plays are credited to (resolved lazily)
}

func NewJellyfinIndex(baseURL, key string) *JellyfinIndex {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || key == "" {
		return nil
	}
	return &JellyfinIndex{base: baseURL, key: key, client: &http.Client{Timeout: 60 * time.Second},
		byTitle: map[string]map[string]bool{}, kick: make(chan struct{}, 1)}
}

// Run refreshes the index at start, every 10 minutes, and when kicked (after a
// library rescan we triggered).
func (j *JellyfinIndex) Run() {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		err := j.Refresh(ctx)
		cancel()
		j.mu.Lock()
		if err != nil {
			j.lastErr = err.Error()
			log.Printf("jellyfin index: %v", err)
		} else {
			j.lastErr = ""
		}
		j.mu.Unlock()
		select {
		case <-time.After(10 * time.Minute):
		case <-j.kick:
			time.Sleep(20 * time.Second) // give Jellyfin a moment to finish scanning
		}
	}
}

// Kick asks for a refresh soon (used after a rescan).
func (j *JellyfinIndex) Kick() {
	if j == nil {
		return
	}
	select {
	case j.kick <- struct{}{}:
	default:
	}
}

// JfSong is one song of the Jellyfin library.
type JfSong struct {
	ID      string
	Title   string
	Artists []string // performers, album artist last
	Album   string
	Seconds int
}

type jfPage struct {
	Items []struct {
		ID           string   `json:"Id"`
		Name         string   `json:"Name"`
		Artists      []string `json:"Artists"`
		AlbumArtist  string   `json:"AlbumArtist"`
		Album        string   `json:"Album"`
		RunTimeTicks int64    `json:"RunTimeTicks"`
	} `json:"Items"`
	Total int `json:"TotalRecordCount"`
}

func (j *JellyfinIndex) Refresh(ctx context.Context) error {
	idx := map[string]map[string]bool{}
	ids := map[string]map[string]string{}
	var songs []JfSong
	count := 0
	for start := 0; ; {
		q := url.Values{"IncludeItemTypes": {"Audio"}, "Recursive": {"true"}, "Fields": {""},
			"StartIndex": {fmt.Sprint(start)}, "Limit": {"1000"}, "EnableTotalRecordCount": {"true"}}
		req, _ := http.NewRequestWithContext(ctx, "GET", j.base+"/Items?"+q.Encode(), nil)
		req.Header.Set("Authorization", `MediaBrowser Token="`+j.key+`"`)
		resp, err := j.client.Do(req)
		if err != nil {
			return err
		}
		var page jfPage
		err = json.NewDecoder(resp.Body).Decode(&page)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("jellyfin answered HTTP %d (check the API key and URL)", resp.StatusCode)
		}
		if err != nil {
			return err
		}
		for _, it := range page.Items {
			artists := map[string]bool{}
			for _, a := range append(append([]string{}, it.Artists...), it.AlbumArtist) {
				if f := fold(a); f != "" {
					artists[f] = true
				}
			}
			for _, k := range titleKeys(it.Name) {
				if idx[k] == nil {
					idx[k] = map[string]bool{}
					ids[k] = map[string]string{}
				}
				for a := range artists {
					idx[k][a] = true
					if ids[k][a] == "" && it.ID != "" {
						ids[k][a] = it.ID
					}
				}
			}
			count++
			perf := it.Artists
			if len(perf) == 0 && it.AlbumArtist != "" {
				perf = []string{it.AlbumArtist}
			}
			if it.Name != "" && len(perf) > 0 {
				songs = append(songs, JfSong{ID: it.ID, Title: it.Name, Artists: perf, Album: it.Album, Seconds: int(it.RunTimeTicks / 10_000_000)})
			}
		}
		start += len(page.Items)
		if len(page.Items) == 0 || start >= page.Total {
			break
		}
	}
	j.mu.Lock()
	j.byTitle, j.idByKey, j.songs, j.count, j.updated = idx, ids, songs, count, time.Now()
	j.mu.Unlock()
	log.Printf("jellyfin index: %d songs", count)
	return nil
}

// Has reports whether a song with this title by one of these artists is
// already in the Jellyfin library.
func (j *JellyfinIndex) Has(title, artist string) bool {
	if j == nil {
		return false
	}
	var want []string
	for _, a := range splitArtists(artist) {
		if f := fold(a); f != "" {
			want = append(want, f)
		}
	}
	if len(want) == 0 {
		return false
	}
	j.mu.RLock()
	defer j.mu.RUnlock()
	for _, k := range titleKeys(title) {
		have := j.byTitle[k]
		for _, a := range want {
			if have[a] {
				return true
			}
		}
	}
	return false
}

type JellyfinStatus struct {
	Enabled bool   `json:"enabled"`
	Songs   int    `json:"songs"`
	Updated int64  `json:"updated,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (j *JellyfinIndex) Status() JellyfinStatus {
	if j == nil {
		return JellyfinStatus{}
	}
	j.mu.RLock()
	defer j.mu.RUnlock()
	st := JellyfinStatus{Enabled: true, Songs: j.count, Error: j.lastErr}
	if !j.updated.IsZero() {
		st.Updated = j.updated.Unix()
	}
	return st
}

// ---------------------------------------------------------------------------
// Matching helpers
// ---------------------------------------------------------------------------

var accentFold = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a", "å", "a", "ā", "a", "ă", "a", "ą", "a",
	"ç", "c", "ć", "c", "č", "c", "ď", "d", "ð", "d",
	"é", "e", "è", "e", "ê", "e", "ë", "e", "ē", "e", "ę", "e", "ě", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i", "ī", "i", "ı", "i",
	"ł", "l", "ñ", "n", "ń", "n", "ň", "n",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o", "ø", "o", "ō", "o", "ő", "o",
	"ř", "r", "ś", "s", "š", "s", "ş", "s", "ß", "ss", "ť", "t",
	"ú", "u", "ù", "u", "û", "u", "ü", "u", "ū", "u", "ů", "u", "ű", "u",
	"ý", "y", "ÿ", "y", "ź", "z", "ż", "z", "ž", "z", "æ", "ae", "œ", "oe", "&", "and")

// fold lowercases, removes accents and keeps only letters and digits.
func fold(s string) string {
	s = accentFold.Replace(strings.ToLower(s))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// baseTitle drops bracketed parts and " - Remastered"-style suffixes:
// "One More Time (Radio Edit)" -> "One More Time".
func baseTitle(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case '(', '[':
			depth++
		case ')', ']':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				b.WriteRune(r)
			}
		}
	}
	out := b.String()
	if i := strings.Index(out, " - "); i > 0 {
		out = out[:i]
	}
	for _, sep := range []string{" feat.", " feat ", " ft.", " featuring "} {
		if i := strings.Index(strings.ToLower(out), sep); i > 0 {
			out = out[:i]
		}
	}
	return out
}

func titleKeys(title string) []string {
	full, base := fold(title), fold(baseTitle(title))
	switch {
	case full == "":
		return nil
	case base == "" || base == full:
		return []string{full}
	}
	return []string{full, base}
}

func splitArtists(s string) []string {
	s = strings.NewReplacer(" & ", ",", " and ", ",", " x ", ",", " feat. ", ",", " ft. ", ",").Replace(s)
	return strings.Split(s, ",")
}

// Songs returns a copy of the indexed library (empty when disabled).
func (j *JellyfinIndex) Songs() []JfSong {
	if j == nil {
		return nil
	}
	j.mu.RLock()
	defer j.mu.RUnlock()
	return append([]JfSong(nil), j.songs...)
}

// Find returns the Jellyfin item id of a song with this title by one of these
// artists (same matching rules as Has), so it can be played straight from the
// local library instead of being fetched from YouTube.
func (j *JellyfinIndex) Find(title, artist string) (string, bool) {
	if j == nil || title == "" || artist == "" {
		return "", false
	}
	var want []string
	for _, a := range splitArtists(artist) {
		if f := fold(a); f != "" {
			want = append(want, f)
		}
	}
	j.mu.RLock()
	defer j.mu.RUnlock()
	for _, k := range titleKeys(title) { // exact title first, then without "(Radio Edit)" etc.
		for _, a := range want {
			if id := j.idByKey[k][a]; id != "" {
				return id, true
			}
		}
	}
	return "", false
}

// Stream proxies the original audio file of a library item (Range supported),
// keeping the API key on the server. It reports false, having written nothing,
// when Jellyfin can't serve it, so the caller can fall back to YouTube.
func (j *JellyfinIndex) Stream(w http.ResponseWriter, r *http.Request, itemID string) bool {
	req, _ := http.NewRequestWithContext(r.Context(), "GET", j.base+"/Audio/"+url.PathEscape(itemID)+"/stream?static=true", nil)
	req.Header.Set("Authorization", `MediaBrowser Token="`+j.key+`"`)
	if rng := r.Header.Get("Range"); rng != "" {
		req.Header.Set("Range", rng)
	}
	resp, err := (&http.Client{}).Do(req) // no timeout: long-lived body
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return false
	}
	for _, h := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges"} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	return true
}
