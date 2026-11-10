package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Innertube client configs. YouTube changes these every so often; if search or
// playback breaks, the first thing to check is these versions (yt-dlp's
// youtube extractor is a good reference for current values).
// ---------------------------------------------------------------------------

type clientCfg struct {
	Name       string // clientName in the request body
	Version    string
	HeaderName string // X-YouTube-Client-Name (numeric id)
	UserAgent  string
	Extra      map[string]any
	Host       string
}

var webRemix = clientCfg{
	Name:       "WEB_REMIX",
	Version:    "1.20250310.01.00",
	HeaderName: "67",
	UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	Host:       "https://music.youtube.com",
}

// Clients tried in order to resolve a stream. These return plain
// (non-ciphered) URLs, so no player JS has to be executed.
var streamClients = []clientCfg{
	{
		Name:       "ANDROID_VR",
		Version:    "1.60.19",
		HeaderName: "28",
		UserAgent:  "com.google.android.apps.youtube.vr.oculus/1.60.19 (Linux; U; Android 12L; eureka-user Build/SQ3A.220605.009.A1) gzip",
		Extra: map[string]any{
			"deviceMake": "Oculus", "deviceModel": "Quest 3",
			"androidSdkVersion": 32, "osName": "Android", "osVersion": "12L",
		},
		Host: "https://www.youtube.com",
	},
	{
		Name:       "IOS",
		Version:    "20.10.4",
		HeaderName: "5",
		UserAgent:  "com.google.ios.youtube/20.10.4 (iPhone16,2; U; CPU iOS 18_3_2 like Mac OS X;)",
		Extra: map[string]any{
			"deviceMake": "Apple", "deviceModel": "iPhone16,2",
			"osName": "iPhone", "osVersion": "18.3.2.22D82",
		},
		Host: "https://www.youtube.com",
	},
}

var httpClient = &http.Client{Timeout: 20 * time.Second}

func innertube(ctx context.Context, c clientCfg, endpoint string, body map[string]any) (map[string]any, error) {
	client := map[string]any{"clientName": c.Name, "clientVersion": c.Version, "hl": "en", "gl": "US"}
	for k, v := range c.Extra {
		client[k] = v
	}
	body["context"] = map[string]any{"client": client}
	buf, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", c.Host+"/youtubei/v1/"+endpoint+"?prettyPrint=false", bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("X-YouTube-Client-Name", c.HeaderName)
	req.Header.Set("X-YouTube-Client-Version", c.Version)
	req.Header.Set("Origin", c.Host)
	req.Header.Set("Referer", c.Host+"/")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("innertube %s/%s: HTTP %d: %.200s", c.Name, endpoint, resp.StatusCode, data)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// JSON helpers: innertube responses are deeply nested and shift around, so we
// walk them generically instead of mapping full structs.
// ---------------------------------------------------------------------------

func get(v any, path ...any) any {
	for _, p := range path {
		switch k := p.(type) {
		case string:
			m, ok := v.(map[string]any)
			if !ok {
				return nil
			}
			v = m[k]
		case int:
			a, ok := v.([]any)
			if !ok {
				return nil
			}
			if k < 0 {
				k = len(a) + k
			}
			if k < 0 || k >= len(a) {
				return nil
			}
			v = a[k]
		}
	}
	return v
}

func str(v any, path ...any) string {
	s, _ := get(v, path...).(string)
	return s
}

// findAll collects every object stored under `key` anywhere in the tree.
func findAll(v any, key string, out *[]map[string]any) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if k == key {
				if m, ok := child.(map[string]any); ok {
					*out = append(*out, m)
					continue
				}
			}
			findAll(child, key, out)
		}
	case []any:
		for _, child := range t {
			findAll(child, key, out)
		}
	}
}

func runsText(v any) []string {
	runs, _ := get(v, "runs").([]any)
	var out []string
	for _, r := range runs {
		if s := str(r, "text"); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func lastThumb(v any) string {
	u := str(v, -1, "url")
	// Ask for a bigger square thumbnail when possible.
	if i := strings.Index(u, "=w"); i > 0 && strings.Contains(u, "googleusercontent") {
		u = u[:i] + "=w400-h400-l90-rj"
	}
	return u
}

// ---------------------------------------------------------------------------
// Types and small helpers shared by the browse/search code (see browse.go)
// ---------------------------------------------------------------------------

type Track struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	ArtistID string `json:"artistId,omitempty"`
	Album    string `json:"album,omitempty"`
	AlbumID  string `json:"albumId,omitempty"`
	Duration string `json:"duration,omitempty"`
	Thumb    string `json:"thumb,omitempty"`
}

// isSep reports whether a text run is just a separator between artists or
// fields (", ", " & ", " • ", ...), which YouTube Music sends as runs.
func isSep(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "•", "·", ",", "e", "&", "and", "y", "et", "und", "feat.", "ft.":
		return true
	}
	return false
}

