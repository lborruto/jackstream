import path from 'node:path'
import WebTorrent from 'webtorrent'
import * as torrentStore from './torrentStore.js'

const STREAM_READY_MB = parseInt(process.env.STREAM_READY_MB || '5', 10)
const STREAM_READY_TIMEOUT_S = parseInt(process.env.STREAM_READY_TIMEOUT_S || '60', 10)
const TORRENT_IDLE_TIMEOUT_MIN = parseInt(process.env.TORRENT_IDLE_TIMEOUT_MIN || '30', 10)
const REQUEST_TIMEOUT_MS = parseInt(process.env.REQUEST_TIMEOUT_MS || '8000', 10)

const CHUNK_MAX = 10 * 1024 * 1024

const EXT_MIME = {
  '.mkv': 'video/x-matroska',
  '.mp4': 'video/mp4',
  '.m4v': 'video/mp4',
  '.mov': 'video/quicktime',
  '.avi': 'video/x-msvideo',
  '.webm': 'video/webm',
  '.ts': 'video/mp2t',
}

const VIDEO_EXTS = Object.keys(EXT_MIME)

let client = null
// torrentId → { torrent, lastAccess, readers }
const active = new Map()

function getClient() {
  if (!client) {
    client = new WebTorrent({ dht: false, lsd: false })
    client.on('error', err => console.error('[webtorrent] client error:', err.message))
  }
  return client
}

function pickVideoFile(torrent) {
  const videos = torrent.files.filter(f => VIDEO_EXTS.includes(path.extname(f.name).toLowerCase()))
  const pool = videos.length ? videos : torrent.files
  return pool.slice().sort((a, b) => b.length - a.length)[0]
}

function mimeFor(name) {
  return EXT_MIME[path.extname(name).toLowerCase()] || 'video/x-matroska'
}

async function fetchTorrentBuffer(torrentUrl) {
  const res = await fetch(torrentUrl, { signal: AbortSignal.timeout(REQUEST_TIMEOUT_MS) })
  if (!res.ok) throw new Error(`.torrent fetch ${res.status}`)
  return Buffer.from(await res.arrayBuffer())
}

async function addNewTorrent({ torrentUrl, magnetUri }) {
  const wt = getClient()
  let source = null
  try {
    if (torrentUrl) source = await fetchTorrentBuffer(torrentUrl)
  } catch (err) {
    console.warn(`[webtorrent] .torrent fetch failed: ${err.message}, trying magnet`)
  }
  if (!source && magnetUri) source = magnetUri
  if (!source) throw new Error('torrent_fetch_failed')

  return new Promise((resolve, reject) => {
    const t = wt.add(source, { path: '/tmp/webtorrent' }, torrent => resolve(torrent))
    t.once('error', reject)
  })
}

function existingByInfoHash(infoHash) {
  const wt = getClient()
  if (!infoHash) return null
  return wt.torrents.find(t => t.infoHash === infoHash) || null
}

export async function addTorrent(torrentId) {
  const existing = active.get(torrentId)
  if (existing) {
    existing.lastAccess = Date.now()
    return existing.torrent
  }
  const meta = torrentStore.get(torrentId)
  if (!meta) throw new Error('torrent_not_found')

  const byHash = existingByInfoHash(meta.infoHash)
  const torrent = byHash || (await addNewTorrent(meta))

  active.set(torrentId, { torrent, lastAccess: Date.now(), readers: 0 })
  return torrent
}

function waitForBytes(file, minBytes, timeoutMs) {
  const deadline = Date.now() + timeoutMs
  return new Promise((resolve, reject) => {
    const tick = () => {
      if (file.downloaded >= minBytes) return resolve()
      if (Date.now() >= deadline) return reject(new Error('stream_timeout'))
      setTimeout(tick, 500)
    }
    tick()
  })
}

function applySequentialPriority(torrent, fileIdx) {
  torrent.files.forEach(f => f.deselect())
  const file = torrent.files[fileIdx]
  file.select()
  torrent.critical(0, 5)
  return file
}

