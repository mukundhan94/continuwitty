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
        body: ['var(--font-body)', 'Nunito', 'sans-serif'],
        display: ['var(--font-display)', 'Comfortaa', 'sans-serif'],
        mono: ['var(--font-mono)', 'JetBrains Mono', 'monospace'],
      },
    },
  },
  plugins: [],
}

export default config
