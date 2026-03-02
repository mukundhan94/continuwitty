package admin

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestListMemoryAdminRequestsForwardSharedObject(t *testing.T) {
	testCases := []struct {
		name  string
		setup func(service *Service, capture *sharedListRequestCapture)
		call  func(service *Service, request MemoryAdminListRequest) (int, error)
	}{
		{
			name: "sessions",
			setup: func(service *Service, capture *sharedListRequestCapture) {
				service.deps.listAdminSessions = func(
					_ context.Context,
					_ repository.Queryer,
					input repository.AdminSessionListInput,
				) ([]models.AdminChatSessionRecord, error) {
					capture.projectID = input.ProjectID
					capture.ownerUserID = input.OwnerUserID
					capture.includeDeleted = input.IncludeDeleted
					capture.limit = input.Limit
					capture.offset = input.Offset
					return []models.AdminChatSessionRecord{}, nil
				}
			},
			call: func(service *Service, request MemoryAdminListRequest) (int, error) {
				listed, err := service.ListSessions(context.Background(), request)
				return len(listed), err
			},
		},
		{
			name: "collections",
			setup: func(service *Service, capture *sharedListRequestCapture) {
				service.deps.listCollections = func(
					_ context.Context,
					_ repository.Queryer,
					input repository.CollectionListInput,
				) ([]models.EngramCollectionRecord, error) {
					capture.projectID = input.ProjectID
					capture.ownerUserID = input.OwnerUserID
					capture.includeDeleted = input.IncludeDeleted
					capture.limit = input.Limit
					capture.offset = input.Offset
					return []models.EngramCollectionRecord{}, nil
				}
			},
			call: func(service *Service, request MemoryAdminListRequest) (int, error) {
				listed, err := service.ListCollections(context.Background(), request)
				return len(listed), err
			},
		},
	}

	for index, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			service := NewService(nil, 256, nil)
			capture := &sharedListRequestCapture{}
			testCase.setup(service, capture)

			ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000f01")
			if index == 1 {
				ownerUserID = uuid.MustParse("00000000-0000-0000-0000-000000000f02")
			}
			projectID := "engram-vault"
			request := MemoryAdminListRequest{
				ProjectID:      &projectID,
				OwnerUserID:    &ownerUserID,
				IncludeDeleted: false,
				Limit:          50,
				Offset:         10,
			}

			listedCount, err := testCase.call(service, request)
			requireNoError(t, err)
			requireEqual(t, 0, listedCount)
			requireEqual(t, projectID, *capture.projectID)
			requireEqual(t, ownerUserID, *capture.ownerUserID)
			requireEqual(t, false, capture.includeDeleted)
			requireEqual(t, 50, capture.limit)
			requireEqual(t, 10, capture.offset)
		})
	}
}

func TestListEngramsUsesRequestObject(t *testing.T) {
	service := NewService(nil, 256, nil)
	captured := repository.AdminEngramListInput{}
	service.deps.listAdminEngrams = func(_ context.Context, _ repository.Queryer, input repository.AdminEngramListInput) ([]models.AdminEngramRecord, error) {
		captured = input
		return []models.AdminEngramRecord{}, nil
	}

	projectID := "engram-vault"
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000f03")
	queryText := "incident"
	request := MemoryAdminEngramListRequest{
		MemoryAdminListRequest: MemoryAdminListRequest{
			ProjectID:      &projectID,
			IncludeDeleted: true,
			Limit:          25,
			Offset:         5,
		},
		SessionID: &sessionID,
		QueryText: &queryText,
	}

	listed, err := service.ListEngrams(context.Background(), request)
	requireNoError(t, err)
	requireEqual(t, 0, len(listed))
	requireEqual(t, projectID, *captured.ProjectID)
	requireEqual(t, sessionID, *captured.SessionID)
	requireEqual(t, queryText, *captured.QueryText)
	requireEqual(t, true, captured.IncludeDeleted)
}

func TestUpdateCollectionRejectsStaleExpectedUpdatedAt(t *testing.T) {
	service := NewService(nil, 256, nil)
	now := time.Now().UTC()
	collectionID := uuid.MustParse("00000000-0000-0000-0000-000000000f04")
	service.deps.getCollection = func(_ context.Context, _ repository.Queryer, _ uuid.UUID, _ bool) (*models.EngramCollectionRecord, error) {
		return &models.EngramCollectionRecord{
			CollectionID: collectionID,
			ProjectID:    "engram-vault",
			OwnerUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000f05"),
			Name:         "Collection",
			Description:  "desc",
			CreatedAt:    now,
			UpdatedAt:    now,
		}, nil
	}
	updateCalled := false
	service.deps.updateCollection = func(_ context.Context, _ repository.Queryer, _ repository.CollectionUpdateInput) (*models.EngramCollectionRecord, error) {
		updateCalled = true
		return nil, nil
	}

	expectedUpdatedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	name := "Updated"
	description := "updated"
	_, err := service.UpdateCollection(
		context.Background(),
		collectionID,
		CollectionUpdateRequest{
			ExpectedUpdatedAt: &expectedUpdatedAt,
			Name:              &name,
			Description:       &description,
		},
	)
	if !errors.Is(err, ErrCollectionStale) {
		t.Fatalf("expected ErrCollectionStale, got %v", err)
	}
	if updateCalled {
		t.Fatalf("expected updateCollection not to be called on stale update")
	}
}

