package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Listening history
// ---------------------------------------------------------------------------

// Play is one counted listen. A play counts once a song has been heard for 30s
// (or half of a short one); the browser reports it.
type Play struct {
	T int64 `json:"t"` // unix seconds
	Track
}

// StatsStore keeps the history in an append-only JSON-lines file and resolves
// artist genres in the background (MusicBrainz, cached on disk).
type StatsStore struct {
	mu      sync.Mutex
	dir     string
	plays   []Play
	genres  map[string]genreEntry // lowercased artist name -> genres
	pending map[string]bool
	queue   chan string
	enabled bool // genre lookups
	client  *http.Client
}

type genreEntry struct {
	Genres []string `json:"genres"`
	At     int64    `json:"at"`
}

const genreRetry = 30 * 24 * time.Hour

func NewStatsStore(dir string, genreLookups bool) (*StatsStore, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	s := &StatsStore{dir: dir, genres: map[string]genreEntry{}, pending: map[string]bool{},
		queue: make(chan string, 500), enabled: genreLookups, client: &http.Client{Timeout: 15 * time.Second}}
	if f, err := os.Open(filepath.Join(dir, "plays.jsonl")); err == nil {
		defer f.Close()
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 64*1024), 1<<20)
		for sc.Scan() {
			var p Play
			if json.Unmarshal(sc.Bytes(), &p) == nil && p.ID != "" && p.T > 0 {
				s.plays = append(s.plays, p)
			}
		}
	}
	if b, err := os.ReadFile(filepath.Join(dir, "genres.json")); err == nil {
		json.Unmarshal(b, &s.genres)
	}
	if s.enabled {
		go s.genreWorker()
		seen := 0
		for _, p := range s.plays { // resolve artists from earlier plays too
			if seen < 200 && s.wantGenre(primaryArtist(p.Track)) {
				s.enqueue(primaryArtist(p.Track))
				seen++
			}
		}
	}
	return s, nil
}

func primaryArtist(t Track) string {
	a := strings.TrimSpace(strings.Split(t.Artist, ",")[0])
	if a == "" {
		return "Unknown artist"
	}
	return a
}

func artistKey(t Track) string {
	if t.ArtistID != "" {
		return t.ArtistID
	}
	return "n:" + strings.ToLower(primaryArtist(t))
}

