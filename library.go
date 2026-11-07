package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Library saves tracks as tagged audio files into a folder (e.g. the one
// Jellyfin watches), using yt-dlp + ffmpeg, and optionally asks Jellyfin to
// rescan afterwards.
type Library struct {
	Dir         string
	JellyfinURL string
	JellyfinKey string

	mu     sync.Mutex
	status map[string]string // id -> queued | running | done | error: ...
	jobs   chan string
	rescan *time.Timer
}

func NewLibrary(dir, jfURL, jfKey string) *Library {
	l := &Library{Dir: dir, JellyfinURL: strings.TrimRight(jfURL, "/"), JellyfinKey: jfKey,
		status: map[string]string{}, jobs: make(chan string, 500)}
	for id := range l.archived() {
		l.status[id] = "done"
	}
	go l.worker()
	return l
}

func (l *Library) archivePath() string { return filepath.Join(l.Dir, ".metroweb-archive.txt") }

// archived reads yt-dlp's --download-archive file ("youtube <id>" per line).
func (l *Library) archived() map[string]bool {
	out := map[string]bool{}
	f, err := os.Open(l.archivePath())
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if parts := strings.Fields(sc.Text()); len(parts) == 2 {
			out[parts[1]] = true
		}
	}
	return out
}

// Check reports missing tools so the UI can say why downloads won't work.
func (l *Library) Check() error {
	var missing []string
	for _, bin := range []string{"yt-dlp", "ffmpeg"} {
		if _, err := exec.LookPath(bin); err != nil {
			missing = append(missing, bin)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("faltando no PATH: %s (brew install %s)", strings.Join(missing, ", "), strings.Join(missing, " "))
	}
	return os.MkdirAll(l.Dir, 0o755)
}

func (l *Library) Enqueue(id string) string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if st := l.status[id]; st == "done" || st == "queued" || st == "running" {
		return st
	}
	l.status[id] = "queued"
	l.jobs <- id
	return "queued"
}

func (l *Library) Status() map[string]string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make(map[string]string, len(l.status))
	for k, v := range l.status {
		out[k] = v
	}
	return out
}

func (l *Library) set(id, st string) {
	l.mu.Lock()
	l.status[id] = st
	l.mu.Unlock()
}

func (l *Library) worker() {
	for id := range l.jobs {
		l.set(id, "running")
		if err := l.download(id); err != nil {
			log.Printf("download %s: %v", id, err)
			l.set(id, "error: "+err.Error())
			continue
		}
		log.Printf("download %s: ok", id)
		l.set(id, "done")
		l.scheduleRescan()
	}
}

type ytInfo struct {
	Title       string   `json:"title"`
	Track       string   `json:"track"`
	Artist      string   `json:"artist"`
	Artists     []string `json:"artists"`
	Creator     string   `json:"creator"`
	Uploader    string   `json:"uploader"`
	Album       string   `json:"album"`
	TrackNumber int      `json:"track_number"`
	ReleaseYear int      `json:"release_year"`
	Genre       string   `json:"genre"`
	Thumbnails  []struct {
		URL string `json:"url"`
	} `json:"thumbnails"`
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

var badPath = strings.NewReplacer("/", "-", ":", "-", "\\", "-", "*", "", "?", "", "\"", "'", "<", "", ">", "", "|", "-")

func safeName(s, def string) string {
	s = strings.Trim(badPath.Replace(strings.TrimSpace(s)), ". ")
	if s == "" {
		return def
	}
	return s
}

// info asks yt-dlp for metadata only; the audio itself is fetched with our own
// resolver because yt-dlp's download requests currently get 403 from googlevideo.
func (l *Library) info(ctx context.Context, id string) (ytInfo, error) {
	var in ytInfo
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "yt-dlp", "-J", "--no-playlist", "--no-warnings",
		"--ignore-no-formats-error", "https://music.youtube.com/watch?v="+id)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if i := strings.LastIndex(msg, "\n"); i >= 0 {
			msg = msg[i+1:]
		}
		return in, fmt.Errorf("yt-dlp: %v: %s", err, msg)
	}
	return in, json.Unmarshal(out, &in)
}

// fetchTo downloads url into dst. For audio streams (ua set) it uses bounded
// Range chunks: googlevideo answers 403 to open-ended ranges on these URLs.
func fetchTo(ctx context.Context, url, ua, dst string) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	const chunk = 4 << 20
	for off := int64(0); ; {
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		if ua != "" {
			req.Header.Set("User-Agent", ua)
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", off, off+chunk-1))
		}
		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode >= 300 {
			resp.Body.Close()
			return fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		n, err := io.Copy(f, resp.Body)
		resp.Body.Close()
		if err != nil {
			return err
		}
		if ua == "" || resp.StatusCode != http.StatusPartialContent || n < chunk {
			return nil
		}
		off += n
	}
}

