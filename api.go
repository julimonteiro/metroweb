package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var playlistIDRe = regexp.MustCompile(`^(liked|[0-9a-f]{12})$`)

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

func registerAPI(mux *http.ServeMux, lib *Library, pls *PlaylistStore, st *StatsStore, jf *JellyfinIndex, ly *LyricsService, im *Importer) {
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
	// POST /api/download/{id} with an optional body {"title","artist","force"}.
	// If the song is already in the Jellyfin library it is not downloaded again
	// (unless force is set) and the answer is {"status":"owned"}.
	mux.HandleFunc("/api/download/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/download/")
		if r.Method != "POST" || !idRe.MatchString(id) {
			apiErr(w, 400, "POST /api/download/{id}")
			return
		}
		var in struct {
			Title  string `json:"title"`
			Artist string `json:"artist"`
			Force  bool   `json:"force"`
		}
		if r.ContentLength != 0 {
			json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in)
		}
		if !in.Force && jf.Has(in.Title, in.Artist) {
			writeJSON(w, 200, map[string]string{"status": "owned"})
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

	mux.HandleFunc("/api/prefetch/", prefetchHandler)

	// --- lyrics and music videos (full-screen player only) ---
	mux.HandleFunc("/api/lyrics", lyricsHandler(ly))
	mux.HandleFunc("/api/clip/info", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		title, artist := strings.TrimSpace(q.Get("title")), strings.TrimSpace(q.Get("artist"))
		if title == "" || artist == "" || len(title) > 200 || len(artist) > 200 {
			apiErr(w, 400, "title and artist are required")
			return
		}
		call(w, r, "clip "+title, func(ctx context.Context) (map[string]string, error) {
			id, err := FindClip(ctx, title, artist, durationFromParam(q.Get("duration")))
			if id != "" {
				allowClip(id)
			}
			return map[string]string{"videoId": id}, err
		})
	})
	// How far the clip is shifted against the song (seconds), so the app can show
	// the video in step with the song that is already playing.
	mux.HandleFunc("/api/clip/offset", func(w http.ResponseWriter, r *http.Request) {
		song, clip := r.URL.Query().Get("song"), r.URL.Query().Get("clip")
		if !idRe.MatchString(song) || !idRe.MatchString(clip) || !clipAllowed(clip) {
			apiErr(w, 400, "unknown clip")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 6*time.Minute)
		defer cancel()
		res, err := MeasureClipOffset(ctx, song, clip)
		if err != nil {
			log.Printf("clip offset %s/%s: %v", song, clip, err)
			apiErr(w, 502, "could not align the video")
			return
		}
		writeJSON(w, 200, res)
	})
	// The clip file itself. Only ids that /api/clip/info vouched for are served,
	// so this can't be used to fetch arbitrary videos onto the server.
	mux.HandleFunc("/api/clip/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/clip/")
		if !idRe.MatchString(id) || !clipAllowed(id) {
			apiErr(w, 404, "unknown clip")
			return
		}
		serveFromCache(w, r, id, "video")
	})

	// --- Jellyfin library awareness ---
	mux.HandleFunc("/api/library/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, jf.Status())
	})
	// Import the whole Jellyfin library into "Liked songs" (background job).
	mux.HandleFunc("/api/library/import", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, 200, im.Status())
		case http.MethodPost:
			if err := im.Start(); err != nil {
				apiErr(w, 409, err.Error())
				return
			}
			writeJSON(w, 200, im.Status())
		default:
			apiErr(w, 405, "method not allowed")
		}
	})
	// POST {"tracks":[{"id","title","artist"}]} -> {"owned":[ids]}
	mux.HandleFunc("/api/library/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			apiErr(w, 405, "method not allowed")
			return
		}
		var in struct {
			Tracks []Track `json:"tracks"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 512<<10)).Decode(&in) != nil || len(in.Tracks) > 500 {
			apiErr(w, 400, "invalid request")
			return
		}
		owned := []string{}
		var ownedTracks []Track
		for _, t := range in.Tracks {
			if idRe.MatchString(t.ID) && jf.Has(t.Title, t.Artist) {
				owned = append(owned, t.ID)
				ownedTracks = append(ownedTracks, t)
			}
		}
		// Songs that are in the Jellyfin library count as liked, automatically.
		liked := pls.AutoLike(ownedTracks)
		if liked == nil {
			liked = []string{}
		}
		writeJSON(w, 200, map[string]any{"enabled": jf != nil, "owned": owned, "liked": liked})
	})

	// --- listening stats ---
	mux.HandleFunc("/api/plays", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			apiErr(w, 405, "method not allowed")
			return
		}
		var in struct {
			Track Track `json:"track"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&in) != nil {
			apiErr(w, 400, "invalid request")
			return
		}
		counted, err := st.Record(in.Track, time.Now())
		if err != nil && !counted {
			apiErr(w, 400, err.Error())
			return
		}
		writeJSON(w, 200, map[string]bool{"counted": counted})
	})
	mux.HandleFunc("/api/plays/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, st.Summary())
	})
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		tz, _ := strconv.Atoi(r.URL.Query().Get("tz")) // minutes east of UTC
		if tz < -14*60 || tz > 14*60 {
			tz = 0
		}
		writeJSON(w, 200, st.Stats(r.URL.Query().Get("range"), tz, time.Now()))
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
		if !playlistIDRe.MatchString(id) {
			apiErr(w, 400, "bad playlist id")
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
			if len(in.Tracks) > 1000 {
				apiErr(w, 400, "too many tracks in one request")
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
			if id == likedID {
				pls.Suppress(parts[2]) // an un-like is respected by automatic liking
			}
			writeJSON(w, 200, map[string]string{"ok": "1"})
		default:
			apiErr(w, 405, "method not allowed")
		}
	})
}

// clips that /api/clip/info found recently (id -> expiry)
var allowedClips sync.Map

func allowClip(id string) { allowedClips.Store(id, time.Now().Add(12*time.Hour)) }
func clipAllowed(id string) bool {
	v, ok := allowedClips.Load(id)
	return ok && time.Now().Before(v.(time.Time))
}
