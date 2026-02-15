import { ThemeProvider } from 'styled-components'

import App from './App'
import { GlobalStyle } from './styles/globalStyles'
import { appThemes } from './styles/theme'
import { useThemeMode } from './styles/useThemeMode'

export default function ThemedApp() {
  const { mode } = useThemeMode()

  return (
    <ThemeProvider theme={appThemes[mode]}>
      <GlobalStyle />
      <App />
    </ThemeProvider>
  )
}
