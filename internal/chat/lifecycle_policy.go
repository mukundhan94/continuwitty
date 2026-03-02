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

type SnapshotIntervalMinutes int
type SnapshotMessageCount int
type SnapshotMessageWindow int
type SnapshotRetentionDays int
type SnapshotRetentionMaxSnapshots int

type timelineTag string
type timelineTagValue string
type timelineTagList []timelineTag
type timelineEventType string
type consolidationMergedCount int
type snapshotText string
type numberText string
type snapshotOrdinal int
type snapshotRetentionLimit int

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
	normalizedTags := normalizeTags(toTimelineTagList(tags))
	tagSet := toLowerTagSet(normalizedTags)
	if _, hasAutosaveSnapshot := tagSet[timelineTag("autosave_snapshot")]; hasAutosaveSnapshot {
		return TimelineEventSemantics{EventType: "autosave_snapshot"}
	}
	if _, hasConsolidated := tagSet[timelineTag("consolidated")]; hasConsolidated {
		groupKey := extractTagValue(normalizedTags, timelineTag(consolidationGroupKeyTagPrefix))
		mergedCount := parsePositiveInt(
			extractTagValue(normalizedTags, timelineTag(consolidationMergedCountTagPrefix)),
		)
		eventType := resolveConsolidationEventType(groupKey, mergedCount)
		return TimelineEventSemantics{
			EventType:                string(eventType),
			ConsolidationGroupKey:    timelineTagValuePtrToString(groupKey),
			ConsolidationMergedCount: mergedCountPtrToInt(mergedCount),
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
	return isLowValueSnapshotAbstract(snapshotText(value))
}

func isLowValueSnapshotAbstract(value snapshotText) bool {
	normalized := normalizeSnapshotText(value)
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
	intervalMinutes SnapshotIntervalMinutes,
) bool {
	if latestSnapshotCreatedAt == nil {
		return true
	}
	threshold := now.Add(-1 * time.Duration(maxInt(int(intervalMinutes), 1)) * time.Minute)
	return !latestSnapshotCreatedAt.After(threshold)
}

// ShouldTakeMessageCountSnapshot determines whether message-count autosave should trigger.
func ShouldTakeMessageCountSnapshot(
	assistantMessageCount SnapshotMessageCount,
	minMessages SnapshotMessageWindow,
) bool {
	if assistantMessageCount <= 0 {
		return false
	}
	window := maxInt(int(minMessages), 1)
	return int(assistantMessageCount)%window == 0
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
	retentionDays SnapshotRetentionDays,
	retentionMaxSnapshots SnapshotRetentionMaxSnapshots,
	now time.Time,
) []uuid.UUID {
	if len(snapshots) == 0 {
		return []uuid.UUID{}
	}
	retentionCutoff, retentionMaxCount := retentionThresholds(
		resolveReferenceTime(now),
		retentionDays,
		retentionMaxSnapshots,
	)
	keepIDs := collectKeepSnapshotIDs(snapshots, retentionCutoff, retentionMaxCount)
	return collectPruneSnapshotIDs(snapshots, keepIDs)
}

func toTimelineTagList(tags []string) timelineTagList {
	converted := make(timelineTagList, 0, len(tags))
	for _, tag := range tags {
		converted = append(converted, timelineTag(tag))
	}
	return converted
}

func normalizeTags(tags timelineTagList) timelineTagList {
	normalized := make(timelineTagList, 0, len(tags))
	for _, item := range tags {
		candidate := strings.TrimSpace(string(item))
		if candidate == "" {
			continue
		}
		normalized = append(normalized, timelineTag(candidate))
	}
	return normalized
}

func toLowerTagSet(tags timelineTagList) map[timelineTag]struct{} {
	set := make(map[timelineTag]struct{}, len(tags))
	for _, tag := range tags {
		set[timelineTag(strings.ToLower(string(tag)))] = struct{}{}
	}
	return set
}

func extractTagValue(tags timelineTagList, prefix timelineTag) *timelineTagValue {
	lowerPrefix := strings.ToLower(string(prefix))
	for _, tag := range tags {
		rawTag := string(tag)
		lowerTag := strings.ToLower(rawTag)
		if !strings.HasPrefix(lowerTag, lowerPrefix) {
			continue
		}
		value := strings.TrimSpace(rawTag[len(prefix):])
		if value == "" {
			continue
		}
		return timelineTagValuePtr(timelineTagValue(value))
	}
	return nil
}

func parsePositiveInt(value *timelineTagValue) *consolidationMergedCount {
	if value == nil {
		return nil
	}
	parsed, err := parseInt(numberText(*value))
	if err != nil || parsed <= 0 {
		return nil
	}
	return consolidationMergedCountPtr(consolidationMergedCount(parsed))
}

func resolveConsolidationEventType(
	groupKey *timelineTagValue,
	mergedCount *consolidationMergedCount,
) timelineEventType {
	if mergedCount != nil && *mergedCount > 1 {
		return timelineEventType("consolidation_merge")
	}
	if groupKey != nil {
		return timelineEventType("consolidation_group")
	}
	return timelineEventType("consolidation")
}

func resolveReferenceTime(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now
}

func normalizeSpaces(value string) string {
	return normalizeSnapshotText(snapshotText(value))
}

func normalizeSnapshotText(value snapshotText) string {
	return strings.Join(strings.Fields(strings.TrimSpace(string(value))), " ")
}

func parseInt(value numberText) (int, error) {
	parsed := 0
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return 0, errInvalidNumber
		}
		parsed = parsed*10 + int(ch-'0')
	}
	return parsed, nil
}

func retentionThresholds(
	reference time.Time,
	retentionDays SnapshotRetentionDays,
	retentionMaxSnapshots SnapshotRetentionMaxSnapshots,
) (time.Time, snapshotRetentionLimit) {
	return reference.AddDate(0, 0, -maxInt(int(retentionDays), 1)),
		snapshotRetentionLimit(maxInt(int(retentionMaxSnapshots), 1))
}

func collectKeepSnapshotIDs(
	snapshots []models.EngramSummary,
	retentionCutoff time.Time,
	retentionMaxCount snapshotRetentionLimit,
) map[uuid.UUID]struct{} {
	keepIDs := make(map[uuid.UUID]struct{}, len(snapshots))
	for index, snapshot := range snapshots {
		if !shouldKeepSnapshot(snapshotOrdinal(index), snapshot, retentionCutoff, retentionMaxCount) {
			continue
		}
		keepIDs[snapshot.EngramID] = struct{}{}
	}
	return keepIDs
}

func shouldKeepSnapshot(
	index snapshotOrdinal,
	snapshot models.EngramSummary,
	retentionCutoff time.Time,
	retentionMaxCount snapshotRetentionLimit,
) bool {
	withinCount := int(index) < int(retentionMaxCount)
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

func timelineTagValuePtr(value timelineTagValue) *timelineTagValue {
	return &value
}

func consolidationMergedCountPtr(value consolidationMergedCount) *consolidationMergedCount {
	return &value
}

func timelineTagValuePtrToString(value *timelineTagValue) *string {
	if value == nil {
		return nil
	}
	text := string(*value)
	return &text
}

func mergedCountPtrToInt(value *consolidationMergedCount) *int {
	if value == nil {
		return nil
	}
	count := int(*value)
	return &count
}

var errInvalidNumber = &invalidNumberError{}

type invalidNumberError struct{}

func (invalidNumberError) Error() string {
	return "invalid number"
}
