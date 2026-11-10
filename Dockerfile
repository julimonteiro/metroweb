# syntax=docker/dockerfile:1
FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /metroweb .

FROM debian:bookworm-slim
COPY --from=denoland/deno:bin /deno /usr/local/bin/deno
# yt-dlp[default] brings yt-dlp-ejs; Deno is the JS runtime it uses to solve YouTube's
# challenges (same setup that works on macOS). Keep yt-dlp fresh: old versions (90+ days) get 403 from YouTube.
RUN apt-get update \
 && apt-get install -y --no-install-recommends ffmpeg python3 python3-pip ca-certificates \
 && rm -rf /var/lib/apt/lists/* \
 && pip3 install --break-system-packages --no-cache-dir -U "yt-dlp[default]" \
 && useradd -m -u 1000 metroweb && mkdir -p /music /data && chown metroweb:metroweb /music /data

COPY --from=build /metroweb /usr/local/bin/metroweb
USER metroweb
# HOME/caches live in /tmp so the root filesystem can be mounted read-only.
ENV METROWEB_LIBRARY=/music METROWEB_DATA=/data HOME=/tmp XDG_CACHE_HOME=/tmp/.cache DENO_DIR=/tmp/.deno
VOLUME ["/music", "/data"]
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=6s --start-period=20s --retries=3 CMD ["metroweb", "-addr", "127.0.0.1:8080", "-healthcheck"]
ENTRYPOINT ["metroweb", "-addr", "0.0.0.0:8080"]
