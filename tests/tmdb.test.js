import { jest } from '@jest/globals'
import { resolveImdbId, parseStremioId, clearCache } from '../src/tmdb.js'

describe('parseStremioId', () => {
  test('movie id', () => {
    expect(parseStremioId('tt0111161')).toEqual({
      imdbId: 'tt0111161',
      season: null,
      episode: null,
    })
  })
  test('series episode id', () => {
    expect(parseStremioId('tt0903747:1:3')).toEqual({
      imdbId: 'tt0903747',
      season: 1,
      episode: 3,
    })
  })
  test('throws on garbage', () => {
    expect(() => parseStremioId('not-an-id')).toThrow()
  })
})

describe('resolveImdbId — movie', () => {
  const config = { tmdbApiKey: 'key', addonPublicUrl: 'https://x' }

  beforeEach(() => {
    clearCache()
    global.fetch = jest.fn(async url => {
      if (url.includes('/find/tt0111161')) {
        return {
          ok: true,
          json: async () => ({
            movie_results: [
              {
                id: 1,
                title: 'The Shawshank Redemption',
                original_title: 'The Shawshank Redemption',
                release_date: '1994-09-23',
              },
            ],
            tv_results: [],
          }),
        }
      }
      return { ok: false, status: 404, json: async () => ({}) }
    })
  })

  afterEach(() => {
    delete global.fetch
  })

  test('returns title + year + type=movie', async () => {
    const out = await resolveImdbId('tt0111161', config)
    expect(out.type).toBe('movie')
    expect(out.year).toBe(1994)
    expect(out.title).toBe('The Shawshank Redemption')
  })

  test('caches repeated calls', async () => {
    await resolveImdbId('tt0111161', config)
    await resolveImdbId('tt0111161', config)
    expect(global.fetch).toHaveBeenCalledTimes(1)
  })
})

describe('resolveImdbId — series episode', () => {
  const config = { tmdbApiKey: 'key' }

  beforeEach(() => {
    clearCache()
    global.fetch = jest.fn(async url => {
      if (url.includes('/find/tt0903747')) {
        return {
          ok: true,
          json: async () => ({
            movie_results: [],
            tv_results: [
              {
                id: 1396,
                name: 'Breaking Bad',
                original_name: 'Breaking Bad',
                first_air_date: '2008-01-20',
              },
            ],
          }),
        }
      }
      if (url.includes('/tv/1396/season/1/episode/3')) {
        return { ok: true, json: async () => ({ name: 'And the Bag’s in the River' }) }
      }
      return { ok: false, status: 404, json: async () => ({}) }
    })
  })

  afterEach(() => {
    delete global.fetch
  })

  test('returns episode metadata', async () => {
    const out = await resolveImdbId('tt0903747:1:3', config)
    expect(out.type).toBe('series')
    expect(out.season).toBe(1)
    expect(out.episode).toBe(3)
    expect(out.episodeTitle).toBe('And the Bag’s in the River')
  })
})
