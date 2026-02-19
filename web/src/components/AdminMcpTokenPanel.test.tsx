import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ThemeProvider } from 'styled-components'
import { describe, expect, it, vi } from 'vitest'

import type { McpTokenCreateResponse, McpTokenSummary } from '../api/types'
import { lightTheme } from '../styles/theme'
import { AdminMcpTokenPanel } from './AdminMcpTokenPanel'

function buildToken(overrides: Partial<McpTokenSummary> = {}): McpTokenSummary {
  return {
    token_id: 'token-1',
    name: 'default token',
    scope: 'read',
    allowed_tools: [],
    allowed_project_ids: ['engram-vault'],
    token_secret_hint: 'abc123...z9yx',
    expires_at: '2026-05-01T00:00:00Z',
    last_used_at: null,
    revoked_at: null,
    created_at: '2026-02-01T00:00:00Z',
    is_active: true,
    ...overrides,
  }
}

function renderPanel(overrides: {
  isOpen?: boolean
  tokens?: McpTokenSummary[]
  latestToken?: McpTokenCreateResponse | null
  availableTools?: string[]
  availableProjects?: string[]
  optionsLoading?: boolean
  onCreate?: (payload: {
    name: string
    scope: 'read' | 'write'
    allowed_tools: string[]
    allowed_project_ids: string[]
    expires_in_days: number
  }) => Promise<void>
  onRevoke?: (tokenId: string) => Promise<void>
} = {}) {
  const onRefresh = vi.fn(async () => {})
  const onCreate = overrides.onCreate ?? vi.fn(async () => {})
  const onRevoke = overrides.onRevoke ?? vi.fn(async () => {})
  const onClose = vi.fn()

  render(
    <ThemeProvider theme={lightTheme}>
      <AdminMcpTokenPanel
        isOpen={overrides.isOpen ?? true}
        loading={false}
        optionsLoading={overrides.optionsLoading ?? false}
        creating={false}
        tokens={overrides.tokens ?? [buildToken()]}
        latestToken={overrides.latestToken ?? null}
        error={null}
        availableTools={overrides.availableTools ?? ['chat.list_sessions', 'engram.query']}
        availableProjects={overrides.availableProjects ?? ['engram-vault', 'project-a']}
        onClose={onClose}
        onRefresh={onRefresh}
        onCreate={onCreate}
        onRevoke={onRevoke}
      />
    </ThemeProvider>,
  )

  return { onRefresh, onCreate, onRevoke, onClose }
}

describe('AdminMcpTokenPanel', () => {
  it('does not render when closed', () => {
    renderPanel({ isOpen: false })
    expect(screen.queryByRole('dialog', { name: /admin mcp tokens/i })).not.toBeInTheDocument()
  })

  it('submits token creation payload from selected tool/project chips', async () => {
    const user = userEvent.setup()
    const onCreate = vi.fn(async () => {})
    renderPanel({ onCreate })

    await user.type(screen.getByLabelText(/token name/i), ' LibreChat Token ')
    await user.selectOptions(screen.getByLabelText(/scope/i), 'write')
    fireEvent.change(screen.getByLabelText(/expiry days/i), { target: { value: '45' } })

    await user.selectOptions(screen.getByLabelText(/tool options/i), 'engram.query')
    await user.click(screen.getByRole('button', { name: /add tool/i }))

    await user.selectOptions(screen.getByLabelText(/project options/i), 'project-a')
    await user.click(screen.getByRole('button', { name: /add project/i }))

    await user.click(screen.getByRole('button', { name: /create token/i }))

    await waitFor(() => {
      expect(onCreate).toHaveBeenCalledWith({
        name: 'LibreChat Token',
        scope: 'write',
        expires_in_days: 45,
        allowed_tools: ['engram.query'],
        allowed_project_ids: ['project-a'],
      })
    })
  })

  it('shows one-time created token and triggers revoke for active token', async () => {
    const user = userEvent.setup()
    const onRevoke = vi.fn(async () => {})
    const latestToken: McpTokenCreateResponse = {
      ...buildToken({ token_id: 'token-created', name: 'new token', scope: 'write' }),
      token: 'engram_mcp_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa_secret',
      allowed_tools: ['engram.query'],
      allowed_project_ids: ['engram-vault'],
      is_active: true,
    }

    renderPanel({
      latestToken,
      onRevoke,
      tokens: [
        buildToken({ token_id: 'token-active', name: 'active token' }),
        buildToken({ token_id: 'token-revoked', name: 'revoked token', revoked_at: '2026-02-01T01:00:00Z', is_active: false }),
      ],
    })

    expect(screen.getByText(/new token created/i)).toBeInTheDocument()
    expect(screen.getByText(/engram_mcp_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa_secret/i)).toBeInTheDocument()

    await user.click(screen.getAllByRole('button', { name: /revoke/i })[0])
    expect(onRevoke).toHaveBeenCalledWith('token-active')

    expect(screen.getByRole('button', { name: /^revoked$/i })).toBeDisabled()
  })
})
