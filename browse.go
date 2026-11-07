package main

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"
)

// ---------------------------------------------------------------------------
// Typed items: songs, artists, albums, playlists, mixes
// ---------------------------------------------------------------------------

// Item is a search/browse result. For songs ID is the videoId and the Track
// fields are filled; for everything else ID is the browseId (or playlistId for
// mixes) and Subtitle describes it.
type Item struct {
	Track
	Type     string `json:"type"` // song | artist | album | playlist | mix
	Subtitle string `json:"subtitle,omitempty"`
}

type Section struct {
	Title string `json:"title"`
	Items []Item `json:"items"`
}

type Artist struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Thumb        string    `json:"thumb"`
	Subtitle     string    `json:"subtitle,omitempty"` // monthly listeners / subscribers
	Description  string    `json:"description,omitempty"`
	RadioID      string    `json:"radioId,omitempty"`
	RadioParams  string    `json:"radioParams,omitempty"`
	RadioVideoID string    `json:"radioVideoId,omitempty"`
	Songs        []Item    `json:"songs"`
	SongsBrowse  string    `json:"songsBrowse,omitempty"`
	SongsParams  string    `json:"songsParams,omitempty"`
	Sections     []Section `json:"sections"`
}

type Album struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	ArtistID string `json:"artistId,omitempty"`
	Year     string `json:"year,omitempty"`
	Thumb    string `json:"thumb"`
	Tracks   []Item `json:"tracks"`
}

// Search filter params on YouTube Music.
var searchFilters = map[string]string{
	"songs":   "EgWKAQIIAWoKEAkQBRAKEAMQBA%3D%3D",
	"artists": "EgWKAQIgAWoKEAkQBRAKEAMQBA%3D%3D",
	"albums":  "EgWKAQIYAWoKEAkQBRAKEAMQBA%3D%3D",
}

func searchRaw(ctx context.Context, q, kind string) ([]Item, error) {
	body := map[string]any{"query": q}
	if f, ok := searchFilters[kind]; ok {
		p, _ := url.QueryUnescape(f)
		body["params"] = p
	}
	resp, err := innertube(ctx, webRemix, "search", body)
	if err != nil {
		return nil, err
	}
	return parseSearch(resp), nil
}

// Search runs a typed search. kind "all" merges the songs-only search (which has
// full artist/album info) with artists and albums from the unfiltered one.
func Search(ctx context.Context, q, kind string) ([]Item, error) {
	if kind != "all" && kind != "" {
		return searchRaw(ctx, q, kind)
	}
	type res struct {
		items []Item
		err   error
	}
	songsC, mixedC := make(chan res, 1), make(chan res, 1)
	go func() { i, e := searchRaw(ctx, q, "songs"); songsC <- res{i, e} }()
	go func() { i, e := searchRaw(ctx, q, ""); mixedC <- res{i, e} }()
	songs, mixed := <-songsC, <-mixedC
	if songs.err != nil && mixed.err != nil {
		return nil, songs.err
	}
	out := append([]Item{}, songs.items...)
	for _, it := range mixed.items {
		if it.Type == "artist" || it.Type == "album" || it.Type == "playlist" {
			out = append(out, it)
		}
	}
	return out, nil
}

func parseSearch(resp map[string]any) []Item {
	var items []map[string]any
	findAll(resp, "musicResponsiveListItemRenderer", &items)
	return itemsFromList(items)
}

func itemsFromList(items []map[string]any) []Item {
	seen := map[string]bool{}
	var out []Item
	for _, it := range items {
		x, ok := itemFromList(it)
		if !ok || seen[x.Type+x.ID] {
			continue
		}
		seen[x.Type+x.ID] = true
		out = append(out, x)
	}
	return out
}

// ---------------------------------------------------------------------------
// Parsing helpers
// ---------------------------------------------------------------------------

