---
name: document-ingestion-rag
description: Use this skill when implementing or extending document ingestion, chunking, and retrieval blending with engram memory.
---

# Document Ingestion RAG

## Use This Skill When
- Adding text/file ingestion endpoints.
- Adjusting chunking strategy and deterministic chunk identifiers.
- Blending document retrieval with engram retrieval in chat/query flows.

## Workflow
1. Normalize text before hashing/chunking so IDs remain deterministic.
2. Generate content hash and deterministic document/chunk IDs.
3. Persist document metadata and replace chunk rows atomically.
4. Store chunk embeddings through `internal/embeddings` abstraction.
5. Blend document chunk results with engram retrieval for chat context.
6. Merge pinned-document context with normal retrieval where supported.
7. Emit source references with `document_id`, `chunk_id`, and `chunk_index`.

## Module Layout (Current)
- `internal/ingestion/chunking.go`: deterministic chunking/hash behavior.
- `internal/ingestion/service.go`: validation, decode guards, orchestration.
- `internal/repository/document*.go`: document/chunk persistence.
- `internal/api/ingestion_api.go`: ingestion REST routes.
- `internal/chat/context*.go`: merges document evidence into chat context.
- `internal/repository/chat_pinning*.go`: session pinned documents.
- `web/src/api/ingestion.ts`: frontend ingestion client.

## Validation
- Unit tests: `internal/ingestion/chunking_test.go`, `internal/ingestion/service_test.go`.
- Integration tests: `internal/api/ingestion_api_test.go`, repository tests.
- Chat-context tests: `internal/chat/context_test.go`, `internal/api/chat_api_messages*_test.go`.
- Run `go test ./... -count=1` and `make check`.
