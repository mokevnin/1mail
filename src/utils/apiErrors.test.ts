import { describe, expect, test } from 'vitest'

import { getApiErrorMessage, isForbiddenError, isRateLimitedError } from './apiErrors.ts'

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

describe('isRateLimitedError', () => {
  test('is true only for a 429 problem body', () => {
    expect(isRateLimitedError({ status: 429, detail: 'rate limit exceeded' })).toBe(true)
    expect(isRateLimitedError({ status: 403 })).toBe(false)
    expect(isRateLimitedError(null)).toBe(false)
  })
})

describe('isForbiddenError', () => {
  test('is true only for a 403 problem body', () => {
    expect(isForbiddenError({ status: 403, detail: 'insufficient role' })).toBe(true)
    expect(isForbiddenError({ status: 400 })).toBe(false)
    expect(isForbiddenError({ detail: 'no status' })).toBe(false)
    expect(isForbiddenError(null)).toBe(false)
  })
})
