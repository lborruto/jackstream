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

  test.each(['jackettUrl', 'jackettApiKey', 'tmdbApiKey', 'addonPublicUrl'])(
    'throws when %s is missing',
    field => {
      const copy = { ...sample }
      delete copy[field]
      expect(() => validateConfig(copy)).toThrow(field)
    }
  )

  test('throws when jackettUrl is not a valid URL', () => {
    expect(() => validateConfig({ ...sample, jackettUrl: 'not a url' })).toThrow()
  })

  test('applies default maxResults=10, minSeeders=1, maxConcurrentTorrents=3', () => {
    const out = validateConfig(sample)
    expect(out.maxResults).toBe(10)
    expect(out.minSeeders).toBe(1)
    expect(out.maxConcurrentTorrents).toBe(3)
  })

  describe('advanced filter fields', () => {
    test('default values when fields are absent', () => {
      const out = validateConfig(sample)
      expect(out.preferredLanguage).toBeNull()
      expect(out.maxQuality).toBeNull()
      expect(out.minQuality).toBeNull()
      expect(out.maxSizeGb).toBeNull()
      expect(out.minSizeMb).toBeNull()
      expect(out.blacklistKeywords).toEqual([])
    })

    test.each(['FRENCH', 'MULTI', 'VOSTFR', 'ENG'])(
      'preferredLanguage accepts %s',
      lang => {
        const out = validateConfig({ ...sample, preferredLanguage: lang })
        expect(out.preferredLanguage).toBe(lang)
      }
    )

    test('preferredLanguage invalid value falls back to null', () => {
      const out = validateConfig({ ...sample, preferredLanguage: 'KLINGON' })
      expect(out.preferredLanguage).toBeNull()
    })

    test.each(['4K', '1080p', '720p', '480p'])('maxQuality accepts %s', q => {
      const out = validateConfig({ ...sample, maxQuality: q })
      expect(out.maxQuality).toBe(q)
    })

    test('maxQuality invalid value falls back to null', () => {
      const out = validateConfig({ ...sample, maxQuality: '8K' })
      expect(out.maxQuality).toBeNull()
    })

    test.each(['4K', '1080p', '720p', '480p'])('minQuality accepts %s', q => {
      const out = validateConfig({ ...sample, minQuality: q })
      expect(out.minQuality).toBe(q)
    })

    test('minQuality invalid value falls back to null', () => {
      const out = validateConfig({ ...sample, minQuality: 'garbage' })
      expect(out.minQuality).toBeNull()
    })

    test('maxSizeGb accepts positive finite number', () => {
      const out = validateConfig({ ...sample, maxSizeGb: 20 })
      expect(out.maxSizeGb).toBe(20)
    })

    test('maxSizeGb zero/negative/NaN falls back to null', () => {
      expect(validateConfig({ ...sample, maxSizeGb: 0 }).maxSizeGb).toBeNull()
      expect(validateConfig({ ...sample, maxSizeGb: -5 }).maxSizeGb).toBeNull()
      expect(validateConfig({ ...sample, maxSizeGb: 'big' }).maxSizeGb).toBeNull()
    })

    test('minSizeMb accepts positive finite number', () => {
      const out = validateConfig({ ...sample, minSizeMb: 500 })
      expect(out.minSizeMb).toBe(500)
    })

    test('minSizeMb zero/negative/NaN falls back to null', () => {
      expect(validateConfig({ ...sample, minSizeMb: 0 }).minSizeMb).toBeNull()
      expect(validateConfig({ ...sample, minSizeMb: -1 }).minSizeMb).toBeNull()
      expect(validateConfig({ ...sample, minSizeMb: 'tiny' }).minSizeMb).toBeNull()
    })

    test('blacklistKeywords accepts array of strings, trims and drops empties', () => {
      const out = validateConfig({
        ...sample,
        blacklistKeywords: ['CAM', '  HDCAM  ', '', '   ', 42, null, 'TELESYNC'],
      })
      expect(out.blacklistKeywords).toEqual(['CAM', 'HDCAM', 'TELESYNC'])
    })

    test('blacklistKeywords non-array falls back to []', () => {
      expect(validateConfig({ ...sample, blacklistKeywords: 'CAM' }).blacklistKeywords).toEqual([])
      expect(validateConfig({ ...sample, blacklistKeywords: null }).blacklistKeywords).toEqual([])
    })
  })
})
