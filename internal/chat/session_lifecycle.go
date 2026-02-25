package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

var autosaveSnapshotTags = []string{"autosave_snapshot", "session-lifecycle", "chat"}

// SessionLifecycleDependencies captures storage and engram operations used by lifecycle maintenance.
type SessionLifecycleDependencies struct {
	ListSessionLinkedEngrams     func(ctx context.Context, sessionID uuid.UUID, actorUserID uuid.UUID, limit int, offset int) ([]models.EngramSummary, error)
	ListChatMessages             func(ctx context.Context, sessionID uuid.UUID, actorUserID uuid.UUID, limit int, offset int) ([]models.ChatMessageRecord, error)
	CountSessionMessagesByRole   func(ctx context.Context, sessionID uuid.UUID, actorUserID uuid.UUID, role string) (int, error)
	DeleteSessionAutosaveEngrams func(ctx context.Context, sessionID uuid.UUID, actorUserID uuid.UUID, engramIDs []uuid.UUID) ([]uuid.UUID, error)
	CreateEngram                 func(ctx context.Context, payload models.MemoryEngramCreate, embeddingDim int, ownerUserID uuid.UUID, enrichmentOrigin string) (*models.EngramCreateResponse, error)
}

// LifecycleMaintenanceResult captures snapshot creation and pruning outputs.
type LifecycleMaintenanceResult struct {
	SnapshotEngramID *uuid.UUID
	PrunedEngramIDs  []uuid.UUID
	SkippedReason    *string
}

// DeriveChatSnapshotAbstract derives a concise abstract from latest assistant/user messages.
func DeriveChatSnapshotAbstract(messages []models.ChatMessageRecord, maxChars int) string {
	normalizedMaxChars := normalizeMaxChars(maxChars)
	for _, role := range []string{"assistant", "user"} {
		for index := len(messages) - 1; index >= 0; index-- {
			message := messages[index]
			if message.Role != role {
				continue
			}
			normalized := normalizeSpaces(message.ContentText)
			if normalized == "" {
				continue
			}
			return truncateText(normalized, normalizedMaxChars)
		}
	}
	return ""
}

// RunSessionLifecycleMaintenance creates autosave snapshots and applies retention pruning.
func RunSessionLifecycleMaintenance(
	ctx context.Context,
	actorUserID uuid.UUID,
	session models.ChatSessionRecord,
	embeddingDim int,
	dependencies SessionLifecycleDependencies,
) (LifecycleMaintenanceResult, error) {
	if !session.AutosaveEnabled || session.AutosaveStrategy == models.ChatAutosaveStrategyOff {
		return LifecycleMaintenanceResult{
			SnapshotEngramID: nil,
			PrunedEngramIDs:  []uuid.UUID{},
			SkippedReason:    stringPtr("autosave_disabled"),
		}, nil
	}
	now := time.Now().UTC()
	autosaveSnapshots, err := listAutosaveSnapshots(ctx, actorUserID, session.SessionID, dependencies)
	if err != nil {
		return LifecycleMaintenanceResult{}, err
	}
	shouldCreate, skippedReason, err := resolveSnapshotCreation(ctx, actorUserID, session, autosaveSnapshots, now, dependencies)
	if err != nil {
		return LifecycleMaintenanceResult{}, err
	}
	createdSnapshotID, refreshedSnapshots, updatedSkippedReason, err := maybeCreateSnapshot(
		ctx,
		actorUserID,
		session,
		autosaveSnapshots,
		shouldCreate,
		skippedReason,
		embeddingDim,
		dependencies,
	)
	if err != nil {
		return LifecycleMaintenanceResult{}, err
	}
	pruneIDs := SelectRetentionPruneIDs(refreshedSnapshots, session.RetentionDays, session.RetentionMaxSnapshots, now)
	prunedIDs, err := dependencies.DeleteSessionAutosaveEngrams(
		ctx,
		session.SessionID,
		actorUserID,
		pruneIDs,
	)
	if err != nil {
		return LifecycleMaintenanceResult{}, err
	}
	return LifecycleMaintenanceResult{
		SnapshotEngramID: createdSnapshotID,
		PrunedEngramIDs:  prunedIDs,
		SkippedReason:    updatedSkippedReason,
	}, nil
}

func listAutosaveSnapshots(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
	dependencies SessionLifecycleDependencies,
) ([]models.EngramSummary, error) {
	linked, err := dependencies.ListSessionLinkedEngrams(ctx, sessionID, actorUserID, 500, 0)
	if err != nil {
		return nil, err
	}
	autosave := make([]models.EngramSummary, 0)
	for _, item := range linked {
		if !isAutosaveSnapshot(item) {
			continue
		}
		autosave = append(autosave, item)
	}
	return autosave, nil
}

func isAutosaveSnapshot(summary models.EngramSummary) bool {
	for _, tag := range summary.Tags {
		if strings.ToLower(strings.TrimSpace(tag)) == "autosave_snapshot" {
			return true
		}
	}
	return false
}

