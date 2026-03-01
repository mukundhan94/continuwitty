package chat

import (
	"slices"

	"engram/internal/models"

	"github.com/google/uuid"
)

type engramContextCandidate struct {
	engramID        uuid.UUID
	score           float64
	semanticScore   float64
	traceScore      float64
	traceDepth      int
	isSeedCandidate bool
}

type engramCandidateScoreInput struct {
	semanticScore float64
	traceScore    float64
	isSeed        bool
}

func rankEngramContextCandidates(
	seedEngramIDs []uuid.UUID,
	linkedEngramIDs []uuid.UUID,
	seedScores map[uuid.UUID]float64,
	tracePaths []EngramTracePath,
	maxEngrams int,
) []uuid.UUID {
	candidates := buildEngramContextCandidates(seedEngramIDs, linkedEngramIDs, seedScores, tracePaths)
	slices.SortStableFunc(candidates, compareEngramContextCandidate)
	if maxEngrams > 1 {
		candidates = ensureLinkedCandidateCoverage(candidates, linkedEngramIDs, maxEngrams)
	}
	ordered := make([]uuid.UUID, 0, len(candidates))
	for _, candidate := range candidates {
		ordered = append(ordered, candidate.engramID)
	}
	return ordered
}

func buildEngramContextCandidates(
	seedEngramIDs []uuid.UUID,
	linkedEngramIDs []uuid.UUID,
	seedScores map[uuid.UUID]float64,
	tracePaths []EngramTracePath,
) []engramContextCandidate {
	seedSet := make(map[uuid.UUID]struct{}, len(seedEngramIDs))
	for _, engramID := range seedEngramIDs {
		seedSet[engramID] = struct{}{}
	}
	traceScoreByTarget, traceDepthByTarget := buildTracePathCandidateStats(tracePaths)

	candidateOrder := dedupeUUIDsPreserveOrder(append(append([]uuid.UUID{}, seedEngramIDs...), linkedEngramIDs...))
	candidates := make([]engramContextCandidate, 0, len(candidateOrder))
	for _, engramID := range candidateOrder {
		candidates = append(candidates, buildEngramContextCandidate(engramID, seedSet, seedScores, traceScoreByTarget, traceDepthByTarget))
	}
	return candidates
}

func buildTracePathCandidateStats(
	tracePaths []EngramTracePath,
) (map[uuid.UUID]float64, map[uuid.UUID]int) {
	traceScoreByTarget := make(map[uuid.UUID]float64, len(tracePaths))
	traceDepthByTarget := make(map[uuid.UUID]int, len(tracePaths))
	for _, path := range tracePaths {
		updateTracePathCandidateStat(traceScoreByTarget, traceDepthByTarget, path)
	}
	return traceScoreByTarget, traceDepthByTarget
}

func updateTracePathCandidateStat(
	traceScoreByTarget map[uuid.UUID]float64,
	traceDepthByTarget map[uuid.UUID]int,
	path EngramTracePath,
) {
	existingScore, scoreExists := traceScoreByTarget[path.TargetEngramID]
	if !scoreExists {
		traceScoreByTarget[path.TargetEngramID] = clampFloat(path.Score, 0.0, 1.0)
		traceDepthByTarget[path.TargetEngramID] = path.Depth
		return
	}
	existingDepth := traceDepthByTarget[path.TargetEngramID]
	if path.Score < existingScore {
		return
	}
	if path.Score == existingScore && path.Depth >= existingDepth {
		return
	}
	traceScoreByTarget[path.TargetEngramID] = clampFloat(path.Score, 0.0, 1.0)
	traceDepthByTarget[path.TargetEngramID] = path.Depth
}

func buildEngramContextCandidate(
	engramID uuid.UUID,
	seedSet map[uuid.UUID]struct{},
	seedScores map[uuid.UUID]float64,
	traceScoreByTarget map[uuid.UUID]float64,
	traceDepthByTarget map[uuid.UUID]int,
) engramContextCandidate {
	semanticScore := clampFloat(seedScores[engramID], 0.0, 1.0)
	traceScore := clampFloat(traceScoreByTarget[engramID], 0.0, 1.0)
	traceDepth := resolveTraceDepth(traceDepthByTarget[engramID])
	_, isSeed := seedSet[engramID]
	return engramContextCandidate{
		engramID: engramID,
		score: fusedCandidateScore(
			engramCandidateScoreInput{
				semanticScore: semanticScore,
				traceScore:    traceScore,
				isSeed:        isSeed,
			},
		),
		semanticScore:   semanticScore,
		traceScore:      traceScore,
		traceDepth:      traceDepth,
		isSeedCandidate: isSeed,
	}
}

func resolveTraceDepth(depth int) int {
	if depth > 0 {
		return depth
	}
	return maxLinkRecallDepth + 1
}

func fusedCandidateScore(input engramCandidateScoreInput) float64 {
	score := (0.58 * input.semanticScore) + (0.42 * input.traceScore)
	if input.isSeed {
		score += 0.04
	}
	if input.traceScore > 0 {
		score += 0.03
	}
	if input.isSeed && input.traceScore > 0 {
		score += 0.05
	}
	if !input.isSeed && input.traceScore > 0 {
		score += 0.04
	}
	return clampFloat(score, 0.0, 1.0)
}

