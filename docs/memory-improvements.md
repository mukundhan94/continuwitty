# Engram Vault - LLM Memory Intelligence Improvements Plan

**Status:** Implemented baseline (Phases 35-40 completed on 2026-03-03)  
**Date Created:** March 1, 2026  
**Target Release:** Q2 2026  
**Complexity:** High | **Impact on LLMs:** Critical

---

## Executive Summary

Current Engram Vault memory system is **static** — LLMs retrieve engrams by semantic similarity alone. This plan transforms it into **adaptive, learning memory** that:

- **Prevents hallucinations** via contradiction detection
- **Respects context windows** via token-aware selection
- **Improves recall** via engagement-based scoring
- **Reduces noise** via automatic consolidation
- **Guides LLMs** via autonomous curation suggestions
- **Builds trust** via feedback-driven learning

## Implementation Alignment (2026-03-03)

This document started as a blueprint. The baseline implementation is now shipped across roadmap Phases 35-40:

1. Improvement 1 (Relevance + Engagement Scoring): implemented via engagement/freshness-aware rerank signals and temporal query controls in Phases 35, 36, and 39.
2. Improvement 2 (Memory Engagement Feedback Loop): implemented in Phase 36 with persisted feedback aggregates and REST/MCP feedback submission (`engram.feedback`).
3. Improvement 3 (Automatic Time-Decay & Consolidation): implemented in Phase 37 with freshness decay refresh, consolidation refresh/list/action APIs, and MCP parity.
4. Improvement 4 (Contradiction Detection & Warning): implemented in Phase 38 with contradiction warning synthesis, contradiction alert persistence, and admin review flows.
5. Improvement 5 (Context Window Optimization): implemented in Phase 39 with `context_token_budget` and retrieval-audit diagnostics (`context_token_estimate`, `context_token_truncated`).
6. Improvement 6 (Autonomous Memory Curation): implemented in Phase 40 with curation suggestion persistence, admin REST + MCP list/action workflows, deterministic generation hooks, acceptance coverage, and benchmarks.

Canonical implementation status now lives in:

- `Plan.md` (phase-level completion + scope)
- `migration/checkpoints/checkpoint.md` (milestone tracker)
- `docs/implementation-log.md` (chronological evidence)

---

## Improvement 1: Relevance + Engagement Scoring

### Current State
- Only vector cosine similarity controls ranking
- All engrams equally weighted regardless of usefulness
- No way to learn which memories actually helped

### Target State
**Four-factor relevance composite score:**

$$\text{relevance\_score} = (s \times 0.4) + (e \times 0.3) + (r \times 0.2) + (a \times 0.1)$$

Where:
- $s$ = semantic relevance (cosine similarity)
- $e$ = engagement score (normalized access count)
- $r$ = recency score (time-decay: $e^{-\lambda t}$)
- $a$ = authority score (from high-quality sessions)

### Schema Changes
```sql
-- Track engagement
ALTER TABLE engrams ADD COLUMN access_count INT DEFAULT 0;
ALTER TABLE engrams ADD COLUMN last_accessed_at TIMESTAMPTZ;
ALTER TABLE engrams ADD COLUMN avg_relevance_feedback FLOAT CHECK (avg_relevance_feedback >= 0 AND avg_relevance_feedback <= 1);

-- Track source quality
ALTER TABLE engrams ADD COLUMN source_session_quality_score FLOAT DEFAULT 0.5;

-- Indexes for efficient ranking
CREATE INDEX engrams_engagement_recency_idx 
  ON engrams (access_count DESC, last_accessed_at DESC);

CREATE INDEX engrams_authority_idx
  ON engrams (source_session_quality_score DESC, created_at DESC);

-- Audit trail
CREATE TABLE engram_access_events (
  event_id UUID PRIMARY KEY,
  engram_id UUID NOT NULL REFERENCES engrams(engram_id) ON DELETE CASCADE,
  session_id UUID,
  accessed_by_user_id UUID REFERENCES users(user_id),
  context_query TEXT,  -- What was LLM searching for?
  timestamp TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX engram_access_events_engram_idx ON engram_access_events (engram_id, accessed_at DESC);
CREATE INDEX engram_access_events_session_idx ON engram_access_events (session_id);
```

### Service Layer Changes
**New method in `EngramService`:**
```go
func (s *EngramService) RankEngrams(
    ctx context.Context,
    engrams []Engram,
    semanticScores map[string]float64,  // from vector query
    decayLambda float64,  // time-decay parameter
    sessionID string,  // optional: to compute authority
) []RankedEngram {
    // Apply four-factor scoring
    // Return sorted by composite relevance
}
```

**Update `engram.query` MCP tool to use multi-factor ranking**

