package chat

import (
	"slices"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

type traceChain struct {
	engramIDs            []uuid.UUID
	linkIDs              []uuid.UUID
	contradictingLinkIDs []uuid.UUID
	hasContradiction     bool
	depth                int
	score                float64
}

func buildTracePathsForRoot(
	rootEngramID uuid.UUID,
	steps []models.EngramLinkTraversalStep,
	seedScore float64,
) []EngramTracePath {
	if len(steps) == 0 {
		return []EngramTracePath{}
	}
	chains := initializeTraceChains(rootEngramID, seedScore)
	for _, step := range sortTraversalSteps(steps) {
		promoteTraceChain(chains, step)
	}
	paths := buildTracePathsFromChains(rootEngramID, chains)
	slices.SortStableFunc(paths, compareTracePathForSelection)
	return paths
}

func initializeTraceChains(rootEngramID uuid.UUID, seedScore float64) map[uuid.UUID]traceChain {
	return map[uuid.UUID]traceChain{
		rootEngramID: {
			engramIDs:            []uuid.UUID{rootEngramID},
			linkIDs:              []uuid.UUID{},
			contradictingLinkIDs: []uuid.UUID{},
			hasContradiction:     false,
			depth:                0,
			score:                clampFloat(seedScore, 0.0, 1.0),
		},
	}
}

func sortTraversalSteps(steps []models.EngramLinkTraversalStep) []models.EngramLinkTraversalStep {
	ordered := append([]models.EngramLinkTraversalStep(nil), steps...)
	slices.SortStableFunc(ordered, compareTraversalStep)
	return ordered
}

func promoteTraceChain(chains map[uuid.UUID]traceChain, step models.EngramLinkTraversalStep) {
	sourceChain, found := sourceTraceChainForStep(chains, step)
	if !found {
		return
	}
	candidate := buildTraceChainCandidate(sourceChain, step)
	existing, exists := chains[step.Link.TargetEngramID]
	if exists && candidate.score <= existing.score {
		return
	}
	chains[step.Link.TargetEngramID] = candidate
}

func sourceTraceChainForStep(
	chains map[uuid.UUID]traceChain,
	step models.EngramLinkTraversalStep,
) (traceChain, bool) {
	sourceChain, exists := chains[step.Link.SourceEngramID]
	if !exists {
		return traceChain{}, false
	}
	if sourceChain.depth+1 != step.Depth {
		return traceChain{}, false
	}
	return sourceChain, true
}

func buildTraceChainCandidate(
	sourceChain traceChain,
	step models.EngramLinkTraversalStep,
) traceChain {
	contradictingLinkIDs := append([]uuid.UUID(nil), sourceChain.contradictingLinkIDs...)
	hasContradiction := sourceChain.hasContradiction
	if step.Link.RelationType == models.EngramLinkRelationContradicts {
		hasContradiction = true
		contradictingLinkIDs = appendUUIDCopy(contradictingLinkIDs, step.Link.LinkID)
	}
	return traceChain{
		engramIDs:            appendUUIDCopy(sourceChain.engramIDs, step.Link.TargetEngramID),
		linkIDs:              appendUUIDCopy(sourceChain.linkIDs, step.Link.LinkID),
		contradictingLinkIDs: contradictingLinkIDs,
		hasContradiction:     hasContradiction,
		depth:                step.Depth,
		score:                scoreTracePath(sourceChain.score, step),
	}
}

func buildTracePathsFromChains(
	rootEngramID uuid.UUID,
	chains map[uuid.UUID]traceChain,
) []EngramTracePath {
	paths := make([]EngramTracePath, 0, len(chains)-1)
	for engramID, chain := range chains {
		if shouldSkipTraceChain(rootEngramID, engramID, chain) {
			continue
		}
		paths = append(paths, tracePathFromChain(rootEngramID, engramID, chain))
	}
	return paths
}

func shouldSkipTraceChain(rootEngramID uuid.UUID, engramID uuid.UUID, chain traceChain) bool {
	if engramID == rootEngramID {
		return true
	}
	return len(chain.linkIDs) == 0
}

func tracePathFromChain(rootEngramID uuid.UUID, engramID uuid.UUID, chain traceChain) EngramTracePath {
	return EngramTracePath{
		RootEngramID:         rootEngramID,
		TargetEngramID:       engramID,
		Depth:                chain.depth,
		LinkIDs:              append([]uuid.UUID(nil), chain.linkIDs...),
		EngramIDs:            append([]uuid.UUID(nil), chain.engramIDs...),
		HasContradiction:     chain.hasContradiction,
		ContradictingLinkIDs: append([]uuid.UUID(nil), chain.contradictingLinkIDs...),
		Score:                chain.score,
	}
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
