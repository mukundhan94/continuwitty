package mcp

import (
	"context"

	"engram/internal/models"
)

type fakeEngramTracePathService struct {
	steps []models.EngramLinkTraversalStep
	err   error
	call  EngramTracePathRequest
}

func (service *fakeEngramTracePathService) TraceEngramPath(
	_ context.Context,
	request EngramTracePathRequest,
) ([]models.EngramLinkTraversalStep, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	return append([]models.EngramLinkTraversalStep(nil), service.steps...), nil
}
