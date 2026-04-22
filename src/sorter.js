const QUALITY_SCORE = { '4K': 4, '1080p': 3, '720p': 2, '480p': 1, CAM: -1 }
const SOURCE_SCORE = { Remux: 5, BluRay: 4, 'WEB-DL': 3, WEBRip: 2, HDTV: 1, CAM: -1 }
const HDR_SCORE = { DV: 3, HDR10: 2, HDR: 1 }

const score = (map, key, fallback = 0) => (key == null ? fallback : map[key] ?? fallback)

function keyOf(torrent) {
  const p = torrent.parsedTitle || {}
  return [
    score(QUALITY_SCORE, p.quality),
    score(SOURCE_SCORE, p.source),
    score(HDR_SCORE, p.hdr),
    Number.isFinite(torrent.seeders) ? torrent.seeders : 0,
  ]
}

export function sortTorrents(torrents) {
  return [...torrents]
    .map((t, i) => ({ t, i, k: keyOf(t) }))
    .sort((a, b) => {
      for (let j = 0; j < a.k.length; j++) {
        if (a.k[j] !== b.k[j]) return b.k[j] - a.k[j]
      }
      return a.i - b.i
    })
    .map(({ t }) => t)
}
