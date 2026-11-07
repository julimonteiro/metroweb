package main

import (
	"encoding/json"
	"testing"
)

const searchFixture = `{"contents":{"tabbedSearchResultsRenderer":{"tabs":[{"tabRenderer":{"content":{"sectionListRenderer":{"contents":[{"musicShelfRenderer":{"contents":[
{"musicResponsiveListItemRenderer":{
 "thumbnail":{"musicThumbnailRenderer":{"thumbnail":{"thumbnails":[{"url":"https://lh3.googleusercontent.com/x=w60-h60-l90-rj"},{"url":"https://lh3.googleusercontent.com/x=w120-h120-l90-rj"}]}}},
 "flexColumns":[
  {"musicResponsiveListItemFlexColumnRenderer":{"text":{"runs":[{"text":"Freak on a Leash"}]}}},
  {"musicResponsiveListItemFlexColumnRenderer":{"text":{"runs":[{"text":"Korn"},{"text":" • "},{"text":"Follow the Leader"},{"text":" • "},{"text":"4:15"}]}}}
 ],
 "playlistItemData":{"videoId":"abcdefghijk"}}}
]}}]}}}}]}}}`

const radioFixture = `{"contents":{"x":[{"playlistPanelVideoRenderer":{"videoId":"abcdefghijk","title":{"runs":[{"text":"Freak on a Leash"}]},"longBylineText":{"runs":[{"text":"Korn"},{"text":" • "},{"text":"Follow the Leader"},{"text":" • "},{"text":"1998"}]},"lengthText":{"runs":[{"text":"4:15"}]},"thumbnail":{"thumbnails":[{"url":"u"}]}}},
{"playlistPanelVideoRenderer":{"videoId":"zzzzzzzzzzz","title":{"runs":[{"text":"Chop Suey!"}]},"longBylineText":{"runs":[{"text":"System Of A Down"}]}}}]}}`

func TestParseSearch(t *testing.T) {
	var m map[string]any
	json.Unmarshal([]byte(searchFixture), &m)
	got := parseSearch(m)
	if len(got) != 1 {
		t.Fatalf("got %d tracks", len(got))
	}
	tr := got[0]
	if tr.ID != "abcdefghijk" || tr.Title != "Freak on a Leash" || tr.Artist != "Korn" ||
		tr.Album != "Follow the Leader" || tr.Duration != "4:15" {
		t.Fatalf("bad parse: %+v", tr)
	}
	if tr.Thumb != "https://lh3.googleusercontent.com/x=w400-h400-l90-rj" {
		t.Fatalf("bad thumb: %s", tr.Thumb)
	}
}

func TestParseRadio(t *testing.T) {
	var m map[string]any
	json.Unmarshal([]byte(radioFixture), &m)
	got := parseRadio(m)
	if len(got) != 2 || got[0].Artist != "Korn" || got[1].Title != "Chop Suey!" || got[0].Duration != "4:15" {
		t.Fatalf("bad parse: %+v", got)
	}
}

const itemFixture = `{"a":[
{"musicResponsiveListItemRenderer":{"flexColumns":[
 {"musicResponsiveListItemFlexColumnRenderer":{"text":{"runs":[{"text":"Get Lucky","navigationEndpoint":{"watchEndpoint":{"videoId":"abcdefghijk"}}}]}}},
 {"musicResponsiveListItemFlexColumnRenderer":{"text":{"runs":[{"text":"Daft Punk","navigationEndpoint":{"browseEndpoint":{"browseId":"UCaaa"}}},{"text":" & "},{"text":"Pharrell","navigationEndpoint":{"browseEndpoint":{"browseId":"UCbbb"}}},{"text":" • "},{"text":"Random Access Memories","navigationEndpoint":{"browseEndpoint":{"browseId":"MPREb_x"}}},{"text":" • "},{"text":"76M plays"},{"text":"6:10"}]}}}]}},
{"musicResponsiveListItemRenderer":{"navigationEndpoint":{"browseEndpoint":{"browseId":"UCaaa"}},"flexColumns":[
 {"musicResponsiveListItemFlexColumnRenderer":{"text":{"runs":[{"text":"Daft Punk"}]}}},
 {"musicResponsiveListItemFlexColumnRenderer":{"text":{"runs":[{"text":"Artist"},{"text":" • "},{"text":"67M monthly audience"}]}}}]}},
{"musicResponsiveListItemRenderer":{"navigationEndpoint":{"browseEndpoint":{"browseId":"MPREb_x"}},"flexColumns":[
 {"musicResponsiveListItemFlexColumnRenderer":{"text":{"runs":[{"text":"Random Access Memories"}]}}},
 {"musicResponsiveListItemFlexColumnRenderer":{"text":{"runs":[{"text":"Album • Daft Punk • 2013"}]}}}]}}
]}`

func TestParseTypedItems(t *testing.T) {
	var m map[string]any
	json.Unmarshal([]byte(itemFixture), &m)
	got := parseSearch(m)
	if len(got) != 3 {
		t.Fatalf("got %d items: %+v", len(got), got)
	}
	s := got[0]
	if s.Type != "song" || s.ID != "abcdefghijk" || s.Artist != "Daft Punk, Pharrell" || s.ArtistID != "UCaaa" ||
		s.Album != "Random Access Memories" || s.AlbumID != "MPREb_x" || s.Duration != "6:10" {
		t.Fatalf("bad song: %+v", s)
	}
	if got[1].Type != "artist" || got[1].ID != "UCaaa" || got[2].Type != "album" || got[2].ID != "MPREb_x" {
		t.Fatalf("bad cards: %+v %+v", got[1], got[2])
	}
}
