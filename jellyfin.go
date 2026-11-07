package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	byTitle map[string]map[string]bool // folded title -> set of folded artists
	count   int
	updated time.Time
	lastErr string
	kick    chan struct{}
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

type jfPage struct {
	Items []struct {
		Name        string   `json:"Name"`
		Artists     []string `json:"Artists"`
		AlbumArtist string   `json:"AlbumArtist"`
	} `json:"Items"`
	Total int `json:"TotalRecordCount"`
}

func (j *JellyfinIndex) Refresh(ctx context.Context) error {
	idx := map[string]map[string]bool{}
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
				}
				for a := range artists {
					idx[k][a] = true
				}
			}
			count++
		}
		start += len(page.Items)
		if len(page.Items) == 0 || start >= page.Total {
			break
		}
	}
	j.mu.Lock()
	j.byTitle, j.count, j.updated = idx, count, time.Now()
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
