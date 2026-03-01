package chat

import (
	"context"
	"slices"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

type linkedEngramSelection struct {
	linkedIDs  []uuid.UUID
	tracePaths []EngramTracePath
}

type traceChain struct {
	engramIDs []uuid.UUID
	linkIDs   []uuid.UUID
	depth     int
	score     float64
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

	linkedBudget := linkedContextBudget(len(seedEngramIDs), request.MaxEngrams)
	selectedPaths := make([]EngramTracePath, 0, linkedBudget)
	selectedIDs := make([]uuid.UUID, 0, linkedBudget)
	for _, path := range candidates {
		if len(selectedIDs) >= linkedBudget {
			break
		}
		selectedIDs = append(selectedIDs, path.TargetEngramID)
		selectedPaths = append(selectedPaths, path)
	}
	return linkedEngramSelection{
		linkedIDs:  selectedIDs,
		tracePaths: selectedPaths,
	}, nil
}

func linkedContextBudget(seedCount int, maxEngrams int) int {
	if maxEngrams <= 1 {
		return 0
	}
	baseBudget := min(seedCount, maxEngrams-1)
	return maxEngrams - baseBudget
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
