# jackstream — notes for Claude

Self-hosted Stremio addon: it turns the torrents a user's Jackett finds into Stremio sources and
streams them through its own BitTorrent client (anacrolix/torrent). Go 1.26, a static binary on
a `scratch` image, published to `ghcr.io/lborruto/jackstream` (and Docker Hub) for linux/amd64
and linux/arm64. Public repository, MIT.

The addon is stateless: every setting (Jackett URL and API key, TMDB key, filters) lives in the
base64 JSON of the addon URL Stremio installs. HTTPS on a LAN uses the public
`*.local-ip.medicmobile.org` wildcard certificate, refreshed at runtime, with an embedded copy
as a cold-start fallback. v1.0.0 was Node.js; v1.1.0 rewrote it in Go with the same URLs.

## Permanent rules

- Commit as `Luca Borruto <luca.borruto@gmail.com>` (set in the local git config), never the
  work address. Check `git log --format=%ae` before pushing.
- Never push `main` without the owner's go-ahead: branch, pull request, "CI result" green, the
  owner merges. No tag or release without the owner's go-ahead.
- Installed addon URLs keep working: `config.Decode` keeps accepting every form it ever did; a
  new setting is optional with a backward-compatible default.
- The server stores no key and logs none (no database, no settings file); nothing new may log an
  API key, a passkey or an addon URL.
- One container, no required volume, no debrid, no external torrent client.
- DHT and µTP off, no seeding, `.torrent` through Jackett's proxied link.
- Unit tests never touch the network. Never commit an API key, addon URL, passkey, `.torrent`
  or real host name.
- Edit `public/configure.html` and `certs/`, never their copies in `internal/server/`; then
  `go generate ./...` (CI fails when the copies differ).
- A PR that changes what README.md or CLAUDE.md says updates them in the same PR.

## Layout

| Path | Role |
| --- | --- |
| `cmd/jackstream/main.go` | Environment, listeners (HTTP 7000, HTTPS 7001), cleanup ticker, shutdown. |
| `internal/server/` | Routes, CORS, `Version` (checked against the release tag), playback, certificate manager, embedded copies. |
| `internal/config/` | Addon URL config: base64 JSON decode, validation, defaults. |
| `internal/tmdb/` | IMDB id → titles, episode titles, 24 h cache. |
| `internal/jackett/` | Parallel searches across title variants, infohash de-duplication. |
| `internal/parser/`, `filter/`, `sorter/` | Release-name parsing, hard filters, sort order. |
| `internal/bt/` | anacrolix/torrent singleton, readahead, `ServeContent`, idle cleanup and disk deletion. |
| `internal/store/`, `cache/` | 2 h torrent map, generic TTL cache. |
| `public/configure.html` | The configure page (source of the embedded copy). |
| `certs/` | Fallback keypair (source of the embedded copy), refreshed by `scripts/refresh-certs.sh`. |

## Build and test

    gofmt -l . && go vet ./... && go test -race ./...
    go generate ./... && git diff --exit-code
    docker build -t jackstream:dev .
    trivy image --severity HIGH,CRITICAL --ignore-unfixed jackstream:dev   # what CI fails on

CI (`ci.yml`): Changed files, actionlint + shellcheck, gitleaks (whole history), Go, Image (build
both arches, start, Trivy), "CI result". Release (`release.yml`): tag `vX.Y.Z` on main → CI →
`Version` = tag and CHANGELOG section → image to GHCR/Docker Hub → Trivy on the digest → GitHub
release. Steps in CONTRIBUTING.md, "Releasing".

## Known traps

- The fallback key in `certs/key.pem` is public on purpose (local-ip.medicmobile.org publishes
  it). gitleaks allow-lists those two paths for the private-key rule only.
- `scripts/refresh-certs.sh` updates `certs/` only: run `go generate ./...` after it.
- The two routes `/{config}/stream/...` and `/stream/{config}/...` overlap for Go's ServeMux; they
  are split by first path segment in `BuildHandler`.
- The torrent map is in memory: a restart makes Stremio's cached stream URLs fail until the user
  reopens the title.
- CI and CodeQL pin Go `1.26` by hand; Dependabot moves the builder image only.
- A failed Jackett request logs the request URL, which holds the API key (SECURITY.md, known
  gap).
- The Dockerfile runs as root on `scratch` (Trivy config check DS-0002); `/tmp/jackstream` has
  to exist and be writable before switching to a non-root user.

## Tasks

✅ done · 🔄 in progress · ⏳ to do

| # | Task | Status |
| --- | --- | --- |
| 1 | Go rewrite with URL parity (v1.1.0), disk cleanup (v1.1.1), runtime certificate refresh and reader-driven downloads (v1.2.0) | ✅ |
| 2 | Personal-project template: governance files, CI, release on tags, CodeQL, Dependabot, settings | 🔄 (PR open; "Protect main" ruleset after the first "CI result" run) |
| 3 | Stop logging the Jackett API key in request errors | ⏳ |
| 4 | Run as a non-root user in the image | ⏳ |
| 5 | Screenshots in `docs/screenshots/` (webp, demo data) for the README and the portfolio page | ⏳ |
