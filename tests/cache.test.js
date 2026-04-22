import { jest } from '@jest/globals'
import { createCache } from '../src/cache.js'

describe('cache', () => {
  beforeEach(() => jest.useFakeTimers({ now: 0 }))
  afterEach(() => jest.useRealTimers())

  test('set + get returns the value before TTL', () => {
    const c = createCache()
    c.set('a', 42, 1000)
    expect(c.get('a')).toBe(42)
  })

  test('get returns undefined after TTL', () => {
    const c = createCache()
    c.set('a', 42, 1000)
    jest.advanceTimersByTime(1001)
    expect(c.get('a')).toBeUndefined()
  })

  test('clear removes all entries', () => {
    const c = createCache()
    c.set('a', 1, 10000)
    c.set('b', 2, 10000)
    c.clear()
    expect(c.get('a')).toBeUndefined()
    expect(c.get('b')).toBeUndefined()
  })

  test('overwriting resets TTL', () => {
    const c = createCache()
    c.set('a', 1, 1000)
    jest.advanceTimersByTime(500)
    c.set('a', 2, 1000)
    jest.advanceTimersByTime(700)
    expect(c.get('a')).toBe(2)
  })
})
