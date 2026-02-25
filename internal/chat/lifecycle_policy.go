package chat

import (
	"strings"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

const (
	consolidationGroupKeyTagPrefix    = "consolidation_group_key:"
	consolidationMergedCountTagPrefix = "consolidation_merged_count:"
)

// SessionLifecyclePolicy captures chat autosave and retention configuration.
type SessionLifecyclePolicy struct {
	AutosaveEnabled       bool
	AutosaveStrategy      models.ChatAutosaveStrategy
	AutosaveIntervalMins  int
	AutosaveMinMessages   int
	RetentionDays         int
	RetentionMaxSnapshots int
}

// TimelineEventSemantics captures timeline classification details.
type TimelineEventSemantics struct {
	EventType                string
	ConsolidationGroupKey    *string
	ConsolidationMergedCount *int
}

// NormalizeAutosavePolicy keeps legacy autosave-enabled and strategy flags coherent.
func NormalizeAutosavePolicy(
	autosaveEnabled bool,
	autosaveStrategy models.ChatAutosaveStrategy,
) (bool, models.ChatAutosaveStrategy) {
	if !autosaveEnabled {
		return false, models.ChatAutosaveStrategyOff
	}
	if autosaveStrategy == models.ChatAutosaveStrategyOff {
		return true, models.ChatAutosaveStrategyInterval
	}
	return true, autosaveStrategy
}

// ClassifyTimelineEvent classifies timeline tags into event semantics.
func ClassifyTimelineEvent(tags []string) TimelineEventSemantics {
	normalizedTags := normalizeTags(tags)
	tagSet := toLowerTagSet(normalizedTags)
	if _, hasAutosaveSnapshot := tagSet["autosave_snapshot"]; hasAutosaveSnapshot {
		return TimelineEventSemantics{EventType: "autosave_snapshot"}
	}
	if _, hasConsolidated := tagSet["consolidated"]; hasConsolidated {
		groupKey := extractTagValue(normalizedTags, consolidationGroupKeyTagPrefix)
		mergedCount := parsePositiveInt(extractTagValue(normalizedTags, consolidationMergedCountTagPrefix))
		eventType := resolveConsolidationEventType(groupKey, mergedCount)
		return TimelineEventSemantics{
			EventType:                eventType,
			ConsolidationGroupKey:    groupKey,
			ConsolidationMergedCount: mergedCount,
		}
	}
	return TimelineEventSemantics{EventType: "manual_snapshot"}
}

// ClassifyTimelineEventType returns only the classified event type.
func ClassifyTimelineEventType(tags []string) string {
	return ClassifyTimelineEvent(tags).EventType
}

// IsLowValueSnapshotAbstract detects generic or short snapshot abstracts.
func IsLowValueSnapshotAbstract(value string) bool {
	normalized := normalizeSpaces(value)
	if normalized == "" {
		return true
	}
	if len(normalized) < 36 {
		return true
	}
	return len(strings.Fields(normalized)) < 7
}

// ShouldTakeIntervalSnapshot determines whether interval-based autosave should trigger.
func ShouldTakeIntervalSnapshot(
	now time.Time,
	latestSnapshotCreatedAt *time.Time,
	intervalMinutes int,
) bool {
	if latestSnapshotCreatedAt == nil {
		return true
	}
	threshold := now.Add(-1 * time.Duration(maxInt(intervalMinutes, 1)) * time.Minute)
	return !latestSnapshotCreatedAt.After(threshold)
}

// ShouldTakeMessageCountSnapshot determines whether message-count autosave should trigger.
func ShouldTakeMessageCountSnapshot(assistantMessageCount int, minMessages int) bool {
	if assistantMessageCount <= 0 {
		return false
	}
	window := maxInt(minMessages, 1)
	return assistantMessageCount%window == 0
}

// DuplicateSnapshotExists detects whether an abstract already exists in snapshots.
func DuplicateSnapshotExists(abstract string, existingSnapshots []models.EngramSummary) bool {
	normalized := strings.ToLower(normalizeSpaces(abstract))
	if normalized == "" {
		return false
	}
	for _, snapshot := range existingSnapshots {
		candidate := strings.ToLower(normalizeSpaces(snapshot.Abstract))
		if candidate == normalized {
			return true
		}
	}
	return false
}

// SelectRetentionPruneIDs determines which snapshot ids should be pruned by retention rules.
func SelectRetentionPruneIDs(
	snapshots []models.EngramSummary,
	retentionDays int,
	retentionMaxSnapshots int,
	now time.Time,
) []uuid.UUID {
	if len(snapshots) == 0 {
		return []uuid.UUID{}
	}
	retentionCutoff, retentionMaxCount := retentionThresholds(resolveReferenceTime(now), retentionDays, retentionMaxSnapshots)
	keepIDs := collectKeepSnapshotIDs(snapshots, retentionCutoff, retentionMaxCount)
	return collectPruneSnapshotIDs(snapshots, keepIDs)
}

func normalizeTags(tags []string) []string {
	normalized := make([]string, 0, len(tags))
	for _, item := range tags {
		candidate := strings.TrimSpace(item)
		if candidate == "" {
			continue
		}
		normalized = append(normalized, candidate)
	}
	return normalized
}

func toLowerTagSet(tags []string) map[string]struct{} {
	set := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		set[strings.ToLower(tag)] = struct{}{}
	}
	return set
}

