package graph

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

const (
	defaultLinkHygieneLimit             = 500
	defaultLinkHygieneStaleAfterDays    = 120
	defaultLinkHygieneLowValueThreshold = 0.25
)

// LinkHygieneInput captures recommendation controls for one source engram.
type LinkHygieneInput struct {
	SourceEngramID    uuid.UUID
	ActorUserID       uuid.UUID
	IncludeArchived   bool
	Limit             int
	StaleAfterDays    int
	LowValueThreshold float64
}

// LinkHygieneService detects duplicate/conflict/stale graph patterns.
type LinkHygieneService struct {
	listEngramLinks func(
		ctx context.Context,
		input repository.EngramLinkListInput,
	) ([]models.EngramLinkRecord, error)
	nowUTC func() time.Time
}

// NewLinkHygieneService builds repository-backed link hygiene recommendations.
func NewLinkHygieneService(db repository.Queryer) *LinkHygieneService {
	if db == nil {
		return nil
	}
	return &LinkHygieneService{
		listEngramLinks: func(
			ctx context.Context,
			input repository.EngramLinkListInput,
		) ([]models.EngramLinkRecord, error) {
			return repository.ListEngramLinks(ctx, db, input)
		},
		nowUTC: func() time.Time { return time.Now().UTC() },
	}
}

