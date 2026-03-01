package main

import (
	"context"

	"engram/internal/graph"
	"engram/internal/mcp"
	"engram/internal/models"
	"engram/internal/repository"
)

type mcpEngramLinkAdapter struct {
	db                repository.Queryer
	suggestionService *graph.LinkSuggestionService
}

func newMCPEngramLinkGetAdapter(db repository.Queryer) mcp.EngramLinkGetService {
	if db == nil {
		return nil
	}
	return mcpEngramLinkAdapter{db: db}
}

func newMCPEngramLinkCreateAdapter(db repository.Queryer) mcp.EngramLinkCreateService {
	if db == nil {
		return nil
	}
	return mcpEngramLinkAdapter{db: db}
}

func newMCPEngramLinkListAdapter(db repository.Queryer) mcp.EngramLinkListService {
	if db == nil {
		return nil
	}
	return mcpEngramLinkAdapter{db: db}
}

func newMCPEngramLinkUpdateAdapter(db repository.Queryer) mcp.EngramLinkUpdateService {
	if db == nil {
		return nil
	}
	return mcpEngramLinkAdapter{db: db}
}

func newMCPEngramLinkArchiveAdapter(db repository.Queryer) mcp.EngramLinkArchiveService {
	if db == nil {
		return nil
	}
	return mcpEngramLinkAdapter{db: db}
}

func newMCPEngramLinkSuggestAdapter(
	db repository.Queryer,
	embeddingDim int,
) mcp.EngramLinkSuggestService {
	suggestionService := graph.NewLinkSuggestionService(db, embeddingDim)
	if suggestionService == nil {
		return nil
	}
	return mcpEngramLinkAdapter{
		db:                db,
		suggestionService: suggestionService,
	}
}

func newMCPEngramTracePathAdapter(db repository.Queryer) mcp.EngramTracePathService {
	if db == nil {
		return nil
	}
	return mcpEngramLinkAdapter{db: db}
}

func (adapter mcpEngramLinkAdapter) GetEngramLink(
	ctx context.Context,
	request mcp.EngramLinkGetRequest,
) (*models.EngramLinkRecord, error) {
	return repository.GetEngramLink(
		ctx,
		adapter.db,
		repository.EngramLinkGetInput{
			LinkID:          request.LinkID,
			ActorUserID:     request.ActorUserID,
			IncludeArchived: request.IncludeArchived,
		},
	)
}

func (adapter mcpEngramLinkAdapter) CreateEngramLink(
	ctx context.Context,
	request mcp.EngramLinkCreateRequest,
) (*models.EngramLinkRecord, error) {
	return repository.CreateEngramLink(
		ctx,
		adapter.db,
		repository.EngramLinkCreateInput{
			SourceEngramID:   request.SourceEngramID,
			TargetEngramID:   request.TargetEngramID,
			RelationType:     request.RelationType,
			Weight:           request.Weight,
			TemporalWeight:   request.TemporalWeight,
			Confidence:       request.Confidence,
			Origin:           request.Origin,
			Status:           request.Status,
			EvidenceJSON:     request.EvidenceJSON,
			CreatedByUserID:  request.ActorUserID,
			ActorUserID:      request.ActorUserID,
			LastReinforcedAt: request.LastReinforcedAt,
		},
	)
}

func (adapter mcpEngramLinkAdapter) ListEngramLinks(
	ctx context.Context,
	request mcp.EngramLinkListRequest,
) ([]models.EngramLinkRecord, error) {
	return repository.ListEngramLinks(
		ctx,
		adapter.db,
		repository.EngramLinkListInput{
			SourceEngramID:  request.SourceEngramID,
			ActorUserID:     request.ActorUserID,
			RelationType:    request.RelationType,
			IncludeArchived: request.IncludeArchived,
			Limit:           request.Limit,
			Offset:          request.Offset,
		},
	)
}

func (adapter mcpEngramLinkAdapter) UpdateEngramLink(
	ctx context.Context,
	request mcp.EngramLinkUpdateRequest,
) (*models.EngramLinkRecord, error) {
	return repository.UpdateEngramLink(
		ctx,
		adapter.db,
		repository.EngramLinkUpdateInput{
			LinkID:           request.LinkID,
			ActorUserID:      request.ActorUserID,
			Weight:           request.Weight,
			TemporalWeight:   request.TemporalWeight,
			Confidence:       request.Confidence,
			Status:           request.Status,
			EvidenceJSON:     request.EvidenceJSON,
			LastReinforcedAt: request.LastReinforcedAt,
		},
	)
}

func (adapter mcpEngramLinkAdapter) ArchiveEngramLink(
	ctx context.Context,
	request mcp.EngramLinkArchiveRequest,
) (*models.EngramLinkRecord, error) {
	return repository.ArchiveEngramLink(
		ctx,
		adapter.db,
		repository.EngramLinkArchiveInput{
			LinkID:      request.LinkID,
			ActorUserID: request.ActorUserID,
		},
	)
}

func (adapter mcpEngramLinkAdapter) SuggestEngramLinks(
	ctx context.Context,
	request mcp.EngramLinkSuggestRequest,
) ([]models.EngramLinkSuggestion, error) {
	if adapter.suggestionService == nil {
		return []models.EngramLinkSuggestion{}, nil
	}
	return adapter.suggestionService.SuggestLinks(
		ctx,
		graph.LinkSuggestionInput{
			SourceEngramID:  request.SourceEngramID,
			ActorUserID:     request.ActorUserID,
			Limit:           request.Limit,
			MaxCandidates:   request.MaxCandidates,
			MinimumScore:    request.MinimumScore,
			IncludeArchived: request.IncludeArchived,
		},
	)
}

func (adapter mcpEngramLinkAdapter) TraceEngramPath(
	ctx context.Context,
	request mcp.EngramTracePathRequest,
) ([]models.EngramLinkTraversalStep, error) {
	return repository.TraverseEngramLinks(
		ctx,
		adapter.db,
		repository.EngramLinkTraverseInput{
			RootEngramID:    request.RootEngramID,
			ActorUserID:     request.ActorUserID,
			MaxDepth:        request.MaxDepth,
			MaxNeighbors:    request.MaxNeighbors,
			IncludeArchived: request.IncludeArchived,
		},
	)
}