// Record stores a play. Repeats of the same song within 20 seconds are ignored.
func (s *StatsStore) Record(raw Track, now time.Time) (bool, error) {
	t, ok := sanitizeTrack(raw)
	if !ok {
		return false, fmt.Errorf("invalid track")
	}
	p := Play{T: now.Unix(), Track: t}
	s.mu.Lock()
	if n := len(s.plays); n > 0 && s.plays[n-1].ID == t.ID && p.T-s.plays[n-1].T < 20 {
		s.mu.Unlock()
		return false, nil
	}
	s.plays = append(s.plays, p)
	s.mu.Unlock()

	line, _ := json.Marshal(p)
	f, err := os.OpenFile(filepath.Join(s.dir, "plays.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return true, err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	if s.enabled && s.wantGenre(primaryArtist(t)) {
		s.enqueue(primaryArtist(t))
	}
	return true, err
}

// ---------------------------------------------------------------------------
// Genres (MusicBrainz)
// ---------------------------------------------------------------------------

func (s *StatsStore) wantGenre(name string) bool {
	key := strings.ToLower(name)
	if key == "" || key == "unknown artist" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.genres[key]; ok && time.Since(time.Unix(e.At, 0)) < genreRetry {
		return false
	}
	return !s.pending[key]
}

func (s *StatsStore) enqueue(name string) {
	key := strings.ToLower(name)
	s.mu.Lock()
	s.pending[key] = true
	s.mu.Unlock()
	select {
	case s.queue <- name:
	default:
		s.mu.Lock()
		delete(s.pending, key)
		s.mu.Unlock()
	}
}

func (s *StatsStore) genreWorker() {
	for name := range s.queue {
		key := strings.ToLower(name)
		genres, err := s.lookupGenres(context.Background(), name)
		s.mu.Lock()
		delete(s.pending, key)
		if err == nil {
			s.genres[key] = genreEntry{Genres: genres, At: time.Now().Unix()}
		}
		snapshot, _ := json.Marshal(s.genres)
		s.mu.Unlock()
		if err != nil {
			log.Printf("genres %q: %v", name, err)
		} else {
			os.WriteFile(filepath.Join(s.dir, "genres.json"), snapshot, 0o600)
		}
		time.Sleep(1200 * time.Millisecond) // MusicBrainz asks for at most 1 request/second
	}
}

func (s *StatsStore) mbGet(ctx context.Context, u string, out any) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	req.Header.Set("User-Agent", "metroweb/1.0 (https://github.com/julimonteiro/metroweb)")
	req.Header.Set("Accept", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("musicbrainz: HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(out)
}

// lookupGenres returns up to 3 genres for an artist name, or none when the
// artist can't be matched confidently. Only the artist name leaves the server.
func (s *StatsStore) lookupGenres(ctx context.Context, name string) ([]string, error) {
	q := strings.NewReplacer(`"`, "", `\`, "").Replace(name)
	var search struct {
		Artists []struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Score int    `json:"score"`
		} `json:"artists"`
	}
	if err := s.mbGet(ctx, "https://musicbrainz.org/ws/2/artist/?fmt=json&limit=5&query="+url.QueryEscape(`artist:"`+q+`"`), &search); err != nil {
		return nil, err
	}
	id := ""
	for _, a := range search.Artists {
		if a.Score >= 90 && strings.EqualFold(a.Name, name) {
			id = a.ID
			break
		}
	}
	if id == "" {
		return nil, nil
	}
	time.Sleep(1200 * time.Millisecond)
	var art struct {
		Genres []struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		} `json:"genres"`
	}
	if err := s.mbGet(ctx, "https://musicbrainz.org/ws/2/artist/"+id+"?fmt=json&inc=genres", &art); err != nil {
		return nil, err
	}
	sort.SliceStable(art.Genres, func(i, j int) bool { return art.Genres[i].Count > art.Genres[j].Count })
	var out []string
	for _, g := range art.Genres {
		if len(out) == 3 {
			break
		}
		out = append(out, titleCase(g.Name))
	}
	return out, nil
}

func titleCase(s string) string {
	parts := strings.Fields(strings.ToLower(s))
	for i, p := range parts {
		if p == "r&b" || p == "edm" || p == "idm" {
			parts[i] = strings.ToUpper(p)
		} else if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

// ---------------------------------------------------------------------------
// Aggregation
// ---------------------------------------------------------------------------

type ArtistStat struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name"`
	Thumb   string `json:"thumb,omitempty"`
	Plays   int    `json:"plays"`
	Minutes int    `json:"minutes"`
}
type SongStat struct {
	Track
	Plays int `json:"plays"`
}
type AlbumStat struct {
	ID     string `json:"id,omitempty"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Thumb  string `json:"thumb,omitempty"`
	Plays  int    `json:"plays"`
}
type GenreStat struct {
	Name  string  `json:"name"`
	Plays float64 `json:"plays"`
	Pct   float64 `json:"pct"`
}
type Bucket struct {
	Key       string      `json:"key"`
	Label     string      `json:"label"`
	Plays     int         `json:"plays"`
	Minutes   int         `json:"minutes"`
	TopArtist *ArtistStat `json:"topArtist,omitempty"`
}
type Series struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
	Data []int  `json:"data"`
}
type Totals struct {
	Plays      int `json:"plays"`
	Minutes    int `json:"minutes"`
	Artists    int `json:"artists"`
	Songs      int `json:"songs"`
	ActiveDays int `json:"activeDays"`
	Streak     int `json:"streak"`
}
type StatsResp struct {
	Range         string       `json:"range"`
	Totals        Totals       `json:"totals"`
	TopArtists    []ArtistStat `json:"topArtists"`
	TopSongs      []SongStat   `json:"topSongs"`
	TopAlbums     []AlbumStat  `json:"topAlbums"`
	Genres        []GenreStat  `json:"genres"`
	GenresPending int          `json:"genresPending"`
	GenresEnabled bool         `json:"genresEnabled"`
	Timeline      []Bucket     `json:"timeline"`
	Granularity   string       `json:"granularity"`
	Series        []Series     `json:"series"`
	Months        []Bucket     `json:"months"`
	ByHour        [24]int      `json:"byHour"`
	ByWeekday     [7]int       `json:"byWeekday"`
}

func durationSeconds(d string) int {
	parts := strings.Split(d, ":")
	n := 0
	for _, p := range parts {
		v, err := strconv.Atoi(p)
		if err != nil {
			return 210
		}
		n = n*60 + v
	}
	if n <= 0 || n > 4*3600 {
		return 210
	}
	return n
}

// Stats aggregates plays within a range, in the viewer's timezone
// (tzOffsetMin = minutes east of UTC).
func (s *StatsStore) Stats(rng string, tzOffsetMin int, now time.Time) StatsResp {
	loc := time.FixedZone("viewer", tzOffsetMin*60)
	now = now.In(loc)
	var from time.Time
	switch rng {
	case "30d":
		from = now.AddDate(0, 0, -30)
	case "90d":
		from = now.AddDate(0, 0, -90)
	case "365d":
		from = now.AddDate(0, 0, -365)
	default:
		rng, from = "all", time.Unix(0, 0)
	}

	s.mu.Lock()
	all := append([]Play(nil), s.plays...)
	genres := make(map[string]genreEntry, len(s.genres))
	for k, v := range s.genres {
		genres[k] = v
	}
	pendingCount := len(s.pending)
	s.mu.Unlock()

	res := StatsResp{Range: rng, GenresEnabled: s.enabled, TopArtists: []ArtistStat{}, TopSongs: []SongStat{},
		TopAlbums: []AlbumStat{}, Genres: []GenreStat{}, Timeline: []Bucket{}, Series: []Series{}, Months: []Bucket{}}

	var in []Play
	for _, p := range all {
		if !time.Unix(p.T, 0).Before(from) {
			in = append(in, p)
		}
	}
	artists := map[string]*ArtistStat{}
	songs := map[string]*SongStat{}
	albums := map[string]*AlbumStat{}
	genreW := map[string]float64{}
	days := map[string]bool{}
	for _, p := range in {
		tm := time.Unix(p.T, 0).In(loc)
		sec := durationSeconds(p.Duration)
		name := primaryArtist(p.Track)
		k := artistKey(p.Track)
		a := artists[k]
		if a == nil {
			a = &ArtistStat{ID: p.ArtistID, Name: name}
			artists[k] = a
		}
		a.Plays++
		a.Minutes += sec
		if a.Thumb == "" {
			a.Thumb = p.Thumb
		}
		sg := songs[p.ID]
		if sg == nil {
			sg = &SongStat{Track: p.Track}
			songs[p.ID] = sg
		}
		sg.Plays++
		if p.Album != "" {
			ak := p.AlbumID
			if ak == "" {
				ak = "n:" + strings.ToLower(p.Album+"|"+name)
			}
			al := albums[ak]
			if al == nil {
				al = &AlbumStat{ID: p.AlbumID, Title: p.Album, Artist: name, Thumb: p.Thumb}
				albums[ak] = al
			}
			al.Plays++
		}
		if e, ok := genres[strings.ToLower(name)]; ok && len(e.Genres) > 0 {
			weights := []float64{0.5, 0.3, 0.2}[:len(e.Genres)]
			var sum float64
			for _, w := range weights {
				sum += w
			}
			for i, g := range e.Genres {
				genreW[g] += weights[i] / sum
			}
		}
		res.Totals.Plays++
		res.Totals.Minutes += sec
		res.ByHour[tm.Hour()]++
		res.ByWeekday[int(tm.Weekday())]++
		days[tm.Format("2006-01-02")] = true
	}
	res.Totals.Minutes /= 60
	res.Totals.Artists, res.Totals.Songs, res.Totals.ActiveDays = len(artists), len(songs), len(days)
	res.Totals.Streak = streak(days, now)

	topA := make([]*ArtistStat, 0, len(artists))
	for _, a := range artists {
		a.Minutes /= 60
		topA = append(topA, a)
	}
	sort.Slice(topA, func(i, j int) bool {
		if topA[i].Plays != topA[j].Plays {
			return topA[i].Plays > topA[j].Plays
		}
		return topA[i].Name < topA[j].Name
	})
	for i, a := range topA {
		if i == 10 {
			break
		}
		res.TopArtists = append(res.TopArtists, *a)
	}
	topS := make([]*SongStat, 0, len(songs))
	for _, sg := range songs {
		topS = append(topS, sg)
	}
	sort.Slice(topS, func(i, j int) bool {
		if topS[i].Plays != topS[j].Plays {
			return topS[i].Plays > topS[j].Plays
		}
		return topS[i].Title < topS[j].Title
	})
	for i, sg := range topS {
		if i == 10 {
			break
		}
		res.TopSongs = append(res.TopSongs, *sg)
	}
	topAl := make([]*AlbumStat, 0, len(albums))
	for _, al := range albums {
		topAl = append(topAl, al)
	}
	sort.Slice(topAl, func(i, j int) bool {
		if topAl[i].Plays != topAl[j].Plays {
			return topAl[i].Plays > topAl[j].Plays
		}
		return topAl[i].Title < topAl[j].Title
	})
	for i, al := range topAl {
		if i == 6 {
			break
		}
		res.TopAlbums = append(res.TopAlbums, *al)
	}

	var gsum float64
	for _, w := range genreW {
		gsum += w
	}
	for g, w := range genreW {
		res.Genres = append(res.Genres, GenreStat{Name: g, Plays: round1(w), Pct: round1(w / gsum * 100)})
	}
	sort.Slice(res.Genres, func(i, j int) bool {
		if res.Genres[i].Plays != res.Genres[j].Plays {
			return res.Genres[i].Plays > res.Genres[j].Plays
		}
		return res.Genres[i].Name < res.Genres[j].Name
	})
	if len(res.Genres) > 8 {
		var rest GenreStat
		rest.Name = "Other"
		for _, g := range res.Genres[7:] {
			rest.Plays += g.Plays
			rest.Pct += g.Pct
		}
		rest.Plays, rest.Pct = round1(rest.Plays), round1(rest.Pct)
		res.Genres = append(res.Genres[:7], rest)
	}
	res.GenresPending = pendingCount

	// Timeline buckets (days / weeks / months depending on the range) and the
	// per-artist series for the top five, plus the fixed last-12-months view.
	res.Granularity, res.Timeline = buckets(rng, in, loc, now)
	topKeys := []string{}
	for i, a := range topA {
		if i == 5 {
			break
		}
		topKeys = append(topKeys, artistKeyOf(a))
		res.Series = append(res.Series, Series{ID: a.ID, Name: a.Name, Data: make([]int, len(res.Timeline))})
	}
	idxOf := map[string]int{}
	for i, b := range res.Timeline {
		idxOf[b.Key] = i
	}
	gran := res.Granularity
	for _, p := range in {
		i, ok := idxOf[bucketKey(gran, time.Unix(p.T, 0).In(loc), now)]
		if !ok {
			continue
		}
		for si, k := range topKeys {
			if artistKey(p.Track) == k {
				res.Series[si].Data[i]++
			}
		}
	}
	_, res.Months = buckets("365d", all, loc, now)
	fillTopArtists(res.Timeline, in, gran, loc, now)
	fillTopArtists(res.Months, all, "month", loc, now)
	return res
}

func artistKeyOf(a *ArtistStat) string {
	if a.ID != "" {
		return a.ID
	}
	return "n:" + strings.ToLower(a.Name)
}

func round1(v float64) float64 { return float64(int(v*10+0.5)) / 10 }

// streak counts consecutive days with at least one play, ending today (or
// yesterday, so an evening streak isn't lost before you press play).
func streak(days map[string]bool, now time.Time) int {
	d := now
	if !days[d.Format("2006-01-02")] {
		d = d.AddDate(0, 0, -1)
	}
	n := 0
	for days[d.Format("2006-01-02")] {
		n++
		d = d.AddDate(0, 0, -1)
	}
	return n
}

func bucketKey(gran string, t, now time.Time) string {
	switch gran {
	case "day":
		return t.Format("2006-01-02")
	case "week":
		y, w := t.ISOWeek()
		return fmt.Sprintf("%d-W%02d", y, w)
	default:
		return t.Format("2006-01")
	}
}

// buckets builds the empty (zero-filled) buckets for the range and fills plays/minutes.
func buckets(rng string, plays []Play, loc *time.Location, now time.Time) (string, []Bucket) {
	var gran string
	var out []Bucket
	switch rng {
	case "30d":
		gran = "day"
		for i := 29; i >= 0; i-- {
			d := now.AddDate(0, 0, -i)
			out = append(out, Bucket{Key: d.Format("2006-01-02"), Label: d.Format("Jan 2")})
		}
	case "90d":
		gran = "week"
		seen := map[string]bool{}
		for i := 90; i >= 0; i-- {
			d := now.AddDate(0, 0, -i)
			k := bucketKey("week", d, now)
			if !seen[k] {
				seen[k] = true
				out = append(out, Bucket{Key: k, Label: d.Format("Jan 2")})
			}
		}
	default:
		gran = "month"
		months := 12
		if rng == "all" && len(plays) > 0 {
			first := time.Unix(plays[0].T, 0).In(loc)
			if m := (now.Year()-first.Year())*12 + int(now.Month()-first.Month()) + 1; m > 12 {
				months = m
				if months > 36 {
					months = 36
				}
			}
		}
		first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		for i := months - 1; i >= 0; i-- {
			d := first.AddDate(0, -i, 0)
			label := d.Format("Jan")
			if d.Month() == time.January || i == months-1 {
				label = d.Format("Jan 06")
			}
			out = append(out, Bucket{Key: d.Format("2006-01"), Label: label})
		}
	}
	idx := map[string]int{}
	for i, b := range out {
		idx[b.Key] = i
	}
	for _, p := range plays {
		if i, ok := idx[bucketKey(gran, time.Unix(p.T, 0).In(loc), now)]; ok {
			out[i].Plays++
			out[i].Minutes += durationSeconds(p.Duration)
		}
	}
	for i := range out {
		out[i].Minutes /= 60
	}
	return gran, out
}

func fillTopArtists(bs []Bucket, plays []Play, gran string, loc *time.Location, now time.Time) {
	per := map[string]map[string]*ArtistStat{}
	for _, p := range plays {
		k := bucketKey(gran, time.Unix(p.T, 0).In(loc), now)
		m := per[k]
		if m == nil {
			m = map[string]*ArtistStat{}
			per[k] = m
		}
		ak := artistKey(p.Track)
		a := m[ak]
		if a == nil {
			a = &ArtistStat{ID: p.ArtistID, Name: primaryArtist(p.Track), Thumb: p.Thumb}
			m[ak] = a
		}
		a.Plays++
	}
	for i := range bs {
		var best *ArtistStat
		for _, a := range per[bs[i].Key] {
			if best == nil || a.Plays > best.Plays || (a.Plays == best.Plays && a.Name < best.Name) {
				best = a
			}
		}
		if best != nil {
			cp := *best
			bs[i].TopArtist = &cp
		}
	}
}

// PlaySummary is what the shuffle needs to know about a song: how often it was
// heard and when it was heard last.
type PlaySummary struct {
	N    int   `json:"n"`
	Last int64 `json:"last"`
}

// Summary returns the play count and last play time of every song heard so far.
func (s *StatsStore) Summary() map[string]PlaySummary {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]PlaySummary, len(s.plays)/2)
	for _, p := range s.plays {
		e := out[p.ID]
		e.N++
		if p.T > e.Last {
			e.Last = p.T
		}
		out[p.ID] = e
	}
	return out
}
