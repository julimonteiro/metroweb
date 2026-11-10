package main

import (
	"fmt"
	"testing"
	"time"
)

func rt(i int, artist string) Track {
	return Track{ID: fmt.Sprintf("trk%08d", i), Title: fmt.Sprintf("Song %d", i), Artist: artist, ArtistID: "UC" + artist, Duration: "3:00", Thumb: "http://x/" + artist + ".jpg"}
}

func at(y int, m time.Month, d, h int) int64 { return time.Date(y, m, d, h, 0, 0, 0, time.UTC).Unix() }

func reviewFixture() (YearReview, []Play) {
	var plays []Play
	add := func(tr Track, t int64) { plays = append(plays, Play{T: t, Track: tr}) }
	// history from before the year: Old Band was already known
	add(rt(1, "Old Band"), at(2025, 6, 1, 12))
	// 2026: Old Band every day for a week in March (a streak), 6 plays of song 1 on Mar 3
	for d := 1; d <= 7; d++ {
		add(rt(1, "Old Band"), at(2026, 3, d, 12))
	}
	for k := 0; k < 5; k++ {
		add(rt(1, "Old Band"), at(2026, 3, 3, 13+k)) // Mar 3: 1 + 5 = 6 plays of song 1
	}
	// a new artist found in May, played at 2 AM twice
	add(rt(2, "Fresh"), at(2026, 5, 10, 2))
	add(rt(2, "Fresh"), at(2026, 5, 11, 3))
	add(rt(3, "Fresh"), at(2026, 5, 12, 20))
	// outside the year
	add(rt(9, "Future"), at(2027, 1, 5, 12))
	s := &StatsStore{plays: plays, genres: map[string]genreEntry{}}
	now := time.Date(2026, 12, 15, 12, 0, 0, 0, time.UTC)
	return s.Review(2026, 0, now), plays
}

func TestYearReviewBasics(t *testing.T) {
	rv, _ := reviewFixture()
	if rv.Year != 2026 || len(rv.Days) != 365 {
		t.Fatalf("year/days = %d/%d", rv.Year, len(rv.Days))
	}
	total := 0
	for _, n := range rv.Days {
		total += n
	}
	if total != 7+5+3 {
		t.Fatalf("plays counted in 2026 = %d, want 15 (the 2025 and 2027 plays are out)", total)
	}
	if rv.First == nil || rv.First.Date != "2026-03-01" || rv.Last == nil || rv.Last.Date != "2026-05-12" {
		t.Fatalf("first/last = %+v %+v", rv.First, rv.Last)
	}
	if rv.Busiest == nil || rv.Busiest.Date != "2026-03-03" || rv.Busiest.Plays != 6 || rv.Busiest.TopArtist != "Old Band" {
		t.Fatalf("busiest = %+v", rv.Busiest)
	}
	if rv.Streak.Longest != 7 || rv.Streak.Start != "2026-03-01" || rv.Streak.End != "2026-03-07" {
		t.Fatalf("streak = %+v", rv.Streak)
	}
}

func TestYearReviewHighlights(t *testing.T) {
	rv, _ := reviewFixture()
	if rv.Obsession == nil || rv.Obsession.Track.ID != "trk00000001" || rv.Obsession.Count != 6 || rv.Obsession.Date != "2026-03-03" {
		t.Fatalf("obsession = %+v", rv.Obsession)
	}
	if rv.NightSong == nil || rv.NightSong.Track.ID != "trk00000002" || rv.NightSong.Count != 2 {
		t.Fatalf("night song = %+v", rv.NightSong)
	}
	if rv.TopArtistSong == nil || rv.TopArtistSong.Track.ID != "trk00000001" || rv.TopArtistSong.Count != 12 {
		t.Fatalf("top artist song = %+v", rv.TopArtistSong)
	}
	if rv.Months[2].Song == nil || rv.Months[2].Song.ID != "trk00000001" || rv.Months[2].Plays != 12 || rv.Months[4].Song == nil || rv.Months[4].Song.ID != "trk00000002" || rv.Months[0].Song != nil {
		t.Fatalf("months = %+v", rv.Months)
	}
	if len(rv.TopSongs) != 3 || rv.TopSongs[0].Track.ID != "trk00000001" || rv.TopSongs[0].Count != 12 || rv.TopSongs[1].Track.ID != "trk00000002" {
		t.Fatalf("top songs = %+v", rv.TopSongs)
	}
	if rv.Discovery == nil || rv.Discovery.NewArtists != 1 || rv.Discovery.Top[0].Name != "Fresh" || rv.Discovery.TopDate != "2026-05-10" {
		t.Fatalf("discovery = %+v", rv.Discovery)
	}
}

