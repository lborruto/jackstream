---
name: Bug report
about: Something fails or does not do what the README says
labels: bug
---

<!--
Never paste your addon URL (the long base64 part holds your Jackett and TMDB API keys), an API
key, a tracker passkey, a .torrent file or a magnet link from a private tracker. None of it is
needed here. If a log line holds any of it, replace it with "[removed]". A security problem goes
to a private report instead (see SECURITY.md), not to this issue.
-->

**What happened**

**What you expected**

**Where** (the configure page, the list of sources in Stremio, playback…) **and the title** (a
movie or an episode: its name or IMDB id is fine)

**The addon's log** (`docker logs jackstream`, the lines around the problem; no API key, passkey
or addon URL)

**Version** (the image tag, `1.2.0`…, or the commit: `git rev-parse --short HEAD`)

**System**

- Host and how it runs (Docker, Compose, from source):
- Architecture (amd64, arm64):
- Jackett version:
- Stremio app and device (desktop, Android, TV…):
- HTTP or HTTPS, default certificate or your own:

**Anything else** (does it happen every time, for one tracker only, after a while…)
