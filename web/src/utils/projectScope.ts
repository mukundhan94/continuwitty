import { WEB_CONFIG } from '../config'

export const PROJECT_ID_STORAGE_KEY = 'engram.lastProjectId'

export function normalizeProjectId(value: string): string {
  const trimmed = value.trim()
  return trimmed || WEB_CONFIG.defaultProjectId
}

export function initialProjectId(
  storedProjectId: string | null = typeof window === 'undefined'
    ? null
    : window.localStorage.getItem(PROJECT_ID_STORAGE_KEY),
): string {
  return normalizeProjectId(storedProjectId || WEB_CONFIG.defaultProjectId)
}