func extractTagValue(tags []string, prefix string) *string {
	lowerPrefix := strings.ToLower(prefix)
	for _, tag := range tags {
		lowerTag := strings.ToLower(tag)
		if !strings.HasPrefix(lowerTag, lowerPrefix) {
			continue
		}
		value := strings.TrimSpace(tag[len(prefix):])
		if value == "" {
			continue
		}
		return stringPtr(value)
	}
	return nil
}

func parsePositiveInt(value *string) *int {
	if value == nil {
		return nil
	}
	parsed, err := parseInt(*value)
	if err != nil || parsed <= 0 {
		return nil
	}
	return intPtr(parsed)
}

func resolveConsolidationEventType(groupKey *string, mergedCount *int) string {
	if mergedCount != nil && *mergedCount > 1 {
		return "consolidation_merge"
	}
	if groupKey != nil {
		return "consolidation_group"
	}
	return "consolidation"
}

func resolveReferenceTime(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now
}

func normalizeSpaces(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func parseInt(value string) (int, error) {
	parsed := 0
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return 0, errInvalidNumber
		}
		parsed = parsed*10 + int(ch-'0')
	}
	return parsed, nil
}

func retentionThresholds(reference time.Time, retentionDays int, retentionMaxSnapshots int) (time.Time, int) {
	return reference.AddDate(0, 0, -maxInt(retentionDays, 1)), maxInt(retentionMaxSnapshots, 1)
}

func collectKeepSnapshotIDs(
	snapshots []models.EngramSummary,
	retentionCutoff time.Time,
	retentionMaxCount int,
) map[uuid.UUID]struct{} {
	keepIDs := make(map[uuid.UUID]struct{}, len(snapshots))
	for index, snapshot := range snapshots {
		if !shouldKeepSnapshot(index, snapshot, retentionCutoff, retentionMaxCount) {
			continue
		}
		keepIDs[snapshot.EngramID] = struct{}{}
	}
	return keepIDs
}

func shouldKeepSnapshot(
	index int,
	snapshot models.EngramSummary,
	retentionCutoff time.Time,
	retentionMaxCount int,
) bool {
	withinCount := index < retentionMaxCount
	withinTime := !snapshot.CreatedAt.Before(retentionCutoff)
	return withinCount && withinTime
}

func collectPruneSnapshotIDs(
	snapshots []models.EngramSummary,
	keepIDs map[uuid.UUID]struct{},
) []uuid.UUID {
	pruneIDs := make([]uuid.UUID, 0)
	for _, snapshot := range snapshots {
		if _, shouldKeep := keepIDs[snapshot.EngramID]; shouldKeep {
			continue
		}
		pruneIDs = append(pruneIDs, snapshot.EngramID)
	}
	return pruneIDs
}

func maxInt(first int, second int) int {
	if first > second {
		return first
	}
	return second
}

func stringPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

var errInvalidNumber = &invalidNumberError{}

type invalidNumberError struct{}

func (invalidNumberError) Error() string {
	return "invalid number"
}
