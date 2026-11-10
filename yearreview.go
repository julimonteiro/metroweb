package main

import (
	"math"
	"sort"
	"time"
)

// The year in review: everything the story tells you about a calendar year of
// listening, computed from your own play history. Nothing is invented: a
// section is left out (nil) when there is not enough data to say it honestly.

type DatedTrack struct {
	Track
	T    int64  `json:"t"`
	Date string `json:"date"` // 2006-01-02 in the viewer's timezone
}

type CountedTrack struct {
	Track
	Count int    `json:"count"`
	Date  string `json:"date,omitempty"`
}

type BusiestDay struct {
	Date      string `json:"date"`
	Plays     int    `json:"plays"`
	Minutes   int    `json:"minutes"`
	TopArtist string `json:"topArtist,omitempty"`
}

type StreakInfo struct {
	Longest int    `json:"longest"`
	Start   string `json:"start,omitempty"`
	End     string `json:"end,omitempty"`
}

type Discovery struct {
	NewArtists int          `json:"newArtists"`
	Top        []ArtistStat `json:"top"`               // new artists you played most
	TopDate    string       `json:"topDate,omitempty"` // when you first played the top one
}

type MonthTop struct {
	Month int    `json:"month"` // 1-12
	Label string `json:"label"`
	Plays int    `json:"plays"`
	Song  *Track `json:"song,omitempty"`
	Count int    `json:"count"`
}

type Metrics struct {
	Plays          int     `json:"plays"`
	NightShare     float64 `json:"nightShare"`   // plays between 22:00 and 04:59
	MorningShare   float64 `json:"morningShare"` // 05:00 to 09:59
	WeekendShare   float64 `json:"weekendShare"`
	TopArtistShare float64 `json:"topArtistShare"`
	Top3Share      float64 `json:"top3Share"`
	ArtistsPerPlay float64 `json:"artistsPerPlay"`
	Variety        float64 `json:"variety"` // 0..1: how evenly your plays spread over artists (normalised entropy)
	PlaysPerSong   float64 `json:"playsPerSong"`
	ActiveDayShare float64 `json:"activeDayShare"` // active days / days elapsed
}

type Persona struct {
	Key    string   `json:"key"`
	Name   string   `json:"name"`
	Line   string   `json:"line"`
	Second string   `json:"second,omitempty"` // runner-up name
	Traits []string `json:"traits"`
}

type YearReview struct {
	Year          int            `json:"year"`
	Stats         StatsResp      `json:"stats"`
	Days          []int          `json:"days"` // plays per day, index 0 = 1 January
	First         *DatedTrack    `json:"first,omitempty"`
	Last          *DatedTrack    `json:"last,omitempty"`
	Busiest       *BusiestDay    `json:"busiest,omitempty"`
	Streak        StreakInfo     `json:"streak"`
	Obsession     *CountedTrack  `json:"obsession,omitempty"` // most plays of one song in one day
	NightSong     *CountedTrack  `json:"nightSong,omitempty"` // most played after midnight
	TopArtistSong *CountedTrack  `json:"topArtistSong,omitempty"`
	Discovery     *Discovery     `json:"discovery,omitempty"`
	Months        []MonthTop     `json:"months"`
	TopSongs      []CountedTrack `json:"topSongs"` // the year's most played songs, best first (up to 100)
	Metrics       Metrics        `json:"metrics"`
	Persona       Persona        `json:"persona"`
}

func daysIn(year int) int {
	if time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC).YearDay() == 366 {
		return 366
	}
	return 365
}

// Review builds the year in review for a calendar year in the viewer's timezone.
func (s *StatsStore) Review(year, tzMin int, now time.Time) YearReview {
	loc := time.FixedZone("viewer", tzMin*60)
	st := s.Stats("year:"+itoa(year), tzMin, now)
	s.mu.Lock()
	all := append([]Play(nil), s.plays...)
	s.mu.Unlock()
	return buildReview(year, loc, now, st, all)
}

