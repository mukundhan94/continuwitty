package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"engram/internal/mcp"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestMountMCPRoutesProbeEndpoints(t *testing.T) {
	router := chi.NewRouter()
	MountMCPRoutes(router, &fakeMCPRouteService{}, newTestMCPActorResolver(), nil)

	getRequest := httptest.NewRequest(http.MethodGet, "/api/v1/mcp/stream", nil)
	getResponse := httptest.NewRecorder()
	router.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("expected probe status 200, got %d", getResponse.Code)
	}

	payload := map[string]string{}
	if err := json.Unmarshal(getResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode probe payload: %v", err)
	}
	if payload["endpoint"] != "/api/v1/mcp/stream" {
		t.Fatalf("expected probe endpoint field")
	}

	headRequest := httptest.NewRequest(http.MethodHead, "/api/v1/mcp/stream", nil)
	headResponse := httptest.NewRecorder()
	router.ServeHTTP(headResponse, headRequest)
	if headResponse.Code != http.StatusOK {
		t.Fatalf("expected head status 200, got %d", headResponse.Code)
	}
}

func TestMountMCPRoutesNotificationAccepted(t *testing.T) {
	notificationCalled := false
	router := chi.NewRouter()
	MountMCPRoutes(router, &fakeMCPRouteService{
		handleNotificationFn: func(_ context.Context, request mcp.JSONRPCRequest) error {
			notificationCalled = request.Method == "notifications/initialized"
			return nil
		},
	}, newTestMCPActorResolver(), nil)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/mcp/stream",
		strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`),
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected notification status 202, got %d", response.Code)
	}
	if !notificationCalled {
		t.Fatalf("expected notification handler to be called")
	}
}

func TestMountMCPRoutesJSONResponseMode(t *testing.T) {
	router := chi.NewRouter()
	MountMCPRoutes(router, &fakeMCPRouteService{
		streamCallFn: func(_ context.Context, request mcp.StreamCallRequest) <-chan mcp.Frame {
			if request.Request.Method != "initialize" {
				t.Fatalf("expected initialize method")
			}
			if request.Actor.UserID == uuid.Nil {
				t.Fatalf("expected actor id in stream request")
			}
			frames := make(chan mcp.Frame, 1)
			frames <- mcp.Frame{
				"jsonrpc": "2.0",
				"id":      "init",
				"result":  map[string]any{"protocolVersion": "2024-11-05"},
			}
			close(frames)
			return frames
		},
	}, newTestMCPActorResolver(), nil)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/mcp/stream",
		strings.NewReader(`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{}}`),
	)
	request.Header.Set("Accept", "application/json, text/event-stream")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected json mode status 200, got %d", response.Code)
	}
	if !strings.Contains(response.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("expected json content type, got %q", response.Header().Get("Content-Type"))
	}

	payload := map[string]any{}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json response: %v", err)
	}
	result, ok := payload["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result payload")
	}
	if result["protocolVersion"] != "2024-11-05" {
		t.Fatalf("expected protocol version in response")
	}
}

func TestMountMCPRoutesTokenAllowedToolsFiltersToolsList(t *testing.T) {
	router := chi.NewRouter()
	MountMCPRoutes(
		router,
		mcp.NewCompatibilityService("1.2.3"),
		newTokenScopedMCPActorResolver(&models.MCPTokenAuthContext{
			Scope:        models.MCPTokenScopeWrite,
			AllowedTools: []string{"engram.query"},
		}),
		nil,
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/mcp/stream",
		strings.NewReader(`{"jsonrpc":"2.0","id":"tools-list","method":"tools/list","params":{}}`),
	)
	request.Header.Set("Accept", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected tools/list status 200, got %d", response.Code)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json response: %v", err)
	}
	result, ok := payload["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result payload")
	}
	rawTools, ok := result["tools"].([]any)
	if !ok || len(rawTools) != 1 {
		t.Fatalf("expected exactly one visible tool")
	}
	tool, ok := rawTools[0].(map[string]any)
	if !ok || tool["name"] != "engram_query" {
		t.Fatalf("expected only engram_query visible for allowed-tools token")
	}
}

func TestMountMCPRoutesTokenReadScopeRejectsWriteTool(t *testing.T) {
	router := chi.NewRouter()
	MountMCPRoutes(
		router,
		mcp.NewCompatibilityService("1.2.3"),
		newTokenScopedMCPActorResolver(&models.MCPTokenAuthContext{
			Scope: models.MCPTokenScopeRead,
		}),
		nil,
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/mcp/stream",
		strings.NewReader(`{"jsonrpc":"2.0","id":"tools-call","method":"tools/call","params":{"name":"chat_create_session","arguments":{"project_id":"p1","title":"Denied"}}}`),
	)
	request.Header.Set("Accept", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected tools/call status 200, got %d", response.Code)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json response: %v", err)
	}
	errorPayload, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected json-rpc error payload")
	}
	if errorPayload["code"] != float64(-32003) {
		t.Fatalf("expected insufficient-scope code -32003")
	}
	data, ok := errorPayload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected scope detail payload for write-denied read token")
	}
	if data["required_scope"] != "write" {
		t.Fatalf("expected required_scope write")
	}
	if data["token_scope"] != "read" {
		t.Fatalf("expected token_scope read")
	}
}

func TestMountMCPRoutesJSONResponseModeChoosesTerminalFrameAfterStreamEvents(t *testing.T) {
	router := newStreamRouterWithFrames(streamFramesWithEventAndResult("chat.send_message")...)
	response := postMCPStreamRequest(router, "application/json")

	if response.Code != http.StatusOK {
		t.Fatalf("expected json mode status 200, got %d", response.Code)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json response: %v", err)
	}
	result, ok := payload["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected terminal result payload")
	}
	message, ok := result["message"].(map[string]any)
	if !ok || message["message_id"] != "m1" {
		t.Fatalf("expected json mode to return terminal stream result")
	}
}

func TestMountMCPRoutesSSEResponseMode(t *testing.T) {
	router := chi.NewRouter()
	MountMCPRoutes(router, &fakeMCPRouteService{
		streamCallFn: func(_ context.Context, _ mcp.StreamCallRequest) <-chan mcp.Frame {
			frames := make(chan mcp.Frame, 1)
			frames <- mcp.Frame{
				"jsonrpc": "2.0",
				"id":      "init",
				"result":  map[string]any{"ok": true},
			}
			close(frames)
			return frames
		},
	}, newTestMCPActorResolver(), nil)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/mcp/stream",
		strings.NewReader(`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{}}`),
	)
	request.Header.Set("Accept", "text/event-stream")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected sse mode status 200, got %d", response.Code)
	}
	if !strings.Contains(response.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatalf("expected text/event-stream content type")
	}
	if !strings.Contains(response.Body.String(), "event: jsonrpc") {
		t.Fatalf("expected jsonrpc event in response body")
	}
}

func TestMountMCPRoutesSSEResponseModeWritesEventAndTerminalFrames(t *testing.T) {
	router := newStreamRouterWithFrames(streamFramesWithEventAndResult("chat_send_message")...)
	response := postMCPStreamRequest(router, "text/event-stream")

	if response.Code != http.StatusOK {
		t.Fatalf("expected sse mode status 200, got %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, `"method":"mcp.event"`) {
		t.Fatalf("expected mcp.event frame in sse body")
	}
	if !strings.Contains(body, `"result":{"message":{"message_id":"m1"}}`) {
		t.Fatalf("expected terminal result frame in sse body")
	}
}

func newStreamRouterWithFrames(streamFrames ...mcp.Frame) *chi.Mux {
	router := chi.NewRouter()
	MountMCPRoutes(router, &fakeMCPRouteService{
		streamCallFn: func(_ context.Context, _ mcp.StreamCallRequest) <-chan mcp.Frame {
			frames := make(chan mcp.Frame, len(streamFrames))
			for _, frame := range streamFrames {
				frames <- frame
			}
			close(frames)
			return frames
		},
	}, newTestMCPActorResolver(), nil)
	return router
}

func streamFramesWithEventAndResult(toolName string) []mcp.Frame {
	return []mcp.Frame{
		{
			"jsonrpc": "2.0",
			"method":  "mcp.event",
			"params": map[string]any{
				"id":    "msg-1",
				"tool":  toolName,
				"event": "chunk",
				"data":  map[string]any{"text": "hello"},
			},
		},
		{
			"jsonrpc": "2.0",
			"id":      "msg-1",
			"result": map[string]any{
				"message": map[string]any{"message_id": "m1"},
			},
		},
	}
}

func postMCPStreamRequest(router http.Handler, acceptHeader string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/mcp/stream",
		strings.NewReader(`{"jsonrpc":"2.0","id":"msg-1","method":"chat.send_message","params":{"session_id":"s1"}}`),
	)
	request.Header.Set("Accept", acceptHeader)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestMountMCPRoutesJSONFallbackFrame(t *testing.T) {
	router := chi.NewRouter()
	MountMCPRoutes(router, &fakeMCPRouteService{
		streamCallFn: func(_ context.Context, _ mcp.StreamCallRequest) <-chan mcp.Frame {
			frames := make(chan mcp.Frame, 1)
			frames <- mcp.Frame{
				"jsonrpc": "2.0",
				"id":      "different-id",
				"result":  map[string]any{"ok": true},
			}
			close(frames)
			return frames
		},
	}, newTestMCPActorResolver(), nil)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/mcp/stream",
		strings.NewReader(`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{}}`),
	)
	request.Header.Set("Accept", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	payload := map[string]any{}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode fallback payload: %v", err)
	}
	errorPayload, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error payload")
	}
	if errorPayload["code"] != float64(-32603) {
		t.Fatalf("expected fallback internal error code")
	}
}

func TestMountMCPRoutesWritesAuthErrorFromResolver(t *testing.T) {
	router := chi.NewRouter()
	MountMCPRoutes(
		router,
		&fakeMCPRouteService{},
		&fakeMCPActorResolver{
			resolveFn: func(_ *http.Request) (mcp.ResolvedActor, error) {
				return mcp.ResolvedActor{}, &mcp.AuthError{
					StatusCode: http.StatusUnauthorized,
					Detail:     "Authentication required",
					Headers: map[string]string{
						"WWW-Authenticate": `Bearer realm="engram-mcp"`,
					},
				}
			},
		},
		nil,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/mcp/stream",
		strings.NewReader(`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{}}`),
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.Code)
	}
	if response.Header().Get("WWW-Authenticate") == "" {
		t.Fatalf("expected WWW-Authenticate header on auth failure")
	}
}

func TestMountMCPRoutesResolverFailureDefaultsToInternalError(t *testing.T) {
	router := chi.NewRouter()
	MountMCPRoutes(
		router,
		&fakeMCPRouteService{},
		&fakeMCPActorResolver{
			resolveFn: func(_ *http.Request) (mcp.ResolvedActor, error) {
				return mcp.ResolvedActor{}, errors.New("resolver failure")
			},
		},
		nil,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/mcp/stream",
		strings.NewReader(`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{}}`),
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", response.Code)
	}
}

func TestMountMCPRoutesWritesRateLimitError(t *testing.T) {
	router := chi.NewRouter()
	MountMCPRoutes(
		router,
		&fakeMCPRouteService{},
		newTestMCPActorResolver(),
		&fakeMCPTransportRateLimiter{
			consumeFn: func(_ string) (bool, int) {
				return false, 9
			},
		},
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/mcp/stream",
		strings.NewReader(`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{}}`),
	)
	request.RemoteAddr = "10.0.0.8:4343"
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", response.Code)
	}
	if response.Header().Get("Retry-After") != "9" {
		t.Fatalf("expected retry-after header 9")
	}
	if !strings.Contains(response.Body.String(), "Too many MCP transport requests") {
		t.Fatalf("expected rate-limit detail in payload")
	}
}

func TestTransportRateLimitKeyUsesIPAndAuthorizationFingerprint(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/stream", nil)
	request.RemoteAddr = "203.0.113.14:443"
	request.Header.Set("Authorization", "Bearer secret-token")

	key := transportRateLimitKey(request)
	if !strings.HasPrefix(key, "203.0.113.14:") {
		t.Fatalf("expected key prefix with client ip, got %q", key)
	}
	parts := strings.Split(key, ":")
	if len(parts) != 2 {
		t.Fatalf("expected key to contain two segments, got %q", key)
	}
	if len(parts[1]) != 24 {
		t.Fatalf("expected 24-char auth fingerprint, got %d", len(parts[1]))
	}
}

func TestTransportRateLimitKeyAnonymousWhenAuthorizationMissing(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/stream", nil)
	request.RemoteAddr = "127.0.0.1:8000"

	key := transportRateLimitKey(request)
	if key != "127.0.0.1:anonymous" {
		t.Fatalf("expected anonymous fingerprint key, got %q", key)
	}
}

type fakeMCPRouteService struct {
	handleNotificationFn func(context.Context, mcp.JSONRPCRequest) error
	streamCallFn         func(context.Context, mcp.StreamCallRequest) <-chan mcp.Frame
}

func (service *fakeMCPRouteService) HandleNotification(ctx context.Context, request mcp.JSONRPCRequest) error {
	if service.handleNotificationFn != nil {
		return service.handleNotificationFn(ctx, request)
	}
	return nil
}

func (service *fakeMCPRouteService) StreamCall(ctx context.Context, request mcp.StreamCallRequest) <-chan mcp.Frame {
	if service.streamCallFn != nil {
		return service.streamCallFn(ctx, request)
	}
	frames := make(chan mcp.Frame)
	close(frames)
	return frames
}

type fakeMCPActorResolver struct {
	resolveFn func(*http.Request) (mcp.ResolvedActor, error)
}

func (resolver *fakeMCPActorResolver) ResolveActor(request *http.Request) (mcp.ResolvedActor, error) {
	if resolver.resolveFn != nil {
		return resolver.resolveFn(request)
	}
	return mcp.ResolvedActor{}, nil
}

func newTestMCPActorResolver() *fakeMCPActorResolver {
	return &fakeMCPActorResolver{
		resolveFn: func(_ *http.Request) (mcp.ResolvedActor, error) {
			return mcp.ResolvedActor{
				Actor: mcp.Actor{
					UserID: uuid.MustParse("40000000-0000-0000-0000-000000000004"),
					Role:   "admin",
				},
			}, nil
		},
	}
}

func newTokenScopedMCPActorResolver(tokenAuth *models.MCPTokenAuthContext) *fakeMCPActorResolver {
	return &fakeMCPActorResolver{
		resolveFn: func(_ *http.Request) (mcp.ResolvedActor, error) {
			return mcp.ResolvedActor{
				Actor: mcp.Actor{
					UserID: uuid.MustParse("40000000-0000-0000-0000-000000000004"),
					Role:   "admin",
				},
				TokenAuth: tokenAuth,
			}, nil
		},
	}
}

type fakeMCPTransportRateLimiter struct {
	consumeFn func(key string) (bool, int)
}

func (limiter *fakeMCPTransportRateLimiter) Consume(key string) (bool, int) {
	if limiter.consumeFn != nil {
		return limiter.consumeFn(key)
	}
	return true, 0
}
