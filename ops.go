package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Operations: health check, status page data, yt-dlp update and backups.

var startedAt = time.Now()

type DiskInfo struct {
	Free  uint64 `json:"free"`
	Total uint64 `json:"total"`
}

type ServerStatus struct {
	Uptime   int64          `json:"uptime"`
	Go       string         `json:"go"`
	YtDlp    YtDlpStatus    `json:"ytdlp"`
	FFmpeg   string         `json:"ffmpeg"`
	Jellyfin JellyfinStatus `json:"jellyfin"`
	Library  DiskInfo       `json:"library"`
	Data     DiskInfo       `json:"data"`
	CacheB   int64          `json:"cacheBytes"`
	CacheN   int            `json:"cacheFiles"`
	Plays    int            `json:"plays"`
	Backups  []BackupInfo   `json:"backups"`
}

func diskInfo(path string) DiskInfo {
	var s syscall.Statfs_t
	if syscall.Statfs(path, &s) != nil {
		return DiskInfo{}
	}
	return DiskInfo{Free: uint64(s.Bavail) * uint64(s.Bsize), Total: uint64(s.Blocks) * uint64(s.Bsize)}
}

var (
	ffOnce sync.Once
	ffVer  string
)

func ffmpegVersion() string {
	ffOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, "ffmpeg", "-version").Output()
		if err == nil {
			f := strings.Fields(strings.SplitN(string(out), "\n", 2)[0])
			if len(f) >= 3 {
				ffVer = f[2]
			}
		}
	})
	return ffVer
}

func dirSize(dir string) (int64, int) {
	var n int64
	var c int
	filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if fi, e := d.Info(); e == nil {
				n += fi.Size()
				c++
			}
		}
		return nil
	})
	return n, c
}

func registerOps(mux *http.ServeMux, lib *Library, jf *JellyfinIndex, st *StatsStore, dataDir string) {
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		io.WriteString(w, "ok")
	})
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		s := ServerStatus{Uptime: int64(time.Since(startedAt).Seconds()), Go: runtime.Version(), YtDlp: ytdlpStatus(), FFmpeg: ffmpegVersion(),
			Jellyfin: jf.Status(), Library: diskInfo(lib.Dir), Data: diskInfo(dataDir), Plays: st.Count(), Backups: ListBackups(dataDir)}
		if streamCacheDir != "" {
			s.CacheB, s.CacheN = dirSize(streamCacheDir)
		}
		if s.Backups == nil {
			s.Backups = []BackupInfo{}
		}
		writeJSON(w, 200, s)
	})
	mux.HandleFunc("/api/status/update-ytdlp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			apiErr(w, 405, "method not allowed")
			return
		}
		if ytdlpDir == "" {
			apiErr(w, 400, "updates are not available here")
			return
		}
		go func() {
			if err := UpdateYtDlp(context.Background()); err != nil {
				fmt.Fprintf(os.Stderr, "yt-dlp update: %v\n", err)
			}
		}()
		writeJSON(w, 202, map[string]bool{"started": true})
	})
	mux.HandleFunc("/api/backup/now", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			apiErr(w, 405, "method not allowed")
			return
		}
		if _, err := SaveBackup(dataDir, time.Now()); err != nil {
			apiErr(w, 500, "could not save the backup: "+err.Error())
			return
		}
		writeJSON(w, 200, ListBackups(dataDir))
	})
	mux.HandleFunc("/api/backup/download", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", `attachment; filename="metroweb-backup-`+time.Now().Format("2006-01-02")+`.tar.gz"`)
		w.Header().Set("Cache-Control", "no-store")
		WriteBackup(w, dataDir)
	})
}

// runHealthcheck is what `metroweb -healthcheck` does: ask the running server.
func runHealthcheck(addr string) int {
	c := http.Client{Timeout: 4 * time.Second}
	resp, err := c.Get("http://" + addr + "/healthz")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return 1
	}
	return 0
}
