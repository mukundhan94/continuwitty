package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"engram/internal/admin"
	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramUpdateParity(t *testing.T) {
	fixture := buildEngramUpdateParityFixture(t)
	for _, testCase := range engramUpdateParityCases(fixture.actorUserID, fixture.params) {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			runEngramUpdateParityCase(t, fixture, testCase)
		})
	}
}

func TestCompatibilityServiceEngramUpdateUsesDefaults(t *testing.T) {
	actorUserID := uuid.MustParse("39870000-0000-0000-0000-000000000398")
	engramID := uuid.MustParse("39870000-0000-0000-0000-000000000399")
	service := &fakeEngramUpdateService{
		updated: &models.AdminEngramRecord{EngramID: engramID},
	}
	frame := runCompatibilityRequestWithService(
		t,
		newEngramUpdateCompatibilityService(service),
		directToolRequest(
			actorUserID.String(),
			"engram.update",
			map[string]any{"engram_id": engramID.String()},
		),
	)
	_ = engramUpdateResponseFromFrame(t, frame, false)
	if service.call.ExpectedUpdatedAt != nil {
		t.Fatalf("expected expected_updated_at omitted by default")
	}
	if service.call.Title != nil || service.call.Abstract != nil {
		t.Fatalf("expected optional text fields omitted by default")
	}
	if service.call.Tags != nil || service.call.Keywords != nil {
		t.Fatalf("expected optional tag fields omitted by default")
	}
	if service.call.VisibilityScope != nil || service.call.Sources != nil {
		t.Fatalf("expected optional update metadata omitted by default")
	}
}

func TestCompatibilityServiceEngramUpdateValidationAndErrors(t *testing.T) {
	service := newEngramUpdateCompatibilityService(&fakeEngramUpdateService{})
	assertEngramUpdateValidationErrors(t, service)
	assertEngramUpdateErrorMappings(t)
}

type engramUpdateParityFixture struct {
	actorUserID    uuid.UUID
	params         map[string]any
	updated        *models.AdminEngramRecord
	expectedCall   engramUpdateCallExpectation
	expectedEngram models.AdminEngramRecord
}

type engramUpdateParityCase struct {
	name            string
	request         StreamCallRequest
	asToolsCallPath bool
	expectedRole    models.UserRole
}

func buildEngramUpdateParityFixture(t *testing.T) engramUpdateParityFixture {
	t.Helper()
	actorUserID := uuid.MustParse("39860000-0000-0000-0000-000000000398")
	engramID := uuid.MustParse("39860000-0000-0000-0000-000000000399")
	expectedUpdatedAt := mustParseUpdateRFC3339(t, "2026-02-21T00:00:00Z")
	sourceCapturedAt := mustParseUpdateRFC3339(t, "2026-02-21T01:02:03Z")
	visibilityScope := models.VisibilityScopeProject
	tags := []string{"incident"}
	keywords := []string{"memory"}
	sources := []models.AdminEngramSourceInput{
		{
			CapturedAt: sourceCapturedAt,
			URL:        "https://example.com/source",
			Title:      stringPtr("Source title"),
			Snippet:    stringPtr("Source snippet"),
		},
	}
	updated := models.AdminEngramRecord{
		EngramID:                engramID,
		ProjectID:               "proj-alpha",
		Title:                   "Updated title",
		Abstract:                "Updated abstract",
		DetailedSummaryMarkdown: "## Updated",
		Tags:                    tags,
		Keywords:                keywords,
		VisibilityScope:         visibilityScope,
	}
	return engramUpdateParityFixture{
		actorUserID: actorUserID,
		params: map[string]any{
			"engram_id":                 engramID.String(),
			"title":                     "Updated title",
			"abstract":                  "Updated abstract",
			"detailed_summary_markdown": "## Updated",
			"tags":                      []any{"incident"},
			"keywords":                  []any{"memory"},
			"visibility_scope":          "project",
			"expected_updated_at":       expectedUpdatedAt.Format(time.RFC3339),
			"sources": []any{
				map[string]any{
					"captured_at": sourceCapturedAt.Format(time.RFC3339),
					"url":         "https://example.com/source",
					"title":       "Source title",
					"snippet":     "Source snippet",
				},
			},
		},
		expectedEngram: updated,
		updated:        &updated,
		expectedCall: engramUpdateCallExpectation{
			actorUserID:             actorUserID,
			engramID:                engramID,
			expectedUpdatedAt:       &expectedUpdatedAt,
			title:                   stringPtr("Updated title"),
			abstract:                stringPtr("Updated abstract"),
			detailedSummaryMarkdown: stringPtr("## Updated"),
			tags:                    &tags,
			keywords:                &keywords,
			visibilityScope:         &visibilityScope,
			sources:                 &sources,
		},
	}
}

