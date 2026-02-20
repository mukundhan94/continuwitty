import { useCallback } from 'react'

import { getSessionProfile, loginWithPassword, logoutCurrentUser } from '../api/auth'
import type { UserProfile } from '../api/types'

interface AuthActionsConfig {
  projectId: string
  loadDefaultProject: () => Promise<void>
  loadSessions: (projectId: string, preferredSessionId: string | null) => Promise<void>
  loadProjectDocuments: (projectId: string) => Promise<void>
  setUser: (value: UserProfile | null) => void
  setAuthSubmitting: (value: boolean) => void
  setAuthError: (value: string | null) => void
  describeError: (error: unknown) => string
  resetWorkspaceState: () => void
}

export function useAuthActions(config: AuthActionsConfig) {
  const {
    projectId,
    loadDefaultProject,
    loadSessions,
    loadProjectDocuments,
    setUser,
    setAuthSubmitting,
    setAuthError,
    describeError,
    resetWorkspaceState,
  } = config

  const hydrateWorkspaceAfterLogin = useCallback(async () => {
    const profile = await getSessionProfile()
    setUser(profile)
    await Promise.all([
      loadDefaultProject(),
      loadSessions(projectId, null),
      loadProjectDocuments(projectId),
    ])
  }, [
    loadDefaultProject,
    loadProjectDocuments,
    loadSessions,
    projectId,
    setUser,
  ])

  const handleLogin = useCallback(
    async (username: string, password: string) => {
      setAuthSubmitting(true)
      setAuthError(null)
      try {
        await loginWithPassword(username, password)
        await hydrateWorkspaceAfterLogin()
      } catch (error) {
        setAuthError(describeError(error))
      } finally {
        setAuthSubmitting(false)
      }
    },
    [describeError, hydrateWorkspaceAfterLogin, setAuthError, setAuthSubmitting],
  )

  const handleLogout = useCallback(async () => {
    try {
      await logoutCurrentUser()
    } catch {
      // Ignore local logout errors and reset state.
    }
    resetWorkspaceState()
  }, [resetWorkspaceState])

  return {
    handleLogin,
    handleLogout,
  }
}