func TestYearReviewDiscoveryNeedsHistory(t *testing.T) {
	// the very first play ever is inside the year: "new artists" would be everything, so it is left out
	s := &StatsStore{plays: []Play{{T: at(2026, 2, 1, 12), Track: rt(1, "A")}, {T: at(2026, 2, 2, 12), Track: rt(2, "B")}}, genres: map[string]genreEntry{}}
	rv := s.Review(2026, 0, time.Date(2026, 12, 15, 0, 0, 0, 0, time.UTC))
	if rv.Discovery != nil {
		t.Fatalf("no discovery claim without earlier history: %+v", rv.Discovery)
	}
}

func TestYearReviewTimezoneAndEmpty(t *testing.T) {
	// 31 Dec 23:30 UTC is already 1 Jan in UTC+2
	s := &StatsStore{plays: []Play{{T: time.Date(2025, 12, 31, 23, 30, 0, 0, time.UTC).Unix(), Track: rt(1, "A")}}, genres: map[string]genreEntry{}}
	if rv := s.Review(2026, 120, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)); rv.Days[0] != 1 || rv.First == nil || rv.First.Date != "2026-01-01" {
		t.Fatalf("UTC+2 should see it on 1 Jan: %+v", rv.First)
	}
	empty := (&StatsStore{genres: map[string]genreEntry{}}).Review(2026, 0, time.Now())
	if len(empty.Days) != 365 || len(empty.Months) != 12 || empty.Persona.Key != "listener" || empty.Busiest != nil || empty.TopSongs == nil {
		t.Fatalf("empty review must be well formed: %+v", empty)
	}
	if daysIn(2028) != 366 {
		t.Fatal("2028 is a leap year")
	}
}

func TestPickPersona(t *testing.T) {
	base := Metrics{Plays: 100, NightShare: .1, MorningShare: .1, WeekendShare: .28, TopArtistShare: .1, Top3Share: .25, ArtistsPerPlay: .2, PlaysPerSong: 1.5, ActiveDayShare: .3}
	if p := PickPersona(base); p.Key != "listener" {
		t.Fatalf("an average listener has no strong trait: %+v", p)
	}
	night := base
	night.NightShare = .6
	if p := PickPersona(night); p.Key != "night_owl" || len(p.Traits) != 1 {
		t.Fatalf("night = %+v", p)
	}
	both := night
	both.TopArtistShare = .45 // also a loyalist, but less pronounced than the night share
	p := PickPersona(both)
	if p.Key != "night_owl" || p.Second != "The Loyalist" || len(p.Traits) != 2 {
		t.Fatalf("runner-up = %+v", p)
	}
	few := night
	few.Plays = 5
	if p := PickPersona(few); p.Key != "listener" {
		t.Fatalf("too few plays to say: %+v", p)
	}
}

func TestReplaceRankedPlaylist(t *testing.T) {
	pls, err := NewPlaylistStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := Track{ID: "aaaaaaaaaaa", Title: "A", Artist: "X", Thumb: "https://x/a.jpg"}
	b := Track{ID: "bbbbbbbbbbb", Title: "B", Artist: "X", Thumb: "https://x/b.jpg"}
	first, err := pls.ReplaceRanked("Your Top Songs 2026", []Track{a, b, a})
	if err != nil || first.Count != 2 || first.Thumb != a.Thumb {
		t.Fatalf("first = %+v %v (cover is the first song, duplicates collapse)", first, err)
	}
	second, err := pls.ReplaceRanked("Your Top Songs 2026", []Track{b, a})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, s := range pls.List() {
		if s.Name == "Your Top Songs 2026" {
			n++
			if s.ID != second.ID || s.Thumb != b.Thumb {
				t.Fatalf("summary = %+v", s)
			}
		}
	}
	if n != 1 {
		t.Fatalf("the playlist must be replaced, not duplicated: %d", n)
	}
	p, _ := pls.Get(second.ID)
	if len(p.Tracks) != 2 || p.Tracks[0].ID != b.ID {
		t.Fatalf("order must be kept: %+v", p.Tracks)
	}
	if _, ok := pls.Get(likedID); !ok {
		t.Fatal("Liked songs must survive")
	}
}

func TestVarietyMetric(t *testing.T) {
	even := &StatsStore{genres: map[string]genreEntry{}}
	skewed := &StatsStore{genres: map[string]genreEntry{}}
	for i := 0; i < 40; i++ {
		even.plays = append(even.plays, Play{T: at(2026, 3, 1+i%20, 12), Track: rt(i, fmt.Sprintf("A%d", i%4))})
		a := "Big"
		if i%10 == 0 {
			a = "Small"
		}
		skewed.plays = append(skewed.plays, Play{T: at(2026, 3, 1+i%20, 12), Track: rt(i, a)})
	}
	now := time.Date(2026, 12, 15, 0, 0, 0, 0, time.UTC)
	ev, sk := even.Review(2026, 0, now).Metrics.Variety, skewed.Review(2026, 0, now).Metrics.Variety
	if ev < 0.99 || sk > 0.6 || ev <= sk {
		t.Fatalf("variety even=%.2f skewed=%.2f", ev, sk)
	}
}