func engramUpdateParityCases(
	actorUserID uuid.UUID,
	params map[string]any,
) []engramUpdateParityCase {
	return []engramUpdateParityCase{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "engram.update", params),
			asToolsCallPath: false,
			expectedRole:    models.UserRoleViewer,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "engram_update", params),
			asToolsCallPath: true,
			expectedRole:    models.UserRoleAnalyst,
		},
	}
}

func runEngramUpdateParityCase(
	t *testing.T,
	fixture engramUpdateParityFixture,
	testCase engramUpdateParityCase,
) {
	t.Helper()
	service := &fakeEngramUpdateService{updated: fixture.updated}
	frame := runCompatibilityRequestWithService(
		t,
		newEngramUpdateCompatibilityService(service),
		testCase.request,
	)
	engram := engramUpdateResponseFromFrame(t, frame, testCase.asToolsCallPath)
	if !reflect.DeepEqual(fixture.expectedEngram, engram) {
		t.Fatalf("expected update payload to match service output")
	}
	expectedCall := fixture.expectedCall
	expectedCall.actorRole = testCase.expectedRole
	assertEngramUpdateCall(t, service.call, expectedCall)
}

func assertEngramUpdateValidationErrors(t *testing.T, service Service) {
	t.Helper()
	testCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing engram id", params: map[string]any{}},
		{name: "invalid engram id", params: map[string]any{"engram_id": "bad"}},
		{name: "invalid tags", params: map[string]any{"engram_id": uuid.NewString(), "tags": "bad"}},
		{name: "invalid visibility", params: map[string]any{"engram_id": uuid.NewString(), "visibility_scope": "bad"}},
		{name: "invalid expected updated at", params: map[string]any{"engram_id": uuid.NewString(), "expected_updated_at": "bad"}},
		{name: "invalid sources", params: map[string]any{"engram_id": uuid.NewString(), "sources": "bad"}},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39880000-0000-0000-0000-000000000398",
					"engram_update",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}
}

func assertEngramUpdateErrorMappings(t *testing.T) {
	t.Helper()
	notFoundID := uuid.MustParse("39880000-0000-0000-0000-000000000399")
	assertEngramUpdateNotFound(t, newEngramUpdateCompatibilityService(&fakeEngramUpdateService{}), notFoundID)
	assertEngramUpdateStale(t, notFoundID)
	assertEngramUpdateNotFoundError(t, notFoundID)
	assertEngramUpdateInternalError(t, notFoundID)
}

func assertEngramUpdateNotFound(t *testing.T, service Service, engramID uuid.UUID) {
	t.Helper()
	frame := runCompatibilityRequestWithService(
		t,
		service,
		toolsCallRequest(
			"39880000-0000-0000-0000-000000000400",
			"engram_update",
			map[string]any{"engram_id": engramID.String()},
		),
	)
	errorPayload := errorPayloadFromFrame(t, frame)
	requireErrorCode(t, errorPayload, -32004)
	assertEngramMutationNotFoundData(t, errorPayload, engramID)
}

func assertEngramUpdateStale(t *testing.T, engramID uuid.UUID) {
	t.Helper()
	frame := runCompatibilityRequestWithService(
		t,
		newEngramUpdateCompatibilityService(
			&fakeEngramUpdateService{err: admin.ErrEngramStale},
		),
		toolsCallRequest(
			"39880000-0000-0000-0000-000000000401",
			"engram_update",
			map[string]any{"engram_id": engramID.String()},
		),
	)
	errorPayload := errorPayloadFromFrame(t, frame)
	requireErrorCode(t, errorPayload, -32602)
	assertErrorStatusCode(t, errorPayload, 409)
	assertErrorDetail(t, errorPayload, admin.ErrEngramStale.Error())
}

func assertEngramUpdateNotFoundError(t *testing.T, engramID uuid.UUID) {
	t.Helper()
	assertEngramUpdateNotFound(
		t,
		newEngramUpdateCompatibilityService(
			&fakeEngramUpdateService{err: admin.ErrEngramNotFound},
		),
		engramID,
	)
}

func assertEngramUpdateInternalError(t *testing.T, engramID uuid.UUID) {
	t.Helper()
	frame := runCompatibilityRequestWithService(
		t,
		newEngramUpdateCompatibilityService(
			&fakeEngramUpdateService{err: errors.New("boom")},
		),
		toolsCallRequest(
			"39880000-0000-0000-0000-000000000403",
			"engram_update",
			map[string]any{"engram_id": engramID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, frame), -32603)
}

func newEngramUpdateCompatibilityService(service EngramUpdateService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramUpdate: service},
	)
}

func engramUpdateResponseFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) models.AdminEngramRecord {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	engram, ok := payload["engram"].(models.AdminEngramRecord)
	if !ok {
		t.Fatalf("expected engram update payload")
	}
	return engram
}

