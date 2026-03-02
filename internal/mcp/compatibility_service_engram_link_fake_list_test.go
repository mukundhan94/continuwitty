package mcp

import (
	"context"

	"engram/internal/models"
)

type fakeEngramLinkListService struct {
	links []models.EngramLinkRecord
	err   error
	call  EngramLinkListRequest
}

func (service *fakeEngramLinkListService) ListEngramLinks(
	_ context.Context,
	request EngramLinkListRequest,
) ([]models.EngramLinkRecord, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	return append([]models.EngramLinkRecord(nil), service.links...), nil
}
