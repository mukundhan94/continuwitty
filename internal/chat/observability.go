package chat

import (
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

// ProviderFailureSample captures a categorized provider execution failure.
type ProviderFailureSample struct {
	Provider  models.ChatProvider
	Operation string
	ErrorCode string
}

// StreamHealthSample captures stream outcome telemetry.
type StreamHealthSample struct {
	Provider   models.ChatProvider
	Operation  string
	Outcome    string
	ErrorCode  string
	ChunkCount int
	Duration   time.Duration
}

// LifecycleTraceSample captures lifecycle stage telemetry for chat flows.
type LifecycleTraceSample struct {
	TraceID   string
	Operation string
	Stage     string
	Provider  models.ChatProvider
	SessionID uuid.UUID
	MessageID uuid.UUID
	ErrorCode string
	Duration  time.Duration
}

// ObservabilityRecorder captures optional chat observability hooks.
type ObservabilityRecorder interface {
	RecordProviderFailure(sample ProviderFailureSample)
	RecordStreamHealth(sample StreamHealthSample)
	RecordLifecycleTrace(sample LifecycleTraceSample)
}

type noopObservabilityRecorder struct{}

func (noopObservabilityRecorder) RecordProviderFailure(ProviderFailureSample) {}
func (noopObservabilityRecorder) RecordStreamHealth(StreamHealthSample)       {}
func (noopObservabilityRecorder) RecordLifecycleTrace(LifecycleTraceSample)   {}
