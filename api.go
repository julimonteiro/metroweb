package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var browseIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]{4,80}$`)

func apiErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// call runs a lookup with a timeout and writes the JSON result (or a 502).
func call[T any](w http.ResponseWriter, r *http.Request, what string, fn func(ctx context.Context) (T, error)) {
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	v, err := fn(ctx)
	if err != nil {
		log.Printf("%s: %v", what, err)
		apiErr(w, 502, err.Error())
		return
	}
	writeJSON(w, 200, v)
}

func registerAPI(mux *http.ServeMux, lib *Library, pls *PlaylistStore) {
	mux.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if q == "" || len(q) > 200 {
			apiErr(w, 400, "missing or invalid q")
			return
		}
		kind := r.URL.Query().Get("type")
		call(w, r, "search "+q, func(ctx context.Context) ([]Item, error) {
			items, err := Search(ctx, q, kind)
			if items == nil {
				items = []Item{}
			}
			return items, err
		})
	})

	mux.HandleFunc("/api/home", func(w http.ResponseWriter, r *http.Request) {
		call(w, r, "home", func(ctx context.Context) ([]Section, error) { return Home(ctx) })
	})

	mux.HandleFunc("/api/artist/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/artist/")
		if !browseIDRe.MatchString(id) {
			apiErr(w, 400, "bad id")
			return
		}
		call(w, r, "artist "+id, func(ctx context.Context) (Artist, error) { return GetArtist(ctx, id) })
	})

	mux.HandleFunc("/api/songs", func(w http.ResponseWriter, r *http.Request) {
		b := r.URL.Query().Get("b")
		if !browseIDRe.MatchString(b) {
			apiErr(w, 400, "bad browse id")
			return
		}
		call(w, r, "songs "+b, func(ctx context.Context) ([]Item, error) {
			s, err := BrowseSongs(ctx, b, r.URL.Query().Get("p"))
			if s == nil {
				s = []Item{}
			}
			return s, err
		})
	})

	mux.HandleFunc("/api/album/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/album/")
		if !browseIDRe.MatchString(id) {
			apiErr(w, 400, "bad id")
			return
		}
		call(w, r, "album "+id, func(ctx context.Context) (Album, error) { return GetAlbum(ctx, id) })
	})

	mux.HandleFunc("/api/radio/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/radio/")
		if !idRe.MatchString(id) {
			apiErr(w, 400, "bad id")
			return
		}
		call(w, r, "radio "+id, func(ctx context.Context) ([]Track, error) { return Radio(ctx, id) })
	})

	// Radio from a playlist id (artist mix, album radio, ...).
	mux.HandleFunc("/api/mix", func(w http.ResponseWriter, r *http.Request) {
		pl := r.URL.Query().Get("id")
		if !browseIDRe.MatchString(pl) {
			apiErr(w, 400, "bad id")
			return
		}
		call(w, r, "mix "+pl, func(ctx context.Context) ([]Track, error) {
			return RadioFor(ctx, r.URL.Query().Get("v"), pl, r.URL.Query().Get("p"))
		})
	})

	mux.HandleFunc("/api/track/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/track/")
		if !idRe.MatchString(id) {
			apiErr(w, 400, "bad id")
			return
		}
		call(w, r, "track "+id, func(ctx context.Context) (Track, error) { return TrackInfo(ctx, id) })
	})

	// --- downloads ---
	mux.HandleFunc("/api/download/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/download/")
		if r.Method != "POST" || !idRe.MatchString(id) {
			apiErr(w, 400, "POST /api/download/{id}")
			return
		}
		if err := lib.Check(); err != nil {
			apiErr(w, 503, err.Error())
			return
		}
		writeJSON(w, 200, map[string]string{"status": lib.Enqueue(id)})
	})
	mux.HandleFunc("/api/downloads", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, lib.Status())
	})

	// --- playlists (and "Liked songs") ---
	mux.HandleFunc("/api/playlists", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, 200, pls.List())
		case http.MethodPost:
			var in struct {
				Name   string  `json:"name"`
				Tracks []Track `json:"tracks"`
			}
			if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&in) != nil {
				apiErr(w, 400, "invalid request")
				return
			}
			p, err := pls.Create(in.Name)
			if err != nil {
				apiErr(w, 400, err.Error())
				return
			}
			if len(in.Tracks) > 0 {
				pls.AddTracks(p.ID, in.Tracks)
				p.Count = len(in.Tracks)
			}
			writeJSON(w, 200, p)
		default:
			apiErr(w, 405, "method not allowed")
		}
	})

	// /api/playlists/{id}, /api/playlists/{id}/tracks, /api/playlists/{id}/tracks/{trackId}
	mux.HandleFunc("/api/playlists/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/playlists/"), "/"), "/")
		id := parts[0]
		if id == "" {
			apiErr(w, 400, "missing id")
			return
		}
		switch {
		case len(parts) == 1 && r.Method == http.MethodGet:
			p, ok := pls.Get(id)
			if !ok {
				apiErr(w, 404, "playlist not found")
				return
			}
			writeJSON(w, 200, p)
		case len(parts) == 1 && r.Method == http.MethodPatch:
			var in struct {
				Name string `json:"name"`
			}
			if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil {
				apiErr(w, 400, "invalid request")
				return
			}
			if err := pls.Rename(id, in.Name); err != nil {
				apiErr(w, 400, err.Error())
				return
			}
			writeJSON(w, 200, map[string]string{"ok": "1"})
		case len(parts) == 1 && r.Method == http.MethodDelete:
			if err := pls.Delete(id); err != nil {
				apiErr(w, 400, err.Error())
				return
			}
			writeJSON(w, 200, map[string]string{"ok": "1"})
		case len(parts) == 2 && parts[1] == "tracks" && r.Method == http.MethodPost:
			var in struct {
				Tracks []Track `json:"tracks"`
			}
			if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&in) != nil {
				apiErr(w, 400, "invalid request")
				return
			}
			n, err := pls.AddTracks(id, in.Tracks)
			if err != nil {
				apiErr(w, 404, err.Error())
				return
			}
			writeJSON(w, 200, map[string]int{"added": n})
		case len(parts) == 3 && parts[1] == "tracks" && r.Method == http.MethodDelete:
			if err := pls.RemoveTrack(id, parts[2]); err != nil {
				apiErr(w, 404, err.Error())
				return
			}
			writeJSON(w, 200, map[string]string{"ok": "1"})
		default:
			apiErr(w, 405, "method not allowed")
		}
	})
}
