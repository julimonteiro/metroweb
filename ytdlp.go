package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// yt-dlp has to stay fresh: YouTube breaks versions that are a few months old.
// The image ships one, but its filesystem is read-only, so the server keeps its
// own, newer copy in <data>/ytdlp (installed with pip --target) and prefers it.
// It checks once a day, and the Status page can trigger it.

var ytdlpDir string // <data>/ytdlp; empty = always use the yt-dlp on PATH

type ytdlpState struct {
	mu        sync.Mutex
	version   string
	versionAt time.Time
	checkedAt time.Time
	updatedAt time.Time
	lastErr   string
	running   bool
}

var ytState ytdlpState

// runPipInstall installs yt-dlp into dir. A variable so tests can fake it.
var runPipInstall = func(ctx context.Context, dir string) error {
	cmd := exec.CommandContext(ctx, "python3", "-m", "pip", "install", "--upgrade", "--no-cache-dir", "--quiet",
		"--disable-pip-version-check", "--target", dir, "yt-dlp[default]")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if len(msg) > 300 {
			msg = msg[len(msg)-300:]
		}
		return fmt.Errorf("pip: %v: %s", err, msg)
	}
	return nil
}

// ytdlpCommand builds a yt-dlp command, preferring the self-updated copy.
func ytdlpCommand(ctx context.Context, args ...string) *exec.Cmd {
	if ytdlpDir != "" {
		if _, err := os.Stat(filepath.Join(ytdlpDir, "yt_dlp", "__init__.py")); err == nil {
			cmd := exec.CommandContext(ctx, "python3", append([]string{"-m", "yt_dlp"}, args...)...)
			cmd.Env = append(os.Environ(), "PYTHONPATH="+ytdlpDir)
			return cmd
		}
	}
	return exec.CommandContext(ctx, "yt-dlp", args...)
}

// ytdlpVersion reports the version in use (cached for a few minutes).
func ytdlpVersion() string {
	ytState.mu.Lock()
	if ytState.version != "" && time.Since(ytState.versionAt) < 10*time.Minute {
		v := ytState.version
		ytState.mu.Unlock()
		return v
	}
	ytState.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	out, err := ytdlpCommand(ctx, "--version").Output()
	v := strings.TrimSpace(string(out))
	if err != nil || v == "" {
		return ""
	}
	ytState.mu.Lock()
	ytState.version, ytState.versionAt = v, time.Now()
	ytState.mu.Unlock()
	return v
}

// UpdateYtDlp installs the latest yt-dlp into a fresh folder and swaps it in
// only once it runs, so a failed update never leaves a broken copy behind.
func UpdateYtDlp(ctx context.Context) error {
	if ytdlpDir == "" {
		return errors.New("no data folder")
	}
	ytState.mu.Lock()
	if ytState.running {
		ytState.mu.Unlock()
		return errors.New("an update is already running")
	}
	ytState.running = true
	ytState.mu.Unlock()
	defer func() { ytState.mu.Lock(); ytState.running = false; ytState.mu.Unlock() }()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	next := ytdlpDir + ".new"
	os.RemoveAll(next)
	err := runPipInstall(ctx, next)
	if err == nil {
		if _, serr := os.Stat(filepath.Join(next, "yt_dlp", "__init__.py")); serr != nil {
			err = errors.New("update produced no yt_dlp package")
		}
	}
	ytState.mu.Lock()
	ytState.checkedAt = time.Now()
	ytState.mu.Unlock()
	if err != nil {
		os.RemoveAll(next)
		ytState.mu.Lock()
		ytState.lastErr = err.Error()
		ytState.mu.Unlock()
		return err
	}
	old := ytdlpDir + ".old"
	os.RemoveAll(old)
	os.Rename(ytdlpDir, old)
	if rerr := os.Rename(next, ytdlpDir); rerr != nil {
		os.Rename(old, ytdlpDir)
		ytState.mu.Lock()
		ytState.lastErr = rerr.Error()
		ytState.mu.Unlock()
		return rerr
	}
	os.RemoveAll(old)
	ytState.mu.Lock()
	ytState.lastErr, ytState.updatedAt, ytState.version = "", time.Now(), ""
	ytState.mu.Unlock()
	log.Printf("yt-dlp updated to %s", ytdlpVersion())
	return nil
}

// ScheduleYtDlpUpdates checks shortly after start-up and then every day.
func ScheduleYtDlpUpdates() {
	if ytdlpDir == "" {
		return
	}
	go func() {
		time.Sleep(2 * time.Minute)
		for {
			if err := UpdateYtDlp(context.Background()); err != nil {
				log.Printf("yt-dlp update: %v", err)
			}
			time.Sleep(24 * time.Hour)
		}
	}()
}

type YtDlpStatus struct {
	Version   string `json:"version"`
	Updated   int64  `json:"updated,omitempty"`
	Checked   int64  `json:"checked,omitempty"`
	Error     string `json:"error,omitempty"`
	Running   bool   `json:"running"`
	SelfCopy  bool   `json:"selfCopy"`
	CanUpdate bool   `json:"canUpdate"`
}

func ytdlpStatus() YtDlpStatus {
	v := ytdlpVersion()
	ytState.mu.Lock()
	defer ytState.mu.Unlock()
	st := YtDlpStatus{Version: v, Error: ytState.lastErr, Running: ytState.running, CanUpdate: ytdlpDir != ""}
	if !ytState.updatedAt.IsZero() {
		st.Updated = ytState.updatedAt.Unix()
	}
	if !ytState.checkedAt.IsZero() {
		st.Checked = ytState.checkedAt.Unix()
	}
	if ytdlpDir != "" {
		_, err := os.Stat(filepath.Join(ytdlpDir, "yt_dlp", "__init__.py"))
		st.SelfCopy = err == nil
	}
	return st
}
