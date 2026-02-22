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

const defaultOpenAIBaseURL = "https://api.openai.com"

// HTTPDoer abstracts outbound HTTP requests for provider adapters.
type HTTPDoer interface {
	Do(request *http.Request) (*http.Response, error)
}

// OpenAIProvider adapts OpenAI chat-completion APIs to the provider interface.
type OpenAIProvider struct {
	apiKey   string
	baseURL  string
	httpDoer HTTPDoer
}

func NewOpenAIProvider(apiKey string, baseURL string, client HTTPDoer) *OpenAIProvider {
	normalizedBaseURL := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if normalizedBaseURL == "" {
		normalizedBaseURL = defaultOpenAIBaseURL
	}
	httpDoer := client
	if httpDoer == nil {
		httpDoer = http.DefaultClient
	}
	return &OpenAIProvider{
		apiKey:   apiKey,
		baseURL:  normalizedBaseURL,
		httpDoer: httpDoer,
	}
}

func (provider *OpenAIProvider) Provider() models.ChatProvider {
	return models.ChatProviderOpenAI
}

func (provider *OpenAIProvider) Generate(ctx context.Context, request ProviderGenerateRequest) (ProviderGenerateResult, error) {
	httpRequest, err := provider.newGenerateRequest(ctx, request)
	if err != nil {
		return ProviderGenerateResult{}, err
	}
	response, err := provider.httpDoer.Do(httpRequest)
	if err != nil {
		return ProviderGenerateResult{}, NewProviderAPIError(fmt.Sprintf("OpenAI request failed: %v", err))
	}
	defer response.Body.Close()

	if statusErr := providerStatusCodeError("OpenAI", response.StatusCode); statusErr != nil {
		return ProviderGenerateResult{}, statusErr
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return ProviderGenerateResult{}, NewProviderAPIError("failed to read OpenAI response body")
	}
	parsedBody := map[string]any{}
	if err := json.Unmarshal(bodyBytes, &parsedBody); err != nil {
		return ProviderGenerateResult{}, NewProviderAPIError("failed to parse OpenAI response")
	}

	return ProviderGenerateResult{
		Provider: models.ChatProviderOpenAI,
		ModelID:  request.ModelID,
		Text:     extractOpenAIResponseText(parsedBody),
		TokenUsage: map[string]int{
			"input_tokens":  extractOpenAITokenUsage(parsedBody, "prompt_tokens"),
			"output_tokens": extractOpenAITokenUsage(parsedBody, "completion_tokens"),
			"total_tokens":  extractOpenAITokenUsage(parsedBody, "total_tokens"),
		},
	}, nil
}

func (provider *OpenAIProvider) StreamGenerate(ctx context.Context, request ProviderGenerateRequest) (<-chan string, error) {
	result, err := provider.Generate(ctx, request)
	if err != nil {
		return nil, err
	}
	chunks := make(chan string, 1)
	chunks <- result.Text
	close(chunks)
	return chunks, nil
}

func (provider *OpenAIProvider) Healthcheck() (map[string]string, error) {
	if strings.TrimSpace(provider.apiKey) == "" {
		return nil, NewProviderAuthError("OPENAI_API_KEY is not configured")
	}
	return map[string]string{
		"provider": string(models.ChatProviderOpenAI),
		"status":   "configured",
	}, nil
}

func (provider *OpenAIProvider) newGenerateRequest(ctx context.Context, request ProviderGenerateRequest) (*http.Request, error) {
	if strings.TrimSpace(provider.apiKey) == "" {
		return nil, NewProviderAuthError("OPENAI_API_KEY is not configured")
	}
	payload, err := json.Marshal(buildOpenAIPayload(request))
	if err != nil {
		return nil, NewProviderRequestError("failed to encode OpenAI request payload")
	}
	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		provider.baseURL+"/v1/chat/completions",
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, NewProviderRequestError("failed to create OpenAI request")
	}
	httpRequest.Header.Set("Authorization", "Bearer "+provider.apiKey)
	httpRequest.Header.Set("Content-Type", "application/json")
	return httpRequest, nil
}

func buildOpenAIPayload(request ProviderGenerateRequest) map[string]any {
	messages := make([]map[string]string, 0, len(request.Messages)+1)
	if strings.TrimSpace(request.SystemPrompt) != "" {
		messages = append(messages, map[string]string{
			"role":    "system",
			"content": request.SystemPrompt,
		})
	}
	for _, message := range request.Messages {
		messages = append(messages, map[string]string{
			"role":    message.Role,
			"content": message.Content,
		})
	}
	return map[string]any{
		"model":       request.ModelID,
		"messages":    messages,
		"temperature": request.Temperature,
		"max_tokens":  request.MaxTokens,
	}
}

func extractOpenAIResponseText(body map[string]any) string {
	choices, ok := body["choices"].([]any)
	if !ok || len(choices) == 0 {
		return ""
	}
	firstChoice, ok := choices[0].(map[string]any)
	if !ok {
		return ""
	}
	message, ok := firstChoice["message"].(map[string]any)
	if !ok {
		return ""
	}
	content := message["content"]
	switch typed := content.(type) {
	case string:
		return typed
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			itemMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			textValue, ok := itemMap["text"].(string)
			if !ok {
				continue
			}
			parts = append(parts, textValue)
		}
		return strings.Join(parts, "")
	default:
		return ""
	}
}

func extractOpenAITokenUsage(body map[string]any, key string) int {
	usage, ok := body["usage"].(map[string]any)
	if !ok {
		return 0
	}
	return intFromDynamicValue(usage[key])
}

func intFromDynamicValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		return int(parsed)
	default:
		return 0
	}
}
