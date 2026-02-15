export const colorTokens = {
  bgSoft: '#f7f3ed',
  bgStrong: '#e8dfd2',
  ink: '#1d2430',
  inkMuted: '#5b6372',
  accent: '#be4f2a',
  accentAlt: '#305d9c',
  card: '#ffffff',
  line: '#d8dce4',
  success: '#295ea8',
  error: '#a53a2a',
} as const

export const fontTokens = {
  display: "'Avenir Next', 'Trebuchet MS', 'Segoe UI', sans-serif",
  body: "'Avenir Next', 'Trebuchet MS', 'Segoe UI', sans-serif",
  mono: "'IBM Plex Mono', 'Menlo', monospace",
} as const

export const radiusTokens = {
  md: '10px',
  lg: '16px',
  xl: '20px',
} as const

export const shadowTokens = {
  panel: '0 16px 35px rgba(31, 42, 48, 0.08)',
  nav: '0 14px 38px rgba(30, 42, 48, 0.1)',
  card: '0 24px 60px rgba(26, 35, 39, 0.12)',
  modal: '0 28px 70px rgba(19, 29, 34, 0.3)',
} as const

export const appTheme = {
  colors: colorTokens,
  fonts: fontTokens,
  radius: radiusTokens,
  shadows: shadowTokens,
} as const

export type AppTheme = typeof appTheme
