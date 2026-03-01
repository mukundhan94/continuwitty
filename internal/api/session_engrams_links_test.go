package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestMountSessionAuthRoutesCreateEngramLinkUsesRepository(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleAnalyst)
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-000000009101")
	targetEngramID := uuid.MustParse("00000000-0000-0000-0000-000000009102")
	linkID := uuid.MustParse("00000000-0000-0000-0000-000000009103")
	handler, manager := buildSessionEngramRoutesTestHandler(
		t,
		sessionEngramRoutesHandlerOptions{
			actor: actor,
			createEngramLink: func(
				_ context.Context,
				input SessionEngramLinkCreateInput,
			) (*models.EngramLinkRecord, error) {
				requireEqual(t, sourceEngramID, input.SourceEngramID)
				requireEqual(t, targetEngramID, input.TargetEngramID)
				requireEqual(t, models.EngramLinkRelationSupports, input.RelationType)
				requireEqual(t, 0.8, input.Weight)
				requireEqual(t, actor.UserID, input.ActorUserID)
				return &models.EngramLinkRecord{
					LinkID:          linkID,
					ProjectID:       "proj-1",
					SourceEngramID:  sourceEngramID,
					TargetEngramID:  targetEngramID,
					RelationType:    models.EngramLinkRelationSupports,
					Weight:          0.8,
					TemporalWeight:  0.7,
					Confidence:      0.6,
					Origin:          models.EngramLinkOriginManual,
					Status:          models.EngramLinkStatusActive,
					EvidenceJSON:    map[string]any{"reason": "shared source"},
					CreatedByUserID: actor.UserID,
					CreatedAt:       time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
					UpdatedAt:       time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
				}, nil
			},
		},
	)
	loginCookie := loginSessionEngramActor(
		t,
		handler,
		manager,
		sessionEngramLoginCredentials{username: actor.Username, password: "StrongPassword-12345"},
	)
	response := executeEngramRequest(
		t,
		handler,
		loginCookie,
		engramRequestSpec{
			method: http.MethodPost,
			path:   "/api/v1/engrams/" + sourceEngramID.String() + "/links",
			body: map[string]any{
				"target_engram_id": targetEngramID.String(),
				"relation_type":    "supports",
				"weight":           0.8,
				"temporal_weight":  0.7,
				"confidence":       0.6,
			},
		},
	)
	requireEqual(t, http.StatusOK, response.Code)
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode create link response: %v", err)
	}
	requireEqual(t, linkID.String(), payload["link_id"].(string))
}

func TestMountSessionAuthRoutesCreateEngramLinkMapsDuplicateError(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleAnalyst)
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-000000009104")
	targetEngramID := uuid.MustParse("00000000-0000-0000-0000-000000009105")
	handler, manager := buildSessionEngramRoutesTestHandler(
		t,
		sessionEngramRoutesHandlerOptions{
			actor: actor,
			createEngramLink: func(
				_ context.Context,
				_ SessionEngramLinkCreateInput,
			) (*models.EngramLinkRecord, error) {
				return nil, repository.ErrEngramLinkExists
			},
		},
	)
	loginCookie := loginSessionEngramActor(
		t,
		handler,
		manager,
		sessionEngramLoginCredentials{username: actor.Username, password: "StrongPassword-12345"},
	)
	response := executeEngramRequest(
		t,
		handler,
		loginCookie,
		engramRequestSpec{
			method: http.MethodPost,
			path:   "/api/v1/engrams/" + sourceEngramID.String() + "/links",
			body: map[string]any{
				"target_engram_id": targetEngramID.String(),
			},
		},
	)
	requireEqual(t, http.StatusConflict, response.Code)
}

