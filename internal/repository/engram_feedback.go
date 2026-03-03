package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	errEngramFeedbackEngramIDRequired = errors.New("engram id is required")
	errEngramFeedbackSessionIDInvalid = errors.New("session id must be non-empty when provided")
	errEngramFeedbackActorIDRequired  = errors.New("actor user id is required")
	errEngramFeedbackTypeRequired     = errors.New("feedback type is required")
	errEngramFeedbackRelevanceInvalid = errors.New("relevance score must be between 1 and 5")

	newEngramFeedbackUUID = uuid.New
	nowEngramFeedbackUTC  = func() time.Time { return time.Now().UTC() }
)

// EngramFeedbackCreateInput captures feedback write-path dependencies.
type EngramFeedbackCreateInput struct {
	EngramID         uuid.UUID
	SessionID        *uuid.UUID
	ActorUserID      uuid.UUID
	FeedbackType     models.EngramFeedbackType
	IntegrationDepth *models.EngramFeedbackIntegrationDepth
	Note             *string
	RelevanceScore   *int
	CreatedAt        time.Time
}

// RecordEngramFeedback persists explicit feedback and updates aggregate counters.
func RecordEngramFeedback(
	ctx context.Context,
	db Queryer,
	input EngramFeedbackCreateInput,
) (*models.EngramFeedbackRecord, error) {
	normalized, err := normalizeEngramFeedbackInput(input)
	if err != nil {
		return nil, err
	}
	noteValue := feedbackNoteSQLValue(normalized.Note)
	integrationDepthValue := feedbackIntegrationDepthSQLValue(normalized.IntegrationDepth)
	relevanceScoreValue := feedbackRelevanceScoreSQLValue(normalized.RelevanceScore)
	authoritySignalValue := feedbackAuthoritySignalSQLValue(
		feedbackAuthoritySignal(
			normalized.RelevanceScore,
			normalized.FeedbackType,
			normalized.IntegrationDepth,
		),
	)
	row := db.QueryRow(
		ctx,
		recordEngramFeedbackSQL(),
		newEngramFeedbackUUID(),
		normalized.EngramID,
		feedbackSessionIDSQLValue(normalized.SessionID),
		normalized.ActorUserID,
		string(normalized.FeedbackType),
		integrationDepthValue,
		noteValue,
		relevanceScoreValue,
		authoritySignalValue,
		normalized.CreatedAt,
	)

	record := models.EngramFeedbackRecord{}
	if err := row.Scan(
		&record.FeedbackID,
		&record.EngramID,
		&record.SessionID,
		&record.ActorUserID,
		&record.FeedbackType,
		&record.IntegrationDepth,
		&record.Note,
		&record.RelevanceScore,
		&record.CreatedAt,
		&record.UsefulCount,
		&record.ContradictionCount,
		&record.FeedbackCount,
		&record.AvgRelevanceFeedback,
	); errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &record, nil
}

func normalizeEngramFeedbackInput(
	input EngramFeedbackCreateInput,
) (EngramFeedbackCreateInput, error) {
	if input.EngramID == uuid.Nil {
		return EngramFeedbackCreateInput{}, errEngramFeedbackEngramIDRequired
	}
	if input.ActorUserID == uuid.Nil {
		return EngramFeedbackCreateInput{}, errEngramFeedbackActorIDRequired
	}
	sessionID, err := normalizeFeedbackSessionID(input.SessionID)
	if err != nil {
		return EngramFeedbackCreateInput{}, err
	}
	input.SessionID = sessionID
	feedbackType, err := normalizeFeedbackType(input.FeedbackType)
	if err != nil {
		return EngramFeedbackCreateInput{}, err
	}
	input.FeedbackType = feedbackType
	integrationDepth, err := normalizeFeedbackIntegrationDepth(input.IntegrationDepth)
	if err != nil {
		return EngramFeedbackCreateInput{}, err
	}
	input.IntegrationDepth = integrationDepth
	input.Note = normalizeFeedbackNote(input.Note)
	relevanceScore, err := normalizeFeedbackRelevanceScore(input.RelevanceScore)
	if err != nil {
		return EngramFeedbackCreateInput{}, err
	}
	input.RelevanceScore = relevanceScore
	return applyDefaultFeedbackCreatedAt(input), nil
}

func normalizeFeedbackType(
	feedbackType models.EngramFeedbackType,
) (models.EngramFeedbackType, error) {
	if strings.TrimSpace(string(feedbackType)) == "" {
		return "", errEngramFeedbackTypeRequired
	}
	return models.ParseEngramFeedbackType(string(feedbackType))
}

func applyDefaultFeedbackCreatedAt(
	input EngramFeedbackCreateInput,
) EngramFeedbackCreateInput {
	if input.CreatedAt.IsZero() {
		input.CreatedAt = nowEngramFeedbackUTC()
	}
	return input
}

func normalizeFeedbackNote(note *string) *string {
	if note == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*note)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func feedbackNoteSQLValue(note *string) any {
	if note == nil {
		return nil
	}
	return *note
}

func feedbackIntegrationDepthSQLValue(depth *models.EngramFeedbackIntegrationDepth) any {
	if depth == nil {
		return nil
	}
	return string(*depth)
}

func feedbackRelevanceScoreSQLValue(relevanceScore *int) any {
	if relevanceScore == nil {
		return nil
	}
	return *relevanceScore
}

func feedbackAuthoritySignalSQLValue(authoritySignal *float64) any {
	if authoritySignal == nil {
		return nil
	}
	return *authoritySignal
}

func feedbackSessionIDSQLValue(sessionID *uuid.UUID) any {
	if sessionID == nil {
		return nil
	}
	return *sessionID
}

