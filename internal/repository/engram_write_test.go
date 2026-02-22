package repository

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"engram/internal/embeddings"
	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCreateEngramWithReportPersistsEngramSourcesAndArtifacts(t *testing.T) {
	fixture := buildCreateEngramWithReportFixture()
	db := buildCreateEngramWithReportFakeQueryer(fixture)
	setupCreateEngramWithReportStubs(t, fixture)

	response, report, err := CreateEngramWithReport(
		context.Background(),
		db,
		CreateEngramInput{
			Payload:          fixture.InputPayload,
			EmbeddingDim:     8,
			OwnerUserID:      &fixture.OwnerUserID,
			EnrichmentOrigin: "test",
		},
	)
	requireNoError(t, err)
	assertCreateEngramWithReportResult(t, fixture, response, report)
	assertCreateEngramWithReportWrites(t, fixture, db)
}

func TestCreateEngramWithReportReturnsEmbedErrorAndSkipsWrites(t *testing.T) {
	db := &fakeQueryer{}
	expectedErr := errors.New("embed failed")

	originalEmbed := embedEngramText
	embedEngramText = func(_ string, _ int) (embeddings.Result, error) {
		return embeddings.Result{}, expectedErr
	}
	t.Cleanup(func() { embedEngramText = originalEmbed })

	_, _, err := CreateEngramWithReport(
		context.Background(),
		db,
		CreateEngramInput{
			Payload: models.MemoryEngramCreate{
				ProjectID:               "engram-vault",
				Title:                   "Checkpoint",
				DetailedSummaryMarkdown: "Body",
			},
			EmbeddingDim: 8,
		},
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected embed error %v, got %v", expectedErr, err)
	}
	requireEqual(t, 0, len(db.queryRowSQL))
}

