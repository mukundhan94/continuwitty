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

// SessionLifecycleRunInput captures request inputs for lifecycle maintenance.
type SessionLifecycleRunInput struct {
	ActorUserID  uuid.UUID
	Session      models.ChatSessionRecord
	EmbeddingDim int
	Dependencies SessionLifecycleDependencies
	Now          time.Time
}

type sessionLifecycleRun struct {
	ctx                  context.Context
	actorUserID          uuid.UUID
	session              models.ChatSessionRecord
	embeddingDim         int
	dependencies         SessionLifecycleDependencies
	now                  time.Time
	autosaveSnapshots    []models.EngramSummary
	shouldCreateSnapshot bool
	skippedReason        *string
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
	input SessionLifecycleRunInput,
) (LifecycleMaintenanceResult, error) {
	run := newSessionLifecycleRun(ctx, input)
	if !run.autosaveEnabled() {
		return run.disabledResult(), nil
	}
	if err := run.loadAutosaveSnapshots(); err != nil {
		return LifecycleMaintenanceResult{}, err
	}
	if err := run.resolveSnapshotCreationPlan(); err != nil {
		return LifecycleMaintenanceResult{}, err
	}
	createdSnapshotID, err := run.createSnapshotIfNeeded()
	if err != nil {
		return LifecycleMaintenanceResult{}, err
	}
	prunedIDs, err := run.pruneSnapshots()
	if err != nil {
		return LifecycleMaintenanceResult{}, err
	}
	return LifecycleMaintenanceResult{
		SnapshotEngramID: createdSnapshotID,
		PrunedEngramIDs:  prunedIDs,
		SkippedReason:    run.skippedReason,
	}, nil
}

func newSessionLifecycleRun(ctx context.Context, input SessionLifecycleRunInput) sessionLifecycleRun {
	return sessionLifecycleRun{
		ctx:               ctx,
		actorUserID:       input.ActorUserID,
		session:           input.Session,
		embeddingDim:      input.EmbeddingDim,
		dependencies:      input.Dependencies,
		now:               resolveReferenceTime(input.Now),
		autosaveSnapshots: []models.EngramSummary{},
	}
}

func (run *sessionLifecycleRun) autosaveEnabled() bool {
	return run.session.AutosaveEnabled && run.session.AutosaveStrategy != models.ChatAutosaveStrategyOff
}

func (run *sessionLifecycleRun) disabledResult() LifecycleMaintenanceResult {
	return LifecycleMaintenanceResult{
		SnapshotEngramID: nil,
		PrunedEngramIDs:  []uuid.UUID{},
		SkippedReason:    stringPtr("autosave_disabled"),
	}
}

func (run *sessionLifecycleRun) loadAutosaveSnapshots() error {
	autosaveSnapshots, err := listAutosaveSnapshots(
		run.ctx,
		run.actorUserID,
		run.session.SessionID,
		run.dependencies,
	)
	if err != nil {
		return err
	}
	run.autosaveSnapshots = autosaveSnapshots
	return nil
}

func (run *sessionLifecycleRun) resolveSnapshotCreationPlan() error {
	switch run.session.AutosaveStrategy {
	case models.ChatAutosaveStrategyInterval:
		run.resolveIntervalSnapshotCreationPlan()
		return nil
	case models.ChatAutosaveStrategyMessageCount:
		return run.resolveMessageCountSnapshotCreationPlan()
	default:
		run.shouldCreateSnapshot = false
		run.skippedReason = nil
		return nil
	}
}

func (run *sessionLifecycleRun) resolveIntervalSnapshotCreationPlan() {
	latestCreatedAt := latestSnapshotCreatedAt(run.autosaveSnapshots)
	run.shouldCreateSnapshot = ShouldTakeIntervalSnapshot(
		run.now,
		latestCreatedAt,
		SnapshotIntervalMinutes(run.session.AutosaveIntervalMinutes),
	)
	if run.shouldCreateSnapshot {
		run.skippedReason = nil
		return
	}
	run.skippedReason = stringPtr("interval_not_elapsed")
}

func (run *sessionLifecycleRun) resolveMessageCountSnapshotCreationPlan() error {
	assistantMessageCount, err := run.dependencies.CountSessionMessagesByRole(
		run.ctx,
		run.session.SessionID,
		run.actorUserID,
		"assistant",
	)
	if err != nil {
		return err
	}
	run.shouldCreateSnapshot = ShouldTakeMessageCountSnapshot(
		SnapshotMessageCount(assistantMessageCount),
		SnapshotMessageWindow(run.session.AutosaveMinMessages),
	)
	if run.shouldCreateSnapshot {
		run.skippedReason = nil
		return nil
	}
	run.skippedReason = stringPtr("message_count_threshold_not_met")
	return nil
}

func (run *sessionLifecycleRun) createSnapshotIfNeeded() (*uuid.UUID, error) {
	if !run.shouldCreateSnapshot {
		return nil, nil
	}
	createdSnapshotID, err := run.createAutosaveSnapshot()
	if err != nil {
		return nil, err
	}
	if createdSnapshotID == nil {
		run.skippedReason = stringPtr("duplicate_or_low_value_snapshot")
		return nil, nil
	}
	if err := run.loadAutosaveSnapshots(); err != nil {
		return nil, err
	}
	run.skippedReason = nil
	return createdSnapshotID, nil
}

func (run *sessionLifecycleRun) pruneSnapshots() ([]uuid.UUID, error) {
	pruneIDs := SelectRetentionPruneIDs(
		run.autosaveSnapshots,
		SnapshotRetentionDays(run.session.RetentionDays),
		SnapshotRetentionMaxSnapshots(run.session.RetentionMaxSnapshots),
		run.now,
	)
	return run.dependencies.DeleteSessionAutosaveEngrams(
		run.ctx,
		run.session.SessionID,
		run.actorUserID,
		pruneIDs,
	)
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

func latestSnapshotCreatedAt(snapshots []models.EngramSummary) *time.Time {
	if len(snapshots) == 0 {
		return nil
	}
	latest := snapshots[0].CreatedAt
	return &latest
}

func (run *sessionLifecycleRun) createAutosaveSnapshot() (*uuid.UUID, error) {
	messages, err := run.dependencies.ListChatMessages(
		run.ctx,
		run.session.SessionID,
		run.actorUserID,
		500,
		0,
	)
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
	if DuplicateSnapshotExists(abstract, run.autosaveSnapshots) {
		return nil, nil
	}
	now := time.Now().UTC()
	threadID := fmt.Sprintf("chat-session:%s:autosave", run.session.SessionID)
	retrievalText := retrievalTextFromMessages(messages, 8)
	created, err := run.dependencies.CreateEngram(
		run.ctx,
		models.MemoryEngramCreate{
			ProjectID:               run.session.ProjectID,
			ThreadID:                &threadID,
			Title:                   fmt.Sprintf("%s Autosave %s", run.session.Title, now.Format("2006-01-02 15:04:05")),
			Abstract:                abstract,
			DetailedSummaryMarkdown: transcriptMarkdown(run.session, messages),
			Tags: append(
				[]string{},
				append(autosaveSnapshotTags, fmt.Sprintf("autosave_strategy:%s", run.session.AutosaveStrategy))...,
			),
			Keywords:        []string{"autosave", "snapshot", string(run.session.Provider), run.session.ModelID},
			VisibilityScope: string(run.session.VisibilityScope),
			SourceSessionID: &run.session.SessionID,
			RetrievalText:   &retrievalText,
		},
		run.embeddingDim,
		run.actorUserID,
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
