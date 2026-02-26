package workflow

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"engram/internal/models"
)

const (
	defaultSnapshotEveryNNotes  = 3
	defaultRunStatusCollected   = "collected"
	defaultRunStatusSynthesized = "synthesized"
	defaultRunStatusSnapshotted = "snapshotted"
	defaultRunStatusCompleted   = "completed"
)

// AgentSourceInput captures source evidence passed to workflow runs.
type AgentSourceInput struct {
	URL        string     `json:"url"`
	Title      *string    `json:"title,omitempty"`
	Snippet    string     `json:"snippet"`
	CapturedAt *time.Time `json:"captured_at,omitempty"`
}

// AgentDecision captures synthesized decision output.
type AgentDecision struct {
	Decision  string `json:"decision"`
	Rationale string `json:"rationale"`
}

// AgentState stores persisted workflow state by thread id.
type AgentState struct {
	ProjectID           string             `json:"project_id"`
	ThreadID            string             `json:"thread_id"`
	Objective           string             `json:"objective"`
	Notes               []string           `json:"notes,omitempty"`
	Assumptions         []string           `json:"assumptions,omitempty"`
	Tags                []string           `json:"tags,omitempty"`
	Keywords            []string           `json:"keywords,omitempty"`
	Sources             []AgentSourceInput `json:"sources,omitempty"`
	SynthesisTitle      string             `json:"synthesis_title,omitempty"`
	SynthesisAbstract   string             `json:"synthesis_abstract,omitempty"`
	SynthesisMarkdown   string             `json:"synthesis_markdown,omitempty"`
	Decisions           []AgentDecision    `json:"decisions,omitempty"`
	OpenQuestions       []string           `json:"open_questions,omitempty"`
	Status              string             `json:"status"`
	AutoPersistEngram   bool               `json:"auto_persist_engram"`
	EngramID            *string            `json:"engram_id,omitempty"`
	SnapshotEnabled     bool               `json:"snapshot_enabled"`
	SnapshotEveryNNotes int                `json:"snapshot_every_n_notes"`
	SnapshotCount       int                `json:"snapshot_count"`
	SnapshotEngramIDs   []string           `json:"snapshot_engram_ids,omitempty"`
}

// AgentRunRequest starts a fresh workflow run.
type AgentRunRequest struct {
	ProjectID           string             `json:"project_id"`
	ThreadID            string             `json:"thread_id"`
	Objective           string             `json:"objective"`
	Notes               []string           `json:"notes,omitempty"`
	Assumptions         []string           `json:"assumptions,omitempty"`
	Tags                []string           `json:"tags,omitempty"`
	Keywords            []string           `json:"keywords,omitempty"`
	Sources             []AgentSourceInput `json:"sources,omitempty"`
	AutoPersistEngram   bool               `json:"auto_persist_engram"`
	SnapshotEnabled     bool               `json:"snapshot_enabled"`
	SnapshotEveryNNotes int                `json:"snapshot_every_n_notes"`
}

// AgentResumeRequest appends updates to an existing workflow thread.
type AgentResumeRequest struct {
	Notes               []string           `json:"notes,omitempty"`
	Assumptions         []string           `json:"assumptions,omitempty"`
	Tags                []string           `json:"tags,omitempty"`
	Keywords            []string           `json:"keywords,omitempty"`
	Sources             []AgentSourceInput `json:"sources,omitempty"`
	AutoPersistEngram   *bool              `json:"auto_persist_engram,omitempty"`
	SnapshotEnabled     *bool              `json:"snapshot_enabled,omitempty"`
	SnapshotEveryNNotes *int               `json:"snapshot_every_n_notes,omitempty"`
}

// EngramCreator persists an engram payload and returns created metadata.
type EngramCreator func(
	ctx context.Context,
	payload models.MemoryEngramCreate,
	enrichmentOrigin string,
) (*models.EngramCreateResponse, error)

// Service executes and checkpoints agent workflow runs.
type Service struct {
	deps   serviceDeps
	mu     sync.RWMutex
	states map[string]AgentState
}

type serviceDeps struct {
	nowUTC       func() time.Time
	createEngram EngramCreator
}

