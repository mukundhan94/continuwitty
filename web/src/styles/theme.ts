export const colorTokens = {
  bgSoft: '#f6f1e9',
  bgStrong: '#e9d9c6',
  ink: '#1f2a30',
  inkMuted: '#5d6c75',
  accent: '#c35528',
  accentAlt: '#0f7a7b',
  card: '#ffffff',
  line: '#d6d9dc',
  success: '#0d7a5f',
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