### Implementation Complexity
- **Schema:** Low (column additions)
- **Service:** Medium (ranking algorithm)
- **MCP:** Low (use existing query endpoint)

### Success Metrics
- Top-3 retrieval precision increases 15%
- Engagement-biased results preferred in user feedback
- Less "false positive" irrelevant-but-similar matches

---

## Improvement 2: Memory Engagement Feedback Loop

### Current State
- LLM retrieves engrams but never tells system if they were useful
- No learning signal — all retrievals equally weighted
- Memory becomes a passive archive

### Target State
**LLM provides structured feedback after using retrieved engrams:**

```
chat.send_message(...)
→ Receives response with retrieved engrams + session_id
→ Later: engram.provide_feedback({
    engram_id,
    session_id,
    was_useful: true|false,
    was_contradicted: true|false,
    relevance_score: 1-5,
    integration_depth: "mentioned"|"elaborated"|"contradicted"|"ignored"
  })
```

### Schema Changes
```sql
CREATE TABLE engram_feedback (
  feedback_id UUID PRIMARY KEY,
  engram_id UUID NOT NULL REFERENCES engrams(engram_id) ON DELETE CASCADE,
  session_id UUID,
  feedback_from_user_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
  was_useful BOOLEAN NOT NULL,
  was_contradicted BOOLEAN NOT NULL,
  relevance_score INT NOT NULL CHECK (relevance_score >= 1 AND relevance_score <= 5),
  integration_depth TEXT NOT NULL CHECK (integration_depth IN ('mentioned', 'elaborated', 'contradicted', 'ignored')),
  feedback_text TEXT,  -- LLM can explain why it was/wasn't useful
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX engram_feedback_engram_created_idx ON engram_feedback (engram_id, created_at DESC);
CREATE INDEX engram_feedback_useful_idx ON engram_feedback (was_useful) WHERE was_useful = true;

-- Compute running averages
ALTER TABLE engrams ADD COLUMN useful_count INT DEFAULT 0;
ALTER TABLE engrams ADD COLUMN contradiction_count INT DEFAULT 0;
ALTER TABLE engrams ADD COLUMN feedback_count INT DEFAULT 0;

-- Update on_insert trigger for engram_feedback to aggregate stats
```

### MCP Tool: New
```
Tool: engram.provide_feedback
Scope: write
Parameters:
  - engram_id (required)
  - session_id (optional)
  - was_useful (required)
  - was_contradicted (required)
  - relevance_score (1-5)
  - integration_depth (required)
  - feedback_text (optional)
Returns:
  - feedback_recorded: true
  - updated_engram_stats: {...}
```

### Service Layer Changes
**New service: `EngramFeedbackService`**
```go
type EngramFeedbackService struct {
    feedbackRepo FeedbackRepository
    engramService EngramService
}

func (s *EngramFeedbackService) RecordFeedback(
    ctx context.Context,
    req ProvideFeedbackRequest,
) (*FeedbackResponse, error) {
    // Validate engram exists & is accessible
    // Record feedback
    // Update engram aggregate stats
    // Re-compute avg_relevance_feedback
    // Trigger contradiction detection if contradicted=true
}
```

### Implementation Complexity
- **Schema:** Medium (new table + indexes + aggregates)
- **Service:** Medium (feedback recording + stats aggregation)
- **MCP:** Low (new tool registration)

### Success Metrics
- 70%+ of retrieved engrams receive feedback within 24h
- Useful-feedback distribution stabilizes system learning
- Contradiction rate < 5% (good signal quality)

---

## Improvement 3: Automatic Time-Decay & Consolidation

### Current State
- All engrams equally weighted over time
- No mechanism to merge duplicates/related memories
- Memory bloat: "What was that thing I learned 6 months ago?" hard to find among noise

### Target State
**Memories decay unless reinforced; system suggests consolidation:**

```
freshness_score = 1.0 * exp(-λ * days_since_access)
// λ = 0.01 (half-life ≈ 70 days)

// Every day:
//  - Compute freshness for all engrams
//  - Find candidates for consolidation:
//    - Exact duplicates (hash match)
//    - Theme duplicates (semantic similarity + same tags)
//    - Superseded knowledge (contradicted)
//  - Suggest to LLM/user
```