func defaultServiceDeps(createEngram EngramCreator) serviceDeps {
	return serviceDeps{
		nowUTC:       func() time.Time { return time.Now().UTC() },
		createEngram: createEngram,
	}
}

// NewService creates an in-memory workflow service.
func NewService(createEngram EngramCreator) *Service {
	return &Service{
		deps:   defaultServiceDeps(createEngram),
		states: map[string]AgentState{},
	}
}

// Run executes a fresh workflow thread.
func (service *Service) Run(ctx context.Context, request AgentRunRequest) (AgentState, error) {
	state := service.collectState(agentStateFromRequest(request))
	state = service.synthesizeState(state)
	var err error
	state, err = service.snapshotState(ctx, state)
	if err != nil {
		return AgentState{}, err
	}
	state, err = service.persistState(ctx, state)
	if err != nil {
		return AgentState{}, err
	}
	service.saveState(state.ThreadID, state)
	return cloneAgentState(state), nil
}

// GetState returns persisted state for a thread if available.
func (service *Service) GetState(threadID string) (AgentState, bool) {
	trimmedThreadID := strings.TrimSpace(threadID)
	if trimmedThreadID == "" {
		return AgentState{}, false
	}
	service.mu.RLock()
	defer service.mu.RUnlock()
	state, exists := service.states[trimmedThreadID]
	if !exists {
		return AgentState{}, false
	}
	return cloneAgentState(state), true
}

// Resume merges updates into a prior thread and reruns workflow steps.
func (service *Service) Resume(
	ctx context.Context,
	threadID string,
	updates AgentResumeRequest,
) (AgentState, bool, error) {
	prior, found := service.GetState(threadID)
	if !found {
		return AgentState{}, false, nil
	}
	merged := mergeResumeState(prior, updates)
	state := service.collectState(merged)
	state = service.synthesizeState(state)
	var err error
	state, err = service.snapshotState(ctx, state)
	if err != nil {
		return AgentState{}, true, err
	}
	state, err = service.persistState(ctx, state)
	if err != nil {
		return AgentState{}, true, err
	}
	service.saveState(threadID, state)
	return cloneAgentState(state), true, nil
}

func (service *Service) saveState(threadID string, state AgentState) {
	service.mu.Lock()
	defer service.mu.Unlock()
	service.states[threadID] = cloneAgentState(state)
}

func agentStateFromRequest(request AgentRunRequest) AgentState {
	snapshotEvery := request.SnapshotEveryNNotes
	if snapshotEvery < 1 {
		snapshotEvery = defaultSnapshotEveryNNotes
	}
	return AgentState{
		ProjectID:           strings.TrimSpace(request.ProjectID),
		ThreadID:            strings.TrimSpace(request.ThreadID),
		Objective:           strings.TrimSpace(request.Objective),
		Notes:               normalizeStringList(request.Notes),
		Assumptions:         normalizeStringList(request.Assumptions),
		Tags:                normalizeStringList(request.Tags),
		Keywords:            normalizeStringList(request.Keywords),
		Sources:             normalizeSourceList(request.Sources),
		AutoPersistEngram:   request.AutoPersistEngram,
		SnapshotEnabled:     request.SnapshotEnabled,
		SnapshotEveryNNotes: snapshotEvery,
		SnapshotCount:       0,
		SnapshotEngramIDs:   []string{},
	}
}