type engramUpdateCallExpectation struct {
	actorUserID             uuid.UUID
	actorRole               models.UserRole
	engramID                uuid.UUID
	expectedUpdatedAt       *time.Time
	title                   *string
	abstract                *string
	detailedSummaryMarkdown *string
	tags                    *[]string
	keywords                *[]string
	visibilityScope         *models.VisibilityScope
	sources                 *[]models.AdminEngramSourceInput
}

func assertEngramUpdateCall(
	t *testing.T,
	call EngramUpdateRequest,
	expected engramUpdateCallExpectation,
) {
	t.Helper()
	assertEngramUpdateRequiredFields(t, call, expected)
	assertEngramUpdateTextFields(t, call, expected)
	assertEngramUpdateCollectionFields(t, call, expected)
}

func assertEngramUpdateRequiredFields(
	t *testing.T,
	call EngramUpdateRequest,
	expected engramUpdateCallExpectation,
) {
	t.Helper()
	if call.ActorUserID != expected.actorUserID {
		t.Fatalf("expected actor user id forwarded")
	}
	if call.ActorRole != expected.actorRole {
		t.Fatalf("expected normalized actor role forwarded")
	}
	if call.EngramID != expected.engramID {
		t.Fatalf("expected engram id forwarded")
	}
	assertOptionalField(
		t,
		optionalFieldAssertion[time.Time]{
			label:    "expected_updated_at",
			actual:   call.ExpectedUpdatedAt,
			expected: expected.expectedUpdatedAt,
		},
		func(a, b time.Time) bool { return a.Equal(b) },
	)
}

func assertEngramUpdateTextFields(
	t *testing.T,
	call EngramUpdateRequest,
	expected engramUpdateCallExpectation,
) {
	t.Helper()
	assertOptionalField(
		t,
		optionalFieldAssertion[string]{label: "title", actual: call.Title, expected: expected.title},
		func(a, b string) bool { return a == b },
	)
	assertOptionalField(
		t,
		optionalFieldAssertion[string]{label: "abstract", actual: call.Abstract, expected: expected.abstract},
		func(a, b string) bool { return a == b },
	)
	assertOptionalField(
		t,
		optionalFieldAssertion[string]{
			label:    "detailed_summary_markdown",
			actual:   call.DetailedSummaryMarkdown,
			expected: expected.detailedSummaryMarkdown,
		},
		func(a, b string) bool { return a == b },
	)
}

func assertEngramUpdateCollectionFields(
	t *testing.T,
	call EngramUpdateRequest,
	expected engramUpdateCallExpectation,
) {
	t.Helper()
	assertOptionalField(
		t,
		optionalFieldAssertion[[]string]{
			label:    "tags",
			actual:   call.Tags,
			expected: expected.tags,
		},
		func(actual []string, expected []string) bool { return reflect.DeepEqual(actual, expected) },
	)
	assertOptionalField(
		t,
		optionalFieldAssertion[[]string]{
			label:    "keywords",
			actual:   call.Keywords,
			expected: expected.keywords,
		},
		func(actual []string, expected []string) bool { return reflect.DeepEqual(actual, expected) },
	)
	assertOptionalField(
		t,
		optionalFieldAssertion[models.VisibilityScope]{
			label:    "visibility_scope",
			actual:   call.VisibilityScope,
			expected: expected.visibilityScope,
		},
		func(a, b models.VisibilityScope) bool { return a == b },
	)
	assertOptionalField(
		t,
		optionalFieldAssertion[[]models.AdminEngramSourceInput]{
			label:    "sources",
			actual:   call.Sources,
			expected: expected.sources,
		},
		func(actual []models.AdminEngramSourceInput, expected []models.AdminEngramSourceInput) bool {
			return reflect.DeepEqual(actual, expected)
		},
	)
}

type optionalFieldAssertion[T any] struct {
	label    string
	actual   *T
	expected *T
}

func assertOptionalField[T any](
	t *testing.T,
	assertion optionalFieldAssertion[T],
	equals func(T, T) bool,
) {
	t.Helper()
	if assertion.expected == nil {
		if assertion.actual != nil {
			t.Fatalf("expected %s omitted", assertion.label)
		}
		return
	}
	if assertion.actual == nil || !equals(*assertion.actual, *assertion.expected) {
		t.Fatalf("expected %s forwarded", assertion.label)
	}
}

func mustParseUpdateRFC3339(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("expected RFC3339 time in test: %v", err)
	}
	return parsed
}

type fakeEngramUpdateService struct {
	updated *models.AdminEngramRecord
	err     error
	call    EngramUpdateRequest
}

func (service *fakeEngramUpdateService) UpdateEngram(
	_ context.Context,
	request EngramUpdateRequest,
) (*models.AdminEngramRecord, error) {
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
