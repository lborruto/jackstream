import { jest } from '@jest/globals'
import { buildApp, encodeConfig } from '../src/addon.js'

async function getJson(app, url) {
  return new Promise((resolve, reject) => {
    const server = app.listen(0, async () => {
      const { port } = server.address()
      try {
        const res = await fetch(`http://127.0.0.1:${port}${url}`)
        const body = await res.json().catch(() => null)
        server.close()
        resolve({ status: res.status, body })
      } catch (err) {
        server.close()
        reject(err)
      }
    })
  })
}

describe('addon smoke', () => {
  jest.setTimeout(10000)

  test('GET /manifest.json returns manifest with configurationRequired=true', async () => {
    const app = buildApp()
    const { status, body } = await getJson(app, '/manifest.json')
    expect(status).toBe(200)
    expect(body.id).toBe('community.jackstream')
    expect(body.behaviorHints.configurationRequired).toBe(true)
  })

  test('GET /{config}/manifest.json with valid config returns configurationRequired=false', async () => {
    const app = buildApp()
    const cfg = encodeConfig({
      jackettUrl: 'http://127.0.0.1:9117',
      jackettApiKey: 'x',
      tmdbApiKey: 'y',
      addonPublicUrl: 'http://127.0.0.1:7000',
    })
    const { status, body } = await getJson(app, `/${cfg}/manifest.json`)
    expect(status).toBe(200)
    expect(body.behaviorHints.configurationRequired).toBe(false)
  })

  test('GET /{bad}/manifest.json returns 400', async () => {
    const app = buildApp()
    const { status, body } = await getJson(app, '/not-a-config/manifest.json')
    expect(status).toBe(400)
    expect(body.error).toBe('invalid_config')
  })

  test('GET /health returns ok', async () => {
    const app = buildApp()
    const { status, body } = await getJson(app, '/health')
    expect(status).toBe(200)
    expect(body).toEqual({ ok: true })
  })
})
