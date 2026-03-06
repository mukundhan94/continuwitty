import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BrowserRouter } from 'react-router-dom'
import { ThemeProvider } from 'styled-components'
import { describe, expect, it, vi } from 'vitest'

import type { UserProfile } from '../api/types'
import { lightTheme } from '../styles/theme'
import { WorkspaceTopNav } from './WorkspaceTopNav'

function buildUser(role: UserProfile['role']): UserProfile {
  return {
    user_id: 'user-1',
    username: 'admin',
    role,
  }
}

function renderTopNav(
  overrides: Partial<{
    user: UserProfile
    isAdmin: boolean
    mode: 'light' | 'dark'
    onToggleTheme: () => void
    onLogout: () => void
  }> = {},
) {
  const onToggleTheme = overrides.onToggleTheme ?? vi.fn()
  const onLogout = overrides.onLogout ?? vi.fn()

  render(
    <ThemeProvider theme={lightTheme}>
      <BrowserRouter>
        <WorkspaceTopNav
          user={overrides.user ?? buildUser('admin')}
          isAdmin={overrides.isAdmin ?? true}
          mode={overrides.mode ?? 'light'}
          breadcrumbs={[
            { label: 'App', to: '/app/workspace' },
            { label: 'Workspace', to: '/app/workspace' },
          ]}
          onToggleTheme={onToggleTheme}
          onLogout={onLogout}
        />
      </BrowserRouter>
    </ThemeProvider>,
  )

  return {
    onToggleTheme,
    onLogout,
  }
}

describe('WorkspaceTopNav', () => {
  it('renders identity, nav links, and common actions', async () => {
    const user = userEvent.setup()
    const { onToggleTheme, onLogout } = renderTopNav({ mode: 'dark' })

    expect(screen.getByText('admin · admin')).toBeInTheDocument()
    expect(screen.getAllByRole('link', { name: 'Workspace' }).length).toBeGreaterThan(0)
    expect(screen.getByRole('link', { name: 'Agents' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Transfer' })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Light Theme' }))
    await user.click(screen.getByRole('button', { name: 'Logout' }))

    expect(onToggleTheme).toHaveBeenCalledTimes(1)
    expect(onLogout).toHaveBeenCalledTimes(1)
  })

  it('shows admin nav link for admins only', () => {
    renderTopNav({ isAdmin: true })
    expect(screen.getByRole('link', { name: 'Admin' })).toBeInTheDocument()
  })

  it('hides admin nav link for non-admin users', () => {
    renderTopNav({ user: buildUser('analyst'), isAdmin: false })
    expect(screen.queryByRole('link', { name: 'Admin' })).not.toBeInTheDocument()
  })
})
