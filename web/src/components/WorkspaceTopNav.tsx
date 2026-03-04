import { NavLink } from 'react-router-dom'
import styled from 'styled-components'

import type { UserProfile } from '../api/types'
import { MemoryStrandMark } from './MemoryStrandMark'
import type { ThemeMode } from '../styles/theme'
import { APP_ROUTES } from '../routes/constants'
import type { BreadcrumbSpec } from '../routes/types'
import { TopNavShell, TopNavTitleBlock, TopNavUserBlock } from '../styles/primitives'

const Brand = styled.div`
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;

  h1 {
    margin: 0;
    font-family: var(--font-display);
    font-size: 1.1rem;
    letter-spacing: 0.01em;
    color: var(--color-ink);
  }

  p {
    margin: 0;
    font-size: 0.72rem;
    text-transform: uppercase;
    letter-spacing: 0.22em;
    color: var(--color-ink-muted);
  }
`

const BrandAccent = styled.span`
  color: var(--color-accent);
`

const BrandLabel = styled.div`
  display: grid;
  gap: 0.08rem;
`

const BreadcrumbRow = styled.div`
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.28rem;
  font-size: 0.74rem;
  color: var(--color-ink-muted);

  a {
    color: var(--color-ink-muted);
    text-decoration: none;
  }
`

const NavStrip = styled.nav`
  display: flex;
  flex-wrap: wrap;
  gap: 0.3rem;

  a {
    border-radius: 9999px;
    border: 1px solid transparent;
    padding: 0.32rem 0.6rem;
    text-decoration: none;
    font-size: 0.75rem;
    color: var(--color-ink-muted);
    letter-spacing: 0.02em;
    transition: all 200ms ease;
  }

  a:hover,
  a.active {
    color: var(--color-ink);
    border-color: var(--session-active-border);
    background: var(--session-active-bg);
  }
`

interface WorkspaceTopNavProps {
  user: UserProfile
  isAdmin: boolean
  mode: ThemeMode
  breadcrumbs: BreadcrumbSpec[]
  onToggleTheme: () => void
  onLogout: () => void
}

export function WorkspaceTopNav({
  user,
  isAdmin,
  mode,
  breadcrumbs,
  onToggleTheme,
  onLogout,
}: WorkspaceTopNavProps) {
  return (
    <TopNavShell>
      <TopNavTitleBlock>
        <Brand>
          <MemoryStrandMark size="sm" />
          <BrandLabel>
            <p>Intelligence that flows</p>
            <h1>
              Continu<BrandAccent>Witty</BrandAccent> Memory Console
            </h1>
          </BrandLabel>
        </Brand>
        <BreadcrumbRow>
          {breadcrumbs.map((item, index) => (
            <span key={`${item.label}-${index}`}>
              {item.to ? <NavLink to={item.to}>{item.label}</NavLink> : item.label}
              {index < breadcrumbs.length - 1 ? ' / ' : ''}
            </span>
          ))}
        </BreadcrumbRow>
      </TopNavTitleBlock>

      <TopNavUserBlock>
        <NavStrip>
          <NavLink to={APP_ROUTES.workspace}>Workspace</NavLink>
          <NavLink to={APP_ROUTES.sessions}>Sessions</NavLink>
          <NavLink to={APP_ROUTES.engrams}>Engrams</NavLink>
          <NavLink to={APP_ROUTES.documents}>Documents</NavLink>
          <NavLink to={APP_ROUTES.projects}>Projects</NavLink>
          <NavLink to={APP_ROUTES.transferExport}>Transfer</NavLink>
          {isAdmin ? <NavLink to={APP_ROUTES.adminSessions}>Admin</NavLink> : null}
        </NavStrip>
        <p className="text-sm text-inkMuted">
          {user.username} · {user.role}
        </p>
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
