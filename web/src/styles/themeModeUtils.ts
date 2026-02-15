import type { ThemeMode } from './theme'

export function normalizeThemeMode(value: string | null | undefined): ThemeMode | null {
  const normalized = (value || '').trim().toLowerCase()
  if (normalized === 'light' || normalized === 'dark') {
    return normalized
  }
  return null
}

export function resolveInitialThemeMode(
  storedValue: string | null | undefined,
  prefersDark: boolean,
): ThemeMode {
  const normalized = normalizeThemeMode(storedValue)
  if (normalized) {
    return normalized
  }
  return prefersDark ? 'dark' : 'light'
}
