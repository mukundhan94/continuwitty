import { useEffect, useState } from 'react'
import styled from 'styled-components'

import { apiJson } from '../api/http'
import { describeError } from '../utils/errors'
import { ErrorText, GlassPane, MutedText, PaneHeader } from '../styles/primitives'

const Grid = styled.div`
  display: grid;
  gap: 0.8rem;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);

  @media (max-width: 1180px) {
    grid-template-columns: 1fr;
  }
`

const JsonBlock = styled.pre`
  margin: 0;
  max-height: 55vh;
  overflow: auto;
  border: 1px dashed var(--color-line);
  border-radius: 12px;
  background: var(--surface-raised);
  padding: 0.7rem;
  font-size: 0.75rem;
`

interface VersionPayload {
  semantic_version: string
  release: string
  commit_id: string
  chat_prompt_policy_version: string
  mcp_tool_policy_version: string
  eval_suite_version: string
}

interface ObservabilityPayload {
  metrics: Record<string, unknown>
  version: VersionPayload
}

interface ObservabilityDataState {
  loading: boolean
  error: string | null
  metrics: Record<string, unknown> | null
  version: VersionPayload | null
}

interface JsonPaneProps {
  title: string
  loadingLabel: string
  loading: boolean
  error: string | null
  payload: Record<string, unknown> | VersionPayload | null
}

async function loadObservabilityPayload(): Promise<ObservabilityPayload> {
  const [metrics, version] = await Promise.all([
    apiJson<Record<string, unknown>>('/api/v1/metrics'),
    apiJson<VersionPayload>('/api/v1/version'),
  ])
  return { metrics, version }
}

function useObservabilityData(): ObservabilityDataState {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [metrics, setMetrics] = useState<Record<string, unknown> | null>(null)
  const [version, setVersion] = useState<VersionPayload | null>(null)

  useEffect(() => {
    let cancelled = false

    const load = async () => {
      setLoading(true)
      setError(null)
      try {
        const payload = await loadObservabilityPayload()
        if (!cancelled) {
          setMetrics(payload.metrics)
          setVersion(payload.version)
        }
      } catch (loadError) {
        if (!cancelled) {
          setError(describeError(loadError))
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  }, [])

  return { loading, error, metrics, version }
}

function JsonPane({ title, loadingLabel, loading, error, payload }: JsonPaneProps) {
  return (
    <GlassPane>
      <PaneHeader>
        <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">{title}</h2>
      </PaneHeader>
      {loading ? <MutedText>{loadingLabel}</MutedText> : null}
      {error ? <ErrorText>{error}</ErrorText> : null}
      {payload ? <JsonBlock>{JSON.stringify(payload, null, 2)}</JsonBlock> : null}
    </GlassPane>
  )
}

export function AdminObservabilityPage() {
  const { loading, error, metrics, version } = useObservabilityData()

  return (
    <Grid>
      <JsonPane
        title="Runtime Version"
        loadingLabel="Loading runtime metadata..."
        loading={loading}
        error={error}
        payload={version}
      />
      <JsonPane
        title="Observability Snapshot"
        loadingLabel="Loading metrics..."
        loading={loading}
        error={error}
        payload={metrics}
      />
    </Grid>
  )
}
