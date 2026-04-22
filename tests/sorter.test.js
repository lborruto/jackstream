import { sortTorrents } from '../src/sorter.js'

const t = (overrides = {}) => ({
  parsedTitle: {
    quality: 'quality' in overrides ? overrides.quality : '1080p',
    source: 'source' in overrides ? overrides.source : 'WEB-DL',
    hdr: 'hdr' in overrides ? overrides.hdr : null,
    codec: 'codec' in overrides ? overrides.codec : 'x264',
    audio: 'audio' in overrides ? overrides.audio : null,
  },
  seeders: overrides.seeders ?? 10,
  _id: overrides._id ?? Math.random(),
})

describe('sortTorrents', () => {
  test('quality dominates seeders', () => {
    const a = t({ quality: '1080p', seeders: 5 })
    const b = t({ quality: '4K', seeders: 1 })
    expect(sortTorrents([a, b])[0]).toBe(b)
  })

  test('source breaks tie when quality equal', () => {
    const a = t({ quality: '1080p', source: 'WEBRip', seeders: 100 })
    const b = t({ quality: '1080p', source: 'BluRay', seeders: 1 })
    expect(sortTorrents([a, b])[0]).toBe(b)
  })

  test('hdr breaks tie when quality+source equal', () => {
    const a = t({ quality: '4K', source: 'BluRay', hdr: null, seeders: 100 })
    const b = t({ quality: '4K', source: 'BluRay', hdr: 'DV', seeders: 1 })
    expect(sortTorrents([a, b])[0]).toBe(b)
  })

  test('seeders break tie when quality+source+hdr equal', () => {
    const a = t({ quality: '1080p', source: 'WEB-DL', seeders: 5 })
    const b = t({ quality: '1080p', source: 'WEB-DL', seeders: 50 })
    expect(sortTorrents([a, b])[0]).toBe(b)
  })

  test('unknown quality sorts below known but above CAM', () => {
    const unknown = t({ quality: null, seeders: 1000 })
    const cam = t({ quality: 'CAM', seeders: 1000 })
    const hd = t({ quality: '720p', seeders: 1 })
    const sorted = sortTorrents([unknown, cam, hd])
    expect(sorted).toEqual([hd, unknown, cam])
  })

  test('CAM is always last regardless of seeders', () => {
    const cam = t({ quality: 'CAM', seeders: 100000 })
    const low = t({ quality: '480p', seeders: 1 })
    expect(sortTorrents([cam, low])[0]).toBe(low)
  })

  test('sort is stable for equal keys', () => {
    const a = t({ _id: 'a' })
    const b = t({ _id: 'b' })
    const c = t({ _id: 'c' })
    const sorted = sortTorrents([a, b, c])
    expect(sorted.map(x => x._id)).toEqual(['a', 'b', 'c'])
  })

  test('does not mutate input', () => {
    const input = [t({ quality: '1080p' }), t({ quality: '4K' })]
    const copy = [...input]
    sortTorrents(input)
    expect(input).toEqual(copy)
  })
})
