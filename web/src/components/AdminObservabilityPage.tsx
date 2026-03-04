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

export function AdminObservabilityPage() {
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
        const [metricsPayload, versionPayload] = await Promise.all([
          apiJson<Record<string, unknown>>('/api/v1/metrics'),
          apiJson<VersionPayload>('/api/v1/version'),
        ])
        if (cancelled) {
          return
        }
        setMetrics(metricsPayload)
        setVersion(versionPayload)
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

  return (
    <Grid>
      <GlassPane>
        <PaneHeader>
          <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">Runtime Version</h2>
        </PaneHeader>
        {loading ? <MutedText>Loading runtime metadata...</MutedText> : null}
        {error ? <ErrorText>{error}</ErrorText> : null}
        {version ? <JsonBlock>{JSON.stringify(version, null, 2)}</JsonBlock> : null}
      </GlassPane>

      <GlassPane>
        <PaneHeader>
          <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">Observability Snapshot</h2>
        </PaneHeader>
        {loading ? <MutedText>Loading metrics...</MutedText> : null}
        {error ? <ErrorText>{error}</ErrorText> : null}
        {metrics ? <JsonBlock>{JSON.stringify(metrics, null, 2)}</JsonBlock> : null}
      </GlassPane>
    </Grid>
  )
}
