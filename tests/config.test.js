import { encodeConfig, decodeConfig, validateConfig } from '../src/config.js'

const sample = {
  jackettUrl: 'http://192.168.1.10:9117',
  jackettApiKey: 'abc',
  tmdbApiKey: 'def',
  addonPublicUrl: 'https://addon.example.com',
}

describe('encodeConfig / decodeConfig', () => {
  test('round-trips an object', () => {
    const encoded = encodeConfig(sample)
    expect(decodeConfig(encoded)).toEqual(sample)
  })

  test('encoded form is URL-safe (no + / = chars)', () => {
    const encoded = encodeConfig(sample)
    expect(encoded).not.toMatch(/[+/=]/)
  })

  test('decodeConfig accepts standard base64 too', () => {
    const json = JSON.stringify(sample)
    const stdB64 = Buffer.from(json).toString('base64')
    expect(decodeConfig(stdB64)).toEqual(sample)
  })

  test('decodeConfig throws on invalid JSON', () => {
    expect(() => decodeConfig('!!!not-b64!!!')).toThrow()
  })
})

describe('validateConfig', () => {
  test('passes with all required fields', () => {
    expect(() => validateConfig(sample)).not.toThrow()
  })

  test.each([
    'jackettUrl',
    'jackettApiKey',
    'tmdbApiKey',
    'addonPublicUrl',
  ])('throws when %s is missing', field => {
    const copy = { ...sample }
    delete copy[field]
    expect(() => validateConfig(copy)).toThrow(field)
  })

  test('throws when jackettUrl is not a valid URL', () => {
    expect(() => validateConfig({ ...sample, jackettUrl: 'not a url' })).toThrow()
  })

  test('applies default maxResults=10, minSeeders=1, maxConcurrentTorrents=3', () => {
    const out = validateConfig(sample)
    expect(out.maxResults).toBe(10)
    expect(out.minSeeders).toBe(1)
    expect(out.maxConcurrentTorrents).toBe(3)
  })
})
