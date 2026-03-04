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
  display: "'Comfortaa', 'Quicksand', 'Nunito', sans-serif",
  body: "'Nunito', 'Quicksand', sans-serif",
  mono: "'JetBrains Mono', 'IBM Plex Mono', monospace",
} as const

export const radiusTokens = {
  md: '14px',
  lg: '20px',
  xl: '20px',
} as const

export const shadowTokens = {
  panel: '0 24px 56px rgba(1, 8, 16, 0.42)',
  nav: '0 20px 52px rgba(1, 8, 16, 0.46)',
  card: '0 26px 72px rgba(1, 8, 16, 0.48)',
  modal: '0 32px 86px rgba(1, 8, 16, 0.56)',
} as const

const lightColors: ThemeColors = {
  bgSoft: '#edf4f4',
  bgStrong: '#d8e8e7',
  bgGlowPrimary: 'rgba(14, 116, 144, 0.08)',
  bgGlowSecondary: 'rgba(20, 184, 166, 0.1)',
  ink: '#103f3d',
  inkMuted: '#1b5a66',
  accent: '#0f766e',
  accentAlt: '#155e75',
  card: '#f9fcfc',
  line: 'rgba(15, 118, 110, 0.2)',
  success: '#0f766e',
  error: '#fb923c',
  surfaceGlass: 'rgba(252, 255, 255, 0.82)',
  surfaceGlassBorder: 'rgba(122, 190, 183, 0.42)',
  surfaceRaised: '#f6fbfb',
  surfaceRaisedBorder: 'rgba(15, 118, 110, 0.16)',
  inputBg: '#fdfefe',
  noticeBg: 'rgba(45, 212, 191, 0.14)',
  noticeBorder: 'rgba(15, 118, 110, 0.3)',
  bubbleUserStart: '#7ed8cc',
  bubbleUserEnd: '#62c7bb',
  bubbleAssistantStart: '#d6ebfb',
  bubbleAssistantEnd: '#b8d9f5',
  markdownCodeBg: 'rgba(12, 74, 110, 0.09)',
  markdownBlockquoteBorder: 'rgba(15, 118, 110, 0.34)',
  markdownBlockquoteBg: 'rgba(45, 212, 191, 0.1)',
  markdownTableBorder: 'rgba(15, 118, 110, 0.22)',
  markdownTableHeaderBg: 'rgba(45, 212, 191, 0.14)',
  modalBackdrop: 'rgba(2, 24, 30, 0.42)',
  modalBorder: 'rgba(15, 118, 110, 0.18)',
  scrollbarTrack: 'rgba(173, 212, 206, 0.4)',
  scrollbarThumb: 'rgba(15, 118, 110, 0.56)',
  scrollbarThumbHover: 'rgba(12, 74, 110, 0.78)',
  sessionActiveBg: 'rgba(45, 212, 191, 0.12)',
  sessionActiveBorder: '#0f766e',
  sessionActiveShadow: 'rgba(15, 118, 110, 0.2)',
}

const darkColors: ThemeColors = {
  bgSoft: '#0b1a2b',
  bgStrong: '#0c3547',
  bgGlowPrimary: 'rgba(14, 116, 144, 0.12)',
  bgGlowSecondary: 'rgba(20, 184, 166, 0.15)',
  ink: '#ffffff',
  inkMuted: 'rgba(178, 245, 234, 0.5)',
  accent: '#5eead4',
  accentAlt: '#99f6e4',
  card: '#0c3547',
  line: 'rgba(94, 234, 212, 0.22)',
  success: '#5eead4',
  error: '#fb923c',
  surfaceGlass: 'rgba(11, 26, 43, 0.5)',
  surfaceGlassBorder: 'rgba(94, 234, 212, 0.08)',
  surfaceRaised: '#0c3547',
  surfaceRaisedBorder: 'rgba(94, 234, 212, 0.08)',
  inputBg: 'rgba(11, 26, 43, 0.5)',
  noticeBg: 'rgba(13, 148, 136, 0.18)',
  noticeBorder: 'rgba(94, 234, 212, 0.32)',
  bubbleUserStart: 'rgba(6, 182, 212, 0.35)',
  bubbleUserEnd: 'rgba(20, 184, 166, 0.42)',
  bubbleAssistantStart: 'rgba(12, 74, 110, 0.5)',
  bubbleAssistantEnd: 'rgba(13, 148, 136, 0.42)',
  markdownCodeBg: 'rgba(204, 251, 241, 0.16)',
  markdownBlockquoteBorder: 'rgba(94, 234, 212, 0.4)',
  markdownBlockquoteBg: 'rgba(94, 234, 212, 0.08)',
  markdownTableBorder: 'rgba(94, 234, 212, 0.2)',
  markdownTableHeaderBg: 'rgba(94, 234, 212, 0.14)',
  modalBackdrop: 'rgba(3, 10, 18, 0.8)',
  modalBorder: 'rgba(94, 234, 212, 0.3)',
  scrollbarTrack: 'rgba(12, 53, 71, 0.76)',
  scrollbarThumb: 'rgba(94, 234, 212, 0.52)',
  scrollbarThumbHover: 'rgba(94, 234, 212, 0.76)',
  sessionActiveBg: 'rgba(94, 234, 212, 0.18)',
  sessionActiveBorder: 'rgba(94, 234, 212, 0.66)',
  sessionActiveShadow: 'rgba(94, 234, 212, 0.28)',
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
