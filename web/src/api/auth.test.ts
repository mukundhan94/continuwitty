import { describe, expect, it } from 'vitest'

import { ApiError } from './http'
import { extractCsrfTokenFromHtml } from './auth'

describe('extractCsrfTokenFromHtml', () => {
  it('extracts csrf token from login markup', () => {
    const html = '<form><input type="hidden" name="csrf_token" value="abc123" /></form>'
    expect(extractCsrfTokenFromHtml(html)).toBe('abc123')
  })

  it('throws when token is missing', () => {
    expect(() => extractCsrfTokenFromHtml('<html></html>')).toThrow(ApiError)
  })
})
