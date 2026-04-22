import crypto from 'node:crypto'
import { parseTorrentTitle } from './parser.js'

const REQUEST_TIMEOUT_MS = parseInt(process.env.REQUEST_TIMEOUT_MS || '8000', 10)

const CATEGORIES = {
  movie: ['2000'],
  series: ['5000', '5070'], // 5070 covers anime — always included for series
}

function buildQuery(title, meta, type) {
  if (type === 'series' && meta.season != null) {
    const s = String(meta.season).padStart(2, '0')
    const e = String(meta.episode).padStart(2, '0')
    return `${title} S${s}E${e}`
  }
  if (type === 'movie' && meta.year) {
    return `${title} ${meta.year}`
  }
  return title
}

function extractInfoHash(magnetUri) {
  if (!magnetUri) return null
  const m = magnetUri.match(/xt=urn:btih:([a-zA-Z0-9]+)/)
  return m ? m[1].toLowerCase() : null
}

function torrentIdOf(torrentUrl) {
  return crypto
    .createHash('sha1')
    .update(torrentUrl || '')
    .digest('hex')
    .slice(0, 12)
}

async function searchOneVariant(title, meta, type, config) {
  const query = buildQuery(title, meta, type)
  const cats = CATEGORIES[type] || []
  const params = new URLSearchParams({
    apikey: config.jackettApiKey,
    Query: query,
  })
  for (const c of cats) params.append('Category[]', c)
  const url = `${config.jackettUrl.replace(/\/$/, '')}/api/v2.0/indexers/all/results?${params}`

  try {
    const res = await fetch(url, { signal: AbortSignal.timeout(REQUEST_TIMEOUT_MS) })
    if (!res.ok) {
      console.warn(`[jackett] variant "${title}" → HTTP ${res.status}`)
      return []
    }
    const data = await res.json()
    return data.Results || []
  } catch (err) {
    console.warn(`[jackett] variant "${title}" failed: ${err.message}`)
    return []
  }
}

export async function searchJackett(titleVariants, meta, type, config) {
  const results = await Promise.all(titleVariants.map(t => searchOneVariant(t, meta, type, config)))
  const flat = results.flat()

  const byKey = new Map()
  for (const r of flat) {
    const torrentUrl = r.Link || null
    const magnetUri = r.MagnetUri || null
    const infoHash = extractInfoHash(magnetUri)
    const dedupKey = infoHash || torrentUrl
    if (!dedupKey) continue

    const seeders = Number.isFinite(r.Seeders) ? r.Seeders : 0
    if (seeders < (config.minSeeders ?? 1)) continue
    if (!torrentUrl && !magnetUri) continue

    if (!byKey.has(dedupKey)) {
      byKey.set(dedupKey, {
        title: r.Title,
        torrentUrl,
        magnetUri,
        size: r.Size,
        seeders,
        infoHash,
        parsedTitle: parseTorrentTitle(r.Title || ''),
        torrentId: torrentIdOf(torrentUrl || magnetUri),
      })
    }
  }
  return [...byKey.values()]
}

export async function pingJackett(jackettUrl, apiKey) {
  const url = `${jackettUrl.replace(/\/$/, '')}/api/v2.0/indexers/all/results?apikey=${apiKey}&Query=test`
  const res = await fetch(url, { signal: AbortSignal.timeout(REQUEST_TIMEOUT_MS) })
  return res.ok
}
