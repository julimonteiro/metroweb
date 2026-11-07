# syntax=docker/dockerfile:1
FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /metroweb .

FROM node:22-bookworm-slim
# yt-dlp[default] brings yt-dlp-ejs; Node is the JS runtime it uses to solve YouTube's
# challenges. Keep yt-dlp fresh: old versions (90+ days) get 403 from YouTube.
RUN apt-get update \
 && apt-get install -y --no-install-recommends ffmpeg python3 python3-pip ca-certificates \
 && rm -rf /var/lib/apt/lists/* \
 && pip3 install --break-system-packages --no-cache-dir -U "yt-dlp[default]" \
 && printf -- '--js-runtimes node\n' > /etc/yt-dlp.conf \
 && useradd -m -u 1000 metroweb && mkdir -p /music /data && chown metroweb:metroweb /music /data

COPY --from=build /metroweb /usr/local/bin/metroweb
USER metroweb
ENV METROWEB_LIBRARY=/music METROWEB_DATA=/data METROWEB_TRUST_PROXY=1
VOLUME ["/music", "/data"]
EXPOSE 8080
ENTRYPOINT ["metroweb", "-addr", "0.0.0.0:8080"]
