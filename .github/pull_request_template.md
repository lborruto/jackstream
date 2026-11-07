## What and why

## How it was checked

- [ ] `gofmt -l .` prints nothing
- [ ] `go vet ./...`
- [ ] `go test -race ./...`
- [ ] `go generate ./...` leaves no diff (configure page, fallback keypair)
- [ ] `docker build .` and the addon starts (`/health`, `/configure`)
- [ ] Changed searching or streaming: tried with my own Jackett and Stremio

## Checklist

- [ ] The addon URL format still decodes: URLs already installed in Stremio keep working
- [ ] Nothing stores or logs an API key, a passkey or an addon URL
- [ ] Still one container, no volume required, no new service it depends on
- [ ] No new dependency (or it was discussed in an issue first)
- [ ] No API key, passkey, addon URL, `.torrent` or real host name in code, tests, docs or
      screenshots; a fix comes with its unit test
- [ ] `CHANGELOG.md` has a line under `## [Unreleased]` if users will notice the change
