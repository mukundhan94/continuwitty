package audit

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	maxTextFieldLength = 1_024
	maxMetadataItems   = 50
	maxMetadataDepth   = 4
	minMaxEventBytes   = 1_024
	defaultMaxEventB   = 32_768
)

// LoggerOptions configures the audit logger.
type LoggerOptions struct {
	Path          string
	StdoutEnabled bool
	MaxEventBytes int
	Now           func() time.Time
}

// Logger writes sanitized JSONL audit events.
type Logger struct {
	path          string
	stdoutEnabled bool
	maxEventBytes int
	now           func() time.Time

	mutex sync.Mutex
}

// NewLogger creates a request audit logger.
func NewLogger(options LoggerOptions) *Logger {
	now := options.Now
	if now == nil {
		now = time.Now
	}
	maxEventBytes := options.MaxEventBytes
	if maxEventBytes == 0 {
		maxEventBytes = defaultMaxEventB
	}
	if maxEventBytes < minMaxEventBytes {
		maxEventBytes = minMaxEventBytes
	}
	return &Logger{
		path:          strings.TrimSpace(options.Path),
		stdoutEnabled: options.StdoutEnabled,
		maxEventBytes: maxEventBytes,
		now:           now,
	}
}

// LogRequestEvent appends a single event for an HTTP request.
func (logger *Logger) LogRequestEvent(
	request *http.Request,
	eventType string,
	success bool,
	username string,
	detail string,
	metadata map[string]any,
) error {
	if logger == nil || logger.path == "" {
		return nil
	}
	payload := logger.buildPayload(request, eventType, success, username, detail, metadata)
	serialized, err := serializePayload(payload, logger.maxEventBytes)
	if err != nil {
		return err
	}
	if err := logger.appendLine(serialized); err != nil {
		return err
	}
	if logger.stdoutEnabled {
		fmt.Println(serialized)
	}
	return nil
}

func (logger *Logger) buildPayload(
	request *http.Request,
	eventType string,
	success bool,
	username string,
	detail string,
	metadata map[string]any,
) map[string]any {
	payload := map[string]any{
		"timestamp":  logger.now().UTC().Format(time.RFC3339Nano),
		"event_type": eventType,
		"success":    success,
		"ip":         clientIP(request),
		"method":     requestMethod(request),
		"path":       requestPath(request),
	}
	if cleanUsername := sanitizeText(username, maxTextFieldLength); cleanUsername != "" {
		payload["username"] = cleanUsername
	}
	if cleanDetail := sanitizeText(detail, maxTextFieldLength); cleanDetail != "" {
		payload["detail"] = cleanDetail
	}
	if len(metadata) > 0 {
		payload["metadata"] = sanitizeMetadataMap(metadata, 0)
	}
	return payload
}

func requestMethod(request *http.Request) string {
	if request == nil {
		return "UNKNOWN"
	}
	return request.Method
}

func requestPath(request *http.Request) string {
	if request == nil || request.URL == nil {
		return ""
	}
	return request.URL.Path
}

func clientIP(request *http.Request) string {
	if request == nil {
		return "unknown"
	}
	remoteAddress := strings.TrimSpace(request.RemoteAddr)
	if remoteAddress == "" {
		return "unknown"
	}
	host, _, err := net.SplitHostPort(remoteAddress)
	if err == nil && strings.TrimSpace(host) != "" {
		return host
	}
	return remoteAddress
}

func sanitizeText(value string, maxLength int) string {
	normalized := strings.Join(strings.Fields(value), " ")
	if normalized == "" {
		return ""
	}
	if maxLength < 4 || len(normalized) <= maxLength {
		return normalized
	}
	return normalized[:maxLength-3] + "..."
}

func sanitizeMetadataMap(values map[string]any, depth int) map[string]any {
	if depth >= maxMetadataDepth {
		return map[string]any{"truncated": true}
	}
	sanitized := map[string]any{}
	count := 0
	for key, value := range values {
		if count >= maxMetadataItems {
			sanitized["truncated"] = true
			break
		}
		sanitized[key] = sanitizeMetadataValue(value, depth+1)
		count++
	}
	return sanitized
}

func sanitizeMetadataList(values []any, depth int) []any {
	if depth >= maxMetadataDepth {
		return []any{"<truncated-depth>"}
	}
	limit := len(values)
	truncated := false
	if limit > maxMetadataItems {
		limit = maxMetadataItems
		truncated = true
	}
	sanitized := make([]any, 0, limit+1)
	for index := 0; index < limit; index++ {
		sanitized = append(sanitized, sanitizeMetadataValue(values[index], depth+1))
	}
	if truncated {
		sanitized = append(sanitized, "<truncated-list>")
	}
	return sanitized
}

func sanitizeMetadataValue(value any, depth int) any {
	if depth >= maxMetadataDepth {
		return "<truncated-depth>"
	}
	switch typedValue := value.(type) {
	case nil, bool, int, int32, int64, uint, uint32, uint64, float32, float64:
		return typedValue
	case string:
		return sanitizeText(typedValue, maxTextFieldLength)
	case map[string]any:
		return sanitizeMetadataMap(typedValue, depth)
	case []any:
		return sanitizeMetadataList(typedValue, depth)
	default:
		return sanitizeText(fmt.Sprint(value), maxTextFieldLength)
	}
}

func serializePayload(payload map[string]any, maxEventBytes int) (string, error) {
	serialized, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	if len(serialized) <= maxEventBytes {
		return string(serialized), nil
	}
	reduced := copyPayloadMap(payload)
	reduced["detail"] = "audit payload exceeded max size and was truncated"
	reduced["metadata"] = map[string]any{"truncated": true}
	serialized, err = json.Marshal(reduced)
	if err != nil {
		return "", err
	}
	return string(serialized), nil
}

func copyPayloadMap(payload map[string]any) map[string]any {
	copied := make(map[string]any, len(payload))
	for key, value := range payload {
		copied[key] = value
	}
	return copied
}

func (logger *Logger) appendLine(line string) error {
	logger.mutex.Lock()
	defer logger.mutex.Unlock()
	auditPath := filepath.Clean(logger.path)
	if err := os.MkdirAll(filepath.Dir(auditPath), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(auditPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(line + "\n")
	return err
}
