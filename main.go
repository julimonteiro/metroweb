package main

import (
	"embed"
	"encoding/json"
	"flag"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

//go:embed web
var webFS embed.FS

var idRe = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

func main() {
	addr := flag.String("addr", "127.0.0.1:8787", "listen address (127.0.0.1 = this machine only; use 0.0.0.0:8787 only behind HTTPS or a private network)")
	home, _ := os.UserHomeDir()
	libDir := flag.String("library", envOr("METROWEB_LIBRARY", filepath.Join(home, "Music", "metroweb")), "folder where songs are saved (the one Jellyfin watches)")
	jfURL := flag.String("jellyfin-url", os.Getenv("JELLYFIN_URL"), "e.g. http://localhost:8096 (optional, enables automatic library rescan)")
	jfKey := flag.String("jellyfin-key", os.Getenv("JELLYFIN_API_KEY"), "Jellyfin API key (Dashboard > API Keys)")
	user := flag.String("user", envOr("METROWEB_USER", "admin"), "initial username (first run only; change it in the app afterwards)")
	pass := flag.String("password", os.Getenv("METROWEB_PASSWORD"), "initial password (default admin; first run only)")
	dataDir := flag.String("data", envOr("METROWEB_DATA", filepath.Join(userConfigDir(home), "metroweb")), "server data folder (credentials, playlists, session secret)")
	trustProxy := flag.Bool("trust-proxy", os.Getenv("METROWEB_TRUST_PROXY") == "1", "trust X-Forwarded-For (behind a reverse proxy/tunnel)")
	genres := flag.Bool("genres", os.Getenv("METROWEB_GENRES") != "0", "look up artist genres on MusicBrainz for the stats page (sends artist names only)")
	lyrics := flag.Bool("lyrics", os.Getenv("METROWEB_LYRICS") != "0", "look up synced lyrics on LRCLIB for the full-screen player (sends title/artist only)")
	health := flag.Bool("healthcheck", false, "ask the running server if it is healthy, then exit (used by Docker)")
	flag.Parse()
	if *health {
		os.Exit(runHealthcheck(*addr))
	}

	lib := NewLibrary(*libDir, *jfURL, *jfKey)
	if err := lib.Check(); err != nil {
		log.Printf("downloads disabled until fixed: %v", err)
	}
	log.Printf("library: %s", *libDir)

	mux := http.NewServeMux()
	static, _ := fs.Sub(webFS, "web")
	loginPage, _ := fs.ReadFile(static, "login.html")
	mux.Handle("/", http.FileServer(http.FS(static)))

	mux.HandleFunc("/api/stream/", handleStream)

	playlists, perr := NewPlaylistStore(*dataDir)
	if perr != nil {
		log.Fatal(perr)
	}
	stats, serr := NewStatsStore(*dataDir, *genres)
	if serr != nil {
		log.Fatal(serr)
	}
	jf := NewJellyfinIndex(*jfURL, *jfKey)
	streamJF = jf
	if jf != nil {
		go jf.Run()
		lib.OnRescan = jf.Kick
	}
	ytdlpDir = filepath.Join(*dataDir, "ytdlp")
	ScheduleYtDlpUpdates()
	ScheduleBackups(*dataDir)
	registerOps(mux, lib, jf, stats, *dataDir)
	imp := NewImporter(jf, playlists, *dataDir)
	imp.Schedule()
	registerAPI(mux, lib, playlists, stats, jf, NewLyricsService(*lyrics), imp)
	mux.HandleFunc("/api/img", handleImg)

	streamCacheDir = filepath.Join(*dataDir, "cache")
	auth, err := NewAuth(*dataDir, *user, *pass, *trustProxy, loginPage)
	if err != nil {
		log.Fatal(err)
	}
	if c := auth.snapshot(); c.Default {
		log.Printf("WARNING: default login admin/admin in use; change it from the account menu in the app")
	}
	handler := auth.Middleware(mux)

	log.Printf("metroweb listening on %s", *addr)
	srv := &http.Server{Addr: *addr, Handler: logReq(handler), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second,
		IdleTimeout: 2 * time.Minute, MaxHeaderBytes: 16 << 10}
	log.Fatal(srv.ListenAndServe())
}

// handleStream proxies the googlevideo audio through us. googlevideo URLs are
// tied to the IP/UA that requested them and don't send CORS headers, so the
// browser can't reliably use them directly. Range requests are forwarded so
// seeking works.
func handleStream(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/stream/")
	if !idRe.MatchString(id) {
		http.Error(w, "bad id", 400)
		return
	}
	pref := r.URL.Query().Get("fmt")
	if pref != "aac" {
		pref = "opus"
	}
	q := r.URL.Query()

	// A song you already have in Jellyfin plays straight from your own library:
	// no YouTube round trip, so it starts instantly.
	if q.Get("nojf") != "1" && len(q.Get("t")) <= 200 && len(q.Get("a")) <= 200 {
		if jid, ok := streamJF.Find(q.Get("t"), q.Get("a")); ok && streamJF.Stream(w, r, jid) {
			return
		}
	}

	if _, bad := directFails.Load(id + "|" + pref); bad || directBroken() || (streamCacheDir != "" && cachedFile(id, pref) != "") {
		serveFromCache(w, r, id, pref)
		return
	}
	for attempt := 0; attempt < 2; attempt++ {
		s, err := ResolveStream(r.Context(), id, pref)
		if err != nil {
			log.Printf("resolve %s: %v", id, err)
			break
		}
		req, _ := http.NewRequestWithContext(r.Context(), "GET", s.URL, nil)
		for k, v := range s.Headers {
			req.Header.Set(k, v)
		}
		req.Header.Set("User-Agent", s.UserAgent)
		rng := r.Header.Get("Range")
		if rng == "" {
			rng = "bytes=0-"
		}
		req.Header.Set("Range", rng)

		resp, err := (&http.Client{}).Do(req) // no timeout: long-lived body
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		if resp.StatusCode == 403 {
			// URL expired or got rejected: drop cache and resolve again.
			resp.Body.Close()
			cache.mu.Lock()
			delete(cache.m, id+"|"+pref)
			cache.mu.Unlock()
			continue
		}
		defer resp.Body.Close()
		directStreak.Store(0) // direct proxying works on this network
		for _, h := range []string{"Content-Length", "Content-Range", "Accept-Ranges"} {
			if v := resp.Header.Get(h); v != "" {
				w.Header().Set(h, v)
			}
		}
		w.Header().Set("Content-Type", s.Mime)
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		return
	}
	// Direct proxying failed (googlevideo rejects URLs it only serves to yt-dlp's
	// own downloader, e.g. TLS-fingerprint checks). Let yt-dlp fetch the file into
	// a local cache and serve that, with full Range support.
	directFails.Store(id+"|"+pref, true)
	if directStreak.Add(1) >= 2 {
		directSkipUntil.Store(time.Now().Add(10 * time.Minute).UnixNano()) // stop wasting ~6 s per song on a path that fails here
	}
	serveFromCache(w, r, id, pref)
}

// streamJF serves songs from the Jellyfin library when they are in it (nil = off).
var streamJF *JellyfinIndex

// When direct proxying fails for two songs in a row (YouTube rejects the URLs
// from this network), skip it for a while and go straight to the cached download.
var (
	directStreak    atomic.Int32
	directSkipUntil atomic.Int64
)

func directBroken() bool {
	return directStreak.Load() >= 2 && time.Now().UnixNano() < directSkipUntil.Load()
}

// directFails remembers tracks whose googlevideo URL can't be proxied directly.
var directFails sync.Map

// handleImg proxies album art so the UI can read pixels (dynamic theme colors)
// without CORS issues. Only Google/YouTube image hosts are allowed.
func handleImg(w http.ResponseWriter, r *http.Request) {
	u, err := url.Parse(r.URL.Query().Get("u"))
	if err != nil || u.Scheme != "https" {
		http.Error(w, "bad url", 400)
		return
	}
	h := strings.ToLower(u.Hostname())
	okHost := false
	for _, suf := range []string{".googleusercontent.com", ".ggpht.com", ".ytimg.com"} {
		if strings.HasSuffix(h, suf) {
			okHost = true
		}
	}
	if !okHost {
		http.Error(w, "host not allowed", 400)
		return
	}
	req, _ := http.NewRequestWithContext(r.Context(), "GET", u.String(), nil)
	// Never follow redirects: an allowed host must not be able to bounce us to
	// an internal address.
	noRedirect := &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := noRedirect.Do(req)
	if err != nil {
		http.Error(w, "image unavailable", 502)
		return
	}
	defer resp.Body.Close()
	ct := resp.Header.Get("Content-Type")
	if resp.StatusCode != 200 || !strings.HasPrefix(ct, "image/") {
		http.Error(w, "upstream error", 502)
		return
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "private, max-age=604800")
	io.Copy(w, io.LimitReader(resp.Body, 5<<20))
}

func userConfigDir(home string) string {
	if d, err := os.UserConfigDir(); err == nil {
		return d
	}
	return filepath.Join(home, ".config")
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func logReq(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") && !strings.HasPrefix(r.URL.Path, "/api/stream/") {
			log.Printf("%s %s", r.Method, r.URL.RequestURI())
		}
		h.ServeHTTP(w, r)
	})
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
