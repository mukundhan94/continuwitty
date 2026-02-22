import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
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
    isAdminMemoryRoute: boolean
    isProjectTransferRoute: boolean
    mode: 'light' | 'dark'
    onOpenAdminTokenPanel: () => void | Promise<void>
    onToggleAdminMemoryRoute: () => void
    onToggleProjectTransferRoute: () => void
    onToggleTheme: () => void
    onLogout: () => void
  }> = {},
) {
  const onOpenAdminTokenPanel = overrides.onOpenAdminTokenPanel ?? vi.fn()
  const onToggleAdminMemoryRoute = overrides.onToggleAdminMemoryRoute ?? vi.fn()
  const onToggleProjectTransferRoute =
    overrides.onToggleProjectTransferRoute ?? vi.fn()
  const onToggleTheme = overrides.onToggleTheme ?? vi.fn()
  const onLogout = overrides.onLogout ?? vi.fn()
  render(
    <ThemeProvider theme={lightTheme}>
      <WorkspaceTopNav
        user={overrides.user ?? buildUser('admin')}
        isAdmin={overrides.isAdmin ?? true}
        isAdminMemoryRoute={overrides.isAdminMemoryRoute ?? false}
        isProjectTransferRoute={overrides.isProjectTransferRoute ?? false}
        mode={overrides.mode ?? 'light'}
        onOpenAdminTokenPanel={onOpenAdminTokenPanel}
        onToggleAdminMemoryRoute={onToggleAdminMemoryRoute}
        onToggleProjectTransferRoute={onToggleProjectTransferRoute}
        onToggleTheme={onToggleTheme}
        onLogout={onLogout}
      />
    </ThemeProvider>,
  )
  return {
    onOpenAdminTokenPanel,
    onToggleAdminMemoryRoute,
    onToggleProjectTransferRoute,
    onToggleTheme,
    onLogout,
  }
}

describe('WorkspaceTopNav', () => {
  it('renders user identity and common controls', async () => {
    const user = userEvent.setup()
    const { onToggleProjectTransferRoute, onToggleTheme, onLogout } = renderTopNav({
      mode: 'dark',
    })

    expect(screen.getByText('admin · admin')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Export / Import' }))
    await user.click(screen.getByRole('button', { name: 'Light Theme' }))
    await user.click(screen.getByRole('button', { name: 'Logout' }))

    expect(onToggleProjectTransferRoute).toHaveBeenCalledTimes(1)
    expect(onToggleTheme).toHaveBeenCalledTimes(1)
    expect(onLogout).toHaveBeenCalledTimes(1)
  })

  it('shows admin controls and routes to memory admin toggle', async () => {
    const user = userEvent.setup()
    const { onOpenAdminTokenPanel, onToggleAdminMemoryRoute, onToggleProjectTransferRoute } = renderTopNav({
      isAdmin: true,
      isAdminMemoryRoute: false,
    })

    await user.click(screen.getByTestId('open-admin-token-panel'))
    await user.click(screen.getByRole('button', { name: 'Memory Admin' }))
    await user.click(screen.getByRole('button', { name: 'Export / Import' }))

    expect(onOpenAdminTokenPanel).toHaveBeenCalledTimes(1)
    expect(onToggleAdminMemoryRoute).toHaveBeenCalledTimes(1)
    expect(onToggleProjectTransferRoute).toHaveBeenCalledTimes(1)
  })

  it('hides admin-only controls for non-admin users', () => {
    renderTopNav({
      user: buildUser('analyst'),
      isAdmin: false,
    })

    expect(screen.queryByTestId('open-admin-token-panel')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Memory Admin' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Chat Workspace' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Export / Import' })).toBeInTheDocument()
  })
})
