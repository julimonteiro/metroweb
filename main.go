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
	"time"
)

//go:embed web
var webFS embed.FS

var idRe = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

func main() {
	addr := flag.String("addr", ":8080", "listen address (use 0.0.0.0:8080 to reach it from your phone on the LAN)")
	home, _ := os.UserHomeDir()
	libDir := flag.String("library", envOr("METROWEB_LIBRARY", filepath.Join(home, "Music", "metroweb")), "pasta onde salvar as músicas (a que o Jellyfin monitora)")
	jfURL := flag.String("jellyfin-url", os.Getenv("JELLYFIN_URL"), "ex: http://localhost:8096 (opcional, pra rescan automático)")
	jfKey := flag.String("jellyfin-key", os.Getenv("JELLYFIN_API_KEY"), "API key do Jellyfin (Painel > Chaves de API)")
	user := flag.String("user", envOr("METROWEB_USER", "admin"), "usuário inicial (só vale na primeira execução; depois troque pelo app)")
	pass := flag.String("password", os.Getenv("METROWEB_PASSWORD"), "senha inicial (padrão admin; só vale na primeira execução)")
	dataDir := flag.String("data", envOr("METROWEB_DATA", filepath.Join(userConfigDir(home), "metroweb")), "pasta de dados do servidor (segredo da sessão)")
	trustProxy := flag.Bool("trust-proxy", os.Getenv("METROWEB_TRUST_PROXY") == "1", "confiar em X-Forwarded-For (atrás de proxy reverso/túnel)")
	flag.Parse()

	lib := NewLibrary(*libDir, *jfURL, *jfKey)
	if err := lib.Check(); err != nil {
		log.Printf("downloads desativados até resolver: %v", err)
	}
	log.Printf("biblioteca: %s", *libDir)

	mux := http.NewServeMux()
	static, _ := fs.Sub(webFS, "web")
	loginPage, _ := fs.ReadFile(static, "login.html")
	mux.Handle("/", http.FileServer(http.FS(static)))

	mux.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if q == "" {
			writeJSON(w, 400, map[string]string{"error": "missing q"})
			return
		}
		tracks, err := Search(r.Context(), q)
		if err != nil {
			log.Printf("search %q: %v", q, err)
			writeJSON(w, 502, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, tracks)
	})

	mux.HandleFunc("/api/radio/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/radio/")
		if !idRe.MatchString(id) {
			writeJSON(w, 400, map[string]string{"error": "bad id"})
			return
		}
		tracks, err := Radio(r.Context(), id)
		if err != nil {
			log.Printf("radio %s: %v", id, err)
			writeJSON(w, 502, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, tracks)
	})

	mux.HandleFunc("/api/stream/", handleStream)

	mux.HandleFunc("/api/download/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/download/")
		if r.Method != "POST" || !idRe.MatchString(id) {
			writeJSON(w, 400, map[string]string{"error": "POST /api/download/{id}"})
			return
		}
		if err := lib.Check(); err != nil {
			writeJSON(w, 503, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]string{"status": lib.Enqueue(id)})
	})
	mux.HandleFunc("/api/downloads", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, lib.Status())
	})

	mux.HandleFunc("/api/img", handleImg)

	auth, err := NewAuth(*dataDir, *user, *pass, *trustProxy, loginPage)
	if err != nil {
		log.Fatal(err)
	}
	if c := auth.snapshot(); c.Default {
		log.Printf("AVISO: login padrão admin/admin em uso; troque pelo menu de conta no app")
	}
	handler := auth.Middleware(mux)

	log.Printf("metroweb ouvindo em %s", *addr)
	srv := &http.Server{Addr: *addr, Handler: logReq(handler), ReadHeaderTimeout: 10 * time.Second}
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

	for attempt := 0; attempt < 2; attempt++ {
		s, err := ResolveStream(r.Context(), id, pref)
		if err != nil {
			log.Printf("resolve %s: %v", id, err)
			http.Error(w, err.Error(), 502)
			return
		}
		req, _ := http.NewRequestWithContext(r.Context(), "GET", s.URL, nil)
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
		if resp.StatusCode == 403 && attempt == 0 {
			// URL expired or got rejected: drop cache and resolve again.
			resp.Body.Close()
			cache.mu.Lock()
			delete(cache.m, id+"|"+pref)
			cache.mu.Unlock()
			continue
		}
		defer resp.Body.Close()
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
	http.Error(w, "upstream rejected stream", 502)
}

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
	resp, err := httpClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), 502)
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
