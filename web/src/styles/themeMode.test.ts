import { describe, expect, it } from 'vitest'

import { normalizeThemeMode, resolveInitialThemeMode } from './themeModeUtils'

describe('theme mode helpers', () => {
  it('normalizes valid theme modes', () => {
    expect(normalizeThemeMode('LIGHT')).toBe('light')
    expect(normalizeThemeMode(' dark ')).toBe('dark')
  })

  it('returns null for invalid theme modes', () => {
    expect(normalizeThemeMode('system')).toBeNull()
    expect(normalizeThemeMode('')).toBeNull()
    expect(normalizeThemeMode(undefined)).toBeNull()
  })

  it('prefers stored mode over browser preference', () => {
    expect(resolveInitialThemeMode('dark', false)).toBe('dark')
    expect(resolveInitialThemeMode('light', true)).toBe('light')
  })

  it('falls back to browser preference when storage is empty', () => {
    expect(resolveInitialThemeMode(null, true)).toBe('dark')
    expect(resolveInitialThemeMode(null, false)).toBe('light')
  })
})
