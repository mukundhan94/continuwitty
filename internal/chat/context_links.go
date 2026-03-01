package chat

import (
	"context"
	"slices"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

type linkedEngramSelection struct {
	linkedIDs               []uuid.UUID
	tracePaths              []EngramTracePath
	suppressedTraceCount    int
	truncatedTracePathCount int
}

type traceChain struct {
	engramIDs []uuid.UUID
	linkIDs   []uuid.UUID
	depth     int
	score     float64
}

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

func selectLinkedEngramContext(
	ctx context.Context,
	request ChatContextRequest,
	dependencies ChatContextDependencies,
	seedEngramIDs []uuid.UUID,
	seedScores map[uuid.UUID]float64,
) (linkedEngramSelection, error) {
	enabled := request.LinkRecallEnabled != nil && *request.LinkRecallEnabled
	if !enabled || request.MaxEngrams <= 1 || len(seedEngramIDs) == 0 || dependencies.TraverseEngramLinks == nil {
		return linkedEngramSelection{}, nil
	}

	candidateByTarget := make(map[uuid.UUID]EngramTracePath)
	suppressedTraceCount := 0
	for _, rootEngramID := range seedEngramIDs {
		steps, err := dependencies.TraverseEngramLinks(
			ctx,
			rootEngramID,
			request.ActorUserID,
			*request.LinkRecallDepth,
			*request.LinkRecallMaxNeighbors,
			false,
		)
		if err != nil {
			return linkedEngramSelection{}, err
		}
		for _, path := range buildTracePathsForRoot(rootEngramID, steps, seedScores[rootEngramID]) {
			if containsUUID(seedEngramIDs, path.TargetEngramID) {
				continue
			}
			if shouldSuppressNoisyTracePath(request, path) {
				suppressedTraceCount++
				continue
			}
			if existing, exists := candidateByTarget[path.TargetEngramID]; !exists || path.Score > existing.Score {
				candidateByTarget[path.TargetEngramID] = path
			}
		}
	}

	if len(candidateByTarget) == 0 {
		return linkedEngramSelection{}, nil
	}

	candidates := make([]EngramTracePath, 0, len(candidateByTarget))
	for _, path := range candidateByTarget {
		candidates = append(candidates, path)
	}
	slices.SortStableFunc(candidates, compareTracePathForSelection)

	candidatePoolLimit := linkedCandidatePoolLimit(request.MaxEngrams, len(seedEngramIDs))
	truncatedTracePathCount := max(len(candidates)-candidatePoolLimit, 0)
	selectedPaths := make([]EngramTracePath, 0, candidatePoolLimit)
	selectedIDs := make([]uuid.UUID, 0, candidatePoolLimit)
	for _, path := range candidates {
		if len(selectedIDs) >= candidatePoolLimit {
			break
		}
		selectedIDs = append(selectedIDs, path.TargetEngramID)
		selectedPaths = append(selectedPaths, path)
	}
	return linkedEngramSelection{
		linkedIDs:               selectedIDs,
		tracePaths:              selectedPaths,
		suppressedTraceCount:    suppressedTraceCount,
		truncatedTracePathCount: truncatedTracePathCount,
	}, nil
}

func linkedCandidatePoolLimit(maxEngrams int, seedCount int) int {
	if maxEngrams <= 1 {
		return 0
	}
	minimumLinked := linkedContextBudget(seedCount, maxEngrams)
	expanded := max(minimumLinked*2, maxEngrams)
	return min(expanded, maxLinkRecallCandidatePool)
}

func linkedContextBudget(seedCount int, maxEngrams int) int {
	if maxEngrams <= 1 {
		return 0
	}
	baseBudget := min(seedCount, maxEngrams-1)
	return maxEngrams - baseBudget
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

	for _, linkedID := range linkedOrdered {
		if len(merged) >= maxEngrams {
			break
		}
		if containsUUID(merged, linkedID) {
			continue
		}
		merged = append(merged, linkedID)
	}
	for _, seedID := range seedOrdered[baseBudget:] {
		if len(merged) >= maxEngrams {
			break
		}
		if containsUUID(merged, seedID) {
			continue
		}
		merged = append(merged, seedID)
	}
	return truncateUUIDs(merged, maxEngrams)
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

func buildTracePathsForRoot(
	rootEngramID uuid.UUID,
	steps []models.EngramLinkTraversalStep,
	seedScore float64,
) []EngramTracePath {
	if len(steps) == 0 {
		return []EngramTracePath{}
	}
	ordered := append([]models.EngramLinkTraversalStep(nil), steps...)
	slices.SortStableFunc(ordered, compareTraversalStep)

	chains := map[uuid.UUID]traceChain{
		rootEngramID: {
			engramIDs: []uuid.UUID{rootEngramID},
			linkIDs:   []uuid.UUID{},
			depth:     0,
			score:     clampFloat(seedScore, 0.0, 1.0),
		},
	}
	for _, step := range ordered {
		sourceChain, exists := chains[step.Link.SourceEngramID]
		if !exists {
			continue
		}
		if sourceChain.depth+1 != step.Depth {
			continue
		}
		pathScore := scoreTracePath(sourceChain.score, step)
		candidate := traceChain{
			engramIDs: appendUUIDCopy(sourceChain.engramIDs, step.Link.TargetEngramID),
			linkIDs:   appendUUIDCopy(sourceChain.linkIDs, step.Link.LinkID),
			depth:     step.Depth,
			score:     pathScore,
		}
		existing, exists := chains[step.Link.TargetEngramID]
		if !exists || candidate.score > existing.score {
			chains[step.Link.TargetEngramID] = candidate
		}
	}

	paths := make([]EngramTracePath, 0, len(chains)-1)
	for engramID, chain := range chains {
		if engramID == rootEngramID || len(chain.linkIDs) == 0 {
			continue
		}
		paths = append(
			paths,
			EngramTracePath{
				RootEngramID:   rootEngramID,
				TargetEngramID: engramID,
				Depth:          chain.depth,
				LinkIDs:        append([]uuid.UUID(nil), chain.linkIDs...),
				EngramIDs:      append([]uuid.UUID(nil), chain.engramIDs...),
				Score:          chain.score,
			},
		)
	}
	slices.SortStableFunc(paths, compareTracePathForSelection)
	return paths
}

func compareTraversalStep(left models.EngramLinkTraversalStep, right models.EngramLinkTraversalStep) int {
	if left.Depth != right.Depth {
		return left.Depth - right.Depth
	}
	if left.Link.Weight != right.Link.Weight {
		if left.Link.Weight > right.Link.Weight {
			return -1
		}
		return 1
	}
	return compareUUID(left.Link.LinkID, right.Link.LinkID)
}

func compareTracePathForSelection(left EngramTracePath, right EngramTracePath) int {
	if left.Score != right.Score {
		if left.Score > right.Score {
			return -1
		}
		return 1
	}
	if left.Depth != right.Depth {
		return left.Depth - right.Depth
	}
	return compareUUID(left.TargetEngramID, right.TargetEngramID)
}

func compareUUID(left uuid.UUID, right uuid.UUID) int {
	leftText := left.String()
	rightText := right.String()
	switch {
	case leftText < rightText:
		return -1
	case leftText > rightText:
		return 1
	default:
		return 0
	}
}

func scoreTracePath(seedScore float64, step models.EngramLinkTraversalStep) float64 {
	depthFactor := 1.0 / float64(max(step.Depth, 1))
	linkScore := scoreLinkQuality(step.Link)
	score := (0.42 * seedScore) + (0.48 * linkScore) + (0.10 * depthFactor)
	return clampFloat(score, 0.0, 1.0)
}

func scoreLinkQuality(link models.EngramLinkRecord) float64 {
	now := time.Now().UTC()
	recency := scoreLinkRecencyAt(now, link)
	decayedTemporal := DecayedLinkTemporalWeight(now, link)
	score := (0.30 * link.Weight) + (0.25 * link.Confidence) + (0.30 * decayedTemporal) + (0.15 * recency)
	return clampFloat(score, 0.0, 1.0)
}

func scoreLinkRecency(link models.EngramLinkRecord) float64 {
	return scoreLinkRecencyAt(time.Now().UTC(), link)
}

func shouldSuppressNoisyTracePath(
	request ChatContextRequest,
	path EngramTracePath,
) bool {
	enabled := request.LinkNoiseSuppressionEnabled != nil && *request.LinkNoiseSuppressionEnabled
	if !enabled {
		return false
	}
	threshold := defaultLinkNoiseScoreThreshold
	if request.LinkNoiseScoreThreshold != nil {
		threshold = clampFloat(*request.LinkNoiseScoreThreshold, 0.0, 1.0)
	}
	return path.Score < threshold
}

func filterTracePathsForUsedEngrams(
	tracePaths []EngramTracePath,
	usedEngramIDs []uuid.UUID,
) []EngramTracePath {
	if len(tracePaths) == 0 || len(usedEngramIDs) == 0 {
		return []EngramTracePath{}
	}
	usedSet := make(map[uuid.UUID]struct{}, len(usedEngramIDs))
	for _, engramID := range usedEngramIDs {
		usedSet[engramID] = struct{}{}
	}
	filtered := make([]EngramTracePath, 0, len(tracePaths))
	for _, path := range tracePaths {
		if _, exists := usedSet[path.TargetEngramID]; !exists {
			continue
		}
		filtered = append(filtered, path)
	}
	slices.SortStableFunc(filtered, compareTracePathForSelection)
	return filtered
}

func collectUsedLinkIDs(tracePaths []EngramTracePath) []uuid.UUID {
	ordered := make([]uuid.UUID, 0, len(tracePaths))
	seen := make(map[uuid.UUID]struct{})
	for _, path := range tracePaths {
		for _, linkID := range path.LinkIDs {
			if _, exists := seen[linkID]; exists {
				continue
			}
			seen[linkID] = struct{}{}
			ordered = append(ordered, linkID)
		}
	}
	return ordered
}

func truncateUUIDs(ids []uuid.UUID, limit int) []uuid.UUID {
	if limit <= 0 {
		return []uuid.UUID{}
	}
	if len(ids) <= limit {
		return append([]uuid.UUID(nil), ids...)
	}
	return append([]uuid.UUID(nil), ids[:limit]...)
}

func containsUUID(ids []uuid.UUID, value uuid.UUID) bool {
	for _, item := range ids {
		if item == value {
			return true
		}
	}
	return false
}

func appendUUIDCopy(input []uuid.UUID, value uuid.UUID) []uuid.UUID {
	output := make([]uuid.UUID, 0, len(input)+1)
	output = append(output, input...)
	output = append(output, value)
	return output
}

func clamp(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func clampFloat(value float64, minValue float64, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