// Recommend returns deterministic graph hygiene recommendations for one source engram.
func (service *LinkHygieneService) Recommend(
	ctx context.Context,
	input LinkHygieneInput,
) ([]models.EngramLinkHygieneRecommendation, error) {
	if err := validateLinkHygieneInput(input); err != nil {
		return nil, err
	}
	if service == nil {
		return []models.EngramLinkHygieneRecommendation{}, nil
	}
	limit := normalizeLinkHygieneLimit(input.Limit)
	staleAfterDays := normalizeStaleAfterDays(input.StaleAfterDays)
	lowValueThreshold := normalizeLowValueThreshold(input.LowValueThreshold)

	links, err := service.listEngramLinks(
		ctx,
		repository.EngramLinkListInput{
			SourceEngramID:  input.SourceEngramID,
			ActorUserID:     input.ActorUserID,
			IncludeArchived: input.IncludeArchived,
			Limit:           limit,
			Offset:          0,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return []models.EngramLinkHygieneRecommendation{}, nil
	}
	recommendations := make([]models.EngramLinkHygieneRecommendation, 0)
	grouped := groupLinksByTarget(links)
	for _, targetLinks := range grouped {
		recommendations = append(
			recommendations,
			buildDuplicateTargetRecommendations(input.SourceEngramID, targetLinks)...,
		)
		if conflictRecommendation := buildConflictRecommendation(input.SourceEngramID, targetLinks); conflictRecommendation != nil {
			recommendations = append(recommendations, *conflictRecommendation)
		}
	}
	for _, link := range links {
		if staleRecommendation := buildStaleLowValueRecommendation(
			input.SourceEngramID,
			link,
			service.nowUTC(),
			staleAfterDays,
			lowValueThreshold,
		); staleRecommendation != nil {
			recommendations = append(recommendations, *staleRecommendation)
		}
	}
	sort.SliceStable(recommendations, func(left, right int) bool {
		if severityRank(recommendations[left].Severity) != severityRank(recommendations[right].Severity) {
			return severityRank(recommendations[left].Severity) > severityRank(recommendations[right].Severity)
		}
		if recommendations[left].Score != recommendations[right].Score {
			return recommendations[left].Score < recommendations[right].Score
		}
		if recommendations[left].TargetEngramID != recommendations[right].TargetEngramID {
			return recommendations[left].TargetEngramID.String() < recommendations[right].TargetEngramID.String()
		}
		return recommendations[left].Category < recommendations[right].Category
	})
	return recommendations, nil
}

func validateLinkHygieneInput(input LinkHygieneInput) error {
	if input.SourceEngramID == uuid.Nil {
		return errors.New("source_engram_id is required")
	}
	if input.ActorUserID == uuid.Nil {
		return errors.New("actor_user_id is required")
	}
	return nil
}

func normalizeLinkHygieneLimit(value int) int {
	if value <= 0 {
		return defaultLinkHygieneLimit
	}
	if value > 1000 {
		return 1000
	}
	return value
}

func normalizeStaleAfterDays(value int) int {
	if value <= 0 {
		return defaultLinkHygieneStaleAfterDays
	}
	if value < 7 {
		return 7
	}
	if value > 3650 {
		return 3650
	}
	return value
}

func normalizeLowValueThreshold(value float64) float64 {
	if value <= 0 {
		return defaultLinkHygieneLowValueThreshold
	}
	if value > 1 {
		return 1
	}
	return value
}

func groupLinksByTarget(links []models.EngramLinkRecord) map[uuid.UUID][]models.EngramLinkRecord {
	grouped := make(map[uuid.UUID][]models.EngramLinkRecord, len(links))
	for _, link := range links {
		grouped[link.TargetEngramID] = append(grouped[link.TargetEngramID], link)
	}
	return grouped
}

func linkStrength(link models.EngramLinkRecord) float64 {
	return clampHygieneScore((link.Weight + link.Confidence + link.TemporalWeight) / 3.0)
}

func buildDuplicateTargetRecommendations(
	sourceEngramID uuid.UUID,
	targetLinks []models.EngramLinkRecord,
) []models.EngramLinkHygieneRecommendation {
	if len(targetLinks) <= 1 {
		return []models.EngramLinkHygieneRecommendation{}
	}
	ordered := append([]models.EngramLinkRecord(nil), targetLinks...)
	sort.SliceStable(ordered, func(left, right int) bool {
		return linkStrength(ordered[left]) > linkStrength(ordered[right])
	})
	recommendations := make([]models.EngramLinkHygieneRecommendation, 0, len(ordered)-1)
	primary := ordered[0]
	for _, weaker := range ordered[1:] {
		recommendations = append(recommendations, models.EngramLinkHygieneRecommendation{
			Category:        models.EngramLinkHygieneCategoryDuplicateTarget,
			Severity:        "medium",
			SourceEngramID:  sourceEngramID,
			TargetEngramID:  weaker.TargetEngramID,
			LinkIDs:         []uuid.UUID{weaker.LinkID, primary.LinkID},
			Detail:          "Multiple links point to the same target; weaker duplicates increase graph noise.",
			SuggestedAction: "archive_weaker_duplicate",
			Score:           linkStrength(weaker),
		})
	}
	return recommendations
}

func buildConflictRecommendation(
	sourceEngramID uuid.UUID,
	targetLinks []models.EngramLinkRecord,
) *models.EngramLinkHygieneRecommendation {
	if len(targetLinks) <= 1 {
		return nil
	}
	hasContradicts := false
	hasSupportiveRelation := false
	linkIDs := make([]uuid.UUID, 0, len(targetLinks))
	for _, link := range targetLinks {
		linkIDs = append(linkIDs, link.LinkID)
		if link.RelationType == models.EngramLinkRelationContradicts {
			hasContradicts = true
			continue
		}
		if link.RelationType == models.EngramLinkRelationSupports ||
			link.RelationType == models.EngramLinkRelationDependsOn ||
			link.RelationType == models.EngramLinkRelationDerivedFrom {
			hasSupportiveRelation = true
		}
	}
	if !hasContradicts || !hasSupportiveRelation {
		return nil
	}
	lowestStrength := 1.0
	for _, link := range targetLinks {
		lowestStrength = min(lowestStrength, linkStrength(link))
	}
	return &models.EngramLinkHygieneRecommendation{
		Category:        models.EngramLinkHygieneCategoryConflictRelation,
		Severity:        "high",
		SourceEngramID:  sourceEngramID,
		TargetEngramID:  targetLinks[0].TargetEngramID,
		LinkIDs:         linkIDs,
		Detail:          "Conflicting relation types exist for the same source-target pair.",
		SuggestedAction: "review_relation_conflict",
		Score:           clampHygieneScore(lowestStrength),
	}
}

func buildStaleLowValueRecommendation(
	sourceEngramID uuid.UUID,
	link models.EngramLinkRecord,
	now time.Time,
	staleAfterDays int,
	lowValueThreshold float64,
) *models.EngramLinkHygieneRecommendation {
	if link.Status == models.EngramLinkStatusArchived || link.Status == models.EngramLinkStatusRejected {
		return nil
	}
	reference := link.CreatedAt
	if link.LastReinforcedAt != nil && !link.LastReinforcedAt.IsZero() {
		reference = *link.LastReinforcedAt
	}
	if reference.IsZero() {
		return nil
	}
	ageDays := now.Sub(reference).Hours() / 24.0
	score := linkStrength(link)
	if ageDays < float64(staleAfterDays) || score >= lowValueThreshold {
		return nil
	}
	return &models.EngramLinkHygieneRecommendation{
		Category:        models.EngramLinkHygieneCategoryStaleLowValue,
		Severity:        "low",
		SourceEngramID:  sourceEngramID,
		TargetEngramID:  link.TargetEngramID,
		LinkIDs:         []uuid.UUID{link.LinkID},
		Detail:          "Link has low confidence/weight and has not been reinforced recently.",
		SuggestedAction: "archive_stale_low_value",
		Score:           score,
	}
}

func severityRank(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func clampHygieneScore(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
