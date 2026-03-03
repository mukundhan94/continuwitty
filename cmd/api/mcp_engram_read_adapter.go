package main

import (
	"context"

	"engram/internal/mcp"
	"engram/internal/models"
	"engram/internal/repository"
)

type mcpEngramReadAdapter struct {
	db           repository.Queryer
	embeddingDim int
}

func newMCPEngramQueryAdapter(
	db repository.Queryer,
	embeddingDim int,
) mcp.EngramQueryService {
	if db == nil {
		return nil
	}
	return mcpEngramReadAdapter{db: db, embeddingDim: embeddingDim}
}

func newMCPEngramRehydrateAdapter(db repository.Queryer) mcp.EngramRehydrateService {
	if db == nil {
		return nil
	}
	return mcpEngramReadAdapter{db: db}
}

func newMCPEngramFeedbackAdapter(db repository.Queryer) mcp.EngramFeedbackService {
	if db == nil {
		return nil
	}
	return mcpEngramReadAdapter{db: db}
}

func (adapter mcpEngramReadAdapter) QueryEngrams(
	ctx context.Context,
	request mcp.EngramQueryDispatchRequest,
) ([]models.EngramQueryResult, error) {
	queryLiteral, err := repository.BuildLocalQueryLiteral(
		request.Payload.Query,
		adapter.embeddingDim,
	)
	if err != nil {
		return nil, err
	}
	return repository.QueryEngrams(
		ctx,
		adapter.db,
		repository.QueryEngramsInput{
			Request:      request.Payload,
			QueryLiteral: queryLiteral,
			ActorUserID:  &request.ActorUserID,
		},
	)
}

func (adapter mcpEngramReadAdapter) RehydrateEngram(
	ctx context.Context,
	request mcp.EngramRehydrateRequest,
) (*models.RehydrationBundle, error) {
	return repository.GetRehydrationBundle(
		ctx,
		adapter.db,
		repository.RehydrationInput{
			EngramID:    request.EngramID,
			ActorUserID: &request.ActorUserID,
		},
	)
}

func (adapter mcpEngramReadAdapter) SubmitEngramFeedback(
	ctx context.Context,
	request mcp.EngramFeedbackRequest,
) (*models.EngramFeedbackRecord, error) {
	return repository.RecordEngramFeedback(
		ctx,
		adapter.db,
		repository.EngramFeedbackCreateInput{
			EngramID:       request.EngramID,
			SessionID:      request.SessionID,
			ActorUserID:    request.ActorUserID,
			FeedbackType:   request.FeedbackType,
			Note:           request.Note,
			RelevanceScore: request.RelevanceScore,
		},
	)
}