func itoa(n int) string { return time.Date(n, 1, 1, 0, 0, 0, 0, time.UTC).Format("2006") }

func buildReview(year int, loc *time.Location, now time.Time, st StatsResp, all []Play) YearReview {
	from := time.Date(year, time.January, 1, 0, 0, 0, 0, loc)
	to := time.Date(year+1, time.January, 1, 0, 0, 0, 0, loc)
	rv := YearReview{Year: year, Stats: st, Days: make([]int, daysIn(year)), Months: make([]MonthTop, 12), TopSongs: []CountedTrack{}}
	for m := 0; m < 12; m++ {
		rv.Months[m] = MonthTop{Month: m + 1, Label: time.Month(m + 1).String()[:3]}
	}

	var in []Play
	firstEver := int64(1 << 62)
	firstSeen := map[string]int64{} // artist -> first play ever (any year)
	for _, p := range all {
		if p.T < firstEver {
			firstEver = p.T
		}
		k := artistKey(p.Track)
		if t, ok := firstSeen[k]; !ok || p.T < t {
			firstSeen[k] = p.T
		}
		if t := time.Unix(p.T, 0); !t.Before(from) && t.Before(to) {
			in = append(in, p)
		}
	}
	if len(in) == 0 {
		rv.Persona = Persona{Key: "listener", Name: "The Listener", Line: "Not much to go on yet.", Traits: []string{}}
		return rv
	}
	sort.Slice(in, func(i, j int) bool { return in[i].T < in[j].T })

	type dayKey struct {
		day int
		id  string
	}
	dayPlays := map[int]int{}
	dayMin := map[int]int{}
	dayArtists := map[int]map[string]int{}
	sameDay := map[dayKey]int{}
	meta := map[string]Track{}
	night := map[string]int{}
	monthSongs := make([]map[string]int, 12)
	artistSongs := map[string]map[string]int{}
	artistPlays := map[string]int{}
	artistName := map[string]string{}
	songs := map[string]bool{}
	var nightN, morningN, weekendN int

	for _, p := range in {
		tm := time.Unix(p.T, 0).In(loc)
		di := tm.YearDay() - 1
		if di >= len(rv.Days) {
			di = len(rv.Days) - 1
		}
		rv.Days[di]++
		dayPlays[di]++
		dayMin[di] += durationSeconds(p.Duration) / 60
		k := artistKey(p.Track)
		if dayArtists[di] == nil {
			dayArtists[di] = map[string]int{}
		}
		dayArtists[di][k]++
		artistName[k] = primaryArtist(p.Track)
		sameDay[dayKey{di, p.ID}]++
		meta[p.ID] = p.Track
		songs[p.ID] = true
		artistPlays[k]++
		if artistSongs[k] == nil {
			artistSongs[k] = map[string]int{}
		}
		artistSongs[k][p.ID]++
		mi := int(tm.Month()) - 1
		if monthSongs[mi] == nil {
			monthSongs[mi] = map[string]int{}
		}
		monthSongs[mi][p.ID]++
		rv.Months[mi].Plays++
		h := tm.Hour()
		if h >= 22 || h < 5 {
			nightN++
		}
		if h >= 5 && h < 10 {
			morningN++
		}
		if h < 5 {
			night[p.ID]++
		}
		if wd := tm.Weekday(); wd == time.Saturday || wd == time.Sunday {
			weekendN++
		}
	}
	dated := func(p Play) *DatedTrack {
		return &DatedTrack{Track: p.Track, T: p.T, Date: time.Unix(p.T, 0).In(loc).Format("2006-01-02")}
	}
	rv.First, rv.Last = dated(in[0]), dated(in[len(in)-1])
	dateOf := func(di int) string { return from.AddDate(0, 0, di).Format("2006-01-02") }

	// busiest day and the longest run of days in a row
	bestDay := -1
	for di, n := range dayPlays {
		if bestDay < 0 || n > dayPlays[bestDay] || (n == dayPlays[bestDay] && di < bestDay) {
			bestDay = di
		}
	}
	b := &BusiestDay{Date: dateOf(bestDay), Plays: dayPlays[bestDay], Minutes: dayMin[bestDay]}
	topK, topN := "", 0
	for k, n := range dayArtists[bestDay] {
		if n > topN || (n == topN && k < topK) {
			topK, topN = k, n
		}
	}
	b.TopArtist = artistName[topK]
	rv.Busiest = b

	run, runStart := 0, 0
	for di := 0; di < len(rv.Days); di++ {
		if rv.Days[di] > 0 {
			if run == 0 {
				runStart = di
			}
			run++
			if run > rv.Streak.Longest {
				rv.Streak = StreakInfo{Longest: run, Start: dateOf(runStart), End: dateOf(di)}
			}
		} else {
			run = 0
		}
	}

	// the song you could not stop playing in a single day
	var ob *CountedTrack
	for dk, n := range sameDay {
		if n >= 3 && (ob == nil || n > ob.Count || (n == ob.Count && dateOf(dk.day) < ob.Date)) {
			ob = &CountedTrack{Track: meta[dk.id], Count: n, Date: dateOf(dk.day)}
		}
	}
	rv.Obsession = ob

	// your after-midnight song
	var ns *CountedTrack
	for id, n := range night {
		if n >= 2 && (ns == nil || n > ns.Count || (n == ns.Count && id < ns.Track.ID)) {
			ns = &CountedTrack{Track: meta[id], Count: n}
		}
	}
	rv.NightSong = ns

	// most played song of the top artist
	var topArtist string
	for k, n := range artistPlays {
		if topArtist == "" || n > artistPlays[topArtist] || (n == artistPlays[topArtist] && k < topArtist) {
			topArtist = k
		}
	}
	for id, n := range artistSongs[topArtist] {
		if rv.TopArtistSong == nil || n > rv.TopArtistSong.Count || (n == rv.TopArtistSong.Count && id < rv.TopArtistSong.Track.ID) {
			rv.TopArtistSong = &CountedTrack{Track: meta[id], Count: n}
		}
	}

	// the year's most played songs, best first (this is what the "Top Songs" playlist holds)
	songCount := map[string]int{}
	lastPlayed := map[string]int64{}
	for _, p := range in {
		songCount[p.ID]++
		lastPlayed[p.ID] = p.T
	}
	for id, n := range songCount {
		rv.TopSongs = append(rv.TopSongs, CountedTrack{Track: meta[id], Count: n})
	}
	sort.Slice(rv.TopSongs, func(i, j int) bool {
		a, b := rv.TopSongs[i], rv.TopSongs[j]
		if a.Count != b.Count {
			return a.Count > b.Count
		}
		if lastPlayed[a.Track.ID] != lastPlayed[b.Track.ID] {
			return lastPlayed[a.Track.ID] > lastPlayed[b.Track.ID]
		}
		return a.Track.ID < b.Track.ID
	})
	if len(rv.TopSongs) > 100 {
		rv.TopSongs = rv.TopSongs[:100]
	}

	// song of each month
	for mi, ms := range monthSongs {
		for id, n := range ms {
			if rv.Months[mi].Song == nil || n > rv.Months[mi].Count || (n == rv.Months[mi].Count && id < rv.Months[mi].Song.ID) {
				t := meta[id]
				rv.Months[mi].Song, rv.Months[mi].Count = &t, n
			}
		}
	}

	// discovery only means something when there is history from before the year
	if firstEver < from.Unix() {
		d := &Discovery{Top: []ArtistStat{}}
		var fresh []ArtistStat
		for k, n := range artistPlays {
			if firstSeen[k] >= from.Unix() {
				d.NewArtists++
				a := ArtistStat{Name: artistName[k], Plays: n}
				for _, p := range in {
					if artistKey(p.Track) == k {
						a.ID, a.Thumb = p.ArtistID, p.Thumb
						break
					}
				}
				fresh = append(fresh, a)
			}
		}
		sort.Slice(fresh, func(i, j int) bool {
			if fresh[i].Plays != fresh[j].Plays {
				return fresh[i].Plays > fresh[j].Plays
			}
			return fresh[i].Name < fresh[j].Name
		})
		if len(fresh) > 5 {
			fresh = fresh[:5]
		}
		d.Top = fresh
		if len(fresh) > 0 {
			for k, name := range artistName {
				if name == fresh[0].Name {
					d.TopDate = time.Unix(firstSeen[k], 0).In(loc).Format("2006-01-02")
				}
			}
		}
		if d.NewArtists > 0 {
			rv.Discovery = d
		}
	}

	// who you are as a listener
	n := float64(len(in))
	var ranked []int
	for _, c := range artistPlays {
		ranked = append(ranked, c)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(ranked)))
	top3 := 0
	for i := 0; i < len(ranked) && i < 3; i++ {
		top3 += ranked[i]
	}
	elapsed := float64(len(rv.Days))
	if now.Before(to) {
		elapsed = float64(now.In(loc).YearDay())
	}
	active := 0
	for _, c := range rv.Days {
		if c > 0 {
			active++
		}
	}
	variety := 0.0
	if len(ranked) > 1 {
		for _, c := range ranked {
			p := float64(c) / n
			variety -= p * math.Log(p)
		}
		variety /= math.Log(float64(len(ranked)))
	}
	rv.Metrics = Metrics{Variety: variety, Plays: len(in), NightShare: float64(nightN) / n, MorningShare: float64(morningN) / n, WeekendShare: float64(weekendN) / n,
		TopArtistShare: float64(ranked[0]) / n, Top3Share: float64(top3) / n, ArtistsPerPlay: float64(len(artistPlays)) / n,
		PlaysPerSong: n / float64(len(songs)), ActiveDayShare: float64(active) / elapsed}
	rv.Persona = PickPersona(rv.Metrics)
	return rv
}

