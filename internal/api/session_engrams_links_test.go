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

type engramLinkRouteFixture struct {
	actor       *models.UserAuthRecord
	handler     http.Handler
	loginCookie *http.Cookie
}

type engramLinkRecordFixture struct {
	linkID          uuid.UUID
	sourceEngramID  uuid.UUID
	targetEngramID  uuid.UUID
	relationType    models.EngramLinkRelationType
	weight          float64
	temporalWeight  float64
	confidence      float64
	origin          models.EngramLinkOrigin
	status          models.EngramLinkStatus
	evidence        map[string]any
	createdByUserID uuid.UUID
	createdAt       time.Time
	updatedAt       time.Time
}

type engramLinkSuggestionFixture struct {
	sourceEngramID uuid.UUID
	targetEngramID uuid.UUID
	relationType   models.EngramLinkRelationType
	weight         float64
	temporalWeight float64
	confidence     float64
	score          float64
	origin         models.EngramLinkOrigin
	status         models.EngramLinkStatus
	reasons        []string
}

func newEngramLinkRecord(fixture engramLinkRecordFixture) models.EngramLinkRecord {
	return models.EngramLinkRecord{
		LinkID:          fixture.linkID,
		ProjectID:       "proj-1",
		SourceEngramID:  fixture.sourceEngramID,
		TargetEngramID:  fixture.targetEngramID,
		RelationType:    fixture.relationType,
		Weight:          fixture.weight,
		TemporalWeight:  fixture.temporalWeight,
		Confidence:      fixture.confidence,
		Origin:          fixture.origin,
		Status:          fixture.status,
		EvidenceJSON:    fixture.evidence,
		CreatedByUserID: fixture.createdByUserID,
		CreatedAt:       fixture.createdAt,
		UpdatedAt:       fixture.updatedAt,
	}
}

func newEngramLinkSuggestion(fixture engramLinkSuggestionFixture) models.EngramLinkSuggestion {
	return models.EngramLinkSuggestion{
		SourceEngramID:  fixture.sourceEngramID,
		TargetEngramID:  fixture.targetEngramID,
		ProjectID:       "proj-1",
		TargetTitle:     "Candidate",
		TargetAbstract:  "Candidate abstract",
		TargetCreatedAt: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
		RelationType:    fixture.relationType,
		Weight:          fixture.weight,
		TemporalWeight:  fixture.temporalWeight,
		Confidence:      fixture.confidence,
		Score:           fixture.score,
		Origin:          fixture.origin,
		Status:          fixture.status,
		Reasons:         fixture.reasons,
		EvidenceJSON:    map[string]any{},
	}
}

func decodeLinkPayloadList(t *testing.T, responseBody []byte, contextName string) []map[string]any {
	t.Helper()
	var payload []map[string]any
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		t.Fatalf("decode %s response: %v", contextName, err)
	}
	return payload
}

func decodeLinkPayloadObject(t *testing.T, responseBody []byte, contextName string) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		t.Fatalf("decode %s response: %v", contextName, err)
	}
	return payload
}

func setupEngramLinkRouteFixture(
	t *testing.T,
	role models.UserRole,
	configure func(actor *models.UserAuthRecord) sessionEngramRoutesHandlerOptions,
) engramLinkRouteFixture {
	t.Helper()
	actor := newSessionRoutesTestActor(t, role)
	options := configure(actor)
	options.actor = actor
	handler, manager := buildSessionEngramRoutesTestHandler(t, options)
	loginCookie := loginSessionEngramActor(
		t,
		handler,
		manager,
		sessionEngramLoginCredentials{username: actor.Username, password: "StrongPassword-12345"},
	)
	return engramLinkRouteFixture{
		actor:       actor,
		handler:     handler,
		loginCookie: loginCookie,
	}
}

