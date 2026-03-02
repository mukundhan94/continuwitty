package main

import (
	"context"
	"errors"
	"testing"

	"engram/internal/admin"
	internalapi "engram/internal/api"
	"engram/internal/models"
	"engram/internal/projects"

	"github.com/google/uuid"
)

type fakeProjectResolutionService struct {
	resolveFn func(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		projectID string,
	) (projects.Resolution, error)
}

func (service fakeProjectResolutionService) ResolveProjectIDForWrite(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	projectID string,
) (projects.Resolution, error) {
	if service.resolveFn == nil {
		return projects.Resolution{}, nil
	}
	return service.resolveFn(ctx, actorUserID, actorRole, projectID)
}

func TestProjectResolutionAdaptersReturnServiceResolution(t *testing.T) {
	testCases := []struct {
		name               string
		actorUserID        uuid.UUID
		role               models.UserRole
		returnedResolution projects.Resolution
		resolveWithAdapter func(service projectResolutionService, actorUserID uuid.UUID, actorRole models.UserRole, projectID string) (adapterResult, error)
	}{
		{
			name:               "session dependency",
			actorUserID:        uuid.MustParse("00000000-0000-0000-0000-000000000811"),
			role:               models.UserRoleAnalyst,
			returnedResolution: projects.Resolution{ProjectID: "project-default", UsedDefaultProject: true},
			resolveWithAdapter: resolveWithSessionDependency,
		},
		{
			name:               "admin resolver",
			actorUserID:        uuid.MustParse("00000000-0000-0000-0000-000000000812"),
			role:               models.UserRoleAdmin,
			returnedResolution: projects.Resolution{ProjectID: "project-admin", UsedDefaultProject: false},
			resolveWithAdapter: resolveWithAdminProjectResolver,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assertAdapterResolution(
				t,
				testCase.actorUserID,
				testCase.role,
				"project-input",
				testCase.returnedResolution,
				testCase.resolveWithAdapter,
			)
		})
	}
}

func TestMapSessionProjectResolutionErrorMapsKnownErrors(t *testing.T) {
	testCases := []struct {
		name     string
		source   error
		expected error
	}{
		{
			name:     "missing default",
			source:   projects.ErrProjectIDRequiredWhenNoDefaultProject,
			expected: internalapi.ErrProjectIDRequiredWhenNoDefaultProject,
		},
		{
			name:     "default inaccessible",
			source:   projects.ErrDefaultProjectNotAccessible,
			expected: internalapi.ErrDefaultProjectNotAccessible,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mapped := mapSessionProjectResolutionError(testCase.source)
			if !errors.Is(mapped, testCase.expected) {
				t.Fatalf("expected mapped error %v, got %v", testCase.expected, mapped)
			}
		})
	}
}

func TestMapSessionProjectResolutionErrorReturnsUnknownError(t *testing.T) {
	unexpected := errors.New("boom")
	mapped := mapSessionProjectResolutionError(unexpected)
	if !errors.Is(mapped, unexpected) {
		t.Fatalf("expected unknown error passthrough, got %v", mapped)
	}
}

func TestMapAdminProjectResolutionErrorMapsKnownErrors(t *testing.T) {
	testCases := []struct {
		name   string
		source error
	}{
		{name: "blank", source: projects.ErrProjectIDMustNotBeBlank},
		{name: "missing default", source: projects.ErrProjectIDRequiredWhenNoDefaultProject},
		{name: "default inaccessible", source: projects.ErrDefaultProjectNotAccessible},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mapped := mapAdminProjectResolutionError(testCase.source)
			if !errors.Is(mapped, admin.ErrProjectIDRequired) {
				t.Fatalf("expected mapped error %v, got %v", admin.ErrProjectIDRequired, mapped)
			}
		})
	}
}

func TestMapAdminProjectResolutionErrorReturnsUnknownError(t *testing.T) {
	unexpected := errors.New("boom")
	mapped := mapAdminProjectResolutionError(unexpected)
	if !errors.Is(mapped, unexpected) {
		t.Fatalf("expected unknown error passthrough, got %v", mapped)
	}
}

func requireEqualString[T ~string](t *testing.T, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %q, got %q", expected, actual)
	}
}

func requireEqualUUID(t *testing.T, expected, actual uuid.UUID) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

type adapterResult struct {
	projectID          string
	usedDefaultProject bool
}

func resolveWithSessionDependency(
	service projectResolutionService,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	projectID string,
) (adapterResult, error) {
	dependency := resolveProjectIDForWriteDependency(service)
	resolution, err := dependency(context.Background(), actorUserID, actorRole, projectID)
	return adapterResult{
		projectID:          resolution.ProjectID,
		usedDefaultProject: resolution.UsedDefaultProject,
	}, err
}

func resolveWithAdminProjectResolver(
	service projectResolutionService,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	projectID string,
) (adapterResult, error) {
	resolver := newAdminProjectResolver(service)
	resolution, err := resolver.ResolveProjectIDForWrite(
		context.Background(),
		admin.ResolveProjectWriteInput{
			ActorUserID: actorUserID,
			ActorRole:   string(actorRole),
			ProjectID:   projectID,
		},
	)
	return adapterResult{
		projectID:          resolution.ProjectID,
		usedDefaultProject: resolution.UsedDefaultProject,
	}, err
}

func newCapturingResolutionService(
	t *testing.T,
	expectedActorUserID uuid.UUID,
	returnedResolution projects.Resolution,
) (fakeProjectResolutionService, *models.UserRole, *string) {
	t.Helper()
	capturedRole := models.UserRole("")
	capturedProjectID := ""
	service := fakeProjectResolutionService{
		resolveFn: func(
			_ context.Context,
			receivedActorUserID uuid.UUID,
			receivedActorRole models.UserRole,
			receivedProjectID string,
		) (projects.Resolution, error) {
			requireEqualUUID(t, expectedActorUserID, receivedActorUserID)
			capturedRole = receivedActorRole
			capturedProjectID = receivedProjectID
			return returnedResolution, nil
		},
	}
	return service, &capturedRole, &capturedProjectID
}

func assertAdapterResolution(
	t *testing.T,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	requestedProjectID string,
	returnedResolution projects.Resolution,
	resolveWithAdapter func(service projectResolutionService, actorUserID uuid.UUID, actorRole models.UserRole, projectID string) (adapterResult, error),
) {
	t.Helper()
	service, capturedRole, capturedProjectID := newCapturingResolutionService(t, actorUserID, returnedResolution)
	result, err := resolveWithAdapter(service, actorUserID, actorRole, requestedProjectID)
	requireNoError(t, err)
	requireEqualString(t, actorRole, *capturedRole)
	requireEqualString(t, requestedProjectID, *capturedProjectID)
	requireEqualString(t, returnedResolution.ProjectID, result.projectID)
	if result.usedDefaultProject != returnedResolution.UsedDefaultProject {
		t.Fatalf(
			"expected used_default_project=%v, got %v",
			returnedResolution.UsedDefaultProject,
			result.usedDefaultProject,
		)
	}
}