func TestUpdateEngramRejectsStaleExpectedUpdatedAt(t *testing.T) {
	service := NewService(nil, 256, nil)
	now := time.Now().UTC()
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000f06")
	service.deps.getAdminEngram = func(_ context.Context, _ repository.Queryer, _ uuid.UUID, _ bool) (*models.AdminEngramRecord, error) {
		return &models.AdminEngramRecord{
			EngramID:                engramID,
			ProjectID:               "engram-vault",
			Title:                   "Engram",
			Abstract:                "",
			DetailedSummaryMarkdown: "",
			Tags:                    []string{},
			Keywords:                []string{},
			VisibilityScope:         models.VisibilityScopePrivate,
			Sources:                 []models.AdminEngramSourceRecord{},
			CreatedAt:               now,
			UpdatedAt:               now,
		}, nil
	}
	updateCalled := false
	service.deps.updateAdminEngram = func(_ context.Context, _ repository.Queryer, _ repository.AdminEngramUpdateInput) (*models.AdminEngramRecord, error) {
		updateCalled = true
		return nil, nil
	}

	expectedUpdatedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	title := "Updated"
	abstract := "updated"
	detailed := "updated"
	tags := []string{}
	keywords := []string{}
	visibility := models.VisibilityScopePrivate
	sources := []models.AdminEngramSourceInput{}
	_, err := service.UpdateEngram(
		context.Background(),
		engramID,
		uuid.MustParse("00000000-0000-0000-0000-000000000f07"),
		EngramUpdateRequest{
			ExpectedUpdatedAt:       &expectedUpdatedAt,
			Title:                   &title,
			Abstract:                &abstract,
			DetailedSummaryMarkdown: &detailed,
			Tags:                    &tags,
			Keywords:                &keywords,
			VisibilityScope:         &visibility,
			Sources:                 &sources,
		},
	)
	if !errors.Is(err, ErrEngramStale) {
		t.Fatalf("expected ErrEngramStale, got %v", err)
	}
	if updateCalled {
		t.Fatalf("expected updateAdminEngram not to be called on stale update")
	}
}

func TestUpdateEngramUsesRepositoryRequestObject(t *testing.T) {
	service := NewService(nil, 256, nil)
	now := time.Now().UTC()
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000f08")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000f09")

	current := &models.AdminEngramRecord{
		EngramID:                engramID,
		ProjectID:               "engram-vault",
		Title:                   "Engram",
		Abstract:                "",
		DetailedSummaryMarkdown: "",
		Tags:                    []string{},
		Keywords:                []string{},
		VisibilityScope:         models.VisibilityScopePrivate,
		Sources:                 []models.AdminEngramSourceRecord{},
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	service.deps.getAdminEngram = func(_ context.Context, _ repository.Queryer, _ uuid.UUID, _ bool) (*models.AdminEngramRecord, error) {
		return current, nil
	}
	captured := repository.AdminEngramUpdateInput{}
	service.deps.updateAdminEngram = func(_ context.Context, _ repository.Queryer, input repository.AdminEngramUpdateInput) (*models.AdminEngramRecord, error) {
		captured = input
		return current, nil
	}

	title := "Updated"
	abstract := "updated"
	detailed := "updated"
	tags := []string{"tag-1"}
	keywords := []string{"keyword-1"}
	visibility := models.VisibilityScopePrivate
	sources := []models.AdminEngramSourceInput{}
	updated, err := service.UpdateEngram(
		context.Background(),
		engramID,
		actorUserID,
		EngramUpdateRequest{
			ExpectedUpdatedAt:       &now,
			Title:                   &title,
			Abstract:                &abstract,
			DetailedSummaryMarkdown: &detailed,
			Tags:                    &tags,
			Keywords:                &keywords,
			VisibilityScope:         &visibility,
			Sources:                 &sources,
		},
	)
	requireNoError(t, err)
	requireEqual(t, current, updated)
	requireEqual(t, engramID, captured.EngramID)
	requireEqual(t, actorUserID, captured.ActorUserID)
	requireEqual(t, "Updated", derefString(captured.Title))
	requireEqual(t, "updated", derefString(captured.Abstract))
	requireEqual(t, []string{"tag-1"}, derefStringSlice(captured.Tags))
	requireEqual(t, 256, captured.EmbeddingDim)
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func derefStringSlice(value *[]string) []string {
	if value == nil {
		return nil
	}
	return *value
}

type sharedListRequestCapture struct {
	projectID      *string
	ownerUserID    *uuid.UUID
	includeDeleted bool
	limit          int
	offset         int
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func requireEqual[T any](t *testing.T, expected, actual T) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %#v, got %#v", expected, actual)
	}
}
