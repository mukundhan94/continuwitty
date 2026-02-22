package audit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogRequestEventWritesJSONLine(t *testing.T) {
	auditPath := filepath.Join(t.TempDir(), "audit.log")
	logger := NewLogger(LoggerOptions{
		Path:          auditPath,
		MaxEventBytes: 32_768,
		Now: func() time.Time {
			return time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
		},
	})
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("username=admin"))
	request.RemoteAddr = "203.0.113.7:4321"

	if err := logger.LogRequestEvent(
		request,
		"login_failed",
		false,
		"admin",
		"invalid username/password",
		map[string]any{"attempt": 1},
	); err != nil {
		t.Fatalf("expected audit event logging to succeed: %v", err)
	}

	content, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("expected audit log file read to succeed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected one log line, got %d", len(lines))
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &payload); err != nil {
		t.Fatalf("expected valid json log line: %v", err)
	}
	if payload["event_type"] != "login_failed" {
		t.Fatalf("expected event_type login_failed, got %#v", payload["event_type"])
	}
	if payload["ip"] != "203.0.113.7" {
		t.Fatalf("expected ip 203.0.113.7, got %#v", payload["ip"])
	}
	if payload["path"] != "/login" {
		t.Fatalf("expected path /login, got %#v", payload["path"])
	}
}

func TestLogRequestEventTruncatesOversizedPayload(t *testing.T) {
	auditPath := filepath.Join(t.TempDir(), "audit.log")
	logger := NewLogger(LoggerOptions{
		Path:          auditPath,
		MaxEventBytes: 1_024,
	})
	request := httptest.NewRequest(http.MethodPost, "/login", nil)
	oversized := strings.Repeat("x", 10_000)

	if err := logger.LogRequestEvent(
		request,
		"login_failed",
		false,
		"admin",
		oversized,
		map[string]any{"oversized": oversized},
	); err != nil {
		t.Fatalf("expected audit event logging to succeed: %v", err)
	}

	content, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("expected audit log file read to succeed: %v", err)
	}
	if !strings.Contains(string(content), "\"truncated\":true") {
		t.Fatalf("expected truncated marker in audit payload")
	}
}
