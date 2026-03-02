package api

import (
	"context"
	"net/http"
	"strings"

	"engram/internal/workflow"

	"github.com/go-chi/chi/v5"
)

const defaultAgentSnapshotEveryNNotes = 3

// AgentWorkflowRouteService captures workflow operations used by API routes.
type AgentWorkflowRouteService interface {
	Run(ctx context.Context, request workflow.AgentRunRequest) (workflow.AgentState, error)
	GetState(threadID string) (workflow.AgentState, bool)
	Resume(
		ctx context.Context,
		threadID string,
		updates workflow.AgentResumeRequest,
	) (workflow.AgentState, bool, error)
}

type agentRunRouteRequest struct {
	ProjectID           string                      `json:"project_id"`
	ThreadID            string                      `json:"thread_id"`
	Objective           string                      `json:"objective"`
	Notes               []string                    `json:"notes,omitempty"`
	Assumptions         []string                    `json:"assumptions,omitempty"`
	Tags                []string                    `json:"tags,omitempty"`
	Keywords            []string                    `json:"keywords,omitempty"`
	Sources             []workflow.AgentSourceInput `json:"sources,omitempty"`
	AutoPersistEngram   *bool                       `json:"auto_persist_engram,omitempty"`
	SnapshotEnabled     *bool                       `json:"snapshot_enabled,omitempty"`
	SnapshotEveryNNotes *int                        `json:"snapshot_every_n_notes,omitempty"`
}

type agentRunResponse struct {
	ThreadID          string              `json:"thread_id"`
	Status            string              `json:"status"`
	EngramID          *string             `json:"engram_id,omitempty"`
	SnapshotEngramIDs []string            `json:"snapshot_engram_ids,omitempty"`
	State             workflow.AgentState `json:"state"`
}

// MountAgentWorkflowRoutes registers workflow run/state/resume endpoints.
func MountAgentWorkflowRoutes(router chi.Router, service AgentWorkflowRouteService) {
	router.Post("/api/v1/agent-runs", func(writer http.ResponseWriter, request *http.Request) {
		handleAgentRunCreate(writer, request, service)
	})
	router.Get("/api/v1/agent-runs/{thread_id}", func(writer http.ResponseWriter, request *http.Request) {
		handleAgentRunGet(writer, request, service)
	})
	router.Post("/api/v1/agent-runs/{thread_id}/resume", func(writer http.ResponseWriter, request *http.Request) {
		handleAgentRunResume(writer, request, service)
	})
}

func handleAgentRunCreate(
	writer http.ResponseWriter,
	request *http.Request,
	service AgentWorkflowRouteService,
) {
	if service == nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "agent workflow service is not configured"})
		return
	}
	runRequest, ok := decodeAgentRunRequest(writer, request)
	if !ok {
		return
	}
	state, err := service.Run(request.Context(), runRequest)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}
	writeJSON(writer, http.StatusOK, buildAgentRunResponse(state.ThreadID, state))
}

func handleAgentRunGet(
	writer http.ResponseWriter,
	request *http.Request,
	service AgentWorkflowRouteService,
) {
	if service == nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "agent workflow service is not configured"})
		return
	}
	threadID := strings.TrimSpace(chi.URLParam(request, "thread_id"))
	if threadID == "" {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "thread_id is required"})
		return
	}
	state, found := service.GetState(threadID)
	if !found {
		writeJSON(writer, http.StatusNotFound, map[string]string{"detail": "Thread state not found"})
		return
	}
	writeJSON(writer, http.StatusOK, buildAgentRunResponse(threadID, state))
}

func handleAgentRunResume(
	writer http.ResponseWriter,
	request *http.Request,
	service AgentWorkflowRouteService,
) {
	if service == nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "agent workflow service is not configured"})
		return
	}
	threadID := strings.TrimSpace(chi.URLParam(request, "thread_id"))
	if threadID == "" {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "thread_id is required"})
		return
	}
	updates, ok := decodeAgentResumeRequest(writer, request)
	if !ok {
		return
	}
	state, found, err := service.Resume(request.Context(), threadID, updates)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}
	if !found {
		writeJSON(writer, http.StatusNotFound, map[string]string{"detail": "Thread state not found"})
		return
	}
	writeJSON(writer, http.StatusOK, buildAgentRunResponse(threadID, state))
}

func decodeAgentRunRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (workflow.AgentRunRequest, bool) {
	payload, ok := decodeAgentRunPayload(writer, request)
	if !ok {
		return workflow.AgentRunRequest{}, false
	}
	if !validateAgentRunRequiredFields(writer, payload) {
		return workflow.AgentRunRequest{}, false
	}
	snapshotEvery, ok := resolveAgentSnapshotEveryNNotes(writer, payload.SnapshotEveryNNotes)
	if !ok {
		return workflow.AgentRunRequest{}, false
	}
	return buildAgentRunRequest(payload, snapshotEvery), true
}

func decodeAgentRunPayload(
	writer http.ResponseWriter,
	request *http.Request,
) (agentRunRouteRequest, bool) {
	payload := agentRunRouteRequest{}
	if !decodeJSONAllowEmpty(writer, request, &payload) {
		return agentRunRouteRequest{}, false
	}
	return payload, true
}

func validateAgentRunRequiredFields(writer http.ResponseWriter, payload agentRunRouteRequest) bool {
	if strings.TrimSpace(payload.ProjectID) != "" {
		if strings.TrimSpace(payload.ThreadID) != "" {
			if strings.TrimSpace(payload.Objective) != "" {
				return true
			}
		}
	}
	writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "project_id, thread_id, and objective are required"})
	return false
}

func resolveAgentSnapshotEveryNNotes(writer http.ResponseWriter, value *int) (int, bool) {
	snapshotEvery := defaultAgentSnapshotEveryNNotes
	if value != nil {
		snapshotEvery = *value
	}
	if snapshotEvery >= 1 {
		return snapshotEvery, true
	}
	writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "snapshot_every_n_notes must be >= 1"})
	return 0, false
}

func buildAgentRunRequest(payload agentRunRouteRequest, snapshotEvery int) workflow.AgentRunRequest {
	return workflow.AgentRunRequest{
		ProjectID:           strings.TrimSpace(payload.ProjectID),
		ThreadID:            strings.TrimSpace(payload.ThreadID),
		Objective:           strings.TrimSpace(payload.Objective),
		Notes:               payload.Notes,
		Assumptions:         payload.Assumptions,
		Tags:                payload.Tags,
		Keywords:            payload.Keywords,
		Sources:             payload.Sources,
		AutoPersistEngram:   optionalBool(payload.AutoPersistEngram, true),
		SnapshotEnabled:     optionalBool(payload.SnapshotEnabled, false),
		SnapshotEveryNNotes: snapshotEvery,
	}
}

func optionalBool(value *bool, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}
	return *value
}

func decodeAgentResumeRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (workflow.AgentResumeRequest, bool) {
	payload := workflow.AgentResumeRequest{}
	if !decodeJSONAllowEmpty(writer, request, &payload) {
		return workflow.AgentResumeRequest{}, false
	}
	if payload.SnapshotEveryNNotes != nil && *payload.SnapshotEveryNNotes < 1 {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "snapshot_every_n_notes must be >= 1"})
		return workflow.AgentResumeRequest{}, false
	}
	return payload, true
}

func buildAgentRunResponse(threadID string, state workflow.AgentState) agentRunResponse {
	return agentRunResponse{
		ThreadID:          threadID,
		Status:            state.Status,
		EngramID:          state.EngramID,
		SnapshotEngramIDs: append([]string{}, state.SnapshotEngramIDs...),
		State:             state,
	}
}
