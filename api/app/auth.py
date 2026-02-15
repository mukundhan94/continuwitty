from __future__ import annotations

import hashlib
import hmac
import secrets

PBKDF2_ALGO = "pbkdf2_sha256"
PBKDF2_ITERATIONS = 390_000


def generate_csrf_token() -> str:
    return secrets.token_urlsafe(32)


def hash_password(password: str, salt: bytes | None = None) -> str:
    if salt is None:
        salt = secrets.token_bytes(16)
    digest = hashlib.pbkdf2_hmac(
        "sha256",
        password.encode(),
        salt,
        PBKDF2_ITERATIONS,
    )
    return f"{PBKDF2_ALGO}${PBKDF2_ITERATIONS}${salt.hex()}${digest.hex()}"


def verify_password(password: str, encoded_hash: str) -> bool:
    try:
        algo, iterations_text, salt_hex, digest_hex = encoded_hash.split("$", 3)
    except ValueError:
        return False

    if algo != PBKDF2_ALGO:
        return False

    try:
        iterations = int(iterations_text)
        salt = bytes.fromhex(salt_hex)
        expected_digest = bytes.fromhex(digest_hex)
    except ValueError:
        return False

    computed_digest = hashlib.pbkdf2_hmac(
        "sha256",
        password.encode(),
        salt,
        iterations,
    )
    return hmac.compare_digest(computed_digest, expected_digest)
