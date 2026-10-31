<h1 align="center">jackstream</h1>

<p align="center">
  <strong>A Stremio addon that streams torrents from your Jackett. Self-hosted, one container.</strong><br>
  Pick a movie or an episode in Stremio and the torrents Jackett finds appear as sources; the addon downloads what you watch and serves it to the player. A static Go binary in a 20 MB image, no debrid, no qBittorrent, no volumes.
</p>

<p align="center">
  <a href="https://hub.docker.com/r/lborruto/jackstream"><img alt="Docker pulls" src="https://img.shields.io/docker/pulls/lborruto/jackstream.svg"></a>
  <a href="https://ghcr.io/lborruto/jackstream"><img alt="GHCR" src="https://img.shields.io/badge/ghcr.io-lborruto%2Fjackstream-blue"></a>
  <a href="https://github.com/lborruto/jackstream/actions/workflows/docker-publish.yml"><img alt="CI" src="https://github.com/lborruto/jackstream/actions/workflows/docker-publish.yml/badge.svg"></a>
  <a href="LICENSE"><img alt="Licence: MIT" src="https://img.shields.io/badge/license-MIT-green.svg"></a>
</p>

## What it does

- **Jackett results as Stremio sources.** Open a movie or an episode (anime included, through
  category `5070`) and the torrents Jackett returns are listed as streams, sorted by quality,
  source, HDR and seeders. Nothing goes through a debrid service or a separate torrent client.
- **Streams through its own BitTorrent client.** The addon embeds anacrolix/torrent. Pieces are
  fetched in playback order with 16 MB of readahead, so only the parts you watch are downloaded.
  Idle torrents, and their pieces on disk, are removed after 30 minutes.
- **Private-tracker friendly.** The `.torrent` is downloaded through Jackett's proxied URL, so
  the announce URLs carry your passkeys. DHT, LSD and µTP are off.