func normalizeFeedbackSessionID(sessionID *uuid.UUID) (*uuid.UUID, error) {
	if sessionID == nil {
		return nil, nil
	}
	if *sessionID == uuid.Nil {
		return nil, errEngramFeedbackSessionIDInvalid
	}
	normalized := *sessionID
	return &normalized, nil
}

func normalizeFeedbackRelevanceScore(relevanceScore *int) (*int, error) {
	if relevanceScore == nil {
		return nil, nil
	}
	if *relevanceScore < 1 || *relevanceScore > 5 {
		return nil, errEngramFeedbackRelevanceInvalid
	}
	normalized := *relevanceScore
	return &normalized, nil
}

func feedbackAuthoritySignal(
	relevanceScore *int,
	feedbackType models.EngramFeedbackType,
	integrationDepth *models.EngramFeedbackIntegrationDepth,
) *float64 {
	if relevanceScore != nil {
		value := clamp01(float64(*relevanceScore-1) / 4.0)
		return &value
	}
	if integrationDepth != nil {
		value := authoritySignalFromIntegrationDepth(*integrationDepth)
		return &value
	}
	value := authoritySignalFromFeedbackType(feedbackType)
	return &value
}

func authoritySignalFromIntegrationDepth(
	integrationDepth models.EngramFeedbackIntegrationDepth,
) float64 {
	switch integrationDepth {
	case models.EngramFeedbackIntegrationDepthElaborated:
		return 0.85
	case models.EngramFeedbackIntegrationDepthMentioned:
		return 0.70
	case models.EngramFeedbackIntegrationDepthIgnored:
		return 0.35
	case models.EngramFeedbackIntegrationDepthContradicted:
		return 0.10
	default:
		return 0.50
	}
}

func authoritySignalFromFeedbackType(
	feedbackType models.EngramFeedbackType,
) float64 {
	switch feedbackType {
	case models.EngramFeedbackTypeUseful:
		return 0.60
	case models.EngramFeedbackTypeContradiction:
		return 0.15
	default:
		return 0.50
	}
}

func normalizeFeedbackIntegrationDepth(
	integrationDepth *models.EngramFeedbackIntegrationDepth,
) (*models.EngramFeedbackIntegrationDepth, error) {
	if integrationDepth == nil {
		return nil, nil
	}
	parsed, err := models.ParseEngramFeedbackIntegrationDepth(string(*integrationDepth))
	if err != nil {
		return nil, err
	}
	normalized := parsed
	return &normalized, nil
}

func recordEngramFeedbackSQL() string {
	actorPlaceholder := pgxPlaceholder(4)
	engramReadClause := buildMembershipReadClause(
		membershipReadClauseInput{
			ownerColumn:      "e.owner_user_id",
			visibilityColumn: "e.visibility_scope",
			projectColumn:    "e.project_id",
			actorPlaceholder: actorPlaceholder,
			includeOwnerless: true,
		},
	)
	return strings.ReplaceAll(
		recordEngramFeedbackSQLTemplate,
		"{{ENGRAM_READ_CLAUSE}}",
		engramReadClause,
	)
}

const recordEngramFeedbackSQLTemplate = `
		WITH visible_engram AS (
			SELECT e.engram_id
			FROM engrams e
			WHERE
				e.engram_id = $2
				AND e.deleted_at IS NULL
				AND ({{ENGRAM_READ_CLAUSE}})
		),
		inserted AS (
			INSERT INTO engram_feedback (
				feedback_id,
				engram_id,
				session_id,
				actor_user_id,
				feedback_type,
				integration_depth,
				note,
				relevance_score,
				created_at
			)
			SELECT
				$1,
				ve.engram_id,
				$3,
				$4,
				$5,
				$6,
				COALESCE($7, ''),
				$8,
				$10
			FROM visible_engram ve
			RETURNING
				feedback_id,
				engram_id,
				session_id,
				actor_user_id,
				feedback_type,
				integration_depth,
				note,
				relevance_score,
				created_at
		),
		updated AS (
			UPDATE engrams e
			SET
				useful_count = COALESCE(useful_count, 0) + CASE WHEN $5 = 'useful' THEN 1 ELSE 0 END,
				contradiction_count = COALESCE(contradiction_count, 0) + CASE WHEN $5 = 'contradiction' THEN 1 ELSE 0 END,
				feedback_count = COALESCE(feedback_count, 0) + 1,
				avg_relevance_feedback = CASE
					WHEN $8 IS NULL THEN avg_relevance_feedback
					ELSE (
						(COALESCE(avg_relevance_feedback, 0.0) * COALESCE(feedback_count, 0)::DOUBLE PRECISION) +
						$8::DOUBLE PRECISION
					) / (COALESCE(feedback_count, 0)::DOUBLE PRECISION + 1.0)
				END,
				source_session_quality_score = CASE
					WHEN $9 IS NULL THEN source_session_quality_score
					ELSE (
						(COALESCE(source_session_quality_score, 0.5) * COALESCE(feedback_count, 0)::DOUBLE PRECISION) +
						$9::DOUBLE PRECISION
					) / (COALESCE(feedback_count, 0)::DOUBLE PRECISION + 1.0)
				END,
				updated_at = GREATEST(updated_at, $10)
			FROM inserted i
			WHERE e.engram_id = i.engram_id
			RETURNING
				e.useful_count,
				e.contradiction_count,
				e.feedback_count,
				e.avg_relevance_feedback
		)
		SELECT
			i.feedback_id,
			i.engram_id,
			i.session_id,
			i.actor_user_id,
			i.feedback_type,
			i.integration_depth,
			i.note,
			i.relevance_score,
			i.created_at,
			u.useful_count,
			u.contradiction_count,
			u.feedback_count,
			u.avg_relevance_feedback
		FROM inserted i
		JOIN updated u ON TRUE;
	`
