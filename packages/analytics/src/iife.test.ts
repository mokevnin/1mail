import { beforeEach, expect, test, vi } from 'vitest'

const { initTracking } = vi.hoisted(() => ({
  initTracking: vi.fn<(config?: { collectKey: string; baseUrl: string }) => void>(),
}))

vi.mock('./index.ts', () => ({ initTracking }))

// iife.ts runs on import, so a unique query string re-executes it for every test.
let loads = 0
async function runIife(currentScript: unknown) {
  loads += 1
  Object.defineProperty(document, 'currentScript', { value: currentScript, configurable: true })
  await import(/* @vite-ignore */ `./iife.ts?load=${loads}`)
}

beforeEach(() => {
  initTracking.mockClear()
})

test('reads the collect key and url from the script tag data attributes', async () => {
  const script = document.createElement('script')
  script.dataset.collectKey = 'ck_1'
  script.dataset.collectUrl = 'https://host'

  await runIife(script)

  expect(initTracking).toHaveBeenCalledWith({ collectKey: 'ck_1', baseUrl: 'https://host' })
})

test('defaults the base url to empty when only the key is given', async () => {
  const script = document.createElement('script')
  script.dataset.collectKey = 'ck_1'

  await runIife(script)

  expect(initTracking).toHaveBeenCalledWith({ collectKey: 'ck_1', baseUrl: '' })
})

test('leaves config to the _omq queue when the script has no key', async () => {
  await runIife(document.createElement('script'))
  await runIife(null)

  expect(initTracking).toHaveBeenCalledTimes(2)
  expect(initTracking).toHaveBeenNthCalledWith(1, undefined)
  expect(initTracking).toHaveBeenNthCalledWith(2, undefined)
})
