import { createRequire } from 'node:module'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import express from 'express'
import cors from 'cors'

import { encodeConfig, decodeConfig, validateConfig } from './config.js'
import { resolveImdbId, titleVariants, parseStremioId } from './tmdb.js'
import { searchJackett, pingJackett } from './jackett.js'
import { sortTorrents } from './sorter.js'
import * as torrentStore from './torrentStore.js'
import * as webtorrent from './webtorrent.js'

const require = createRequire(import.meta.url)
const { addonBuilder } = require('stremio-addon-sdk')

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const PORT = parseInt(process.env.PORT || '7000', 10)

const manifest = {
  id: 'community.jackstream',
  version: '1.0.0',
  name: 'jackstream',
  description:
    'Stream torrents from your Jackett instance directly in Stremio. Self-hosted, no debrid required.',
  resources: ['stream'],
  types: ['movie', 'series'],
  idPrefixes: ['tt'],
  behaviorHints: { configurable: true, configurationRequired: true },
  catalogs: [],
}

// Validate via SDK (throws on invalid schema), then discard the interface.
// The SDK requires at least a stream handler to be defined before getInterface().
const _validationBuilder = new addonBuilder(manifest)
_validationBuilder.defineStreamHandler(async () => ({ streams: [] }))
_validationBuilder.getInterface()

function configuredManifest() {
  return { ...manifest, behaviorHints: { configurable: true, configurationRequired: false } }
}

function redact(obj) {
  const copy = { ...obj }
  if (copy.jackettApiKey) copy.jackettApiKey = '***'
  if (copy.tmdbApiKey) copy.tmdbApiKey = '***'
  return copy
}

function loadConfig(configParam) {
  const decoded = decodeConfig(configParam)
  return validateConfig(decoded)
}

function formatSize(bytes) {
  if (!Number.isFinite(bytes)) return '?'
  const gb = bytes / 1024 / 1024 / 1024
  if (gb >= 1) return `${gb.toFixed(1)} GB`
  const mb = bytes / 1024 / 1024
  return `${mb.toFixed(0)} MB`
}

function streamName(parsed) {
  const bits = ['🎬 Jackett']
  const q = parsed.quality
  const h = parsed.hdr
  bits.push([q, h].filter(Boolean).join(' ') || '—')
  return bits.join('\n')
}

function streamDescription(t) {
  const seeds = `👥 ${t.seeders} seeds`
  const size = `💾 ${formatSize(t.size)}`
  const src = t.parsedTitle.source ? `🔵 ${t.parsedTitle.source}` : ''
  const aud = t.parsedTitle.audio ? ` • ${t.parsedTitle.audio}` : ''
  return `${t.title}\n${seeds} • ${size}${src ? ' • ' + src : ''}${aud}`
}

export function buildApp() {
  const app = express()
  app.use(cors())
  app.use(express.json())

  app.get('/health', (_req, res) => res.json({ ok: true }))

  app.get('/manifest.json', (_req, res) => res.json(manifest))

  app.get('/:config/manifest.json', (req, res) => {
    try {
      loadConfig(req.params.config)
      res.json(configuredManifest())
    } catch (err) {
      res.status(400).json({ error: 'invalid_config', detail: err.message })
    }
  })

  app.get('/:config/stream/:type/:id.json', async (req, res) => {
    let config
    try {
      config = loadConfig(req.params.config)
    } catch (err) {
      return res.status(400).json({ error: 'invalid_config', detail: err.message })
    }

    const type = req.params.type
    const rawId = req.params.id.replace(/\.json$/, '')

    try {
      const meta = await resolveImdbId(rawId, config)
      const variants = titleVariants(meta)
      const results = await searchJackett(variants, meta, type, config)
      const sorted = sortTorrents(results).slice(0, config.maxResults)

      const streams = sorted.map(t => {
        torrentStore.set(t.torrentId, {
          torrentUrl: t.torrentUrl,
          magnetUri: t.magnetUri,
          parsedTitle: t.parsedTitle,
          size: t.size,
          seeders: t.seeders,
          infoHash: t.infoHash,
        })
        const base = config.addonPublicUrl.replace(/\/$/, '')
        return {
          name: streamName(t.parsedTitle),
          description: streamDescription(t),
          url: `${base}/stream/${req.params.config}/${t.torrentId}/0`,
          behaviorHints: { notWebReady: true, bingeGroup: 'jackstream' },
        }
      })

      res.json({ streams })
    } catch (err) {
      console.error('[stream handler]', err.message, redact(parseStremioIdSafe(rawId)))
      res.json({ streams: [] })
    }
  })

  app.get('/stream/:config/:torrentId/:fileIdx', async (req, res) => {
    try {
      loadConfig(req.params.config) // validate but do not use
    } catch (err) {
      return res.status(400).json({ error: 'invalid_config', detail: err.message })
    }
    try {
      await webtorrent.streamFile(req.params.torrentId, parseInt(req.params.fileIdx, 10), req, res)
    } catch (err) {
      console.error('[stream]', err.message)
      if (!res.headersSent) res.status(500).json({ error: 'internal' })
    }
  })

  app.get('/api/test-jackett', async (req, res) => {
    const { url, key } = req.query
    if (!url || !key) return res.status(400).json({ ok: false, error: 'missing url or key' })
    try {
      const ok = await pingJackett(url, key)
      res.json({ ok })
    } catch (err) {
      res.json({ ok: false, error: err.message })
    }
  })

  app.get('/api/test-tmdb', async (req, res) => {
    const { key } = req.query
    if (!key) return res.status(400).json({ ok: false, error: 'missing key' })
    try {
      const r = await fetch(
        `https://api.themoviedb.org/3/configuration?api_key=${encodeURIComponent(key)}`,
        { signal: AbortSignal.timeout(8000) }
      )
      res.json({ ok: r.ok })
    } catch (err) {
      res.json({ ok: false, error: err.message })
    }
  })

  app.get('/configure', (_req, res) => {
    res.sendFile(path.join(__dirname, '..', 'public', 'configure.html'))
  })

  app.get('/', (_req, res) => res.redirect('/configure'))

  return app
}

function parseStremioIdSafe(id) {
  try {
    return parseStremioId(id)
  } catch {
    return { imdbId: id }
  }
}

export { encodeConfig }

setInterval(() => {
  try {
    webtorrent.cleanup(parseInt(process.env.MAX_CONCURRENT_TORRENTS || '3', 10))
  } catch (err) {
    console.warn('[cleanup]', err.message)
  }
}, 60 * 1000).unref()

const isMain = fileURLToPath(import.meta.url) === process.argv[1]
if (isMain) {
  const app = buildApp()
  app.listen(PORT, () => {
    console.log(`jackstream listening on :${PORT}`)
  })
  for (const sig of ['SIGINT', 'SIGTERM']) {
    process.once(sig, () => {
      console.log(`received ${sig}, shutting down`)
      webtorrent.shutdown()
      process.exit(0)
    })
  }
}
