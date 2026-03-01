package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramLinkCreateParity(t *testing.T) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000007001")
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-000000007002")
	targetEngramID := uuid.MustParse("00000000-0000-0000-0000-000000007003")
	link := sampleCompatibilityLinkRecord(
		uuid.MustParse("00000000-0000-0000-0000-000000007004"),
		sourceEngramID,
		targetEngramID,
	)
	params := map[string]any{
		"engram_id":        sourceEngramID.String(),
		"target_engram_id": targetEngramID.String(),
		"relation_type":    "supports",
		"weight":           0.8,
		"temporal_weight":  0.7,
		"confidence":       0.6,
	}
	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "engram.link_create", params),
			asToolsCallPath: false,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "engram_link_create", params),
			asToolsCallPath: true,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			service := &fakeEngramLinkCreateService{created: &link}
			frame := runCompatibilityRequestWithService(
				t,
				newEngramLinkCompatibilityService(
					CompatibilityServiceDependencies{EngramLinkCreate: service},
				),
				testCase.request,
			)
			created := engramLinkFromFrame(t, frame, testCase.asToolsCallPath)
			if created.LinkID != link.LinkID {
				t.Fatalf("expected created link id %s", link.LinkID)
			}
			if service.call.ActorUserID != actorUserID {
				t.Fatalf("expected actor user id in create call")
			}
			if service.call.SourceEngramID != sourceEngramID || service.call.TargetEngramID != targetEngramID {
				t.Fatalf("expected source/target ids in create call")
			}
		})
	}
}

func TestCompatibilityServiceEngramLinkCreateDuplicateErrorMapping(t *testing.T) {
	frame := runCompatibilityRequestWithService(
		t,
		newEngramLinkCompatibilityService(
			CompatibilityServiceDependencies{
				EngramLinkCreate: &fakeEngramLinkCreateService{err: repository.ErrEngramLinkExists},
			},
		),
		toolsCallRequest(
			"00000000-0000-0000-0000-000000007010",
			"engram_link_create",
			map[string]any{
				"engram_id":        "00000000-0000-0000-0000-000000007011",
				"target_engram_id": "00000000-0000-0000-0000-000000007012",
			},
		),
	)
	errorPayload := errorPayloadFromFrame(t, frame)
	requireErrorCode(t, errorPayload, -32602)
	assertErrorStatusCode(t, errorPayload, 409)
	assertErrorDetail(t, errorPayload, repository.ErrEngramLinkExists.Error())
}

func TestCompatibilityServiceEngramLinkListSuggestAndTrace(t *testing.T) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000007020")
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-000000007021")
	targetEngramID := uuid.MustParse("00000000-0000-0000-0000-000000007022")
	listService := &fakeEngramLinkListService{
		links: []models.EngramLinkRecord{
			sampleCompatibilityLinkRecord(
				uuid.MustParse("00000000-0000-0000-0000-000000007023"),
				sourceEngramID,
				targetEngramID,
			),
		},
	}
	suggestService := &fakeEngramLinkSuggestService{
		suggestions: []models.EngramLinkSuggestion{
			{
				SourceEngramID:  sourceEngramID,
				TargetEngramID:  targetEngramID,
				ProjectID:       "proj-1",
				TargetTitle:     "candidate",
				TargetAbstract:  "candidate abstract",
				TargetCreatedAt: time.Date(2026, 3, 1, 13, 0, 0, 0, time.UTC),
				RelationType:    models.EngramLinkRelationRelatedTo,
				Weight:          0.6,
				TemporalWeight:  0.5,
				Confidence:      0.7,
				Score:           0.74,
				Origin:          models.EngramLinkOriginSuggested,
				Status:          models.EngramLinkStatusSuggested,
				Reasons:         []string{"semantic_overlap=0.75"},
				EvidenceJSON:    map[string]any{},
			},
		},
	}
	traceService := &fakeEngramTracePathService{
		steps: []models.EngramLinkTraversalStep{
			{
				Depth: 1,
				Link: sampleCompatibilityLinkRecord(
					uuid.MustParse("00000000-0000-0000-0000-000000007024"),
					sourceEngramID,
					targetEngramID,
				),
			},
		},
	}
	service := newEngramLinkCompatibilityService(
		CompatibilityServiceDependencies{
			EngramLinkList:    listService,
			EngramLinkSuggest: suggestService,
			EngramTracePath:   traceService,
		},
	)

	listFrame := runCompatibilityRequestWithService(
		t,
		service,
		directToolRequest(
			actorUserID.String(),
			"engram.link_list",
			map[string]any{
				"engram_id":        sourceEngramID.String(),
				"relation_type":    "supports",
				"include_archived": true,
				"limit":            3,
				"offset":           1,
			},
		),
	)
	links := engramLinksFromFrame(t, listFrame, false)
	if len(links) != 1 {
		t.Fatalf("expected one link response")
	}
	if listService.call.SourceEngramID != sourceEngramID || listService.call.Limit != 3 || listService.call.Offset != 1 {
		t.Fatalf("expected list call to capture parsed paging/source params")
	}

	suggestFrame := runCompatibilityRequestWithService(
		t,
		service,
		toolsCallRequest(
			actorUserID.String(),
			"engram_link_suggest",
			map[string]any{
				"engram_id":      sourceEngramID.String(),
				"limit":          4,
				"max_candidates": 12,
				"minimum_score":  0.3,
			},
		),
	)
	suggestions := engramLinkSuggestionsFromFrame(t, suggestFrame, true)
	if len(suggestions) != 1 {
		t.Fatalf("expected one suggestion response")
	}
	if suggestService.call.SourceEngramID != sourceEngramID || suggestService.call.Limit != 4 {
		t.Fatalf("expected suggest call inputs to be parsed")
	}

	traceFrame := runCompatibilityRequestWithService(
		t,
		service,
		directToolRequest(
			actorUserID.String(),
			"engram.trace_path",
			map[string]any{
				"engram_id":     sourceEngramID.String(),
				"max_depth":     3,
				"max_neighbors": 8,
			},
		),
	)
	steps := engramLinkTraceStepsFromFrame(t, traceFrame, false)
	if len(steps) != 1 {
		t.Fatalf("expected one trace step")
	}
	if traceService.call.RootEngramID != sourceEngramID || traceService.call.MaxDepth != 3 {
		t.Fatalf("expected trace call to capture parsed parameters")
	}
}