### Schema Changes
```sql
-- Time-decay tracking
ALTER TABLE engrams ADD COLUMN freshness_score FLOAT DEFAULT 1.0;
ALTER TABLE engrams ADD COLUMN freshness_last_computed_at TIMESTAMPTZ;

-- Consolidation tracking
CREATE TABLE engram_consolidation_suggestions (
  suggestion_id UUID PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(project_id),
  source_engram_ids UUID[] NOT NULL,
  target_engram_id UUID REFERENCES engrams(engram_id) ON DELETE SET NULL,
  consolidation_type TEXT NOT NULL CHECK (consolidation_type IN ('exact_duplicate', 'theme_duplicate', 'superseded', 'complementary')),
  reason TEXT NOT NULL,
  consolidation_hash TEXT,  -- Group similar ones
  confidence_score FLOAT CHECK (confidence_score >= 0 AND confidence_score <= 1),
  status TEXT DEFAULT 'suggested' CHECK (status IN ('suggested', 'merged', 'rejected')),
  suggested_at TIMESTAMPTZ DEFAULT now(),
  actioned_at TIMESTAMPTZ,
  action_taken_by UUID REFERENCES users(user_id)
);

CREATE INDEX consolidation_suggestions_project_status_idx 
  ON engram_consolidation_suggestions (project_id, status);

CREATE INDEX consolidation_suggestions_source_engram_idx 
  ON engram_consolidation_suggestions USING GIN (source_engram_ids);

-- Result of consolidation
CREATE TABLE engram_consolidation_history (
  history_id UUID PRIMARY KEY,
  suggestion_id UUID REFERENCES engram_consolidation_suggestions,
  source_engrams UUID[] NOT NULL,
  merged_into_engram_id UUID REFERENCES engrams(engram_id),
  merge_summary TEXT,  -- AI-generated summary
  merged_at TIMESTAMPTZ DEFAULT now()
);
```

### Service Layer Changes
**New service: `EngramConsolidationService`**
```go
type ConsolidationCandidate struct {
    SourceIDs         []uuid.UUID
    Type              string  // exact_duplicate|theme_duplicate|superseded
    Reason            string
    ConfidenceScore   float64
}

func (s *EngramConsolidationService) FindConsolidationCandidates(
    ctx context.Context,
    projectID string,
) ([]ConsolidationCandidate, error) {
    // Find duplicates (content_hash)
    // Find theme duplicates (semantic + tag overlap)
    // Find superseded (contradicted_by + old)
    // Rank by confidence
}

func (s *EngramConsolidationService) MergeEngrams(
    ctx context.Context,
    req MergeRequest,  // source_ids, target_id, merge_strategy
) (*MergedEngram, error) {
    // Merge metadata (combine keywords, tags)
    // Merge links (deduplicate)
    // Archive sources
    // Generate summary of what was merged
    // Record history
}
```

**Background job:**
```go
// Daily job: update freshness scores
func UpdateEngramFreshnessScores(ctx context.Context) error {
    lambda := 0.01
    // For each engram: freshness = exp(-lambda * daysSinceAccess)
    // Update freshness_score
}
```

### MCP Tools: New
```
Tool: engram.suggest_consolidations
Scope: read
Parameters:
  - project_id (optional, defaults to actor's default)
  - filter_type (optional: "exact_duplicate"|"theme_duplicate"|all)
  - min_confidence (optional, default 0.7)
Returns:
  - suggestions: ConsolidationCandidate[]

Tool: engram.consolidate
Scope: write
Parameters:
  - suggestion_id
  - merge_strategy ("keep_newest"|"keep_largest"|"ai_generated_summary")
Returns:
  - merged_engram_id
  - merge_summary
  - archived_engram_ids
```

### Implementation Complexity
- **Schema:** High (multiple new tables + indexes)
- **Service:** High (duplicate detection, merge logic, summary generation)
- **Background Jobs:** Medium (daily freshness updates)
- **MCP:** Medium (new tools)

### Success Metrics
- Consolidation suggestions > 50 per 1000 engrams (healthy ratio)
- Top 20% of engrams retain freshness > 0.8 (actively used)
- User consolidation acceptance rate > 70%

---

## Improvement 4: Contradiction Detection & Warning

### Current State
- No conflict awareness between memories
- LLM can retrieve contradictory engrams in same query
- Hallucination risk: "Did I learn X or ¬X?"

### Target State
**System detects contradictions between engrams and alerts LLM:**

```
Before saving new engram:
  check_contradictions(text) → [
    {
      contradicting_engram_id: "...",
      conflicting_statement: "...",
      severity: "critical|medium|low",
      suggested_action: "review|merge|archive"
    }
  ]

During retrieval:
  If top-K results include contradictions:
    Include "⚠️ Contradiction detected between engram-X and engram-Y"
```

