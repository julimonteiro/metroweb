# metroweb

A self-hosted YouTube Music web client, modelled on [Metrolist](https://github.com/MetrolistGroup/Metrolist) and built for phones and desktops. Search, listen, like songs, build playlists, and **download music into a folder Jellyfin watches**. Go backend (stdlib only) with the frontend embedded in the binary, and its own login so you can host it at home or on a server and reach it from anywhere.

**Features**

- Mobile-first Material 3 UI (bottom navigation, mini player, full-screen player with colors taken from the cover art) and a desktop layout with a sidebar and a persistent player bar
- Search for songs, **artists** and albums; artist pages with top songs, all songs, albums, singles, videos, similar artists, **artist radio**
- Home with quick picks based on what you played, recently played and YouTube Music suggestions
- **Playlists** and **Liked songs**, stored on the server so they follow you across devices
- ♥ likes a song **and** downloads it to your library; the ⋮ menu has radio, play next, queue, add to playlist, download, go to artist/album and **share link**
- Paste a YouTube / YouTube Music link into the search box to open that song, artist, album or playlist
- Downloads land in `Artist/Album/Title.m4a` with tags and a square cover, are never duplicated, and trigger a Jellyfin library rescan if you configure it
- Installable as a PWA

## Run with Docker (recommended)

```sh
git clone https://github.com/julimonteiro/metroweb && cd metroweb
# edit docker-compose.yml: point ./music at the folder Jellyfin watches
docker compose up -d --build
```

On Linux, make sure the music folder is writable by UID 1000 (`sudo chown 1000:1000 music`).

Open `http://localhost:8080`. The initial login is **`admin` / `admin`**: change it from the account menu (avatar on phones, "admin" in the sidebar on desktop). A banner reminds you until you do.

The image bundles `yt-dlp`, `ffmpeg` and Deno. Rebuild it now and then (`docker compose build --no-cache`) to pick up a fresh `yt-dlp`.

## Run without Docker

You need Go 1.24+, `yt-dlp` and `ffmpeg` on your PATH (plus Deno or Node for yt-dlp's JavaScript challenges).

```sh
go run . -library ~/Music/metroweb -addr 0.0.0.0:8080
```

| Flag | Env | What it does |
|---|---|---|
| `-addr` | | Listen address (default `:8080`) |
| `-library` | `METROWEB_LIBRARY` | Folder where songs are saved (the one Jellyfin watches) |
| `-jellyfin-url` | `JELLYFIN_URL` | e.g. `http://jellyfin:8096`; enables the automatic rescan |
| `-jellyfin-key` | `JELLYFIN_API_KEY` | API key (Dashboard > API Keys) |
| `-user` / `-password` | `METROWEB_USER` / `METROWEB_PASSWORD` | **Initial** credentials, used on first run only (default `admin`/`admin`) |
| `-data` | `METROWEB_DATA` | Data folder: credentials (hashed), playlists, session secret, stream cache |
| `-trust-proxy` | `METROWEB_TRUST_PROXY=1` | Trust `X-Forwarded-For` (behind a reverse proxy or tunnel) for login rate limiting |

## Reaching it from outside your home

The server has a login (PBKDF2-hashed password, signed `HttpOnly` session cookie, 5-minute lockout after 5 failed attempts per IP), but **always use HTTPS** when going over the internet, otherwise your password travels in clear text. Options:

1. **Tailscale** (simplest and safest): install it on the server and your phone and browse to `http://your-server:8080`; nothing is exposed to the internet. `tailscale serve` gives you HTTPS for free.
2. **Cloudflare Tunnel**: `cloudflared tunnel --url http://localhost:8080` (or a named tunnel on your domain). HTTPS included.
3. **A reverse proxy** (Caddy, nginx, Traefik) with a Let's Encrypt certificate. With Caddy:

   ```
   music.example.com {
       reverse_proxy localhost:8080
   }
   ```

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
| `GET /api/stream/{id}?fmt=opus\|aac` | Audio proxy with Range support |
| `POST /api/download/{id}` · `GET /api/downloads` | Queue a download / download states |
| `GET/POST /api/playlists` · `GET/PATCH/DELETE /api/playlists/{id}` | Playlists (`liked` is the built-in Liked songs) |
| `POST /api/playlists/{id}/tracks` · `DELETE /api/playlists/{id}/tracks/{trackId}` | Edit a playlist |
| `GET /api/img?u=` | Cover art proxy (Google/YouTube hosts only) |
| `GET/POST /api/account` | Current user / change username and password |

## Shortcuts (desktop)

Space play/pause · Shift+←/→ previous/next · `/` search · Esc closes the player and menus.

## Disclaimer

A personal project, not affiliated with YouTube, Google, Metrolist or Jellyfin. Downloading content from YouTube may violate its terms of service and copyright law where you live: only download what you have the right to.