func TestMountSessionAuthRoutesListEngramLinksUsesRepository(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleViewer)
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-000000009106")
	linkID := uuid.MustParse("00000000-0000-0000-0000-000000009107")
	handler, manager := buildSessionEngramRoutesTestHandler(
		t,
		sessionEngramRoutesHandlerOptions{
			actor: actor,
			listEngramLinks: func(
				_ context.Context,
				input SessionEngramLinkListInput,
			) ([]models.EngramLinkRecord, error) {
				requireEqual(t, sourceEngramID, input.SourceEngramID)
				requireEqual(t, actor.UserID, input.ActorUserID)
				requireEqual(t, true, input.IncludeArchived)
				requireEqual(t, 3, input.Limit)
				requireEqual(t, 1, input.Offset)
				if input.RelationType == nil {
					t.Fatalf("expected relation filter")
				}
				requireEqual(t, models.EngramLinkRelationRelatedTo, *input.RelationType)
				return []models.EngramLinkRecord{
					{
						LinkID:          linkID,
						ProjectID:       "proj-1",
						SourceEngramID:  sourceEngramID,
						TargetEngramID:  uuid.MustParse("00000000-0000-0000-0000-000000009108"),
						RelationType:    models.EngramLinkRelationRelatedTo,
						Weight:          0.5,
						TemporalWeight:  0.4,
						Confidence:      0.7,
						Origin:          models.EngramLinkOriginManual,
						Status:          models.EngramLinkStatusActive,
						EvidenceJSON:    map[string]any{},
						CreatedByUserID: actor.UserID,
						CreatedAt:       time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC),
						UpdatedAt:       time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC),
					},
				}, nil
			},
		},
	)
	loginCookie := loginSessionEngramActor(
		t,
		handler,
		manager,
		sessionEngramLoginCredentials{username: actor.Username, password: "StrongPassword-12345"},
	)
	response := executeEngramRequest(
		t,
		handler,
		loginCookie,
		engramRequestSpec{
			method: http.MethodGet,
			path:   "/api/v1/engrams/" + sourceEngramID.String() + "/links?relation_type=related_to&include_archived=true&limit=3&offset=1",
		},
	)
	requireEqual(t, http.StatusOK, response.Code)
	var payload []map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode list link response: %v", err)
	}
	requireEqual(t, 1, len(payload))
	requireEqual(t, linkID.String(), payload[0]["link_id"].(string))
}

func TestMountSessionAuthRoutesUpdateEngramLinkUsesRepository(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleAnalyst)
	linkID := uuid.MustParse("00000000-0000-0000-0000-000000009109")
	handler, manager := buildSessionEngramRoutesTestHandler(
		t,
		sessionEngramRoutesHandlerOptions{
			actor: actor,
			updateEngramLink: func(
				_ context.Context,
				input SessionEngramLinkUpdateInput,
			) (*models.EngramLinkRecord, error) {
				requireEqual(t, linkID, input.LinkID)
				requireEqual(t, actor.UserID, input.ActorUserID)
				if input.Status == nil {
					t.Fatalf("expected status field")
				}
				requireEqual(t, models.EngramLinkStatusRejected, *input.Status)
				return &models.EngramLinkRecord{
					LinkID:          linkID,
					ProjectID:       "proj-1",
					SourceEngramID:  uuid.MustParse("00000000-0000-0000-0000-000000009110"),
					TargetEngramID:  uuid.MustParse("00000000-0000-0000-0000-000000009111"),
					RelationType:    models.EngramLinkRelationContradicts,
					Weight:          0.3,
					TemporalWeight:  0.2,
					Confidence:      0.1,
					Origin:          models.EngramLinkOriginSuggested,
					Status:          models.EngramLinkStatusRejected,
					EvidenceJSON:    map[string]any{"reviewed": true},
					CreatedByUserID: actor.UserID,
					CreatedAt:       time.Date(2026, 3, 1, 11, 0, 0, 0, time.UTC),
					UpdatedAt:       time.Date(2026, 3, 1, 11, 5, 0, 0, time.UTC),
				}, nil
			},
		},
	)
	loginCookie := loginSessionEngramActor(
		t,
		handler,
		manager,
		sessionEngramLoginCredentials{username: actor.Username, password: "StrongPassword-12345"},
	)
	response := executeEngramRequest(
		t,
		handler,
		loginCookie,
		engramRequestSpec{
			method: http.MethodPatch,
			path:   "/api/v1/engrams/links/" + linkID.String(),
			body: map[string]any{
				"status":     "rejected",
				"confidence": 0.1,
			},
		},
	)
	requireEqual(t, http.StatusOK, response.Code)
}

