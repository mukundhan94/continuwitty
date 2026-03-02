package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const (
	PBKDF2Algo       = "pbkdf2_sha256"
	PBKDF2Iterations = 390000
	passwordSaltSize = 16
	csrfTokenBytes   = 32
)

// GenerateCSRFToken returns a URL-safe random token.
func GenerateCSRFToken() (string, error) {
	randomBytes := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate csrf token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

// HashPassword creates a PBKDF2-SHA256 password hash in the Python-compatible format:
// pbkdf2_sha256$390000$<salt-hex>$<digest-hex>
func HashPassword(password string, salt []byte) (string, error) {
	if len(salt) == 0 {
		salt = make([]byte, passwordSaltSize)
		if _, err := rand.Read(salt); err != nil {
			return "", fmt.Errorf("generate salt: %w", err)
		}
	}

	digest := pbkdf2.Key([]byte(password), salt, PBKDF2Iterations, sha256.Size, sha256.New)
	return fmt.Sprintf(
		"%s$%d$%s$%s",
		PBKDF2Algo,
		PBKDF2Iterations,
		hex.EncodeToString(salt),
		hex.EncodeToString(digest),
	), nil
}

// VerifyPassword verifies a plaintext password against a PBKDF2 encoded hash.
func VerifyPassword(password, encodedHash string) bool {
	algo, iterations, salt, expectedDigest, ok := parsePasswordHash(encodedHash)
	if !ok {
		return false
	}
	if algo != PBKDF2Algo {
		return false
	}

	computedDigest := pbkdf2.Key([]byte(password), salt, iterations, sha256.Size, sha256.New)
	return hmac.Equal(computedDigest, expectedDigest)
}

func parsePasswordHash(encodedHash string) (string, int, []byte, []byte, bool) {
	parts := strings.SplitN(encodedHash, "$", 4)
	if len(parts) != 4 {
		return "", 0, nil, nil, false
	}

	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations <= 0 {
		return "", 0, nil, nil, false
	}

	salt, err := hex.DecodeString(parts[2])
	if err != nil {
		return "", 0, nil, nil, false
	}
	expectedDigest, err := hex.DecodeString(parts[3])
	if err != nil {
		return "", 0, nil, nil, false
	}
	return parts[0], iterations, salt, expectedDigest, true
}
