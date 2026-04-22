# jackstream

[![Docker Pulls](https://img.shields.io/docker/pulls/OWNER/jackstream.svg)](https://hub.docker.com/r/OWNER/jackstream)
[![GHCR](https://img.shields.io/badge/ghcr.io-OWNER%2Fjackstream-blue)](https://ghcr.io/OWNER/jackstream)
[![CI](https://github.com/OWNER/jackstream/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/OWNER/jackstream/actions)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Node](https://img.shields.io/badge/node-20%2B-brightgreen.svg)](https://nodejs.org)

Self-hosted Stremio addon — query your Jackett and stream torrents instantly via an embedded WebTorrent client. No debrid, no qBittorrent, one Docker container.

## Architecture

```
Stremio app
    │ GET /{config}/manifest.json
    │ GET /{config}/stream/{type}/{id}.json
    ▼
[Express + Stremio addon]
    │
    ├─ Resolve IMDB id → titles via TMDB (24 h cache)
    ├─ Search Jackett in parallel across title variants
    ├─ Parse + sort torrents (quality > source > hdr > seeders)
    └─ Return streams pointing back to /stream/{config}/{torrentId}/{fileIdx}

    │ GET /stream/:config/:torrentId/:fileIdx
    ▼
[WebTorrent singleton]
    │
    ├─ Download .torrent via Jackett (with passkeys)
    ├─ Sequential priority, critical first pieces
    ├─ Wait for STREAM_READY_MB then serve with Range support
    └─ Clean up idle torrents, respect maxConcurrentTorrents
```

## Features

- 🧲 Direct streaming from Jackett results, no debrid, no qBittorrent
- 🔒 Private-tracker friendly: uses Jackett's proxied `.torrent` URLs with passkeys; DHT/LSD disabled
- ⚙️ Per-user credentials encoded in the addon URL (no server-side storage)
- 🎬 Movies + TV series (anime covered via category `5070`)
- 🌍 TMDB-powered title resolution with FR/EN variants for better search coverage
- 🏷️ Quality / source / HDR / codec / audio detection
- 📦 Single Docker container, no volumes, ARM64-ready

## Prerequisites

- A running **Jackett** instance reachable from this container.
- A free **TMDB** API key → <https://www.themoviedb.org/settings/api>.
- **HTTPS** in front of the addon if you expose it outside `127.0.0.1` — Stremio refuses non-localhost HTTP.

## Installation

### Docker (one-liner)

```bash
docker run -d --name jackstream -p 7000:7000 --restart unless-stopped ghcr.io/OWNER/jackstream:latest
```

### Docker Compose

```yaml
version: "3.8"
services:
  jackstream:
    image: ghcr.io/OWNER/jackstream:latest
    container_name: jackstream
    ports:
      - "7000:7000"
    environment:
      - PORT=7000
      - NODE_ENV=production
    restart: unless-stopped
```

### From source (development)

```bash
git clone https://github.com/OWNER/jackstream.git
cd jackstream
npm install
npm start
```

## Configuration

1. Open `http://<host>:7000/configure`.
2. Fill in your Jackett URL + API key, TMDB API key, and the **public URL** where this addon is reachable (e.g. `https://addon.example.com`).
3. Click *Test Jackett* and *Test TMDB* to verify connectivity.
4. Click *Install in Stremio* — the page launches `stremio://...` with your base64url-encoded config.
5. The generated HTTP URL is shown for manual install on other devices.

## HTTPS (required outside localhost)

Stremio refuses non-localhost HTTP addons. Put a reverse proxy in front.

### Caddy (one line)

```
addon.example.com {
    reverse_proxy 127.0.0.1:7000
}
```

### Nginx

```nginx
server {
    listen 443 ssl;
    server_name addon.example.com;
    # … ssl_certificate, ssl_certificate_key …
    location / {
        proxy_pass http://127.0.0.1:7000;
        proxy_set_header Host $host;
        proxy_http_version 1.1;
    }
}
```

## FAQ

**Does it work with private trackers?**
Yes — Jackett proxies the `.torrent` URL so announce URLs carry passkeys. DHT and LSD are disabled in the WebTorrent client to avoid leaking infohashes to the public swarm.

**Why did I see "Stream session expired"?**
The in-memory torrent map has a 2 h TTL. Go back to the source list and click the stream again — it's instant.

**Does it seed after I'm done watching?**
No — V1 stops idle torrents after `TORRENT_IDLE_TIMEOUT_MIN` (default 30 min) and frees the disk. Seeding is on the V2 roadmap.

**Why not debrid?**
Out of scope. The spec targets homelab-only streaming with zero third-party dependencies.

**Why WebTorrent instead of qBittorrent?**
No shared volume, no second container, native HTTP Range support, and the Jackett-proxied `.torrent` carries the passkey so private trackers work out of the box.

**Stremio desktop vs TV?**
Both work. `notWebReady: true` tells Stremio to use the native player rather than the web player.

## Environment variables

| Variable | Default | Meaning |
|---|---|---|
| `PORT` | `7000` | Port to listen on |
| `CACHE_TTL_MINUTES` | `1440` | TMDB cache TTL |
| `REQUEST_TIMEOUT_MS` | `8000` | Jackett/TMDB timeout |
| `STREAM_READY_MB` | `5` | MB to buffer before streaming |
| `STREAM_READY_TIMEOUT_S` | `60` | Max wait for first pieces |
| `TORRENT_IDLE_TIMEOUT_MIN` | `30` | Idle minutes before stopping a torrent |

## Contributing

PRs welcome. Please run `npm test` and `npm run lint` before opening one.

## License

MIT — see [LICENSE](LICENSE).
