package api

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"engram/internal/mcp"

	"github.com/go-chi/chi/v5"
)

// MountMCPRoutes registers MCP stream transport endpoints.
func MountMCPRoutes(router chi.Router, service mcp.Service) {
	deps := mcpRouteDependencies{service: service}
	router.Route("/api/v1/mcp", func(mcpRouter chi.Router) {
		mcpRouter.Get("/stream", deps.handleProbe)
		mcpRouter.Head("/stream", deps.handleProbeHead)
		mcpRouter.Post("/stream", deps.handleStream)
	})
}

type mcpRouteDependencies struct {
	service mcp.Service
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

	payload := mcp.JSONRPCRequest{}
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "Invalid payload"})
		return
	}

	if payload.IsNotification() {
		if err := dependencies.service.HandleNotification(request.Context(), payload); err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "Internal server error"})
			return
		}
		writer.WriteHeader(http.StatusAccepted)
		return
	}

	frames := dependencies.service.StreamCall(request.Context(), mcp.StreamCallRequest{Request: payload})
	if prefersSSE(request.Header.Get("accept")) {
		writeSSEFrames(writer, frames)
		return
	}
	writeJSONRPCResponse(writer, payload.ID, frames)
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
