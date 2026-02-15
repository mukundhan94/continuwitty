import { useState } from 'react'
import type { FormEvent } from 'react'

import type { VisibilityScope } from '../api/types'

interface SaveEngramModalProps {
  defaultTitle: string
  saving: boolean
  onClose: () => void
  onSave: (payload: {
    title: string
    abstract: string
    visibility_scope: VisibilityScope
    tags: string[]
    keywords: string[]
  }) => Promise<void>
}

function splitCsv(input: string): string[] {
  return input
    .split(',')
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
}

export function SaveEngramModal({ defaultTitle, saving, onClose, onSave }: SaveEngramModalProps) {
  const [title, setTitle] = useState(defaultTitle)
  const [abstract, setAbstract] = useState('Snapshot from active chat session.')
  const [visibilityScope, setVisibilityScope] = useState<VisibilityScope>('private')
  const [tagsText, setTagsText] = useState('chat,snapshot')
  const [keywordsText, setKeywordsText] = useState('continuity,engram')

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    await onSave({
      title: title.trim(),
      abstract: abstract.trim(),
      visibility_scope: visibilityScope,
      tags: splitCsv(tagsText),
      keywords: splitCsv(keywordsText),
    })
  }

  return (
    <div className="modal-backdrop" role="presentation" onClick={onClose}>
      <div className="modal" role="dialog" aria-modal="true" onClick={(event) => event.stopPropagation()}>
        <h3>Save Session as Engram</h3>
        <form className="modal-form" onSubmit={handleSubmit}>
          <label htmlFor="save-title">Title</label>
          <input id="save-title" value={title} onChange={(event) => setTitle(event.target.value)} required />

          <label htmlFor="save-abstract">Abstract</label>
          <textarea
            id="save-abstract"
            value={abstract}
            onChange={(event) => setAbstract(event.target.value)}
            rows={3}
            required
          />

          <label htmlFor="save-visibility">Visibility</label>
          <select
            id="save-visibility"
            value={visibilityScope}
            onChange={(event) => setVisibilityScope(event.target.value as VisibilityScope)}
          >
            <option value="private">Private</option>
            <option value="project">Project</option>
          </select>

          <label htmlFor="save-tags">Tags (comma separated)</label>
          <input id="save-tags" value={tagsText} onChange={(event) => setTagsText(event.target.value)} />

          <label htmlFor="save-keywords">Keywords (comma separated)</label>
          <input id="save-keywords" value={keywordsText} onChange={(event) => setKeywordsText(event.target.value)} />

          <div className="modal-actions">
            <button type="button" onClick={onClose} disabled={saving}>
              Cancel
            </button>
            <button type="submit" disabled={saving || !title.trim() || !abstract.trim()}>
              {saving ? 'Saving...' : 'Save Engram'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
