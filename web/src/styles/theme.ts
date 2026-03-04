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
  bgSoft: '#f0fdfa',
  bgStrong: '#d7f6ef',
  bgGlowPrimary: 'rgba(14, 165, 233, 0.22)',
  bgGlowSecondary: 'rgba(20, 184, 166, 0.24)',
  ink: '#0f3741',
  inkMuted: '#2f6068',
  accent: '#0d9488',
  accentAlt: '#0c4a6e',
  card: '#ffffff',
  line: 'rgba(13, 148, 136, 0.24)',
  success: '#0d9488',
  error: '#b45309',
  surfaceGlass: 'rgba(255, 255, 255, 0.88)',
  surfaceGlassBorder: 'rgba(167, 243, 208, 0.72)',
  surfaceRaised: '#ffffff',
  surfaceRaisedBorder: 'rgba(13, 148, 136, 0.2)',
  inputBg: '#ffffff',
  noticeBg: 'rgba(94, 234, 212, 0.2)',
  noticeBorder: 'rgba(13, 148, 136, 0.36)',
  bubbleUserStart: '#d1fae5',
  bubbleUserEnd: '#99f6e4',
  bubbleAssistantStart: '#dbeafe',
  bubbleAssistantEnd: '#bae6fd',
  markdownCodeBg: 'rgba(12, 74, 110, 0.09)',
  markdownBlockquoteBorder: 'rgba(13, 148, 136, 0.44)',
  markdownBlockquoteBg: 'rgba(94, 234, 212, 0.16)',
  markdownTableBorder: 'rgba(13, 148, 136, 0.28)',
  markdownTableHeaderBg: 'rgba(94, 234, 212, 0.22)',
  modalBackdrop: 'rgba(2, 24, 30, 0.5)',
  modalBorder: 'rgba(13, 148, 136, 0.24)',
  scrollbarTrack: 'rgba(148, 230, 216, 0.45)',
  scrollbarThumb: 'rgba(13, 148, 136, 0.62)',
  scrollbarThumbHover: 'rgba(12, 74, 110, 0.78)',
  sessionActiveBg: 'rgba(94, 234, 212, 0.2)',
  sessionActiveBorder: '#0d9488',
  sessionActiveShadow: 'rgba(13, 148, 136, 0.24)',
}

const darkColors: ThemeColors = {
  bgSoft: '#081420',
  bgStrong: '#0b1f31',
  bgGlowPrimary: 'rgba(6, 182, 212, 0.26)',
  bgGlowSecondary: 'rgba(20, 184, 166, 0.3)',
  ink: '#e8fffb',
  inkMuted: 'rgba(178, 245, 234, 0.72)',
  accent: '#5eead4',
  accentAlt: '#2bb5d9',
  card: '#0d1f31',
  line: 'rgba(94, 234, 212, 0.22)',
  success: '#5eead4',
  error: '#fb923c',
  surfaceGlass: 'rgba(8, 20, 32, 0.76)',
  surfaceGlassBorder: 'rgba(94, 234, 212, 0.22)',
  surfaceRaised: 'rgba(12, 53, 71, 0.55)',
  surfaceRaisedBorder: 'rgba(94, 234, 212, 0.18)',
  inputBg: 'rgba(8, 20, 32, 0.68)',
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
