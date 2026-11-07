package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// streamCacheDir holds audio fetched by yt-dlp for the fallback path.
var streamCacheDir string

var (
	dlLocks    sync.Map                 // id|pref -> *sync.Mutex
	cacheKeep  = 150                    // max cached files
	cacheBytes = int64(3 << 30)         // max total size of the cache (3 GB)
	videoSlots = make(chan struct{}, 2) // at most two clip downloads at once
)

func cachedFile(id, pref string) string {
	matches, _ := filepath.Glob(filepath.Join(streamCacheDir, id+"."+pref+".*"))
	for _, m := range matches {
		if !strings.HasSuffix(m, ".part") && !strings.HasSuffix(m, ".ytdl") {
			return m
		}
	}
	return ""
}

// ensureCached makes sure the file for id/pref is in the cache (downloading it
// with yt-dlp once, even when several requests ask at the same time) and
// returns its path.
func ensureCached(ctx context.Context, id, pref string) (string, error) {
	if streamCacheDir == "" {
		return "", fmt.Errorf("cache unavailable")
	}
	if p := cachedFile(id, pref); p != "" {
		return p, nil
	}
	mu, _ := dlLocks.LoadOrStore(id+"|"+pref, &sync.Mutex{})
	l := mu.(*sync.Mutex)
	l.Lock()
	defer l.Unlock()
	if p := cachedFile(id, pref); p != "" {
		return p, nil
	}
	if err := fetchToCache(ctx, id, pref); err != nil {
		return "", err
	}
	if p := cachedFile(id, pref); p != "" {
		return p, nil
	}
	return "", fmt.Errorf("download finished but the file is missing")
}

func serveFromCache(w http.ResponseWriter, r *http.Request, id, pref string) {
	path, err := ensureCached(r.Context(), id, pref)
	if err != nil {
		log.Printf("stream cache %s: %v", id, err)
		http.Error(w, "stream unavailable", 502)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, "stream unavailable", 502)
		return
	}
	defer f.Close()
	st, _ := f.Stat()
	ct := "audio/webm"
	switch {
	case pref == "video" && strings.HasSuffix(path, ".mp4"):
		ct = "video/mp4"
	case pref == "video":
		ct = "video/webm"
	case strings.HasSuffix(path, ".m4a") || strings.HasSuffix(path, ".mp4"):
		ct = "audio/mp4"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, "", st.ModTime(), f)
}

func fetchToCache(ctx context.Context, id, pref string) error {
	if err := os.MkdirAll(streamCacheDir, 0o755); err != nil {
		return err
	}
	format := "bestaudio[acodec=opus]/bestaudio/best"
	extra := []string{}
	timeout := 2 * time.Minute
	switch pref {
	case "aac":
		format = "bestaudio[ext=m4a]/bestaudio/best"
	case "video":
		// 720p at most, merged into one mp4 so a single file carries picture and sound.
		format = "bv*[height<=720][vcodec^=avc1]+ba[ext=m4a]/b[height<=720][vcodec^=avc1]/bv*[height<=720][ext=mp4]+ba[ext=m4a]/bv*[height<=720]+ba/b[height<=720]/b" // H.264 first: it plays everywhere
		extra = []string{"--merge-output-format", "mp4"}
		timeout = 5 * time.Minute
		videoSlots <- struct{}{}
		defer func() { <-videoSlots }()
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	args := append([]string{"--ignore-config", "-f", format}, extra...)
	args = append(args, "--no-playlist", "--no-warnings", "--no-progress",
		"-o", filepath.Join(streamCacheDir, id+"."+pref+".%(ext)s"),
		"https://music.youtube.com/watch?v="+id)
	out, err := exec.CommandContext(ctx, "yt-dlp", args...).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if i := strings.LastIndex(msg, "\n"); i >= 0 {
			msg = msg[i+1:]
		}
		return fmt.Errorf("yt-dlp: %v: %s", err, msg)
	}
	go pruneCache()
	return nil
}

func pruneCache() {
	entries, err := os.ReadDir(streamCacheDir)
	if err != nil {
		return
	}
	type fi struct {
		path string
		mod  time.Time
		size int64
	}
	var files []fi
	var total int64
	for _, e := range entries {
		if info, err := e.Info(); err == nil && !e.IsDir() {
			files = append(files, fi{filepath.Join(streamCacheDir, e.Name()), info.ModTime(), info.Size()})
			total += info.Size()
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod.Before(files[j].mod) })
	for i := 0; i < len(files) && (len(files)-i > cacheKeep || total > cacheBytes); i++ {
		os.Remove(files[i].path)
		total -= files[i].size
	}
}
