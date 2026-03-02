package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const engramLinkColumns = `
	link_id,
	project_id,
	source_engram_id,
	target_engram_id,
	relation_type,
	weight,
	temporal_weight,
	confidence,
	origin,
	status,
	evidence_json,
	created_by_user_id,
	last_reinforced_at,
	created_at,
	updated_at
`

const engramLinkColumnsWithAlias = `
	link.link_id,
	link.project_id,
	link.source_engram_id,
	link.target_engram_id,
	link.relation_type,
	link.weight,
	link.temporal_weight,
	link.confidence,
	link.origin,
	link.status,
	link.evidence_json,
	link.created_by_user_id,
	link.last_reinforced_at,
	link.created_at,
	link.updated_at
`

var (
	// ErrEngramLinkExists indicates link duplication for active/suggested relation pairs.
	ErrEngramLinkExists = errors.New("engram link already exists")

	newEngramLinkUUID = uuid.New
	nowEngramLinkUTC  = func() time.Time { return time.Now().UTC() }
)

// EngramLinkCreateInput captures create-link requirements.
type EngramLinkCreateInput struct {
	SourceEngramID   uuid.UUID
	TargetEngramID   uuid.UUID
	RelationType     models.EngramLinkRelationType
	Weight           float64
	TemporalWeight   float64
	Confidence       float64
	Origin           models.EngramLinkOrigin
	Status           models.EngramLinkStatus
	EvidenceJSON     map[string]any
	CreatedByUserID  uuid.UUID
	ActorUserID      uuid.UUID
	LastReinforcedAt *time.Time
}

// EngramLinkListInput captures list-link filters.
type EngramLinkListInput struct {
	SourceEngramID  uuid.UUID
	ActorUserID     uuid.UUID
	RelationType    *models.EngramLinkRelationType
	IncludeArchived bool
	Limit           int
	Offset          int
}

// EngramLinkGetInput captures link-lookup requirements.
type EngramLinkGetInput struct {
	LinkID          uuid.UUID
	ActorUserID     uuid.UUID
	IncludeArchived bool
}

// EngramLinkUpdateInput captures optional mutable link fields.
type EngramLinkUpdateInput struct {
	LinkID           uuid.UUID
	ActorUserID      uuid.UUID
	Weight           *float64
	TemporalWeight   *float64
	Confidence       *float64
	Status           *models.EngramLinkStatus
	EvidenceJSON     *map[string]any
	LastReinforcedAt *time.Time
}

// EngramLinkArchiveInput captures archive-link requirements.
type EngramLinkArchiveInput struct {
	LinkID      uuid.UUID
	ActorUserID uuid.UUID
}

// EngramLinkTraverseInput captures traversal controls.
type EngramLinkTraverseInput struct {
	RootEngramID    uuid.UUID
	ActorUserID     uuid.UUID
	MaxDepth        int
	MaxNeighbors    int
	IncludeArchived bool
}

type traversalQueryInput struct {
	SourceEngramIDs []uuid.UUID
	ActorUserID     uuid.UUID
	IncludeArchived bool
	Limit           int
}

type visibleEngramLinkSelectSpec struct {
	ActorPlaceholder           string
	IncludeArchivedPlaceholder string
	WhereClauses               []string
	IncludeOrdering            bool
	LimitClause                string
}

type traversalState struct {
	visitedEngrams map[uuid.UUID]struct{}
	seenLinks      map[uuid.UUID]struct{}
	steps          []models.EngramLinkTraversalStep
}

