const REQUIRED = ['jackettUrl', 'jackettApiKey', 'tmdbApiKey', 'addonPublicUrl']

const VALID_LANGS = new Set(['FRENCH', 'MULTI', 'VOSTFR', 'ENG'])
const VALID_QUALITIES = new Set(['4K', '1080p', '720p', '480p'])

const toBase64Url = buf =>
  buf.toString('base64').replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')

const fromBase64Url = str => {
  const pad = str.length % 4 === 0 ? '' : '='.repeat(4 - (str.length % 4))
  const normalized = str.replace(/-/g, '+').replace(/_/g, '/') + pad
  return Buffer.from(normalized, 'base64')
}

export function encodeConfig(obj) {
  return toBase64Url(Buffer.from(JSON.stringify(obj), 'utf8'))
}

export function decodeConfig(str) {
  const buf = fromBase64Url(str)
  return JSON.parse(buf.toString('utf8'))
}

export function validateConfig(obj) {
  if (!obj || typeof obj !== 'object') {
    throw new Error('config must be an object')
  }
  for (const field of REQUIRED) {
    if (!obj[field] || typeof obj[field] !== 'string') {
      throw new Error(`config.${field} is required`)
    }
  }
  for (const urlField of ['jackettUrl', 'addonPublicUrl']) {
    try {
      // eslint-disable-next-line no-new
      new URL(obj[urlField])
    } catch {
      throw new Error(`config.${urlField} is not a valid URL`)
    }
  }
  return {
    ...obj,
    maxResults: Number.isFinite(obj.maxResults) ? obj.maxResults : 10,
    minSeeders: Number.isFinite(obj.minSeeders) ? obj.minSeeders : 1,
    maxConcurrentTorrents: Number.isFinite(obj.maxConcurrentTorrents)
      ? obj.maxConcurrentTorrents
      : 3,
    preferredLanguage: VALID_LANGS.has(obj.preferredLanguage) ? obj.preferredLanguage : null,
    maxQuality: VALID_QUALITIES.has(obj.maxQuality) ? obj.maxQuality : null,
    minQuality: VALID_QUALITIES.has(obj.minQuality) ? obj.minQuality : null,
    maxSizeGb: Number.isFinite(obj.maxSizeGb) && obj.maxSizeGb > 0 ? obj.maxSizeGb : null,
    minSizeMb: Number.isFinite(obj.minSizeMb) && obj.minSizeMb > 0 ? obj.minSizeMb : null,
    blacklistKeywords: Array.isArray(obj.blacklistKeywords)
      ? obj.blacklistKeywords.filter(s => typeof s === 'string' && s.trim()).map(s => s.trim())
      : [],
  }
}