func TestCreateEngramReturnsCreatedResponse(t *testing.T) {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000451")
	createdAt := time.Date(2026, 2, 20, 12, 30, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowResults: []*fakeRow{
			{values: []any{engramID}},
		},
	}

	originalNowUTC := nowUTC
	nowUTC = func() time.Time { return createdAt }
	t.Cleanup(func() { nowUTC = originalNowUTC })

	originalEngramUUID := newEngramUUID
	newEngramUUID = func() uuid.UUID { return engramID }
	t.Cleanup(func() { newEngramUUID = originalEngramUUID })

	originalEmbed := embedEngramText
	embedEngramText = func(_ string, _ int) (embeddings.Result, error) {
		return embeddings.Result{Vector: []float64{0.0, 0.0}, ProviderID: "local-deterministic-v1"}, nil
	}
	t.Cleanup(func() { embedEngramText = originalEmbed })

	response, err := CreateEngram(
		context.Background(),
		db,
		CreateEngramInput{
			Payload: models.MemoryEngramCreate{
				ProjectID:               "engram-vault",
				Title:                   "Checkpoint",
				DetailedSummaryMarkdown: "Body",
			},
			EmbeddingDim: 2,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, response)
	requireEqual(t, engramID, response.EngramID)
	requireEqual(t, createdAt, response.CreatedAt)
}

type createEngramWithReportFixture struct {
	EngramID         uuid.UUID
	SourceID         uuid.UUID
	ArtifactID       uuid.UUID
	OwnerUserID      uuid.UUID
	SourceSessionID  uuid.UUID
	CapturedAt       time.Time
	CreatedAt        time.Time
	Retrieval        string
	InputPayload     models.MemoryEngramCreate
	EnrichmentReport map[string]any
}

func buildCreateEngramWithReportFixture() createEngramWithReportFixture {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000401")
	sourceID := uuid.MustParse("00000000-0000-0000-0000-000000000402")
	artifactID := uuid.MustParse("00000000-0000-0000-0000-000000000403")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000404")
	sourceSessionID := uuid.MustParse("00000000-0000-0000-0000-000000000405")
	capturedAt := time.Date(2026, 2, 20, 12, 5, 0, 0, time.UTC)
	createdAt := time.Date(2026, 2, 20, 12, 10, 0, 0, time.UTC)
	retrieval := "override retrieval"
	title := "Primary source"
	return createEngramWithReportFixture{
		EngramID:        engramID,
		SourceID:        sourceID,
		ArtifactID:      artifactID,
		OwnerUserID:     ownerUserID,
		SourceSessionID: sourceSessionID,
		CapturedAt:      capturedAt,
		CreatedAt:       createdAt,
		Retrieval:       retrieval,
		InputPayload: models.MemoryEngramCreate{
			ProjectID:               "engram-vault",
			ThreadID:                ptr("thread-1"),
			Title:                   "Checkpoint Summary",
			Abstract:                "Summary",
			DetailedSummaryMarkdown: "Detailed markdown",
			Claims: []models.Claim{
				{
					Claim: "A claim",
					SupportingSources: []models.SupportingSource{
						{
							URL:        "https://example.com/source",
							Title:      &title,
							Snippet:    "snippet",
							CapturedAt: capturedAt,
						},
					},
				},
			},
			Artifacts: []models.ArtifactIn{
				{
					ArtifactType: "trace",
					StorageURI:   "s3://bucket/trace.json",
					Metadata:     map[string]any{"kind": "trace"},
				},
			},
			Tags:            []string{"tag-a"},
			Keywords:        []string{"keyword-a"},
			RetrievalText:   &retrieval,
			VisibilityScope: "project",
			SourceSessionID: &sourceSessionID,
		},
		EnrichmentReport: map[string]any{
			"schema_version":     "1.0",
			"origin":             "test",
			"enrichment_applied": false,
		},
	}
}

func buildCreateEngramWithReportFakeQueryer(fixture createEngramWithReportFixture) *fakeQueryer {
	return &fakeQueryer{
		queryRowResults: []*fakeRow{
			{values: []any{fixture.EngramID}},
			{values: []any{fixture.SourceID}},
			{values: []any{fixture.ArtifactID}},
		},
	}
}

func setupCreateEngramWithReportStubs(t *testing.T, fixture createEngramWithReportFixture) {
	t.Helper()
	originalNowUTC := nowUTC
	nowUTC = func() time.Time { return fixture.CreatedAt }
	t.Cleanup(func() { nowUTC = originalNowUTC })

	originalEngramUUID := newEngramUUID
	newEngramUUID = func() uuid.UUID { return fixture.EngramID }
	t.Cleanup(func() { newEngramUUID = originalEngramUUID })

	originalWriteUUID := newWriteUUID
	writeIDs := []uuid.UUID{fixture.SourceID, fixture.ArtifactID}
	newWriteUUID = func() uuid.UUID {
		next := writeIDs[0]
		writeIDs = writeIDs[1:]
		return next
	}
	t.Cleanup(func() { newWriteUUID = originalWriteUUID })

	originalEmbed := embedEngramText
	embedEngramText = func(text string, dim int) (embeddings.Result, error) {
		requireEqual(t, fixture.Retrieval, text)
		requireEqual(t, 8, dim)
		return embeddings.Result{
			Vector:     []float64{0.1, 0.2},
			ProviderID: "local-deterministic-v1",
		}, nil
	}
	t.Cleanup(func() { embedEngramText = originalEmbed })

	originalResolve := resolveEnrichedPayload
	resolveEnrichedPayload = func(
		payload models.MemoryEngramCreate,
		enrichmentOrigin string,
	) (models.MemoryEngramCreate, map[string]any) {
		requireEqual(t, "test", enrichmentOrigin)
		return payload, fixture.EnrichmentReport
	}
	t.Cleanup(func() { resolveEnrichedPayload = originalResolve })
}

func assertCreateEngramWithReportResult(
	t *testing.T,
	fixture createEngramWithReportFixture,
	response *models.EngramCreateResponse,
	report map[string]any,
) {
	t.Helper()
	requireNotNil(t, response)
	requireEqual(t, fixture.EngramID, response.EngramID)
	requireEqual(t, fixture.CreatedAt, response.CreatedAt)
	if !reflect.DeepEqual(report, fixture.EnrichmentReport) {
		t.Fatalf("expected enrichment report %#v, got %#v", fixture.EnrichmentReport, report)
	}
}

func assertCreateEngramWithReportWrites(
	t *testing.T,
	fixture createEngramWithReportFixture,
	db *fakeQueryer,
) {
	t.Helper()
	requireEqual(t, 3, len(db.queryRowSQL))
	if !strings.Contains(db.queryRowSQL[0], "INSERT INTO engrams") {
		t.Fatalf("expected engram insert sql, got %q", db.queryRowSQL[0])
	}
	if !strings.Contains(db.queryRowSQL[1], "INSERT INTO sources") {
		t.Fatalf("expected sources insert sql, got %q", db.queryRowSQL[1])
	}
	if !strings.Contains(db.queryRowSQL[2], "INSERT INTO artifacts") {
		t.Fatalf("expected artifacts insert sql, got %q", db.queryRowSQL[2])
	}
	firstArgs := db.queryRowArgs[0]
	if got := firstArgs[11]; got != "project" {
		t.Fatalf("expected visibility_scope arg 'project', got %#v", got)
	}
	if got := firstArgs[13]; got != fixture.Retrieval {
		t.Fatalf("expected retrieval_text arg %q, got %#v", fixture.Retrieval, got)
	}
	if got := firstArgs[14]; got != "local-deterministic-v1" {
		t.Fatalf("expected embedding model arg, got %#v", got)
	}
	if got := firstArgs[15]; got != "[0.100000,0.200000]" {
		t.Fatalf("expected embedding literal arg, got %#v", got)
	}
}
