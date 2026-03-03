package api

import (
	"context"
	"testing"
	"time"

	"engram/internal/admin"
)

func TestMountMemoryAdminRoutesRefreshRoutesUsePayload(t *testing.T) {
	t.Run("freshness", testRefreshEngramFreshnessPayload)
	t.Run("consolidation", testRefreshEngramConsolidationPayload)
	t.Run("contradiction", testRefreshEngramContradictionPayload)
}

func testRefreshEngramFreshnessPayload(t *testing.T) {
	captured := admin.EngramFreshnessRefreshRequest{}
	referenceTime := time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC)
	service := &fakeMemoryAdminService{
		refreshEngramFreshness: func(
			_ context.Context,
			request admin.EngramFreshnessRefreshRequest,
		) (admin.EngramFreshnessRefreshResponse, error) {
			captured = request
			return admin.EngramFreshnessRefreshResponse{
				ProjectID:     request.ProjectID,
				HalfLifeDays:  40,
				ReferenceTime: referenceTime,
				UpdatedCount:  12,
			}, nil
		},
	}
	executeMemoryAdminRefreshRequest(
		t,
		service,
		"/api/v1/admin/memory/engrams/freshness/refresh",
		[]byte(`{"project_id":"engram-vault","half_life_days":40}`),
	)
	requireEqual(t, "engram-vault", derefString(captured.ProjectID))
	if captured.HalfLifeDays == nil {
		t.Fatalf("expected half_life_days to be forwarded")
	}
	requireEqual(t, 40.0, *captured.HalfLifeDays)
}

func testRefreshEngramConsolidationPayload(t *testing.T) {
	captured := admin.EngramConsolidationSuggestionRefreshRequest{}
	suggestedAt := time.Date(2026, 3, 3, 13, 5, 0, 0, time.UTC)
	service := &fakeMemoryAdminService{
		refreshConsolidationFn: func(
			_ context.Context,
			request admin.EngramConsolidationSuggestionRefreshRequest,
		) (admin.EngramConsolidationSuggestionRefreshResponse, error) {
			captured = request
			return admin.EngramConsolidationSuggestionRefreshResponse{
				ProjectID:    request.ProjectID,
				MinGroupSize: 3,
				SuggestedAt:  suggestedAt,
				UpdatedCount: 5,
			}, nil
		},
	}
	executeMemoryAdminRefreshRequest(
		t,
		service,
		"/api/v1/admin/memory/engrams/consolidation/refresh",
		[]byte(`{"project_id":"engram-vault","min_group_size":3}`),
	)
	requireEqual(t, "engram-vault", derefString(captured.ProjectID))
	if captured.MinGroupSize == nil {
		t.Fatalf("expected min_group_size to be forwarded")
	}
	requireEqual(t, 3, *captured.MinGroupSize)
}

func testRefreshEngramContradictionPayload(t *testing.T) {
	captured := admin.EngramContradictionAlertRefreshRequest{}
	detectedAt := time.Date(2026, 3, 3, 13, 15, 0, 0, time.UTC)
	service := &fakeMemoryAdminService{
		refreshContradictionFn: func(
			_ context.Context,
			request admin.EngramContradictionAlertRefreshRequest,
		) (admin.EngramContradictionAlertRefreshResponse, error) {
			captured = request
			return admin.EngramContradictionAlertRefreshResponse{
				ProjectID:    request.ProjectID,
				DetectedAt:   detectedAt,
				UpdatedCount: 6,
			}, nil
		},
	}
	executeMemoryAdminRefreshRequest(
		t,
		service,
		"/api/v1/admin/memory/engrams/contradictions/refresh",
		[]byte(`{"project_id":"engram-vault"}`),
	)
	requireEqual(t, "engram-vault", derefString(captured.ProjectID))
}