func resolveSnapshotCreation(
	ctx context.Context,
	actorUserID uuid.UUID,
	session models.ChatSessionRecord,
	autosaveSnapshots []models.EngramSummary,
	now time.Time,
	dependencies SessionLifecycleDependencies,
) (bool, *string, error) {
	if session.AutosaveStrategy == models.ChatAutosaveStrategyInterval {
		latestCreatedAt := latestSnapshotCreatedAt(autosaveSnapshots)
		shouldCreate := ShouldTakeIntervalSnapshot(now, latestCreatedAt, session.AutosaveIntervalMinutes)
		if !shouldCreate {
			return false, stringPtr("interval_not_elapsed"), nil
		}
		return true, nil, nil
	}
	if session.AutosaveStrategy == models.ChatAutosaveStrategyMessageCount {
		assistantMessageCount, err := dependencies.CountSessionMessagesByRole(
			ctx,
			session.SessionID,
			actorUserID,
			"assistant",
		)
		if err != nil {
			return false, nil, err
		}
		shouldCreate := ShouldTakeMessageCountSnapshot(assistantMessageCount, session.AutosaveMinMessages)
		if !shouldCreate {
			return false, stringPtr("message_count_threshold_not_met"), nil
		}
		return true, nil, nil
	}
	return false, nil, nil
}

func latestSnapshotCreatedAt(snapshots []models.EngramSummary) *time.Time {
	if len(snapshots) == 0 {
		return nil
	}
	latest := snapshots[0].CreatedAt
	return &latest
}

func maybeCreateSnapshot(
	ctx context.Context,
	actorUserID uuid.UUID,
	session models.ChatSessionRecord,
	autosaveSnapshots []models.EngramSummary,
	shouldCreate bool,
	skippedReason *string,
	embeddingDim int,
	dependencies SessionLifecycleDependencies,
) (*uuid.UUID, []models.EngramSummary, *string, error) {
	if !shouldCreate {
		return nil, autosaveSnapshots, skippedReason, nil
	}
	createdSnapshotID, err := createAutosaveSnapshot(
		ctx,
		actorUserID,
		session,
		autosaveSnapshots,
		embeddingDim,
		dependencies,
	)
	if err != nil {
		return nil, autosaveSnapshots, skippedReason, err
	}
	if createdSnapshotID == nil {
		return nil, autosaveSnapshots, stringPtr("duplicate_or_low_value_snapshot"), nil
	}
	refreshedSnapshots, err := listAutosaveSnapshots(ctx, actorUserID, session.SessionID, dependencies)
	if err != nil {
		return nil, autosaveSnapshots, skippedReason, err
	}
	return createdSnapshotID, refreshedSnapshots, skippedReason, nil
}

func createAutosaveSnapshot(
	ctx context.Context,
	actorUserID uuid.UUID,
	session models.ChatSessionRecord,
	existingSnapshots []models.EngramSummary,
	embeddingDim int,
	dependencies SessionLifecycleDependencies,
) (*uuid.UUID, error) {
	messages, err := dependencies.ListChatMessages(ctx, session.SessionID, actorUserID, 500, 0)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, nil
	}
	abstract := DeriveChatSnapshotAbstract(messages, 320)
	if IsLowValueSnapshotAbstract(abstract) {
		return nil, nil
	}
	if DuplicateSnapshotExists(abstract, existingSnapshots) {
		return nil, nil
	}
	now := time.Now().UTC()
	threadID := fmt.Sprintf("chat-session:%s:autosave", session.SessionID)
	retrievalText := retrievalTextFromMessages(messages, 8)
	created, err := dependencies.CreateEngram(
		ctx,
		models.MemoryEngramCreate{
			ProjectID:               session.ProjectID,
			ThreadID:                &threadID,
			Title:                   fmt.Sprintf("%s Autosave %s", session.Title, now.Format("2006-01-02 15:04:05")),
			Abstract:                abstract,
			DetailedSummaryMarkdown: transcriptMarkdown(session, messages),
			Tags:                    append([]string{}, append(autosaveSnapshotTags, fmt.Sprintf("autosave_strategy:%s", session.AutosaveStrategy))...),
			Keywords:                []string{"autosave", "snapshot", string(session.Provider), session.ModelID},
			VisibilityScope:         string(session.VisibilityScope),
			SourceSessionID:         &session.SessionID,
			RetrievalText:           &retrievalText,
		},
		embeddingDim,
		actorUserID,
		"chat.autosave_snapshot",
	)
	if err != nil {
		return nil, err
	}
	return &created.EngramID, nil
}

func transcriptMarkdown(session models.ChatSessionRecord, messages []models.ChatMessageRecord) string {
	lines := []string{
		"# Chat Session Snapshot: " + session.Title,
		"",
		"- Session ID: " + session.SessionID.String(),
		"- Provider/Model: " + string(session.Provider) + "/" + session.ModelID,
		"",
	}
	for _, message := range messages {
		lines = append(lines,
			"## "+strings.ToUpper(message.Role)+" ("+message.CreatedAt.Format(time.RFC3339)+")",
			normalizeEmptyMessage(message.ContentText),
			"",
		)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func normalizeEmptyMessage(contentText string) string {
	trimmed := strings.TrimSpace(contentText)
	if trimmed == "" {
		return "(empty)"
	}
	return trimmed
}

func retrievalTextFromMessages(messages []models.ChatMessageRecord, tailCount int) string {
	if tailCount <= 0 {
		return ""
	}
	start := len(messages) - tailCount
	if start < 0 {
		start = 0
	}
	parts := make([]string, 0)
	for _, message := range messages[start:] {
		trimmed := strings.TrimSpace(message.ContentText)
		if trimmed == "" {
			continue
		}
		parts = append(parts, trimmed)
	}
	return strings.Join(parts, " ")
}

func truncateText(value string, maxChars int) string {
	if len(value) <= maxChars {
		return value
	}
	if maxChars <= 3 {
		return value[:maxChars]
	}
	return strings.TrimRight(value[:maxChars-3], " ") + "..."
}

func normalizeMaxChars(maxChars int) int {
	if maxChars <= 0 {
		return 320
	}
	return maxChars
}
