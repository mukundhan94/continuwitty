package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	errEngramAccessEngramIDRequired = errors.New("engram id is required")

	newEngramAccessEventUUID = uuid.New
	nowEngramAccessUTC       = func() time.Time { return time.Now().UTC() }
)

// EngramAccessSource identifies the surface that used an engram.
type EngramAccessSource string

const (
	EngramAccessSourceUnknown    EngramAccessSource = "unknown"
	EngramAccessSourceChatSend   EngramAccessSource = "chat_send"
	EngramAccessSourceChatStream EngramAccessSource = "chat_stream"
)

// EngramAccessEventRecord captures a single engram access event for persistence.
type EngramAccessEventRecord struct {
	EngramID     uuid.UUID
	SessionID    *uuid.UUID
	AccessSource EngramAccessSource
	AccessedAt   time.Time
}

// RecordEngramAccessEvents appends access events and updates aggregate counters.
func RecordEngramAccessEvents(
	ctx context.Context,
	db Queryer,
	events []EngramAccessEventRecord,
) error {
	for _, rawEvent := range events {
		event, err := normalizeEngramAccessEvent(rawEvent)
		if err != nil {
			return err
		}
		if err := recordSingleEngramAccessEvent(ctx, db, event); err != nil {
			return err
		}
	}
	return nil
}

func normalizeEngramAccessEvent(
	event EngramAccessEventRecord,
) (EngramAccessEventRecord, error) {
	if event.EngramID == uuid.Nil {
		return EngramAccessEventRecord{}, errEngramAccessEngramIDRequired
	}
	if strings.TrimSpace(string(event.AccessSource)) == "" {
		event.AccessSource = EngramAccessSourceUnknown
	}
	if event.AccessedAt.IsZero() {
		event.AccessedAt = nowEngramAccessUTC()
	}
	return event, nil
}

func recordSingleEngramAccessEvent(
	ctx context.Context,
	db Queryer,
	event EngramAccessEventRecord,
) error {
	eventID := newEngramAccessEventUUID()
	updated := false
	err := db.QueryRow(
		ctx,
		`
		WITH inserted AS (
			INSERT INTO engram_access_events (
				event_id,
				engram_id,
				session_id,
				access_source,
				accessed_at,
				created_at
			)
			VALUES ($1, $2, $3, $4, $5, $5)
		),
		updated AS (
			UPDATE engrams
			SET
				access_count = COALESCE(access_count, 0) + 1,
				last_accessed_at = CASE
					WHEN last_accessed_at IS NULL THEN $5
					WHEN $5 > last_accessed_at THEN $5
					ELSE last_accessed_at
				END,
				updated_at = GREATEST(updated_at, $5)
			WHERE engram_id = $2
			RETURNING engram_id
		)
		SELECT EXISTS(SELECT 1 FROM updated);
		`,
		eventID,
		event.EngramID,
		event.SessionID,
		string(event.AccessSource),
		event.AccessedAt,
	).Scan(&updated)
	if err != nil {
		return err
	}
	return nil
}