func TestMountSessionAuthRoutesCreateEngramLinkUsesRepository(t *testing.T) {
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-000000009101")
	targetEngramID := uuid.MustParse("00000000-0000-0000-0000-000000009102")
	linkID := uuid.MustParse("00000000-0000-0000-0000-000000009103")
	fixture := setupEngramLinkRouteFixture(
		t,
		models.UserRoleAnalyst,
		func(actor *models.UserAuthRecord) sessionEngramRoutesHandlerOptions {
			return sessionEngramRoutesHandlerOptions{
				createEngramLink: func(
					_ context.Context,
					input SessionEngramLinkCreateInput,
				) (*models.EngramLinkRecord, error) {
					requireEqual(t, sourceEngramID, input.SourceEngramID)
					requireEqual(t, targetEngramID, input.TargetEngramID)
					requireEqual(t, models.EngramLinkRelationSupports, input.RelationType)
					requireEqual(t, 0.8, input.Weight)
					requireEqual(t, actor.UserID, input.ActorUserID)
					record := newEngramLinkRecord(engramLinkRecordFixture{
						linkID:          linkID,
						sourceEngramID:  sourceEngramID,
						targetEngramID:  targetEngramID,
						relationType:    models.EngramLinkRelationSupports,
						weight:          0.8,
						temporalWeight:  0.7,
						confidence:      0.6,
						origin:          models.EngramLinkOriginManual,
						status:          models.EngramLinkStatusActive,
						evidence:        map[string]any{"reason": "shared source"},
						createdByUserID: actor.UserID,
						createdAt:       time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
						updatedAt:       time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
					})
					return &record, nil
				},
			}
		},
	)
	response := executeEngramRequest(
		t,
		fixture.handler,
		fixture.loginCookie,
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
	payload := decodeLinkPayloadObject(t, response.Body.Bytes(), "create link")
	requireEqual(t, linkID.String(), payload["link_id"].(string))
}

func TestMountSessionAuthRoutesCreateEngramLinkMapsDuplicateError(t *testing.T) {
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-000000009104")
	targetEngramID := uuid.MustParse("00000000-0000-0000-0000-000000009105")
	fixture := setupEngramLinkRouteFixture(
		t,
		models.UserRoleAnalyst,
		func(_ *models.UserAuthRecord) sessionEngramRoutesHandlerOptions {
			return sessionEngramRoutesHandlerOptions{
				createEngramLink: func(
					_ context.Context,
					_ SessionEngramLinkCreateInput,
				) (*models.EngramLinkRecord, error) {
					return nil, repository.ErrEngramLinkExists
				},
			}
		},
	)
	response := executeEngramRequest(
		t,
		fixture.handler,
		fixture.loginCookie,
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
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-000000009106")
	linkID := uuid.MustParse("00000000-0000-0000-0000-000000009107")
	fixture := setupEngramLinkRouteFixture(
		t,
		models.UserRoleViewer,
		func(actor *models.UserAuthRecord) sessionEngramRoutesHandlerOptions {
			return sessionEngramRoutesHandlerOptions{
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
					record := newEngramLinkRecord(engramLinkRecordFixture{
						linkID:          linkID,
						sourceEngramID:  sourceEngramID,
						targetEngramID:  uuid.MustParse("00000000-0000-0000-0000-000000009108"),
						relationType:    models.EngramLinkRelationRelatedTo,
						weight:          0.5,
						temporalWeight:  0.4,
						confidence:      0.7,
						origin:          models.EngramLinkOriginManual,
						status:          models.EngramLinkStatusActive,
						evidence:        map[string]any{},
						createdByUserID: actor.UserID,
						createdAt:       time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC),
						updatedAt:       time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC),
					})
					return []models.EngramLinkRecord{record}, nil
				},
			}
		},
	)
	response := executeEngramRequest(
		t,
		fixture.handler,
		fixture.loginCookie,
		engramRequestSpec{
			method: http.MethodGet,
			path:   "/api/v1/engrams/" + sourceEngramID.String() + "/links?relation_type=related_to&include_archived=true&limit=3&offset=1",
		},
	)
	requireEqual(t, http.StatusOK, response.Code)
	payload := decodeLinkPayloadList(t, response.Body.Bytes(), "list link")
	requireEqual(t, 1, len(payload))
	requireEqual(t, linkID.String(), payload[0]["link_id"].(string))
}

func TestMountSessionAuthRoutesUpdateEngramLinkUsesRepository(t *testing.T) {
	linkID := uuid.MustParse("00000000-0000-0000-0000-000000009109")
	fixture := setupEngramLinkRouteFixture(
		t,
		models.UserRoleAnalyst,
		func(actor *models.UserAuthRecord) sessionEngramRoutesHandlerOptions {
			return sessionEngramRoutesHandlerOptions{
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
					record := newEngramLinkRecord(engramLinkRecordFixture{
						linkID:          linkID,
						sourceEngramID:  uuid.MustParse("00000000-0000-0000-0000-000000009110"),
						targetEngramID:  uuid.MustParse("00000000-0000-0000-0000-000000009111"),
						relationType:    models.EngramLinkRelationContradicts,
						weight:          0.3,
						temporalWeight:  0.2,
						confidence:      0.1,
						origin:          models.EngramLinkOriginSuggested,
						status:          models.EngramLinkStatusRejected,
						evidence:        map[string]any{"reviewed": true},
						createdByUserID: actor.UserID,
						createdAt:       time.Date(2026, 3, 1, 11, 0, 0, 0, time.UTC),
						updatedAt:       time.Date(2026, 3, 1, 11, 5, 0, 0, time.UTC),
					})
					return &record, nil
				},
			}
		},
	)
	response := executeEngramRequest(
		t,
		fixture.handler,
		fixture.loginCookie,
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
	linkID := uuid.MustParse("00000000-0000-0000-0000-000000009112")
	fixture := setupEngramLinkRouteFixture(
		t,
		models.UserRoleAnalyst,
		func(_ *models.UserAuthRecord) sessionEngramRoutesHandlerOptions {
			return sessionEngramRoutesHandlerOptions{
				archiveEngramLink: func(
					_ context.Context,
					input SessionEngramLinkArchiveInput,
				) (*models.EngramLinkRecord, error) {
					requireEqual(t, linkID, input.LinkID)
					return nil, nil
				},
			}
		},
	)
	response := executeEngramRequest(
		t,
		fixture.handler,
		fixture.loginCookie,
		engramRequestSpec{method: http.MethodDelete, path: "/api/v1/engrams/links/" + linkID.String()},
	)
	requireEqual(t, http.StatusNotFound, response.Code)
}

