import { filterTorrents } from '../src/filter.js'

const t = (overrides = {}) => ({
  title: overrides.title ?? 'Some Movie 2024 1080p WEB-DL x264',
  parsedTitle: {
    quality: 'quality' in overrides ? overrides.quality : '1080p',
    source: overrides.source ?? 'WEB-DL',
    hdr: overrides.hdr ?? null,
    audio: overrides.audio ?? null,
  },
  size: 'size' in overrides ? overrides.size : 2 * 1024 ** 3, // 2 GB default
  seeders: overrides.seeders ?? 10,
})

const GB = 1024 ** 3
const MB = 1024 ** 2

describe('filterTorrents', () => {
  test('no config options → everything passes through', () => {
    const items = [
      t({ quality: '4K' }),
      t({ quality: '1080p' }),
      t({ quality: '720p' }),
      t({ quality: '480p' }),
      t({ quality: 'CAM' }),
      t({ quality: null }),
    ]
    expect(filterTorrents(items, {})).toHaveLength(items.length)
  })

  test("maxQuality: '720p' drops 1080p and 4K, keeps 720p, 480p, null", () => {
    const items = [
      t({ quality: '4K' }),
      t({ quality: '1080p' }),
      t({ quality: '720p' }),
      t({ quality: '480p' }),
      t({ quality: null }),
    ]
    const out = filterTorrents(items, { maxQuality: '720p' })
    const qualities = out.map(x => x.parsedTitle.quality)
    expect(qualities).toEqual(['720p', '480p', null])
  })

  test("minQuality: '720p' drops 480p and null, keeps 720p+", () => {
    const items = [
      t({ quality: '4K' }),
      t({ quality: '1080p' }),
      t({ quality: '720p' }),
      t({ quality: '480p' }),
      t({ quality: null }),
    ]
    const out = filterTorrents(items, { minQuality: '720p' })
    const qualities = out.map(x => x.parsedTitle.quality)
    expect(qualities).toEqual(['4K', '1080p', '720p'])
  })

  test('maxSizeGb: 10 drops >10 GB', () => {
    const items = [
      t({ size: 5 * GB }),
      t({ size: 10 * GB }), // on the boundary — kept (not strictly greater)
      t({ size: 15 * GB }),
    ]
    const out = filterTorrents(items, { maxSizeGb: 10 })
    expect(out.map(x => x.size)).toEqual([5 * GB, 10 * GB])
  })

  test('minSizeMb: 500 drops <500 MB', () => {
    const items = [
      t({ size: 100 * MB }),
      t({ size: 500 * MB }), // on the boundary — kept (not strictly less)
      t({ size: 900 * MB }),
    ]
    const out = filterTorrents(items, { minSizeMb: 500 })
    expect(out.map(x => x.size)).toEqual([500 * MB, 900 * MB])
  })

  test("blacklistKeywords drops matching titles, word-boundary (SCAM not matched by CAM)", () => {
    const items = [
      t({ title: 'Movie CAM Rip' }),
      t({ title: 'Movie HDCAM 2024' }),
      t({ title: 'A SCAM Story' }),
      t({ title: 'Clean Title 1080p' }),
    ]
    const out = filterTorrents(items, { blacklistKeywords: ['CAM', 'HDCAM'] })
    expect(out.map(x => x.title)).toEqual(['A SCAM Story', 'Clean Title 1080p'])
  })

  test("regex injection: blacklist like ['.*'] is escaped and doesn't match everything", () => {
    const items = [
      t({ title: 'Movie 1080p' }),
      t({ title: 'Literal .* match' }),
    ]
    const out = filterTorrents(items, { blacklistKeywords: ['.*'] })
    // The first title does NOT contain ".*" literally, so must not be dropped.
    expect(out.some(x => x.title === 'Movie 1080p')).toBe(true)
  })

  test('unknown size passes through size filters (size not finite)', () => {
    const items = [t({ size: undefined }), t({ size: null })]
    const out = filterTorrents(items, { maxSizeGb: 1, minSizeMb: 50 })
    expect(out).toHaveLength(2)
  })

  test('combined filters compose', () => {
    const items = [
      t({ quality: '4K', size: 30 * GB, title: 'Big 4K' }),
      t({ quality: '1080p', size: 5 * GB, title: 'Good 1080p' }),
      t({ quality: '720p', size: 50 * MB, title: 'Tiny 720p' }),
      t({ quality: '480p', size: 1 * GB, title: 'Old 480p' }),
    ]
    const out = filterTorrents(items, {
      maxQuality: '1080p',
      minQuality: '720p',
      maxSizeGb: 10,
      minSizeMb: 100,
    })
    expect(out.map(x => x.title)).toEqual(['Good 1080p'])
  })
})
