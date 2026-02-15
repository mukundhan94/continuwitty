export type ThemeMode = 'light' | 'dark'

export interface ThemeColors {
  bgSoft: string
  bgStrong: string
  bgGlowPrimary: string
  bgGlowSecondary: string
  ink: string
  inkMuted: string
  accent: string
  accentAlt: string
  card: string
  line: string
  success: string
  error: string
  surfaceGlass: string
  surfaceGlassBorder: string
  surfaceRaised: string
  surfaceRaisedBorder: string
  inputBg: string
  noticeBg: string
  noticeBorder: string
  bubbleUserStart: string
  bubbleUserEnd: string
  bubbleAssistantStart: string
  bubbleAssistantEnd: string
  markdownCodeBg: string
  markdownBlockquoteBorder: string
  markdownBlockquoteBg: string
  markdownTableBorder: string
  markdownTableHeaderBg: string
  modalBackdrop: string
  modalBorder: string
  scrollbarTrack: string
  scrollbarThumb: string
  scrollbarThumbHover: string
  sessionActiveBg: string
  sessionActiveBorder: string
  sessionActiveShadow: string
}

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

const lightColors: ThemeColors = {
  bgSoft: '#f7f3ed',
  bgStrong: '#e8dfd2',
  bgGlowPrimary: '#ffd8bf',
  bgGlowSecondary: '#d7e0ff',
  ink: '#1d2430',
  inkMuted: '#5b6372',
  accent: '#be4f2a',
  accentAlt: '#305d9c',
  card: '#ffffff',
  line: '#d8dce4',
  success: '#295ea8',
  error: '#a53a2a',
  surfaceGlass: 'rgba(255, 255, 255, 0.9)',
  surfaceGlassBorder: 'rgba(255, 255, 255, 0.75)',
  surfaceRaised: '#ffffff',
  surfaceRaisedBorder: '#f0f0f0',
  inputBg: '#ffffff',
  noticeBg: '#e8efff',
  noticeBorder: '#c5d4fb',
  bubbleUserStart: '#fbe2cb',
  bubbleUserEnd: '#f8c8aa',
  bubbleAssistantStart: '#e7ecff',
  bubbleAssistantEnd: '#d1dcff',
  markdownCodeBg: 'rgba(15, 23, 42, 0.08)',
  markdownBlockquoteBorder: '#b8c7ef',
  markdownBlockquoteBg: 'rgba(48, 93, 156, 0.05)',
  markdownTableBorder: '#cfd9ec',
  markdownTableHeaderBg: 'rgba(48, 93, 156, 0.1)',
  modalBackdrop: 'rgba(18, 28, 32, 0.45)',
  modalBorder: '#f0f0f0',
  scrollbarTrack: 'rgba(183, 196, 226, 0.55)',
  scrollbarThumb: 'rgba(70, 98, 153, 0.7)',
  scrollbarThumbHover: 'rgba(48, 93, 156, 0.9)',
  sessionActiveBg: '#e7efff',
  sessionActiveBorder: '#7da0e5',
  sessionActiveShadow: 'rgba(48, 93, 156, 0.24)',
}

const darkColors: ThemeColors = {
  bgSoft: '#0f1420',
  bgStrong: '#1a2133',
  bgGlowPrimary: 'rgba(190, 79, 42, 0.32)',
  bgGlowSecondary: 'rgba(77, 108, 172, 0.36)',
  ink: '#e7ebf4',
  inkMuted: '#9aa6bc',
  accent: '#d9673a',
  accentAlt: '#8ab4ff',
  card: '#1b2233',
  line: '#2f3b56',
  success: '#8ab4ff',
  error: '#ff9987',
  surfaceGlass: 'rgba(23, 30, 45, 0.9)',
  surfaceGlassBorder: 'rgba(112, 130, 166, 0.28)',
  surfaceRaised: '#141c2d',
  surfaceRaisedBorder: '#2f3b56',
  inputBg: '#0f1626',
  noticeBg: '#1a2946',
  noticeBorder: '#3a5e9b',
  bubbleUserStart: '#6c3c26',
  bubbleUserEnd: '#8f4e32',
  bubbleAssistantStart: '#263a63',
  bubbleAssistantEnd: '#345184',
  markdownCodeBg: 'rgba(138, 180, 255, 0.2)',
  markdownBlockquoteBorder: '#5e7fbe',
  markdownBlockquoteBg: 'rgba(94, 127, 190, 0.16)',
  markdownTableBorder: '#3f5277',
  markdownTableHeaderBg: 'rgba(94, 127, 190, 0.28)',
  modalBackdrop: 'rgba(6, 9, 16, 0.72)',
  modalBorder: '#304263',
  scrollbarTrack: 'rgba(49, 62, 89, 0.8)',
  scrollbarThumb: 'rgba(124, 150, 203, 0.72)',
  scrollbarThumbHover: 'rgba(160, 184, 232, 0.92)',
  sessionActiveBg: '#182948',
  sessionActiveBorder: '#8ab4ff',
  sessionActiveShadow: 'rgba(138, 180, 255, 0.34)',
}

export interface ThemeDefinition {
  colors: ThemeColors
  fonts: typeof fontTokens
  radius: typeof radiusTokens
  shadows: typeof shadowTokens
}

export const lightTheme: ThemeDefinition = {
  colors: lightColors,
  fonts: fontTokens,
  radius: radiusTokens,
  shadows: shadowTokens,
}

export const darkTheme: ThemeDefinition = {
  colors: darkColors,
  fonts: fontTokens,
  radius: radiusTokens,
  shadows: shadowTokens,
}

export const appThemes: Record<ThemeMode, ThemeDefinition> = {
  light: lightTheme,
  dark: darkTheme,
}

export type AppTheme = ThemeDefinition
