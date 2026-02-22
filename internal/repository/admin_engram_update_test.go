package repository

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"engram/internal/embeddings"
	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestBuildAdminEngramUpdateFieldsUsesPayloadValuesAndDefaults(t *testing.T) {
	current := sampleAdminEngramRecord()
	title := "  Updated title  "
	detailed := "Updated markdown"
	tags := []string{"new-tag"}
	visibility := models.VisibilityScopeProject

	fields := buildAdminEngramUpdateFields(
		current,
		AdminEngramUpdateInput{
			Title:                   &title,
			DetailedSummaryMarkdown: &detailed,
			Tags:                    &tags,
			VisibilityScope:         &visibility,
		},
	)

	requireEqual(t, "Updated title", fields.Title)
	requireEqual(t, current.Abstract, fields.Abstract)
	requireEqual(t, detailed, fields.DetailedSummaryMarkdown)
	if !reflect.DeepEqual([]string{"new-tag"}, fields.Tags) {
		t.Fatalf("expected tags %#v, got %#v", []string{"new-tag"}, fields.Tags)
	}
	if !reflect.DeepEqual(current.Keywords, fields.Keywords) {
		t.Fatalf("expected keywords %#v, got %#v", current.Keywords, fields.Keywords)
	}
	requireEqual(t, models.VisibilityScopeProject, fields.VisibilityScope)
}

func TestBuildAdminEngramRetrievalTextUsesUpdateFields(t *testing.T) {
	fields := adminEngramUpdateFields{
		Title:                   "Updated title",
		Abstract:                "Updated abstract",
		DetailedSummaryMarkdown: "## details",
		Tags:                    []string{"tag-one", "tag-two"},
		Keywords:                []string{"queue", "latency"},
		VisibilityScope:         models.VisibilityScopeProject,
	}

	retrievalText := buildAdminEngramRetrievalText(fields)
	requireEqual(t, "Updated title Updated abstract ## details tag-one tag-two queue latency", retrievalText)
}

func TestBuildAdminEngramJSONPayloadOverridesMutableFields(t *testing.T) {
	current := sampleAdminEngramRecord()
	fields := adminEngramUpdateFields{
		Title:                   "Updated title",
		Abstract:                "Updated abstract",
		DetailedSummaryMarkdown: "Updated markdown",
		Tags:                    []string{"new-tag"},
		Keywords:                []string{"updated-keyword"},
		VisibilityScope:         models.VisibilityScopeProject,
	}
	updatedAt := time.Date(2026, 2, 22, 19, 0, 0, 0, time.UTC)
	originalNow := nowAdminEngramUTC
	nowAdminEngramUTC = func() time.Time { return updatedAt }
	t.Cleanup(func() { nowAdminEngramUTC = originalNow })

	payload := buildAdminEngramJSONPayload(current, fields)

	requireEqual(t, "Updated title", payload["title"])
	requireEqual(t, "Updated abstract", payload["abstract"])
	requireEqual(t, "Updated markdown", payload["detailed_summary_markdown"])
	requireEqual(t, "project", payload["visibility_scope"])
	requireEqual(t, updatedAt.Format("2006-01-02T15:04:05-07:00"), payload["updated_at"].(string))
}

func TestReplaceAdminEngramSourcesReplacesRows(t *testing.T) {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000e11")
	oldSourceID := uuid.MustParse("00000000-0000-0000-0000-000000000e12")
	newSourceA := uuid.MustParse("00000000-0000-0000-0000-000000000e13")
	newSourceB := uuid.MustParse("00000000-0000-0000-0000-000000000e14")
	capturedAt := time.Date(2026, 2, 22, 19, 5, 0, 0, time.UTC)
	title := "source one"
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{values: [][]any{{oldSourceID}}},
		queryRowResults: []*fakeRow{{values: []any{newSourceA}}, {values: []any{newSourceB}}},
	}
	sources := []models.AdminEngramSourceInput{
		{CapturedAt: capturedAt, URL: "https://example.com/1", Title: &title},
		{CapturedAt: capturedAt, URL: "https://example.com/2"},
	}

	generatedIDs := []uuid.UUID{newSourceA, newSourceB}
	originalNewSourceUUID := newAdminSourceUUID
	newAdminSourceUUID = func() uuid.UUID {
		next := generatedIDs[0]
		generatedIDs = generatedIDs[1:]
		return next
	}
	t.Cleanup(func() { newAdminSourceUUID = originalNewSourceUUID })

	err := replaceAdminEngramSources(context.Background(), db, engramID, sources)
	requireNoError(t, err)
	requireEqual(t, 1, len(db.querySQL))
	requireEqual(t, 2, len(db.queryRowSQL))
	if !reflect.DeepEqual([]any{engramID}, db.queryArgs[0]) {
		t.Fatalf("expected delete args %#v, got %#v", []any{engramID}, db.queryArgs[0])
	}
	if db.queryRowArgs[0][0] != newSourceA {
		t.Fatalf("expected first generated source id %s, got %#v", newSourceA, db.queryRowArgs[0][0])
	}
	if db.queryRowArgs[0][1] != engramID {
		t.Fatalf("expected engram id %s in insert args", engramID)
	}
}