export async function streamFile(torrentId, _fileIdx, req, res) {
  let torrent
  try {
    torrent = await addTorrent(torrentId)
  } catch (err) {
    if (err.message === 'torrent_not_found') {
      return res.status(404).json({
        error: 'torrent_not_found',
        message: 'Stream session expired. Please go back and select the stream again.',
      })
    }
    if (err.message === 'torrent_fetch_failed') {
      return res.status(502).json({ error: 'torrent_fetch_failed' })
    }
    throw err
  }

  const readyIfNeeded = torrent.ready
    ? Promise.resolve()
    : new Promise(resolve => torrent.once('ready', resolve))
  await readyIfNeeded

  const target = pickVideoFile(torrent)
  const fileIdx = torrent.files.indexOf(target)
  const file = applySequentialPriority(torrent, fileIdx)

  try {
    await waitForBytes(file, STREAM_READY_MB * 1024 * 1024, STREAM_READY_TIMEOUT_S * 1000)
  } catch {
    return res.status(503).json({ error: 'stream_timeout' })
  }

  const entry = active.get(torrentId)
  if (entry) {
    entry.readers += 1
    entry.lastAccess = Date.now()
  }
  const releaseReader = () => {
    if (!entry) return
    entry.readers = Math.max(0, entry.readers - 1)
    entry.lastAccess = Date.now()
  }
  req.on('close', releaseReader)

  const total = file.length
  const contentType = mimeFor(file.name)
  const range = req.headers.range

  if (!range) {
    res.writeHead(200, {
      'Content-Length': total,
      'Content-Type': contentType,
      'Accept-Ranges': 'bytes',
    })
    file.createReadStream().pipe(res)
    return
  }

  const match = range.match(/^bytes=(\d*)-(\d*)$/)
  if (!match) {
    res.setHeader('Content-Range', `bytes */${total}`)
    return res.status(416).json({ error: 'bad_range' })
  }

  const start = parseInt(match[1] || '0', 10)
  const endPart = match[2]
  const end = endPart
    ? Math.min(parseInt(endPart, 10), total - 1)
    : Math.min(start + CHUNK_MAX, total - 1)

  if (
    !Number.isFinite(start) ||
    !Number.isFinite(end) ||
    start > end ||
    start < 0 ||
    end >= total
  ) {
    res.setHeader('Content-Range', `bytes */${total}`)
    return res.status(416).json({ error: 'bad_range' })
  }

  res.writeHead(206, {
    'Content-Range': `bytes ${start}-${end}/${total}`,
    'Accept-Ranges': 'bytes',
    'Content-Length': end - start + 1,
    'Content-Type': contentType,
  })
  file.createReadStream({ start, end }).pipe(res)
}

export function cleanup(maxConcurrent = 3) {
  const now = Date.now()
  const idleMs = TORRENT_IDLE_TIMEOUT_MIN * 60 * 1000

  for (const [id, entry] of active) {
    if (entry.readers === 0 && now - entry.lastAccess > idleMs) {
      destroy(id)
    }
  }

  if (active.size > maxConcurrent) {
    const sorted = [...active.entries()]
      .filter(([, e]) => e.readers === 0)
      .sort((a, b) => a[1].lastAccess - b[1].lastAccess)
    while (active.size > maxConcurrent && sorted.length) {
      const [id] = sorted.shift()
      destroy(id)
    }
  }
}

function destroy(torrentId) {
  const entry = active.get(torrentId)
  if (!entry) return
  active.delete(torrentId)
  try {
    entry.torrent.destroy({ destroyStore: true })
  } catch (err) {
    console.warn(`[webtorrent] destroy failed: ${err.message}`)
  }
}

export function shutdown() {
  for (const id of [...active.keys()]) destroy(id)
  if (client) {
    try {
      client.destroy()
    } catch {
      // ignore
    }
    client = null
  }
}

export function _activeCount() {
  return active.size
}
