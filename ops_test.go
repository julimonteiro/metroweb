package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fakeInstall(t *testing.T, content string) func(ctx context.Context, dir string) error {
	return func(ctx context.Context, dir string) error {
		if err := os.MkdirAll(filepath.Join(dir, "yt_dlp"), 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, "yt_dlp", "__init__.py"), []byte(content), 0o644)
	}
}

func TestYtdlpCommandPrefersSelfCopy(t *testing.T) {
	old := ytdlpDir
	defer func() { ytdlpDir = old }()
	ytdlpDir = filepath.Join(t.TempDir(), "ytdlp")
	if got := ytdlpCommand(context.Background(), "--version").Args[0]; got != "yt-dlp" {
		t.Fatalf("without a self copy want yt-dlp, got %s", got)
	}
	os.MkdirAll(filepath.Join(ytdlpDir, "yt_dlp"), 0o755)
	os.WriteFile(filepath.Join(ytdlpDir, "yt_dlp", "__init__.py"), nil, 0o644)
	c := ytdlpCommand(context.Background(), "--version")
	if c.Args[0] != "python3" || c.Args[1] != "-m" || c.Args[2] != "yt_dlp" {
		t.Fatalf("self copy should run through python3 -m yt_dlp, got %v", c.Args)
	}
	found := false
	for _, e := range c.Env {
		if e == "PYTHONPATH="+ytdlpDir {
			found = true
		}
	}
	if !found {
		t.Fatal("PYTHONPATH not set")
	}
}

func TestUpdateYtDlpSwapsOnlyOnSuccess(t *testing.T) {
	oldDir, oldRun := ytdlpDir, runPipInstall
	defer func() { ytdlpDir, runPipInstall = oldDir, oldRun }()
	ytdlpDir = filepath.Join(t.TempDir(), "ytdlp")
	runPipInstall = fakeInstall(t, "v1")
	if err := UpdateYtDlp(context.Background()); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(ytdlpDir, "yt_dlp", "__init__.py"))
	if string(b) != "v1" {
		t.Fatalf("first install missing: %q", b)
	}
	runPipInstall = func(ctx context.Context, dir string) error { return errors.New("no network") }
	if err := UpdateYtDlp(context.Background()); err == nil {
		t.Fatal("expected failure")
	}
	b, _ = os.ReadFile(filepath.Join(ytdlpDir, "yt_dlp", "__init__.py"))
	if string(b) != "v1" {
		t.Fatalf("a failed update must keep the working copy, got %q", b)
	}
	if _, err := os.Stat(ytdlpDir + ".new"); err == nil {
		t.Fatal("leftover .new folder")
	}
	if st := ytdlpStatus(); st.Error == "" {
		t.Fatal("failure should be reported in the status")
	}
	runPipInstall = fakeInstall(t, "v2")
	if err := UpdateYtDlp(context.Background()); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(filepath.Join(ytdlpDir, "yt_dlp", "__init__.py"))
	if string(b) != "v2" {
		t.Fatalf("update not applied: %q", b)
	}
	if st := ytdlpStatus(); st.Error != "" || st.Updated == 0 {
		t.Fatalf("status after success: %+v", st)
	}
}

func TestBackupRoundTripAndPrune(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "playlists.json"), []byte(`{"a":1}`), 0o600)
	os.WriteFile(filepath.Join(dir, "plays.jsonl"), []byte("x\n"), 0o600)
	os.WriteFile(filepath.Join(dir, "credentials.json"), []byte("SECRET"), 0o600)
	os.WriteFile(filepath.Join(dir, "session-secret"), []byte("SECRET"), 0o600)
	var buf bytes.Buffer
	if err := WriteBackup(&buf, dir); err != nil {
		t.Fatal(err)
	}
	gz, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	got := map[string]string{}
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(tr)
		got[h.Name] = string(b)
	}
	if got["playlists.json"] != `{"a":1}` || got["plays.jsonl"] != "x\n" {
		t.Fatalf("archive content wrong: %v", got)
	}
	for name, c := range got {
		if strings.Contains(c, "SECRET") || name == "credentials.json" || name == "session-secret" {
			t.Fatalf("secrets must not be in the backup: %s", name)
		}
	}
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	for i := 0; i < backupKeep+4; i++ {
		if _, err := SaveBackup(dir, base.Add(time.Duration(i)*24*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	l := ListBackups(dir)
	if len(l) != backupKeep {
		t.Fatalf("kept %d backups, want %d", len(l), backupKeep)
	}
	if !strings.Contains(l[0].Name, "20260118") { // newest: day 17 after base (Jan 1 + 17)
		t.Fatalf("newest backup should come first: %s", l[0].Name)
	}
}

func TestHealthz(t *testing.T) {
	mux := http.NewServeMux()
	registerOps(mux, NewLibrary(t.TempDir(), "", ""), nil, mustStats(t), t.TempDir())
	srv := httptest.NewServer(mux)
	defer srv.Close()
	if got := runHealthcheck(strings.TrimPrefix(srv.URL, "http://")); got != 0 {
		t.Fatalf("healthcheck = %d", got)
	}
	if got := runHealthcheck("127.0.0.1:1"); got != 1 {
		t.Fatalf("healthcheck on a dead port = %d", got)
	}
	if !publicPath("/healthz") {
		t.Fatal("/healthz must be reachable without a login")
	}
}

func mustStats(t *testing.T) *StatsStore {
	s, err := NewStatsStore(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