- **HTTPS on your LAN with no certificate to manage.** A wildcard certificate from
  `local-ip.medicmobile.org` gives any LAN IP a valid HTTPS name, which Stremio clients on other
  devices need. The addon fetches the current certificate itself and swaps it in while running:
  no reverse proxy, no rebuild when it rotates. See [HTTPS](#https) for how it works and what it
  costs.
- **Better searches through TMDB.** The IMDB id is resolved to the localised and original titles,
  and Jackett is searched in parallel for each variant.
- **Filters of your own.** Preferred audio language, a quality floor and ceiling, size bounds and
  blacklisted keywords, all optional. Quality, source, HDR, codec and audio are detected from
  the release name.
- **Your settings live in the addon URL.** Credentials and filters are encoded in the URL that
  Stremio installs; the server stores nothing. The configure page keeps your inputs in the
  browser's localStorage, and `/<config>/configure` can be bookmarked.
- **Small.** A static Go binary on a `scratch` image: about 20 MB, under 50 ms to start, about
  30 MB of memory. One container, no volumes, `linux/amd64` and `linux/arm64`.

## Get started

**You need:** Docker with Compose on the host, a running [Jackett](https://github.com/Jackett/Jackett)
the container can reach, a free [TMDB API key](https://www.themoviedb.org/settings/api), and the
Stremio app on the devices you watch on.

Put this in a `docker-compose.yml` (the one in the repository is the same):

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

```bash
docker compose up -d
```

Or without Compose:

```bash
docker run -d --name jackstream \
  -p 7000:7000 -p 7001:7001 \
  --restart unless-stopped \
  ghcr.io/lborruto/jackstream:latest
```

Then:

1. **Open the configure page** at `http://<server-ip>:7000/configure` from any device on your
   LAN.
2. **Fill in** the Jackett URL (`http://<server-ip>:9117`, say) and its API key, the TMDB API
   key, and under **Addon public URL / LAN IP** your server's LAN IP, `192.168.1.10` for
   example. The page turns it into the HTTPS URL that Stremio clients on other devices can reach
   with a valid certificate: `https://192-168-1-10.local-ip.medicmobile.org:7001`.
3. **Test Jackett** and **Test TMDB** ping both from the server, so you know the settings work
   before you install.
4. **Advanced filters** (optional): expand the section to set a language preference, quality
   bounds, size limits or blacklisted keywords. See [Advanced filters](#advanced-filters).
5. **Install in Stremio.** The native Stremio app opens and installs the addon. The HTTPS URL is
   shown next to the button too, to paste into Stremio by hand on a device without a browser,
   a TV for instance.
6. **Pick a movie or an episode.** Jackett's torrents appear as sources.

No domain, no certificate to manage, no per-device trust setup.

Reloading `/configure` fills the form again from the browser's localStorage. To carry the full
config to another browser, bookmark `http://<server-ip>:7000/<base64>/configure`.

### From source

```bash
git clone https://github.com/lborruto/jackstream.git
cd jackstream
go build -o jackstream ./cmd/jackstream
./jackstream
```

Needs Go 1.24 or newer. Every dependency is in the standard library except
`github.com/anacrolix/torrent`, which `go build` fetches.

## How it works

```
Stremio app
    │ GET /{config}/manifest.json
    │ GET /{config}/stream/{type}/{id}.json
    ▼
[Go net/http + Stremio addon]
    │
    ├─ Resolve IMDB id → titles via TMDB (24 h cache)
    ├─ Search Jackett in parallel across title variants
    ├─ Parse + sort torrents (quality > source > hdr > seeders)
    └─ Return streams pointing back to /stream/{config}/{torrentId}/{fileIdx}

    │ GET /stream/:config/:torrentId/:fileIdx
    ▼
[anacrolix/torrent singleton]
    │
    ├─ Download .torrent via Jackett (with passkeys)
    ├─ Reader-driven piece prioritization (responsive + 16 MB readahead):
    │  only the parts you actually watch are downloaded
    ├─ Wait for torrent metadata, then serve via http.ServeContent (Range-aware)
    └─ Clean up idle torrents + their on-disk pieces, respect maxConcurrentTorrents
```

## Advanced filters

All optional. With the defaults, results are the same as in earlier versions.

| Setting | Effect |
|---|---|
| **Preferred audio language** | Sort boost for matching audio (FRENCH, MULTI, VOSTFR, ENG). `FRENCH` also boosts MULTI releases; so does `ENG`. Results that do not match are not hidden. |
| **Min / Max quality** | Hard filter. Drops results above or below the chosen tier. |
| **Min size (MB)** | Drops "fake" low-size releases. |
| **Max size (GB)** | Drops oversized releases (useful on a slow connection or a small disk). |
| **Blacklist keywords** | Comma-separated list, case-insensitive, matched on word boundaries. Drops results whose title contains any keyword, `CAM, HDCAM, TELESYNC` for example. |

Every filter is part of the base64-encoded addon URL, so two Stremio installs can use different
filter profiles against the same server.

## HTTPS

jackstream uses the wildcard TLS certificate from <https://local-ip.medicmobile.org>, a
community service that issues a public Let's Encrypt certificate for
`*.local-ip.medicmobile.org` and runs a DNS server where `<dashed-ip>.local-ip.medicmobile.org`
resolves to `<ip>`. Any LAN IP gets a valid HTTPS name with nothing to configure.

The certificate is **refreshed while the addon runs**: on startup and every 12 hours it fetches
the current certificate and key from the service and swaps them into the running HTTPS
listener. No restart and no image rebuild when Let's Encrypt rotates it, about every 60 days. A
copy is embedded in the binary as a fallback, so HTTPS still comes up when the service cannot be
reached for a while.

The HTTPS listener is on port **7001**, next to the HTTP listener on **7000**. Stremio clients
use the HTTPS URL; the TCP connection still goes straight to your LAN host.

### Tradeoffs

- **The private key is public.** That is how the service works: anyone can download it. Anyone
  who also controls DNS on your network could sit in the middle of the connection. On a trusted
  home LAN that is not a concern. It is not suitable for an addon exposed to the public
  internet.
- **The service is community-run** (Medic Mobile). If it goes offline, the last certificate the
  addon fetched keeps serving until it expires (a warning is logged from 14 days before). After
  that, switch to your own certificate, below.

### Your own certificate

Mount a certificate and key and point the addon at them. This pins your certificate:
auto-refresh is off and renewal is yours to handle.

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

Works with certificates from Caddy, mkcert, Let's Encrypt or anything else.

### HTTP only

Set `HTTPS_DISABLED=1` to skip the HTTPS listener. Useful when Caddy, Nginx or Traefik already
terminates TLS in front of the addon.

### Refreshing the embedded fallback certificate

Rotation happens at runtime, so running installs do not need this. The embedded copy only
matters for the first seconds after a cold start, while the current certificate is being
fetched. Refreshing it now and then before a release keeps that window covered:

```bash
./scripts/refresh-certs.sh
git commit -am "chore: refresh embedded local-ip fallback cert"
```

## Configuration reference

Everything is an environment variable, and every one has a default.

| Variable | Default | What it does |
|---|---|---|
| `PORT` | `7000` | Port of the HTTP listener. |
| `DATA_DIR` | `/tmp/jackstream` | Where in-flight torrent pieces are stored. Cleaned up automatically. |
| `CACHE_TTL_MINUTES` | `1440` | How long TMDB results are cached. |
| `REQUEST_TIMEOUT_MS` | `8000` | Timeout for Jackett and TMDB requests. |
| `STREAM_READY_TIMEOUT_S` | `60` | Longest wait for the first pieces. |
| `TORRENT_IDLE_TIMEOUT_MIN` | `30` | Idle minutes before a torrent is stopped and its pieces deleted. |
| `HTTPS_PORT` | `7001` | Port of the HTTPS listener. |
| `HTTPS_CERT_PATH` | unset | Path to your own TLS certificate (PEM). Turns off certificate auto-refresh. |
| `HTTPS_KEY_PATH` | unset | Path to your own TLS private key (PEM). Both paths must be set. |
| `HTTPS_DISABLED` | unset | `1` disables the HTTPS listener. |

## FAQ

**Does it work with private trackers?**
Yes. Jackett proxies the `.torrent` URL, so the announce URLs carry your passkeys. DHT and LSD
are disabled in the BitTorrent client, so infohashes are not leaked to the public swarm.

**I saw "Stream session expired".**
The in-memory torrent map keeps entries for 2 hours. Go back to the source list and click the
stream again; it starts again at once.

**Does it seed after I finish watching?**
No. V1 stops idle torrents after `TORRENT_IDLE_TIMEOUT_MIN` (30 minutes by default) and frees
the disk. Seeding is planned for V2.

**Why no debrid?**
Out of scope: the project targets streaming inside a homelab, with no third-party service.

**Why an embedded BitTorrent client rather than qBittorrent?**
No shared volume, no second container, HTTP Range requests served natively, and the `.torrent`
proxied by Jackett carries the passkey, so private trackers work with no extra setup.

**Stremio desktop or TV?**
Both. `notWebReady: true` tells Stremio to use its native player rather than the web player.

**The *Install in Stremio* button did nothing, or the URL lost the `:7001` port.**
Some Stremio clients (older Windows builds in particular) mishandle the port in `stremio://`
deep links. Copy the `https://…:7001/<base64>/manifest.json` URL shown under the button, then
in Stremio click the puzzle-piece icon and paste it into *Add-on Repository URL*.

**Loading failed when I clicked a stream.**
Most often the container restarted (a redeploy, say) and the in-memory `torrentStore` was wiped
while Stremio still held a cached list of stream URLs. Close and reopen the movie in Stremio to
fetch a fresh list.

## Updating

```bash
docker compose pull && docker compose up -d
```

What changed in each version is in [CHANGELOG.md](CHANGELOG.md). No release so far has changed
the addon URL format: URLs already installed in Stremio keep working.

## Contributing

Pull requests are welcome. Run `go test ./...` and `go vet ./...` before opening one.

## Licence

MIT, see [LICENSE](LICENSE).
