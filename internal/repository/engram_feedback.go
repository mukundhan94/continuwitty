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
	errEngramFeedbackActorIDRequired  = errors.New("actor user id is required")
	errEngramFeedbackTypeRequired     = errors.New("feedback type is required")

	newEngramFeedbackUUID = uuid.New
	nowEngramFeedbackUTC  = func() time.Time { return time.Now().UTC() }
)

// EngramFeedbackCreateInput captures feedback write-path dependencies.
type EngramFeedbackCreateInput struct {
	EngramID     uuid.UUID
	ActorUserID  uuid.UUID
	FeedbackType models.EngramFeedbackType
	Note         *string
	CreatedAt    time.Time
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
	row := db.QueryRow(
		ctx,
		recordEngramFeedbackSQL(),
		newEngramFeedbackUUID(),
		normalized.EngramID,
		normalized.ActorUserID,
		string(normalized.FeedbackType),
		noteValue,
		normalized.CreatedAt,
	)

	record := models.EngramFeedbackRecord{}
	if err := row.Scan(
		&record.FeedbackID,
		&record.EngramID,
		&record.ActorUserID,
		&record.FeedbackType,
		&record.Note,
		&record.CreatedAt,
		&record.UsefulCount,
		&record.ContradictionCount,
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
	if strings.TrimSpace(string(input.FeedbackType)) == "" {
		return EngramFeedbackCreateInput{}, errEngramFeedbackTypeRequired
	}
	parsedType, err := models.ParseEngramFeedbackType(string(input.FeedbackType))
	if err != nil {
		return EngramFeedbackCreateInput{}, err
	}
	input.FeedbackType = parsedType
	input.Note = normalizeFeedbackNote(input.Note)
	if input.CreatedAt.IsZero() {
		input.CreatedAt = nowEngramFeedbackUTC()
	}
	return input, nil
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

func recordEngramFeedbackSQL() string {
	actorPlaceholder := pgxPlaceholder(3)
	engramReadClause := buildMembershipReadClause(
		membershipReadClauseInput{
			ownerColumn:      "e.owner_user_id",
			visibilityColumn: "e.visibility_scope",
			projectColumn:    "e.project_id",
			actorPlaceholder: actorPlaceholder,
			includeOwnerless: true,
		},
	)
	return `
		WITH visible_engram AS (
			SELECT e.engram_id
			FROM engrams e
			WHERE
				e.engram_id = $2
				AND e.deleted_at IS NULL
				AND (` + engramReadClause + `)
		),
		inserted AS (
			INSERT INTO engram_feedback (
				feedback_id,
				engram_id,
				actor_user_id,
				feedback_type,
				note,
				created_at
			)
			SELECT
				$1,
				ve.engram_id,
				$3,
				$4,
				COALESCE($5, ''),
				$6
			FROM visible_engram ve
			RETURNING feedback_id, engram_id, actor_user_id, feedback_type, note, created_at
		),
		updated AS (
			UPDATE engrams e
			SET
				useful_count = COALESCE(useful_count, 0) + CASE WHEN $4 = 'useful' THEN 1 ELSE 0 END,
				contradiction_count = COALESCE(contradiction_count, 0) + CASE WHEN $4 = 'contradiction' THEN 1 ELSE 0 END,
				updated_at = GREATEST(updated_at, $6)
			FROM inserted i
			WHERE e.engram_id = i.engram_id
			RETURNING e.useful_count, e.contradiction_count
		)
		SELECT
			i.feedback_id,
			i.engram_id,
			i.actor_user_id,
			i.feedback_type,
			i.note,
			i.created_at,
			u.useful_count,
			u.contradiction_count
		FROM inserted i
		JOIN updated u ON TRUE;
	`
}
