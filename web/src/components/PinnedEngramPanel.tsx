import type { EngramSummary } from '../api/types'

interface PinnedEngramPanelProps {
  selectedSessionId: string | null
  pinnedEngrams: EngramSummary[]
  availableEngrams: EngramSummary[]
  search: string
  loading: boolean
  onSearchChange: (value: string) => void
  onRefresh: () => Promise<void>
  onPin: (engramId: string) => Promise<void>
  onUnpin: (engramId: string) => Promise<void>
  onCopyId: (engramId: string) => Promise<void>
}

export function PinnedEngramPanel({
  selectedSessionId,
  pinnedEngrams,
  availableEngrams,
  search,
  loading,
  onSearchChange,
  onRefresh,
  onPin,
  onUnpin,
  onCopyId,
}: PinnedEngramPanelProps) {
  const pinnedIds = new Set(pinnedEngrams.map((item) => item.engram_id))
  const normalized = search.trim().toLowerCase()

  const filteredAvailable = availableEngrams.filter((item) => {
    if (!normalized) {
      return true
    }
    return (
      item.title.toLowerCase().includes(normalized) ||
      item.abstract.toLowerCase().includes(normalized) ||
      item.engram_id.toLowerCase().includes(normalized)
    )
  })

  return (
    <aside className="pane pane-engrams">
      <div className="pane-header">
        <h2>Pinned Engrams</h2>
        <button onClick={onRefresh} disabled={!selectedSessionId || loading}>
          Refresh
        </button>
      </div>

      {!selectedSessionId ? <p className="muted">Select a session to manage engrams.</p> : null}

      <div className="engram-section">
        <h3>Session Pins</h3>
        {pinnedEngrams.length === 0 ? <p className="muted">No pinned engrams yet.</p> : null}
        {pinnedEngrams.map((engram) => (
          <article key={engram.engram_id} className="engram-card">
            <p className="engram-title">{engram.title}</p>
            <p className="engram-abstract">{engram.abstract}</p>
            <div className="engram-actions">
              <button onClick={() => onCopyId(engram.engram_id)}>Copy ID</button>
              <button onClick={() => onUnpin(engram.engram_id)}>Unpin</button>
            </div>
          </article>
        ))}
      </div>

      <div className="engram-section">
        <h3>Search Project Engrams</h3>
        <input
          value={search}
          onChange={(event) => onSearchChange(event.target.value)}
          placeholder="Search title, abstract, or engram id"
        />
        {filteredAvailable.map((engram) => (
          <article key={engram.engram_id} className="engram-card">
            <p className="engram-title">{engram.title}</p>
            <p className="engram-abstract">{engram.abstract}</p>
            <div className="engram-actions">
              <button onClick={() => onCopyId(engram.engram_id)}>Copy ID</button>
              <button
                onClick={() => onPin(engram.engram_id)}
                disabled={!selectedSessionId || pinnedIds.has(engram.engram_id)}
              >
                {pinnedIds.has(engram.engram_id) ? 'Pinned' : 'Pin to Session'}
              </button>
            </div>
          </article>
        ))}
      </div>
    </aside>
  )
}