func TestMountSessionAuthRoutesArchiveEngramLinkReturns404WhenMissing(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleAnalyst)
	linkID := uuid.MustParse("00000000-0000-0000-0000-000000009112")
	handler, manager := buildSessionEngramRoutesTestHandler(
		t,
		sessionEngramRoutesHandlerOptions{
			actor: actor,
			archiveEngramLink: func(
				_ context.Context,
				input SessionEngramLinkArchiveInput,
			) (*models.EngramLinkRecord, error) {
				requireEqual(t, linkID, input.LinkID)
				return nil, nil
			},
		},
	)
	loginCookie := loginSessionEngramActor(
		t,
		handler,
		manager,
		sessionEngramLoginCredentials{username: actor.Username, password: "StrongPassword-12345"},
	)
	response := executeEngramRequest(
		t,
		handler,
		loginCookie,
		engramRequestSpec{method: http.MethodDelete, path: "/api/v1/engrams/links/" + linkID.String()},
	)
	requireEqual(t, http.StatusNotFound, response.Code)
}

func TestMountSessionAuthRoutesSuggestEngramLinksUsesRepository(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleViewer)
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-000000009113")
	targetEngramID := uuid.MustParse("00000000-0000-0000-0000-000000009114")
	handler, manager := buildSessionEngramRoutesTestHandler(
		t,
		sessionEngramRoutesHandlerOptions{
			actor: actor,
			suggestEngramLinks: func(
				_ context.Context,
				input SessionEngramLinkSuggestInput,
			) ([]models.EngramLinkSuggestion, error) {
				requireEqual(t, sourceEngramID, input.SourceEngramID)
				requireEqual(t, actor.UserID, input.ActorUserID)
				requireEqual(t, 6, input.Limit)
				requireEqual(t, 16, input.MaxCandidates)
				requireEqual(t, 0.25, input.MinimumScore)
				return []models.EngramLinkSuggestion{
					{
						SourceEngramID:  sourceEngramID,
						TargetEngramID:  targetEngramID,
						ProjectID:       "proj-1",
						TargetTitle:     "Candidate",
						TargetAbstract:  "Candidate abstract",
						TargetCreatedAt: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
						RelationType:    models.EngramLinkRelationRelatedTo,
						Weight:          0.6,
						TemporalWeight:  0.5,
						Confidence:      0.7,
						Score:           0.72,
						Origin:          models.EngramLinkOriginSuggested,
						Status:          models.EngramLinkStatusSuggested,
						Reasons:         []string{"semantic_overlap"},
						EvidenceJSON:    map[string]any{},
					},
				}, nil
			},
		},
	)
	loginCookie := loginSessionEngramActor(
		t,
		handler,
		manager,
		sessionEngramLoginCredentials{username: actor.Username, password: "StrongPassword-12345"},
	)
	response := executeEngramRequest(
		t,
		handler,
		loginCookie,
		engramRequestSpec{
			method: http.MethodPost,
			path:   "/api/v1/engrams/" + sourceEngramID.String() + "/links/suggest",
			body: map[string]any{
				"limit":          6,
				"max_candidates": 16,
				"minimum_score":  0.25,
			},
		},
	)
	requireEqual(t, http.StatusOK, response.Code)
	var payload []map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode suggest response: %v", err)
	}
	requireEqual(t, 1, len(payload))
	requireEqual(t, targetEngramID.String(), payload[0]["target_engram_id"].(string))
}

