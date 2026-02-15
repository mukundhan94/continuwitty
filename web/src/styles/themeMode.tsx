import { useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'

import type { ThemeMode } from './theme'
import type { ThemeModeContextValue } from './themeModeContext'
import { ThemeModeContext } from './themeModeContext'
import { normalizeThemeMode, resolveInitialThemeMode } from './themeModeUtils'

const THEME_MODE_STORAGE_KEY = 'engram.theme.mode'

function resolveBrowserPreferredMode(): ThemeMode {
  if (typeof window === 'undefined') {
    return 'light'
  }

  const storedMode = normalizeThemeMode(window.localStorage.getItem(THEME_MODE_STORAGE_KEY))
  const prefersDark = window.matchMedia?.('(prefers-color-scheme: dark)').matches ?? false
  return resolveInitialThemeMode(storedMode, prefersDark)
}

export function ThemeModeProvider({ children }: { children: ReactNode }) {
  const [mode, setMode] = useState<ThemeMode>(() => resolveBrowserPreferredMode())

  useEffect(() => {
    window.localStorage.setItem(THEME_MODE_STORAGE_KEY, mode)
    document.documentElement.dataset.themeMode = mode
  }, [mode])

  const value = useMemo<ThemeModeContextValue>(
    () => ({
      mode,
      setMode,
      toggleMode: () => setMode((current) => (current === 'light' ? 'dark' : 'light')),
    }),
    [mode],
  )

  return <ThemeModeContext.Provider value={value}>{children}</ThemeModeContext.Provider>
}
