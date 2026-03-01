package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"engram/internal/models"
)

var (
	awsAuthErrorCodes = map[string]struct{}{
		"UnrecognizedClientException":  {},
		"InvalidSignatureException":    {},
		"ExpiredTokenException":        {},
		"IncompleteSignatureException": {},
	}
	awsRateLimitErrorCodes = map[string]struct{}{
		"ThrottlingException":      {},
		"TooManyRequestsException": {},
	}
	awsRequestErrorCodes = map[string]struct{}{
		"ValidationException": {},
	}
)

// ErrNoAWSCredentials is used by runtime clients to signal missing credentials.
var ErrNoAWSCredentials = fmt.Errorf("aws credentials not found")

// ErrPartialAWSCredentials is used by runtime clients to signal incomplete credentials.
var ErrPartialAWSCredentials = fmt.Errorf("aws credentials are incomplete")

// AWSRuntimeCredentials captures Bedrock runtime credential configuration.
type AWSRuntimeCredentials struct {
	RegionName      string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

// BedrockInvokeInput captures normalized Bedrock invocation input.
type BedrockInvokeInput struct {
	ModelID     string
	ContentType string
	Accept      string
	Body        []byte
}

// BedrockInvokeOutput captures normalized Bedrock invocation output.
type BedrockInvokeOutput struct {
	Body []byte
}

// BedrockRuntimeClient abstracts Bedrock runtime invocations.
type BedrockRuntimeClient interface {
	InvokeModel(ctx context.Context, input BedrockInvokeInput) (BedrockInvokeOutput, error)
}

// BedrockErrorCodeCarrier allows runtime errors to expose AWS-style error code/message.
type BedrockErrorCodeCarrier interface {
	error
	ErrorCode() string
	ErrorMessage() string
}

var newBedrockRuntimeClient = func(credentials AWSRuntimeCredentials) (BedrockRuntimeClient, error) {
	_ = credentials
	return nil, NewProviderAPIError("Bedrock runtime client is not configured in this server build")
}

// BedrockProvider adapts Bedrock model invocation to the provider interface.
type BedrockProvider struct {
	credentials   AWSRuntimeCredentials
	runtimeClient BedrockRuntimeClient
}

func NewBedrockProvider(credentials AWSRuntimeCredentials, client BedrockRuntimeClient) *BedrockProvider {
	return &BedrockProvider{
		credentials:   credentials,
		runtimeClient: client,
	}
}

func (provider *BedrockProvider) Provider() models.ChatProvider {
	return models.ChatProviderBedrock
}

func (provider *BedrockProvider) Generate(ctx context.Context, request ProviderGenerateRequest) (ProviderGenerateResult, error) {
	runtimeClient, err := provider.runtime()
	if err != nil {
		return ProviderGenerateResult{}, err
	}
	requestBody, err := json.Marshal(buildBedrockPayload(request))
	if err != nil {
		return ProviderGenerateResult{}, NewProviderRequestError("failed to encode Bedrock request payload")
	}
	output, err := runtimeClient.InvokeModel(
		ctx,
		BedrockInvokeInput{
			ModelID:     request.ModelID,
			ContentType: "application/json",
			Accept:      "application/json",
			Body:        requestBody,
		},
	)
	if err != nil {
		return ProviderGenerateResult{}, provider.mapInvocationError(err)
	}
	parsedBody := map[string]any{}
	if err := json.Unmarshal(output.Body, &parsedBody); err != nil {
		return ProviderGenerateResult{}, NewProviderAPIError("Failed to parse Bedrock response")
	}

	inputTokens := extractBedrockTokenUsage(parsedBody, "input_tokens")
	outputTokens := extractBedrockTokenUsage(parsedBody, "output_tokens")
	return ProviderGenerateResult{
		Provider: models.ChatProviderBedrock,
		ModelID:  request.ModelID,
		Text:     extractBedrockResponseText(parsedBody),
		TokenUsage: map[string]int{
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
			"total_tokens":  inputTokens + outputTokens,
		},
	}, nil
}

func (provider *BedrockProvider) StreamGenerate(ctx context.Context, request ProviderGenerateRequest) (<-chan string, error) {
	result, err := provider.Generate(ctx, request)
	if err != nil {
		return nil, err
	}
	chunks := make(chan string, 1)
	chunks <- result.Text
	close(chunks)
	return chunks, nil
}

func (provider *BedrockProvider) Healthcheck() (map[string]string, error) {
	if strings.TrimSpace(provider.credentials.RegionName) == "" {
		return nil, NewProviderAuthError("AWS_REGION is not configured")
	}
	return map[string]string{
		"provider": string(models.ChatProviderBedrock),
		"status":   "configured",
	}, nil
}

func (provider *BedrockProvider) runtime() (BedrockRuntimeClient, error) {
	if provider.runtimeClient != nil {
		return provider.runtimeClient, nil
	}
	runtimeClient, err := newBedrockRuntimeClient(provider.credentials)
	if err != nil {
		return nil, err
	}
	provider.runtimeClient = runtimeClient
	return provider.runtimeClient, nil
}

func (provider *BedrockProvider) mapInvocationError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrNoAWSCredentials):
		return NewProviderAuthError("Bedrock credentials not found in environment or AWS profile")
	case errors.Is(err, ErrPartialAWSCredentials):
		return NewProviderAuthError("Bedrock credentials are incomplete")
	}
	var codedError BedrockErrorCodeCarrier
	if !errors.As(err, &codedError) {
		return NewProviderAPIError(fmt.Sprintf("Bedrock invocation failed: %v", err))
	}
	errorCode := strings.TrimSpace(codedError.ErrorCode())
	errorMessage := strings.TrimSpace(codedError.ErrorMessage())
	if errorMessage == "" {
		errorMessage = codedError.Error()
	}
	switch {
	case inCodeSet(awsRateLimitErrorCodes, errorCode):
		return NewProviderRateLimitError(fmt.Sprintf("Bedrock %s: %s", errorCode, errorMessage))
	case inCodeSet(awsRequestErrorCodes, errorCode):
		return NewProviderRequestError(fmt.Sprintf("Bedrock %s: %s", errorCode, errorMessage))
	case inCodeSet(awsAuthErrorCodes, errorCode):
		return NewProviderAuthError(fmt.Sprintf("Bedrock %s: %s", errorCode, errorMessage))
	default:
		return NewProviderAPIError(fmt.Sprintf("Bedrock %s: %s", errorCode, errorMessage))
	}
}

func buildBedrockPayload(request ProviderGenerateRequest) map[string]any {
	messages := make([]map[string]any, 0, len(request.Messages))
	for _, message := range request.Messages {
		messages = append(messages, map[string]any{
			"role": message.Role,
			"content": []map[string]string{
				{
					"type": "text",
					"text": message.Content,
				},
			},
		})
	}
	return map[string]any{
		"anthropic_version": "bedrock-2023-05-31",
		"max_tokens":        request.MaxTokens,
		"temperature":       request.Temperature,
		"system":            request.SystemPrompt,
		"messages":          messages,
	}
}

func extractBedrockResponseText(body map[string]any) string {
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

func extractBedrockTokenUsage(body map[string]any, key string) int {
	usage, ok := body["usage"].(map[string]any)
	if !ok {
		return 0
	}
	return intFromDynamicValue(usage[key])
}

func inCodeSet(set map[string]struct{}, value string) bool {
	_, ok := set[value]
	return ok
}