func TestCompatibilityServiceEngramLinkUpdateAndArchive(t *testing.T) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000007030")
	linkID := uuid.MustParse("00000000-0000-0000-0000-000000007031")
	updatedRecord := sampleCompatibilityLinkRecord(
		linkID,
		uuid.MustParse("00000000-0000-0000-0000-000000007032"),
		uuid.MustParse("00000000-0000-0000-0000-000000007033"),
	)
	updateService := &fakeEngramLinkUpdateService{updated: &updatedRecord}
	archiveService := &fakeEngramLinkArchiveService{archived: &updatedRecord}
	service := newEngramLinkCompatibilityService(
		CompatibilityServiceDependencies{
			EngramLinkUpdate:  updateService,
			EngramLinkArchive: archiveService,
		},
	)

	updateFrame := runCompatibilityRequestWithService(
		t,
		service,
		directToolRequest(
			actorUserID.String(),
			"engram.link_update",
			map[string]any{
				"link_id":       linkID.String(),
				"status":        "rejected",
				"confidence":    0.2,
				"evidence_json": map[string]any{"reviewed": true},
			},
		),
	)
	_ = engramLinkFromFrame(t, updateFrame, false)
	if updateService.call.LinkID != linkID {
		t.Fatalf("expected update call link id")
	}

	archiveFrame := runCompatibilityRequestWithService(
		t,
		service,
		directToolRequest(
			actorUserID.String(),
			"engram.link_archive",
			map[string]any{"link_id": linkID.String()},
		),
	)
	_ = engramLinkFromFrame(t, archiveFrame, false)
	if archiveService.call.LinkID != linkID {
		t.Fatalf("expected archive call link id")
	}

	validationFrame := runCompatibilityRequestWithService(
		t,
		service,
		toolsCallRequest(
			actorUserID.String(),
			"engram_link_update",
			map[string]any{"link_id": linkID.String()},
		),
	)
	errorPayload := errorPayloadFromFrame(t, validationFrame)
	requireErrorCode(t, errorPayload, -32602)
	assertErrorStatusCode(t, errorPayload, 422)
}

func newEngramLinkCompatibilityService(dependencies CompatibilityServiceDependencies) Service {
	return NewCompatibilityServiceWithDependencies("1.2.3", dependencies)
}

func engramLinkFromFrame(t *testing.T, frame Frame, asToolsCallPath bool) models.EngramLinkRecord {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	link, ok := payload["link"].(models.EngramLinkRecord)
	if !ok {
		t.Fatalf("expected link payload")
	}
	return link
}

func engramLinksFromFrame(t *testing.T, frame Frame, asToolsCallPath bool) []models.EngramLinkRecord {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	links, ok := payload["links"].([]models.EngramLinkRecord)
	if !ok {
		t.Fatalf("expected links payload")
	}
	return links
}

func engramLinkSuggestionsFromFrame(t *testing.T, frame Frame, asToolsCallPath bool) []models.EngramLinkSuggestion {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	suggestions, ok := payload["suggestions"].([]models.EngramLinkSuggestion)
	if !ok {
		t.Fatalf("expected suggestions payload")
	}
	return suggestions
}

