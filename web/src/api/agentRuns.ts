import { apiJson } from './http'
import type { AgentRunCreateInput, AgentRunResponse, AgentRunResumeInput } from './types'

export function createAgentRun(input: AgentRunCreateInput): Promise<AgentRunResponse> {
  return apiJson<AgentRunResponse>('/api/v1/agent-runs', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function getAgentRun(threadId: string): Promise<AgentRunResponse> {
  return apiJson<AgentRunResponse>(`/api/v1/agent-runs/${encodeURIComponent(threadId)}`, {
    method: 'GET',
  })
}

export function resumeAgentRun(threadId: string, input: AgentRunResumeInput): Promise<AgentRunResponse> {
  return apiJson<AgentRunResponse>(`/api/v1/agent-runs/${encodeURIComponent(threadId)}/resume`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}