### Schema Changes
```sql
-- Contradiction tracking on links
ALTER TABLE engram_links ADD COLUMN conflict_detected BOOLEAN DEFAULT FALSE;
ALTER TABLE engram_links ADD COLUMN conflict_severity TEXT CHECK (conflict_severity IN ('low', 'medium', 'critical'));
ALTER TABLE engram_links ADD COLUMN conflict_evidence TEXT;

-- Explicit contradiction records
CREATE TABLE contradiction_alerts (
  alert_id UUID PRIMARY KEY,
  engram_id UUID NOT NULL REFERENCES engrams(engram_id) ON DELETE CASCADE,
  contradicting_engram_id UUID NOT NULL REFERENCES engrams(engram_id) ON DELETE CASCADE,
  new_info_text TEXT NOT NULL,  -- What did LLM try to save?
  contradicting_statement TEXT NOT NULL,  -- What does existing memory say?
  confidence_of_conflict FLOAT NOT NULL CHECK (confidence_of_conflict >= 0 AND confidence_of_conflict <= 1),
  severity TEXT NOT NULL DEFAULT 'medium' CHECK (severity IN ('low', 'medium', 'critical')),
  detected_by_model TEXT,  -- Which embedding model detected it?
  created_at TIMESTAMPTZ DEFAULT now(),
  resolved_at TIMESTAMPTZ,
  resolution_action TEXT CHECK (resolution_action IN ('merged', 'archived', 'accepted', 'marked_as_outdated'))
);

CREATE INDEX contradiction_alerts_engram_idx ON contradiction_alerts (engram_id);
CREATE INDEX contradiction_alerts_unresolved_idx ON contradiction_alerts (resolved_at) WHERE resolved_at IS NULL;
```

### Service Layer Changes
**New service: `ContradictionDetectionService`**
```go
type ContradictionAnalysis struct {
    EngramID                uuid.UUID
    ContradictingEngramID   uuid.UUID
    ConflictingStatement    string
    Severity                string  // low|medium|critical
    ConfidenceOfConflict    float64
    SuggestedAction         string
}

func (s *ContradictionDetectionService) CheckNewEngram(
    ctx context.Context,
    req CheckContradictionRequest,
) ([]ContradictionAnalysis, error) {
    // Embed the new text
    // Search for similar engrams
    // For each candidate:
    //   - Check if it contradicts
    //   - Compute confidence
    //   - Determine severity (new_fact vs revisited_assumption vs critical_safety)
    // Return ranked by severity
}

func (s *ContradictionDetectionService) ResolveContradiction(
    ctx context.Context,
    alertID uuid.UUID,
    action string,
) error {
    // action: "merged"|"archived"|"accepted"|"marked_as_outdated"
}
```

### MCP Tools: New
```
Tool: engram.check_contradictions
Scope: read
Parameters:
  - query_text (required)
  - project_id (optional)
  - min_severity (optional, default "low")
Returns:
  - contradictions: ContradictionAnalysis[]
  - recommendation: "proceed_with_caution"|"conflict_found"|"safe_to_save"

Tool: engram.resolve_contradiction
Scope: write
Parameters:
  - alert_id
  - resolution_action ("merged"|"archived"|"accepted"|"marked_as_outdated")
Returns:
  - alert_updated: true
```

### Implementation Complexity
- **Schema:** Medium (new table + indexes on links)
- **Service:** High (contradiction detection via semantic analysis)
- **MCP:** Low (new tools)
- **Dependencies:** Requires embedding infrastructure

### Success Metrics
- Contradiction detection precision > 90%
- Contradiction recall > 70% (find most real contradictions)
- False positive rate < 5%

---

## Improvement 5: Temporal Memory Queries

### Current State
```
engram.query({
  q: "migration checkpoints",
  top_k: 5
})
// Returns: top-5 semantic matches, no time control
```

### Target State
```
engram.query({
  q: "migration checkpoints",
  top_k: 5,
  time_range: "recent",  // last_7d|last_30d|all
  min_access_count: 3,   // popular
  relation_type: "derived_from",  // what built on what?
  exclude_contradicted: true
})
```

### Schema Changes
**No new schema — only query enhancements**

**Add to engrams table (if not already present):**
```sql
-- Already have: access_count, last_accessed_at, freshness_score
-- Already have: created_at, updated_at
-- Ensure we have efficient indexes for temporal queries
CREATE INDEX IF NOT EXISTS engrams_time_access_count_idx 
  ON engrams (created_at DESC, access_count DESC);
```

### Service Layer Changes
**Enhance `EngramQueryService`:**
```go
type EngramQueryRequest struct {
    Q                    string
    TopK                 int
    ProjectID            string
    
    // NEW: Temporal filters
    TimeRange            string  // "recent"|"last_7d"|"last_30d"|"all"
    MinAccessCount       int
    MaxAge               int     // days
    OnlyHighEngagement   bool    // access_count > threshold
    ExcludeContradicted  bool    // filter out alert_exists
    
    // NEW: Relationship filters
    RelationType         string  // "supports"|"depends_on"|"related_to"|"derived_from"
    TraceDepth          int     // Follow links N hops (0=no links)
}

func (s *EngramQueryService) QueryWithFilters(
    ctx context.Context,
    req EngramQueryRequest,
) ([]Engram, error) {
    // Embed query
    // Search with time/engagement filters
    // If TraceDepth > 0: follow links to adjacent engrams
    // Return ranked by composite score
}
```

