package chat

import (
	"slices"

	"github.com/google/uuid"
)

func buildContradictionWarnings(paths []EngramTracePath) []ChatContradictionWarning {
	warnings := make([]ChatContradictionWarning, 0)
	for _, path := range paths {
		if !path.HasContradiction || len(path.ContradictingLinkIDs) == 0 {
			continue
		}
		warnings = append(
			warnings,
			ChatContradictionWarning{
				RootEngramID:         path.RootEngramID,
				TargetEngramID:       path.TargetEngramID,
				ContradictingLinkIDs: append([]uuid.UUID(nil), path.ContradictingLinkIDs...),
				Severity:             contradictionSeverity(path.Depth),
				Message:              contradictionWarningMessage(path.Depth),
			},
		)
	}
	slices.SortStableFunc(warnings, compareContradictionWarnings)
	return warnings
}

func contradictionSeverity(depth int) string {
	switch {
	case depth <= 1:
		return "high"
	case depth == 2:
		return "medium"
	default:
		return "low"
	}
}

func contradictionWarningMessage(depth int) string {
	switch contradictionSeverity(depth) {
	case "high":
		return "Direct contradiction detected in recalled memory context."
	case "medium":
		return "Contradiction risk detected in linked memory context."
	default:
		return "Possible contradiction detected in extended memory trace."
	}
}

func compareContradictionWarnings(left ChatContradictionWarning, right ChatContradictionWarning) int {
	if left.Severity != right.Severity {
		return contradictionSeverityRank(left.Severity) - contradictionSeverityRank(right.Severity)
	}
	if left.RootEngramID != right.RootEngramID {
		return compareUUID(left.RootEngramID, right.RootEngramID)
	}
	return compareUUID(left.TargetEngramID, right.TargetEngramID)
}

func contradictionSeverityRank(value string) int {
	switch value {
	case "high":
		return 0
	case "medium":
		return 1
	default:
		return 2
	}
}
