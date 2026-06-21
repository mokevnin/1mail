import { describe, expect, test } from 'vitest'
import { getApiErrorMessage } from './apiErrors.ts'

describe('getApiErrorMessage', () => {
  test('prefers detail over every other field', () => {
    const error = { detail: 'detail', form: 'form', error: 'error', title: 'title' }
    expect(getApiErrorMessage(error, 'fallback')).toBe('detail')
  })

  test('falls through detail → form → error → title → fallback', () => {
    expect(getApiErrorMessage({ form: 'form', error: 'error', title: 'title' }, 'fallback')).toBe(
      'form',
    )
    expect(getApiErrorMessage({ error: 'error', title: 'title' }, 'fallback')).toBe('error')
    expect(getApiErrorMessage({ title: 'title' }, 'fallback')).toBe('title')
  })

  test('returns the fallback when error is null, undefined, or empty', () => {
    expect(getApiErrorMessage(null, 'fallback')).toBe('fallback')
    expect(getApiErrorMessage(undefined, 'fallback')).toBe('fallback')
    expect(getApiErrorMessage({}, 'fallback')).toBe('fallback')
  })
})
