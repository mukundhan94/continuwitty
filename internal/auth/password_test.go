package auth

import (
	"regexp"
	"strings"
	"testing"
)

const bootstrapAdminHash = "pbkdf2_sha256$390000$00112233445566778899aabbccddeeff$45c0bdc16f1609a69d14a4e8d89974fd24556c45af75ee01c34bdb9de1c1832a"

func TestHashPasswordWithProvidedSaltProducesExpectedFormat(t *testing.T) {
	hash, err := HashPassword("admin123", []byte{0x00, 0x11, 0x22, 0x33})
	if err != nil {
		t.Fatalf("expected hashing to succeed: %v", err)
	}

	parts := strings.Split(hash, "$")
	if len(parts) != 4 {
		t.Fatalf("expected four hash parts, got %d", len(parts))
	}
	if parts[0] != PBKDF2Algo {
		t.Fatalf("expected algo %q, got %q", PBKDF2Algo, parts[0])
	}
	if parts[1] != "390000" {
		t.Fatalf("expected iterations 390000, got %q", parts[1])
	}
	if parts[2] != "00112233" {
		t.Fatalf("expected provided salt hex, got %q", parts[2])
	}
	if len(parts[3]) != 64 {
		t.Fatalf("expected sha256 digest hex length 64, got %d", len(parts[3]))
	}
}

func TestVerifyPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("StrongPassword-12345", nil)
	if err != nil {
		t.Fatalf("expected hashing to succeed: %v", err)
	}

	if !VerifyPassword("StrongPassword-12345", hash) {
		t.Fatalf("expected password verification to succeed")
	}
	if VerifyPassword("wrong-password", hash) {
		t.Fatalf("expected password verification to fail for wrong password")
	}
}

func TestVerifyPasswordMatchesBootstrapAdminHash(t *testing.T) {
	if !VerifyPassword("admin123", bootstrapAdminHash) {
		t.Fatalf("expected bootstrap hash to verify for admin123")
	}
	if VerifyPassword("admin", bootstrapAdminHash) {
		t.Fatalf("expected bootstrap hash mismatch for wrong password")
	}
}

func TestVerifyPasswordRejectsInvalidEncodings(t *testing.T) {
	invalid := []string{
		"",
		"not-a-hash",
		"pbkdf2_sha256$390000$nothex$also-not-hex",
		"pbkdf2_sha256$bad-iterations$001122$deadbeef",
		"other_algo$390000$001122$deadbeef",
	}
	for _, encoded := range invalid {
		if VerifyPassword("password", encoded) {
			t.Fatalf("expected encoded hash %q to fail verification", encoded)
		}
	}
}

func TestGenerateCSRFTokenIsURLSafeAndRandom(t *testing.T) {
	first, err := GenerateCSRFToken()
	requireNoError(t, err)
	second, err := GenerateCSRFToken()
	requireNoError(t, err)
	requireNotEmpty(t, first)
	requireNotEmpty(t, second)

	re := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	requireMatch(t, re, first)
	requireMatch(t, re, second)
	requireNoPadding(t, first)
	requireNoPadding(t, second)
	requireDifferent(t, first, second)
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected operation to succeed: %v", err)
	}
}

func requireNotEmpty(t *testing.T, value string) {
	t.Helper()
	if value == "" {
		t.Fatalf("expected value to be non-empty")
	}
}

func requireMatch(t *testing.T, pattern *regexp.Regexp, value string) {
	t.Helper()
	if !pattern.MatchString(value) {
		t.Fatalf("expected value to match pattern")
	}
}

func requireNoPadding(t *testing.T, value string) {
	t.Helper()
	if strings.Contains(value, "=") {
		t.Fatalf("expected token to have no base64 padding")
	}
}

func requireDifferent(t *testing.T, first string, second string) {
	t.Helper()
	if first == second {
		t.Fatalf("expected csrf tokens to differ")
	}
}
