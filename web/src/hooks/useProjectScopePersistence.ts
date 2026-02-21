import { useEffect } from 'react'

interface ProjectScopePersistenceConfig {
  projectId: string
  normalizeProjectId: (value: string) => string
  storageKey: string
}

export function useProjectScopePersistence(config: ProjectScopePersistenceConfig) {
  const {
    projectId,
    normalizeProjectId,
    storageKey,
  } = config

  useEffect(() => {
    if (typeof window === 'undefined') {
      return
    }
    // Keep the current project scope sticky across hard refreshes.
    window.localStorage.setItem(storageKey, normalizeProjectId(projectId))
  }, [normalizeProjectId, projectId, storageKey])
}
