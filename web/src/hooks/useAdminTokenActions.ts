import { useCallback, useEffect, useMemo, useState } from 'react'

import {
  createMcpToken,
  listAvailableMcpProjects,
  listAvailableMcpTools,
  listMcpTokens,
  revokeMcpToken,
} from '../api/mcpTokens'
import type {
  McpTokenCreateRequest,
  McpTokenCreateResponse,
  McpTokenSummary,
} from '../api/types'

interface AdminTokenActionsConfig {
  isAdmin: boolean
  setNotice: (value: string | null) => void
  describeError: (error: unknown) => string
}

interface AdminTokenState {
  adminTokenPanelOpen: boolean
  adminTokensLoading: boolean
  adminTokensCreating: boolean
  adminTokens: McpTokenSummary[]
  adminLatestToken: McpTokenCreateResponse | null
  adminTokenError: string | null
  adminTokenOptionsLoading: boolean
  adminAvailableTools: string[]
  adminAvailableProjects: string[]
  setAdminTokenPanelOpen: (value: boolean) => void
  setAdminTokensLoading: (value: boolean) => void
  setAdminTokensCreating: (value: boolean) => void
  setAdminTokens: (value: McpTokenSummary[]) => void
  setAdminLatestToken: (value: McpTokenCreateResponse | null) => void
  setAdminTokenError: (value: string | null) => void
  setAdminTokenOptionsLoading: (value: boolean) => void
  setAdminAvailableTools: (value: string[]) => void
  setAdminAvailableProjects: (value: string[]) => void
  resetAdminTokenState: () => void
  dispatchers: AdminTokenDispatchers
}

interface AdminTokenDispatchers {
  setAdminTokenPanelOpen: (value: boolean) => void
  setAdminTokensLoading: (value: boolean) => void
  setAdminTokensCreating: (value: boolean) => void
  setAdminTokens: (value: McpTokenSummary[]) => void
  setAdminLatestToken: (value: McpTokenCreateResponse | null) => void
  setAdminTokenError: (value: string | null) => void
  setAdminTokenOptionsLoading: (value: boolean) => void
  setAdminAvailableTools: (value: string[]) => void
  setAdminAvailableProjects: (value: string[]) => void
}

function useAdminTokenDispatchers(
  input: AdminTokenDispatchers,
): AdminTokenDispatchers {
  const {
    setAdminTokenPanelOpen,
    setAdminTokensLoading,
    setAdminTokensCreating,
    setAdminTokens,
    setAdminLatestToken,
    setAdminTokenError,
    setAdminTokenOptionsLoading,
    setAdminAvailableTools,
    setAdminAvailableProjects,
  } = input

  return useMemo(
    () => ({
      setAdminTokenPanelOpen,
      setAdminTokensLoading,
      setAdminTokensCreating,
      setAdminTokens,
      setAdminLatestToken,
      setAdminTokenError,
      setAdminTokenOptionsLoading,
      setAdminAvailableTools,
      setAdminAvailableProjects,
    }),
    [
      setAdminTokenPanelOpen,
      setAdminTokensLoading,
      setAdminTokensCreating,
      setAdminTokens,
      setAdminLatestToken,
      setAdminTokenError,
      setAdminTokenOptionsLoading,
      setAdminAvailableTools,
      setAdminAvailableProjects,
    ],
  )
}

