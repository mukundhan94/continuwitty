import { useId, useMemo } from 'react'
import type { KeyboardEvent } from 'react'

import { SessionMeta } from '../styles/primitives'

function normalizeOptionList(options: string[]): string[] {
  const unique = new Set<string>()
  for (const option of options) {
    const normalized = option.trim()
    if (!normalized) {
      continue
    }
    unique.add(normalized)
  }
  return Array.from(unique).sort((left, right) => left.localeCompare(right))
}

function rankOptions(options: string[], query: string): string[] {
  const normalizedQuery = query.trim().toLowerCase()
  if (!normalizedQuery) {
    return options
  }

  const startsWithMatches: string[] = []
  const containsMatches: string[] = []
  for (const option of options) {
    const normalized = option.toLowerCase()
    if (normalized.startsWith(normalizedQuery)) {
      startsWithMatches.push(option)
      continue
    }
    if (normalized.includes(normalizedQuery)) {
      containsMatches.push(option)
    }
  }

  return [...startsWithMatches, ...containsMatches]
}

interface SmartIdDropdownProps {
  id?: string
  value: string
  options: string[]
  placeholder?: string
  disabled?: boolean
  maxOptions?: number
  searchText?: string
  showMatchCount?: boolean
  onChange: (value: string) => void
  onKeyDown?: (event: KeyboardEvent<HTMLInputElement>) => void
  inputTestId?: string
  optionsTestId?: string
  matchCountTestId?: string
}

export function SmartIdDropdown({
  id,
  value,
  options,
  placeholder,
  disabled = false,
  maxOptions = 50,
  searchText,
  showMatchCount = true,
  onChange,
  onKeyDown,
  inputTestId,
  optionsTestId,
  matchCountTestId,
}: SmartIdDropdownProps) {
  const generatedId = useId()
  const listId = `${id ?? generatedId}-list`

  const normalizedOptions = useMemo(() => normalizeOptionList(options), [options])
  const effectiveSearchText = searchText ?? value
  const rankedOptions = useMemo(
    () => rankOptions(normalizedOptions, effectiveSearchText).slice(0, maxOptions),
    [effectiveSearchText, maxOptions, normalizedOptions],
  )

  return (
    <>
      <input
        id={id}
        list={listId}
        value={value}
        placeholder={placeholder}
        disabled={disabled}
        onChange={(event) => onChange(event.target.value)}
        onKeyDown={onKeyDown}
        data-testid={inputTestId}
      />
      <datalist id={listId} data-testid={optionsTestId}>
        {rankedOptions.map((option) => (
          <option key={option} value={option} />
        ))}
      </datalist>
      {showMatchCount ? (
        <SessionMeta data-testid={matchCountTestId}>
          {rankedOptions.length} match{rankedOptions.length === 1 ? '' : 'es'}
        </SessionMeta>
      ) : null}
    </>
  )
}