### MCP Tool: Update
**Enhance `engram.query` endpoint:**
```json
{
  "jsonrpc": "2.0",
  "id": "query-temporal",
  "method": "tools/call",
  "params": {
    "name": "engram.query",
    "arguments": {
      "q": "what did we learn about migration?",
      "top_k": 5,
      "time_range": "last_30d",
      "min_access_count": 2,
      "exclude_contradicted": true,
      "trace_depth": 1
    }
  }
}
```

### Implementation Complexity
- **Schema:** None (use existing columns)
- **Service:** Medium (query builder enhancements)
- **MCP:** Low (existing tool parameter expansion)

### Success Metrics
- Query execution time < 500ms even with filters
- Time-filtered queries return 30%+ more relevant results
- Users find "recent progress" and "foundational" memories easier to locate

---

## Improvement 6: Cost-Aware Context Selection

### Current State
```
engram.query(...) → returns top-K engrams
// LLM must manually decide which to include in context
// Can easily exceed token budget
```

### Target State
```
engram.assemble_context({
  query: "what did we learn about X?",
  token_budget: 2000,
  prefer_recent: true,
  include_contradictions: false
})
// Returns: curated engrams + summary + actual token usage
```

### Schema Changes
```sql
-- Track token estimates
ALTER TABLE engrams ADD COLUMN estimated_tokens INT DEFAULT 0;

-- Trigger to auto-estimate on insert (based on text length)
CREATE OR REPLACE FUNCTION estimate_engram_tokens()
RETURNS TRIGGER AS $$
BEGIN
  NEW.estimated_tokens := greatest(50, (length(NEW.engram_markdown) + length(NEW.abstract)) / 4);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER engram_tokens_estimate_trigger
BEFORE INSERT OR UPDATE ON engrams
FOR EACH ROW
EXECUTE FUNCTION estimate_engram_tokens();

-- Track context assembly history
CREATE TABLE context_assembly_history (
  history_id UUID PRIMARY KEY,
  session_id UUID,
  query TEXT,
  token_budget INT,
  selected_engram_ids UUID[] NOT NULL,
  actual_tokens_used INT,
  coverage_score FLOAT,  -- How well did selection cover query?
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX context_assembly_history_session_idx 
  ON context_assembly_history (session_id, created_at DESC);
```

### Service Layer Changes
**New service: `ContextAssemblyService`**
```go
type ContextAssemblyRequest struct {
    Query                   string
    TokenBudget             int    // max tokens available
    PreferRecent            bool
    IncludeContradictions   bool
    MaxEngrams              int    // cap on count
}

type ContextAssemblyResult struct {
    SelectedEngrams        []EngramForContext  // with token counts
    Summary                string              // AI-generated brief of context
    TokensUsed             int
    CoverageScore          float64             // 0-1: how well does it cover query?
    Recommendation         string              // "sufficient"|"partial"|"insufficient"
}

func (s *ContextAssemblyService) AssembleContext(
    ctx context.Context,
    req ContextAssemblyRequest,
) (*ContextAssemblyResult, error) {
    // Semantic search for relevant engrams
    // Estimate tokens for each
    // Greedily select by relevance-per-token until budget exhausted
    // Generate summary of assembled context
    // Return with coverage metrics
}
```

### MCP Tool: New
```
Tool: engram.assemble_context
Scope: read
Parameters:
  - query (required)
  - token_budget (required)
  - prefer_recent (optional, default true)
  - include_contradictions (optional, default false)
  - max_engrams (optional, default 20)
Returns:
  - selected_engrams: {
      engram_id,
      title,
      abstract,
      estimated_tokens,
      relevance_score
    }[]
  - summary: string
  - tokens_used: int
  - coverage_score: float
  - recommendation: string
```

### Implementation Complexity
- **Schema:** Medium (token tracking + history)
- **Service:** High (token estimation + greedy selection algorithm)
- **MCP:** Low (new tool)

### Success Metrics
- Context assembly respects token budget within 5% error
- Coverage score > 0.8 for most queries
- LLMs report 40% faster context assembly with less token waste

---

## Improvement 7: Autonomous Memory Insights & Suggestions

### Current State
- LLM manually decides what to save
- No system guidance on memorization priorities
- Misses opportunities to consolidate/link insights

