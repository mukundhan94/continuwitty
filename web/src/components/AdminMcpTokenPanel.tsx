import { useEffect, useMemo, useState } from 'react'
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
  const [name, setName] = useState('')
  const [scope, setScope] = useState<McpTokenScope>('read')
  const [expiresInDays, setExpiresInDays] = useState(90)
  const [selectedTools, setSelectedTools] = useState<string[]>([])
  const [selectedProjects, setSelectedProjects] = useState<string[]>([])
  const [pendingTool, setPendingTool] = useState('')
  const [pendingProject, setPendingProject] = useState('')

  const activeCount = useMemo(() => tokens.filter((item) => item.is_active).length, [tokens])

  useEffect(() => {
    if (!isOpen) {
      return
    }
    if (!pendingTool && availableTools.length > 0) {
      setPendingTool(availableTools[0])
    }
    if (!pendingProject && availableProjects.length > 0) {
      setPendingProject(availableProjects[0])
    }
  }, [availableProjects, availableTools, isOpen, pendingProject, pendingTool])

  if (!isOpen) {
    return null
  }

  const addTool = () => {
    if (!pendingTool || selectedTools.includes(pendingTool)) {
      return
    }
    setSelectedTools((current) => [...current, pendingTool])
    const nextCandidate = availableTools.find((item) => item !== pendingTool && !selectedTools.includes(item))
    setPendingTool(nextCandidate || pendingTool)
  }

  const addProject = () => {
    if (!pendingProject || selectedProjects.includes(pendingProject)) {
      return
    }
    setSelectedProjects((current) => [...current, pendingProject])
    const nextCandidate = availableProjects.find(
      (item) => item !== pendingProject && !selectedProjects.includes(item),
    )
    setPendingProject(nextCandidate || pendingProject)
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
                onChange={(event) => setExpiresInDays(Number(event.target.value || 90))}
              />
            </Field>
          </Grid>

          <Field>
            Allowed Tools (optional)
            <OptionSelectRow>
              <Select
                aria-label="Tool Options"
                data-testid="admin-tool-options"
                value={pendingTool}
                onChange={(event) => setPendingTool(event.target.value)}
                disabled={availableTools.length === 0}
              >
                {availableTools.length === 0 ? (
                  <option value="">No tools available</option>
                ) : (
                  availableTools.map((toolName) => (
                    <option key={toolName} value={toolName}>
                      {toolName}
                    </option>
                  ))
                )}
              </Select>
              <button type="button" onClick={addTool} disabled={!pendingTool} data-testid="admin-add-tool-chip">
                Add Tool
              </button>
            </OptionSelectRow>
            <ChipWrap>
              {selectedTools.length === 0 ? (
                <Hint>No tool restrictions selected.</Hint>
              ) : (
                selectedTools.map((toolName) => (
                  <Chip key={toolName}>
                    {toolName}
                    <button
                      type="button"
                      aria-label={`Remove tool ${toolName}`}
                      onClick={() => setSelectedTools((current) => current.filter((item) => item !== toolName))}
                    >
                      ×
                    </button>
                  </Chip>
                ))
              )}
            </ChipWrap>
          </Field>

          <Field>
            Allowed Project IDs (optional)
            <OptionSelectRow>
              <Select
                aria-label="Project Options"
                data-testid="admin-project-options"
                value={pendingProject}
                onChange={(event) => setPendingProject(event.target.value)}
                disabled={availableProjects.length === 0}
              >
                {availableProjects.length === 0 ? (
                  <option value="">No projects available</option>
                ) : (
                  availableProjects.map((projectName) => (
                    <option key={projectName} value={projectName}>
                      {projectName}
                    </option>
                  ))
                )}
              </Select>
              <button
                type="button"
                onClick={addProject}
                disabled={!pendingProject}
                data-testid="admin-add-project-chip"
              >
                Add Project
              </button>
            </OptionSelectRow>
            <ChipWrap>
              {selectedProjects.length === 0 ? (
                <Hint>No project restrictions selected.</Hint>
              ) : (
                selectedProjects.map((projectName) => (
                  <Chip key={projectName}>
                    {projectName}
                    <button
                      type="button"
                      aria-label={`Remove project ${projectName}`}
                      onClick={() =>
                        setSelectedProjects((current) => current.filter((item) => item !== projectName))
                      }
                    >
                      ×
                    </button>
                  </Chip>
                ))
              )}
            </ChipWrap>
          </Field>

          <button type="submit" disabled={creating}>
            {creating ? 'Creating...' : 'Create Token'}
          </button>
        </FormGrid>

        <h3 className="font-display text-lg font-semibold tracking-tight text-ink">Issued Tokens</h3>
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
              {tokens.length === 0 ? null :
                tokens.map((token) => (
                  <tr key={token.token_id}>
                    <td>{token.name}</td>
                    <td>{token.scope}</td>
                    <td>
                      <code>{token.token_secret_hint}</code>
                    </td>
                    <td>{token.allowed_project_ids.length > 0 ? token.allowed_project_ids.join(', ') : 'all visible'}</td>
                    <td>{token.allowed_tools.length > 0 ? token.allowed_tools.join(', ') : 'scope defaults'}</td>
                    <td>{new Date(token.expires_at).toLocaleString()}</td>
                    <td>{token.last_used_at ? new Date(token.last_used_at).toLocaleString() : '-'}</td>
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

        {tokens.length === 0 ? <EmptyText>No tokens issued yet.</EmptyText> : null}
      </Dialog>
    </Overlay>
  )
}
