import type { FormEvent, KeyboardEvent } from 'react'

import type { ChatMessage, ChatSession, ChatSourceReference } from '../api/types'

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
    <article className={`message message-${role}`}>
      <p className="message-role">{role}</p>
      <p className="message-text">{text || '(empty)'}</p>
    </article>
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
    <section className="pane pane-chat">
      <header className="pane-header chat-header">
        <div>
          <h2>{session ? session.title : 'Select or create a session'}</h2>
          <p className="muted">
            {session ? `${session.provider}/${session.model_id} · ${session.visibility_scope}` : 'No active session'}
          </p>
        </div>
        <div className="chat-actions">
          <button onClick={onOpenSaveModal} disabled={!hasSession || sending}>
            Save as Engram
          </button>
          <button onClick={onContinueSession} disabled={!hasSession || sending}>
            Continue in New Chat
          </button>
        </div>
      </header>

      <div className="transcript">
        {messages.map((message) => (
          <MessageBubble key={message.message_id} role={message.role} text={message.content_text} />
        ))}
        {pendingUserText ? <MessageBubble role="user" text={pendingUserText} /> : null}
        {streamingAssistantText ? <MessageBubble role="assistant" text={streamingAssistantText} /> : null}
        {!hasSession ? <p className="muted">Session messages will appear here.</p> : null}
      </div>

      {sourceReferences.length > 0 ? (
        <div className="source-strip">
          <p className="source-title">Source references used:</p>
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
        </div>
      ) : null}

      <form className="composer" onSubmit={handleSubmit}>
        <textarea
          value={composerText}
          onChange={(event) => onComposerChange(event.target.value)}
          onKeyDown={handleComposerKeyDown}
          rows={3}
          placeholder={hasSession ? 'Ask something and stream a response...' : 'Create a session first'}
          disabled={!hasSession || sending}
        />
        <div className="composer-actions">
          <button type="submit" disabled={!hasSession || sending || !composerText.trim()}>
            {sending ? 'Streaming...' : 'Send'}
          </button>
          <button type="button" onClick={onRetry} disabled={!hasSession || sending}>
            Retry Last Prompt
          </button>
        </div>
      </form>

      {error ? <p className="error-line">{error}</p> : null}
    </section>
  )
}
