# metroweb

A self-hosted YouTube Music web client, modelled on [Metrolist](https://github.com/MetrolistGroup/Metrolist) and built for phones and desktops. Search, listen, like songs, build playlists, and **download music into a folder Jellyfin watches**. Go backend (stdlib only) with the frontend embedded in the binary, and its own login so you can host it at home or on a server and reach it from anywhere.

**Features**

- Mobile-first Material 3 UI (bottom navigation, mini player, full-screen player with colors taken from the cover art) and a desktop layout with a sidebar and a persistent player bar
- Search for songs, **artists** and albums; artist pages with top songs, all songs, albums, singles, videos, similar artists, **artist radio**
- Home with quick picks based on what you played, recently played and YouTube Music suggestions
- **Playlists** and **Liked songs**, stored on the server so they follow you across devices
- ♥ likes a song **and** downloads it to your library; the ⋮ menu has radio, play next, queue, add to playlist, download, go to artist/album and **share link**
- Paste a YouTube / YouTube Music link into the search box to open that song, artist, album or playlist
- **Your Jellyfin songs show up in Liked songs by themselves**: about once a week metroweb matches the songs in your Jellyfin library to their YouTube Music versions (strictly: title, artist and length must agree) and likes them, so you can play them here. It only looks up new songs, never re-likes something you un-liked, and has a discreet "Sync now" in the account menu
- **Knows what you already own**: with a Jellyfin URL and API key, songs that are already in your Jellyfin library get a green ✓ and are never downloaded twice (matching by title and artist, ignoring accents and "(Radio Edit)"-style suffixes)
- Downloads land in `Artist/Album/Title.m4a` with tags and a square cover, are never duplicated, and trigger a Jellyfin library rescan if you configure it
- **Lyrics and music video** in the full-screen player (Song · Lyrics · Video switch): time-synced lyrics that follow the song (tap a line to jump there) and the official music video. Opening the video never interrupts the song: the music keeps playing and the clip fades in on top, aligned to the exact moment of the song (the server measures the clip's offset by comparing both audio tracks)
- **Stats**: your top artists, songs and albums, an artists-over-time chart, genre breakdown, month-by-month favourites, listening streak and the hours you listen most (last 30 days, 3 months, year or all time)
- Installable as a PWA

## Run with Docker (recommended)

```sh
git clone https://github.com/julimonteiro/metroweb && cd metroweb
# edit docker-compose.yml: point ./music at the folder Jellyfin watches
docker compose up -d --build
```

On Linux, make sure the music folder is writable by UID 1000 (`sudo chown 1000:1000 music`).

Open `http://localhost:8787`. The initial login is **`admin` / `admin`** (accepted only from your own network until you change it): change it from the account menu (avatar on phones, "admin" in the sidebar on desktop). A banner reminds you until you do.

The image bundles `yt-dlp`, `ffmpeg` and Deno. Rebuild it now and then (`docker compose build --no-cache`) to pick up a fresh `yt-dlp`.

## Run without Docker

You need Go 1.24+, `yt-dlp` and `ffmpeg` on your PATH (plus Deno or Node for yt-dlp's JavaScript challenges).

```sh
go run . -library ~/Music/metroweb
```

| Flag | Env | What it does |
|---|---|---|
| `-addr` | | Listen address (default `127.0.0.1:8787`, this machine only) |
| `-library` | `METROWEB_LIBRARY` | Folder where songs are saved (the one Jellyfin watches) |
| `-jellyfin-url` | `JELLYFIN_URL` | e.g. `http://jellyfin:8096`; enables the automatic rescan |
| `-jellyfin-key` | `JELLYFIN_API_KEY` | API key (Dashboard > API Keys) |
| `-user` / `-password` | `METROWEB_USER` / `METROWEB_PASSWORD` | **Initial** credentials, used on first run only (default `admin`/`admin`) |
| `-data` | `METROWEB_DATA` | Data folder: credentials (hashed), playlists, session secret, stream cache |
| `-genres` | `METROWEB_GENRES=0` | Turn off genre lookups for the Stats page (on by default) |
| `-lyrics` | `METROWEB_LYRICS=0` | Turn off lyrics lookups (on by default) |
| `-trust-proxy` | `METROWEB_TRUST_PROXY=1` | Trust `X-Forwarded-For` (behind a reverse proxy or tunnel) for login rate limiting |

## Security

metroweb is meant to be reachable from the internet, so it is locked down by default:

- **Login required for everything** (except the sign-in page): PBKDF2-hashed password, signed `HttpOnly` + `SameSite=Lax` session cookie (`Secure` over HTTPS), 5-minute lockout after 5 failed attempts per IP plus a global slowdown against distributed guessing. Changing the password signs out every other device.
- **The default `admin`/`admin` password is never accepted from the internet.** While it is still in place, sign-in only works from localhost or a private network (LAN, Docker, Tailscale); anything arriving through a proxy or with forwarding headers counts as remote. Change it first, from home.
- New passwords need at least 10 characters and cannot be trivially guessable.
- **CSRF protection** (cross-site writes are rejected), a strict **Content-Security-Policy** (own scripts only, no framing), HSTS over HTTPS, and no referrer leakage.
- **Everything from the client is validated**: video/browse ids against strict patterns before they reach `yt-dlp`, playlist data sanitised and size-capped, the cover-art proxy limited to Google/YouTube hosts with redirects disabled (no SSRF), request bodies and timeouts bounded.
- **Nothing can run commands on your machine through the app.** The only external programs are `yt-dlp` and `ffmpeg`, always started with fixed arguments and validated ids (never through a shell), and with the user's yt-dlp config ignored.
- **Listens on `127.0.0.1` by default.** Expose it only through HTTPS or a private network. The Docker setup publishes `127.0.0.1:8787` only, runs as a non-root user with a read-only root filesystem, no Linux capabilities, `no-new-privileges`, and write access only to `/music`, `/data` and `/tmp`. Prefer Docker over running the binary directly: that way a bug in a downloader could at worst touch your music folder.

Still your job: keep `yt-dlp` and the image updated, use a strong unique password, and keep HTTPS in front of it.

## Jellyfin integration

Set `JELLYFIN_URL` and `JELLYFIN_API_KEY` (Jellyfin: Dashboard > Advanced > API Keys > +). metroweb then:

- reads the list of songs in your library (titles, artists, durations and item ids; nothing else) every 10 minutes and marks songs you already have, skipping duplicate downloads;
- **plays songs you already have straight from your Jellyfin library**, so they start instantly instead of waiting for YouTube (the API key stays on the server; if a file can't be played it falls back to YouTube);
- treats every song it recognises as yours as **liked** (un-liking one is remembered and respected);
- asks Jellyfin to rescan after it downloads something, and refreshes its own index afterwards;
- once a week (and shortly after the first start) matches every Jellyfin song to its YouTube Music version and adds it to Liked songs; the state is kept in `<data>/import-state.json`.

Jellyfin API keys have admin rights, so keep the key in a private `.env` file (never in git) and only on the server that runs metroweb.

## Reaching it from outside your home

Always use **HTTPS** over the internet, otherwise your password travels in clear text. Set `METROWEB_TRUST_PROXY=1` only when a reverse proxy or tunnel on the same machine is the *only* way in (otherwise anyone could spoof `X-Forwarded-For`). Options:

1. **Tailscale** (simplest and safest): install it on the server and your phone and browse to `http://your-server:8787`; nothing is exposed to the internet. `tailscale serve` gives you HTTPS for free.
2. **A reverse proxy with HTTPS**, the same way you would publish Jellyfin. With Caddy:

   ```
   music.example.com {
       encode zstd gzip
       reverse_proxy localhost:8787
   }
   ```

   (With Cloudflare DNS challenge: add `tls { dns cloudflare {env.CF_API_TOKEN} }` as in the Caddy-Cloudflare build.)
3. **Cloudflare Tunnel**: `cloudflared tunnel --url http://localhost:8787` (or a named tunnel on your domain). HTTPS included.

**Change the `admin` password before exposing the server.** Anyone who is signed in can queue downloads onto your disk.

## When YouTube answers 403

Almost always an outdated `yt-dlp`: versions older than ~90 days stop working. Update it (`pip install -U "yt-dlp[default]"`, or rebuild the Docker image) and make sure a JavaScript runtime (Deno or Node) is available. If it still fails, the [bgutil-ytdlp-pot-provider](https://github.com/Brainicism/bgutil-ytdlp-pot-provider) plugin supplies the PO token YouTube sometimes demands.

Streaming tries the googlevideo URL directly first; if YouTube rejects it (some clients only work with yt-dlp's own downloader) the server lets `yt-dlp` fetch the file into `<data>/cache` and serves that with full seek support.

## API

All routes except `/login` and the PWA files require a session.

| Route | What it does |
|---|---|
| `GET /api/search?q=&type=all\|songs\|artists\|albums` | Typed search |
| `GET /api/home` | YouTube Music home sections |
| `GET /api/artist/{id}` | Artist page (top songs, sections, radio ids) |
| `GET /api/songs?b=&p=` | Every song of an artist / playlist browse id |
| `GET /api/album/{id}` | Album with tracks |
| `GET /api/radio/{videoId}` · `GET /api/mix?id=&p=&v=` | Song radio / artist or playlist mix |
| `GET /api/track/{videoId}` | Resolve one song (used for pasted links) |
| `GET /api/stream/{id}?fmt=opus\|aac&t=&a=` | Audio with Range support: from Jellyfin when the song is in your library (`t`/`a` = title/artist), otherwise from YouTube |
| `GET /api/prefetch/{id}` | Warm up the next song of the queue so it starts without a gap |
| `POST /api/download/{id}` · `GET /api/downloads` | Queue a download / download states |
| `GET/POST /api/playlists` · `GET/PATCH/DELETE /api/playlists/{id}` | Playlists (`liked` is the built-in Liked songs) |
| `POST /api/playlists/{id}/tracks` · `DELETE /api/playlists/{id}/tracks/{trackId}` | Edit a playlist |
| `GET /api/lyrics?title=&artist=&album=&duration=` | Time-synced (or plain) lyrics from LRCLIB |
| `GET /api/clip/info?title=&artist=` · `GET /api/clip/offset?song=&clip=` · `GET /api/clip/{id}` | Find the official music video / stream it (downloaded once into the cache, Range supported) |
| `GET /api/img?u=` | Cover art proxy (Google/YouTube hosts only) |
| `GET /api/mixes?tz=` | Mixes for you (on repeat, rewind, hidden gems, time of day, artists, genres) |
| `GET /api/plays/summary` | Play count and last play time per song (feeds the smart shuffle) |
| `POST /api/plays` · `GET /api/stats?range=30d\|90d\|365d\|all&tz=` | Record a listen (the app reports it after 30 s) / aggregated stats |
| `GET/POST /api/account` | Current user / change username and password |

## Stats and privacy

A song counts as a play after 30 seconds of listening. The history is a plain file on your server (`<data>/plays.jsonl`); nothing is sent to analytics services. For the genre chart, the server asks [MusicBrainz](https://musicbrainz.org) for each artist's genres (only the artist *name* is sent, at most one request per second, results cached in `<data>/genres.json`). Start with `METROWEB_GENRES=0` to turn that off.

## Mixes for you

The Home page opens with mixes built from your own listening history and your Liked songs. They are computed on the server, instantly, only contain songs you already know, and stay the same through the day:

- **On repeat**: what you played most this month.
- **Rewind**: songs you played a lot and haven't in 45+ days.
- **Hidden gems**: Liked songs you haven't played here yet (your Jellyfin songs end up here).
- **Morning / Afternoon / Evening / Late night mix**: what you usually play at this time of day.
- **Artist mixes**: your songs by each of your top 3 artists.
- **Genre mixes**: your two biggest genres (from the Stats genre lookups).

A mix only shows up once it has enough songs, so the list grows as you listen. Open one to play it, shuffle it, add it to a playlist (save it) or download it. `GET /api/mixes?tz=` returns them.

## Smart shuffle

Shuffle is not plain random. When you turn it on, the queue is dealt once, in a smart order (so "next", "previous", the queue list and track prefetching all behave normally):

- **Forgotten favourites first.** Each song is weighted by how much you like it (liked, play count) and how long since you last heard it. Songs you love but haven't played in weeks, and songs you own but never played here, tend to surface early; songs you just heard sink to the end.
- **A dash of luck.** About 15% of the picks ignore the weights, so it stays unpredictable.
- **No repeats.** Every song plays once per round; the next round is dealt fresh and never opens with the artist you just heard.
- **Artists spread out.** The same artist is kept a few songs apart whenever the queue allows it.
- Turning shuffle off puts the queue back in its original order.

The weights use your own listening history (`GET /api/plays/summary`); nothing leaves the server. The algorithm lives in `web/shuffle.js` (tests: `node --test web/shuffle.test.js`).

## Lyrics and music videos

Lyrics come from [LRCLIB](https://lrclib.net), a free open lyrics database; only the song title, artist, album and duration are sent (set `METROWEB_LYRICS=0` to disable). The music video is the official clip YouTube Music lists for the song (covers and fan uploads are ignored); the server downloads it once with `yt-dlp` (720p at most, H.264) into `<data>/cache`, which is capped at 3 GB, and only for ids the app looked up itself, so it cannot be used to fetch arbitrary videos.

## Shortcuts (desktop)

Space play/pause · Shift+←/→ previous/next · `/` search · Esc closes the player and menus.

## Disclaimer

A personal project, not affiliated with YouTube, Google, Metrolist or Jellyfin. Downloading content from YouTube may violate its terms of service and copyright law where you live: only download what you have the right to.
