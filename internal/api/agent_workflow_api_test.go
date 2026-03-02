package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"engram/internal/workflow"

	"github.com/go-chi/chi/v5"
)

func TestMountAgentWorkflowRoutesRegistersEndpoints(t *testing.T) {
	service := fakeAgentWorkflowRouteService{
		runFn: func(_ context.Context, request workflow.AgentRunRequest) (workflow.AgentState, error) {
			return workflow.AgentState{
				ThreadID:          request.ThreadID,
				Status:            "completed",
				SnapshotEngramIDs: []string{},
			}, nil
		},
		getStateFn: func(threadID string) (workflow.AgentState, bool) {
			return workflow.AgentState{ThreadID: threadID, Status: "completed"}, true
		},
		resumeFn: func(
			_ context.Context,
			threadID string,
			_ workflow.AgentResumeRequest,
		) (workflow.AgentState, bool, error) {
			return workflow.AgentState{ThreadID: threadID, Status: "completed"}, true, nil
		},
	}
	router := chi.NewRouter()
	MountAgentWorkflowRoutes(router, service)

	createBody := `{"project_id":"project-agent","thread_id":"thread-1","objective":"workflow objective"}`
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/agent-runs", bytes.NewReader([]byte(createBody)))
	createRequest.Header.Set("Content-Type", "application/json")
	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusOK {
		t.Fatalf("expected create status 200, got %d", createResponse.Code)
	}

	stateRequest := httptest.NewRequest(http.MethodGet, "/api/v1/agent-runs/thread-1", nil)
	stateResponse := httptest.NewRecorder()
	router.ServeHTTP(stateResponse, stateRequest)
	if stateResponse.Code != http.StatusOK {
		t.Fatalf("expected state status 200, got %d", stateResponse.Code)
	}

	resumeRequest := httptest.NewRequest(http.MethodPost, "/api/v1/agent-runs/thread-1/resume", bytes.NewReader([]byte(`{"notes":["next"]}`)))
	resumeRequest.Header.Set("Content-Type", "application/json")
	resumeResponse := httptest.NewRecorder()
	router.ServeHTTP(resumeResponse, resumeRequest)
	if resumeResponse.Code != http.StatusOK {
		t.Fatalf("expected resume status 200, got %d", resumeResponse.Code)
	}
}

func TestAgentWorkflowRoutesValidationErrors(t *testing.T) {
	router := chi.NewRouter()
	MountAgentWorkflowRoutes(router, fakeAgentWorkflowRouteService{})

	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/agent-runs",
		bytes.NewReader([]byte(`{"project_id":"project-agent","thread_id":"thread-1","objective":"","snapshot_every_n_notes":0}`)),
	)
	createRequest.Header.Set("Content-Type", "application/json")
	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid create status 400, got %d", createResponse.Code)
	}

	resumeRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/agent-runs/thread-1/resume",
		bytes.NewReader([]byte(`{"snapshot_every_n_notes":0}`)),
	)
	resumeRequest.Header.Set("Content-Type", "application/json")
	resumeResponse := httptest.NewRecorder()
	router.ServeHTTP(resumeResponse, resumeRequest)
	if resumeResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid resume status 400, got %d", resumeResponse.Code)
	}

	emptyThreadRequest := httptest.NewRequest(http.MethodGet, "/api/v1/agent-runs/%20", nil)
	emptyThreadResponse := httptest.NewRecorder()
	router.ServeHTTP(emptyThreadResponse, emptyThreadRequest)
	if emptyThreadResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected empty thread status 400, got %d", emptyThreadResponse.Code)
	}
}

func TestAgentWorkflowRoutesNotFoundAndErrors(t *testing.T) {
	service := fakeAgentWorkflowRouteService{
		runFn: func(_ context.Context, _ workflow.AgentRunRequest) (workflow.AgentState, error) {
			return workflow.AgentState{}, errors.New("run failed")
		},
		getStateFn: func(_ string) (workflow.AgentState, bool) {
			return workflow.AgentState{}, false
		},
		resumeFn: func(
			_ context.Context,
			_ string,
			_ workflow.AgentResumeRequest,
		) (workflow.AgentState, bool, error) {
			return workflow.AgentState{}, false, nil
		},
	}
	router := chi.NewRouter()
	MountAgentWorkflowRoutes(router, service)

	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/agent-runs",
		bytes.NewReader([]byte(`{"project_id":"project-agent","thread_id":"thread-1","objective":"objective"}`)),
	)
	createRequest.Header.Set("Content-Type", "application/json")
	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected create error status 500, got %d", createResponse.Code)
	}

	stateRequest := httptest.NewRequest(http.MethodGet, "/api/v1/agent-runs/missing-thread", nil)
	stateResponse := httptest.NewRecorder()
	router.ServeHTTP(stateResponse, stateRequest)
	if stateResponse.Code != http.StatusNotFound {
		t.Fatalf("expected missing state status 404, got %d", stateResponse.Code)
	}

	resumeRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/agent-runs/missing-thread/resume",
		bytes.NewReader([]byte(`{"notes":["next"]}`)),
	)
	resumeRequest.Header.Set("Content-Type", "application/json")
	resumeResponse := httptest.NewRecorder()
	router.ServeHTTP(resumeResponse, resumeRequest)
	if resumeResponse.Code != http.StatusNotFound {
		t.Fatalf("expected missing resume status 404, got %d", resumeResponse.Code)
	}
}

func TestBuildAgentRunResponseCopiesSnapshotIDs(t *testing.T) {
	state := workflow.AgentState{
		ThreadID:          "thread-1",
		Status:            "completed",
		SnapshotEngramIDs: []string{"engram-1"},
	}
	response := buildAgentRunResponse("thread-1", state)
	state.SnapshotEngramIDs[0] = "mutated"
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("encode response: %v", err)
	}
	if !bytes.Contains(encoded, []byte("engram-1")) {
		t.Fatalf("expected copied snapshot id in response, got %s", string(encoded))
	}
}

type fakeAgentWorkflowRouteService struct {
	runFn      func(ctx context.Context, request workflow.AgentRunRequest) (workflow.AgentState, error)
	getStateFn func(threadID string) (workflow.AgentState, bool)
	resumeFn   func(ctx context.Context, threadID string, updates workflow.AgentResumeRequest) (workflow.AgentState, bool, error)
}

func (service fakeAgentWorkflowRouteService) Run(
	ctx context.Context,
	request workflow.AgentRunRequest,
) (workflow.AgentState, error) {
	if service.runFn == nil {
		return workflow.AgentState{ThreadID: request.ThreadID, Status: "completed"}, nil
	}
	return service.runFn(ctx, request)
}

func (service fakeAgentWorkflowRouteService) GetState(threadID string) (workflow.AgentState, bool) {
	if service.getStateFn == nil {
		return workflow.AgentState{}, false
	}
	return service.getStateFn(threadID)
}

func (service fakeAgentWorkflowRouteService) Resume(
	ctx context.Context,
	threadID string,
	updates workflow.AgentResumeRequest,
) (workflow.AgentState, bool, error) {
	if service.resumeFn == nil {
		return workflow.AgentState{}, false, nil
	}
	return service.resumeFn(ctx, threadID, updates)
}
