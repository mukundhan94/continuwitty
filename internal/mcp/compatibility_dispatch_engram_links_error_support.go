package mcp

import (
	"errors"

	"engram/internal/repository"
)

func mapEngramLinkDispatchError(err error) *toolDispatchError {
	switch {
	case errors.Is(err, repository.ErrEngramLinkExists):
		return invalidParamsWithStatus(409, repository.ErrEngramLinkExists.Error())
	default:
		return internalToolDispatchError()
	}
}
