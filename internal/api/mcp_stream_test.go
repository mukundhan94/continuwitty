package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"engram/internal/mcp"

	"github.com/go-chi/chi/v5"
)

func TestMountMCPRoutesProbeEndpoints(t *testing.T) {
	router := chi.NewRouter()
	MountMCPRoutes(router, &fakeMCPRouteService{})

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
	})

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
			frames := make(chan mcp.Frame, 1)
			frames <- mcp.Frame{
				"jsonrpc": "2.0",
				"id":      "init",
				"result":  map[string]any{"protocolVersion": "2024-11-05"},
			}
			close(frames)
			return frames
		},
	})

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
	})

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
	})

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
