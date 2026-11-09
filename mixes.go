package main

import (
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Made-for-you mixes, computed from your own listening history and your Liked
// songs. Nothing is fetched from YouTube: they are instant, they only contain
// songs you already know, and they stay the same for the whole day.

type Mix struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Subtitle string  `json:"subtitle"`
	Thumb    string  `json:"thumb,omitempty"`
	Tracks   []Track `json:"tracks"`
}

const (
	mixMin = 6  // a mix needs at least this many songs
	mixMax = 50 // and never has more than this
)

type songStat struct {
	n, n30 int
	last   int64
}

func (s *StatsStore) Mixes(liked []Track, tzMin int, now time.Time) []Mix {
	s.mu.Lock()
	plays := append([]Play(nil), s.plays...)
	genres := make(map[string]genreEntry, len(s.genres))
	for k, v := range s.genres {
		genres[k] = v
	}
	s.mu.Unlock()
	return buildMixes(plays, liked, genres, tzMin, now)
}

func buildMixes(plays []Play, liked []Track, genres map[string]genreEntry, tzMin int, now time.Time) []Mix {
	loc := now.UTC().Add(time.Duration(tzMin) * time.Minute)
	rng := rand.New(rand.NewSource(int64(loc.YearDay())*1000 + int64(loc.Year()))) // stable through the day

	meta := map[string]Track{}
	st := map[string]*songStat{}
	likedSet := map[string]bool{}
	for _, t := range liked {
		meta[t.ID] = t
		likedSet[t.ID] = true
	}
	cut30 := now.Add(-30 * 24 * time.Hour).Unix()
	for _, p := range plays {
		meta[p.ID] = p.Track
		e := st[p.ID]
		if e == nil {
			e = &songStat{}
			st[p.ID] = e
		}
		e.n++
		if p.T >= cut30 {
			e.n30++
		}
		if p.T > e.last {
			e.last = p.T
		}
	}
	stat := func(id string) songStat {
		if e := st[id]; e != nil {
			return *e
		}
		return songStat{}
	}

	var out []Mix
	add := func(id, title, sub string, ids []string) {
		if len(ids) < mixMin {
			return
		}
		if len(ids) > mixMax {
			ids = ids[:mixMax]
		}
		m := Mix{ID: id, Title: title, Subtitle: sub}
		for _, i := range ids {
			t := meta[i]
			if m.Thumb == "" {
				m.Thumb = t.Thumb
			}
			m.Tracks = append(m.Tracks, t)
		}
		out = append(out, m)
	}
	// ids sorted by key (desc), ties broken by the day's luck
	ranked := func(ids []string, key func(string) float64) []string {
		rng.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
		sort.SliceStable(ids, func(i, j int) bool { return key(ids[i]) > key(ids[j]) })
		return ids
	}
	shuffled := func(ids []string) []string {
		rng.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
		return ids
	}
	var all []string
	for id := range meta {
		all = append(all, id)
	}
	sort.Strings(all) // deterministic base order before any shuffling

	// 1. On repeat: what you played most this month
	var rep []string
	for _, id := range all {
		if stat(id).n30 >= 2 {
			rep = append(rep, id)
		}
	}
	add("repeat", "On repeat", "What you can't stop playing this month", ranked(rep, func(id string) float64 { return float64(stat(id).n30) }))

	// 2. Rewind: songs you played a lot, then forgot
	var rew []string
	old := now.Add(-45 * 24 * time.Hour).Unix()
	for _, id := range all {
		if e := stat(id); e.n >= 2 && e.last < old {
			rew = append(rew, id)
		}
	}
	add("rewind", "Rewind", "Songs you used to play, and haven't in a while", ranked(rew, func(id string) float64 { return float64(stat(id).n) }))

	// 3. Hidden gems: liked, never played here
	var gems []string
	for _, id := range all {
		if likedSet[id] && stat(id).n == 0 {
			gems = append(gems, id)
		}
	}
	add("gems", "Hidden gems", "Liked songs you haven't played here yet", shuffled(gems))

	// 4. Around now: what you usually play at this time of day
	bucket := func(h int) int { // 0 night, 1 morning, 2 afternoon, 3 evening
		switch {
		case h >= 5 && h < 12:
			return 1
		case h >= 12 && h < 18:
			return 2
		case h >= 18 && h < 23:
			return 3
		}
		return 0
	}
	names := [4][2]string{{"Late night mix", "What you play after dark"}, {"Morning mix", "What you play to start the day"},
		{"Afternoon mix", "What you play in the afternoon"}, {"Evening mix", "What you play in the evening"}}
	nowB := bucket(loc.Hour())
	cnt := map[string]int{}
	for _, p := range plays {
		if bucket(time.Unix(p.T, 0).UTC().Add(time.Duration(tzMin)*time.Minute).Hour()) == nowB {
			cnt[p.ID]++
		}
	}
	var around []string
	for _, id := range all {
		if cnt[id] > 0 {
			around = append(around, id)
		}
	}
	add("now", names[nowB][0], names[nowB][1], ranked(around, func(id string) float64 { return float64(cnt[id]) }))

	// 5. Top artists: everything of theirs you know
	type art struct {
		name  string
		plays int
		ids   []string
	}
	byArtist := map[string]*art{}
	for _, id := range all {
		t := meta[id]
		k := artistKey(t)
		a := byArtist[k]
		if a == nil {
			a = &art{name: primaryArtist(t)}
			byArtist[k] = a
		}
		a.plays += stat(id).n
		a.ids = append(a.ids, id)
	}
	var arts []*art
	for _, a := range byArtist {
		if a.plays > 0 && a.name != "Unknown artist" {
			arts = append(arts, a)
		}
	}
	sort.Slice(arts, func(i, j int) bool {
		if arts[i].plays != arts[j].plays {
			return arts[i].plays > arts[j].plays
		}
		return arts[i].name < arts[j].name
	})
	n := 0
	for _, a := range arts {
		if n == 3 {
			break
		}
		if len(a.ids) >= mixMin+2 {
			id := slug(a.name)
			if id == "" {
				id = strconv.Itoa(n)
			}
			add("artist-"+id, a.name+" mix", "Your songs by "+a.name, shuffled(append([]string(nil), a.ids...)))
			n++
		}
	}

	// 6. Genres of your favourite artists
	gIDs := map[string][]string{}
	for _, id := range all {
		if e, ok := genres[strings.ToLower(primaryArtist(meta[id]))]; ok {
			for i, g := range e.Genres {
				if i < 2 {
					gIDs[g] = append(gIDs[g], id)
				}
			}
		}
	}
	var gs []string
	for g, ids := range gIDs {
		if len(ids) >= 12 {
			gs = append(gs, g)
		}
	}
	sort.Slice(gs, func(i, j int) bool {
		if len(gIDs[gs[i]]) != len(gIDs[gs[j]]) {
			return len(gIDs[gs[i]]) > len(gIDs[gs[j]])
		}
		return gs[i] < gs[j]
	})
	if len(gs) > 2 {
		gs = gs[:2]
	}
	for _, g := range gs {
		add("genre-"+slug(g), titleCase(g)+" mix", "Your "+strings.ToLower(g)+" songs", shuffled(append([]string(nil), gIDs[g]...)))
	}
	return out
}

func slug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
