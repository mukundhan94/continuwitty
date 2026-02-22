import { afterEach, describe, expect, it, vi } from 'vitest'

import { exportProjectBundle, importProjectBundle } from './export'

function mockJsonResponse(body: unknown): Response {
  return {
    ok: true,
    status: 200,
    statusText: 'OK',
    headers: new Headers({ 'content-type': 'application/json' }),
    json: async () => body,
  } as Response
}

function mockBlobResponse(options: {
  body: string
  contentType?: string
  contentDisposition?: string
}): Response {
  const headers = new Headers({
    'content-type': options.contentType ?? 'application/json',
  })
  if (options.contentDisposition) {
    headers.set('content-disposition', options.contentDisposition)
  }
  return {
    ok: true,
    status: 200,
    statusText: 'OK',
    headers,
    blob: async () => new Blob([options.body], { type: options.contentType ?? 'application/json' }),
  } as Response
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('export api', () => {
  it('builds project export query with collection ids and include embeddings', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockBlobResponse({
        body: '{"schema_version":"1.0"}',
        contentDisposition: 'attachment; filename="phase33-export.json"',
      }),
    )

    const payload = await exportProjectBundle({
      projectId: 'engram-vault',
      format: 'json',
      includeEmbeddings: true,
      collectionIds: ['col-1', 'col-2'],
    })

    expect(payload.filename).toBe('phase33-export.json')
    const [path] = fetchMock.mock.calls[0]
    const resolved = String(path)
    expect(resolved).toContain('/api/v1/projects/engram-vault/export?')
    expect(resolved).toContain('format=json')
    expect(resolved).toContain('include_embeddings=true')
    expect(resolved).toContain('collection_ids=col-1')
    expect(resolved).toContain('collection_ids=col-2')
  })

  it('uses default filename when response header is missing', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockBlobResponse({
        body: '{"schema_version":"1.0"}',
      }),
    )

    const payload = await exportProjectBundle({
      projectId: 'my project/alpha',
      format: 'zip',
    })

    expect(payload.filename).toBe('engram-export-my-project-alpha.zip')
  })

  it('posts multipart import payload with conflict policy', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockJsonResponse({
        target_project_id: 'phase33-target',
        imported_engrams: 2,
        skipped_engrams: 0,
        overwritten_engrams: 0,
        imported_collections: 1,
        reused_collections: 0,
        imported_collection_items: 2,
        conflict_policy: 'rename',
      }),
    )

    const file = new File(['{}'], 'export.json', { type: 'application/json' })
    const payload = await importProjectBundle({
      projectId: 'phase33-target',
      file,
      conflictPolicy: 'rename',
    })

    expect(payload.target_project_id).toBe('phase33-target')
    const [path, init] = fetchMock.mock.calls[0]
    expect(String(path)).toContain('/api/v1/projects/phase33-target/import?conflict_policy=rename')
    expect((init as RequestInit).method).toBe('POST')
    expect((init as RequestInit).body instanceof FormData).toBe(true)
  })
})
