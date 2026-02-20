import { useCallback, type Dispatch, type SetStateAction } from 'react'

import { continueSession, saveSessionAsEngram } from '../api/chat'
import type { SaveEngramPayload } from '../api/chatTypes'
import type { ChatSession } from '../api/types'

interface SessionActionsConfig {
  selectedSessionId: string | null
  refreshFromSession: (sessionId: string) => Promise<void>
  setSessions: Dispatch<SetStateAction<ChatSession[]>>
  setSelectedSessionId: (value: string | null) => void
  setSaveModalOpen: (value: boolean) => void
  setSaveSubmitting: (value: boolean) => void
  setNotice: (value: string | null) => void
  setChatError: (value: string | null) => void
  describeError: (error: unknown) => string
  copyText?: (value: string) => Promise<void>
}

function defaultCopyText(value: string): Promise<void> {
  return navigator.clipboard.writeText(value)
}

export function useSessionActions(config: SessionActionsConfig) {
  const {
    selectedSessionId,
    refreshFromSession,
    setSessions,
    setSelectedSessionId,
    setSaveModalOpen,
    setSaveSubmitting,
    setNotice,
    setChatError,
    describeError,
    copyText = defaultCopyText,
  } = config

  const handleCopyEngramId = useCallback(
    async (engramId: string) => {
      try {
        await copyText(engramId)
        setNotice(`Copied engram id: ${engramId}`)
      } catch {
        setChatError('Clipboard access failed. Copy manually from the card.')
      }
    },
    [copyText, setChatError, setNotice],
  )

  const handleRefreshEngrams = useCallback(async () => {
    if (!selectedSessionId) {
      return
    }
    await refreshFromSession(selectedSessionId)
  }, [refreshFromSession, selectedSessionId])

  const handleContinueSession = useCallback(async () => {
    if (!selectedSessionId) {
      return
    }
    try {
      const continued = await continueSession(selectedSessionId)
      setSessions((current) => [continued.session, ...current])
      setSelectedSessionId(continued.session.session_id)
      setNotice(`Created continuation with ${continued.carried_engram_ids.length} carried engrams.`)
    } catch (error) {
      setChatError(describeError(error))
    }
  }, [describeError, selectedSessionId, setChatError, setNotice, setSelectedSessionId, setSessions])

  const handleSaveEngram = useCallback(
    async (payload: SaveEngramPayload) => {
      if (!selectedSessionId) {
        return
      }
      setSaveSubmitting(true)
      try {
        const created = await saveSessionAsEngram(selectedSessionId, payload)
        setNotice(`Saved session as engram ${created.engram_id}`)
        setSaveModalOpen(false)
        await refreshFromSession(selectedSessionId)
      } catch (error) {
        setChatError(describeError(error))
      } finally {
        setSaveSubmitting(false)
      }
    },
    [describeError, refreshFromSession, selectedSessionId, setChatError, setNotice, setSaveModalOpen, setSaveSubmitting],
  )

  return {
    handleCopyEngramId,
    handleRefreshEngrams,
    handleContinueSession,
    handleSaveEngram,
  }
}
