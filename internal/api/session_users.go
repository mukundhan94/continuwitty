package api

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

const (
	sessionUserNameMinLength = 3
	sessionUserNameMaxLength = 64
	sessionPasswordMinLength = 8
	sessionPasswordMaxLength = 128
)

var sessionUsernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

// SessionUserCreateInput captures normalized user creation input for route dependencies.
type SessionUserCreateInput struct {
	Username     string
	PasswordHash string
	Role         models.UserRole
	IsActive     bool
}

// SessionUserUpdateInput captures normalized user update input for route dependencies.
type SessionUserUpdateInput struct {
	Role         *models.UserRole
	IsActive     *bool
	PasswordHash *string
}

type sessionUserCreateRequest struct {
	Username string  `json:"username"`
	Password string  `json:"password"`
	Role     *string `json:"role"`
	IsActive *bool   `json:"is_active"`
}

type sessionUserUpdateRequest struct {
	Role     *string `json:"role"`
	IsActive *bool   `json:"is_active"`
	Password *string `json:"password"`
}

type validatedSessionUserCreateRequest struct {
	username string
	password string
	role     models.UserRole
	isActive bool
}

type validatedSessionUserUpdateRequest struct {
	role     *models.UserRole
	isActive *bool
	password *string
}

type updateUserOperationRequest struct {
	userID        uuid.UUID
	updateRequest validatedSessionUserUpdateRequest
	updateInput   SessionUserUpdateInput
}

func (dependencies sessionAuthDependencies) handleListUsers(writer http.ResponseWriter, request *http.Request) {
	if _, ok := dependencies.requireAdminAPIActor(writer, request); !ok {
		return
	}
	if dependencies.listUsers == nil {
		writeSessionUserDependenciesError(writer)
		return
	}

	limit, ok := parseOptionalIntQuery(
		writer,
		request,
		"limit",
		intQuerySpec{Default: 200, Min: 1, Max: 500},
	)
	if !ok {
		return
	}
	offset, ok := parseOptionalIntQuery(
		writer,
		request,
		"offset",
		intQuerySpec{Default: 0, Min: 0, Max: 1_000_000},
	)
	if !ok {
		return
	}

	users, err := dependencies.listUsers(request.Context(), limit, offset)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}
	writeJSON(writer, http.StatusOK, users)
}

func (dependencies sessionAuthDependencies) handleCreateUser(writer http.ResponseWriter, request *http.Request) {
	actor, ok := dependencies.requireAdminAPIActor(writer, request)
	if !ok {
		return
	}
	if dependencies.createUser == nil || dependencies.hashPassword == nil {
		writeSessionUserDependenciesError(writer)
		return
	}

	createRequest, ok := decodeCreateUserRequest(writer, request)
	if !ok {
		return
	}
	passwordHash, err := dependencies.hashPassword(createRequest.password)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}

	created, err := dependencies.createUser(
		request.Context(),
		SessionUserCreateInput{
			Username:     createRequest.username,
			PasswordHash: passwordHash,
			Role:         createRequest.role,
			IsActive:     createRequest.isActive,
		},
	)
	if err != nil {
		dependencies.logAuditEventRequest(
			request,
			sessionAuditEvent{
				eventType: "user_create_conflict",
				success:   false,
				username:  actor.Username,
				detail:    err.Error(),
				metadata: map[string]any{
					"target_username": createRequest.username,
				},
			},
		)
		if isUserCreateConflictError(err) {
			writeJSON(writer, http.StatusConflict, map[string]string{"detail": err.Error()})
			return
		}
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}

	dependencies.logAuditEventRequest(
		request,
		sessionAuditEvent{
			eventType: "user_created",
			success:   true,
			username:  actor.Username,
			metadata: map[string]any{
				"target_username": createRequest.username,
				"role":            createRequest.role,
			},
		},
	)
	writeJSON(writer, http.StatusCreated, created)
}

