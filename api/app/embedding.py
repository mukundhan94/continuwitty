import hashlib


def embed_text_local(text: str, dim: int) -> list[float]:
    """
    Deterministic local embedding for MVP development.
    This is intentionally simple and provider-free.
    """
    if dim <= 0:
        raise ValueError("dim must be > 0")

    if not text:
        text = " "

    output: list[float] = []
    counter = 0
    while len(output) < dim:
        digest = hashlib.sha256(f"{text}:{counter}".encode()).digest()
        for byte in digest:
            value = (byte / 127.5) - 1.0
            output.append(value)
            if len(output) == dim:
                break
        counter += 1
    return output
