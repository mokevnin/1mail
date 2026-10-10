import { describe, expect, test } from 'vitest'

import { dispositionFilename } from './download.ts'

describe('dispositionFilename', () => {
  test('reads the filename the server suggested', () => {
    const response = new Response('', {
      headers: { 'Content-Disposition': 'attachment; filename="contact-7-export.json"' },
    })
    expect(dispositionFilename(response)).toBe('contact-7-export.json')
  })

  test('is undefined without the header', () => {
    expect(dispositionFilename(new Response(''))).toBeUndefined()
  })
})
