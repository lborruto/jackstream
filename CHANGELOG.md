# Changelog

All notable changes to jackstream are listed here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0] - 2026-08-14

Self-refreshing HTTPS and leaner downloads. No breaking changes: existing addon install URLs
keep working unchanged.

### Added

- `DATA_DIR` environment variable to choose where in-flight pieces are stored (default
  `/tmp/jackstream`).
- The detected codec (x264, x265, AV1) is shown in the stream description.

### Changed

- The TLS certificate refreshes at runtime. The addon fetches the current
  `local-ip.medicmobile.org` certificate on startup and every 12 hours and swaps it into the
  HTTPS listener, so the image no longer needs a rebuild every 60 days or so. The embedded
  certificate is now only a cold-start fallback; a warning is logged if refresh keeps failing
  within 14 days of expiry. `HTTPS_CERT_PATH` and `HTTPS_KEY_PATH` still pin your own
  certificate (auto-refresh off).
- Streams no longer download the whole file in the background. Piece prioritisation follows
  the playback position (plus 16 MB of readahead), so skipping through or abandoning a stream
  no longer pulls the full release to disk. Less bandwidth, less disk, kinder to
  private-tracker ratios.
- Internal: dead code removed.

### Fixed

- A data race on the max-concurrent-torrents setting.
- Expired stream-session entries are reclaimed even when they are never read again.
- Torrent-file fetches are bounded in time and size.
- Deleting pieces from disk no longer blocks stream requests.

## [1.1.2] - 2026-08-13

### Fixed

- Refreshed the bundled `*.local-ip.medicmobile.org` certificate, which had expired on
  2026-07-02 and caused TLS handshake failures on the HTTPS listener. The new certificate is
  valid until 2026-10-30.

## [1.1.1] - 2026-04-23

Disk cleanup. No user-facing behaviour change; existing installs keep working, upgrade at any
time.

### Fixed

- `bt.Cleanup()` and `Shutdown()` now delete downloaded pieces from disk, not only from memory.
  Before, the container's writable layer (`/tmp/jackstream`) grew without bound until
  `docker rm`. Long-running containers now reclaim disk as torrents go idle and at shutdown.
- Internal: a `dropAndDelete()` helper replaces direct `torrent.Drop()` calls, and `Shutdown()`
  also wipes `DataDir` to catch orphan files left by failed fetches.

## [1.1.0] - 2026-04-23

Full rewrite from Node.js to Go with strict behavioural parity: same URLs, same config format,
same UX. No user-facing changes; existing addon install URLs keep working unchanged.

### Changed

- Docker image 93% smaller: about 20 MB on a `scratch` base, down from about 280 MB.
- Startup under 50 ms, from about 250 ms with Node.
- About 30 MB RSS, from 100 to 150 MB.
- HTTP Range streaming through `http.ServeContent` replaces about 245 lines of manual Range
  handling.

### Fixed

- No more `utp-native` segfaults: anacrolix/torrent is pure Go, and µTP is off by default.

## [1.0.0] - 2026-04-22

Initial release.

### Added

- Node.js 20, stremio-addon-sdk and webtorrent.
- HTTP and HTTPS listeners (ports 7000 and 7001) with the bundled `local-ip.medicmobile.org`
  certificate.
- Configure page with localStorage persistence and advanced filters.

[1.2.0]: https://github.com/lborruto/jackstream/compare/v1.1.2...v1.2.0
[1.1.2]: https://github.com/lborruto/jackstream/compare/v1.1.1...v1.1.2
[1.1.1]: https://github.com/lborruto/jackstream/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/lborruto/jackstream/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/lborruto/jackstream/releases/tag/v1.0.0