// CreateEngramLink persists a directed link when source/target visibility and write access are satisfied.
// Cross-project links are allowed when the actor can read both nodes and write in the source project.
func CreateEngramLink(
	ctx context.Context,
	db Queryer,
	input EngramLinkCreateInput,
) (*models.EngramLinkRecord, error) {
	if err := validateCreateEngramLinkInput(input); err != nil {
		return nil, err
	}
	row := db.QueryRow(
		ctx,
		buildCreateEngramLinkSQL(),
		newEngramLinkUUID(),
		input.SourceEngramID,
		input.TargetEngramID,
		string(input.RelationType),
		input.Weight,
		input.TemporalWeight,
		input.Confidence,
		string(input.Origin),
		string(input.Status),
		emptyMapIfNil(input.EvidenceJSON),
		input.CreatedByUserID,
		input.LastReinforcedAt,
		nowEngramLinkUTC(),
		input.ActorUserID,
	)
	record, err := scanEngramLinkRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if isUniqueViolation(err) {
		return nil, ErrEngramLinkExists
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetEngramLink returns one visible link by link id.
func GetEngramLink(
	ctx context.Context,
	db Queryer,
	input EngramLinkGetInput,
) (*models.EngramLinkRecord, error) {
	if input.LinkID == uuid.Nil {
		return nil, errors.New("link_id is required")
	}
	if input.ActorUserID == uuid.Nil {
		return nil, errors.New("actor_user_id is required")
	}
	row := db.QueryRow(
		ctx,
		buildGetEngramLinkSQL(),
		input.LinkID,
		input.ActorUserID,
		input.IncludeArchived,
	)
	record, err := scanEngramLinkRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// ListEngramLinks returns visible links from one source engram.
func ListEngramLinks(
	ctx context.Context,
	db Queryer,
	input EngramLinkListInput,
) ([]models.EngramLinkRecord, error) {
	relationRaw, err := normalizeRelationFilter(input.RelationType)
	if err != nil {
		return nil, err
	}
	limit := normalizeLimit(input.Limit, 50)
	offset := normalizeOffset(input.Offset)
	rows, err := db.Query(
		ctx,
		buildListEngramLinksSQL(),
		input.SourceEngramID,
		input.ActorUserID,
		relationRaw,
		input.IncludeArchived,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]models.EngramLinkRecord, 0)
	for rows.Next() {
		record, scanErr := scanEngramLinkRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

// UpdateEngramLink mutates selected fields for an existing link.
func UpdateEngramLink(
	ctx context.Context,
	db Queryer,
	input EngramLinkUpdateInput,
) (*models.EngramLinkRecord, error) {
	if err := validateUpdateEngramLinkInput(input); err != nil {
		return nil, err
	}
	statusRaw, err := statusPointerToRaw(input.Status)
	if err != nil {
		return nil, err
	}
	row := db.QueryRow(
		ctx,
		buildUpdateEngramLinkSQL(),
		input.LinkID,
		input.Weight,
		input.TemporalWeight,
		input.Confidence,
		statusRaw,
		mapPointerToValue(input.EvidenceJSON),
		input.LastReinforcedAt,
		nowEngramLinkUTC(),
		input.ActorUserID,
	)
	record, err := scanEngramLinkRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// ArchiveEngramLink marks a link as archived.
func ArchiveEngramLink(
	ctx context.Context,
	db Queryer,
	input EngramLinkArchiveInput,
) (*models.EngramLinkRecord, error) {
	row := db.QueryRow(
		ctx,
		buildArchiveEngramLinkSQL(),
		input.LinkID,
		nowEngramLinkUTC(),
		input.ActorUserID,
	)
	record, err := scanEngramLinkRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// TraverseEngramLinks performs depth-limited traversal from one root engram.
func TraverseEngramLinks(
	ctx context.Context,
	db Queryer,
	input EngramLinkTraverseInput,
) ([]models.EngramLinkTraversalStep, error) {
	maxDepth := normalizeMaxDepth(input.MaxDepth)
	maxNeighbors := normalizeLimit(input.MaxNeighbors, 30)
	state := newTraversalState(input.RootEngramID)
	frontier := []uuid.UUID{input.RootEngramID}
	for depth := 1; depth <= maxDepth && len(frontier) > 0; depth++ {
		neighbors, err := listTraversalNeighbors(
			ctx,
			db,
			traversalQueryInput{
				SourceEngramIDs: frontier,
				ActorUserID:     input.ActorUserID,
				IncludeArchived: input.IncludeArchived,
				Limit:           maxNeighbors,
			},
		)
		if err != nil {
			return nil, err
		}
		frontier = state.collectNextFrontier(depth, neighbors)
	}
	return state.steps, nil
}

func buildCreateEngramLinkSQL() string {
	sourceAccess := buildMembershipReadClause(membershipReadClauseInput{ownerColumn: "source_engram.owner_user_id", visibilityColumn: "source_engram.visibility_scope", projectColumn: "source_engram.project_id", actorPlaceholder: "$14", includeOwnerless: true})
	targetAccess := buildMembershipReadClause(membershipReadClauseInput{ownerColumn: "target_engram.owner_user_id", visibilityColumn: "target_engram.visibility_scope", projectColumn: "target_engram.project_id", actorPlaceholder: "$14", includeOwnerless: true})
	writeAccess := buildMembershipWriteClause("source_engram.project_id", "$14")
	return fmt.Sprintf(
		`
		WITH source_engram AS (
			SELECT engram_id, project_id, owner_user_id, visibility_scope
			FROM engrams
			WHERE
				engram_id = $2
				AND deleted_at IS NULL
				AND %s
		),
		target_engram AS (
			SELECT engram_id, project_id, owner_user_id, visibility_scope
			FROM engrams
			WHERE
				engram_id = $3
				AND deleted_at IS NULL
				AND %s
		)
		INSERT INTO engram_links (
			link_id,
			project_id,
			source_engram_id,
			target_engram_id,
			relation_type,
			weight,
			temporal_weight,
			confidence,
			origin,
			status,
			evidence_json,
			created_by_user_id,
			last_reinforced_at,
			created_at,
			updated_at
		)
		SELECT
			$1,
			source_engram.project_id,
			source_engram.engram_id,
			target_engram.engram_id,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11,
			$12,
			$13,
			$13
		FROM source_engram
		JOIN target_engram
		  ON TRUE
		WHERE %s
		RETURNING %s
		`,
		sourceAccess,
		targetAccess,
		writeAccess,
		engramLinkColumns,
	)
}

func buildGetEngramLinkSQL() string {
	return buildVisibleEngramLinkSelectSQL(visibleEngramLinkSelectSpec{
		ActorPlaceholder:           "$2",
		IncludeArchivedPlaceholder: "$3",
		WhereClauses:               []string{"link.link_id = $1"},
	})
}

func buildListEngramLinksSQL() string {
	return buildVisibleEngramLinkSelectSQL(visibleEngramLinkSelectSpec{
		ActorPlaceholder:           "$2",
		IncludeArchivedPlaceholder: "$4",
		WhereClauses: []string{
			"link.source_engram_id = $1",
			"($3::text IS NULL OR link.relation_type = $3)",
		},
		IncludeOrdering: true,
		LimitClause:     "LIMIT $5 OFFSET $6",
	})
}

func buildUpdateEngramLinkSQL() string {
	writeAccess := buildMembershipWriteClause("source_engram.project_id", "$9")
	return fmt.Sprintf(
		`
		UPDATE engram_links link
		SET
			weight = COALESCE($2, link.weight),
			temporal_weight = COALESCE($3, link.temporal_weight),
			confidence = COALESCE($4, link.confidence),
			status = COALESCE($5, link.status),
			evidence_json = COALESCE($6, link.evidence_json),
			last_reinforced_at = COALESCE($7, link.last_reinforced_at),
			updated_at = $8
		FROM engrams source_engram, engrams target_engram
		WHERE
			link.link_id = $1
			AND source_engram.engram_id = link.source_engram_id
			AND target_engram.engram_id = link.target_engram_id
			AND source_engram.deleted_at IS NULL
			AND target_engram.deleted_at IS NULL
			AND source_engram.project_id = link.project_id
			AND %s
		RETURNING %s
		`,
		writeAccess,
		engramLinkColumnsWithAlias,
	)
}

func buildArchiveEngramLinkSQL() string {
	writeAccess := buildMembershipWriteClause("source_engram.project_id", "$3")
	return fmt.Sprintf(
		`
		UPDATE engram_links link
		SET
			status = 'archived',
			updated_at = $2
		FROM engrams source_engram
		WHERE
			link.link_id = $1
			AND source_engram.engram_id = link.source_engram_id
			AND source_engram.deleted_at IS NULL
			AND source_engram.project_id = link.project_id
			AND %s
		RETURNING %s
		`,
		writeAccess,
		engramLinkColumnsWithAlias,
	)
}

func listTraversalNeighbors(
	ctx context.Context,
	db Queryer,
	input traversalQueryInput,
) ([]models.EngramLinkRecord, error) {
	if len(input.SourceEngramIDs) == 0 {
		return []models.EngramLinkRecord{}, nil
	}
	rows, err := db.Query(
		ctx,
		buildTraversalNeighborSQL(),
		input.SourceEngramIDs,
		input.ActorUserID,
		input.IncludeArchived,
		input.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]models.EngramLinkRecord, 0)
	for rows.Next() {
		record, scanErr := scanEngramLinkRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func newTraversalState(rootEngramID uuid.UUID) traversalState {
	return traversalState{
		visitedEngrams: map[uuid.UUID]struct{}{rootEngramID: {}},
		seenLinks:      make(map[uuid.UUID]struct{}),
		steps:          make([]models.EngramLinkTraversalStep, 0),
	}
}

func (state *traversalState) collectNextFrontier(
	depth int,
	neighbors []models.EngramLinkRecord,
) []uuid.UUID {
	nextFrontier := make([]uuid.UUID, 0)
	for _, link := range neighbors {
		if state.hasSeenLink(link.LinkID) {
			continue
		}
		state.seenLinks[link.LinkID] = struct{}{}
		state.steps = append(
			state.steps,
			models.EngramLinkTraversalStep{
				Depth: depth,
				Link:  link,
			},
		)
		if state.hasVisitedEngram(link.TargetEngramID) {
			continue
		}
		state.visitedEngrams[link.TargetEngramID] = struct{}{}
		nextFrontier = append(nextFrontier, link.TargetEngramID)
	}
	return nextFrontier
}

func (state traversalState) hasSeenLink(linkID uuid.UUID) bool {
	_, exists := state.seenLinks[linkID]
	return exists
}

func (state traversalState) hasVisitedEngram(engramID uuid.UUID) bool {
	_, exists := state.visitedEngrams[engramID]
	return exists
}

func buildTraversalNeighborSQL() string {
	return buildVisibleEngramLinkSelectSQL(visibleEngramLinkSelectSpec{
		ActorPlaceholder:           "$2",
		IncludeArchivedPlaceholder: "$3",
		WhereClauses:               []string{"link.source_engram_id = ANY($1)"},
		IncludeOrdering:            true,
		LimitClause:                "LIMIT $4",
	})
}

func buildVisibleEngramLinkSelectSQL(
	spec visibleEngramLinkSelectSpec,
) string {
	sourceAccess := buildMembershipReadClause(membershipReadClauseInput{ownerColumn: "source_engram.owner_user_id", visibilityColumn: "source_engram.visibility_scope", projectColumn: "source_engram.project_id", actorPlaceholder: spec.ActorPlaceholder, includeOwnerless: true})
	targetAccess := buildMembershipReadClause(membershipReadClauseInput{ownerColumn: "target_engram.owner_user_id", visibilityColumn: "target_engram.visibility_scope", projectColumn: "target_engram.project_id", actorPlaceholder: spec.ActorPlaceholder, includeOwnerless: true})
	filters := append([]string{
		"source_engram.deleted_at IS NULL",
		"target_engram.deleted_at IS NULL",
		sourceAccess,
		targetAccess,
		fmt.Sprintf("(%s::boolean OR link.status <> 'archived')", spec.IncludeArchivedPlaceholder),
	}, spec.WhereClauses...)
	orderClause := ""
	if spec.IncludeOrdering {
		orderClause = `
			ORDER BY
				link.weight DESC,
				link.confidence DESC,
				link.temporal_weight DESC,
				link.last_reinforced_at DESC NULLS LAST,
				link.created_at DESC`
	}
	limitFragment := strings.TrimSpace(spec.LimitClause)
	if limitFragment != "" {
		limitFragment = "\n\t\t" + limitFragment
	}
	return fmt.Sprintf(
		`
		SELECT %s
		FROM engram_links link
		JOIN engrams source_engram
		  ON source_engram.engram_id = link.source_engram_id
		JOIN engrams target_engram
		  ON target_engram.engram_id = link.target_engram_id
		WHERE
			%s%s%s
		`,
		engramLinkColumnsWithAlias,
		strings.Join(filters, "\n\t\t\tAND "),
		orderClause,
		limitFragment,
	)
}

func scanEngramLinkRecord(row interface{ Scan(dest ...any) error }) (models.EngramLinkRecord, error) {
	record := models.EngramLinkRecord{}
	var relationRaw string
	var originRaw string
	var statusRaw string
	var evidenceRaw any
	err := row.Scan(
		&record.LinkID,
		&record.ProjectID,
		&record.SourceEngramID,
		&record.TargetEngramID,
		&relationRaw,
		&record.Weight,
		&record.TemporalWeight,
		&record.Confidence,
		&originRaw,
		&statusRaw,
		&evidenceRaw,
		&record.CreatedByUserID,
		&record.LastReinforcedAt,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return models.EngramLinkRecord{}, err
	}
	relationType, err := models.ParseEngramLinkRelationType(strings.TrimSpace(relationRaw))
	if err != nil {
		return models.EngramLinkRecord{}, err
	}
	origin, err := models.ParseEngramLinkOrigin(strings.TrimSpace(originRaw))
	if err != nil {
		return models.EngramLinkRecord{}, err
	}
	status, err := models.ParseEngramLinkStatus(strings.TrimSpace(statusRaw))
	if err != nil {
		return models.EngramLinkRecord{}, err
	}
	evidenceJSON, err := decodeMapJSON(evidenceRaw)
	if err != nil {
		return models.EngramLinkRecord{}, err
	}
	record.RelationType = relationType
	record.Origin = origin
	record.Status = status
	record.EvidenceJSON = evidenceJSON
	return record, nil
}

func decodeMapJSON(value any) (map[string]any, error) {
	switch typed := value.(type) {
	case nil:
		return map[string]any{}, nil
	case map[string]any:
		if typed == nil {
			return map[string]any{}, nil
		}
		return typed, nil
	case []byte:
		return unmarshalMapJSON(typed)
	case string:
		return unmarshalMapJSON([]byte(typed))
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return nil, fmt.Errorf("marshal map json: %w", err)
		}
		return unmarshalMapJSON(encoded)
	}
}

func unmarshalMapJSON(encoded []byte) (map[string]any, error) {
	trimmed := bytes.TrimSpace(encoded)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return map[string]any{}, nil
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(trimmed, &decoded); err != nil {
		return nil, fmt.Errorf("decode map json: %w", err)
	}
	if decoded == nil {
		return map[string]any{}, nil
	}
	return decoded, nil
}

func validateCreateEngramLinkInput(input EngramLinkCreateInput) error {
	validators := []func(EngramLinkCreateInput) error{
		validateCreateEngramLinkRequiredFields,
		validateCreateEngramLinkEnums,
		validateCreateEngramLinkScores,
	}
	for _, validator := range validators {
		if err := validator(input); err != nil {
			return err
		}
	}
	return nil
}

func validateCreateEngramLinkRequiredFields(input EngramLinkCreateInput) error {
	requiredUUIDs := []struct {
		name  string
		value uuid.UUID
	}{
		{name: "source_engram_id", value: input.SourceEngramID},
		{name: "target_engram_id", value: input.TargetEngramID},
		{name: "created_by_user_id", value: input.CreatedByUserID},
		{name: "actor_user_id", value: input.ActorUserID},
	}
	for _, required := range requiredUUIDs {
		if required.value == uuid.Nil {
			return fmt.Errorf("%s is required", required.name)
		}
	}
	return nil
}

func validateCreateEngramLinkEnums(input EngramLinkCreateInput) error {
	if _, err := models.ParseEngramLinkRelationType(string(input.RelationType)); err != nil {
		return err
	}
	if _, err := models.ParseEngramLinkOrigin(string(input.Origin)); err != nil {
		return err
	}
	if _, err := models.ParseEngramLinkStatus(string(input.Status)); err != nil {
		return err
	}
	return nil
}

func validateCreateEngramLinkScores(input EngramLinkCreateInput) error {
	scoreFields := []struct {
		name  string
		value float64
	}{
		{name: "weight", value: input.Weight},
		{name: "temporal_weight", value: input.TemporalWeight},
		{name: "confidence", value: input.Confidence},
	}
	for _, score := range scoreFields {
		if err := validateNormalizedScore(score.name, score.value); err != nil {
			return err
		}
	}
	return nil
}

func validateUpdateEngramLinkInput(input EngramLinkUpdateInput) error {
	if input.LinkID == uuid.Nil {
		return errors.New("link_id is required")
	}
	if input.ActorUserID == uuid.Nil {
		return errors.New("actor_user_id is required")
	}
	if err := validateOptionalScore("weight", input.Weight); err != nil {
		return err
	}
	if err := validateOptionalScore("temporal_weight", input.TemporalWeight); err != nil {
		return err
	}
	if err := validateOptionalScore("confidence", input.Confidence); err != nil {
		return err
	}
	if input.Status != nil {
		if _, err := models.ParseEngramLinkStatus(string(*input.Status)); err != nil {
			return err
		}
	}
	return nil
}

func validateOptionalScore(name string, value *float64) error {
	if value == nil {
		return nil
	}
	return validateNormalizedScore(name, *value)
}

func validateNormalizedScore(name string, value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return fmt.Errorf("%s must be a finite number", name)
	}
	if value < 0 || value > 1 {
		return fmt.Errorf("%s must be between 0 and 1", name)
	}
	return nil
}

func normalizeRelationFilter(
	relationType *models.EngramLinkRelationType,
) (*string, error) {
	if relationType == nil {
		return nil, nil
	}
	if _, err := models.ParseEngramLinkRelationType(string(*relationType)); err != nil {
		return nil, err
	}
	raw := string(*relationType)
	return &raw, nil
}

func statusPointerToRaw(status *models.EngramLinkStatus) (*string, error) {
	if status == nil {
		return nil, nil
	}
	if _, err := models.ParseEngramLinkStatus(string(*status)); err != nil {
		return nil, err
	}
	raw := string(*status)
	return &raw, nil
}

func normalizeLimit(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func normalizeOffset(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func normalizeMaxDepth(value int) int {
	if value <= 0 {
		return 1
	}
	if value > 6 {
		return 6
	}
	return value
}

func emptyMapIfNil(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func mapPointerToValue(value *map[string]any) any {
	if value == nil {
		return nil
	}
	if *value == nil {
		return map[string]any{}
	}
	return *value
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505"
}
