package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"engram/internal/mcp"

	"github.com/go-chi/chi/v5"
)

// MountMCPRoutes registers MCP stream transport endpoints.
func MountMCPRoutes(
	router chi.Router,
	service mcp.Service,
	actorResolver mcp.HTTPActorResolver,
	transportRateLimiter MCPTransportRateLimiter,
) {
	deps := mcpRouteDependencies{
		service:              service,
		actorResolver:        actorResolver,
		transportRateLimiter: transportRateLimiter,
	}
	router.Route("/api/v1/mcp", func(mcpRouter chi.Router) {
		mcpRouter.Get("/stream", deps.handleProbe)
		mcpRouter.Head("/stream", deps.handleProbeHead)
		mcpRouter.Post("/stream", deps.handleStream)
	})
}

// MCPTransportRateLimiter captures request-burst limiting behavior for MCP transport routes.
type MCPTransportRateLimiter interface {
	Consume(key string) (bool, int)
}

type mcpRouteDependencies struct {
	service              mcp.Service
	actorResolver        mcp.HTTPActorResolver
	transportRateLimiter MCPTransportRateLimiter
}

func (dependencies mcpRouteDependencies) handleProbe(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{
		"name":           "engram-mcp-stream",
		"transport":      "sse-jsonrpc",
		"endpoint":       "/api/v1/mcp/stream",
		"request_method": "POST",
		"response_type":  "text/event-stream",
	})
}

func (dependencies mcpRouteDependencies) handleProbeHead(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusOK)
}

func (dependencies mcpRouteDependencies) handleStream(writer http.ResponseWriter, request *http.Request) {
	if dependencies.service == nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "MCP service is not configured"})
		return
	}
	payload, ok := decodeMCPJSONRPCPayload(writer, request)
	if !ok {
		return
	}
	if err := dependencies.enforceTransportRateLimit(request); err != nil {
		writeMCPRouteError(writer, err)
		return
	}
	if payload.IsNotification() {
		dependencies.handleNotificationStreamRequest(writer, request, payload)
		return
	}
	dependencies.handleCallStreamRequest(writer, request, payload)
}

func decodeMCPJSONRPCPayload(
	writer http.ResponseWriter,
	request *http.Request,
) (mcp.JSONRPCRequest, bool) {
	payload := mcp.JSONRPCRequest{}
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "Invalid payload"})
		return mcp.JSONRPCRequest{}, false
	}
	return payload, true
}

func (dependencies mcpRouteDependencies) handleNotificationStreamRequest(
	writer http.ResponseWriter,
	request *http.Request,
	payload mcp.JSONRPCRequest,
) {
	if _, err := dependencies.resolveActor(request); err != nil {
		writeMCPRouteError(writer, err)
		return
	}
	if err := dependencies.service.HandleNotification(request.Context(), payload); err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "Internal server error"})
		return
	}
	writer.WriteHeader(http.StatusAccepted)
}

func (dependencies mcpRouteDependencies) handleCallStreamRequest(
	writer http.ResponseWriter,
	request *http.Request,
	payload mcp.JSONRPCRequest,
) {
	resolvedActor, err := dependencies.resolveActor(request)
	if err != nil {
		writeMCPRouteError(writer, err)
		return
	}
	frames := dependencies.service.StreamCall(request.Context(), mcp.StreamCallRequest{
		Request:   payload,
		Actor:     resolvedActor.Actor,
		TokenAuth: resolvedActor.TokenAuth,
	})
	if prefersSSE(request.Header.Get("accept")) {
		writeSSEFrames(writer, frames)
		return
	}
	writeJSONRPCResponse(writer, payload.ID, frames)
}

func (dependencies mcpRouteDependencies) resolveActor(request *http.Request) (mcp.ResolvedActor, error) {
	if dependencies.actorResolver == nil {
		return mcp.ResolvedActor{}, errors.New("mcp actor resolver is not configured")
	}
	return dependencies.actorResolver.ResolveActor(request)
}

func (dependencies mcpRouteDependencies) enforceTransportRateLimit(request *http.Request) error {
	if dependencies.transportRateLimiter == nil {
		return nil
	}
	key := transportRateLimitKey(request)
	allowed, retrySeconds := dependencies.transportRateLimiter.Consume(key)
	if allowed {
		return nil
	}
	return &mcpRouteHTTPError{
		statusCode: http.StatusTooManyRequests,
		detail:     fmt.Sprintf("Too many MCP transport requests. Retry in %d seconds.", retrySeconds),
		headers: map[string]string{
			"Retry-After": strconv.Itoa(retrySeconds),
		},
	}
}

func writeSSEFrames(writer http.ResponseWriter, frames <-chan mcp.Frame) {
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.WriteHeader(http.StatusOK)
	flusher, _ := writer.(http.Flusher)
	for frame := range frames {
		_, _ = writer.Write([]byte(formatSSEEvent("jsonrpc", frame)))
		if flusher != nil {
			flusher.Flush()
		}
	}
}

func writeJSONRPCResponse(writer http.ResponseWriter, requestID any, frames <-chan mcp.Frame) {
	buffered := collectFrames(frames)
	terminal, ok := terminalFrameForRequestID(buffered, requestID)
	if !ok {
		writeJSON(writer, http.StatusOK, map[string]any{
			"jsonrpc": "2.0",
			"id":      requestID,
			"error": map[string]any{
				"code":    -32603,
				"message": "Internal error",
			},
		})
		return
	}
	writeJSON(writer, http.StatusOK, terminal)
}

