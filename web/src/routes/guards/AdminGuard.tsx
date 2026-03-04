import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'

import { APP_ROUTES } from '../constants'

interface AdminGuardProps {
  isAdmin: boolean
  children: ReactNode
}

export function AdminGuard({ isAdmin, children }: AdminGuardProps) {
  if (!isAdmin) {
    return <Navigate to={APP_ROUTES.workspace} replace />
  }

  return <>{children}</>
}
