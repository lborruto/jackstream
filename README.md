# jackstream

[![Docker Pulls](https://img.shields.io/docker/pulls/lborruto/jackstream.svg)](https://hub.docker.com/r/lborruto/jackstream)
[![GHCR](https://img.shields.io/badge/ghcr.io-lborruto%2Fjackstream-blue)](https://ghcr.io/lborruto/jackstream)
[![CI](https://github.com/lborruto/jackstream/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/lborruto/jackstream/actions)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Node](https://img.shields.io/badge/node-20%2B-brightgreen.svg)](https://nodejs.org)

Self-hosted Stremio addon — query your Jackett and stream torrents instantly via an embedded WebTorrent client. No debrid, no qBittorrent, one Docker container.

## Quick start (homelab on LAN)

1. On your homelab/server:
   ```bash
   docker compose up -d
   ```
2. From any device on the same LAN, open `http://<server-ip>:7000/configure`.
3. Fill in your Jackett URL + API key, your TMDB API key. For **Addon public URL** just enter your server's LAN IP (e.g. `192.168.0.15`) — the page auto-converts it to an HTTPS URL (`https://192-168-0-15.local-ip.medicmobile.org:7001`) so Stremio clients on other devices can reach it with a valid TLS cert.
4. Click *Install in Stremio* — the native Stremio app opens and installs the addon.
5. Pick a movie or episode — Jackett torrents appear as sources.

No domain, no cert management, no per-device trust setup. HTTPS uses a wildcard cert from `local-ip.medicmobile.org` (see [HTTPS details](#https-details) for how it works and its tradeoffs).

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

- 🪶 Static Go binary — ~26 MB image on `scratch` base
- 🧲 Direct streaming from Jackett results, no debrid, no qBittorrent
- 🔒 Private-tracker friendly: uses Jackett's proxied `.torrent` URLs with passkeys; DHT / LSD / µTP disabled
- 🔐 HTTPS out of the box via a bundled wildcard cert — no reverse proxy needed for LAN clients
- ⚙️ Per-user credentials encoded in the addon URL (zero server-side storage)
- 💾 Client-side config persistence (localStorage) + bookmarkable `/<config>/configure` URL
- 🎬 Movies + TV series (anime covered via category `5070`)
- 🌍 TMDB-powered title resolution with FR/EN variants for better search coverage
- 🏷️ Quality / source / HDR / codec / audio detection
- 🎚️ Advanced filters: preferred language, quality floor/ceiling, size bounds, blacklist keywords
- 📦 Single Docker container, no volumes, multi-arch (amd64 + arm64)

## Prerequisites

- A running **Jackett** instance reachable from this container.
- A free **TMDB** API key → <https://www.themoviedb.org/settings/api>.
- Docker + Docker Compose on the host.

## Installation

### Docker (one-liner)

```bash
docker run -d --name jackstream \
  -p 7000:7000 -p 7001:7001 \
  --restart unless-stopped \
  ghcr.io/lborruto/jackstream:latest
```

### Docker Compose

```yaml
services:
  jackstream:
    image: ghcr.io/lborruto/jackstream:latest
    container_name: jackstream
    ports:
      - "7000:7000"   # HTTP (configure page + localhost Stremio)
      - "7001:7001"   # HTTPS (for Stremio clients on other LAN devices)
    restart: unless-stopped
```

### From source (development)

```bash
git clone https://github.com/lborruto/jackstream.git
cd jackstream
go build -o jackstream ./cmd/jackstream
./jackstream
```

Requires Go 1.24+. All deps are stdlib except `github.com/anacrolix/torrent` (fetched automatically on build).

## Configuration

1. Open `http://<server-ip>:7000/configure` from any device on your LAN.
2. Fill in:
   - **Jackett URL** (e.g. `http://<server-ip>:9117`) and its API key.
   - **TMDB API key**.
   - **Addon public URL / LAN IP** — just type your server's LAN IP (e.g. `192.168.0.15`). The page auto-converts it to a valid HTTPS URL (see [HTTPS details](#https-details)).
3. Click *Test Jackett* / *Test TMDB* to verify connectivity (live ping of both).
4. *(Optional)* expand **Advanced filters** to set language preference, quality bounds, size limits, or blacklist keywords.
5. Click *Install in Stremio* — the native Stremio app opens and installs.
6. The HTTPS URL is also shown so you can paste it into Stremio manually on any device (useful for TVs without a browser).

Your inputs are saved to `localStorage` so reloading `/configure` re-populates the form. You can also bookmark `http://<server-ip>:7000/<base64>/configure` to carry the full config between browsers.

## Advanced filters

All optional. Defaults produce the same behavior as earlier versions.

| Setting                      | Effect                                                                                                                                                  |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Preferred audio language** | Sort boost for matching audio (FRENCH / MULTI / VOSTFR / ENG). `FRENCH` also boosts MULTI releases; same for `ENG`. Does not hide non-matching results. |
| **Min / Max quality**        | Hard filter. Drops results above/below the chosen tier.                                                                                                 |
| **Min size (MB)**            | Drops "fake" low-size releases.                                                                                                                         |
| **Max size (GB)**            | Drops oversized releases (useful for slow connections / small disks).                                                                                   |
| **Blacklist keywords**       | Comma-separated list (case-insensitive, word-boundary). Drops results whose title matches any keyword — e.g. `CAM, HDCAM, TELESYNC`.                    |

All filters are stored in the base64-encoded addon URL, so different Stremio installs can use different filter profiles against the same backend.

## HTTPS details

jackstream ships a pre-baked TLS cert + key from <https://local-ip.medicmobile.org> — a community service that issues a public `*.local-ip.medicmobile.org` Let's Encrypt certificate and operates a DNS server where `<dashed-ip>.local-ip.medicmobile.org` resolves to `<ip>`. This gives you valid HTTPS for any LAN IP with zero setup.

The HTTPS listener binds on port **7001** alongside the HTTP listener on **7000**. Stremio clients hit the HTTPS URL; the TCP connection still goes directly to your LAN host.

### Tradeoffs

- The TLS private key is public (by design — same key in every copy of this image). Anyone who can also control DNS on your network could MITM; on a trusted home LAN this is a non-issue. Not suitable for addons exposed to the public internet.
- The service is community-run (Medic Mobile). If it goes offline, bundled certs will eventually expire and you'll need to swap them. Run `./scripts/refresh-certs.sh` before building a new image to pull fresh certs.

### Using your own cert instead

Mount a cert + key and point the addon at them:

```yaml
services:
  jackstream:
    image: ghcr.io/lborruto/jackstream:latest
    ports:
      - "7000:7000"
      - "7001:7001"
    environment:
      - HTTPS_CERT_PATH=/certs/fullchain.pem
      - HTTPS_KEY_PATH=/certs/privkey.pem
    volumes:
      - /path/to/your/certs:/certs:ro
    restart: unless-stopped
```

Works with Caddy-issued certs, mkcert, Let's Encrypt, or anything else.

### Disabling HTTPS

Set `HTTPS_DISABLED=1` to skip the HTTPS listener (HTTP-only). Useful if you front the addon with Caddy / Nginx / Traefik that already terminates TLS.

### Refreshing the bundled cert

The cert expires every ~60 days (Let's Encrypt rotation). Before a rebuild:

```bash
./scripts/refresh-certs.sh
git commit -am "chore: refresh bundled local-ip cert"
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

**The *Install in Stremio* button did nothing / the URL lost the `:7001` port.**
Some Stremio clients (older Windows builds in particular) mishandle the port in `stremio://` deep-link URIs. Workaround: copy the `https://…:7001/<base64>/manifest.json` URL shown below the button, then in Stremio click the puzzle-piece icon → paste into *Add-on Repository URL*.

**Loading failed when I click a stream.**
Most often because the container restarted (e.g. Komodo redeploy) and the in-memory `torrentStore` was wiped while Stremio still had a cached list of stream URLs. Close and re-open the movie in Stremio to force a fresh stream-list fetch.

## Environment variables

| Variable                   | Default                 | Meaning                                  |
| -------------------------- | ----------------------- | ---------------------------------------- |
| `PORT`                     | `7000`                  | Port to listen on                        |
| `CACHE_TTL_MINUTES`        | `1440`                  | TMDB cache TTL                           |
| `REQUEST_TIMEOUT_MS`       | `8000`                  | Jackett/TMDB timeout                     |
| `STREAM_READY_MB`          | `5`                     | MB to buffer before streaming            |
| `STREAM_READY_TIMEOUT_S`   | `60`                    | Max wait for first pieces                |
| `TORRENT_IDLE_TIMEOUT_MIN` | `30`                    | Idle minutes before stopping a torrent   |
| `HTTPS_PORT`               | `7001`                  | Port for the HTTPS listener              |
| `HTTPS_CERT_PATH`          | `./certs/fullchain.pem` | Path to TLS certificate (PEM)            |
| `HTTPS_KEY_PATH`           | `./certs/key.pem`       | Path to TLS private key (PEM)            |
| `HTTPS_DISABLED`           | _(unset)_               | Set to `1` to disable the HTTPS listener |

## Contributing

PRs welcome. Please run `go test ./...` and `go vet ./...` before opening one.

## License

MIT — see [LICENSE](LICENSE).
