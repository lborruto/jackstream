# Security policy

## Reporting a vulnerability

Please report privately through GitHub: **Security › Report a vulnerability**
(https://github.com/lborruto/jackstream/security/advisories/new). Do not open a public issue.
Include the version (the image tag, the release tag or the commit), what an attacker needs
(access to your LAN, an addon URL, a torrent or a Jackett result they control) and the steps.
Never include your Jackett or TMDB API keys, your addon URL (it holds both), tracker passkeys or
`.torrent` files from a private tracker: they are not needed. jackstream is maintained in spare
time: reports are handled on a best-effort basis, with no guaranteed response time.

## Supported versions

The latest release only: the most recent `vX.Y.Z` tag and its image,
`ghcr.io/lborruto/jackstream:X.Y.Z` (also `latest`). Fixes land on `main` and in the next
release; older tags are not patched. Each image published by the release workflow carries
build provenance and an SBOM as attestations:
`docker buildx imagetools inspect ghcr.io/lborruto/jackstream:X.Y.Z --format '{{ json .Provenance }}'`.

## Security model

Read this before installing. jackstream is a Stremio addon for a home network. It is not
affiliated with, endorsed by or supported by Stremio, Jackett, TMDB or Medic Mobile.

- **No login: whoever reaches the ports can use it.** The addon listens on port 7000 (HTTP) and
  7001 (HTTPS) with no authentication. Anyone who can reach those ports can open the configure
  page, ask the server to test a Jackett URL or a TMDB key, and, with a valid addon URL, search
  and stream through your Jackett and your connection. Keep it on a trusted LAN; do not publish
  the ports to the internet.
- **The addon URL is the credential.** Your Jackett URL and API key, your TMDB API key and your
  filters are encoded (base64, not encrypted) in the addon URL that Stremio installs. The server
  writes none of them to disk: no database, no settings file. Anyone who sees that URL can read
  the keys back. Stremio keeps installed addon URLs in your Stremio account, and the configure
  page keeps your inputs in the browser's localStorage. Known gap: when a request to Jackett
  fails (Jackett down, a timeout), the error line in the container's log includes the request
  URL, and with it the Jackett API key; treat `docker logs jackstream` as holding it.
- **What it talks to.** Your Jackett (searches and `.torrent` downloads through Jackett's proxied
  links, so private-tracker passkeys stay between Jackett and the addon), TMDB's API
  (`api.themoviedb.org`, title resolution), `local-ip.medicmobile.org` (the HTTPS certificate,
  at startup and every 12 hours, unless you mount your own) and the BitTorrent peers of the
  torrents you play. No telemetry, no update check.
- **BitTorrent from your IP address.** Streaming a torrent downloads it from its peers with your
  public IP address, like any torrent client. DHT and µTP are off, seeding is off, and idle
  torrents and their pieces are deleted after 30 minutes (`TORRENT_IDLE_TIMEOUT_MIN`). Pieces
  are written under `DATA_DIR` (`/tmp/jackstream` in the container).
- **The default HTTPS key is public.** The default certificate is the community
  `*.local-ip.medicmobile.org` wildcard, whose private key the service publishes for anyone to
  download (README, "HTTPS"). It gives Stremio clients a valid certificate on a LAN; it does not
  protect against someone who controls DNS on your network. Mount your own certificate
  (`HTTPS_CERT_PATH`, `HTTPS_KEY_PATH`) or set `HTTPS_DISABLED=1` behind your own proxy if that
  matters to you.
- **Small image.** A static Go binary on a `scratch` image: no shell, no package manager. CI scans
  the image with Trivy on every change and again when a release is published.

### What it does not protect against

- Anyone on your network, or on the internet if you publish the ports: there is no login.
- Someone who sees your addon URL (a screenshot, a shared Stremio account, a browser history)
  or the container's log: they hold your keys.
- Someone who controls DNS on your LAN, while you use the default `local-ip.medicmobile.org`
  certificate.
- What you stream: the legality of a torrent, and what your trackers and your internet provider
  see, are your responsibility.
- A malicious Jackett result or `.torrent`: the addon downloads what Jackett returns.

## In scope

The addon's code, the configure page, the image and its Dockerfile, the HTTPS certificate
handling, the GitHub workflows and the published images.

## Out of scope

- Stremio, Jackett, TMDB and the trackers themselves: report to them.
- `local-ip.medicmobile.org` and its published key: that is how the service works.
- Exposing the ports to the internet, or running it on an untrusted network.
- An image or fork distributed from somewhere else than this repository's releases.
