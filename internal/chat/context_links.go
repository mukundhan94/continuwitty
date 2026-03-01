package chat

import (
	"context"
	"slices"

	"github.com/google/uuid"
)

type linkedEngramSelection struct {
	linkedIDs               []uuid.UUID
	tracePaths              []EngramTracePath
	suppressedTraceCount    int
	truncatedTracePathCount int
}

type traceRootCollectionInput struct {
	request      ChatContextRequest
	dependencies ChatContextDependencies
	rootEngramID uuid.UUID
	seedScore    float64
}

func selectLinkedEngramContext(
	ctx context.Context,
	request ChatContextRequest,
	dependencies ChatContextDependencies,
	seedEngramIDs []uuid.UUID,
	seedScores map[uuid.UUID]float64,
) (linkedEngramSelection, error) {
	if !shouldSelectLinkedContext(request, dependencies, seedEngramIDs) {
		return linkedEngramSelection{}, nil
	}
	candidateByTarget, suppressedTraceCount, err := collectLinkedTracePathCandidates(
		ctx,
		request,
		dependencies,
		seedEngramIDs,
		seedScores,
	)
	if err != nil {
		return linkedEngramSelection{}, err
	}
	if len(candidateByTarget) == 0 {
		return linkedEngramSelection{}, nil
	}
	candidates := sortLinkedTraceCandidates(candidateByTarget)
	candidatePoolLimit := linkedCandidatePoolLimit(request.MaxEngrams, len(seedEngramIDs))
	selectedIDs, selectedPaths, truncatedTracePathCount := selectLinkedTraceCandidates(candidates, candidatePoolLimit)
	return linkedEngramSelection{
		linkedIDs:               selectedIDs,
		tracePaths:              selectedPaths,
		suppressedTraceCount:    suppressedTraceCount,
		truncatedTracePathCount: truncatedTracePathCount,
	}, nil
}

func shouldSelectLinkedContext(
	request ChatContextRequest,
	dependencies ChatContextDependencies,
	seedEngramIDs []uuid.UUID,
) bool {
	if request.LinkRecallEnabled == nil || !*request.LinkRecallEnabled {
		return false
	}
	if request.MaxEngrams <= 1 || len(seedEngramIDs) == 0 {
		return false
	}
	if request.LinkRecallDepth == nil || request.LinkRecallMaxNeighbors == nil {
		return false
	}
	return dependencies.TraverseEngramLinks != nil
}

func collectLinkedTracePathCandidates(
	ctx context.Context,
	request ChatContextRequest,
	dependencies ChatContextDependencies,
	seedEngramIDs []uuid.UUID,
	seedScores map[uuid.UUID]float64,
) (map[uuid.UUID]EngramTracePath, int, error) {
	candidateByTarget := make(map[uuid.UUID]EngramTracePath)
	suppressedTraceCount := 0
	for _, rootEngramID := range seedEngramIDs {
		rootTracePaths, err := collectTracePathsForRoot(
			ctx,
			traceRootCollectionInput{
				request:      request,
				dependencies: dependencies,
				rootEngramID: rootEngramID,
				seedScore:    seedScores[rootEngramID],
			},
		)
		if err != nil {
			return nil, 0, err
		}
		suppressedTraceCount += mergeTracePathCandidates(candidateByTarget, rootTracePaths, request, seedEngramIDs)
	}
	return candidateByTarget, suppressedTraceCount, nil
}

func collectTracePathsForRoot(
	ctx context.Context,
	input traceRootCollectionInput,
) ([]EngramTracePath, error) {
	steps, err := input.dependencies.TraverseEngramLinks(
		ctx,
		input.rootEngramID,
		input.request.ActorUserID,
		*input.request.LinkRecallDepth,
		*input.request.LinkRecallMaxNeighbors,
		false,
	)
	if err != nil {
		return nil, err
	}
	return buildTracePathsForRoot(input.rootEngramID, steps, input.seedScore), nil
}

func mergeTracePathCandidates(
	candidateByTarget map[uuid.UUID]EngramTracePath,
	paths []EngramTracePath,
	request ChatContextRequest,
	seedEngramIDs []uuid.UUID,
) int {
	suppressedTraceCount := 0
	for _, path := range paths {
		if shouldSkipTracePathCandidate(path, request, seedEngramIDs) {
			if shouldSuppressNoisyTracePath(request, path) {
				suppressedTraceCount++
			}
			continue
		}
		promoteTracePathCandidate(candidateByTarget, path)
	}
	return suppressedTraceCount
}

func shouldSkipTracePathCandidate(
	path EngramTracePath,
	request ChatContextRequest,
	seedEngramIDs []uuid.UUID,
) bool {
	if containsUUID(seedEngramIDs, path.TargetEngramID) {
		return true
	}
	return shouldSuppressNoisyTracePath(request, path)
}

func promoteTracePathCandidate(candidateByTarget map[uuid.UUID]EngramTracePath, path EngramTracePath) {
	existing, exists := candidateByTarget[path.TargetEngramID]
	if exists && path.Score <= existing.Score {
		return
	}
	candidateByTarget[path.TargetEngramID] = path
}

func sortLinkedTraceCandidates(candidateByTarget map[uuid.UUID]EngramTracePath) []EngramTracePath {
	candidates := make([]EngramTracePath, 0, len(candidateByTarget))
	for _, path := range candidateByTarget {
		candidates = append(candidates, path)
	}
	slices.SortStableFunc(candidates, compareTracePathForSelection)
	return candidates
}

func selectLinkedTraceCandidates(
	candidates []EngramTracePath,
	candidatePoolLimit int,
) ([]uuid.UUID, []EngramTracePath, int) {
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
	return selectedIDs, selectedPaths, truncatedTracePathCount
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
