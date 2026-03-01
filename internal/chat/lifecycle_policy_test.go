package chat

import (
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestNormalizeAutosavePolicyKeepsBackwardCompatibility(t *testing.T) {
	enabled, strategy := NormalizeAutosavePolicy(true, models.ChatAutosaveStrategyOff)
	requireTrue(t, enabled)
	requireEqualString(t, models.ChatAutosaveStrategyInterval, strategy)

	enabledDisabled, strategyDisabled := NormalizeAutosavePolicy(false, models.ChatAutosaveStrategyMessageCount)
	requireFalse(t, enabledDisabled)
	requireEqualString(t, models.ChatAutosaveStrategyOff, strategyDisabled)
}

func TestIntervalSnapshotTriggerRules(t *testing.T) {
	now := time.Date(2026, 2, 25, 15, 0, 0, 0, time.UTC)
	requireTrue(t, ShouldTakeIntervalSnapshot(now, nil, SnapshotIntervalMinutes(15)))

	fiveMinutesAgo := now.Add(-5 * time.Minute)
	requireFalse(t, ShouldTakeIntervalSnapshot(now, &fiveMinutesAgo, SnapshotIntervalMinutes(15)))

	twentyMinutesAgo := now.Add(-20 * time.Minute)
	requireTrue(t, ShouldTakeIntervalSnapshot(now, &twentyMinutesAgo, SnapshotIntervalMinutes(15)))
}

func TestMessageCountSnapshotTriggerRules(t *testing.T) {
	requireTrue(t, ShouldTakeMessageCountSnapshot(SnapshotMessageCount(6), SnapshotMessageWindow(3)))
	requireFalse(t, ShouldTakeMessageCountSnapshot(SnapshotMessageCount(5), SnapshotMessageWindow(3)))
}

func TestDuplicateAndLowValueGuards(t *testing.T) {
	snapshots := []models.EngramSummary{snapshotSummary(10, "Snapshot", "Primary issue is cache invalidation lag.")}
	requireTrue(t, DuplicateSnapshotExists("Primary issue is cache invalidation lag.", snapshots))
	requireTrue(t, IsLowValueSnapshotAbstract("too short"))
	requireFalse(t, IsLowValueSnapshotAbstract("Primary issue is cache invalidation lag causing stale reads in checkout flow."))
}

func TestRetentionPruningRespectsAgeAndMaxCount(t *testing.T) {
	now := time.Date(2026, 2, 25, 15, 0, 0, 0, time.UTC)
	snapshots := make([]models.EngramSummary, 0, 6)
	for index := 0; index < 6; index++ {
		snapshots = append(
			snapshots,
			models.EngramSummary{
				EngramID:  uuid.MustParse(nextTestUUID(index + 1)),
				ProjectID: "project-a",
				ThreadID:  stringPtr("chat-session:abc:autosave"),
				Title:     "Snapshot " + nextIndexLabel(index),
				Abstract:  "Useful long summary for retention checks.",
				CreatedAt: now.AddDate(0, 0, -index),
				Tags:      []string{"autosave_snapshot"},
				Keywords:  []string{"snapshot"},
			},
		)
	}
	pruneIDs := SelectRetentionPruneIDs(
		snapshots,
		SnapshotRetentionDays(2),
		SnapshotRetentionMaxSnapshots(3),
		now,
	)
	if len(pruneIDs) < 3 {
		t.Fatalf("expected at least three prune ids, got %d", len(pruneIDs))
	}
}

func TestTimelineEventClassification(t *testing.T) {
	requireEqualString(t, "autosave_snapshot", ClassifyTimelineEventType([]string{"autosave_snapshot", "chat"}))
	requireEqualString(t, "consolidation", ClassifyTimelineEventType([]string{"consolidated", "maintenance"}))
	requireEqualString(t, "consolidation_group", ClassifyTimelineEventType([]string{"consolidated", "consolidation_group_key:incident-42"}))
	requireEqualString(t, "consolidation_merge", ClassifyTimelineEventType([]string{"consolidated", "consolidation_group_key:incident-42", "consolidation_merged_count:3"}))
	requireEqualString(t, "manual_snapshot", ClassifyTimelineEventType([]string{"incident", "handoff"}))

	semantics := ClassifyTimelineEvent([]string{"consolidated", "consolidation_group_key:incident-42", "consolidation_merged_count:3"})
	requireEqualString(t, "incident-42", valueOrEmpty(semantics.ConsolidationGroupKey))
	requireEqualInt(t, 3, valueOrZero(semantics.ConsolidationMergedCount))
}

func snapshotSummary(minutesAgo int, title string, abstract string) models.EngramSummary {
	now := time.Date(2026, 2, 25, 15, 0, 0, 0, time.UTC)
	return models.EngramSummary{
		EngramID:  uuid.MustParse(nextTestUUID(minutesAgo + 100)),
		ProjectID: "project-a",
		ThreadID:  stringPtr("chat-session:abc:autosave"),
		Title:     title,
		Abstract:  abstract,
		CreatedAt: now.Add(-1 * time.Duration(minutesAgo) * time.Minute),
		Tags:      []string{"autosave_snapshot"},
		Keywords:  []string{"snapshot"},
	}
}

func nextTestUUID(seed int) string {
	return "00000000-0000-0000-0000-" + pad12(seed)
}

func pad12(value int) string {
	text := nextIndexLabel(value)
	for len(text) < 12 {
		text = "0" + text
	}
	return text
}

func nextIndexLabel(value int) string {
	if value == 0 {
		return "0"
	}
	digits := make([]byte, 0)
	remaining := value
	for remaining > 0 {
		digits = append([]byte{byte('0' + (remaining % 10))}, digits...)
		remaining /= 10
	}
	return string(digits)
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func valueOrZero(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func requireTrue(t *testing.T, value bool) {
	t.Helper()
	if !value {
		t.Fatalf("expected true")
	}
}

func requireFalse(t *testing.T, value bool) {
	t.Helper()
	if value {
		t.Fatalf("expected false")
	}
}

func requireEqualString[T ~string](t *testing.T, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %q, got %q", expected, actual)
	}
}

func requireEqualInt(t *testing.T, expected int, actual int) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %d, got %d", expected, actual)
	}
}
