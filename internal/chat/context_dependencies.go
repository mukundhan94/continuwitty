package chat

import (
	"context"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func listPinnedEngramSummariesDependency(
	db repository.Queryer,
) func(ctx context.Context, sessionID uuid.UUID, actorUserID uuid.UUID) ([]models.EngramSummary, error) {
	return func(ctx context.Context, sessionID uuid.UUID, actorUserID uuid.UUID) ([]models.EngramSummary, error) {
		return repository.ListPinnedEngramSummaries(
			ctx,
			db,
			repository.ChatPinnedListInput{SessionID: sessionID, ActorUserID: actorUserID},
		)
	}
}

func listPinnedDocumentsDependency(
	db repository.Queryer,
) func(ctx context.Context, sessionID uuid.UUID, actorUserID uuid.UUID) ([]models.PinnedDocumentRecord, error) {
	return func(ctx context.Context, sessionID uuid.UUID, actorUserID uuid.UUID) ([]models.PinnedDocumentRecord, error) {
		return repository.ListPinnedDocuments(
			ctx,
			db,
			repository.ChatPinnedListInput{SessionID: sessionID, ActorUserID: actorUserID},
		)
	}
}

func queryEngramsDependency(
	db repository.Queryer,
) func(ctx context.Context, actorUserID uuid.UUID, request models.EngramQueryRequest, embeddingDim int) ([]models.EngramQueryResult, error) {
	return func(
		ctx context.Context,
		actorUserID uuid.UUID,
		request models.EngramQueryRequest,
		embeddingDim int,
	) ([]models.EngramQueryResult, error) {
		queryLiteral, err := repository.BuildLocalQueryLiteral(request.Query, embeddingDim)
		if err != nil {
			return nil, err
		}
		return repository.QueryEngrams(
			ctx,
			db,
			repository.QueryEngramsInput{
				Request:      request,
				QueryLiteral: queryLiteral,
				ActorUserID:  &actorUserID,
			},
		)
	}
}

func getRehydrationBundleDependency(
	db repository.Queryer,
) func(ctx context.Context, engramID uuid.UUID, actorUserID uuid.UUID) (*models.RehydrationBundle, error) {
	return func(ctx context.Context, engramID uuid.UUID, actorUserID uuid.UUID) (*models.RehydrationBundle, error) {
		return repository.GetRehydrationBundle(
			ctx,
			db,
			repository.RehydrationInput{EngramID: engramID, ActorUserID: &actorUserID},
		)
	}
}

func traverseEngramLinksDependency(
	db repository.Queryer,
) func(
	ctx context.Context,
	rootEngramID uuid.UUID,
	actorUserID uuid.UUID,
	maxDepth int,
	maxNeighbors int,
	includeArchived bool,
) ([]models.EngramLinkTraversalStep, error) {
	return func(
		ctx context.Context,
		rootEngramID uuid.UUID,
		actorUserID uuid.UUID,
		maxDepth int,
		maxNeighbors int,
		includeArchived bool,
	) ([]models.EngramLinkTraversalStep, error) {
		return repository.TraverseEngramLinks(
			ctx,
			db,
			repository.EngramLinkTraverseInput{
				RootEngramID:    rootEngramID,
				ActorUserID:     actorUserID,
				MaxDepth:        maxDepth,
				MaxNeighbors:    maxNeighbors,
				IncludeArchived: includeArchived,
			},
		)
	}
}

func queryDocumentChunksDependency(
	db repository.Queryer,
) func(
	ctx context.Context,
	actorUserID uuid.UUID,
	request models.DocumentChunkQueryRequest,
	embeddingDim int,
) ([]models.DocumentChunkQueryResult, error) {
	return func(
		ctx context.Context,
		actorUserID uuid.UUID,
		request models.DocumentChunkQueryRequest,
		embeddingDim int,
	) ([]models.DocumentChunkQueryResult, error) {
		return repository.QueryDocumentChunks(
			ctx,
			db,
			repository.DocumentChunkQueryInput{
				ActorUserID:  actorUserID,
				Request:      request,
				EmbeddingDim: embeddingDim,
			},
		)
	}
}