func TestMountSessionAuthRoutesSuggestEngramLinksUsesRepository(t *testing.T) {
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-000000009113")
	targetEngramID := uuid.MustParse("00000000-0000-0000-0000-000000009114")
	fixture := setupEngramLinkRouteFixture(
		t,
		models.UserRoleViewer,
		func(actor *models.UserAuthRecord) sessionEngramRoutesHandlerOptions {
			return sessionEngramRoutesHandlerOptions{
				suggestEngramLinks: func(
					_ context.Context,
					input SessionEngramLinkSuggestInput,
				) ([]models.EngramLinkSuggestion, error) {
					requireEqual(t, sourceEngramID, input.SourceEngramID)
					requireEqual(t, actor.UserID, input.ActorUserID)
					requireEqual(t, 6, input.Limit)
					requireEqual(t, 16, input.MaxCandidates)
					requireEqual(t, 0.25, input.MinimumScore)
					suggestion := newEngramLinkSuggestion(engramLinkSuggestionFixture{
						sourceEngramID: sourceEngramID,
						targetEngramID: targetEngramID,
						relationType:   models.EngramLinkRelationRelatedTo,
						weight:         0.6,
						temporalWeight: 0.5,
						confidence:     0.7,
						score:          0.72,
						origin:         models.EngramLinkOriginSuggested,
						status:         models.EngramLinkStatusSuggested,
						reasons:        []string{"semantic_overlap"},
					})
					return []models.EngramLinkSuggestion{suggestion}, nil
				},
			}
		},
	)
	response := executeEngramRequest(
		t,
		fixture.handler,
		fixture.loginCookie,
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
	payload := decodeLinkPayloadList(t, response.Body.Bytes(), "suggest")
	requireEqual(t, 1, len(payload))
	requireEqual(t, targetEngramID.String(), payload[0]["target_engram_id"].(string))
}

func TestMountSessionAuthRoutesHygieneEngramLinksUsesRepository(t *testing.T) {
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-000000009130")
	targetEngramID := uuid.MustParse("00000000-0000-0000-0000-000000009131")
	linkID := uuid.MustParse("00000000-0000-0000-0000-000000009132")
	fixture := setupEngramLinkRouteFixture(
		t,
		models.UserRoleViewer,
		func(actor *models.UserAuthRecord) sessionEngramRoutesHandlerOptions {
			return sessionEngramRoutesHandlerOptions{
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
			}
		},
	)
	response := executeEngramRequest(
		t,
		fixture.handler,
		fixture.loginCookie,
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
	payload := decodeLinkPayloadList(t, response.Body.Bytes(), "hygiene")
	requireEqual(t, 1, len(payload))
	requireEqual(t, "stale_low_value", payload[0]["category"].(string))
}

func TestMountSessionAuthRoutesTraceEngramLinksUsesRepository(t *testing.T) {
	rootEngramID := uuid.MustParse("00000000-0000-0000-0000-000000009115")
	linkID := uuid.MustParse("00000000-0000-0000-0000-000000009116")
	fixture := setupEngramLinkRouteFixture(
		t,
		models.UserRoleViewer,
		func(actor *models.UserAuthRecord) sessionEngramRoutesHandlerOptions {
			return sessionEngramRoutesHandlerOptions{
				traceEngramLinks: func(
					_ context.Context,
					input SessionEngramTraceInput,
				) ([]models.EngramLinkTraversalStep, error) {
					requireEqual(t, rootEngramID, input.RootEngramID)
					requireEqual(t, actor.UserID, input.ActorUserID)
					requireEqual(t, 3, input.MaxDepth)
					requireEqual(t, 7, input.MaxNeighbors)
					requireEqual(t, true, input.IncludeArchived)
					record := newEngramLinkRecord(engramLinkRecordFixture{
						linkID:          linkID,
						sourceEngramID:  rootEngramID,
						targetEngramID:  uuid.MustParse("00000000-0000-0000-0000-000000009117"),
						relationType:    models.EngramLinkRelationSupports,
						weight:          0.9,
						temporalWeight:  0.8,
						confidence:      0.7,
						origin:          models.EngramLinkOriginManual,
						status:          models.EngramLinkStatusActive,
						evidence:        map[string]any{},
						createdByUserID: actor.UserID,
						createdAt:       time.Date(2026, 3, 1, 12, 30, 0, 0, time.UTC),
						updatedAt:       time.Date(2026, 3, 1, 12, 30, 0, 0, time.UTC),
					})
					return []models.EngramLinkTraversalStep{
						{
							Depth: 1,
							Link:  record,
						},
					}, nil
				},
			}
		},
	)
	response := executeEngramRequest(
		t,
		fixture.handler,
		fixture.loginCookie,
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
