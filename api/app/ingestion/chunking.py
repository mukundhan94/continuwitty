from __future__ import annotations

import hashlib
import re
from dataclasses import dataclass
from uuid import UUID, uuid5

_DOCUMENT_NAMESPACE = UUID("6e5bc7c9-4930-4d8a-a62e-d5bdabed95e3")
_CHUNK_NAMESPACE = UUID("5ecdb89d-6222-4fbe-8ef2-9298dc5d8f0e")
_MIN_BREAK_FRACTION = 0.6
_SPACE_PATTERN = re.compile(r"\s+")


@dataclass(frozen=True)
class ChunkDraft:
    chunk_id: UUID
    chunk_index: int
    chunk_text: str
    snippet: str
    char_start: int
    char_end: int
    token_estimate: int
    metadata: dict


def normalize_document_text(text: str) -> str:
    """Normalize whitespace/newlines to keep hashing and chunk IDs deterministic."""

    normalized = text.replace("\r\n", "\n").replace("\r", "\n").strip()
    return normalized


def build_content_hash(text: str) -> str:
    normalized = normalize_document_text(text)
    return hashlib.sha256(normalized.encode("utf-8")).hexdigest()


def build_document_id(owner_user_id: UUID, project_id: str, title: str, content_hash: str) -> UUID:
    fingerprint = f"{owner_user_id}:{project_id}:{title.strip().lower()}:{content_hash}"
    return uuid5(_DOCUMENT_NAMESPACE, fingerprint)


def estimate_token_count(text: str) -> int:
    if not text.strip():
        return 0
    return len(_SPACE_PATTERN.split(text.strip()))


def _pick_chunk_end(text: str, start: int, hard_end: int, chunk_size_chars: int) -> int:
    if hard_end >= len(text):
        return len(text)

    search_start = start + int(chunk_size_chars * _MIN_BREAK_FRACTION)
    if search_start >= hard_end:
        return hard_end

    candidates = [
        text.rfind("\n\n", search_start, hard_end),
        text.rfind("\n", search_start, hard_end),
        text.rfind(" ", search_start, hard_end),
    ]
    chosen = max(candidates)
    if chosen <= start:
        return hard_end

    if text[chosen : chosen + 2] == "\n\n":
        return chosen
    return chosen + 1


def chunk_document_text(
    *,
    content_hash: str,
    text: str,
    chunk_size_chars: int,
    chunk_overlap_chars: int,
    snippet_chars: int = 220,
) -> list[ChunkDraft]:
    """Build deterministic chunk boundaries and UUIDs for persisted retrieval chunks."""

    normalized = normalize_document_text(text)
    if not normalized:
        return []

    start = 0
    chunk_index = 0
    chunks: list[ChunkDraft] = []

    while start < len(normalized):
        hard_end = min(start + chunk_size_chars, len(normalized))
        end = _pick_chunk_end(normalized, start, hard_end, chunk_size_chars)

        chunk_text = normalized[start:end].strip()
        if chunk_text:
            chunk_id = uuid5(_CHUNK_NAMESPACE, f"{content_hash}:{chunk_index}:{start}:{end}")
            snippet = chunk_text[:snippet_chars].replace("\n", " ").strip()
            chunks.append(
                ChunkDraft(
                    chunk_id=chunk_id,
                    chunk_index=chunk_index,
                    chunk_text=chunk_text,
                    snippet=snippet,
                    char_start=start,
                    char_end=end,
                    token_estimate=estimate_token_count(chunk_text),
                    metadata={
                        "content_hash": content_hash,
                        "char_start": start,
                        "char_end": end,
                        "chunk_size_chars": chunk_size_chars,
                        "chunk_overlap_chars": chunk_overlap_chars,
                    },
                )
            )
            chunk_index += 1

        if end >= len(normalized):
            break

        next_start = max(0, end - chunk_overlap_chars)
        if next_start <= start:
            next_start = end
        start = next_start

    return chunks