func engramLinkTraceStepsFromFrame(t *testing.T, frame Frame, asToolsCallPath bool) []models.EngramLinkTraversalStep {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	steps, ok := payload["steps"].([]models.EngramLinkTraversalStep)
	if !ok {
		t.Fatalf("expected steps payload")
	}
	return steps
}

func sampleCompatibilityLinkRecord(
	linkID uuid.UUID,
	sourceEngramID uuid.UUID,
	targetEngramID uuid.UUID,
) models.EngramLinkRecord {
	createdAt := time.Date(2026, 3, 1, 12, 30, 0, 0, time.UTC)
	return models.EngramLinkRecord{
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
		EvidenceJSON:    map[string]any{},
		CreatedByUserID: uuid.MustParse("00000000-0000-0000-0000-000000007099"),
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
	}
}

type fakeEngramLinkCreateService struct {
	created *models.EngramLinkRecord
	err     error
	call    EngramLinkCreateRequest
}

func (service *fakeEngramLinkCreateService) CreateEngramLink(
	_ context.Context,
	request EngramLinkCreateRequest,
) (*models.EngramLinkRecord, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	if service.created == nil {
		return nil, nil
	}
	created := *service.created
	return &created, nil
}

type fakeEngramLinkListService struct {
	links []models.EngramLinkRecord
	err   error
	call  EngramLinkListRequest
}

func (service *fakeEngramLinkListService) ListEngramLinks(
	_ context.Context,
	request EngramLinkListRequest,
) ([]models.EngramLinkRecord, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	return append([]models.EngramLinkRecord(nil), service.links...), nil
}

type fakeEngramLinkUpdateService struct {
	updated *models.EngramLinkRecord
	err     error
	call    EngramLinkUpdateRequest
}

func (service *fakeEngramLinkUpdateService) UpdateEngramLink(
	_ context.Context,
	request EngramLinkUpdateRequest,
) (*models.EngramLinkRecord, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	if service.updated == nil {
		return nil, nil
	}
	updated := *service.updated
	return &updated, nil
}

type fakeEngramLinkArchiveService struct {
	archived *models.EngramLinkRecord
	err      error
	call     EngramLinkArchiveRequest
}

func (service *fakeEngramLinkArchiveService) ArchiveEngramLink(
	_ context.Context,
	request EngramLinkArchiveRequest,
) (*models.EngramLinkRecord, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	if service.archived == nil {
		return nil, nil
	}
	archived := *service.archived
	return &archived, nil
}

type fakeEngramLinkSuggestService struct {
	suggestions []models.EngramLinkSuggestion
	err         error
	call        EngramLinkSuggestRequest
}

func (service *fakeEngramLinkSuggestService) SuggestEngramLinks(
	_ context.Context,
	request EngramLinkSuggestRequest,
) ([]models.EngramLinkSuggestion, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	return append([]models.EngramLinkSuggestion(nil), service.suggestions...), nil
}

type fakeEngramTracePathService struct {
	steps []models.EngramLinkTraversalStep
	err   error
	call  EngramTracePathRequest
}

func (service *fakeEngramTracePathService) TraceEngramPath(
	_ context.Context,
	request EngramTracePathRequest,
) ([]models.EngramLinkTraversalStep, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	return append([]models.EngramLinkTraversalStep(nil), service.steps...), nil
}

type fakeEngramLinkGetService struct {
	link *models.EngramLinkRecord
	err  error
	call EngramLinkGetRequest
}

func (service *fakeEngramLinkGetService) GetEngramLink(
	_ context.Context,
	request EngramLinkGetRequest,
) (*models.EngramLinkRecord, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	if service.link == nil {
		return nil, nil
	}
	copied := *service.link
	return &copied, nil
}

func assertEngramLinkCall[T any](t *testing.T, expected T, actual T) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected call %+v, got %+v", expected, actual)
	}
}

func TestFakeEngramLinkCallAssertionHelper(t *testing.T) {
	assertEngramLinkCall(t, 1, 1)
}

func TestCompatibilityServiceEngramLinkUpdateInternalError(t *testing.T) {
	frame := runCompatibilityRequestWithService(
		t,
		newEngramLinkCompatibilityService(
			CompatibilityServiceDependencies{
				EngramLinkUpdate: &fakeEngramLinkUpdateService{err: errors.New("boom")},
			},
		),
		directToolRequest(
			"00000000-0000-0000-0000-000000007040",
			"engram.link_update",
			map[string]any{
				"link_id":    "00000000-0000-0000-0000-000000007041",
				"confidence": 0.3,
			},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, frame), -32603)
}
