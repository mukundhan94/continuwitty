package audit

import (
	"encoding/json"
	"io"
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
		RequestEvent{
			Request:   request,
			EventType: "login_failed",
			Success:   false,
			Username:  "admin",
			Detail:    "invalid username/password",
			Metadata:  map[string]any{"attempt": 1},
		},
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
		RequestEvent{
			Request:   request,
			EventType: "login_failed",
			Success:   false,
			Username:  "admin",
			Detail:    oversized,
			Metadata:  map[string]any{"oversized": oversized},
		},
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

func TestLogRequestEventPostsToConfiguredSink(t *testing.T) {
	receivedAuth := ""
	receivedBody := ""
	sink := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		receivedAuth = request.Header.Get("Authorization")
		payload, _ := io.ReadAll(request.Body)
		receivedBody = string(payload)
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer sink.Close()

	logger := NewLogger(LoggerOptions{
		SinkURL:       sink.URL,
		SinkAuthToken: "test-token",
		SinkRequired:  true,
	})
	request := httptest.NewRequest(http.MethodPost, "/login", nil)
	if err := logger.LogRequestEvent(
		RequestEvent{
			Request:   request,
			EventType: "login_success",
			Success:   true,
			Username:  "admin",
			Detail:    "ok",
			Metadata:  map[string]any{"channel": "ui"},
		},
	); err != nil {
		t.Fatalf("expected sink emit success, got %v", err)
	}

	if receivedAuth != "Bearer test-token" {
		t.Fatalf("expected bearer auth header to be forwarded")
	}
	if !strings.Contains(receivedBody, "\"event_type\":\"login_success\"") {
		t.Fatalf("expected serialized audit payload in sink body, got %q", receivedBody)
	}
}

func TestLogRequestEventSinkFailureIsFailOpenWhenNotRequired(t *testing.T) {
	sink := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer sink.Close()

	logger := NewLogger(LoggerOptions{
		SinkURL:      sink.URL,
		SinkRequired: false,
	})
	request := httptest.NewRequest(http.MethodPost, "/login", nil)
	if err := logger.LogRequestEvent(
		RequestEvent{
			Request:   request,
			EventType: "login_failed",
			Success:   false,
			Username:  "admin",
			Detail:    "bad",
		},
	); err != nil {
		t.Fatalf("expected fail-open sink behavior, got %v", err)
	}
}

func TestLogRequestEventSinkFailureReturnsErrorWhenRequired(t *testing.T) {
	sink := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer sink.Close()

	logger := NewLogger(LoggerOptions{
		SinkURL:      sink.URL,
		SinkRequired: true,
	})
	request := httptest.NewRequest(http.MethodPost, "/login", nil)
	if err := logger.LogRequestEvent(
		RequestEvent{
			Request:   request,
			EventType: "login_failed",
			Success:   false,
			Username:  "admin",
			Detail:    "bad",
		},
	); err == nil {
		t.Fatalf("expected required sink failures to be returned")
	}
}
