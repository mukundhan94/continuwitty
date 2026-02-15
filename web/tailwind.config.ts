import type { Config } from 'tailwindcss'

const config: Config = {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        bgSoft: 'var(--color-bg-soft)',
        bgStrong: 'var(--color-bg-strong)',
        ink: 'var(--color-ink)',
        inkMuted: 'var(--color-ink-muted)',
        accent: 'var(--color-accent)',
        accentAlt: 'var(--color-accent-alt)',
        line: 'var(--color-line)',
        success: 'var(--color-success)',
        error: 'var(--color-error)',
      },
      boxShadow: {
        panel: 'var(--shadow-panel)',
        nav: 'var(--shadow-nav)',
        card: 'var(--shadow-card)',
        modal: 'var(--shadow-modal)',
      },
      fontFamily: {
        body: ['var(--font-body)'],
        display: ['var(--font-display)'],
        mono: ['var(--font-mono)'],
      },
    },
  },
  plugins: [],
}

export default config