func runsOf(v any) []map[string]any {
	runs, _ := get(v, "runs").([]any)
	var out []map[string]any
	for _, r := range runs {
		if m, ok := r.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func joinRuns(v any) string {
	var sb strings.Builder
	for _, r := range runsOf(v) {
		sb.WriteString(str(r, "text"))
	}
	return strings.TrimSpace(sb.String())
}

func runBrowseID(r map[string]any) string {
	return str(r, "navigationEndpoint", "browseEndpoint", "browseId")
}

func isArtistID(id string) bool { return strings.HasPrefix(id, "UC") || strings.HasPrefix(id, "MPLA") }
func isAlbumID(id string) bool  { return strings.HasPrefix(id, "MPRE") }

// bylineInfo classifies byline runs using their navigation endpoints: artist
// links start with UC, album links with MPRE. Anything left over that looks
// like m:ss is the duration.
type bylineInfo struct {
	artists  []string
	artistID string
	album    string
	albumID  string
	duration string
	plain    []string // runs without endpoints, separators removed
}

func parseByline(runs []map[string]any) bylineInfo {
	var b bylineInfo
	for _, r := range runs {
		text := str(r, "text")
		id := runBrowseID(r)
		switch {
		case isArtistID(id):
			b.artists = append(b.artists, text)
			if b.artistID == "" {
				b.artistID = id
			}
		case isAlbumID(id):
			b.album, b.albumID = text, id
		case isDuration(strings.TrimSpace(text)):
			b.duration = strings.TrimSpace(text)
		case isPlayCount(text):
			// "76M plays": not an artist or album
		case !isSep(text):
			b.plain = append(b.plain, text)
		}
	}
	return b
}

func thumbOf(v any) string { return lastThumb(v) }

// bigThumb asks googleusercontent for a larger square image.
func bigThumb(v any, size string) string {
	u := str(v, -1, "url")
	if i := strings.Index(u, "=w"); i > 0 && strings.Contains(u, "googleusercontent") {
		u = u[:i] + "=w" + size + "-h" + size + "-l90-rj"
	}
	return u
}

func firstVideoID(it map[string]any, cols []any) string {
	if id := str(it, "playlistItemData", "videoId"); id != "" {
		return id
	}
	if id := str(it, "overlay", "musicItemThumbnailOverlayRenderer", "content",
		"musicPlayButtonRenderer", "playNavigationEndpoint", "watchEndpoint", "videoId"); id != "" {
		return id
	}
	if r := runsOf(get(cols, 0, "musicResponsiveListItemFlexColumnRenderer", "text")); len(r) > 0 {
		return str(r[0], "navigationEndpoint", "watchEndpoint", "videoId")
	}
	return ""
}

// itemFromList parses a musicResponsiveListItemRenderer (search results, shelf
// rows, album/playlist rows).
func itemFromList(it map[string]any) (Item, bool) {
	cols, _ := it["flexColumns"].([]any)
	title := joinRuns(get(cols, 0, "musicResponsiveListItemFlexColumnRenderer", "text"))
	thumb := thumbOf(get(it, "thumbnail", "musicThumbnailRenderer", "thumbnail", "thumbnails"))

	if vid := firstVideoID(it, cols); vid != "" {
		var all []map[string]any
		for i := 1; i < len(cols); i++ {
			all = append(all, runsOf(get(cols, i, "musicResponsiveListItemFlexColumnRenderer", "text"))...)
		}
		b := parseByline(all)
		if len(b.artists) == 0 && b.album == "" && len(b.plain) > 0 {
			// No endpoints (older/other layouts): [type?] artist [album].
			plain := b.plain
			if len(plain) > 0 && (plain[0] == "Song" || plain[0] == "Música" || plain[0] == "Video" || plain[0] == "Vídeo") {
				plain = plain[1:]
			}
			if len(plain) >= 2 {
				b.album = plain[len(plain)-1]
				b.artists = plain[:len(plain)-1]
			} else if len(plain) == 1 {
				b.artists = plain
			}
		}
		if b.duration == "" {
			b.duration = joinRuns(get(it, "fixedColumns", 0, "musicResponsiveListItemFixedColumnRenderer", "text"))
		}
		return Item{Type: "song", Track: Track{ID: vid, Title: title, Artist: strings.Join(b.artists, ", "),
			ArtistID: b.artistID, Album: b.album, AlbumID: b.albumID, Duration: b.duration, Thumb: thumb}}, true
	}

	bid := str(it, "navigationEndpoint", "browseEndpoint", "browseId")
	typ := ""
	switch {
	case isArtistID(bid):
		typ = "artist"
	case isAlbumID(bid):
		typ = "album"
	case strings.HasPrefix(bid, "VL"):
		typ = "playlist"
	}
	if typ == "" || title == "" {
		return Item{}, false
	}
	sub := joinRuns(get(cols, 1, "musicResponsiveListItemFlexColumnRenderer", "text"))
	if typ == "artist" && strings.HasPrefix(strings.ToLower(sub), "profile") {
		return Item{}, false // plain channel profiles, not music artists
	}
	return Item{Type: typ, Subtitle: sub, Track: Track{ID: bid, Title: title, Thumb: thumb}}, true
}

// itemFromTwoRow parses a musicTwoRowItemRenderer (carousel cards).
func itemFromTwoRow(it map[string]any) (Item, bool) {
	title := joinRuns(it["title"])
	sub := joinRuns(it["subtitle"])
	thumb := thumbOf(get(it, "thumbnailRenderer", "musicThumbnailRenderer", "thumbnail", "thumbnails"))
	if title == "" {
		return Item{}, false
	}
	if vid := str(it, "navigationEndpoint", "watchEndpoint", "videoId"); vid != "" {
		var artist string
		runs := runsOf(it["subtitle"])
		b := parseByline(runs)
		artist = strings.Join(b.artists, ", ")
		if artist == "" && len(b.plain) > 0 {
			artist = b.plain[0]
		}
		return Item{Type: "song", Track: Track{ID: vid, Title: title, Artist: artist, ArtistID: b.artistID,
			Album: b.album, AlbumID: b.albumID, Duration: b.duration, Thumb: thumb}}, true
	}
	if pl := str(it, "navigationEndpoint", "watchPlaylistEndpoint", "playlistId"); pl != "" {
		return Item{Type: "mix", Subtitle: sub, Track: Track{ID: pl, Title: title, Thumb: thumb}}, true
	}
	bid := str(it, "navigationEndpoint", "browseEndpoint", "browseId")
	typ := ""
	switch {
	case isArtistID(bid):
		typ = "artist"
	case isAlbumID(bid):
		typ = "album"
	case strings.HasPrefix(bid, "VL"):
		typ = "playlist"
	}
	if typ == "" {
		return Item{}, false
	}
	return Item{Type: typ, Subtitle: sub, Track: Track{ID: bid, Title: title, Thumb: thumb}}, true
}

// sectionFromShelf parses carousel/list shelves into a titled Section.
func sectionFromShelf(kind string, shelf map[string]any) (Section, bool) {
	var title string
	switch kind {
	case "musicCarouselShelfRenderer":
		title = joinRuns(get(shelf, "header", "musicCarouselShelfBasicHeaderRenderer", "title"))
	default:
		title = joinRuns(shelf["title"])
	}
	contents, _ := shelf["contents"].([]any)
	var items []Item
	for _, c := range contents {
		m, _ := c.(map[string]any)
		if m == nil {
			continue
		}
		if two, ok := m["musicTwoRowItemRenderer"].(map[string]any); ok {
			if x, ok := itemFromTwoRow(two); ok {
				items = append(items, x)
			}
		} else if li, ok := m["musicResponsiveListItemRenderer"].(map[string]any); ok {
			if x, ok := itemFromList(li); ok {
				items = append(items, x)
			}
		}
	}
	return Section{Title: title, Items: items}, len(items) > 0
}

func sectionContents(resp map[string]any) []any {
	c, _ := get(resp, "contents", "singleColumnBrowseResultsRenderer", "tabs", 0, "tabRenderer", "content", "sectionListRenderer", "contents").([]any)
	return c
}

// ---------------------------------------------------------------------------
// Home, artist, album, playlist rows
// ---------------------------------------------------------------------------

func Home(ctx context.Context) ([]Section, error) {
	resp, err := innertube(ctx, webRemix, "browse", map[string]any{"browseId": "FEmusic_home"})
	if err != nil {
		return nil, err
	}
	var out []Section
	for _, c := range sectionContents(resp) {
		m, _ := c.(map[string]any)
		for kind, v := range m {
			if shelf, ok := v.(map[string]any); ok && (kind == "musicCarouselShelfRenderer" || kind == "musicShelfRenderer") {
				if sec, ok := sectionFromShelf(kind, shelf); ok {
					out = append(out, sec)
				}
			}
		}
	}
	return out, nil
}

func GetArtist(ctx context.Context, id string) (Artist, error) {
	resp, err := innertube(ctx, webRemix, "browse", map[string]any{"browseId": id})
	if err != nil {
		return Artist{}, err
	}
	a := Artist{ID: id, Songs: []Item{}, Sections: []Section{}}
	for _, key := range []string{"musicImmersiveHeaderRenderer", "musicVisualHeaderRenderer"} {
		h, _ := get(resp, "header", key).(map[string]any)
		if h == nil {
			continue
		}
		a.Name = joinRuns(h["title"])
		a.Thumb = bigThumb(get(h, "thumbnail", "musicThumbnailRenderer", "thumbnail", "thumbnails"), "800")
		a.Subtitle = joinRuns(h["monthlyListenerCount"])
		if a.Subtitle == "" {
			a.Subtitle = joinRuns(get(h, "subscriptionButton", "subscribeButtonRenderer", "subscriberCountText"))
		}
		a.Description = joinRuns(h["description"])
		for _, ep := range []string{"watchPlaylistEndpoint", "watchEndpoint"} {
			if pl := str(h, "startRadioButton", "buttonRenderer", "navigationEndpoint", ep, "playlistId"); pl != "" {
				a.RadioID = pl
				a.RadioParams = str(h, "startRadioButton", "buttonRenderer", "navigationEndpoint", ep, "params")
				a.RadioVideoID = str(h, "startRadioButton", "buttonRenderer", "navigationEndpoint", ep, "videoId")
				break
			}
		}
		break
	}
	if a.Name == "" {
		return a, errors.New("artist not found")
	}
	for _, c := range sectionContents(resp) {
		m, _ := c.(map[string]any)
		if shelf, ok := m["musicShelfRenderer"].(map[string]any); ok {
			if sec, ok := sectionFromShelf("musicShelfRenderer", shelf); ok {
				for _, it := range sec.Items {
					if it.Type == "song" {
						if it.Artist == "" {
							it.Artist, it.ArtistID = a.Name, id
						}
						a.Songs = append(a.Songs, it)
					}
				}
			}
			a.SongsBrowse = str(shelf, "bottomEndpoint", "browseEndpoint", "browseId")
			a.SongsParams = str(shelf, "bottomEndpoint", "browseEndpoint", "params")
		} else if shelf, ok := m["musicCarouselShelfRenderer"].(map[string]any); ok {
			if sec, ok := sectionFromShelf("musicCarouselShelfRenderer", shelf); ok {
				a.Sections = append(a.Sections, sec)
			}
		} else if d, ok := m["musicDescriptionShelfRenderer"].(map[string]any); ok && a.Description == "" {
			a.Description = joinRuns(d["description"])
		}
	}
	return a, nil
}

// BrowseSongs lists every song of an artist (or any playlist-like browse page).
func BrowseSongs(ctx context.Context, browseID, params string) ([]Item, error) {
	body := map[string]any{"browseId": browseID}
	if params != "" {
		body["params"] = params
	}
	resp, err := innertube(ctx, webRemix, "browse", body)
	if err != nil {
		return nil, err
	}
	var rows []map[string]any
	findAll(resp, "musicResponsiveListItemRenderer", &rows)
	var songs []Item
	for _, x := range itemsFromList(rows) {
		if x.Type == "song" {
			songs = append(songs, x)
		}
	}
	return songs, nil
}

func GetAlbum(ctx context.Context, id string) (Album, error) {
	resp, err := innertube(ctx, webRemix, "browse", map[string]any{"browseId": id})
	if err != nil {
		return Album{}, err
	}
	al := Album{ID: id, Tracks: []Item{}}
	var h map[string]any
	for _, key := range []string{"musicResponsiveHeaderRenderer", "musicDetailHeaderRenderer"} {
		var found []map[string]any
		findAll(resp, key, &found)
		if len(found) > 0 {
			h = found[0]
			break
		}
	}
	if h != nil {
		al.Title = joinRuns(h["title"])
		al.Thumb = bigThumb(get(h, "thumbnail", "musicThumbnailRenderer", "thumbnail", "thumbnails"), "600")
		var runs []map[string]any
		runs = append(runs, runsOf(h["straplineTextOne"])...)
		runs = append(runs, runsOf(h["subtitle"])...)
		b := parseByline(runs)
		al.Artist, al.ArtistID = strings.Join(b.artists, ", "), b.artistID
		for _, r := range runs {
			if t := strings.TrimSpace(str(r, "text")); len(t) == 4 && strings.Trim(t, "0123456789") == "" {
				al.Year = t
			}
		}
	}
	if al.Title == "" {
		return al, errors.New("album not found")
	}
	var rows []map[string]any
	findAll(resp, "musicResponsiveListItemRenderer", &rows)
	for _, x := range itemsFromList(rows) {
		if x.Type != "song" {
			continue
		}
		if x.Thumb == "" {
			x.Thumb = al.Thumb
		}
		x.Album, x.AlbumID = al.Title, id
		if x.Artist == "" {
			x.Artist, x.ArtistID = al.Artist, al.ArtistID
		}
		al.Tracks = append(al.Tracks, x)
	}
	return al, nil
}

// ---------------------------------------------------------------------------
// Radio / track lookup
// ---------------------------------------------------------------------------

// Radio returns the "up next" mix YouTube Music generates for a song.
func Radio(ctx context.Context, id string) ([]Track, error) {
	return RadioFor(ctx, id, "RDAMVM"+id, "")
}

// RadioFor loads a watch playlist (song radio, artist mix, ...).
func RadioFor(ctx context.Context, videoID, playlistID, params string) ([]Track, error) {
	body := map[string]any{"isAudioOnly": true}
	if videoID != "" {
		body["videoId"] = videoID
	}
	if playlistID != "" {
		body["playlistId"] = playlistID
	}
	if params != "" {
		body["params"] = params
	}
	resp, err := innertube(ctx, webRemix, "next", body)
	if err != nil {
		return nil, err
	}
	return parseRadio(resp), nil
}

func parseRadio(resp map[string]any) []Track {
	var items []map[string]any
	findAll(resp, "playlistPanelVideoRenderer", &items)
	seen := map[string]bool{}
	var tracks []Track
	for _, it := range items {
		id := str(it, "videoId")
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		b := parseByline(runsOf(it["longBylineText"]))
		t := Track{
			ID:       id,
			Title:    joinRuns(it["title"]),
			Duration: joinRuns(it["lengthText"]),
			Thumb:    thumbOf(get(it, "thumbnail", "thumbnails")),
			Artist:   strings.Join(b.artists, ", "),
			ArtistID: b.artistID,
			Album:    b.album,
			AlbumID:  b.albumID,
		}
		if len(b.artists) == 0 && len(b.plain) > 0 { // no endpoints: artist • album
			t.Artist = b.plain[0]
			if len(b.plain) > 1 {
				t.Album = b.plain[1]
			}
		}
		tracks = append(tracks, t)
	}
	return tracks
}

// TrackInfo resolves a video id (e.g. pasted from a YouTube link) into a Track.
func TrackInfo(ctx context.Context, id string) (Track, error) {
	list, err := RadioFor(ctx, id, "", "")
	if err != nil {
		return Track{}, err
	}
	for _, t := range list {
		if t.ID == id {
			return t, nil
		}
	}
	return Track{}, errors.New("track not found")
}

var playCountRe = regexp.MustCompile(`(?i)^[\d.,]+\s*[kmb]?\s*(plays|views|streams|reproduções|visualizações)$`)

func isPlayCount(s string) bool { return playCountRe.MatchString(strings.TrimSpace(s)) }

var videoFilter = "EgWKAQIQAWoKEAkQBRAKEAMQBA%3D%3D"

// FindClip looks for the official music video of a song (YouTube Music marks
// those MUSIC_VIDEO_TYPE_OMV) and returns its video id, or "" when there isn't
// a confident match. Covers, live takes and fan uploads are ignored.
func FindClip(ctx context.Context, title, artist string, durSec int) (string, error) {
	artist = strings.TrimSpace(strings.Split(artist, ",")[0])
	base := strings.TrimSpace(baseTitle(title))
	if base == "" || artist == "" {
		return "", nil
	}
	p, _ := url.QueryUnescape(videoFilter)
	resp, err := innertube(ctx, webRemix, "search", map[string]any{"query": artist + " " + base, "params": p})
	if err != nil {
		return "", err
	}
	var rows []map[string]any
	findAll(resp, "musicResponsiveListItemRenderer", &rows)
	for _, it := range rows {
		cols, _ := it["flexColumns"].([]any)
		runs := runsOf(get(cols, 0, "musicResponsiveListItemFlexColumnRenderer", "text"))
		if len(runs) == 0 || str(runs[0], "navigationEndpoint", "watchEndpoint", "watchEndpointMusicSupportedConfigs", "watchEndpointMusicConfig", "musicVideoType") != "MUSIC_VIDEO_TYPE_OMV" {
			continue
		}
		x, ok := itemFromList(it)
		if !ok || x.Type != "song" {
			continue
		}
		if !strings.Contains(fold(x.Artist), fold(artist)) && !strings.Contains(fold(artist), fold(x.Artist)) {
			continue
		}
		if fold(baseTitle(x.Title)) != fold(base) {
			continue
		}
		if d := durationSeconds(x.Duration); durSec > 0 && x.Duration != "" && absInt(d-durSec) > 60 {
			continue
		}
		return x.ID, nil
	}
	return "", nil
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
