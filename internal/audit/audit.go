package audit

import (
	"bytes"
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
	defaultSinkTimeout = 2 * time.Second
)

// LoggerOptions configures the audit logger.
type LoggerOptions struct {
	Path          string
	StdoutEnabled bool
	MaxEventBytes int
	SinkURL       string
	SinkAuthToken string
	SinkRequired  bool
	SinkTimeout   time.Duration
	SinkClient    sinkHTTPClient
	Now           func() time.Time
}

type sinkHTTPClient interface {
	Do(request *http.Request) (*http.Response, error)
}

// Logger writes sanitized JSONL audit events.
type Logger struct {
	path          string
	stdoutEnabled bool
	maxEventBytes int
	sinkURL       string
	sinkAuthToken string
	sinkRequired  bool
	sinkClient    sinkHTTPClient
	now           func() time.Time

	mutex sync.Mutex
}

// RequestEvent captures one audit event write request.
type RequestEvent struct {
	Request   *http.Request
	EventType string
	Success   bool
	Username  string
	Detail    string
	Metadata  map[string]any
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
	sinkClient := options.SinkClient
	if sinkClient == nil {
		sinkTimeout := options.SinkTimeout
		if sinkTimeout <= 0 {
			sinkTimeout = defaultSinkTimeout
		}
		sinkClient = &http.Client{Timeout: sinkTimeout}
	}
	return &Logger{
		path:          strings.TrimSpace(options.Path),
		stdoutEnabled: options.StdoutEnabled,
		maxEventBytes: maxEventBytes,
		sinkURL:       strings.TrimSpace(options.SinkURL),
		sinkAuthToken: strings.TrimSpace(options.SinkAuthToken),
		sinkRequired:  options.SinkRequired,
		sinkClient:    sinkClient,
		now:           now,
	}
}

// LogRequestEvent appends a single event for an HTTP request.
func (logger *Logger) LogRequestEvent(event RequestEvent) error {
	if logger == nil || !logger.hasOutputDestination() {
		return nil
	}
	payload := logger.buildPayload(
		event.Request,
		event.EventType,
		event.Success,
		event.Username,
		event.Detail,
		event.Metadata,
	)
	serialized, err := serializePayload(payload, logger.maxEventBytes)
	if err != nil {
		return err
	}
	if err := logger.writeLocal(serialized); err != nil {
		return err
	}
	if logger.stdoutEnabled {
		fmt.Println(serialized)
	}
	if err := logger.sendToSink(serialized); err != nil && logger.sinkRequired {
		return err
	}
	return nil
}

func (logger *Logger) hasOutputDestination() bool {
	return logger.path != "" || logger.stdoutEnabled || logger.sinkURL != ""
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

func (logger *Logger) writeLocal(line string) error {
	if logger.path == "" {
		return nil
	}
	return logger.appendLine(line)
}

func (logger *Logger) sendToSink(line string) error {
	if logger.sinkURL == "" || logger.sinkClient == nil {
		return nil
	}
	request, err := http.NewRequest(http.MethodPost, logger.sinkURL, bytes.NewBufferString(line))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Engram-Audit-Format", "jsonl-event-v1")
	if logger.sinkAuthToken != "" {
		request.Header.Set("Authorization", "Bearer "+logger.sinkAuthToken)
	}
	response, err := logger.sinkClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("audit sink returned status %d", response.StatusCode)
	}
	return nil
}
