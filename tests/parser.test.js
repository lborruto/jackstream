import { parseTorrentTitle } from '../src/parser.js'

describe('parser.quality', () => {
  test.each([
    ['Movie.2024.2160p.BluRay.x265', '4K'],
    ['Movie.2024.4K.HDR', '4K'],
    ['Movie.2024.1080p.WEB-DL', '1080p'],
    ['Movie.2024.720p.HDTV', '720p'],
    ['Movie.2024.480p.DVDRip', '480p'],
    ['Movie.2024.CAM.NEW', 'CAM'],
    ['Movie 2024', null],
  ])('%s → %s', (title, expected) => {
    expect(parseTorrentTitle(title).quality).toBe(expected)
  })
})

describe('parser.source', () => {
  test.each([
    ['Movie.2024.1080p.BluRay.REMUX.HEVC', 'Remux'],
    ['Movie.2024.1080p.BluRay.x264', 'BluRay'],
    ['Movie.2024.1080p.BDRip.x264', 'BluRay'],
    ['Movie.2024.1080p.WEB-DL.x264', 'WEB-DL'],
    ['Movie.2024.1080p.WEB.DL.x264', 'WEB-DL'],
    ['Movie.2024.1080p.WEBRip.x264', 'WEBRip'],
    ['Movie.2024.720p.HDTV.x264', 'HDTV'],
    ['Movie.2024.CAM', 'CAM'],
    ['Movie.2024', null],
  ])('%s → %s', (title, expected) => {
    expect(parseTorrentTitle(title).source).toBe(expected)
  })
})

describe('parser.hdr', () => {
  test.each([
    ['Movie.2024.2160p.DV.HDR10.HEVC', 'DV'],
    ['Movie.2024.2160p.Dolby.Vision.HEVC', 'DV'],
    ['Movie.2024.2160p.HDR10.HEVC', 'HDR10'],
    ['Movie.2024.2160p.HDR10+.HEVC', 'HDR10'],
    ['Movie.2024.2160p.HDR.HEVC', 'HDR'],
    ['Movie.2024.1080p.SDR', null],
  ])('%s → %s', (title, expected) => {
    expect(parseTorrentTitle(title).hdr).toBe(expected)
  })
})

describe('parser.codec', () => {
  test.each([
    ['Movie.2024.1080p.x265', 'x265'],
    ['Movie.2024.1080p.HEVC', 'x265'],
    ['Movie.2024.1080p.H.265', 'x265'],
    ['Movie.2024.1080p.x264', 'x264'],
    ['Movie.2024.1080p.H264', 'x264'],
    ['Movie.2024.1080p.AV1', 'AV1'],
    ['Movie.2024', null],
  ])('%s → %s', (title, expected) => {
    expect(parseTorrentTitle(title).codec).toBe(expected)
  })
})

describe('parser.audio priority', () => {
  test('MULTI beats FRENCH', () => {
    expect(parseTorrentTitle('Movie.2024.MULTI.FRENCH').audio).toBe('MULTI')
  })
  test('FRENCH beats VOSTFR', () => {
    expect(parseTorrentTitle('Movie.2024.FRENCH.VOSTFR').audio).toBe('FRENCH')
  })
  test('TRUEFRENCH maps to FRENCH', () => {
    expect(parseTorrentTitle('Movie.2024.TRUEFRENCH').audio).toBe('FRENCH')
  })
  test('VOSTFR beats ENG', () => {
    expect(parseTorrentTitle('Movie.2024.VOSTFR.ENG').audio).toBe('VOSTFR')
  })
  test('ENGLISH → ENG', () => {
    expect(parseTorrentTitle('Movie.2024.ENGLISH').audio).toBe('ENG')
  })
  test('no audio → null', () => {
    expect(parseTorrentTitle('Movie.2024').audio).toBeNull()
  })
})

describe('parser — realistic composite', () => {
  test('4K HDR10 Remux HEVC MULTI', () => {
    const out = parseTorrentTitle(
      'The.Substance.2024.2160p.UHD.BluRay.REMUX.HDR10.HEVC.MULTI-GROUP'
    )
    expect(out).toEqual({
      quality: '4K',
      source: 'Remux',
      hdr: 'HDR10',
      codec: 'x265',
      audio: 'MULTI',
    })
  })
})
