import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { ThemeProvider } from 'styled-components'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { lightTheme } from '../styles/theme'
import { AgentRunsPage } from './AgentRunsPage'

const agentRunApiMocks = vi.hoisted(() => ({
  createAgentRun: vi.fn(),
  getAgentRun: vi.fn(),
  resumeAgentRun: vi.fn(),
}))

vi.mock('../api/agentRuns', () => ({
  createAgentRun: agentRunApiMocks.createAgentRun,
  getAgentRun: agentRunApiMocks.getAgentRun,
  resumeAgentRun: agentRunApiMocks.resumeAgentRun,
}))

const threadResponse = {
  thread_id: 'thread-1',
  status: 'completed',
  engram_id: 'engram-1',
  snapshot_engram_ids: ['snapshot-1'],
  state: {
    project_id: 'engram-vault',
    thread_id: 'thread-1',
    objective: 'Track a durable agent run',
    notes: ['First note'],
    assumptions: ['Initial assumption'],
    tags: ['research'],
    keywords: ['continuity'],
    sources: [],
    synthesis_title: 'Thread synthesis',
    synthesis_abstract: 'Short abstract',
    synthesis_markdown: '## Summary\nCheckpointed successfully.',
    decisions: [{ decision: 'Keep snapshots enabled', rationale: 'Maintains recovery points.' }],
    open_questions: ['What should happen next?'],
    status: 'completed',
    auto_persist_engram: true,
    engram_id: 'engram-1',
    snapshot_enabled: true,
    snapshot_every_n_notes: 3,
    snapshot_count: 1,
    snapshot_engram_ids: ['snapshot-1'],
  },
}

function renderAgentRunsPage(initialThreadId: string | null = null) {
  const onNotice = vi.fn()

  function Harness() {
    const [projectId, setProjectId] = useState('engram-vault')
    const [selectedThreadId, setSelectedThreadId] = useState<string | null>(initialThreadId)

    return (
      <AgentRunsPage
        projectId={projectId}
        onProjectChange={setProjectId}
        selectedThreadId={selectedThreadId}
        onThreadSelect={setSelectedThreadId}
        onNotice={onNotice}
      />
    )
  }

  render(
    <ThemeProvider theme={lightTheme}>
      <Harness />
    </ThemeProvider>,
  )

  return { onNotice }
}

describe('AgentRunsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    window.localStorage.clear()
    agentRunApiMocks.createAgentRun.mockResolvedValue(threadResponse)
    agentRunApiMocks.getAgentRun.mockResolvedValue(threadResponse)
    agentRunApiMocks.resumeAgentRun.mockResolvedValue({
      ...threadResponse,
      state: {
        ...threadResponse.state,
        notes: [...threadResponse.state.notes, 'Next step'],
      },
    })
  })

  it('creates a durable run and remembers the thread', async () => {
    const user = userEvent.setup()
    const { onNotice } = renderAgentRunsPage()

    await user.clear(screen.getByTestId('agent-run-thread-id'))
    await user.type(screen.getByTestId('agent-run-thread-id'), 'thread-1')
    await user.clear(screen.getByTestId('agent-run-objective'))
    await user.type(screen.getByTestId('agent-run-objective'), 'Track a durable agent run')
    await user.type(screen.getByTestId('agent-run-notes'), 'First note')
    await user.click(screen.getByRole('button', { name: 'Start Durable Run' }))

    await waitFor(() => {
      expect(agentRunApiMocks.createAgentRun).toHaveBeenCalledWith(
        expect.objectContaining({
          project_id: 'engram-vault',
          thread_id: 'thread-1',
          objective: 'Track a durable agent run',
          notes: ['First note'],
        }),
      )
    })

    await waitFor(() => {
      expect(onNotice).toHaveBeenCalledWith('Agent run thread-1 is ready.')
    })

    expect(await screen.findByText('Short abstract')).toBeInTheDocument()
    expect(screen.getByTestId('agent-run-lookup')).toHaveValue('thread-1')
  })

  it('loads and resumes an existing thread', async () => {
    const user = userEvent.setup()
    const { onNotice } = renderAgentRunsPage('thread-1')

    expect(await screen.findByText('Short abstract')).toBeInTheDocument()

    await user.type(screen.getByTestId('agent-run-resume-notes'), 'Next step')
    await user.selectOptions(screen.getByTestId('agent-run-resume-auto-persist'), 'disabled')
    await user.click(screen.getByRole('button', { name: 'Resume Thread' }))

    await waitFor(() => {
      expect(agentRunApiMocks.resumeAgentRun).toHaveBeenCalledWith(
        'thread-1',
        expect.objectContaining({
          notes: ['Next step'],
          auto_persist_engram: false,
        }),
      )
    })

    expect(onNotice).toHaveBeenCalledWith('Agent run thread-1 resumed.')
  })
})