func TestUpdateAdminEngramReturnsNilWhenMissing(t *testing.T) {
	db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}

	record, err := UpdateAdminEngram(
		context.Background(),
		db,
		AdminEngramUpdateInput{
			EngramID:     uuid.MustParse("00000000-0000-0000-0000-000000000e21"),
			ActorUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000e22"),
			EmbeddingDim: 8,
		},
	)
	requireNoError(t, err)
	if record != nil {
		t.Fatalf("expected nil when current engram is missing")
	}
	requireEqual(t, 0, len(db.querySQL))
}

func TestUpdateAdminEngramPersistsFieldsAndOptionallySources(t *testing.T) {
	fixture := buildUpdateAdminEngramFixture()
	db := buildUpdateAdminEngramFakeQueryer(fixture)
	setupUpdateAdminEngramStubs(t, fixture)

	record, err := UpdateAdminEngram(
		context.Background(),
		db,
		fixture.Input,
	)
	requireNoError(t, err)
	assertUpdateAdminEngramRecord(t, record)
	assertUpdateAdminEngramWrites(t, fixture, db)
}

type updateAdminEngramFixture struct {
	EngramID          uuid.UUID
	ActorUserID       uuid.UUID
	OwnerUserID       uuid.UUID
	OldSourceID       uuid.UUID
	NewSourceID       uuid.UUID
	CreatedAt         time.Time
	CapturedAt        time.Time
	FixedNow          time.Time
	ExpectedTitle     string
	ExpectedAbstract  string
	ExpectedDetailed  string
	ExpectedTags      []string
	ExpectedKeywords  []string
	ExpectedRetrieval string
	Input             AdminEngramUpdateInput
}

func buildUpdateAdminEngramFixture() updateAdminEngramFixture {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000e31")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000e32")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000e33")
	oldSourceID := uuid.MustParse("00000000-0000-0000-0000-000000000e34")
	newSourceID := uuid.MustParse("00000000-0000-0000-0000-000000000e35")
	createdAt := time.Date(2026, 2, 22, 19, 10, 0, 0, time.UTC)
	capturedAt := time.Date(2026, 2, 22, 19, 11, 0, 0, time.UTC)
	fixedNow := time.Date(2026, 2, 22, 19, 12, 0, 0, time.UTC)
	title := "  Updated title  "
	abstract := "  Updated abstract  "
	detailed := "Updated markdown"
	tags := []string{"new-tag"}
	keywords := []string{"queue", "latency"}
	visibility := models.VisibilityScopeProject
	sources := []models.AdminEngramSourceInput{{
		CapturedAt: capturedAt,
		URL:        "https://example.com/new",
	}}
	return updateAdminEngramFixture{
		EngramID:          engramID,
		ActorUserID:       actorUserID,
		OwnerUserID:       ownerUserID,
		OldSourceID:       oldSourceID,
		NewSourceID:       newSourceID,
		CreatedAt:         createdAt,
		CapturedAt:        capturedAt,
		FixedNow:          fixedNow,
		ExpectedTitle:     "Updated title",
		ExpectedAbstract:  "Updated abstract",
		ExpectedDetailed:  detailed,
		ExpectedTags:      []string{"new-tag"},
		ExpectedKeywords:  []string{"queue", "latency"},
		ExpectedRetrieval: "Updated title Updated abstract Updated markdown new-tag queue latency",
		Input: AdminEngramUpdateInput{
			EngramID:                engramID,
			ActorUserID:             actorUserID,
			Title:                   &title,
			Abstract:                &abstract,
			DetailedSummaryMarkdown: &detailed,
			Tags:                    &tags,
			Keywords:                &keywords,
			VisibilityScope:         &visibility,
			Sources:                 &sources,
			EmbeddingDim:            8,
		},
	}
}

func buildUpdateAdminEngramFakeQueryer(fixture updateAdminEngramFixture) *fakeQueryer {
	return &fakeQueryer{
		queryRowResults: []*fakeRow{
			{values: adminEngramRowValues(
				fixture.EngramID,
				"project-docs",
				nil,
				"Current title",
				"Current abstract",
				"Current markdown",
				[]string{"existing"},
				[]string{"keyword"},
				&fixture.OwnerUserID,
				"private",
				nil,
				fixture.CreatedAt,
				fixture.CreatedAt,
				nil,
				nil,
				nil,
			)},
			{values: []any{fixture.EngramID}},
			{values: []any{fixture.NewSourceID}},
			{values: adminEngramRowValues(
				fixture.EngramID,
				"project-docs",
				nil,
				fixture.ExpectedTitle,
				fixture.ExpectedAbstract,
				fixture.ExpectedDetailed,
				fixture.ExpectedTags,
				fixture.ExpectedKeywords,
				&fixture.OwnerUserID,
				"project",
				nil,
				fixture.CreatedAt,
				fixture.CreatedAt,
				nil,
				nil,
				nil,
			)},
		},
		queryRowsResults: []*fakeRows{
			{values: [][]any{adminEngramSourceRowValues(fixture.OldSourceID, fixture.CapturedAt, "https://example.com/old", nil, nil, nil, nil)}},
			{values: [][]any{{fixture.OldSourceID}}},
			{values: [][]any{adminEngramSourceRowValues(fixture.NewSourceID, fixture.CapturedAt, "https://example.com/new", nil, nil, nil, nil)}},
		},
	}
}

