package api

import (
	"errors"
	"net/http"

	"engram/internal/repository"
)

func writeEngramLinkServiceError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrEngramLinkExists):
		writeJSON(writer, http.StatusConflict, map[string]string{"detail": repository.ErrEngramLinkExists.Error()})
	default:
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
	}
}