func mergeResumeState(prior AgentState, updates AgentResumeRequest) AgentState {
	autoPersist := prior.AutoPersistEngram
	if updates.AutoPersistEngram != nil {
		autoPersist = *updates.AutoPersistEngram
	}
	snapshotEnabled := prior.SnapshotEnabled
	if updates.SnapshotEnabled != nil {
		snapshotEnabled = *updates.SnapshotEnabled
	}
	snapshotEvery := prior.SnapshotEveryNNotes
	if updates.SnapshotEveryNNotes != nil {
		snapshotEvery = *updates.SnapshotEveryNNotes
	}
	if snapshotEvery < 1 {
		snapshotEvery = 1
	}
	return AgentState{
		ProjectID:           prior.ProjectID,
		ThreadID:            prior.ThreadID,
		Objective:           prior.Objective,
		Notes:               append(cloneStringSlice(prior.Notes), normalizeStringList(updates.Notes)...),
		Assumptions:         append(cloneStringSlice(prior.Assumptions), normalizeStringList(updates.Assumptions)...),
		Tags:                mergeUniqueSorted(prior.Tags, updates.Tags),
		Keywords:            mergeUniqueSorted(prior.Keywords, updates.Keywords),
		Sources:             append(cloneSourceSlice(prior.Sources), normalizeSourceList(updates.Sources)...),
		AutoPersistEngram:   autoPersist,
		SnapshotEnabled:     snapshotEnabled,
		SnapshotEveryNNotes: snapshotEvery,
		SnapshotCount:       maxInt(prior.SnapshotCount, len(prior.SnapshotEngramIDs)),
		SnapshotEngramIDs:   cloneStringSlice(prior.SnapshotEngramIDs),
	}
}

func (service *Service) collectState(state AgentState) AgentState {
	state.Notes = normalizeStringList(state.Notes)
	state.Assumptions = normalizeStringList(state.Assumptions)
	state.Tags = normalizeStringList(state.Tags)
	state.Keywords = normalizeStringList(state.Keywords)
	state.Sources = normalizeSourceList(state.Sources)
	if state.SnapshotEveryNNotes < 1 {
		state.SnapshotEveryNNotes = 1
	}
	state.SnapshotCount = maxInt(state.SnapshotCount, len(state.SnapshotEngramIDs))
	state.Status = defaultRunStatusCollected
	return state
}

func (service *Service) synthesizeState(state AgentState) AgentState {
	objective := state.Objective
	if objective == "" {
		objective = "Untitled objective"
	}
	abstract := "No notes were provided for objective: " + objective
	notesSection := "- No notes provided"
	decisions := []AgentDecision{}
	openQuestions := []string{}
	if len(state.Notes) > 0 {
		abstract = trimForSummary(state.Notes[0], 240)
		notesSection = joinBullets(state.Notes)
		decisions = []AgentDecision{
			{
				Decision:  "Consolidate run into memory engram",
				Rationale: "Preserve analysis context across sessions.",
			},
		}
	} else {
		openQuestions = []string{"Capture at least one research note before finalizing."}
	}
	state.SynthesisTitle = "Run Summary: " + trimForSummary(objective, 80)
	state.SynthesisAbstract = abstract
	state.SynthesisMarkdown = buildSynthesisMarkdown(objective, notesSection, state.Assumptions, service.deps.nowUTC())
	state.Decisions = decisions
	state.OpenQuestions = openQuestions
	state.Status = defaultRunStatusSynthesized
	return state
}

func (service *Service) snapshotState(ctx context.Context, state AgentState) (AgentState, error) {
	if !state.SnapshotEnabled {
		state.Status = defaultRunStatusSnapshotted
		return state, nil
	}
	noteCount := len(state.Notes)
	for noteCount >= (state.SnapshotCount+1)*state.SnapshotEveryNNotes {
		snapshotIndex := state.SnapshotCount + 1
		snapshotID, err := service.createSnapshotEngram(ctx, state, snapshotIndex)
		if err != nil {
			return AgentState{}, err
		}
		state.SnapshotEngramIDs = append(state.SnapshotEngramIDs, snapshotID)
		state.SnapshotCount = snapshotIndex
	}
	state.Status = defaultRunStatusSnapshotted
	return state, nil
}

func (service *Service) persistState(ctx context.Context, state AgentState) (AgentState, error) {
	if !state.AutoPersistEngram {
		state.EngramID = nil
		state.Status = defaultRunStatusCompleted
		return state, nil
	}
	if service.deps.createEngram == nil {
		state.EngramID = nil
		state.Status = defaultRunStatusCompleted
		return state, nil
	}
	engramID, err := service.createFinalEngram(ctx, state)
	if err != nil {
		return AgentState{}, err
	}
	state.EngramID = &engramID
	state.Status = defaultRunStatusCompleted
	return state, nil
}

