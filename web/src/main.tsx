import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'

import ThemedApp from './ThemedApp'
import './index.css'
import { ThemeModeProvider } from './styles/themeMode'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ThemeModeProvider>
      <BrowserRouter>
        <ThemedApp />
      </BrowserRouter>
    </ThemeModeProvider>
  </StrictMode>,
)
