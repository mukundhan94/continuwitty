import type { FormEvent, KeyboardEvent } from 'react'
import type { Components } from 'react-markdown'
import ReactMarkdown from 'react-markdown'
import remarkBreaks from 'remark-breaks'
import remarkGfm from 'remark-gfm'
import styled from 'styled-components'

import type {
  ChatDebugTrace,
  ChatMessage,
  ChatSession,
  ChatSourceReference,
  EngramTracePath,
  ChatTimelineEvent,
} from '../api/types'
import {
  ChatMessageBubble,
  ErrorText,
  GlassPane,
  MessageRole,
  MessageText,
  MutedText,
  PaneHeader,
  SectionDivider,
  SourceStrip,
  SourceTitle,
  TranscriptRail,
} from '../styles/primitives'

const HeaderBlock = styled.div`
  display: grid;
  gap: 0.2rem;
`

const ActionRow = styled.div`
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;

  button {
    border: 1px solid var(--color-line);
    background: color-mix(in srgb, var(--surface-raised) 90%, transparent);
    color: var(--color-ink);
    box-shadow: none;
    padding: 0.38rem 0.72rem;
    font-size: 0.76rem;
    letter-spacing: 0.01em;
  }

  button:hover {
    transform: none;
    filter: none;
    box-shadow: none;
    border-color: var(--session-active-border);
    background: var(--session-active-bg);
  }
`

const ComposerActions = styled.div`
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  flex-wrap: wrap;
`

const MinimalButton = styled.button`
  border: 1px solid var(--color-line);
  background: color-mix(in srgb, var(--surface-raised) 90%, transparent);
  color: var(--color-ink);
  box-shadow: none;
  padding: 0.42rem 0.8rem;
  font-size: 0.76rem;
  letter-spacing: 0.01em;

  &:hover {
    transform: none;
    filter: none;
    box-shadow: none;
    border-color: var(--session-active-border);
    background: var(--session-active-bg);
  }
`

const PrimaryMinimalButton = styled(MinimalButton)`
  border-color: var(--session-active-border);
  background: color-mix(in srgb, var(--session-active-bg) 72%, var(--surface-raised));
`

const RecallControls = styled.section`
  display: grid;
  gap: 0.5rem;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  align-items: stretch;
`

const RecallCard = styled.div`
  border: 1px solid color-mix(in srgb, var(--color-line) 75%, transparent);
  border-radius: 12px;
  background: color-mix(in srgb, var(--surface-raised) 66%, transparent);
  padding: 0.5rem 0.6rem;
  display: grid;
  gap: 0.32rem;
  align-content: start;
`

const RecallHeader = styled.div`
  display: flex;
  align-items: center;
  gap: 0.35rem;
`

const RecallLabel = styled.span`
  font-size: 0.74rem;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--color-ink-muted);
  font-weight: 700;
`

const RecallHelp = styled.span`
  position: relative;
  display: inline-flex;
  align-items: center;
`

const RecallHelpTrigger = styled.button`
  width: 1rem;
  height: 1rem;
  min-width: 1rem;
  border-radius: 50%;
  border: 1px solid var(--color-line);
  background: color-mix(in srgb, var(--surface-raised) 92%, transparent);
  color: var(--color-ink-muted);
  box-shadow: none;
  padding: 0;
  font-size: 0.78rem;
  font-weight: 800;
  line-height: 1;
  cursor: help;

  &:hover {
    transform: none;
    filter: none;
    box-shadow: none;
    border-color: var(--session-active-border);
    background: var(--session-active-bg);
    color: var(--color-ink);
  }
`

const RecallHelpText = styled.span`
  position: absolute;
  left: 0;
  bottom: calc(100% + 0.35rem);
  width: min(18rem, 72vw);
  border: 1px solid var(--surface-glass-border);
  border-radius: 10px;
  background: color-mix(in srgb, var(--surface-glass) 97%, transparent);
  box-shadow: var(--shadow-panel);
  padding: 0.45rem 0.55rem;
  font-size: 0.72rem;
  line-height: 1.35;
  color: var(--color-ink);
  opacity: 0;
  transform: translateY(4px);
  pointer-events: none;
  transition:
    opacity 160ms ease,
    transform 160ms ease;
  z-index: 30;
`

