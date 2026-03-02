package repository

import (
	"context"
	"errors"
	"strings"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RehydrationInput captures lookup arguments for rehydration bundle retrieval.
type RehydrationInput struct {
	EngramID    uuid.UUID
	ActorUserID *uuid.UUID
}

// EngramSourceListInput captures source-list lookup controls.
type EngramSourceListInput struct {
	EngramID    uuid.UUID
	Limit       int
	ActorUserID *uuid.UUID
}

type rehydrationRow struct {
	EngramID        uuid.UUID
	ProjectID       string
	Title           string
	Abstract        string
	EngramJSON      map[string]any
	EngramMarkdown  *string
	OwnerUserID     *uuid.UUID
	VisibilityScope *string
}

type rehydrationContent struct {
	CompactSummary          string
	DetailedSummaryMarkdown string
	Decisions               []map[string]any
	OpenQuestions           []string
	PackedCitations         []models.RehydrationCitation
	ContextMarkdown         string
}

// GetRehydrationBundle returns rehydration context when the actor can access the engram.
func GetRehydrationBundle(
	ctx context.Context,
	db Queryer,
	input RehydrationInput,
) (*models.RehydrationBundle, error) {
	row, err := fetchRehydrationEngramRow(ctx, db, input.EngramID, input.ActorUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	sources, err := fetchRehydrationCitationRows(ctx, db, input.EngramID, 25)
	if err != nil {
		return nil, err
	}
	content := buildRehydrationContent(*row, sources)
	bundle := models.RehydrationBundle{
		EngramID:                row.EngramID,
		ProjectID:               row.ProjectID,
		Title:                   row.Title,
		CompactSummary:          content.CompactSummary,
		DetailedSummaryMarkdown: content.DetailedSummaryMarkdown,
		KeyDecisions:            content.Decisions,
		OpenQuestions:           content.OpenQuestions,
		TopCitations:            content.PackedCitations,
		ContextMarkdown:         content.ContextMarkdown,
		OwnerUserID:             row.OwnerUserID,
		VisibilityScope:         visibilityOrDefault(row.VisibilityScope),
	}
	return &bundle, nil
}

// GetEngramSources returns source records for a visible engram.
func GetEngramSources(
	ctx context.Context,
	db Queryer,
	input EngramSourceListInput,
) ([]models.EngramSourceRecord, error) {
	if input.Limit <= 0 {
		input.Limit = 100
	}
	_, err := fetchRehydrationEngramRow(ctx, db, input.EngramID, input.ActorUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return []models.EngramSourceRecord{}, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(
		ctx,
		`
		SELECT source_id, engram_id, captured_at, url, title, snippet
		FROM sources
		WHERE engram_id = $1
		ORDER BY captured_at DESC
			LIMIT $2
			`,
		input.EngramID,
		input.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]models.EngramSourceRecord, 0)
	for rows.Next() {
		var source models.EngramSourceRecord
		if scanErr := rows.Scan(
			&source.SourceID,
			&source.EngramID,
			&source.CapturedAt,
			&source.URL,
			&source.Title,
			&source.Snippet,
		); scanErr != nil {
			return nil, scanErr
		}
		results = append(results, source)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func fetchRehydrationEngramRow(
	ctx context.Context,
	db Queryer,
	engramID uuid.UUID,
	actorUserID *uuid.UUID,
) (*rehydrationRow, error) {
	whereClauses := []string{"engram_id = $1", "deleted_at IS NULL"}
	params := []any{engramID}
	if actorUserID != nil {
		actorPlaceholder := pgxPlaceholder(len(params) + 1)
		whereClauses = append(
			whereClauses,
			buildMembershipReadClause(membershipReadClauseInput{ownerColumn: "owner_user_id", visibilityColumn: "visibility_scope", projectColumn: "project_id", actorPlaceholder: actorPlaceholder, includeOwnerless: true}),
		)
		params = append(params, *actorUserID)
	}
	sql := `
		SELECT
			engram_id, project_id, title, abstract, engram_json, engram_markdown,
			owner_user_id, visibility_scope
		FROM engrams
		WHERE ` + strings.Join(whereClauses, " AND ")
	row := db.QueryRow(ctx, sql, params...)

	record := rehydrationRow{}
	if err := row.Scan(
		&record.EngramID,
		&record.ProjectID,
		&record.Title,
		&record.Abstract,
		&record.EngramJSON,
		&record.EngramMarkdown,
		&record.OwnerUserID,
		&record.VisibilityScope,
	); err != nil {
		return nil, err
	}
	return &record, nil
}

func fetchRehydrationCitationRows(
	ctx context.Context,
	db Queryer,
	engramID uuid.UUID,
	limit int,
) ([]models.RehydrationCitation, error) {
	rows, err := db.Query(
		ctx,
		`
		SELECT url, title, snippet, captured_at
		FROM sources
		WHERE engram_id = $1
		ORDER BY captured_at DESC
		LIMIT $2
		`,
		engramID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	citations := make([]models.RehydrationCitation, 0)
	for rows.Next() {
		citation := models.RehydrationCitation{}
		if scanErr := rows.Scan(
			&citation.URL,
			&citation.Title,
			&citation.Snippet,
			&citation.CapturedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		citations = append(citations, citation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return citations, nil
}

func buildRehydrationContent(
	row rehydrationRow,
	citations []models.RehydrationCitation,
) rehydrationContent {
	engramJSON := row.EngramJSON
	if engramJSON == nil {
		engramJSON = map[string]any{}
	}

	detailedSummaryMarkdown := detailedSummaryMarkdownValue(engramJSON, row.EngramMarkdown)
	compactSummary := resolveCompactSummary(
		compactSummaryInput{
			abstract:                row.Abstract,
			detailedSummaryMarkdown: detailedSummaryMarkdown,
			maxChars:                800,
		},
	)
	detailedExcerpt := extractDetailedExcerpt(
		detailedExcerptInput{
			markdown: detailedSummaryMarkdown,
			maxChars: 2400,
		},
	)
	decisions := decisionMapsFromAny(engramJSON["decisions"])
	openQuestions := stringSliceFromAny(engramJSON["open_questions"])
	packedCitations := packCitations(
		citationPackInput{
			citations: citations,
			limit:     5,
		},
	)
	contextMarkdown := buildRehydrationContextMarkdown(
		rehydrationContextParts{
			Title:           row.Title,
			CompactSummary:  compactSummary,
			DetailedExcerpt: detailedExcerpt,
			Decisions:       decisions,
			OpenQuestions:   openQuestions,
			Citations:       packedCitations,
		},
	)

	return rehydrationContent{
		CompactSummary:          compactSummary,
		DetailedSummaryMarkdown: detailedSummaryMarkdown,
		Decisions:               decisions,
		OpenQuestions:           openQuestions,
		PackedCitations:         packedCitations,
		ContextMarkdown:         contextMarkdown,
	}
}

func detailedSummaryMarkdownValue(engramJSON map[string]any, engramMarkdown *string) string {
	if detailed, ok := engramJSON["detailed_summary_markdown"].(string); ok && strings.TrimSpace(detailed) != "" {
		return strings.TrimSpace(detailed)
	}
	if engramMarkdown != nil {
		return strings.TrimSpace(*engramMarkdown)
	}
	return ""
}

func decisionMapsFromAny(value any) []map[string]any {
	switch typed := value.(type) {
	case []map[string]any:
		return typed
	case []any:
		decisions := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			if decision, ok := item.(map[string]any); ok {
				decisions = append(decisions, decision)
			}
		}
		return decisions
	default:
		return []map[string]any{}
	}
}