// coverURL prefers the square album art (googleusercontent) over the 16:9 video frame.
func coverURL(in ytInfo, id string) string {
	for i := len(in.Thumbnails) - 1; i >= 0; i-- {
		u := in.Thumbnails[i].URL
		if strings.Contains(u, "googleusercontent.com") {
			if k := strings.Index(u, "="); k > 0 {
				u = u[:k] + "=w800-h800-l90-rj"
			}
			return u
		}
	}
	return "https://i.ytimg.com/vi/" + id + "/maxresdefault.jpg"
}

func (l *Library) download(id string) error {
	if err := l.Check(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	in, err := l.info(ctx, id)
	if err != nil {
		return err
	}
	artists := in.Artists
	if len(artists) == 0 {
		artists = []string{firstNonEmpty(in.Artist, in.Creator, in.Uploader, "Desconhecido")}
	}
	albumArtist := artists[0]
	title := firstNonEmpty(in.Track, in.Title, id)
	album := firstNonEmpty(in.Album, "Singles")

	// Layout Jellyfin understands: Artist/Album/NN - Title.m4a
	name := safeName(title, id)
	if in.TrackNumber > 0 {
		name = fmt.Sprintf("%02d - %s", in.TrackNumber, name)
	}
	dir := filepath.Join(l.Dir, safeName(albumArtist, "Desconhecido"), safeName(album, "Singles"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	final := filepath.Join(dir, name+".m4a")

	tmp, err := os.MkdirTemp("", "metroweb-dl-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	audio := filepath.Join(tmp, "audio.m4a")
	for attempt := 0; ; attempt++ {
		s, err := ResolveStream(ctx, id, "aac")
		if err != nil {
			return err
		}
		err = fetchTo(ctx, s.URL, s.UserAgent, audio)
		if err == nil {
			break
		}
		// Cached URL may be stale/rejected: drop it and resolve again once.
		cache.mu.Lock()
		delete(cache.m, id+"|aac")
		cache.mu.Unlock()
		if attempt >= 1 {
			return fmt.Errorf("baixando áudio: %w", err)
		}
	}
	cover := filepath.Join(tmp, "cover.jpg")
	haveCover := fetchTo(ctx, coverURL(in, id), "", cover) == nil

	args := []string{"-y", "-v", "error", "-i", audio}
	if haveCover {
		args = append(args, "-i", cover)
	}
	args = append(args, "-map", "0:a")
	if haveCover {
		// Crop to a centered square so Jellyfin shows a proper album cover.
		args = append(args, "-map", "1:v", "-c:v", "mjpeg", "-vf", "crop='min(iw,ih)':'min(iw,ih)'",
			"-disposition:v:0", "attached_pic")
	}
	args = append(args, "-c:a", "copy",
		"-metadata", "title="+title,
		"-metadata", "artist="+strings.Join(artists, ", "),
		"-metadata", "album_artist="+albumArtist,
		"-metadata", "album="+album,
		"-metadata", "comment=youtube:"+id)
	if in.TrackNumber > 0 {
		args = append(args, "-metadata", fmt.Sprintf("track=%d", in.TrackNumber))
	}
	if in.ReleaseYear > 0 {
		args = append(args, "-metadata", fmt.Sprintf("date=%d", in.ReleaseYear))
	}
	if in.Genre != "" {
		args = append(args, "-metadata", "genre="+in.Genre)
	}
	out := filepath.Join(tmp, "out.m4a")
	args = append(args, "-movflags", "+faststart", out)
	if b, err := exec.CommandContext(ctx, "ffmpeg", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg: %v: %s", err, strings.TrimSpace(string(b)))
	}
	if err := moveFile(out, final); err != nil {
		return err
	}
	f, err := os.OpenFile(l.archivePath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err == nil {
		fmt.Fprintf(f, "youtube %s\n", id)
		f.Close()
	}
	return nil
}

func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// scheduleRescan debounces Jellyfin library refreshes so a batch of downloads
// triggers a single scan.
func (l *Library) scheduleRescan() {
	if l.JellyfinURL == "" || l.JellyfinKey == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.rescan != nil {
		l.rescan.Stop()
	}
	l.rescan = time.AfterFunc(15*time.Second, func() {
		req, _ := http.NewRequest("POST", l.JellyfinURL+"/Library/Refresh", nil)
		req.Header.Set("Authorization", `MediaBrowser Token="`+l.JellyfinKey+`"`)
		resp, err := httpClient.Do(req)
		if err != nil {
			log.Printf("jellyfin rescan: %v", err)
			return
		}
		resp.Body.Close()
		log.Printf("jellyfin rescan: HTTP %d", resp.StatusCode)
	})
}