func (dependencies sessionAuthDependencies) handleUpdateUser(writer http.ResponseWriter, request *http.Request) {
	actor, ok := dependencies.requireAdminAPIActor(writer, request)
	if !ok {
		return
	}
	if !dependencies.validateUpdateUserDependencies(writer) {
		return
	}

	operationRequest, ok := dependencies.prepareUpdateUserOperation(writer, request)
	if !ok {
		return
	}
	dependencies.executeUpdateUserOperation(writer, request, actor, operationRequest)
}

func (dependencies sessionAuthDependencies) validateUpdateUserDependencies(writer http.ResponseWriter) bool {
	if dependencies.updateUser == nil || dependencies.hashPassword == nil {
		writeSessionUserDependenciesError(writer)
		return false
	}
	return true
}

func (dependencies sessionAuthDependencies) prepareUpdateUserOperation(
	writer http.ResponseWriter,
	request *http.Request,
) (updateUserOperationRequest, bool) {
	userID, ok := parsePathUUID(writer, request, "user_id")
	if !ok {
		return updateUserOperationRequest{}, false
	}
	updateRequest, ok := decodeUpdateUserRequest(writer, request)
	if !ok {
		return updateUserOperationRequest{}, false
	}
	updateInput, ok := dependencies.buildUpdateUserInput(writer, updateRequest)
	if !ok {
		return updateUserOperationRequest{}, false
	}
	return updateUserOperationRequest{
		userID:        userID,
		updateRequest: updateRequest,
		updateInput:   updateInput,
	}, true
}

func (dependencies sessionAuthDependencies) executeUpdateUserOperation(
	writer http.ResponseWriter,
	request *http.Request,
	actor *models.UserAuthRecord,
	operation updateUserOperationRequest,
) {
	updated, err := dependencies.updateUser(request.Context(), operation.userID, operation.updateInput)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}
	if updated == nil {
		dependencies.logAuditEventRequest(
			request,
			sessionAuditEvent{
				eventType: "user_update_missing",
				success:   false,
				username:  actor.Username,
				metadata: map[string]any{
					"target_user_id": operation.userID.String(),
				},
			},
		)
		writeJSON(writer, http.StatusNotFound, map[string]string{"detail": "User not found"})
		return
	}

	dependencies.logAuditEventRequest(
		request,
		sessionAuditEvent{
			eventType: "user_updated",
			success:   true,
			username:  actor.Username,
			metadata: map[string]any{
				"target_user_id":   operation.userID.String(),
				"role":             optionalSessionRoleValue(operation.updateRequest.role),
				"is_active":        operation.updateRequest.isActive,
				"password_updated": operation.updateRequest.password != nil,
			},
		},
	)
	writeJSON(writer, http.StatusOK, updated)
}

func (dependencies sessionAuthDependencies) buildUpdateUserInput(
	writer http.ResponseWriter,
	updateRequest validatedSessionUserUpdateRequest,
) (SessionUserUpdateInput, bool) {
	var passwordHash *string
	if updateRequest.password != nil {
		hashedPassword, err := dependencies.hashPassword(*updateRequest.password)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
			return SessionUserUpdateInput{}, false
		}
		passwordHash = &hashedPassword
	}
	return SessionUserUpdateInput{
		Role:         updateRequest.role,
		IsActive:     updateRequest.isActive,
		PasswordHash: passwordHash,
	}, true
}

func (dependencies sessionAuthDependencies) requireAuthenticatedAPIActor(
	writer http.ResponseWriter,
	request *http.Request,
) (*models.UserAuthRecord, bool) {
	if dependencies.lookupUserByID == nil {
		writeSessionUserDependenciesError(writer)
		return nil, false
	}
	actor, ok := AdminActorFromContext(request.Context())
	if !ok {
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"detail": "authentication required"})
		return nil, false
	}
	record, err := dependencies.lookupUserByID(request.Context(), actor.UserID)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return nil, false
	}
	if !isActiveUserRecord(record, nil) {
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"detail": "authentication required"})
		return nil, false
	}
	return record, true
}

