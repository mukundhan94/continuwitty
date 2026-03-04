import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'

import { MARKETING_ROUTES } from '../constants'

interface AuthGuardProps {
  isAuthenticated: boolean
  children: ReactNode
}

export function AuthGuard({ isAuthenticated, children }: AuthGuardProps) {
  const location = useLocation()

  if (!isAuthenticated) {
    const next = encodeURIComponent(location.pathname + location.search)
    return <Navigate to={`${MARKETING_ROUTES.login}?next=${next}`} replace />
  }

  return <>{children}</>
}
