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
`

const ComposerActions = styled.div`
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  flex-wrap: wrap;
`

const ComposerForm = styled.form`
  display: grid;
  gap: 0.5rem;
  padding-top: 0.2rem;
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
  debugTrace: ChatDebugTrace | null
  timelineEvents: ChatTimelineEvent[]
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
}

interface DebugTracePanelProps {
  debugTrace: ChatDebugTrace
}

interface TimelineSectionProps {
  timelineEvents: ChatTimelineEvent[]
}

interface ComposerSectionProps {
  hasSession: boolean
  sending: boolean
  composerText: string
  onComposerChange: (value: string) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => Promise<void>
  onComposerKeyDown: (event: KeyboardEvent<HTMLTextAreaElement>) => void
  onRetry: () => Promise<void>
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

function tokenSourceLabel(debugTrace: ChatDebugTrace): string {
  return debugTrace.llm_calls[0]?.token_usage_is_estimated ? 'estimated' : 'provider'
}

function formatTimelineType(eventType: string): string {
  return eventType.replaceAll('_', ' ')
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
  return (
    <TimelineStrip>
      <SourceTitle>Lifecycle Timeline</SourceTitle>
      {timelineEvents.length === 0 ? (
        <MutedText>No lifecycle events yet.</MutedText>
      ) : (
        <TimelineList>
          {timelineEvents.map((event) => (
            <TimelineItem key={event.event_id}>
              <TimelineMeta>
                {formatTimelineType(event.event_type)} · {new Date(event.created_at).toLocaleString()}
              </TimelineMeta>
              <strong>{event.title}</strong>
              <MutedText>{event.abstract}</MutedText>
            </TimelineItem>
          ))}
        </TimelineList>
      )}
    </TimelineStrip>
  )
}

function ComposerSection({
  hasSession,
  sending,
  composerText,
  onComposerChange,
  onSubmit,
  onComposerKeyDown,
  onRetry,
}: ComposerSectionProps) {
  return (
    <ComposerForm onSubmit={(event) => void onSubmit(event)}>
      <textarea
        value={composerText}
        onChange={(event) => onComposerChange(event.target.value)}
        onKeyDown={onComposerKeyDown}
        rows={3}
        placeholder={hasSession ? 'Ask something and stream a response...' : 'Create a session first'}
        disabled={!hasSession || sending}
      />

      <ComposerActions>
        <button type="submit" disabled={!canSubmitComposer(hasSession, sending, composerText)}>
          {sending ? 'Streaming...' : 'Send'}
        </button>
        <button type="button" onClick={onRetry} disabled={!hasSession || sending}>
          Retry Last Prompt
        </button>
      </ComposerActions>
    </ComposerForm>
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
  debugTrace,
  timelineEvents,
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
      />

      {debugTrace ? <DebugTracePanel debugTrace={debugTrace} /> : null}

      <TimelineSection timelineEvents={timelineEvents} />

      <ComposerSection
        hasSession={hasSession}
        sending={sending}
        composerText={composerText}
        onComposerChange={onComposerChange}
        onSubmit={handleSubmit}
        onComposerKeyDown={handleComposerKeyDown}
        onRetry={onRetry}
      />

      {error ? <ErrorText>{error}</ErrorText> : null}
    </GlassPane>
  )
}
