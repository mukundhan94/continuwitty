import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import ThemedApp from './ThemedApp'
import './index.css'
import { ThemeModeProvider } from './styles/themeMode'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ThemeModeProvider>
      <ThemedApp />
    </ThemeModeProvider>
  </StrictMode>,
)
