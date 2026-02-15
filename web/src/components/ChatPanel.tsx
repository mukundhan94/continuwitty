import type { FormEvent, KeyboardEvent } from 'react'
import styled from 'styled-components'

import type { ChatMessage, ChatSession, ChatSourceReference } from '../api/types'
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

interface ChatPanelProps {
  session: ChatSession | null
  messages: ChatMessage[]
  pendingUserText: string | null
  streamingAssistantText: string
  composerText: string
  sending: boolean
  error: string | null
  sourceReferences: ChatSourceReference[]
  onComposerChange: (value: string) => void
  onSend: () => Promise<void>
  onRetry: () => Promise<void>
  onOpenSaveModal: () => void
  onContinueSession: () => Promise<void>
}

function MessageBubble({ role, text }: { role: string; text: string }) {
  return (
    <ChatMessageBubble $role={role}>
      <MessageRole>{role}</MessageRole>
      <MessageText>{text || '(empty)'}</MessageText>
    </ChatMessageBubble>
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
    if (event.key !== 'Enter') {
      return
    }
    if (event.shiftKey) {
      return
    }
    if (event.nativeEvent.isComposing || event.repeat) {
      return
    }
    if (!hasSession || sending || !composerText.trim()) {
      return
    }
    event.preventDefault()
    void onSend()
  }

  return (
    <GlassPane>
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

      <SectionDivider />

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

      <ComposerForm onSubmit={handleSubmit}>
        <textarea
          value={composerText}
          onChange={(event) => onComposerChange(event.target.value)}
          onKeyDown={handleComposerKeyDown}
          rows={3}
          placeholder={hasSession ? 'Ask something and stream a response...' : 'Create a session first'}
          disabled={!hasSession || sending}
        />

        <ComposerActions>
          <button type="submit" disabled={!hasSession || sending || !composerText.trim()}>
            {sending ? 'Streaming...' : 'Send'}
          </button>
          <button type="button" onClick={onRetry} disabled={!hasSession || sending}>
            Retry Last Prompt
          </button>
        </ComposerActions>
      </ComposerForm>

      {error ? <ErrorText>{error}</ErrorText> : null}
    </GlassPane>
  )
}
