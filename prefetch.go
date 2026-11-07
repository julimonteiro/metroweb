package main

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	prefetching   sync.Map                 // id|pref -> true while a prefetch runs
	prefetchSlots = make(chan struct{}, 2) // at most two background prefetches
)

// prefetch warms up the next song of the queue so it starts without a gap:
// nothing to do for songs served from Jellyfin; otherwise either resolve (and
// cache) the stream URL or, where direct streaming doesn't work, download the
// file into the cache.
func prefetch(id, pref, title, artist string) {
	if _, ok := streamJF.Find(title, artist); ok {
		return
	}
	key := id + "|" + pref
	if _, busy := prefetching.LoadOrStore(key, true); busy {
		return
	}
	defer prefetching.Delete(key)
	select {
	case prefetchSlots <- struct{}{}:
		defer func() { <-prefetchSlots }()
	default:
		return
	}
	if streamCacheDir != "" && cachedFile(id, pref) != "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if _, bad := directFails.Load(key); bad || directBroken() {
		ensureCached(ctx, id, pref)
		return
	}
	ResolveStream(ctx, id, pref)
}

func prefetchHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/prefetch/")
	if !idRe.MatchString(id) {
		http.Error(w, "bad id", 400)
		return
	}
	pref := r.URL.Query().Get("fmt")
	if pref != "aac" {
		pref = "opus"
	}
	t, a := r.URL.Query().Get("t"), r.URL.Query().Get("a")
	if len(t) > 200 || len(a) > 200 {
		http.Error(w, "bad request", 400)
		return
	}
	go prefetch(id, pref, t, a)
	w.WriteHeader(http.StatusNoContent)
}
