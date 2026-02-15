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
1. Normalize text before hashing/chunking to keep deterministic IDs stable.
2. Generate content hash and deterministic document/chunk IDs.
3. Persist document metadata and replace chunk rows atomically.
4. Store chunk embeddings via embedding abstraction (never call provider SDK from repository code).
5. Blend document chunk results with engram retrieval for downstream chat context.
6. Emit source references that identify chunk provenance (`document_id`, `chunk_id`, `chunk_index`).

## Module Layout (Current)
- `api/app/ingestion/chunking.py`: deterministic text normalization/hash/chunking.
- `api/app/ingestion/repository.py`: document/chunk persistence and vector query.
- `api/app/ingestion/service.py`: validation, file decode guards, blended query orchestration.
- `api/app/ingestion/api.py`: HTTP routes for text/file intake and retrieval.
- `api/app/embeddings/`: embedding provider routing and fallback behavior.

## Validation
- Unit tests for chunking determinism and ingestion service validation.
- Integration tests for text/file ingestion + blended query path.
- Chat context tests covering document chunk inclusion and source references.