func collectFrames(frames <-chan mcp.Frame) []mcp.Frame {
	buffered := make([]mcp.Frame, 0)
	for frame := range frames {
		buffered = append(buffered, frame)
	}
	return buffered
}

func terminalFrameForRequestID(frames []mcp.Frame, requestID any) (mcp.Frame, bool) {
	for index := len(frames) - 1; index >= 0; index-- {
		frame := frames[index]
		if !sameJSONRPCID(frame["id"], requestID) {
			continue
		}
		if hasTerminalPayload(frame) {
			return frame, true
		}
	}
	return nil, false
}

func hasTerminalPayload(frame mcp.Frame) bool {
	_, hasResult := frame["result"]
	_, hasError := frame["error"]
	return hasResult || hasError
}

func sameJSONRPCID(left any, right any) bool {
	return reflect.DeepEqual(left, right)
}

func formatSSEEvent(event string, payload any) string {
	encoded, _ := json.Marshal(payload)
	return "event: " + event + "\ndata: " + string(encoded) + "\n\n"
}

func prefersSSE(acceptHeader string) bool {
	normalized := strings.TrimSpace(strings.ToLower(acceptHeader))
	sseQuality := acceptQuality(normalized, "text/event-stream")
	jsonQuality := acceptQuality(normalized, "application/json")
	if sseQuality <= 0 {
		return false
	}
	if jsonQuality <= 0 {
		return true
	}
	return sseQuality > jsonQuality
}

func acceptQuality(acceptHeader string, mediaType string) float64 {
	bestExact := -1.0
	bestWildcard := -1.0

	for _, item := range strings.Split(acceptHeader, ",") {
		token, quality, ok := parseAcceptItem(item)
		if !ok {
			continue
		}
		switch token {
		case mediaType:
			bestExact = maxFloat(bestExact, quality)
		case "*/*":
			bestWildcard = maxFloat(bestWildcard, quality)
		}
	}

	if bestExact >= 0 {
		return bestExact
	}
	if bestWildcard >= 0 {
		return bestWildcard
	}
	return -1
}

func parseQuality(parameters []string) float64 {
	quality := 1.0
	for _, parameter := range parameters {
		parameter = strings.TrimSpace(parameter)
		if !strings.HasPrefix(parameter, "q=") {
			continue
		}
		parsed, err := strconv.ParseFloat(strings.TrimPrefix(parameter, "q="), 64)
		if err != nil {
			continue
		}
		if parsed >= 0 && parsed <= 1 {
			quality = parsed
		}
	}
	return quality
}

func writeMCPRouteError(writer http.ResponseWriter, err error) {
	if writeMCPAuthError(writer, err) {
		return
	}
	if writeMCPHTTPError(writer, err) {
		return
	}
	writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "Internal server error"})
}

func writeMCPAuthError(writer http.ResponseWriter, err error) bool {
	authErr := &mcp.AuthError{}
	if !errors.As(err, &authErr) {
		return false
	}
	writeMCPResponseError(writer, authErr.StatusCode, authErr.Detail, authErr.Headers)
	return true
}

func writeMCPHTTPError(writer http.ResponseWriter, err error) bool {
	routeErr := &mcpRouteHTTPError{}
	if !errors.As(err, &routeErr) {
		return false
	}
	writeMCPResponseError(writer, routeErr.statusCode, routeErr.detail, routeErr.headers)
	return true
}

func writeMCPResponseError(
	writer http.ResponseWriter,
	statusCode int,
	detail string,
	headers map[string]string,
) {
	writeMCPHeaders(writer, headers)
	writeJSON(writer, statusCode, map[string]string{"detail": detail})
}

func writeMCPHeaders(writer http.ResponseWriter, headers map[string]string) {
	for key, value := range headers {
		writer.Header().Set(key, value)
	}
}

type mcpRouteHTTPError struct {
	statusCode int
	detail     string
	headers    map[string]string
}

func (err *mcpRouteHTTPError) Error() string {
	return err.detail
}

func parseAcceptItem(item string) (string, float64, bool) {
	part := strings.TrimSpace(item)
	if part == "" {
		return "", 0, false
	}
	segments := strings.Split(part, ";")
	token := strings.TrimSpace(segments[0])
	if token == "" {
		return "", 0, false
	}
	return token, parseQuality(segments[1:]), true
}

func maxFloat(left float64, right float64) float64 {
	if right > left {
		return right
	}
	return left
}

func transportRateLimitKey(request *http.Request) string {
	return transportClientIP(request) + ":" + authorizationFingerprint(request)
}

func transportClientIP(request *http.Request) string {
	if request == nil {
		return "unknown"
	}
	remoteAddress := strings.TrimSpace(request.RemoteAddr)
	if remoteAddress == "" {
		return "unknown"
	}
	host, _, err := net.SplitHostPort(remoteAddress)
	if err == nil {
		return host
	}
	return remoteAddress
}

func authorizationFingerprint(request *http.Request) string {
	if request == nil {
		return "anonymous"
	}
	authorization := strings.TrimSpace(request.Header.Get("Authorization"))
	if authorization == "" {
		return "anonymous"
	}
	digest := sha256.Sum256([]byte(authorization))
	return hex.EncodeToString(digest[:])[:24]
}
