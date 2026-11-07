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
	dlLocks   sync.Map // id|pref -> *sync.Mutex
	cacheKeep = 150    // max cached files
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

func serveFromCache(w http.ResponseWriter, r *http.Request, id, pref string) {
	if streamCacheDir == "" {
		http.Error(w, "stream unavailable", 502)
		return
	}
	if cachedFile(id, pref) == "" {
		mu, _ := dlLocks.LoadOrStore(id+"|"+pref, &sync.Mutex{})
		l := mu.(*sync.Mutex)
		l.Lock()
		if cachedFile(id, pref) == "" {
			if err := fetchToCache(r.Context(), id, pref); err != nil {
				l.Unlock()
				log.Printf("stream cache %s: %v", id, err)
				http.Error(w, "stream unavailable", 502)
				return
			}
		}
		l.Unlock()
	}
	path := cachedFile(id, pref)
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, "stream unavailable", 502)
		return
	}
	defer f.Close()
	st, _ := f.Stat()
	ct := "audio/webm"
	if strings.HasSuffix(path, ".m4a") || strings.HasSuffix(path, ".mp4") {
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
	if pref == "aac" {
		format = "bestaudio[ext=m4a]/bestaudio/best"
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, "yt-dlp", "--ignore-config", "-f", format, "--no-playlist", "--no-warnings", "--no-progress",
		"-o", filepath.Join(streamCacheDir, id+"."+pref+".%(ext)s"),
		"https://music.youtube.com/watch?v="+id).CombinedOutput()
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
	if err != nil || len(entries) <= cacheKeep {
		return
	}
	type fi struct {
		path string
		mod  time.Time
	}
	var files []fi
	for _, e := range entries {
		if info, err := e.Info(); err == nil && !e.IsDir() {
			files = append(files, fi{filepath.Join(streamCacheDir, e.Name()), info.ModTime()})
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod.Before(files[j].mod) })
	for i := 0; i < len(files)-cacheKeep; i++ {
		os.Remove(files[i].path)
	}
}
