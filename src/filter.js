const QUALITY_ORDER = ['CAM', null, '480p', '720p', '1080p', '4K']
const QUALITY_RANK = new Map()
QUALITY_ORDER.forEach((q, i) => QUALITY_RANK.set(q, i))

function rankOf(quality) {
  return QUALITY_RANK.has(quality) ? QUALITY_RANK.get(quality) : 1 // treat unknown like null
}

function buildBlacklistRegex(keywords) {
  if (!keywords || !keywords.length) return null
  const escaped = keywords
    .map(k => k.trim())
    .filter(Boolean)
    .map(k => k.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))
  if (!escaped.length) return null
  return new RegExp(`\\b(?:${escaped.join('|')})\\b`, 'i')
}

export function filterTorrents(torrents, config) {
  const maxQ = config.maxQuality
  const minQ = config.minQuality
  const maxBytes = config.maxSizeGb ? config.maxSizeGb * 1024 ** 3 : null
  const minBytes = config.minSizeMb ? config.minSizeMb * 1024 ** 2 : null
  const blRe = buildBlacklistRegex(config.blacklistKeywords)

  return torrents.filter(t => {
    const q = t.parsedTitle ? t.parsedTitle.quality : null
    if (maxQ && rankOf(q) > rankOf(maxQ)) return false
    if (minQ && rankOf(q) < rankOf(minQ)) return false
    if (maxBytes && Number.isFinite(t.size) && t.size > maxBytes) return false
    if (minBytes && Number.isFinite(t.size) && t.size < minBytes) return false
    if (blRe && blRe.test(t.title || '')) return false
    return true
  })
}
