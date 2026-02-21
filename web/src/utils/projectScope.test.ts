import { describe, expect, it } from 'vitest'

import { initialProjectId, normalizeProjectId } from './projectScope'

describe('project scope helpers', () => {
  it('normalizes project id values using trim and fallback default', () => {
    expect(normalizeProjectId('  custom-project  ')).toBe('custom-project')
    expect(normalizeProjectId('   ')).toBe('engram-vault')
  })

  it('resolves initial project id from stored value when present', () => {
    expect(initialProjectId('  stored-project  ')).toBe('stored-project')
  })

  it('falls back to default project id when stored value is empty', () => {
    expect(initialProjectId('   ')).toBe('engram-vault')
    expect(initialProjectId(null)).toBe('engram-vault')
  })
})