func TestMountSessionAuthRoutesHygieneEngramLinksUsesRepository(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleViewer)
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-000000009130")
	targetEngramID := uuid.MustParse("00000000-0000-0000-0000-000000009131")
	linkID := uuid.MustParse("00000000-0000-0000-0000-000000009132")
	handler, manager := buildSessionEngramRoutesTestHandler(
		t,
		sessionEngramRoutesHandlerOptions{
			actor: actor,
			hygieneEngramLinks: func(
				_ context.Context,
				input SessionEngramLinkHygieneInput,
			) ([]models.EngramLinkHygieneRecommendation, error) {
				requireEqual(t, sourceEngramID, input.SourceEngramID)
				requireEqual(t, actor.UserID, input.ActorUserID)
				requireEqual(t, 300, input.Limit)
				requireEqual(t, 90, input.StaleAfterDays)
				requireEqual(t, 0.3, input.LowValueThreshold)
				requireEqual(t, true, input.IncludeArchived)
				return []models.EngramLinkHygieneRecommendation{
					{
						Category:        models.EngramLinkHygieneCategoryStaleLowValue,
						Severity:        "low",
						SourceEngramID:  sourceEngramID,
						TargetEngramID:  targetEngramID,
						LinkIDs:         []uuid.UUID{linkID},
						Detail:          "stale low-value link",
						SuggestedAction: "archive_stale_low_value",
						Score:           0.22,
					},
				}, nil
			},
		},
	)
	loginCookie := loginSessionEngramActor(
		t,
		handler,
		manager,
		sessionEngramLoginCredentials{username: actor.Username, password: "StrongPassword-12345"},
	)
	response := executeEngramRequest(
		t,
		handler,
		loginCookie,
		engramRequestSpec{
			method: http.MethodPost,
			path:   "/api/v1/engrams/" + sourceEngramID.String() + "/links/hygiene",
			body: map[string]any{
				"include_archived":    true,
				"limit":               300,
				"stale_after_days":    90,
				"low_value_threshold": 0.3,
			},
		},
	)
	requireEqual(t, http.StatusOK, response.Code)
	var payload []map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode hygiene response: %v", err)
	}
	requireEqual(t, 1, len(payload))
	requireEqual(t, "stale_low_value", payload[0]["category"].(string))
}

func TestMountSessionAuthRoutesTraceEngramLinksUsesRepository(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleViewer)
	rootEngramID := uuid.MustParse("00000000-0000-0000-0000-000000009115")
	linkID := uuid.MustParse("00000000-0000-0000-0000-000000009116")
	handler, manager := buildSessionEngramRoutesTestHandler(
		t,
		sessionEngramRoutesHandlerOptions{
			actor: actor,
			traceEngramLinks: func(
				_ context.Context,
				input SessionEngramTraceInput,
			) ([]models.EngramLinkTraversalStep, error) {
				requireEqual(t, rootEngramID, input.RootEngramID)
				requireEqual(t, actor.UserID, input.ActorUserID)
				requireEqual(t, 3, input.MaxDepth)
				requireEqual(t, 7, input.MaxNeighbors)
				requireEqual(t, true, input.IncludeArchived)
				return []models.EngramLinkTraversalStep{
					{
						Depth: 1,
						Link: models.EngramLinkRecord{
							LinkID:          linkID,
							ProjectID:       "proj-1",
							SourceEngramID:  rootEngramID,
							TargetEngramID:  uuid.MustParse("00000000-0000-0000-0000-000000009117"),
							RelationType:    models.EngramLinkRelationSupports,
							Weight:          0.9,
							TemporalWeight:  0.8,
							Confidence:      0.7,
							Origin:          models.EngramLinkOriginManual,
							Status:          models.EngramLinkStatusActive,
							EvidenceJSON:    map[string]any{},
							CreatedByUserID: actor.UserID,
							CreatedAt:       time.Date(2026, 3, 1, 12, 30, 0, 0, time.UTC),
							UpdatedAt:       time.Date(2026, 3, 1, 12, 30, 0, 0, time.UTC),
						},
					},
				}, nil
			},
		},
	)
	loginCookie := loginSessionEngramActor(
		t,
		handler,
		manager,
		sessionEngramLoginCredentials{username: actor.Username, password: "StrongPassword-12345"},
	)
	response := executeEngramRequest(
		t,
		handler,
		loginCookie,
		engramRequestSpec{
			method: http.MethodPost,
			path:   "/api/v1/engrams/" + rootEngramID.String() + "/trace",
			body: map[string]any{
				"max_depth":        3,
				"max_neighbors":    7,
				"include_archived": true,
			},
		},
	)
	requireEqual(t, http.StatusOK, response.Code)
}