func isDuration(s string) bool {
	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return false
	}
	for _, p := range parts {
		if _, err := strconv.Atoi(p); err != nil {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Stream resolution
// ---------------------------------------------------------------------------

type Stream struct {
	URL       string
	Mime      string
	UserAgent string
	Expires   time.Time
	Source    string            // resolver that produced the URL (for logs)
	Headers   map[string]string // extra request headers the URL needs
}

type streamCache struct {
	mu sync.Mutex
	m  map[string]Stream
}

var cache = streamCache{m: map[string]Stream{}}

// ResolveStream returns a direct audio URL. pref is "opus" or "aac".
func ResolveStream(ctx context.Context, id, pref string) (Stream, error) {
	key := id + "|" + pref
	cache.mu.Lock()
	if s, ok := cache.m[key]; ok && time.Until(s.Expires) > 10*time.Minute {
		cache.mu.Unlock()
		return s, nil
	}
	cache.mu.Unlock()

	var errs []string
	remember := func(s Stream) Stream {
		cache.mu.Lock()
		cache.m[key] = s
		cache.mu.Unlock()
		return s
	}
	// yt-dlp first: with its PO-token plugin it returns URLs that stream the whole
	// file, while the bare innertube clients get 403 after ~1 MB.
	if s, err := resolveYtDlp(ctx, id, pref); err == nil {
		return remember(s), nil
	} else {
		errs = append(errs, err.Error())
	}
	for _, c := range streamClients {
		s, err := resolveWith(ctx, c, id, pref)
		if err == nil {
			return remember(s), nil
		}
		errs = append(errs, err.Error())
	}
	return Stream{}, errors.New(strings.Join(errs, " | "))
}

func resolveWith(ctx context.Context, c clientCfg, id, pref string) (Stream, error) {
	resp, err := innertube(ctx, c, "player", map[string]any{
		"videoId": id, "contentCheckOk": true, "racyCheckOk": true,
	})
	if err != nil {
		return Stream{}, err
	}
	if st := str(resp, "playabilityStatus", "status"); st != "OK" {
		return Stream{}, fmt.Errorf("%s: %s %s", c.Name, st, str(resp, "playabilityStatus", "reason"))
	}
	formats, _ := get(resp, "streamingData", "adaptiveFormats").([]any)
	type cand struct {
		url, mime string
		bitrate   float64
		score     int
	}
	var cands []cand
	for _, f := range formats {
		mime := str(f, "mimeType")
		u := str(f, "url")
		if !strings.HasPrefix(mime, "audio/") || u == "" {
			continue // ciphered formats (signatureCipher) are skipped
		}
		br, _ := get(f, "bitrate").(float64)
		score := 0
		if (pref == "opus" && strings.Contains(mime, "opus")) || (pref == "aac" && strings.Contains(mime, "mp4a")) {
			score = 1
		}
		cands = append(cands, cand{u, mime, br, score})
	}
	if len(cands) == 0 {
		return Stream{}, fmt.Errorf("%s: no direct audio formats", c.Name)
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].score != cands[j].score {
			return cands[i].score > cands[j].score
		}
		return cands[i].bitrate > cands[j].bitrate
	})
	best := cands[0]
	return Stream{URL: best.url, Mime: strings.Split(best.mime, ";")[0], UserAgent: c.UserAgent, Expires: expiry(best.url), Source: c.Name}, nil
}

func resolveYtDlp(ctx context.Context, id, pref string) (Stream, error) {
	if _, err := exec.LookPath("yt-dlp"); err != nil && ytdlpDir == "" {
		return Stream{}, errors.New("yt-dlp: not installed")
	}
	format := "bestaudio[acodec=opus]/bestaudio/best"
	if pref == "aac" {
		format = "bestaudio[ext=m4a]/bestaudio/best"
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	out, err := ytdlpCommand(ctx, "--ignore-config", "-f", format, "-j", "--no-warnings", "--no-playlist",
		"https://music.youtube.com/watch?v="+id).Output()
	if err != nil {
		return Stream{}, fmt.Errorf("yt-dlp: %v", err)
	}
	var info struct {
		URL     string            `json:"url"`
		Headers map[string]string `json:"http_headers"`
		Ext     string            `json:"ext"`
	}
	if err := json.Unmarshal(out, &info); err != nil || info.URL == "" {
		return Stream{}, errors.New("yt-dlp: no url in output")
	}
	// The URL is tied to the client yt-dlp used (TV, iOS, ...), so replay the
	// exact headers it reports instead of assuming a user agent.
	hdr := map[string]string{}
	for k, v := range info.Headers {
		if !strings.EqualFold(k, "range") {
			hdr[k] = v
		}
	}
	mime := "audio/webm"
	if info.Ext == "m4a" || strings.Contains(info.URL, "mime=audio%2Fmp4") {
		mime = "audio/mp4"
	}
	ua := hdr["User-Agent"]
	if ua == "" {
		ua = webRemix.UserAgent
	}
	return Stream{URL: info.URL, Mime: mime, UserAgent: ua, Headers: hdr, Expires: expiry(info.URL), Source: "yt-dlp"}, nil
}

func expiry(u string) time.Time {
	if p, err := url.Parse(u); err == nil {
		if e, err := strconv.ParseInt(p.Query().Get("expire"), 10, 64); err == nil {
			return time.Unix(e, 0)
		}
	}
	return time.Now().Add(3 * time.Hour)
}
