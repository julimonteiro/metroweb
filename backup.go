package main

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Backups: your playlists, listening history and sync state, packed into a
// .tar.gz. A copy is kept in <data>/backups every day (the last 14). Logins and
// the session secret are deliberately left out.

var backupFiles = []string{"playlists.json", "liked-suppressed.json", "plays.jsonl", "genres.json", "import-state.json"}

const backupKeep = 14

// WriteBackup streams the archive to w.
func WriteBackup(w io.Writer, dir string) error {
	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)
	for _, name := range backupFiles {
		f, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		fi, err := f.Stat()
		if err != nil {
			f.Close()
			continue
		}
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o600, Size: fi.Size(), ModTime: fi.ModTime()}); err != nil {
			f.Close()
			return err
		}
		_, err = io.Copy(tw, f)
		f.Close()
		if err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

// SaveBackup writes a dated archive into <dir>/backups and prunes old ones.
func SaveBackup(dir string, now time.Time) (string, error) {
	bdir := filepath.Join(dir, "backups")
	if err := os.MkdirAll(bdir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(bdir, "metroweb-"+now.Format("20060102-1504")+".tar.gz")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", err
	}
	if err := WriteBackup(f, dir); err != nil {
		f.Close()
		os.Remove(path)
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	old := ListBackups(dir)
	for i := backupKeep; i < len(old); i++ {
		os.Remove(filepath.Join(bdir, old[i].Name))
	}
	return path, nil
}

type BackupInfo struct {
	Name string `json:"name"`
	Time int64  `json:"time"`
	Size int64  `json:"size"`
}

// ListBackups returns the stored backups, newest first.
func ListBackups(dir string) []BackupInfo {
	ents, _ := os.ReadDir(filepath.Join(dir, "backups"))
	var out []BackupInfo
	for _, e := range ents {
		if !strings.HasSuffix(e.Name(), ".tar.gz") {
			continue
		}
		if fi, err := e.Info(); err == nil {
			out = append(out, BackupInfo{Name: e.Name(), Time: fi.ModTime().Unix(), Size: fi.Size()})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name > out[j].Name })
	return out
}

// ScheduleBackups makes a backup five minutes after start-up when the last one
// is older than a day, then checks every hour.
func ScheduleBackups(dir string) {
	go func() {
		time.Sleep(5 * time.Minute)
		for {
			if l := ListBackups(dir); len(l) == 0 || time.Since(time.Unix(l[0].Time, 0)) > 24*time.Hour {
				if _, err := SaveBackup(dir, time.Now()); err != nil {
					log.Printf("backup: %v", err)
				}
			}
			time.Sleep(time.Hour)
		}
	}()
}
