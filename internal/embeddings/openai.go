package embeddings

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"
	"time"
)

const defaultOpenAIEmbeddingProviderIDPrefix = "openai"

// OpenAIEmbeddingProviderInput captures OpenAI provider construction fields.
type OpenAIEmbeddingProviderInput struct {
	APIKey     string
	BaseURL    string
	Model      string
	Timeout    time.Duration
	HTTPClient *http.Client
	ProviderID string
}

// OpenAIEmbeddingProvider calls OpenAI embeddings API.
type OpenAIEmbeddingProvider struct {
	apiKey     string
	endpoint   string
	model      string
	httpClient *http.Client
	providerID string
}

type openAIEmbeddingRequest struct {
	Model      string `json:"model"`
	Input      any    `json:"input"`
	Dimensions int    `json:"dimensions,omitempty"`
}

type openAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewOpenAIEmbeddingProvider(input OpenAIEmbeddingProviderInput) (*OpenAIEmbeddingProvider, error) {
	apiKey := strings.TrimSpace(input.APIKey)
	if apiKey == "" {
		return nil, errors.New("openai api key is required")
	}
	baseURL, err := normalizeOpenAIBaseURL(input.BaseURL)
	if err != nil {
		return nil, err
	}
	model := strings.TrimSpace(input.Model)
	if model == "" {
		return nil, errors.New("openai embedding model is required")
	}
	httpClient := input.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: resolveTimeout(input.Timeout)}
	}
	providerID := strings.TrimSpace(input.ProviderID)
	if providerID == "" {
		providerID = fmt.Sprintf("%s:%s", defaultOpenAIEmbeddingProviderIDPrefix, model)
	}
	return &OpenAIEmbeddingProvider{
		apiKey:     apiKey,
		endpoint:   baseURL,
		model:      model,
		httpClient: httpClient,
		providerID: providerID,
	}, nil
}

func normalizeOpenAIBaseURL(baseURL string) (string, error) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return "", errors.New("openai base url is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("parse openai base url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("openai base url must be absolute")
	}
	parsed.Path = path.Clean(strings.TrimRight(parsed.Path, "/") + "/v1/embeddings")
	if parsed.Path == "." {
		parsed.Path = "/v1/embeddings"
	}
	return parsed.String(), nil
}

func (provider *OpenAIEmbeddingProvider) ProviderID() string {
	return provider.providerID
}

func (provider *OpenAIEmbeddingProvider) Embed(text string, dim int) ([]float64, error) {
	vectors, err := provider.EmbedMany([]string{text}, dim)
	if err != nil {
		return nil, err
	}
	if len(vectors) != 1 {
		return nil, &ProviderError{Message: "openai provider returned empty embedding vector"}
	}
	return vectors[0], nil
}

func (provider *OpenAIEmbeddingProvider) EmbedMany(texts []string, dim int) ([][]float64, error) {
	if len(texts) == 0 {
		return [][]float64{}, nil
	}
	if dim <= 0 {
		return nil, errors.New("dim must be > 0")
	}
	request := openAIEmbeddingRequest{
		Model:      provider.model,
		Input:      texts,
		Dimensions: dim,
	}
	response, err := provider.callEmbeddingsAPI(request)
	if err != nil {
		return nil, err
	}
	if len(response.Data) == 0 {
		return nil, &ProviderError{Message: "openai provider returned no embeddings"}
	}
	ordered := append([]struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	}(nil), response.Data...)
	slices.SortStableFunc(ordered, func(left, right struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	}) int {
		return left.Index - right.Index
	})
	vectors := make([][]float64, 0, len(ordered))
	for _, item := range ordered {
		vectors = append(vectors, append([]float64(nil), item.Embedding...))
	}
	return vectors, nil
}

func (provider *OpenAIEmbeddingProvider) callEmbeddingsAPI(
	payload openAIEmbeddingRequest,
) (openAIEmbeddingResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return openAIEmbeddingResponse{}, err
	}
	request, err := http.NewRequest(http.MethodPost, provider.endpoint, bytes.NewReader(body))
	if err != nil {
		return openAIEmbeddingResponse{}, err
	}
	request.Header.Set("Authorization", "Bearer "+provider.apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := provider.httpClient.Do(request)
	if err != nil {
		return openAIEmbeddingResponse{}, &ProviderError{Message: fmt.Sprintf("openai embeddings request failed: %v", err)}
	}
	defer response.Body.Close()

	responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if readErr != nil {
		return openAIEmbeddingResponse{}, &ProviderError{Message: "failed to read openai embeddings response"}
	}

	parsed := openAIEmbeddingResponse{}
	if len(responseBody) > 0 {
		if unmarshalErr := json.Unmarshal(responseBody, &parsed); unmarshalErr != nil && response.StatusCode < http.StatusBadRequest {
			return openAIEmbeddingResponse{}, &ProviderError{Message: "failed to decode openai embeddings response"}
		}
	}
	if response.StatusCode >= http.StatusBadRequest {
		message := strings.TrimSpace(string(responseBody))
		if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
			message = parsed.Error.Message
		}
		if message == "" {
			message = fmt.Sprintf("openai embeddings request failed with status %d", response.StatusCode)
		}
		return openAIEmbeddingResponse{}, &ProviderError{Message: message}
	}
	return parsed, nil
}
