import pytest

from app.embedding import embed_text_local


def test_embed_text_local_is_deterministic_and_fixed_dim() -> None:
    first = embed_text_local("hello world", 16)
    second = embed_text_local("hello world", 16)

    assert first == second
    assert len(first) == 16
    assert all(-1.0 <= v <= 1.0 for v in first)


def test_embed_text_local_handles_empty_text() -> None:
    vec = embed_text_local("", 8)
    assert len(vec) == 8


def test_embed_text_local_rejects_invalid_dim() -> None:
    with pytest.raises(ValueError):
        embed_text_local("abc", 0)