func setupUpdateAdminEngramStubs(t *testing.T, fixture updateAdminEngramFixture) {
	t.Helper()
	originalNow := nowAdminEngramUTC
	nowAdminEngramUTC = func() time.Time { return fixture.FixedNow }
	t.Cleanup(func() { nowAdminEngramUTC = originalNow })

	originalEmbed := embedAdminEngramText
	embedAdminEngramText = func(text string, dim int) (embeddings.Result, error) {
		requireEqual(t, fixture.ExpectedRetrieval, text)
		requireEqual(t, 8, dim)
		return embeddings.Result{
			ProviderID: "deterministic-local",
			Vector:     []float64{1.0, -2.3456789},
		}, nil
	}
	t.Cleanup(func() { embedAdminEngramText = originalEmbed })

	originalNewSourceUUID := newAdminSourceUUID
	newAdminSourceUUID = func() uuid.UUID { return fixture.NewSourceID }
	t.Cleanup(func() { newAdminSourceUUID = originalNewSourceUUID })
}

func assertUpdateAdminEngramRecord(t *testing.T, record *models.AdminEngramRecord) {
	t.Helper()
	requireNotNil(t, record)
	requireEqual(t, "Updated title", record.Title)
	requireEqual(t, models.VisibilityScopeProject, record.VisibilityScope)
	requireEqual(t, 1, len(record.Sources))
}

func assertUpdateAdminEngramWrites(t *testing.T, fixture updateAdminEngramFixture, db *fakeQueryer) {
	t.Helper()
	requireEqual(t, 4, len(db.queryRowArgs))
	updateArgs := db.queryRowArgs[1]
	requireEqual(t, fixture.ExpectedTitle, updateArgs[0].(string))
	requireEqual(t, fixture.ExpectedAbstract, updateArgs[1].(string))
	requireEqual(t, fixture.ExpectedDetailed, updateArgs[2].(string))
	if !reflect.DeepEqual(fixture.ExpectedTags, updateArgs[3]) {
		t.Fatalf("expected tags arg %#v, got %#v", fixture.ExpectedTags, updateArgs[3])
	}
	if !reflect.DeepEqual(fixture.ExpectedKeywords, updateArgs[4]) {
		t.Fatalf("expected keywords arg %#v, got %#v", fixture.ExpectedKeywords, updateArgs[4])
	}
	requireEqual(t, "project", updateArgs[5].(string))
	requireEqual(t, fixture.ExpectedRetrieval, updateArgs[6].(string))
	requireEqual(t, "deterministic-local", updateArgs[7].(string))
	requireEqual(t, "[1.000000,-2.345679]", updateArgs[8].(string))
	requireEqual(t, fixture.ActorUserID, updateArgs[10].(uuid.UUID))
	requireEqual(t, fixture.EngramID, updateArgs[11].(uuid.UUID))

	payloadJSON, ok := updateArgs[9].(string)
	if !ok {
		t.Fatalf("expected engram_json arg to be string, got %#v", updateArgs[9])
	}
	var payload map[string]any
	requireNoError(t, json.Unmarshal([]byte(payloadJSON), &payload))
	requireEqual(t, fixture.ExpectedTitle, payload["title"].(string))
	requireEqual(t, fixture.ExpectedAbstract, payload["abstract"].(string))
	requireEqual(t, "project", payload["visibility_scope"])
	requireEqual(t, fixture.FixedNow.Format("2006-01-02T15:04:05-07:00"), payload["updated_at"].(string))

	requireEqual(t, 3, len(db.queryArgs))
	if !reflect.DeepEqual([]any{fixture.EngramID}, db.queryArgs[1]) {
		t.Fatalf("expected delete-source args %#v, got %#v", []any{fixture.EngramID}, db.queryArgs[1])
	}
	insertArgs := db.queryRowArgs[2]
	requireEqual(t, fixture.NewSourceID, insertArgs[0].(uuid.UUID))
	requireEqual(t, fixture.EngramID, insertArgs[1].(uuid.UUID))
	requireEqual(t, "https://example.com/new", insertArgs[3].(string))
}

func sampleAdminEngramRecord() models.AdminEngramRecord {
	createdAt := time.Date(2026, 2, 22, 19, 0, 0, 0, time.UTC)
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000ef1")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000ef2")
	return models.AdminEngramRecord{
		EngramID:                engramID,
		ProjectID:               "engram-vault",
		Title:                   "Current title",
		Abstract:                "Current abstract",
		DetailedSummaryMarkdown: "Current markdown",
		Tags:                    []string{"existing"},
		Keywords:                []string{"keyword"},
		OwnerUserID:             &ownerUserID,
		VisibilityScope:         models.VisibilityScopePrivate,
		CreatedAt:               createdAt,
		UpdatedAt:               createdAt,
		Sources:                 []models.AdminEngramSourceRecord{},
	}
}