// PickPersona names the listener you are, from how you listen. A trait only
// counts when it is clearly stronger than usual; otherwise you are "The Listener".
func PickPersona(m Metrics) Persona {
	type cand struct {
		key, name, line string
		score           float64
		trait           string
	}
	cs := []cand{
		{"night_owl", "The Night Owl", "Your best listening happens after dark.", m.NightShare / 0.30, "Plays most after 10 PM"},
		{"early_bird", "The Early Bird", "You start the day with a soundtrack.", m.MorningShare / 0.28, "Plays most before 10 AM"},
		{"loyalist", "The Loyalist", "When you find an artist you love, you stay.", maxf(m.TopArtistShare/0.25, m.Top3Share/0.55), "Sticks with favourite artists"},
		{"explorer", "The Explorer", "Always hunting for the next song.", m.ArtistsPerPlay / 0.45, "Plays a wide range of artists"},
		{"repeater", "The Repeater", "A great song deserves another play. And another.", m.PlaysPerSong / 3.0, "Replays favourites a lot"},
		{"marathoner", "The Marathoner", "Music is part of nearly every day.", m.ActiveDayShare / 0.8, "Listens almost every day"},
		{"weekender", "The Weekend Warrior", "The weekend is when your music comes alive.", m.WeekendShare / 0.45, "Plays most on weekends"},
	}
	if m.Plays < 20 {
		return Persona{Key: "listener", Name: "The Listener", Line: "Keep listening and your style will show.", Traits: []string{}}
	}
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].score > cs[j].score })
	p := Persona{Key: "listener", Name: "The Listener", Line: "A little of everything.", Traits: []string{}}
	if cs[0].score >= 1 {
		p = Persona{Key: cs[0].key, Name: cs[0].name, Line: cs[0].line, Traits: []string{cs[0].trait}}
		for _, c := range cs[1:] {
			if c.score >= 1 {
				p.Traits = append(p.Traits, c.trait)
				if p.Second == "" {
					p.Second = c.name
				}
			}
		}
	}
	return p
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