function useAdminTokenState(): AdminTokenState {
  const [adminTokenPanelOpen, setAdminTokenPanelOpen] = useState(false)
  const [adminTokensLoading, setAdminTokensLoading] = useState(false)
  const [adminTokensCreating, setAdminTokensCreating] = useState(false)
  const [adminTokens, setAdminTokens] = useState<McpTokenSummary[]>([])
  const [adminLatestToken, setAdminLatestToken] = useState<McpTokenCreateResponse | null>(null)
  const [adminTokenError, setAdminTokenError] = useState<string | null>(null)
  const [adminTokenOptionsLoading, setAdminTokenOptionsLoading] = useState(false)
  const [adminAvailableTools, setAdminAvailableTools] = useState<string[]>([])
  const [adminAvailableProjects, setAdminAvailableProjects] = useState<string[]>([])

  const dispatchers = useAdminTokenDispatchers({
    setAdminTokenPanelOpen,
    setAdminTokensLoading,
    setAdminTokensCreating,
    setAdminTokens,
    setAdminLatestToken,
    setAdminTokenError,
    setAdminTokenOptionsLoading,
    setAdminAvailableTools,
    setAdminAvailableProjects,
  })

  const resetAdminTokenState = useCallback(() => {
    setAdminTokenPanelOpen(false)
    setAdminTokensLoading(false)
    setAdminTokensCreating(false)
    setAdminTokens([])
    setAdminLatestToken(null)
    setAdminTokenError(null)
    setAdminTokenOptionsLoading(false)
    setAdminAvailableTools([])
    setAdminAvailableProjects([])
  }, [
    setAdminTokenPanelOpen,
    setAdminTokensLoading,
    setAdminTokensCreating,
    setAdminTokens,
    setAdminLatestToken,
    setAdminTokenError,
    setAdminTokenOptionsLoading,
    setAdminAvailableTools,
    setAdminAvailableProjects,
  ])

  return {
    adminTokenPanelOpen,
    adminTokensLoading,
    adminTokensCreating,
    adminTokens,
    adminLatestToken,
    adminTokenError,
    adminTokenOptionsLoading,
    adminAvailableTools,
    adminAvailableProjects,
    setAdminTokenPanelOpen,
    setAdminTokensLoading,
    setAdminTokensCreating,
    setAdminTokens,
    setAdminLatestToken,
    setAdminTokenError,
    setAdminTokenOptionsLoading,
    setAdminAvailableTools,
    setAdminAvailableProjects,
    resetAdminTokenState,
    dispatchers,
  }
}

interface AdminTokenLoadersConfig {
  isAdmin: boolean
  describeError: (error: unknown) => string
  state: Pick<
    AdminTokenState,
    | 'setAdminTokenPanelOpen'
    | 'setAdminTokensLoading'
    | 'setAdminTokens'
    | 'setAdminLatestToken'
    | 'setAdminTokenError'
    | 'setAdminTokenOptionsLoading'
    | 'setAdminAvailableTools'
    | 'setAdminAvailableProjects'
  >
}

function useAdminTokenLoaders(config: AdminTokenLoadersConfig) {
  const { isAdmin, describeError, state } = config
  const loadAdminTokens = useCallback(async () => {
    if (!isAdmin) {
      return
    }
    state.setAdminTokensLoading(true)
    state.setAdminTokenError(null)
    try {
      const tokens = await listMcpTokens()
      state.setAdminTokens(tokens)
    } catch (error) {
      state.setAdminTokenError(describeError(error))
    } finally {
      state.setAdminTokensLoading(false)
    }
  }, [describeError, isAdmin, state])

  const loadAdminTokenOptions = useCallback(async () => {
    if (!isAdmin) {
      return
    }
    state.setAdminTokenOptionsLoading(true)
    try {
      const [tools, projects] = await Promise.all([
        listAvailableMcpTools(),
        listAvailableMcpProjects(),
      ])
      state.setAdminAvailableTools(tools)
      state.setAdminAvailableProjects(projects)
    } catch (error) {
      state.setAdminTokenError(describeError(error))
    } finally {
      state.setAdminTokenOptionsLoading(false)
    }
  }, [describeError, isAdmin, state])

  const handleRefreshAdminTokenPanel = useCallback(async () => {
    await Promise.all([loadAdminTokens(), loadAdminTokenOptions()])
  }, [loadAdminTokenOptions, loadAdminTokens])

  const openAdminTokenPanel = useCallback(async () => {
    if (!isAdmin) {
      return
    }
    state.setAdminTokenPanelOpen(true)
    state.setAdminLatestToken(null)
    await handleRefreshAdminTokenPanel()
  }, [handleRefreshAdminTokenPanel, isAdmin, state])

  const closeAdminTokenPanel = useCallback(() => {
    state.setAdminTokenPanelOpen(false)
  }, [state])

  return {
    loadAdminTokens,
    handleRefreshAdminTokenPanel,
    openAdminTokenPanel,
    closeAdminTokenPanel,
  }
}

