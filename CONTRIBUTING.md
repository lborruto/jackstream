# Contributing to jackstream

Thanks for helping. jackstream sits between someone's Stremio, their Jackett (with their
private-tracker passkeys behind it) and the BitTorrent swarm, so the bar is: the addon URLs
people already installed keep working, no key is stored or shown, and it stays one small
container with nothing else to run.

It is meant to stay small: Jackett results as Stremio sources, streamed by its own BitTorrent
client, nothing more. A new feature has to fit that.

Contents: [Set up](#set-up) · [Checks](#checks) · [Principles](#principles) ·
[Commits and pull requests](#commits-and-pull-requests) ·
[Never commit secrets or real data](#never-commit-secrets-or-real-data) ·
[How it works](#how-it-works) · [Releasing](#releasing).

## Set up

You need Go 1.26 or later, and Docker to build the image. The one direct dependency is
`github.com/anacrolix/torrent`; the rest is the standard library.

    git clone https://github.com/lborruto/jackstream.git && cd jackstream
    go build -o jackstream ./cmd/jackstream
    ./jackstream

It listens on 7000 (HTTP) and 7001 (HTTPS); open `http://localhost:7000/configure`. Every
setting is an environment variable with a default (README, "Configuration reference"). To try
it end to end you need your own Jackett, a TMDB API key and Stremio.

## Checks

Run these before opening a pull request:

    gofmt -l .                  # prints nothing
    go vet ./...
    go test -race ./...         # unit tests: no network, no Jackett, no torrent
    go generate ./... && git diff --exit-code   # the embedded copies match their sources
    docker build -t jackstream:dev .

CI runs the same on Linux with Go 1.26 (`GOTOOLCHAIN=local`, the version the Dockerfile builds
with), builds the image for linux/amd64 and linux/arm64, starts it and asks for `/health`, the
manifest and `/configure`, and scans it with Trivy (fixable high and critical vulnerabilities
fail). It also runs gitleaks over the whole history, actionlint and shellcheck, and CodeQL on the
Go sources. The one check to require is "CI result"; a docs-only pull request skips the build
jobs and still passes it.

When Dependabot moves the Go builder image to a new minor version, move `go-version` in
`ci.yml` and `codeql.yml` with it.

## Principles

These are permanent. A change that needs one of them relaxed needs a conversation first, in an
issue.

- **Installed addon URLs keep working.** The config is base64 JSON in the URL; `config.Decode`
  accepts every form it ever accepted. A new setting is optional with a default that keeps the
  old behaviour.
- **The server stores no key.** Credentials and filters live in the addon URL only. No database,
  no settings file; nothing new may log an API key, a passkey or an addon URL.
- **One container, nothing else to run.** No debrid service, no external torrent client, no
  required volume, a `scratch` image.
- **Private trackers stay private.** `.torrent` files go through Jackett's proxied link; DHT and
  µTP stay off and the client does not seed.
- **Clean up after yourself.** Idle torrents and their pieces on disk are removed; nothing grows
  without bound.
- **No new dependency without discussion.** A new module needs an issue first, with the reason.
- **Every fix comes with a unit test** that fails before the fix and passes after it. Unit tests
  never touch the network (an `httptest` server or an injected client instead).

## Commits and pull requests

- One change per commit, with a subject line that says what changed, in plain words, as in
  `Delete a torrent's pieces when it is dropped`. Say why in the body when it is not obvious.
- One topic per pull request, with its tests. Fill in the template's checklist.
- Keep `CHANGELOG.md` current: a change users will notice adds a line under `## [Unreleased]`.

## Never commit secrets or real data

- No Jackett or TMDB API key, addon URL (its base64 part holds both), tracker passkey,
  `.torrent` file or magnet link from a private tracker, in code, tests, docs, issues,
  screenshots or commit messages. Test fixtures are invented (`192.168.1.10`, `key`…).
- `.env` files are gitignored. CI runs gitleaks over the whole history; its one exception is the
  public `local-ip.medicmobile.org` key in `certs/` and its embedded copy (`.gitleaks.toml`).
- Found something sensitive in the history? Report it privately (see `SECURITY.md`).

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

- **`cmd/jackstream/main.go`**: reads the environment, starts the BitTorrent client, the HTTP
  listener, the HTTPS listener with its certificate manager, the idle-torrent cleanup ticker,
  and shuts everything down on a signal.
- **`internal/server/`**: `server.go` (the routes: manifest, stream list, test endpoints,
  configure page; CORS; `Version`, which the release checks against the tag), `stream.go` (file
  playback), `certmanager.go` (fetches the `local-ip.medicmobile.org` certificate at startup and
  every 12 hours and swaps it in), `certs.go` (the embedded fallback keypair).
- **`internal/config/`**: the addon URL's base64 JSON: decode, validate, defaults.
- **`internal/tmdb/`**: IMDB id to localised and original titles, episode titles, 24 h cache.
- **`internal/jackett/`**: parallel searches across the title variants, de-duplicated by
  infohash.
- **`internal/parser/`**, **`internal/filter/`**, **`internal/sorter/`**: quality, source, HDR,
  codec and audio from the release name; the hard filters; the sort order.
- **`internal/bt/`**: the anacrolix/torrent singleton: `.torrent` through Jackett (magnet as a
  fallback), playback-driven piece priority with 16 MB of readahead, `http.ServeContent`, idle
  cleanup that deletes pieces from disk.
- **`internal/store/`**, **`internal/cache/`**: the 2 h torrent map behind stream URLs and the
  generic TTL cache.
- **`public/configure.html`** is the configure page and **`certs/`** the fallback keypair; both
  are copied next to the code by `go generate` because `//go:embed` does not follow symlinks.
  Edit the originals, then run `go generate ./...`.

### Refreshing the embedded fallback certificate

Rotation happens at runtime, so running installs do not need this. The embedded copy only
covers the first seconds after a cold start. Refresh it now and then before a release:

    ./scripts/refresh-certs.sh
    go generate ./...
    git commit -am "Refresh the embedded local-ip fallback certificate"

## Releasing

1. Bump `Version` in `internal/server/server.go`.
2. In `CHANGELOG.md`, rename `## [Unreleased]` to `## [X.Y.Z] - YYYY-MM-DD`, start a new empty
   `## [Unreleased]` above it, and update the compare links at the bottom.
3. Merge to `main`, then tag: `git tag vX.Y.Z && git push origin vX.Y.Z`.
4. `release.yml` checks that the tagged commit is on `main`, runs CI on it, checks that
   `Version` says `X.Y.Z` and that the changelog has its section, builds the image for
   linux/amd64 and linux/arm64 with provenance and SBOM, pushes it to
   `ghcr.io/lborruto/jackstream` (and Docker Hub when its secrets are set) as `X.Y.Z`, `X.Y` and
   `latest`, scans the pushed digest with Trivy, and creates the GitHub release from the
   changelog section. A tag without its section, or with another `Version`, fails on purpose
   before anything is published.
