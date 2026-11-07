package main

import (
	"testing"
	"time"
)

func tk(id, title, artist, artistID, album, dur string) Track {
	return Track{ID: id, Title: title, Artist: artist, ArtistID: artistID, Album: album, Duration: dur, Thumb: "https://x/" + id}
}

func seed(t *testing.T) (*StatsStore, time.Time) {
	s, err := NewStatsStore(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 7, 20, 0, 0, 0, time.UTC)
	rec := func(tr Track, at time.Time) {
		if _, err := s.Record(tr, at); err != nil {
			t.Fatal(err)
		}
	}
	dp := tk("aaaaaaaaaaa", "One More Time", "Daft Punk, Romanthony", "UCdp", "Discovery", "5:20")
	ww := tk("bbbbbbbbbbb", "Blinding Lights", "The Weeknd", "UCwk", "After Hours", "3:20")
	// October: Weeknd x3, Daft Punk x1. November (this month): Daft Punk x4 on four days.
	for i := 0; i < 3; i++ {
		rec(ww, time.Date(2026, 10, 3+i, 12, 0, 0, 0, time.UTC))
	}
	rec(dp, time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC))
	for d := 4; d <= 7; d++ {
		rec(dp, time.Date(2026, 11, d, 18, 30, 0, 0, time.UTC))
	}
	return s, now
}

func TestStatsAggregation(t *testing.T) {
	s, now := seed(t)
	r := s.Stats("all", 0, now)
	if r.Totals.Plays != 8 || r.Totals.Artists != 2 || r.Totals.Songs != 2 {
		t.Fatalf("totals: %+v", r.Totals)
	}
	if r.TopArtists[0].Name != "Daft Punk" || r.TopArtists[0].Plays != 5 || r.TopArtists[1].Plays != 3 {
		t.Fatalf("top artists: %+v", r.TopArtists)
	}
	if r.TopSongs[0].ID != "aaaaaaaaaaa" || r.TopSongs[0].Plays != 5 {
		t.Fatalf("top songs: %+v", r.TopSongs)
	}
	if r.Totals.Streak != 4 || r.Totals.ActiveDays != 8 {
		t.Fatalf("streak/days: %+v", r.Totals)
	}
	// minutes: 5 plays * 5:20 + 3 plays * 3:20 = 26m40s + 10m = 36 min
	if r.Totals.Minutes != 36 {
		t.Fatalf("minutes = %d", r.Totals.Minutes)
	}
	if r.ByHour[18] != 4 || r.ByHour[12] != 4 {
		t.Fatalf("by hour: %v", r.ByHour)
	}
}

func TestStatsMonthsAndSeries(t *testing.T) {
	s, now := seed(t)
	r := s.Stats("365d", 0, now)
	if r.Granularity != "month" || len(r.Timeline) != 12 {
		t.Fatalf("timeline: %s %d", r.Granularity, len(r.Timeline))
	}
	last, prev := r.Months[11], r.Months[10]
	if last.Key != "2026-11" || last.Plays != 4 || last.TopArtist == nil || last.TopArtist.Name != "Daft Punk" {
		t.Fatalf("november: %+v", last)
	}
	if prev.Key != "2026-10" || prev.Plays != 4 || prev.TopArtist.Name != "The Weeknd" {
		t.Fatalf("october: %+v", prev)
	}
	if len(r.Series) != 2 || r.Series[0].Name != "Daft Punk" || r.Series[0].Data[11] != 4 || r.Series[1].Data[10] != 3 {
		t.Fatalf("series: %+v", r.Series)
	}
	// 30-day view uses days; 90-day view uses weeks
	if r30 := s.Stats("30d", 0, now); r30.Granularity != "day" || len(r30.Timeline) != 30 || r30.Totals.Plays != 5 {
		t.Fatalf("30d: %s %d %d", r30.Granularity, len(r30.Timeline), r30.Totals.Plays)
	}
	if r90 := s.Stats("90d", 0, now); r90.Granularity != "week" {
		t.Fatalf("90d: %s", r90.Granularity)
	}
}

func TestStatsTimezoneMovesBuckets(t *testing.T) {
	s, now := seed(t)
	// 18:30 UTC is 21:30 in UTC+3
	if r := s.Stats("all", 180, now); r.ByHour[21] != 4 || r.ByHour[18] != 0 {
		t.Fatalf("by hour in UTC+3: %v", r.ByHour)
	}
}

func TestRecordDedupesAndValidates(t *testing.T) {
	s, _ := NewStatsStore(t.TempDir(), false)
	tr := tk("aaaaaaaaaaa", "x", "y", "", "", "3:00")
	at := time.Now()
	if ok, _ := s.Record(tr, at); !ok {
		t.Fatal("first play must count")
	}
	if ok, _ := s.Record(tr, at.Add(5*time.Second)); ok {
		t.Fatal("repeat within 20s must be ignored")
	}
	if ok, _ := s.Record(tr, at.Add(5*time.Minute)); !ok {
		t.Fatal("later replay must count")
	}
	if _, err := s.Record(Track{ID: "../../x", Title: "bad"}, at); err == nil {
		t.Fatal("invalid ids must be rejected")
	}
	// history survives a restart
	s2, _ := NewStatsStore(s.dir, false)
	if len(s2.plays) != 2 {
		t.Fatalf("reloaded %d plays", len(s2.plays))
	}
}

func TestGenresWeighted(t *testing.T) {
	s, now := seed(t)
	s.genres = map[string]genreEntry{
		"daft punk":  {Genres: []string{"House", "Electronic"}, At: now.Unix()},
		"the weeknd": {Genres: []string{"Pop"}, At: now.Unix()},
	}
	r := s.Stats("all", 0, now)
	if len(r.Genres) != 3 {
		t.Fatalf("genres: %+v", r.Genres)
	}
	var sum float64
	for _, g := range r.Genres {
		sum += g.Pct
	}
	if sum < 99.5 || sum > 100.5 {
		t.Fatalf("percentages must add up to ~100, got %.1f (%+v)", sum, r.Genres)
	}
	if r.Genres[0].Name != "Pop" && r.Genres[0].Name != "House" {
		t.Fatalf("unexpected leading genre: %+v", r.Genres)
	}
}

func TestStatsEmpty(t *testing.T) {
	s, _ := NewStatsStore(t.TempDir(), false)
	r := s.Stats("all", 0, time.Now())
	if r.Totals.Plays != 0 || r.TopArtists == nil || r.Months == nil || len(r.Months) != 12 {
		t.Fatalf("empty stats must still be well formed: %+v", r.Totals)
	}
}