func (dependencies sessionAuthDependencies) requireAdminAPIActor(
	writer http.ResponseWriter,
	request *http.Request,
) (*models.UserAuthRecord, bool) {
	record, ok := dependencies.requireAuthenticatedAPIActor(writer, request)
	if !ok {
		return nil, false
	}
	if !isAdminRole(record.Role) {
		writeJSON(writer, http.StatusForbidden, map[string]string{"detail": "insufficient role"})
		return nil, false
	}
	return record, true
}

func decodeCreateUserRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (validatedSessionUserCreateRequest, bool) {
	payload := sessionUserCreateRequest{}
	if !decodeJSONAllowEmpty(writer, request, &payload) {
		return validatedSessionUserCreateRequest{}, false
	}

	username := strings.TrimSpace(payload.Username)
	if !isValidSessionUsername(username) {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid username"})
		return validatedSessionUserCreateRequest{}, false
	}
	if !isValidSessionPassword(payload.Password) {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid password"})
		return validatedSessionUserCreateRequest{}, false
	}
	role, ok := parseSessionRoleOrDefault(payload.Role, models.UserRoleAnalyst)
	if !ok {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid role"})
		return validatedSessionUserCreateRequest{}, false
	}
	isActive := payload.IsActive == nil || *payload.IsActive
	return validatedSessionUserCreateRequest{
		username: username,
		password: payload.Password,
		role:     role,
		isActive: isActive,
	}, true
}

func decodeUpdateUserRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (validatedSessionUserUpdateRequest, bool) {
	payload := sessionUserUpdateRequest{}
	if !decodeJSONAllowEmpty(writer, request, &payload) {
		return validatedSessionUserUpdateRequest{}, false
	}
	if hasNoUpdateUserFields(payload) {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "No update fields provided"})
		return validatedSessionUserUpdateRequest{}, false
	}

	role, ok := resolveOptionalUpdateRole(payload.Role)
	if !ok {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid role"})
		return validatedSessionUserUpdateRequest{}, false
	}
	if !isValidOptionalPassword(payload.Password) {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid password"})
		return validatedSessionUserUpdateRequest{}, false
	}

	return validatedSessionUserUpdateRequest{
		role:     role,
		isActive: payload.IsActive,
		password: payload.Password,
	}, true
}

func resolveOptionalUpdateRole(role *string) (*models.UserRole, bool) {
	if role == nil {
		return nil, true
	}
	parsedRole, ok := parseSessionRoleOrDefault(role, "")
	if !ok {
		return nil, false
	}
	return &parsedRole, true
}

func hasNoUpdateUserFields(payload sessionUserUpdateRequest) bool {
	return payload.Role == nil && payload.IsActive == nil && payload.Password == nil
}

func isValidOptionalPassword(password *string) bool {
	return password == nil || isValidSessionPassword(*password)
}

func parseSessionRoleOrDefault(role *string, defaultRole models.UserRole) (models.UserRole, bool) {
	if role == nil {
		return defaultRole, true
	}
	trimmedRole := strings.TrimSpace(*role)
	if trimmedRole == "" {
		if defaultRole == "" {
			return "", false
		}
		return defaultRole, true
	}
	parsedRole, err := models.ParseUserRole(trimmedRole)
	if err != nil {
		return "", false
	}
	return parsedRole, true
}

func isValidSessionUsername(username string) bool {
	usernameLength := len(username)
	if usernameLength < sessionUserNameMinLength || usernameLength > sessionUserNameMaxLength {
		return false
	}
	return sessionUsernamePattern.MatchString(username)
}

func isValidSessionPassword(password string) bool {
	passwordLength := len(password)
	return passwordLength >= sessionPasswordMinLength && passwordLength <= sessionPasswordMaxLength
}

func writeSessionUserDependenciesError(writer http.ResponseWriter) {
	writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "session auth dependencies are not configured"})
}

func optionalSessionRoleValue(role *models.UserRole) any {
	if role == nil {
		return nil
	}
	return *role
}

func isUserCreateConflictError(err error) bool {
	return errors.Is(err, repository.ErrUsernameExists)
}
