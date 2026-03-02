package mcp

import (
	"context"

	"engram/internal/models"
)

type fakeEngramLinkSuggestService struct {
	suggestions []models.EngramLinkSuggestion
	err         error
	call        EngramLinkSuggestRequest
}

func (service *fakeEngramLinkSuggestService) SuggestEngramLinks(
	_ context.Context,
	request EngramLinkSuggestRequest,
) ([]models.EngramLinkSuggestion, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	return append([]models.EngramLinkSuggestion(nil), service.suggestions...), nil
}
