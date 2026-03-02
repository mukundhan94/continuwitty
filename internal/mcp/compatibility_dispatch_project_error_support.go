package mcp

import (
	"errors"

	"engram/internal/projects"

	"github.com/jackc/pgx/v5/pgconn"
)

func mapProjectServiceError(err error) *toolDispatchError {
	switch {
	case errors.Is(err, projects.ErrProjectIDMustNotBeBlank):
		return invalidParamError("project_id")
	case errors.Is(err, projects.ErrProjectNotFound):
		return invalidParamsWithStatus(404, "Project not found")
	case errors.Is(err, projects.ErrUserNotFound):
		return invalidParamsWithStatus(404, "User not found")
	case errors.Is(err, projects.ErrProjectWriteForbidden),
		errors.Is(err, projects.ErrProjectMemberManagementForbidden),
		errors.Is(err, projects.ErrProjectAuditForbidden),
		errors.Is(err, projects.ErrEngramShareForbidden):
		return invalidParamsWithStatus(403, err.Error())
	case errors.Is(err, projects.ErrProjectMemberNotFound),
		errors.Is(err, projects.ErrEngramNotFound):
		return invalidParamsWithStatus(404, err.Error())
	case errors.Is(err, projects.ErrProjectOwnerMembershipImmutable),
		errors.Is(err, projects.ErrProjectOwnerRoleNotAssignable):
		return invalidParamsWithStatus(422, err.Error())
	default:
		return nil
	}
}

func mapProjectCreateError(err error) *toolDispatchError {
	if mapped := mapProjectCreateDomainError(err); mapped != nil {
		return mapped
	}
	if mapped := mapProjectCreateDatabaseError(err); mapped != nil {
		return mapped
	}
	return internalToolDispatchError()
}

func mapProjectCreateDomainError(err error) *toolDispatchError {
	switch {
	case errors.Is(err, projects.ErrProjectIDMustNotBeBlank):
		return invalidParamError("project_id")
	case errors.Is(err, projects.ErrProjectWriteForbidden):
		return invalidParamsWithStatus(403, err.Error())
	case errors.Is(err, projects.ErrProjectNotFound):
		return invalidParamsWithStatus(404, "Project not found")
	case errors.Is(err, projects.ErrUserNotFound):
		return invalidParamsWithStatus(404, "User not found")
	case errors.Is(err, projects.ErrProjectIDRequiredWhenNoDefaultProject),
		errors.Is(err, projects.ErrDefaultProjectNotAccessible):
		return invalidParamsWithStatus(422, err.Error())
	default:
		if mapped := mapProjectServiceError(err); mapped != nil {
			return mapped
		}
		return nil
	}
}

func mapProjectCreateDatabaseError(err error) *toolDispatchError {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return nil
	}
	switch pgErr.Code {
	case "42P01":
		return &toolDispatchError{
			code:    -32603,
			message: "Project collaboration schema is not initialized",
			data: map[string]any{
				"detail": "run database schema initialization/migrations and restart the API",
			},
		}
	case "23503":
		return invalidParamsWithStatus(422, "owner_user_id does not reference an existing user")
	case "23514":
		return invalidParamsWithStatus(422, "project payload violates database constraints")
	default:
		return nil
	}
}
