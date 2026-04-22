import { createCache } from './cache.js'

const TMDB_BASE = 'https://api.themoviedb.org/3'
const DEFAULT_TTL_MS = 24 * 60 * 60 * 1000
const REQUEST_TIMEOUT_MS = parseInt(process.env.REQUEST_TIMEOUT_MS || '8000', 10)

const cache = createCache()

export function parseStremioId(rawId) {
  if (!rawId || typeof rawId !== 'string') {
    throw new Error('invalid id')
  }
  const parts = rawId.split(':')
  const imdbId = parts[0]
  if (!/^tt\d+$/.test(imdbId)) {
    throw new Error(`invalid imdb id: ${rawId}`)
  }
  if (parts.length === 1) {
    return { imdbId, season: null, episode: null }
  }
  if (parts.length === 3) {
    const season = parseInt(parts[1], 10)
    const episode = parseInt(parts[2], 10)
    if (!Number.isFinite(season) || !Number.isFinite(episode)) {
      throw new Error(`invalid series id: ${rawId}`)
    }
    return { imdbId, season, episode }
  }
  throw new Error(`invalid id: ${rawId}`)
}

async function fetchJson(url) {
  const res = await fetch(url, { signal: AbortSignal.timeout(REQUEST_TIMEOUT_MS) })
  if (!res.ok) throw new Error(`TMDB ${res.status}`)
  return res.json()
}

async function findByImdb(imdbId, apiKey) {
  const cacheKey = `find:${imdbId}`
  const cached = cache.get(cacheKey)
  if (cached) return cached
  const url = `${TMDB_BASE}/find/${imdbId}?external_source=imdb_id&api_key=${apiKey}`
  const data = await fetchJson(url)
  cache.set(cacheKey, data, DEFAULT_TTL_MS)
  return data
}

async function fetchEpisodeTitle(tvId, season, episode, apiKey) {
  const cacheKey = `ep:${tvId}:${season}:${episode}`
  const cached = cache.get(cacheKey)
  if (cached !== undefined) return cached
  try {
    const url = `${TMDB_BASE}/tv/${tvId}/season/${season}/episode/${episode}?api_key=${apiKey}`
    const data = await fetchJson(url)
    const name = data.name || null
    cache.set(cacheKey, name, DEFAULT_TTL_MS)
    return name
  } catch {
    return null
  }
}

export async function resolveImdbId(rawId, config) {
  const { imdbId, season, episode } = parseStremioId(rawId)
  const find = await findByImdb(imdbId, config.tmdbApiKey)

  const movie = find.movie_results && find.movie_results[0]
  const tv = find.tv_results && find.tv_results[0]

  if (season != null && tv) {
    const episodeTitle = await fetchEpisodeTitle(tv.id, season, episode, config.tmdbApiKey)
    return {
      type: 'series',
      title: tv.name || tv.original_name,
      titleFr: tv.name,
      titleEn: tv.original_name,
      year: tv.first_air_date ? parseInt(tv.first_air_date.slice(0, 4), 10) : null,
      season,
      episode,
      episodeTitle,
    }
  }

  if (movie) {
    return {
      type: 'movie',
      title: movie.title || movie.original_title,
      titleFr: movie.title,
      titleEn: movie.original_title,
      year: movie.release_date ? parseInt(movie.release_date.slice(0, 4), 10) : null,
    }
  }

  if (tv) {
    return {
      type: 'series',
      title: tv.name || tv.original_name,
      titleFr: tv.name,
      titleEn: tv.original_name,
      year: tv.first_air_date ? parseInt(tv.first_air_date.slice(0, 4), 10) : null,
    }
  }

  throw new Error(`TMDB: no result for ${imdbId}`)
}

export function titleVariants(meta) {
  const set = new Set([meta.title, meta.titleFr, meta.titleEn].filter(Boolean))
  return [...set]
}

export function clearCache() {
  cache.clear()
}