const RecallHint = styled.p`
  margin: 0;
  font-size: 0.72rem;
  line-height: 1.35;
  color: var(--color-ink-muted);
`

const RecallHelpContainer = styled.div`
  display: inline-flex;
  align-items: center;

  &:hover ${RecallHelpText},
  &:focus-within ${RecallHelpText} {
    opacity: 1;
    transform: translateY(0);
  }
`

const RecallCheckbox = styled.label`
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.78rem;
  color: var(--color-ink);
  font-weight: 700;
  text-transform: none;
  letter-spacing: 0;

  input {
    width: 0.95rem;
    height: 0.95rem;
  }
`

const RecallNumberControl = styled.label`
  display: grid;
  gap: 0.3rem;
  font-size: 0.76rem;
  color: var(--color-ink);
  text-transform: none;
  letter-spacing: 0;

  input {
    height: 2.1rem;
    padding: 0.35rem 0.55rem;
    border-color: color-mix(in srgb, var(--color-line) 75%, transparent);
    background: color-mix(in srgb, var(--color-input-bg) 75%, var(--surface-raised));
    font-size: 0.86rem;
    font-weight: 700;
  }
`

const TracePathList = styled.ul`
  margin: 0;
  padding-left: 1rem;
  display: grid;
  gap: 0.25rem;
`

const ComposerForm = styled.form`
  display: grid;
  gap: 0.5rem;
  padding-top: 0.2rem;
`

const ComposerTextarea = styled.textarea`
  min-height: 6.6rem;
  border: 1px solid color-mix(in srgb, var(--color-line) 82%, transparent);
  background: color-mix(in srgb, var(--color-input-bg) 74%, var(--surface-raised));
  border-radius: 14px;
  padding: 0.78rem 0.82rem;
  line-height: 1.45;
  font-size: 1rem;

  &::placeholder {
    color: color-mix(in srgb, var(--color-ink-muted) 82%, transparent);
  }

  &:focus {
    border-color: var(--session-active-border);
    box-shadow: 0 0 0 2px color-mix(in srgb, var(--session-active-shadow) 65%, transparent);
  }
`

const DebugPanel = styled.details`
  border: 1px dashed var(--color-line);
  border-radius: 10px;
  padding: 0.45rem 0.55rem;
  background: var(--surface-raised);
  display: grid;
  gap: 0.35rem;
  min-height: 0;

  &[open] {
    max-height: min(46vh, 34rem);
    overflow: hidden;
  }
`

const DebugPanelBody = styled.div`
  display: grid;
  gap: 0.35rem;
  min-height: 0;
  overflow: auto;
  padding-right: 0.2rem;
`

const DebugGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 0.35rem 0.6rem;
  font-size: 0.78rem;
  color: var(--color-ink-muted);
`

const DebugLabel = styled.strong`
  color: var(--color-ink);
  font-weight: 700;
`

const DebugBlock = styled.pre`
  margin: 0;
  font-size: 0.75rem;
  line-height: 1.45;
  white-space: pre-wrap;
  background: var(--markdown-code-bg);
  border-radius: 8px;
  padding: 0.45rem 0.55rem;
  max-height: min(34vh, 24rem);
  overflow: auto;
`

const TimelineStrip = styled.div`
  border-top: 1px dashed var(--color-line);
  padding-top: 0.5rem;
  display: grid;
  gap: 0.4rem;
`

const TimelineList = styled.ul`
  margin: 0;
  padding-left: 0;
  list-style: none;
  max-height: 12rem;
  overflow: auto;
  display: grid;
  gap: 0.3rem;
`

const TimelineItem = styled.li`
  border: 1px solid var(--color-line);
  border-radius: 8px;
  background: var(--surface-raised);
  padding: 0.38rem 0.45rem;
  display: grid;
  gap: 0.2rem;