### Target State
**System actively suggests what to remember:**

```
session.get_memory_suggestions(session_id) → {
  auto_save_candidates: [
    {
      conversation_excerpt: "...",
      why_important: "Novel insight about X",
      suggested_title: "Discovery: ...",
      suggested_tags: ["insights", "pattern"],
      confidence_score: 0.92
    }
  ],
  consolidation_opportunities: [
    {
      engram_ids: [...],
      reason: "These 3 memories describe same pattern",
      suggested_merge: true
    }
  ],
  contradictions_found: [
    {
      new_statement: "...",
      contradicts: "engram-X",
      severity: "medium"
    }
  ],
  link_opportunities: [
    {
      source_engram_id: "...",
      target_statement: "...",
      relation_type: "derived_from",
      confidence: 0.8
    }
  ]
}
```

### Schema Changes
```sql
CREATE TABLE memory_curation_suggestions (
  suggestion_id UUID PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(project_id),
  session_id UUID NOT NULL REFERENCES chat_sessions(session_id) ON DELETE CASCADE,
  suggestion_type TEXT NOT NULL CHECK (suggestion_type IN ('auto_save', 'consolidate', 'contradiction', 'link')),
  
  -- For auto_save
  conversation_excerpt TEXT,
  suggested_engram_title TEXT,
  suggested_engram_tags TEXT[],
  suggested_engram_abstract TEXT,
  
  -- For consolidate
  candidate_engram_ids UUID[],
  consolidation_reason TEXT,
  
  -- For contradiction
  contradicting_statement TEXT,
  conflicting_source_engram_id UUID,
  
  -- For link
  source_engram_id UUID,
  target_statement TEXT,
  suggested_relation_type TEXT,
  
  why_important TEXT NOT NULL,
  confidence_score FLOAT NOT NULL CHECK (confidence_score >= 0 AND confidence_score <= 1),
  status TEXT DEFAULT 'suggested' CHECK (status IN ('suggested', 'acted_on', 'dismissed')),
  acted_on_at TIMESTAMPTZ,
  acted_on_by UUID REFERENCES users(user_id),
  suggested_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX memory_curation_suggestions_session_idx 
  ON memory_curation_suggestions (session_id, suggestion_type);

CREATE INDEX memory_curation_suggestions_project_status_idx 
  ON memory_curation_suggestions (project_id, status, suggested_at DESC);
```

### Service Layer Changes
**New service: `MemoryCurationService`**
```go
type MemoryCurationSuggestion struct {
    Type            string  // auto_save|consolidate|contradiction|link
    WhyImportant    string
    Confidence      float64
    
    // Variant data per type
    AutoSave        *AutoSaveSuggestion
    Consolidate     *ConsolidateSuggestion
    Contradiction   *ContradictionSuggestion
    Link            *LinkSuggestion
}

func (s *MemoryCurationService) SuggestionsForSession(
    ctx context.Context,
    sessionID uuid.UUID,
) ([]MemoryCurationSuggestion, error) {
    // Fetch session messages
    // Detect novel insights (not in existing engrams)
    // Spot emerging patterns
    // Check for contradictions
    // Identify link opportunities
    // Rank by confidence
    // Return top-N suggestions
}

func (s *MemoryCurationService) ActOnSuggestion(
    ctx context.Context,
    suggestionID uuid.UUID,
    action string,  // "create_engram"|"consolidate"|"link"|"dismiss"
) error {
    // Implement the suggestion
    // Mark as acted_on
}
```

**Analysis sub-services:**
```go
type NoveltyDetector interface {
    IsNovel(ctx context.Context, statement string, projectID string) (score float64, reason string)
}

type PatternDetector interface {
    DetectPatterns(ctx context.Context, sessionMessages []string) []Pattern
}

type LinkSuggester interface {
    SuggestLinks(ctx context.Context, enggramID uuid.UUID) []LinkOpportunity
}
```

### MCP Tools: New
```
Tool: engram.get_memory_suggestions
Scope: read
Parameters:
  - session_id (required)
  - filter_types (optional: ["auto_save", "consolidate", "contradiction", "link"])
  - min_confidence (optional, default 0.7)
Returns:
  - suggestions: MemoryCurationSuggestion[]
  - summary: string (e.g., "5 saveable insights, 2 consolidations possible")

Tool: engram.act_on_suggestion
Scope: write
Parameters:
  - suggestion_id
  - action ("create_engram"|"consolidate"|"link"|"dismiss")
  - params (varies by action)
Returns:
  - success: true
  - created_resource_id: (if action creates engram/link)
```

