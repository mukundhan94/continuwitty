import { useCallback, type Dispatch, type SetStateAction } from 'react'

import { createChatSession } from '../api/chat'
import { setDefaultProject } from '../api/projects'
import type { ChatSession, ChatSessionFormPayload } from '../api/types'

interface WorkspaceActionsConfig {
  projectId: string
  normalizeProjectId: (value: string) => string
  describeError: (error: unknown) => string
  setCreatingSession: (value: boolean) => void
  setChatError: (value: string | null) => void
  setSessions: Dispatch<SetStateAction<ChatSession[]>>
  setSelectedSessionId: (value: string | null) => void
  setComposerText: (value: string) => void
  setSettingDefaultProject: (value: boolean) => void
  setDefaultProjectId: (value: string | null) => void
  setNotice: (value: string | null) => void
}

export function useWorkspaceActions(config: WorkspaceActionsConfig) {
  const {
    projectId,
    normalizeProjectId,
    describeError,
    setCreatingSession,
    setChatError,
    setSessions,
    setSelectedSessionId,
    setComposerText,
    setSettingDefaultProject,
    setDefaultProjectId,
    setNotice,
  } = config

  const handleCreateSession = useCallback(
    async (payload: ChatSessionFormPayload) => {
      setCreatingSession(true)
      setChatError(null)
      try {
        const created = await createChatSession(payload)
        setSessions((current) => [created, ...current])
        setSelectedSessionId(created.session_id)
        setComposerText('')
      } catch (error) {
        setChatError(describeError(error))
      } finally {
        setCreatingSession(false)
      }
    },
    [describeError, setChatError, setComposerText, setCreatingSession, setSelectedSessionId, setSessions],
  )

  const handleSetDefaultProject = useCallback(async () => {
    const normalized = normalizeProjectId(projectId)
    setSettingDefaultProject(true)
    setChatError(null)
    try {
      const updated = await setDefaultProject({ project_id: normalized })
      setDefaultProjectId(updated.default_project_id)
      setNotice(`Default project set to ${updated.default_project_id}`)
    } catch (error) {
      setChatError(describeError(error))
    } finally {
      setSettingDefaultProject(false)
    }
  }, [
    describeError,
    normalizeProjectId,
    projectId,
    setChatError,
    setDefaultProjectId,
    setNotice,
    setSettingDefaultProject,
  ])

  return {
    handleCreateSession,
    handleSetDefaultProject,
  }
}
