import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ThemeProvider } from 'styled-components'
import { describe, expect, it, vi } from 'vitest'

import { appTheme } from '../styles/theme'
import { LoginView } from './LoginView'

describe('LoginView', () => {
  it('submits username and password values', async () => {
    const onSubmit = vi.fn(async () => {})
    const user = userEvent.setup()

    render(
      <ThemeProvider theme={appTheme}>
        <LoginView isSubmitting={false} error={null} onSubmit={onSubmit} />
      </ThemeProvider>,
    )

    await user.clear(screen.getByLabelText(/username/i))
    await user.type(screen.getByLabelText(/username/i), 'analyst')
    await user.clear(screen.getByLabelText(/password/i))
    await user.type(screen.getByLabelText(/password/i), 'StrongPass123')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    expect(onSubmit).toHaveBeenCalledWith('analyst', 'StrongPass123')
  })
})
