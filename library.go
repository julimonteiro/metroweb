package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
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
		return fmt.Errorf("missing from PATH: %s (brew install %s)", strings.Join(missing, ", "), strings.Join(missing, " "))
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

func (l *Library) download(id string) error {
	if err := l.Check(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Layout Jellyfin understands: Artist/Album/NN - Title.m4a
	// First artist only, so the folder matches album_artist and albums don't split.
	tmpl := filepath.Join(l.Dir,
		"%(artists.0,artist,creator,uploader|Desconhecido)s",
		"%(album|Singles)s",
		"%(track_number&{:02d} - |)s%(track,title)s.%(ext)s")

	args := []string{
		"-f", "bestaudio[ext=m4a]/bestaudio/best",
		"-x", "--audio-format", "m4a",
		"--embed-metadata", "--embed-thumbnail", "--convert-thumbnails", "jpg",
		// YT Music thumbnails are 16:9 with bars; crop to a square cover.
		"--ppa", `ThumbnailsConvertor+FFmpeg_o:-c:v mjpeg -vf crop="'if(gt(ih,iw),iw,ih)':'if(gt(iw,ih),ih,iw)'"`,
		"--parse-metadata", "%(artists.0,artist)s:^(?P<album_artist>[^,]+)",
		// Drop the long auto-generated description from the tags.
		"--parse-metadata", ":(?P<meta_description>)", "--parse-metadata", ":(?P<meta_synopsis>)",
		"--download-archive", l.archivePath(),
		"--ignore-config", "--no-playlist", "--no-progress", "--no-warnings",
		"-o", tmpl,
		"https://music.youtube.com/watch?v=" + id,
	}
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if i := strings.LastIndex(msg, "\n"); i >= 0 {
			msg = msg[i+1:]
		}
		return fmt.Errorf("%v: %s", err, msg)
	}
	return nil
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
