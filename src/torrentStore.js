import { createCache } from './cache.js'

const TTL_MS = 2 * 60 * 60 * 1000

const cache = createCache()

export function set(torrentId, data) {
  cache.set(torrentId, data, TTL_MS)
}

export function get(torrentId) {
  return cache.get(torrentId)
}

export function clear() {
  cache.clear()
}
