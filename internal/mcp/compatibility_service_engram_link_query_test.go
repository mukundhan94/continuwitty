package mcp

import (
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

type engramLinkQueryFixture struct {
	actorUserID    uuid.UUID
	sourceEngramID uuid.UUID
	targetEngramID uuid.UUID
	listService    *fakeEngramLinkListService
	suggestService *fakeEngramLinkSuggestService
	traceService   *fakeEngramTracePathService
	service        Service
}

func newEngramLinkQueryFixture() engramLinkQueryFixture {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000007020")
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-000000007021")
	targetEngramID := uuid.MustParse("00000000-0000-0000-0000-000000007022")
	listService := &fakeEngramLinkListService{
		links: []models.EngramLinkRecord{
			sampleCompatibilityLinkRecord(
				uuid.MustParse("00000000-0000-0000-0000-000000007023"),
				sourceEngramID,
				targetEngramID,
			),
		},
	}
	suggestService := &fakeEngramLinkSuggestService{
		suggestions: []models.EngramLinkSuggestion{
			{
				SourceEngramID:  sourceEngramID,
				TargetEngramID:  targetEngramID,
				ProjectID:       "proj-1",
				TargetTitle:     "candidate",
				TargetAbstract:  "candidate abstract",
				TargetCreatedAt: time.Date(2026, 3, 1, 13, 0, 0, 0, time.UTC),
				RelationType:    models.EngramLinkRelationRelatedTo,
				Weight:          0.6,
				TemporalWeight:  0.5,
				Confidence:      0.7,
				Score:           0.74,
				Origin:          models.EngramLinkOriginSuggested,
				Status:          models.EngramLinkStatusSuggested,
				Reasons:         []string{"semantic_overlap=0.75"},
				EvidenceJSON:    map[string]any{},
			},
		},
	}
	traceService := &fakeEngramTracePathService{
		steps: []models.EngramLinkTraversalStep{
			{
				Depth: 1,
				Link: sampleCompatibilityLinkRecord(
					uuid.MustParse("00000000-0000-0000-0000-000000007024"),
					sourceEngramID,
					targetEngramID,
				),
			},
		},
	}
	service := newEngramLinkCompatibilityService(
		CompatibilityServiceDependencies{
			EngramLinkList:    listService,
			EngramLinkSuggest: suggestService,
			EngramTracePath:   traceService,
		},
	)
	return engramLinkQueryFixture{
		actorUserID:    actorUserID,
		sourceEngramID: sourceEngramID,
		targetEngramID: targetEngramID,
		listService:    listService,
		suggestService: suggestService,
		traceService:   traceService,
		service:        service,
	}
}
