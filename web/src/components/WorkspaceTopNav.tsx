import type { UserProfile } from '../api/types'
import type { ThemeMode } from '../styles/theme'
import { TopNavShell, TopNavTitleBlock, TopNavUserBlock } from '../styles/primitives'

interface WorkspaceTopNavProps {
  user: UserProfile
  isAdmin: boolean
  isAdminMemoryRoute: boolean
  isProjectTransferRoute: boolean
  mode: ThemeMode
  onOpenAdminTokenPanel: () => void | Promise<void>
  onToggleAdminMemoryRoute: () => void
  onToggleProjectTransferRoute: () => void
  onToggleTheme: () => void
  onLogout: () => void
}

export function WorkspaceTopNav({
  user,
  isAdmin,
  isAdminMemoryRoute,
  isProjectTransferRoute,
  mode,
  onOpenAdminTokenPanel,
  onToggleAdminMemoryRoute,
  onToggleProjectTransferRoute,
  onToggleTheme,
  onLogout,
}: WorkspaceTopNavProps) {
  return (
    <TopNavShell>
      <TopNavTitleBlock>
        <h1 className="font-display text-lg font-semibold tracking-tight text-ink">Memory Continuity Workbench</h1>
      </TopNavTitleBlock>

      <TopNavUserBlock>
        <p className="text-sm text-inkMuted">
          {user.username} · {user.role}
        </p>
        {isAdmin ? (
          <button type="button" data-testid="open-admin-token-panel" onClick={() => void onOpenAdminTokenPanel()}>
            MCP Tokens
          </button>
        ) : null}
        {isAdmin ? (
          <button type="button" onClick={onToggleAdminMemoryRoute}>
            {isAdminMemoryRoute ? 'Chat Workspace' : 'Memory Admin'}
          </button>
        ) : null}
        <button
          type="button"
          data-testid="open-project-transfer"
          onClick={onToggleProjectTransferRoute}
        >
          {isProjectTransferRoute ? 'Chat Workspace' : 'Export / Import'}
        </button>
        <button type="button" onClick={onToggleTheme}>
          {mode === 'dark' ? 'Light Theme' : 'Dark Theme'}
        </button>
        <button type="button" onClick={onLogout}>
          Logout
        </button>
      </TopNavUserBlock>
    </TopNavShell>
  )
}