### Implementation Complexity
- **Schema:** Medium (new suggestions table + indexes)
- **Service:** Very High (novelty + pattern + link detection)
- **Analysis Logic:** Very High (need heuristics + ML-ready infrastructure)
- **MCP:** Medium (new tools)
- **Dependencies:** Embedding infrastructure, language models for explanation

### Success Metrics
- 70%+ of auto_save suggestions actually acted upon
- Consolidation suggestions > 80% correct (user confirmation rate)
- Pattern detection catches emerging themes > 60% of the time
- User time spent curating memories decreases 40%

---

## Implementation Roadmap

### Phase 1: Foundation (6-8 weeks)
**Goal:** Core infrastructure for engagement & time-decay

**1.1 Schema & Tracking** (Weeks 1-2)
- [ ] Add engagement columns to engrams (access_count, last_accessed_at)
- [ ] Create engram_access_events table
- [ ] Create engram_feedback table
- [ ] Create engram_consolidation_suggestions table
- [ ] Migration scripts + rollback procedures
- [ ] Tests: migration success, data integrity

**1.2 Engagement Feedback Service** (Weeks 2-3)
- [ ] Implement EngramFeedbackService (record feedback)
- [ ] Update engrams table with aggregate stats (useful_count, contradiction_count)
- [ ] Implement triggers for stat updates
- [ ] MCP tool: engram.provide_feedback
- [ ] Tests: 100+ test cases covering feedback paths

**1.3 Relevance Scoring** (Weeks 3-4)
- [ ] Implement four-factor ranking algorithm
- [ ] Add freshness_score + time-decay computation
- [ ] Update engram.query to use composite ranking
- [ ] Benchmarks: query latency < 500ms
- [ ] Tests: scoring correctness, ranking stability

**1.4 Initial Consolidation** (Weeks 4-6)
- [ ] Implement duplicate detection (exact hash + theme)
- [ ] Implement ConsolidationService.FindCandidates()
- [ ] Implement merge logic
- [ ] MCP tool: engram.suggest_consolidations
- [ ] Tests: duplicate detection recall > 90%, precision > 85%

**Phase 1 Deliverable:** Engrams self-tune based on usage; duplicates automatically detected

---

### Phase 2: LLM Safety & Intelligence (8-10 weeks)
**Goal:** Contradiction detection, temporal queries, cost-aware assembly

**2.1 Contradiction Detection** (Weeks 1-3)
- [ ] Implement ContradictionDetectionService
- [ ] Create contradiction_alerts table
- [ ] Add conflict metadata to engram_links
- [ ] MCP tools: engram.check_contradictions, engram.resolve_contradiction
- [ ] Integration with chat.save_as_engram (warn before save)
- [ ] Tests: contradiction detection precision > 90%

**2.2 Temporal Queries** (Weeks 3-4)
- [ ] Extend EngramQueryService with time/engagement filters
- [ ] Implement graph traversal (trace_depth)
- [ ] Update engram.query MCP tool
- [ ] Benchmarks: filtered queries < 300ms
- [ ] Tests: filter correctness, link traversal

**2.3 Cost-Aware Context Assembly** (Weeks 5-7)
- [ ] Implement ContextAssemblyService
- [ ] Token estimation logic + triggers
- [ ] Greedy selection algorithm
- [ ] MCP tool: engram.assemble_context
- [ ] Tests: token budget adherence < 5% error

**2.4 Integration & Hardening** (Weeks 7-10)
- [ ] Integrate all services into chat workflow
- [ ] Update chat.send_message to return retrieved engrams + contradiction warnings
- [ ] Acceptance tests: full workflows
- [ ] Performance profiling & optimization

**Phase 2 Deliverable:** LLMs get safe, cost-aware context with contradiction warnings

---

### Phase 3: Autonomous Intelligence (10-12 weeks)
**Goal:** System suggests what to remember