func (service *Service) createSnapshotEngram(
	ctx context.Context,
	state AgentState,
	snapshotIndex int,
) (string, error) {
	threadID := state.ThreadID
	payload := models.MemoryEngramCreate{
		ProjectID:               state.ProjectID,
		ThreadID:                &threadID,
		Title:                   "Snapshot " + intToString(snapshotIndex) + ": " + trimForSummary(state.Objective, 70),
		Abstract:                resolveSnapshotAbstract(state),
		DetailedSummaryMarkdown: buildSnapshotMarkdown(state, snapshotIndex, service.deps.nowUTC()),
		Decisions: []models.Decision{
			{
				Decision:  "Capture periodic snapshot " + intToString(snapshotIndex),
				Rationale: "Persist evolving context during long-running research.",
			},
		},
		Assumptions:   cloneStringSlice(state.Assumptions),
		OpenQuestions: []string{},
		Claims:        buildClaims(state, "Snapshot "+intToString(snapshotIndex)+" captured for objective '"+state.Objective+"'."),
		Tags:          mergeUniqueSorted(state.Tags, []string{"snapshot"}),
		Keywords:      mergeUniqueSorted(state.Keywords, []string{"snapshot"}),
	}
	created, err := service.deps.createEngram(ctx, payload, "agent.snapshot")
	if err != nil {
		return "", err
	}
	return created.EngramID.String(), nil
}

func (service *Service) createFinalEngram(ctx context.Context, state AgentState) (string, error) {
	threadID := state.ThreadID
	payload := models.MemoryEngramCreate{
		ProjectID:               state.ProjectID,
		ThreadID:                &threadID,
		Title:                   resolveFinalTitle(state),
		Abstract:                resolveFinalAbstract(state),
		DetailedSummaryMarkdown: resolveFinalMarkdown(state),
		Decisions:               toModelDecisions(state.Decisions),
		Assumptions:             cloneStringSlice(state.Assumptions),
		OpenQuestions:           cloneStringSlice(state.OpenQuestions),
		Claims: buildClaims(
			state,
			"Research run for objective '"+state.Objective+"' was captured.",
		),
		Tags:     cloneStringSlice(state.Tags),
		Keywords: cloneStringSlice(state.Keywords),
	}
	created, err := service.deps.createEngram(ctx, payload, "agent.persist")
	if err != nil {
		return "", err
	}
	return created.EngramID.String(), nil
}

func toModelDecisions(decisions []AgentDecision) []models.Decision {
	result := make([]models.Decision, 0, len(decisions))
	for _, decision := range decisions {
		result = append(result, models.Decision{Decision: decision.Decision, Rationale: decision.Rationale})
	}
	return result
}

func buildClaims(state AgentState, claimText string) []models.Claim {
	if len(state.Sources) == 0 {
		return []models.Claim{}
	}
	sources := make([]models.SupportingSource, 0, len(state.Sources))
	for _, source := range state.Sources {
		capturedAt := time.Now().UTC()
		if source.CapturedAt != nil {
			capturedAt = source.CapturedAt.UTC()
		}
		sources = append(sources, models.SupportingSource{
			URL:        strings.TrimSpace(source.URL),
			Title:      source.Title,
			Snippet:    strings.TrimSpace(source.Snippet),
			CapturedAt: capturedAt,
		})
	}
	return []models.Claim{{Claim: claimText, SupportingSources: sources}}
}

func buildSynthesisMarkdown(
	objective string,
	notesSection string,
	assumptions []string,
	generatedAt time.Time,
) string {
	assumptionSection := "- None"
	if len(assumptions) > 0 {
		assumptionSection = joinBullets(assumptions)
	}
	return "## Objective\n" + objective +
		"\n\n## Notes\n" + notesSection +
		"\n\n## Assumptions\n" + assumptionSection +
		"\n\n## Generated At\n" + generatedAt.UTC().Format(time.RFC3339)
}

func buildSnapshotMarkdown(state AgentState, snapshotIndex int, generatedAt time.Time) string {
	return "## Snapshot Index\n" + intToString(snapshotIndex) +
		"\n\n## Objective\n" + resolveObjective(state.Objective) +
		"\n\n## Notes Count\n" + intToString(len(state.Notes)) +
		"\n\n## Notes\n" + resolveBulletSection(state.Notes, "- No notes") +
		"\n\n## Assumptions\n" + resolveBulletSection(state.Assumptions, "- None") +
		"\n\n## Snapshot Generated At\n" + generatedAt.UTC().Format(time.RFC3339)
}