func compareEngramContextCandidate(left engramContextCandidate, right engramContextCandidate) int {
	if comparison, decided := compareDescendingFloat(left.score, right.score); decided {
		return comparison
	}
	if comparison, decided := compareDescendingFloat(left.traceScore, right.traceScore); decided {
		return comparison
	}
	if comparison, decided := compareDescendingFloat(left.semanticScore, right.semanticScore); decided {
		return comparison
	}
	if left.traceDepth != right.traceDepth {
		return left.traceDepth - right.traceDepth
	}
	if left.isSeedCandidate != right.isSeedCandidate {
		if left.isSeedCandidate {
			return -1
		}
		return 1
	}
	return compareUUID(left.engramID, right.engramID)
}

func compareDescendingFloat(left float64, right float64) (int, bool) {
	if left == right {
		return 0, false
	}
	if left > right {
		return -1, true
	}
	return 1, true
}

func ensureLinkedCandidateCoverage(
	candidates []engramContextCandidate,
	linkedEngramIDs []uuid.UUID,
	maxEngrams int,
) []engramContextCandidate {
	if len(candidates) == 0 {
		return candidates
	}
	if len(linkedEngramIDs) == 0 {
		return candidates
	}
	if maxEngrams <= 1 {
		return candidates
	}
	linkedSet := make(map[uuid.UUID]struct{}, len(linkedEngramIDs))
	for _, engramID := range linkedEngramIDs {
		linkedSet[engramID] = struct{}{}
	}
	selectionLimit := min(maxEngrams, len(candidates))
	if hasLinkedCandidateInSelectionWindow(candidates, linkedSet, selectionLimit) {
		return candidates
	}
	linkedCandidate, linkedCandidateIndex, found := findLinkedCandidateOutsideWindow(candidates, linkedSet, selectionLimit)
	if !found {
		return candidates
	}
	insertAt := 1
	if selectionLimit == 1 {
		insertAt = 0
	}
	return insertCandidateAt(candidates, linkedCandidate, linkedCandidateIndex, insertAt)
}

func hasLinkedCandidateInSelectionWindow(
	candidates []engramContextCandidate,
	linkedSet map[uuid.UUID]struct{},
	selectionLimit int,
) bool {
	for index := range selectionLimit {
		if _, exists := linkedSet[candidates[index].engramID]; exists {
			return true
		}
	}
	return false
}

func findLinkedCandidateOutsideWindow(
	candidates []engramContextCandidate,
	linkedSet map[uuid.UUID]struct{},
	startIndex int,
) (engramContextCandidate, int, bool) {
	for index := startIndex; index < len(candidates); index++ {
		if _, exists := linkedSet[candidates[index].engramID]; exists {
			return candidates[index], index, true
		}
	}
	return engramContextCandidate{}, 0, false
}

func insertCandidateAt(
	candidates []engramContextCandidate,
	candidate engramContextCandidate,
	candidateIndex int,
	insertAt int,
) []engramContextCandidate {
	updated := make([]engramContextCandidate, 0, len(candidates))
	updated = append(updated, candidates[:insertAt]...)
	updated = append(updated, candidate)
	updated = append(updated, candidates[insertAt:candidateIndex]...)
	updated = append(updated, candidates[candidateIndex+1:]...)
	return updated
}

func mergeContextEngramIDsWithLinked(
	seedEngramIDs []uuid.UUID,
	linkedEngramIDs []uuid.UUID,
	maxEngrams int,
) []uuid.UUID {
	if maxEngrams <= 0 {
		return []uuid.UUID{}
	}
	seedOrdered := dedupeUUIDsPreserveOrder(seedEngramIDs)
	linkedOrdered := dedupeUUIDsPreserveOrder(linkedEngramIDs)
	if len(linkedOrdered) == 0 {
		return truncateUUIDs(seedOrdered, maxEngrams)
	}
	if len(seedOrdered) == 0 {
		return truncateUUIDs(linkedOrdered, maxEngrams)
	}

	baseBudget := min(len(seedOrdered), maxEngrams-1)
	merged := make([]uuid.UUID, 0, maxEngrams)
	merged = append(merged, seedOrdered[:baseBudget]...)
	merged = appendUniqueUUIDsUntilLimit(merged, linkedOrdered, maxEngrams)
	merged = appendUniqueUUIDsUntilLimit(merged, seedOrdered[baseBudget:], maxEngrams)
	return truncateUUIDs(merged, maxEngrams)
}

func appendUniqueUUIDsUntilLimit(
	target []uuid.UUID,
	candidates []uuid.UUID,
	maxEngrams int,
) []uuid.UUID {
	for _, candidateID := range candidates {
		if len(target) >= maxEngrams {
			return target
		}
		if containsUUID(target, candidateID) {
			continue
		}
		target = append(target, candidateID)
	}
	return target
}

func buildSeedRelevanceScores(
	pinned []models.EngramSummary,
	retrieved []models.EngramQueryResult,
) map[uuid.UUID]float64 {
	scores := make(map[uuid.UUID]float64, len(pinned)+len(retrieved))
	for _, item := range pinned {
		scores[item.EngramID] = max(scores[item.EngramID], 1.0)
	}
	for _, item := range retrieved {
		scores[item.EngramID] = max(scores[item.EngramID], relevanceFromDistance(item.Distance))
	}
	return scores
}

func relevanceFromDistance(distance float64) float64 {
	if distance < 0 {
		distance = 0
	}
	return clampFloat(1.0/(1.0+distance), 0.0, 1.0)
}
