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
  surfaceMute: string
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

type ShadowTokens = {
  panel: string
  nav: string
  card: string
  modal: string
}

const lightShadowTokens: ShadowTokens = {
  panel: '0 14px 36px rgba(4, 43, 57, 0.16), inset 0 1px 0 rgba(255, 255, 255, 0.4)',
  nav: '0 16px 42px rgba(4, 43, 57, 0.18), inset 0 1px 0 rgba(255, 255, 255, 0.42)',
  card: '0 20px 50px rgba(4, 43, 57, 0.2), inset 0 1px 0 rgba(255, 255, 255, 0.44)',
  modal: '0 30px 80px rgba(2, 24, 33, 0.32)',
}

const darkShadowTokens: ShadowTokens = {
  panel: '0 22px 58px rgba(1, 7, 14, 0.64), 0 0 0 1px rgba(99, 226, 209, 0.07), 0 0 26px rgba(34, 199, 180, 0.08)',
  nav: '0 24px 62px rgba(1, 7, 14, 0.68), 0 0 0 1px rgba(99, 226, 209, 0.09), 0 0 34px rgba(34, 199, 180, 0.09)',
  card: '0 28px 74px rgba(1, 7, 14, 0.72), 0 0 0 1px rgba(99, 226, 209, 0.1), 0 0 38px rgba(34, 199, 180, 0.1)',
  modal: '0 34px 90px rgba(1, 7, 14, 0.76), 0 0 0 1px rgba(99, 226, 209, 0.12), 0 0 44px rgba(34, 199, 180, 0.12)',
}

const lightColors: ThemeColors = {
  bgSoft: '#e6f1f0',
  bgStrong: '#c6dad7',
  bgGlowPrimary: 'rgba(14, 116, 144, 0.11)',
  bgGlowSecondary: 'rgba(20, 184, 166, 0.12)',
  ink: '#0f3a40',
  inkMuted: '#2b5f6a',
  accent: '#0f766e',
  accentAlt: '#0b728c',
  card: '#f4f9f8',
  line: 'rgba(15, 118, 110, 0.24)',
  success: '#0f766e',
  error: '#fb923c',
  surfaceGlass: 'rgba(246, 252, 251, 0.86)',
  surfaceGlassBorder: 'rgba(111, 171, 166, 0.44)',
  surfaceRaised: '#eef7f6',
  surfaceRaisedBorder: 'rgba(15, 118, 110, 0.2)',
  surfaceMute: 'rgba(227, 239, 237, 0.92)',
  inputBg: '#f3f9f8',
  noticeBg: 'rgba(15, 118, 110, 0.12)',
  noticeBorder: 'rgba(15, 118, 110, 0.34)',
  bubbleUserStart: '#7ed8cc',
  bubbleUserEnd: '#62c7bb',
  bubbleAssistantStart: '#d6ebfb',
  bubbleAssistantEnd: '#b8d9f5',
  markdownCodeBg: 'rgba(12, 74, 110, 0.09)',
  markdownBlockquoteBorder: 'rgba(15, 118, 110, 0.34)',
  markdownBlockquoteBg: 'rgba(45, 212, 191, 0.1)',
  markdownTableBorder: 'rgba(15, 118, 110, 0.22)',
  markdownTableHeaderBg: 'rgba(45, 212, 191, 0.14)',
  modalBackdrop: 'rgba(3, 18, 26, 0.5)',
  modalBorder: 'rgba(15, 118, 110, 0.24)',
  scrollbarTrack: 'rgba(146, 191, 185, 0.36)',
  scrollbarThumb: 'rgba(15, 118, 110, 0.52)',
  scrollbarThumbHover: 'rgba(12, 74, 110, 0.68)',
  sessionActiveBg: 'rgba(15, 118, 110, 0.14)',
  sessionActiveBorder: 'rgba(15, 118, 110, 0.72)',
  sessionActiveShadow: 'rgba(15, 118, 110, 0.24)',
}

const darkColors: ThemeColors = {
  bgSoft: '#06111d',
  bgStrong: '#0b2f43',
  bgGlowPrimary: 'rgba(45, 212, 191, 0.2)',
  bgGlowSecondary: 'rgba(56, 189, 248, 0.18)',
  ink: '#e8fbf8',
  inkMuted: 'rgba(170, 232, 223, 0.76)',
  accent: '#63e2d1',
  accentAlt: '#22c7b4',
  card: '#10384e',
  line: 'rgba(99, 226, 209, 0.34)',
  success: '#63e2d1',
  error: '#fb923c',
  surfaceGlass: 'rgba(7, 20, 33, 0.74)',
  surfaceGlassBorder: 'rgba(99, 226, 209, 0.2)',
  surfaceRaised: '#123f57',
  surfaceRaisedBorder: 'rgba(99, 226, 209, 0.2)',
  surfaceMute: 'rgba(9, 28, 44, 0.88)',
  inputBg: 'rgba(6, 19, 31, 0.78)',
  noticeBg: 'rgba(45, 212, 191, 0.2)',
  noticeBorder: 'rgba(99, 226, 209, 0.44)',
  bubbleUserStart: 'rgba(45, 212, 191, 0.3)',
  bubbleUserEnd: 'rgba(20, 184, 166, 0.42)',
  bubbleAssistantStart: 'rgba(15, 82, 112, 0.54)',
  bubbleAssistantEnd: 'rgba(14, 116, 144, 0.42)',
  markdownCodeBg: 'rgba(204, 251, 241, 0.18)',
  markdownBlockquoteBorder: 'rgba(99, 226, 209, 0.5)',
  markdownBlockquoteBg: 'rgba(99, 226, 209, 0.12)',
  markdownTableBorder: 'rgba(99, 226, 209, 0.26)',
  markdownTableHeaderBg: 'rgba(99, 226, 209, 0.18)',
  modalBackdrop: 'rgba(1, 7, 13, 0.86)',
  modalBorder: 'rgba(99, 226, 209, 0.36)',
  scrollbarTrack: 'rgba(12, 49, 67, 0.84)',
  scrollbarThumb: 'rgba(99, 226, 209, 0.62)',
  scrollbarThumbHover: 'rgba(167, 244, 231, 0.86)',
  sessionActiveBg: 'rgba(45, 212, 191, 0.2)',
  sessionActiveBorder: 'rgba(99, 226, 209, 0.84)',
  sessionActiveShadow: 'rgba(45, 212, 191, 0.32)',
}

export interface ThemeDefinition {
  colors: ThemeColors
  fonts: typeof fontTokens
  radius: typeof radiusTokens
  shadows: ShadowTokens
}

export const lightTheme: ThemeDefinition = {
  colors: lightColors,
  fonts: fontTokens,
  radius: radiusTokens,
  shadows: lightShadowTokens,
}

export const darkTheme: ThemeDefinition = {
  colors: darkColors,
  fonts: fontTokens,
  radius: radiusTokens,
  shadows: darkShadowTokens,
}

export const appThemes: Record<ThemeMode, ThemeDefinition> = {
  light: lightTheme,
  dark: darkTheme,
}

export type AppTheme = ThemeDefinition