interface AdminTokenMutationsConfig {
  isAdmin: boolean
  setNotice: (value: string | null) => void
  describeError: (error: unknown) => string
  loadAdminTokens: () => Promise<void>
  state: Pick<
    AdminTokenState,
    | 'setAdminTokensCreating'
    | 'setAdminLatestToken'
    | 'setAdminTokenError'
  >
}

function useAdminTokenMutations(config: AdminTokenMutationsConfig) {
  const { isAdmin, setNotice, describeError, loadAdminTokens, state } = config
  const handleCreateAdminToken = useCallback(
    async (payload: McpTokenCreateRequest) => {
      if (!isAdmin) {
        return
      }
      state.setAdminTokensCreating(true)
      state.setAdminTokenError(null)
      try {
        const created = await createMcpToken(payload)
        state.setAdminLatestToken(created)
        setNotice(`Created MCP token ${created.name}.`)
        await loadAdminTokens()
      } catch (error) {
        state.setAdminTokenError(describeError(error))
      } finally {
        state.setAdminTokensCreating(false)
      }
    },
    [describeError, isAdmin, loadAdminTokens, setNotice, state],
  )

  const handleRevokeAdminToken = useCallback(
    async (tokenId: string) => {
      if (!isAdmin) {
        return
      }
      state.setAdminTokenError(null)
      try {
        await revokeMcpToken(tokenId)
        setNotice(`Revoked MCP token ${tokenId}.`)
        await loadAdminTokens()
      } catch (error) {
        state.setAdminTokenError(describeError(error))
      }
    },
    [describeError, isAdmin, loadAdminTokens, setNotice, state],
  )

  return {
    handleCreateAdminToken,
    handleRevokeAdminToken,
  }
}

export function useAdminTokenActions(config: AdminTokenActionsConfig) {
  const state = useAdminTokenState()
  const { dispatchers, resetAdminTokenState } = state
  const { isAdmin, setNotice, describeError } = config
  const loaders = useAdminTokenLoaders({
    isAdmin,
    describeError,
    state: dispatchers,
  })
  const mutations = useAdminTokenMutations({
    isAdmin,
    setNotice,
    describeError,
    loadAdminTokens: loaders.loadAdminTokens,
    state: dispatchers,
  })

  useEffect(() => {
    if (!isAdmin) {
      resetAdminTokenState()
    }
  }, [isAdmin, resetAdminTokenState])

  return {
    adminTokenPanelOpen: state.adminTokenPanelOpen,
    adminTokensLoading: state.adminTokensLoading,
    adminTokensCreating: state.adminTokensCreating,
    adminTokens: state.adminTokens,
    adminLatestToken: state.adminLatestToken,
    adminTokenError: state.adminTokenError,
    adminTokenOptionsLoading: state.adminTokenOptionsLoading,
    adminAvailableTools: state.adminAvailableTools,
    adminAvailableProjects: state.adminAvailableProjects,
    openAdminTokenPanel: loaders.openAdminTokenPanel,
    closeAdminTokenPanel: loaders.closeAdminTokenPanel,
    handleRefreshAdminTokenPanel: loaders.handleRefreshAdminTokenPanel,
    handleCreateAdminToken: mutations.handleCreateAdminToken,
    handleRevokeAdminToken: mutations.handleRevokeAdminToken,
    resetAdminTokenState,
  }
}
