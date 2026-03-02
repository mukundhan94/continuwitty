package providers

import (
	"context"

	"engram/internal/models"
)

// ProviderMessage is a normalized chat message for provider adapters.
type ProviderMessage struct {
	Role    string
	Content string
}

// ProviderGenerateRequest captures provider generation inputs.
type ProviderGenerateRequest struct {
	ModelID      string
	Messages     []ProviderMessage
	SystemPrompt string
	Temperature  float64
	MaxTokens    int
}

// ProviderGenerateResult captures normalized provider generation outputs.
type ProviderGenerateResult struct {
	Provider   models.ChatProvider
	ModelID    string
	Text       string
	TokenUsage map[string]int
}

// ChatProviderAdapter defines the provider contract used by chat services.
type ChatProviderAdapter interface {
	Provider() models.ChatProvider
	Generate(ctx context.Context, request ProviderGenerateRequest) (ProviderGenerateResult, error)
	StreamGenerate(ctx context.Context, request ProviderGenerateRequest) (<-chan string, error)
	Healthcheck() (map[string]string, error)
}