**3.1 Curation Infrastructure** (Weeks 1-3)
- [ ] Create memory_curation_suggestions table
- [ ] Build NoveltyDetector (what's new vs existing?)
- [ ] Build PatternDetector (emerging themes)
- [ ] Build LinkSuggester (connection opportunities)
- [ ] Tests: unit tests per detector

**3.2 Orchestration Service** (Weeks 3-5)
- [ ] Implement MemoryCurationService
- [ ] SuggestionsForSession(): combine all detectors
- [ ] ActOnSuggestion(): implement suggestions
- [ ] MCP tools: engram.get_memory_suggestions, engram.act_on_suggestion
- [ ] Tests: suggestion pipeline integration

**3.3 Quality Evaluation** (Weeks 5-8)
- [ ] Evaluate suggestion quality (precision/recall)
- [ ] Tune detector thresholds
- [ ] Add user feedback loop (was suggestion helpful?)
- [ ] Acceptance tests: user experience

**3.4 Production Readiness** (Weeks 8-12)
- [ ] Performance benchmarks: suggestion latency < 2s
- [ ] Background job: daily freshness updates
- [ ] Monitoring & alerting
- [ ] Documentation + runbooks
- [ ] Load testing

**Phase 3 Deliverable:** System actively guides LLMs on what to remember

---

## Resource Requirements

### Engineering
- 1 backend engineer (primary, all phases)
- 1 data engineer (schema, indexes, migrations)
- 1 ML engineer (novelty detection, pattern ML)
- QA/testing: shared across team

### Data Storage
- PostgreSQL: ~200MB per 100k engrams (with new tables/indexes)
- Estimated growth: 5-10GB/year for typical deployment

### Compute
- Embedding service: existing (reuse for contradiction detection)
- Background jobs: < 1 minute/day per project
- MCP tool latency: maintain SLA < 1s

### Dependencies
- Existing: pgvector, embedding model
- New: (none required; ML can be local or external)

---

## Success Metrics & KPIs

### Adoption
- 60%+ of chat sessions trigger ≥1 memory suggestion
- 70%+ engagement feedback recorded
- 80%+ user adoption of auto-recommendations

### Quality
- Contradiction detection: precision > 90%, recall > 70%
- Consolidation acceptance: > 70% user confirmation
- Curation suggestions helpful: > 75% positive feedback

### Performance
- engram.query (all variants): p95 < 500ms
- engram.assemble_context: p95 < 2s
- engram.get_memory_suggestions: p95 < 3s
- Background jobs: < 2 minutes/day

### Memory Health
- Duplicate ratio < 5% (post-consolidation)
- Median freshness score ≥ 0.7
- Contradiction alerts/1000 engrams < 2 (low false positive)

### LLM Experience
- Token waste in context reduction: 30%
- "Hallucination prevented" incidents: baseline → track
- Memory retrieval relevance: precision increase 20%

---

## Testing Strategy

### Unit Tests
- Each service: >80% coverage
- Scoring algorithm: correctness proofs
- Token estimation: within 5% of actual

### Integration Tests
- Engagement pipeline: feedback → score update → query ranking
- Contradiction workflow: detection → resolution → link updates
- Consolidation pipeline: suggestion → merge → history
- Curation pipeline: detection → suggestion → action

### Acceptance Tests
- End-to-end chat session with full memory pipeline
- Contradictions properly detected & surfaced
- Context assembly respects token budgets
- Suggestions useful & acted upon

### Performance Tests
- Query latency at 10M engrams
- Consolidation scan on 1M engrams
- Background job impact on overall system

### User Acceptance Tests (UAT)
- Beta users on Phase 1 → feedback on engagement scoring
- Beta users on Phase 2 → contradiction detection real-world validation
- Beta users on Phase 3 → autonomy/suggestion value

---

## Risk Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Contradiction detection false positives | Medium | High | Start conservative (high threshold), collect feedback, iterate |
| Token estimation accuracy | Low | Medium | Test against real LLM tokens, adjust heuristic |
| Consolidation mistakes | Medium | High | Always suggest, never auto-execute; 2-phase approval |
| Performance regression | Medium | High | Continuous benchmarking, index optimization pre-release |
| User disinteraction with suggestions | Medium | Low | A/B test UI/UX presentation, track acceptance rates |

---

## Rollout Strategy

### Internal Alpha (2 weeks)
- Deploy all phases to staging
- Team uses system for own research
- Collect feedback, fix bugs

### Closed Beta (4 weeks)
- Invite 5-10 power users
- Gather qualitative feedback
- Monitor metrics

### Open Beta (8 weeks)
- Enable by default, observability on
- Monitor rollback readiness
- Iterate based on production data

### General Availability
- All features stable, documented
- GA release with runbooks
- Post-release support plan (30 days)

---

## Documentation Plan

- **Developer Guide:** How to extend detectors/suggesters
- **User Guide:** How to use new tools in practice
- **API Reference:** All new MCP tools
- **Architecture:** Decision records for each phase
- **Runbook:** Operations, troubleshooting, tuning

---

## Success Criteria

✅ **Minimum viable success:**
- Phase 1 complete (engagement + consolidation)
- 50%+ of retrievals have feedback within 30 days
- Contradictions detected with 85%+ precision

✅ **Full success:**
- All 3 phases shipped with metrics on target
- LLM integration improved (stronger context, fewer hallucinations)
- Memory system becomes "trusted" by LLMs (high reliance)

---

**Next Steps:**
1. Review with team
2. Assign Phase 1 lead
3. Create detailed spike tickets for Weeks 1-2
4. Finalize schema review (data engineer)
5. Kickoff Phase 1