func resolveObjective(objective string) string {
	trimmed := strings.TrimSpace(objective)
	if trimmed == "" {
		return "Untitled objective"
	}
	return trimmed
}

func resolveSnapshotAbstract(state AgentState) string {
	if len(state.Notes) == 0 {
		return "Snapshot for objective: " + resolveObjective(state.Objective)
	}
	return trimForSummary(state.Notes[0], 220)
}

func resolveFinalTitle(state AgentState) string {
	if strings.TrimSpace(state.SynthesisTitle) != "" {
		return state.SynthesisTitle
	}
	return "Run Summary: " + trimForSummary(resolveObjective(state.Objective), 80)
}

func resolveFinalAbstract(state AgentState) string {
	if strings.TrimSpace(state.SynthesisAbstract) != "" {
		return state.SynthesisAbstract
	}
	return "Research objective: " + resolveObjective(state.Objective)
}

func resolveFinalMarkdown(state AgentState) string {
	if strings.TrimSpace(state.SynthesisMarkdown) != "" {
		return state.SynthesisMarkdown
	}
	return "No synthesis markdown."
}

func resolveBulletSection(values []string, fallback string) string {
	if len(values) == 0 {
		return fallback
	}
	return joinBullets(values)
}

func joinBullets(values []string) string {
	lines := make([]string, 0, len(values))
	for _, value := range values {
		lines = append(lines, "- "+value)
	}
	return strings.Join(lines, "\n")
}

func normalizeStringList(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func normalizeSourceList(values []AgentSourceInput) []AgentSourceInput {
	if len(values) == 0 {
		return []AgentSourceInput{}
	}
	normalized := make([]AgentSourceInput, 0, len(values))
	for _, value := range values {
		url := strings.TrimSpace(value.URL)
		snippet := strings.TrimSpace(value.Snippet)
		if url == "" || snippet == "" {
			continue
		}
		normalized = append(normalized, AgentSourceInput{
			URL:        url,
			Title:      value.Title,
			Snippet:    snippet,
			CapturedAt: value.CapturedAt,
		})
	}
	return normalized
}

func mergeUniqueSorted(left []string, right []string) []string {
	seen := map[string]struct{}{}
	for _, value := range normalizeStringList(left) {
		seen[value] = struct{}{}
	}
	for _, value := range normalizeStringList(right) {
		seen[value] = struct{}{}
	}
	merged := make([]string, 0, len(seen))
	for value := range seen {
		merged = append(merged, value)
	}
	sort.Strings(merged)
	return merged
}

func cloneAgentState(state AgentState) AgentState {
	cloned := state
	cloned.Notes = cloneStringSlice(state.Notes)
	cloned.Assumptions = cloneStringSlice(state.Assumptions)
	cloned.Tags = cloneStringSlice(state.Tags)
	cloned.Keywords = cloneStringSlice(state.Keywords)
	cloned.SnapshotEngramIDs = cloneStringSlice(state.SnapshotEngramIDs)
	cloned.Sources = cloneSourceSlice(state.Sources)
	cloned.Decisions = cloneDecisionSlice(state.Decisions)
	cloned.OpenQuestions = cloneStringSlice(state.OpenQuestions)
	return cloned
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func cloneSourceSlice(values []AgentSourceInput) []AgentSourceInput {
	if len(values) == 0 {
		return []AgentSourceInput{}
	}
	cloned := make([]AgentSourceInput, len(values))
	copy(cloned, values)
	return cloned
}

func cloneDecisionSlice(values []AgentDecision) []AgentDecision {
	if len(values) == 0 {
		return []AgentDecision{}
	}
	cloned := make([]AgentDecision, len(values))
	copy(cloned, values)
	return cloned
}

func trimForSummary(value string, maxChars int) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= maxChars {
		return trimmed
	}
	return trimmed[:maxChars]
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func intToString(value int) string {
	return strconv.Itoa(value)
}