`

const TimelineMeta = styled.p`
  font-size: 0.72rem;
  color: var(--color-ink-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
`

interface ChatPanelProps {
  session: ChatSession | null
  messages: ChatMessage[]
  pendingUserText: string | null
  streamingAssistantText: string
  composerText: string
  sending: boolean
  error: string | null
  sourceReferences: ChatSourceReference[]
  usedEngramLinkIds?: string[]
  engramTracePaths?: EngramTracePath[]
  debugTrace: ChatDebugTrace | null
  timelineEvents: ChatTimelineEvent[]
  showDebugTrace?: boolean
  showTimeline?: boolean
  linkRecallEnabled?: boolean
  linkRecallDepth?: number
  linkRecallMaxNeighbors?: number
  onLinkRecallEnabledChange?: (value: boolean) => void
  onLinkRecallDepthChange?: (value: number) => void
  onLinkRecallMaxNeighborsChange?: (value: number) => void
  onComposerChange: (value: string) => void
  onSend: () => Promise<void>
  onRetry: () => Promise<void>
  onOpenSaveModal: () => void
  onContinueSession: () => Promise<void>
}

interface PanelHeaderProps {
  session: ChatSession | null
  sending: boolean
  hasSession: boolean
  onOpenSaveModal: () => void
  onContinueSession: () => Promise<void>
}

interface TranscriptSectionProps {
  messages: ChatMessage[]
  pendingUserText: string | null
  streamingAssistantText: string
  hasSession: boolean
  sourceReferences: ChatSourceReference[]
  usedEngramLinkIds: string[]
  engramTracePaths: EngramTracePath[]
}

interface DebugTracePanelProps {
  debugTrace: ChatDebugTrace
}

interface TimelineSectionProps {
  timelineEvents: ChatTimelineEvent[]
}

interface TimelineRenderItem {
  key: string
  eventType: string
  title: string
  abstract: string
  createdAt: string
  groupedEvents: number
  mergedCount: number | null
}

interface ComposerSectionProps {
  hasSession: boolean
  sending: boolean
  composerText: string
  linkRecallEnabled: boolean
  linkRecallDepth: number
  linkRecallMaxNeighbors: number
  onComposerChange: (value: string) => void
  onLinkRecallEnabledChange: (value: boolean) => void
  onLinkRecallDepthChange: (value: number) => void
  onLinkRecallMaxNeighborsChange: (value: number) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => Promise<void>
  onComposerKeyDown: (event: KeyboardEvent<HTMLTextAreaElement>) => void
  onRetry: () => Promise<void>
}

interface RecallControlsSectionProps {
  hasSession: boolean
  sending: boolean
  linkRecallEnabled: boolean
  linkRecallDepth: number
  linkRecallMaxNeighbors: number
  onLinkRecallEnabledChange: (value: boolean) => void
  onLinkRecallDepthChange: (value: number) => void
  onLinkRecallMaxNeighborsChange: (value: number) => void
}

interface RecallInfoTooltipProps {
  label: string
  description: string
}

function MessageBubble({ role, text }: { role: string; text: string }) {
  const markdownComponents: Components = {
    a: ({ node, ...props }) => {
      void node
      return <a {...props} rel="noreferrer" target="_blank" />
    },
  }

  return (
    <ChatMessageBubble $role={role}>
      <MessageRole>{role}</MessageRole>
      <MessageText>
        <ReactMarkdown components={markdownComponents} remarkPlugins={[remarkGfm, remarkBreaks]}>
          {text || '(empty)'}
        </ReactMarkdown>
      </MessageText>
    </ChatMessageBubble>
  )
}

function isEnterSubmitKey(event: KeyboardEvent<HTMLTextAreaElement>): boolean {
  return event.key === 'Enter' && !event.shiftKey && !event.nativeEvent.isComposing && !event.repeat
}

function canSubmitComposer(hasSession: boolean, sending: boolean, composerText: string): boolean {
  return hasSession && !sending && composerText.trim().length > 0
}

function clampInputNumber(value: number, minValue: number, maxValue: number): number {
  if (!Number.isFinite(value)) {
    return minValue
  }
  if (value < minValue) {
    return minValue
  }
  if (value > maxValue) {
    return maxValue
  }
  return Math.trunc(value)
}

function compactTraceNode(nodeId: string): string {
  if (nodeId.length <= 12) {
    return nodeId
  }
  return `${nodeId.slice(0, 6)}...${nodeId.slice(-4)}`
}

function tokenSourceLabel(debugTrace: ChatDebugTrace): string {
  return debugTrace.llm_calls[0]?.token_usage_is_estimated ? 'estimated' : 'provider'
}

function formatTimelineType(eventType: string): string {
  const labels: Record<string, string> = {
    autosave_snapshot: 'Autosave Snapshot',
    manual_snapshot: 'Manual Snapshot',
    consolidation: 'Consolidation',
    consolidation_merge: 'Consolidation Merge',
    consolidation_group: 'Consolidation Group',
  }
  return labels[eventType] ?? eventType.replaceAll('_', ' ')
}

function isConsolidationTimelineEvent(eventType: string): boolean {
  return (
    eventType === 'consolidation' ||
    eventType === 'consolidation_merge' ||
    eventType === 'consolidation_group'
  )
}

function normalizedMergedCount(value: number | null | undefined): number | null {
  return typeof value === 'number' && Number.isFinite(value) && value > 0 ? value : null
}

function buildTimelineRenderEvents(timelineEvents: ChatTimelineEvent[]): TimelineRenderItem[] {
  const rendered: TimelineRenderItem[] = []
  const groupedIndexByKey = new Map<string, number>()

  for (const event of timelineEvents) {
    const groupKey = (event.consolidation_group_key || '').trim()
    const mergedCount = normalizedMergedCount(event.consolidation_merged_count)

    if (!groupKey || !isConsolidationTimelineEvent(event.event_type)) {
      rendered.push({
        key: event.event_id,
        eventType: event.event_type,
        title: event.title,
        abstract: event.abstract,
        createdAt: event.created_at,
        groupedEvents: 1,
        mergedCount,
      })
      continue
    }

    const existingIndex = groupedIndexByKey.get(groupKey)
    const currentMerged = mergedCount ?? 1
    if (existingIndex === undefined) {
      rendered.push({
        key: `consolidation-group:${groupKey}`,
        eventType: 'consolidation_group',
        title: 'Consolidation Group',
        abstract: `Group ${groupKey}: 1 consolidation snapshot (${currentMerged} merged engrams).`,
        createdAt: event.created_at,
        groupedEvents: 1,
        mergedCount: currentMerged,
      })
      groupedIndexByKey.set(groupKey, rendered.length - 1)
      continue
    }

    const existing = rendered[existingIndex]
    const groupedEvents = existing.groupedEvents + 1
    const nextMergedCount = (existing.mergedCount ?? existing.groupedEvents) + currentMerged
    rendered[existingIndex] = {
      ...existing,
      abstract: `Group ${groupKey}: ${groupedEvents} consolidation snapshots (${nextMergedCount} merged engrams).`,
      groupedEvents,
      mergedCount: nextMergedCount,
    }
  }

  return rendered
}

function buildTimelineMeta(event: TimelineRenderItem): string {
  const parts = [formatTimelineType(event.eventType), new Date(event.createdAt).toLocaleString()]
  if (event.groupedEvents > 1) {
    parts.push(`${event.groupedEvents} grouped`)
  } else if (event.mergedCount && event.mergedCount > 1) {
    parts.push(`${event.mergedCount} merged`)
  }
  return parts.join(' · ')
}

function PanelHeader({ session, sending, hasSession, onOpenSaveModal, onContinueSession }: PanelHeaderProps) {
  return (
    <PaneHeader as="header">
      <HeaderBlock>
        <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">
          {session ? session.title : 'Select or create a session'}
        </h2>
        <MutedText>
          {session ? `${session.provider}/${session.model_id} · ${session.visibility_scope}` : 'No active session'}
        </MutedText>
      </HeaderBlock>

      <ActionRow>
        <button type="button" onClick={onOpenSaveModal} disabled={!hasSession || sending}>
          Save as Engram
        </button>
        <button type="button" onClick={onContinueSession} disabled={!hasSession || sending}>
          Continue in New Chat
        </button>
      </ActionRow>
    </PaneHeader>
  )
}

function TranscriptSection({
  messages,
  pendingUserText,
  streamingAssistantText,
  hasSession,
  sourceReferences,
  usedEngramLinkIds,
  engramTracePaths,
}: TranscriptSectionProps) {
  return (
    <>
      <TranscriptRail>
        {messages.map((message) => (
          <MessageBubble key={message.message_id} role={message.role} text={message.content_text} />
        ))}

        {pendingUserText ? <MessageBubble role="user" text={pendingUserText} /> : null}
        {streamingAssistantText ? <MessageBubble role="assistant" text={streamingAssistantText} /> : null}
        {!hasSession ? <MutedText>Session messages will appear here.</MutedText> : null}
      </TranscriptRail>

      {sourceReferences.length > 0 ? (
        <SourceStrip>
          <SourceTitle>Source references used:</SourceTitle>
          <ul>
            {sourceReferences.map((ref) => (
              <li key={`${ref.engram_id}-${ref.url}`}>
                <a href={ref.url} target="_blank" rel="noreferrer">
                  {ref.title || ref.url}
                </a>
                <span> ({ref.engram_title})</span>
              </li>
            ))}
          </ul>
        </SourceStrip>
      ) : null}

      {usedEngramLinkIds.length > 0 || engramTracePaths.length > 0 ? (
        <SourceStrip data-testid="trace-path-strip">
          <SourceTitle>Linked trace paths used:</SourceTitle>
          <MutedText>
            {usedEngramLinkIds.length} links across {engramTracePaths.length} paths.
          </MutedText>
          {engramTracePaths.length > 0 ? (
            <TracePathList>
              {engramTracePaths.slice(0, 5).map((trace) => (
                <li key={`${trace.root_engram_id}-${trace.target_engram_id}-${trace.depth}`}>
                  {trace.engram_ids.map(compactTraceNode).join(' -> ')} (depth {trace.depth}, score{' '}
                  {trace.score.toFixed(2)})
                </li>
              ))}
            </TracePathList>
          ) : null}
        </SourceStrip>
      ) : null}
    </>
  )
}

function DebugTracePanel({ debugTrace }: DebugTracePanelProps) {
  return (
    <DebugPanel>
      <summary>Debug Trace</summary>
      <DebugPanelBody>
        <DebugGrid>
          <span>
            <DebugLabel>Total:</DebugLabel> {debugTrace.total_duration_ms.toFixed(1)} ms
          </span>
          <span>
            <DebugLabel>Context:</DebugLabel> {debugTrace.context_duration_ms.toFixed(1)} ms
          </span>
          <span>
            <DebugLabel>Embeds:</DebugLabel> {debugTrace.embedding_calls.length}
          </span>
          <span>
            <DebugLabel>LLM Call:</DebugLabel> {debugTrace.llm_call_duration_ms.toFixed(1)} ms
          </span>
          <span>
            <DebugLabel>Input Tokens:</DebugLabel> {debugTrace.llm_calls[0]?.token_usage?.input_tokens ?? 0}
          </span>
          <span>
            <DebugLabel>Output Tokens:</DebugLabel> {debugTrace.llm_calls[0]?.token_usage?.output_tokens ?? 0}
          </span>
          <span>
            <DebugLabel>Token Source:</DebugLabel> {tokenSourceLabel(debugTrace)}
          </span>
        </DebugGrid>
        <DebugBlock>{JSON.stringify(debugTrace, null, 2)}</DebugBlock>
      </DebugPanelBody>
    </DebugPanel>
  )
}

function TimelineSection({ timelineEvents }: TimelineSectionProps) {
  const renderEvents = buildTimelineRenderEvents(timelineEvents)

  return (
    <TimelineStrip>
      <SourceTitle>Lifecycle Timeline</SourceTitle>
      {renderEvents.length === 0 ? (
        <MutedText>No lifecycle events yet.</MutedText>
      ) : (
        <TimelineList>
          {renderEvents.map((event) => (
            <TimelineItem key={event.key} data-testid="timeline-item">
              <TimelineMeta>{buildTimelineMeta(event)}</TimelineMeta>
              <strong>{event.title}</strong>
              <MutedText>{event.abstract}</MutedText>
            </TimelineItem>
          ))}
        </TimelineList>
      )}
    </TimelineStrip>
  )
}

function RecallInfoTooltip({ label, description }: RecallInfoTooltipProps) {
  return (
    <RecallHelpContainer>
      <RecallHelp>
        <RecallHelpTrigger aria-label="Open setting help tooltip" title={`About ${label}`} type="button">
          ?
        </RecallHelpTrigger>
        <RecallHelpText role="tooltip">{description}</RecallHelpText>
      </RecallHelp>
    </RecallHelpContainer>
  )
}

function ComposerSection({
  hasSession,
  sending,
  composerText,
  linkRecallEnabled,
  linkRecallDepth,
  linkRecallMaxNeighbors,
  onComposerChange,
  onLinkRecallEnabledChange,
  onLinkRecallDepthChange,
  onLinkRecallMaxNeighborsChange,
  onSubmit,
  onComposerKeyDown,
  onRetry,
}: ComposerSectionProps) {
  return (
    <ComposerForm onSubmit={(event) => void onSubmit(event)}>
      <RecallControlsSection
        hasSession={hasSession}
        sending={sending}
        linkRecallEnabled={linkRecallEnabled}
        linkRecallDepth={linkRecallDepth}
        linkRecallMaxNeighbors={linkRecallMaxNeighbors}
        onLinkRecallEnabledChange={onLinkRecallEnabledChange}
        onLinkRecallDepthChange={onLinkRecallDepthChange}
        onLinkRecallMaxNeighborsChange={onLinkRecallMaxNeighborsChange}
      />

      <ComposerTextarea
        value={composerText}
        onChange={(event) => onComposerChange(event.target.value)}
        onKeyDown={onComposerKeyDown}
        rows={3}
        placeholder={hasSession ? 'Ask something and stream a response...' : 'Create a session first'}
        disabled={!hasSession || sending}
      />

      <ComposerActions>
        <PrimaryMinimalButton type="submit" disabled={!canSubmitComposer(hasSession, sending, composerText)}>
          {sending ? 'Streaming...' : 'Send'}
        </PrimaryMinimalButton>
        <MinimalButton type="button" onClick={onRetry} disabled={!hasSession || sending}>
          Retry Last Prompt
        </MinimalButton>
      </ComposerActions>
    </ComposerForm>
  )
}

function RecallControlsSection({
  hasSession,
  sending,
  linkRecallEnabled,
  linkRecallDepth,
  linkRecallMaxNeighbors,
  onLinkRecallEnabledChange,
  onLinkRecallDepthChange,
  onLinkRecallMaxNeighborsChange,
}: RecallControlsSectionProps) {
  const controlsDisabled = !hasSession || sending
  const numericDisabled = controlsDisabled || !linkRecallEnabled

  return (
    <RecallControls>
      <RecallCard>
        <RecallHeader>
          <RecallLabel>Link Recall</RecallLabel>
          <RecallInfoTooltip
            label="Link Recall"
            description="When enabled, the assistant traverses related memory links before responding."
          />
        </RecallHeader>
        <RecallHint>Use linked memory context in this response.</RecallHint>
        <RecallCheckbox>
          <input
            aria-label="Link Recall"
            checked={linkRecallEnabled}
            disabled={controlsDisabled}
            onChange={(event) => onLinkRecallEnabledChange(event.target.checked)}
            type="checkbox"
          />
          {linkRecallEnabled ? 'Enabled' : 'Disabled'}
        </RecallCheckbox>
      </RecallCard>

      <RecallCard>
        <RecallHeader>
          <RecallLabel>Recall Depth</RecallLabel>
          <RecallInfoTooltip
            label="Recall Depth"
            description="How many link hops to traverse from initially matched memory nodes. Higher depth broadens recall."
          />
        </RecallHeader>
        <RecallHint>Recommended: 1-2 for focused answers.</RecallHint>
        <RecallNumberControl>
          Hops (1-3)
          <input
            data-testid="link-recall-depth-input"
            disabled={numericDisabled}
            min={1}
            max={3}
            onChange={(event) => onLinkRecallDepthChange(clampInputNumber(Number(event.target.value), 1, 3))}
            type="number"
            value={linkRecallDepth}
          />
        </RecallNumberControl>
      </RecallCard>

      <RecallCard>
        <RecallHeader>
          <RecallLabel>Max Neighbors</RecallLabel>
          <RecallInfoTooltip
            label="Max Neighbors"
            description="Per node, limit how many linked neighbors are considered. Lower values reduce noise and latency."
          />
        </RecallHeader>
        <RecallHint>Controls breadth per hop (1-24).</RecallHint>
        <RecallNumberControl>
          Neighbors
          <input
            data-testid="link-recall-neighbors-input"
            disabled={numericDisabled}
            min={1}
            max={24}
            onChange={(event) =>
              onLinkRecallMaxNeighborsChange(clampInputNumber(Number(event.target.value), 1, 24))
            }
            type="number"
            value={linkRecallMaxNeighbors}
          />
        </RecallNumberControl>
      </RecallCard>
    </RecallControls>
  )
}

export function ChatPanel({
  session,
  messages,
  pendingUserText,
  streamingAssistantText,
  composerText,
  sending,
  error,
  sourceReferences,
  usedEngramLinkIds = [],
  engramTracePaths = [],
  debugTrace,
  timelineEvents,
  showDebugTrace = true,
  showTimeline = true,
  linkRecallEnabled = true,
  linkRecallDepth = 1,
  linkRecallMaxNeighbors = 8,
  onLinkRecallEnabledChange = () => undefined,
  onLinkRecallDepthChange = () => undefined,
  onLinkRecallMaxNeighborsChange = () => undefined,
  onComposerChange,
  onSend,
  onRetry,
  onOpenSaveModal,
  onContinueSession,
}: ChatPanelProps) {
  const hasSession = Boolean(session)

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    await onSend()
  }

  const handleComposerKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (!isEnterSubmitKey(event)) {
      return
    }
    if (!canSubmitComposer(hasSession, sending, composerText)) {
      return
    }
    event.preventDefault()
    void onSend()
  }

  return (
    <GlassPane data-testid="chat-panel">
      <PanelHeader
        session={session}
        sending={sending}
        hasSession={hasSession}
        onOpenSaveModal={onOpenSaveModal}
        onContinueSession={onContinueSession}
      />

      <SectionDivider />

      <TranscriptSection
        messages={messages}
        pendingUserText={pendingUserText}
        streamingAssistantText={streamingAssistantText}
        hasSession={hasSession}
        sourceReferences={sourceReferences}
        usedEngramLinkIds={usedEngramLinkIds}
        engramTracePaths={engramTracePaths}
      />

      {showDebugTrace && debugTrace ? <DebugTracePanel debugTrace={debugTrace} /> : null}

      {showTimeline ? <TimelineSection timelineEvents={timelineEvents} /> : null}

      <ComposerSection
        hasSession={hasSession}
        sending={sending}
        composerText={composerText}
        linkRecallEnabled={linkRecallEnabled}
        linkRecallDepth={linkRecallDepth}
        linkRecallMaxNeighbors={linkRecallMaxNeighbors}
        onComposerChange={onComposerChange}
        onLinkRecallEnabledChange={onLinkRecallEnabledChange}
        onLinkRecallDepthChange={onLinkRecallDepthChange}
        onLinkRecallMaxNeighborsChange={onLinkRecallMaxNeighborsChange}
        onSubmit={handleSubmit}
        onComposerKeyDown={handleComposerKeyDown}
        onRetry={onRetry}
      />

      {error ? <ErrorText>{error}</ErrorText> : null}
    </GlassPane>
  )
}
