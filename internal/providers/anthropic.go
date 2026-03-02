package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"engram/internal/models"
)

const (
	defaultAnthropicBaseURL = "https://api.anthropic.com"
	defaultAnthropicVersion = "2023-06-01"
)

// AnthropicProvider adapts Anthropic Messages API to the provider interface.
type AnthropicProvider struct {
	apiKey           string
	baseURL          string
	anthropicVersion string
	httpDoer         HTTPDoer
}

func NewAnthropicProvider(apiKey string, baseURL string, anthropicVersion string, client HTTPDoer) *AnthropicProvider {
	normalizedBaseURL := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if normalizedBaseURL == "" {
		normalizedBaseURL = defaultAnthropicBaseURL
	}
	normalizedVersion := strings.TrimSpace(anthropicVersion)
	if normalizedVersion == "" {
		normalizedVersion = defaultAnthropicVersion
	}
	httpDoer := client
	if httpDoer == nil {
		httpDoer = http.DefaultClient
	}
	return &AnthropicProvider{
		apiKey:           apiKey,
		baseURL:          normalizedBaseURL,
		anthropicVersion: normalizedVersion,
		httpDoer:         httpDoer,
	}
}

func (provider *AnthropicProvider) Provider() models.ChatProvider {
	return models.ChatProviderAnthropic
}

func (provider *AnthropicProvider) Generate(ctx context.Context, request ProviderGenerateRequest) (ProviderGenerateResult, error) {
	httpRequest, err := provider.newGenerateRequest(ctx, request)
	if err != nil {
		return ProviderGenerateResult{}, err
	}
	response, err := provider.httpDoer.Do(httpRequest)
	if err != nil {
		return ProviderGenerateResult{}, NewProviderAPIError(fmt.Sprintf("Anthropic request failed: %v", err))
	}
	defer response.Body.Close()

	if statusErr := providerStatusCodeError("Anthropic", response.StatusCode); statusErr != nil {
		return ProviderGenerateResult{}, statusErr
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return ProviderGenerateResult{}, NewProviderAPIError("failed to read Anthropic response body")
	}
	parsedBody := map[string]any{}
	if err := json.Unmarshal(bodyBytes, &parsedBody); err != nil {
		return ProviderGenerateResult{}, NewProviderAPIError("failed to parse Anthropic response")
	}

	inputTokens := extractAnthropicTokenUsage(parsedBody, "input_tokens")
	outputTokens := extractAnthropicTokenUsage(parsedBody, "output_tokens")
	return ProviderGenerateResult{
		Provider: models.ChatProviderAnthropic,
		ModelID:  request.ModelID,
		Text:     extractAnthropicResponseText(parsedBody),
		TokenUsage: map[string]int{
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
			"total_tokens":  inputTokens + outputTokens,
		},
	}, nil
}

func (provider *AnthropicProvider) StreamGenerate(ctx context.Context, request ProviderGenerateRequest) (<-chan string, error) {
	result, err := provider.Generate(ctx, request)
	if err != nil {
		return nil, err
	}
	chunks := make(chan string, 1)
	chunks <- result.Text
	close(chunks)
	return chunks, nil
}

func (provider *AnthropicProvider) Healthcheck() (map[string]string, error) {
	if strings.TrimSpace(provider.apiKey) == "" {
		return nil, NewProviderAuthError("ANTHROPIC_API_KEY is not configured")
	}
	return map[string]string{
		"provider": string(models.ChatProviderAnthropic),
		"status":   "configured",
	}, nil
}

func (provider *AnthropicProvider) newGenerateRequest(ctx context.Context, request ProviderGenerateRequest) (*http.Request, error) {
	if strings.TrimSpace(provider.apiKey) == "" {
		return nil, NewProviderAuthError("ANTHROPIC_API_KEY is not configured")
	}
	payload, err := json.Marshal(buildAnthropicPayload(request))
	if err != nil {
		return nil, NewProviderRequestError("failed to encode Anthropic request payload")
	}
	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		provider.baseURL+"/v1/messages",
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, NewProviderRequestError("failed to create Anthropic request")
	}
	httpRequest.Header.Set("x-api-key", provider.apiKey)
	httpRequest.Header.Set("anthropic-version", provider.anthropicVersion)
	httpRequest.Header.Set("Content-Type", "application/json")
	return httpRequest, nil
}

func buildAnthropicPayload(request ProviderGenerateRequest) map[string]any {
	messages := make([]map[string]string, 0, len(request.Messages))
	for _, message := range request.Messages {
		messages = append(messages, map[string]string{
			"role":    message.Role,
			"content": message.Content,
		})
	}
	payload := map[string]any{
		"model":       request.ModelID,
		"messages":    messages,
		"temperature": request.Temperature,
		"max_tokens":  request.MaxTokens,
	}
	if strings.TrimSpace(request.SystemPrompt) != "" {
		payload["system"] = request.SystemPrompt
	}
	return payload
}

func extractAnthropicResponseText(body map[string]any) string {
	content, ok := body["content"].([]any)
	if !ok {
		return ""
	}
	parts := make([]string, 0, len(content))
	for _, item := range content {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		itemType, _ := itemMap["type"].(string)
		if itemType != "text" {
			continue
		}
		textValue, _ := itemMap["text"].(string)
		if textValue == "" {
			continue
		}
		parts = append(parts, textValue)
	}
	return strings.Join(parts, "")
}

func extractAnthropicTokenUsage(body map[string]any, key string) int {
	usage, ok := body["usage"].(map[string]any)
	if !ok {
		return 0
	}
	return intFromDynamicValue(usage[key])
}
