package main

import (
	"context"

	"engram/internal/mcp"
)

func (adapter mcpEngramAdminAdapter) DeleteEngram(
	ctx context.Context,
	request mcp.EngramDeleteRequest,
) (*mcp.EngramDeleteResponse, error) {
	return runEngramMutation(
		ctx,
		adapter,
		engramMutationRequest{
			ActorUserID: request.ActorUserID,
			ActorRole:   request.ActorRole,
			EngramID:    request.EngramID,
			Reason:      request.Reason,
			Kind:        engramMutationDelete,
		},
		func(mutation engramMutationOutcome) *mcp.EngramDeleteResponse {
			return &mcp.EngramDeleteResponse{
				EngramID: mutation.EngramID,
				Deleted:  mutation.Applied,
			}
		},
	)
}
