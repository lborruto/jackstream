const QUALITY_SCORE = { '4K': 4, '1080p': 3, '720p': 2, '480p': 1, CAM: -1 }
const SOURCE_SCORE = { Remux: 5, BluRay: 4, 'WEB-DL': 3, WEBRip: 2, HDTV: 1, CAM: -1 }
const HDR_SCORE = { DV: 3, HDR10: 2, HDR: 1 }

const LANG_INCLUDES = {
  FRENCH: new Set(['FRENCH', 'MULTI']),
  ENG: new Set(['ENG', 'MULTI']),
  VOSTFR: new Set(['VOSTFR']),
  MULTI: new Set(['MULTI']),
}

const score = (map, key, fallback = 0) => (key == null ? fallback : (map[key] ?? fallback))

function languageBoost(audio, preferred) {
  if (!preferred) return 0
  const set = LANG_INCLUDES[preferred]
  if (set && set.has(audio)) return 1
  return 0
}

function keyOf(torrent, preferredLanguage) {
  const p = torrent.parsedTitle || {}
  return [
    score(QUALITY_SCORE, p.quality),
    score(SOURCE_SCORE, p.source),
    score(HDR_SCORE, p.hdr),
    languageBoost(p.audio, preferredLanguage),
    Number.isFinite(torrent.seeders) ? torrent.seeders : 0,
  ]
}

export function sortTorrents(torrents, config = {}) {
  const preferredLanguage = config.preferredLanguage || null
  return [...torrents]
    .map((t, i) => ({ t, i, k: keyOf(t, preferredLanguage) }))
    .sort((a, b) => {
      for (let j = 0; j < a.k.length; j++) {
        if (a.k[j] !== b.k[j]) return b.k[j] - a.k[j]
      }
      return a.i - b.i
    })
    .map(({ t }) => t)
}
