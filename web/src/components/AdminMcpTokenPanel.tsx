import { useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import styled from 'styled-components'

import type {
  McpTokenCreateRequest,
  McpTokenCreateResponse,
  McpTokenScope,
  McpTokenSummary,
} from '../api/types'

type Props = {
  isOpen: boolean
  loading: boolean
  optionsLoading: boolean
  creating: boolean
  tokens: McpTokenSummary[]
  latestToken: McpTokenCreateResponse | null
  error: string | null
  availableTools: string[]
  availableProjects: string[]
  onClose: () => void
  onRefresh: () => Promise<void>
  onCreate: (payload: McpTokenCreateRequest) => Promise<void>
  onRevoke: (tokenId: string) => Promise<void>
}

interface OptionChipSelectorProps {
  label: string
  ariaLabel: string
  testId: string
  addButtonTestId: string
  addButtonLabel: string
  emptyOptionLabel: string
  emptyChipLabel: string
  removeLabelPrefix: string
  availableChoices: string[]
  pendingValues: string[]
  selectedValues: string[]
  onPendingValuesChange: (values: string[]) => void
  onAdd: () => void
  onRemove: (value: string) => void
}

interface TokenCreationFormProps {
  creating: boolean
  availableTools: string[]
  availableProjects: string[]
  onCreate: (payload: McpTokenCreateRequest) => Promise<void>
}

interface IssuedTokensTableProps {
  tokens: McpTokenSummary[]
  onRevoke: (tokenId: string) => Promise<void>
}

const Overlay = styled.div`
  position: fixed;
  inset: 0;
  background: rgba(8, 12, 20, 0.48);
  backdrop-filter: blur(2px);
  display: grid;
  place-items: center;
  padding: 1rem;
  z-index: 50;
`

const Dialog = styled.section`
  width: min(1080px, 100%);
  max-height: min(92vh, 920px);
  overflow: auto;
  background: var(--surface-glass);
  border: 1px solid var(--surface-glass-border);
  border-radius: 16px;
  box-shadow: var(--shadow-panel);
  padding: 1rem;
  display: grid;
  gap: 0.8rem;
`

const Header = styled.header`
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.8rem;
`

const HeaderActions = styled.div`
  display: flex;
  align-items: center;
  gap: 0.5rem;
`

const Hint = styled.p`
  font-size: 0.84rem;
  color: var(--color-ink-muted);
`

const ErrorText = styled.p`
  color: var(--color-error);
  font-size: 0.86rem;
`

const SuccessBox = styled.div`
  border: 1px solid var(--color-notice-border);
  background: var(--color-notice-bg);
  border-radius: 10px;
  padding: 0.6rem;
  display: grid;
  gap: 0.35rem;
`

const TokenCode = styled.code`
  display: block;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
  background: var(--surface-mute);
  border: 1px solid var(--color-line);
  border-radius: 8px;
  padding: 0.5rem;
`

const FormGrid = styled.form`
  display: grid;
  gap: 0.55rem;
  border-top: 1px dashed var(--color-line);
  border-bottom: 1px dashed var(--color-line);
  padding: 0.8rem 0;
`

const Grid = styled.div`
  display: grid;
  gap: 0.6rem;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
`

const Field = styled.label`
  display: grid;
  gap: 0.28rem;
  font-size: 0.82rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-ink-muted);
`

const OptionSelectRow = styled.div`
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 0.45rem;
`

const Input = styled.input`
  width: 100%;
  border: 1px solid var(--color-line);
  border-radius: 10px;
  background: var(--surface-mute);
  color: var(--color-ink);
  padding: 0.55rem 0.65rem;
`

const Select = styled.select`
  width: 100%;
  border: 1px solid var(--color-line);
  border-radius: 10px;
  background: var(--surface-mute);
  color: var(--color-ink);
  padding: 0.55rem 0.65rem;
`

const ChipWrap = styled.div`
  display: flex;
  gap: 0.4rem;
  flex-wrap: wrap;
`

const Chip = styled.span`
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  border-radius: 999px;
  border: 1px solid var(--color-line);
  padding: 0.2rem 0.5rem;
  background: var(--surface-mute);
  font-size: 0.78rem;

  button {
    border: 0;
    border-radius: 999px;
    width: 1rem;
    height: 1rem;
    line-height: 1rem;
    padding: 0;
    background: transparent;
    color: var(--color-ink-muted);
    cursor: pointer;
    font-weight: 700;
  }
`

const TableWrap = styled.div`
  overflow: auto;
  border: 1px solid var(--color-line);
  border-radius: 10px;
`

const Table = styled.table`
  width: 100%;
  border-collapse: collapse;
  min-width: 920px;

  th,
  td {
    border-bottom: 1px solid var(--color-line);
    text-align: left;
    padding: 0.5rem;
    vertical-align: top;
    font-size: 0.85rem;
  }
`

const EmptyText = styled.p`
  color: var(--color-ink-muted);
  font-size: 0.85rem;
`

const STATUS_ACTIVE = 'active'
const DEFAULT_EXPIRY_DAYS = 90

function mergeUnique(current: string[], additions: string[]): string[] {
  const merged = [...current]
  for (const item of additions) {
    if (!merged.includes(item)) {
      merged.push(item)
    }
  }
  return merged
}

function toSelectedOptions(select: HTMLSelectElement): string[] {
  return Array.from(select.selectedOptions, (option) => option.value).filter(Boolean)
}

function effectivePendingValues(pendingValues: string[], availableChoices: string[]): string[] {
  return pendingValues.filter((item) => availableChoices.includes(item))
}

function selectSize(optionCount: number): number {
  return Math.min(Math.max(optionCount, 3), 8)
}

function formatTimestamp(value: string | null): string {
  return value ? new Date(value).toLocaleString() : '-'
}

function OptionChipSelector({
  label,
  ariaLabel,
  testId,
  addButtonTestId,
  addButtonLabel,
  emptyOptionLabel,
  emptyChipLabel,
  removeLabelPrefix,
  availableChoices,
  pendingValues,
  selectedValues,
  onPendingValuesChange,
  onAdd,
  onRemove,
}: OptionChipSelectorProps) {
  return (
    <Field>
      {label}
      <OptionSelectRow>
        <Select
          aria-label={ariaLabel}
          data-testid={testId}
          multiple
          size={selectSize(availableChoices.length)}
          value={pendingValues}
          onChange={(event) => onPendingValuesChange(toSelectedOptions(event.target))}
          disabled={availableChoices.length === 0}
        >
          {availableChoices.length === 0 ? <option value="">{emptyOptionLabel}</option> : null}
          {availableChoices.map((value) => (
            <option key={value} value={value}>
              {value}
            </option>
          ))}
        </Select>
        <button type="button" data-testid={addButtonTestId} onClick={onAdd} disabled={pendingValues.length === 0}>
          {addButtonLabel}
        </button>
      </OptionSelectRow>
      <Hint>{`Select one or more options from the list, then click ${addButtonLabel}.`}</Hint>
      <ChipWrap>
        {selectedValues.length === 0
          ? <Hint>{emptyChipLabel}</Hint>
          : selectedValues.map((value) => (
              <Chip key={value}>
                {value}
                <button type="button" aria-label={`${removeLabelPrefix} ${value}`} onClick={() => onRemove(value)}>
                  ×
                </button>
              </Chip>
            ))}
      </ChipWrap>
    </Field>
  )
}

function TokenCreationForm({ creating, availableTools, availableProjects, onCreate }: TokenCreationFormProps) {
  const [name, setName] = useState('')
  const [scope, setScope] = useState<McpTokenScope>('read')
  const [expiresInDays, setExpiresInDays] = useState(DEFAULT_EXPIRY_DAYS)
  const [selectedTools, setSelectedTools] = useState<string[]>([])
  const [selectedProjects, setSelectedProjects] = useState<string[]>([])
  const [pendingTools, setPendingTools] = useState<string[]>([])
  const [pendingProjects, setPendingProjects] = useState<string[]>([])

  const availableToolChoices = useMemo(
    () => availableTools.filter((item) => !selectedTools.includes(item)),
    [availableTools, selectedTools],
  )
  const availableProjectChoices = useMemo(
    () => availableProjects.filter((item) => !selectedProjects.includes(item)),
    [availableProjects, selectedProjects],
  )

  const effectivePendingTools = useMemo(
    () => effectivePendingValues(pendingTools, availableToolChoices),
    [pendingTools, availableToolChoices],
  )
  const effectivePendingProjects = useMemo(
    () => effectivePendingValues(pendingProjects, availableProjectChoices),
    [pendingProjects, availableProjectChoices],
  )

  const addTools = () => {
    if (effectivePendingTools.length === 0) {
      return
    }
    setSelectedTools((current) => mergeUnique(current, effectivePendingTools))
    setPendingTools([])
  }

  const addProjects = () => {
    if (effectivePendingProjects.length === 0) {
      return
    }
    setSelectedProjects((current) => mergeUnique(current, effectivePendingProjects))
    setPendingProjects([])
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    await onCreate({
      name: name.trim(),
      scope,
      allowed_tools: selectedTools,
      allowed_project_ids: selectedProjects,
      expires_in_days: expiresInDays,
    })
    setSelectedTools([])
    setSelectedProjects([])
    setPendingTools([])
    setPendingProjects([])
  }

  return (
    <FormGrid onSubmit={(event) => void handleSubmit(event)}>
      <Grid>
        <Field>
          Token Name
          <Input
            aria-label="Token Name"
            required
            minLength={3}
            maxLength={120}
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder="LibreChat read token"
          />
        </Field>
        <Field>
          Scope
          <Select
            aria-label="Scope"
            value={scope}
            onChange={(event) => setScope(event.target.value as McpTokenScope)}
          >
            <option value="read">read</option>
            <option value="write">write</option>
          </Select>
        </Field>
        <Field>
          Expiry (days)
          <Input
            aria-label="Expiry Days"
            type="number"
            min={1}
            max={3650}
            value={expiresInDays}
            onChange={(event) => setExpiresInDays(Number(event.target.value || DEFAULT_EXPIRY_DAYS))}
          />
        </Field>
      </Grid>

      <OptionChipSelector
        label="Allowed Tools (optional)"
        ariaLabel="Tool Options"
        testId="admin-tool-options"
        addButtonTestId="admin-add-tool-chip"
        addButtonLabel="Add Tools"
        emptyOptionLabel="No tools available"
        emptyChipLabel="No tool restrictions selected."
        removeLabelPrefix="Remove tool"
        availableChoices={availableToolChoices}
        pendingValues={effectivePendingTools}
        selectedValues={selectedTools}
        onPendingValuesChange={setPendingTools}
        onAdd={addTools}
        onRemove={(value) => setSelectedTools((current) => current.filter((item) => item !== value))}
      />

      <OptionChipSelector
        label="Allowed Project IDs (optional)"
        ariaLabel="Project Options"
        testId="admin-project-options"
        addButtonTestId="admin-add-project-chip"
        addButtonLabel="Add Projects"
        emptyOptionLabel="No projects available"
        emptyChipLabel="No project restrictions selected."
        removeLabelPrefix="Remove project"
        availableChoices={availableProjectChoices}
        pendingValues={effectivePendingProjects}
        selectedValues={selectedProjects}
        onPendingValuesChange={setPendingProjects}
        onAdd={addProjects}
        onRemove={(value) => setSelectedProjects((current) => current.filter((item) => item !== value))}
      />

      <button type="submit" disabled={creating}>
        {creating ? 'Creating...' : 'Create Token'}
      </button>
    </FormGrid>
  )
}

function IssuedTokensTable({ tokens, onRevoke }: IssuedTokensTableProps) {
  if (tokens.length === 0) {
    return <EmptyText>No tokens issued yet.</EmptyText>
  }

  return (
    <TableWrap>
      <Table>
        <thead>
          <tr>
            <th>Name</th>
            <th>Scope</th>
            <th>Hint</th>
            <th>Projects</th>
            <th>Tools</th>
            <th>Expires</th>
            <th>Last Used</th>
            <th>Status</th>
            <th>Action</th>
          </tr>
        </thead>
        <tbody>
          {tokens.map((token) => (
            <tr key={token.token_id}>
              <td>{token.name}</td>
              <td>{token.scope}</td>
              <td>
                <code>{token.token_secret_hint}</code>
              </td>
              <td>{token.allowed_project_ids.length > 0 ? token.allowed_project_ids.join(', ') : 'all visible'}</td>
              <td>{token.allowed_tools.length > 0 ? token.allowed_tools.join(', ') : 'scope defaults'}</td>
              <td>{formatTimestamp(token.expires_at)}</td>
              <td>{formatTimestamp(token.last_used_at)}</td>
              <td>{token.is_active ? STATUS_ACTIVE : 'inactive'}</td>
              <td>
                {token.revoked_at ? (
                  <button type="button" disabled>
                    Revoked
                  </button>
                ) : (
                  <button type="button" onClick={() => void onRevoke(token.token_id)}>
                    Revoke
                  </button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </Table>
    </TableWrap>
  )
}

export function AdminMcpTokenPanel({
  isOpen,
  loading,
  optionsLoading,
  creating,
  tokens,
  latestToken,
  error,
  availableTools,
  availableProjects,
  onClose,
  onRefresh,
  onCreate,
  onRevoke,
}: Props) {
  const activeCount = useMemo(() => tokens.filter((item) => item.is_active).length, [tokens])

  if (!isOpen) {
    return null
  }

  return (
    <Overlay role="dialog" aria-modal="true" aria-label="Admin MCP Tokens" data-testid="admin-mcp-token-panel">
      <Dialog>
        <Header>
          <div>
            <h2 className="font-display text-xl font-semibold tracking-tight text-ink">MCP Tokens</h2>
            <Hint>
              Admin-only workspace for issuing scoped tokens for external MCP clients. Token values are shown only
              once after creation.
            </Hint>
          </div>
          <HeaderActions>
            <button type="button" onClick={() => void onRefresh()} data-testid="admin-token-refresh">
              {loading ? 'Refreshing...' : 'Refresh'}
            </button>
            <button type="button" onClick={onClose}>
              Close
            </button>
          </HeaderActions>
        </Header>

        <Hint>
          Active tokens: <strong>{activeCount}</strong> / {tokens.length}
        </Hint>
        <Hint>
          Available options loaded: {optionsLoading ? 'loading...' : `${availableTools.length} tools, ${availableProjects.length} projects`}
        </Hint>

        {error ? <ErrorText>{error}</ErrorText> : null}

        {latestToken ? (
          <SuccessBox>
            <strong>New token created</strong>
            <TokenCode>{latestToken.token}</TokenCode>
            <Hint>Store this token now. It will not be retrievable from the server after this view refreshes.</Hint>
          </SuccessBox>
        ) : null}

        <TokenCreationForm
          creating={creating}
          availableTools={availableTools}
          availableProjects={availableProjects}
          onCreate={onCreate}
        />

        <h3 className="font-display text-lg font-semibold tracking-tight text-ink">Issued Tokens</h3>
        <IssuedTokensTable tokens={tokens} onRevoke={onRevoke} />
      </Dialog>
    </Overlay>
  )
}
