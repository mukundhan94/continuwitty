import { afterEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from './http'
import {
  ingestFileDocument,
  ingestTextDocument,
  listProjectDocuments,
} from './ingestion'

function mockResponse(params: {
  ok: boolean
  status: number
  statusText?: string
  contentType?: string
  jsonBody?: unknown
  textBody?: string
}): Response {
  return {
    ok: params.ok,
    status: params.status,
    statusText: params.statusText ?? '',
    headers: new Headers(
      params.contentType
        ? {
            'content-type': params.contentType,
          }
        : {},
    ),
    json: async () => params.jsonBody ?? {},
    text: async () => params.textBody ?? '',
  } as Response
}

function documentRecord(documentId: string) {
  return {
    document_id: documentId,
    owner_user_id: 'user-1',
    project_id: 'engram-vault',
    title: 'Incident doc',
    source_type: 'text' as const,
    source_name: null,
    mime_type: null,
    visibility_scope: 'private' as const,
    content_hash: 'hash-1',
    chunk_count: 2,
    created_at: '2026-02-20T00:00:00Z',
    updated_at: '2026-02-20T00:00:00Z',
  }
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('ingestion api', () => {
  it('builds project document query parameters', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({
        ok: true,
        status: 200,
        contentType: 'application/json',
        jsonBody: [],
      }),
    )

    await listProjectDocuments('engram-vault')

    const [path] = fetchMock.mock.calls[0]
    const resolvedPath = String(path)
    expect(resolvedPath).toContain('/api/v1/ingestion/documents?')
    expect(resolvedPath).toContain('project_id=engram-vault')
    expect(resolvedPath).toContain('limit=200')
    expect(resolvedPath).toContain('offset=0')
  })

  it('posts text ingestion payload and returns document', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({
        ok: true,
        status: 200,
        contentType: 'application/json',
        jsonBody: { document: documentRecord('doc-text-1') },
      }),
    )

    const result = await ingestTextDocument({
      project_id: 'engram-vault',
      title: 'Timeline',
      text: 'Investigated incident timeline.',
      visibility_scope: 'private',
      chunk_size_chars: 800,
      chunk_overlap_chars: 120,
      metadata: { source: 'postmortem' },
    })

    expect(result.document_id).toBe('doc-text-1')
    const [, init] = fetchMock.mock.calls[0]
    const requestInit = init as RequestInit
    expect(requestInit.method).toBe('POST')
    expect(requestInit.body).toContain('"title":"Timeline"')
    expect(requestInit.body).toContain('"source":"postmortem"')
  })

  it('builds multipart payload for file ingestion with trimmed title and metadata', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({
        ok: true,
        status: 200,
        contentType: 'application/json',
        jsonBody: { document: documentRecord('doc-file-1') },
      }),
    )
    const file = new File(['line one'], 'incident.txt', { type: 'text/plain' })

    const result = await ingestFileDocument({
      project_id: 'engram-vault',
      title: '  Incident Report  ',
      visibility_scope: 'project',
      chunk_size_chars: 900,
      chunk_overlap_chars: 100,
      metadata: { source: 'upload' },
      file,
    })

    expect(result.document_id).toBe('doc-file-1')
    const [, init] = fetchMock.mock.calls[0]
    const requestInit = init as RequestInit
    expect(requestInit.method).toBe('POST')
    expect(requestInit.credentials).toBe('include')
    const body = requestInit.body as FormData
    expect(body.get('project_id')).toBe('engram-vault')
    expect(body.get('visibility_scope')).toBe('project')
    expect(body.get('chunk_size_chars')).toBe('900')
    expect(body.get('chunk_overlap_chars')).toBe('100')
    expect(body.get('title')).toBe('Incident Report')
    expect(body.get('metadata_json')).toBe('{"source":"upload"}')
    expect(body.get('file')).toBe(file)
  })

  it('omits blank title and surfaces API error for file ingestion failure', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({
        ok: false,
        status: 422,
        statusText: 'Unprocessable Entity',
        contentType: 'text/plain',
        textBody: 'file too large',
      }),
    )
    const file = new File(['text'], 'large.txt', { type: 'text/plain' })

    await expect(
      ingestFileDocument({
        project_id: 'engram-vault',
        title: '   ',
        visibility_scope: 'private',
        chunk_size_chars: 500,
        chunk_overlap_chars: 50,
        file,
      }),
    ).rejects.toBeInstanceOf(ApiError)
    await expect(
      ingestFileDocument({
        project_id: 'engram-vault',
        title: '   ',
        visibility_scope: 'private',
        chunk_size_chars: 500,
        chunk_overlap_chars: 50,
        file,
      }),
    ).rejects.toMatchObject({
      status: 422,
      detail: 'file too large',
    })
  })
})
