import { describe, expect, it } from 'vitest'

import { ApiError } from '../api/http'
import { describeError } from './errors'

describe('describeError', () => {
  it('returns ApiError detail for API failures', () => {
    expect(describeError(new ApiError(404, 'not found'))).toBe('not found')
  })

  it('returns Error message for generic failures', () => {
    expect(describeError(new Error('network down'))).toBe('network down')
  })

  it('returns fallback detail for unknown failure types', () => {
    expect(describeError({ detail: 'hidden' })).toBe('Unexpected error')
  })
})
