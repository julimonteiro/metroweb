package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Play reports. Every counted play (30 s heard) can be sent, in the background,
// to ListenBrainz (open scrobbling service) and marked as played in Jellyfin for
// songs you own, so both keep your play counts. Both are optional and off
// until configured; a failed report never affects playback.

var scrobbler *Scrobbler

type Scrobbler struct {
	lbToken, lbBase string
	jf              *JellyfinIndex
	jfUser          string // Jellyfin user name; empty = the first user
	client          *http.Client
	queue           chan scrobbleJob

	mu                    sync.Mutex
	sent, failed, jfMarks int
	lastErr               string
}

type scrobbleJob struct {
	t  Track
	at time.Time
}

func NewScrobbler(lbToken string, jf *JellyfinIndex, jfUser string, markJellyfin bool) *Scrobbler {
	lbToken = strings.TrimSpace(lbToken)
	if !markJellyfin {
		jf = nil
	}
	if lbToken == "" && jf == nil {
		return nil
	}
	s := &Scrobbler{lbToken: lbToken, lbBase: "https://api.listenbrainz.org", jf: jf, jfUser: jfUser,
		client: &http.Client{Timeout: 20 * time.Second}, queue: make(chan scrobbleJob, 200)}
	go s.worker()
	return s
}

// Play queues a report; it never blocks.
func (s *Scrobbler) Play(t Track, at time.Time) {
	if s == nil {
		return
	}
	select {
	case s.queue <- scrobbleJob{t, at}:
	default: // queue full: drop rather than block playback
	}
}

func (s *Scrobbler) worker() {
	for job := range s.queue {
		s.handle(job)
	}
}

func (s *Scrobbler) handle(job scrobbleJob) {
	if s.lbToken != "" {
		err := withRetry(3, func() (bool, error) { return s.submitListenBrainz(job) })
		s.mu.Lock()
		if err != nil {
			s.failed++
			s.lastErr = "ListenBrainz: " + err.Error()
			log.Printf("listenbrainz: %v", err)
		} else {
			s.sent++
		}
		s.mu.Unlock()
	}
	if s.jf != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		ok, err := s.jf.MarkPlayed(ctx, job.t.Title, job.t.Artist, s.jfUser, job.at)
		cancel()
		s.mu.Lock()
		if err != nil {
			s.lastErr = "Jellyfin: " + err.Error()
			log.Printf("jellyfin played: %v", err)
		} else if ok {
			s.jfMarks++
		}
		s.mu.Unlock()
	}
}

// withRetry runs f up to n times with growing pauses; f says whether a failure is worth retrying.
func withRetry(n int, f func() (retry bool, err error)) error {
	var err error
	for i := 0; i < n; i++ {
		var retry bool
		if retry, err = f(); err == nil || !retry {
			return err
		}
		time.Sleep(time.Duration(i+1) * 2 * time.Second)
	}
	return err
}

func (s *Scrobbler) submitListenBrainz(job scrobbleJob) (bool, error) {
	body, _ := json.Marshal(map[string]any{
		"listen_type": "single",
		"payload": []map[string]any{{
			"listened_at": job.at.Unix(),
			"track_metadata": map[string]any{
				"artist_name":  primaryArtist(job.t),
				"track_name":   job.t.Title,
				"release_name": job.t.Album,
				"additional_info": map[string]any{
					"media_player":      "metroweb",
					"submission_client": "metroweb",
					"music_service":     "youtube.com",
					"origin_url":        "https://music.youtube.com/watch?v=" + job.t.ID,
				},
			},
		}},
	})
	req, _ := http.NewRequest("POST", s.lbBase+"/1/submit-listens", bytes.NewReader(body))
	req.Header.Set("Authorization", "Token "+s.lbToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return true, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		return false, nil
	}
	msg, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
	err = fmt.Errorf("HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	return resp.StatusCode >= 500 || resp.StatusCode == 429, err
}

type ScrobbleStatus struct {
	ListenBrainz bool   `json:"listenbrainz"`
	Jellyfin     bool   `json:"jellyfin"`
	Sent         int    `json:"sent"`
	Failed       int    `json:"failed"`
	JfMarked     int    `json:"jfMarked"`
	Error        string `json:"error,omitempty"`
}

func (s *Scrobbler) Status() ScrobbleStatus {
	if s == nil {
		return ScrobbleStatus{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return ScrobbleStatus{ListenBrainz: s.lbToken != "", Jellyfin: s.jf != nil, Sent: s.sent, Failed: s.failed, JfMarked: s.jfMarks, Error: s.lastErr}
}

// ---- Jellyfin: mark a song you own as played ----

// jfUserID finds the Jellyfin user plays are credited to.
func (j *JellyfinIndex) jfUserID(ctx context.Context, name string) (string, error) {
	j.mu.RLock()
	id := j.userID
	j.mu.RUnlock()
	if id != "" {
		return id, nil
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", j.base+"/Users", nil)
	req.Header.Set("Authorization", `MediaBrowser Token="`+j.key+`"`)
	resp, err := j.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Jellyfin users: HTTP %d", resp.StatusCode)
	}
	var users []struct {
		ID     string `json:"Id"`
		Name   string `json:"Name"`
		Policy struct {
			IsDisabled      bool `json:"IsDisabled"`
			IsAdministrator bool `json:"IsAdministrator"`
		} `json:"Policy"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&users); err != nil {
		return "", err
	}
	for _, u := range users {
		if u.Policy.IsDisabled {
			continue
		}
		if name == "" || strings.EqualFold(u.Name, name) {
			id = u.ID
			break
		}
	}
	if id == "" {
		return "", errors.New("Jellyfin user not found (set JELLYFIN_USER)")
	}
	j.mu.Lock()
	j.userID = id
	j.mu.Unlock()
	return id, nil
}

// MarkPlayed marks the song as played for the Jellyfin user (so its play count
// and "recently played" stay right). false, nil = the song is not in Jellyfin.
func (j *JellyfinIndex) MarkPlayed(ctx context.Context, title, artist, user string, at time.Time) (bool, error) {
	item, ok := j.Find(title, artist)
	if !ok {
		return false, nil
	}
	uid, err := j.jfUserID(ctx, user)
	if err != nil {
		return false, err
	}
	u := fmt.Sprintf("%s/Users/%s/PlayedItems/%s?datePlayed=%s", j.base, url.PathEscape(uid), url.PathEscape(item), url.QueryEscape(at.UTC().Format("2006-01-02T15:04:05Z")))
	req, _ := http.NewRequestWithContext(ctx, "POST", u, nil)
	req.Header.Set("Authorization", `MediaBrowser Token="`+j.key+`"`)
	resp, err := j.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode/100 != 2 {
		return false, fmt.Errorf("Jellyfin: HTTP %d", resp.StatusCode)
	}
	return true, nil
}
