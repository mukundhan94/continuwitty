package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"engram/internal/models"
)

type dummyHTTPDoer struct {
	response *http.Response
	err      error
	requests []*http.Request
}

func (doer *dummyHTTPDoer) Do(request *http.Request) (*http.Response, error) {
	doer.requests = append(doer.requests, request)
	if doer.err != nil {
		return nil, doer.err
	}
	return doer.response, nil
}

type dummyBedrockClient struct {
	output BedrockInvokeOutput
	err    error
	calls  []BedrockInvokeInput
}

func (client *dummyBedrockClient) InvokeModel(_ context.Context, input BedrockInvokeInput) (BedrockInvokeOutput, error) {
	client.calls = append(client.calls, input)
	if client.err != nil {
		return BedrockInvokeOutput{}, client.err
	}
	return client.output, nil
}

type fakeBedrockClientError struct {
	code    string
	message string
}

func (err fakeBedrockClientError) Error() string {
	return fmt.Sprintf("%s: %s", err.code, err.message)
}

func (err fakeBedrockClientError) ErrorCode() string {
	return err.code
}

func (err fakeBedrockClientError) ErrorMessage() string {
	return err.message
}

func TestTextProvidersGenerateNormalizeResponse(t *testing.T) {
	testCases := []struct {
		name          string
		provider      ChatProviderAdapter
		expected      models.ChatProvider
		body          string
		expectedText  string
		expectedUsage map[string]int
	}{
		{
			name: "openai",
			provider: NewOpenAIProvider(
				"k",
				"https://api.openai.com",
				&dummyHTTPDoer{
					response: buildJSONResponse(200, `{
						"choices":[{"message":{"content":"OpenAI reply"}}],
						"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}
					}`),
				},
			),
			expected:     models.ChatProviderOpenAI,
			expectedText: "OpenAI reply",
			expectedUsage: map[string]int{
				"total_tokens": 15,
			},
		},
		{
			name: "anthropic",
			provider: NewAnthropicProvider(
				"k",
				"https://api.anthropic.com",
				"2023-06-01",
				&dummyHTTPDoer{
					response: buildJSONResponse(200, `{
						"content":[{"type":"text","text":"Anthropic reply"}],
						"usage":{"input_tokens":11,"output_tokens":7}
					}`),
				},
			),
			expected:     models.ChatProviderAnthropic,
			expectedText: "Anthropic reply",
			expectedUsage: map[string]int{
				"input_tokens": 11,
				"total_tokens": 18,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			result, err := testCase.provider.Generate(context.Background(), providerTestRequest())
			requireNoError(t, err)
			requireEqual(t, testCase.expected, result.Provider)
			requireEqual(t, testCase.expectedText, result.Text)
			for key, value := range testCase.expectedUsage {
				requireEqual(t, value, result.TokenUsage[key])
			}
		})
	}
}

func TestBedrockProviderGenerateNormalizesResponse(t *testing.T) {
	client := &dummyBedrockClient{
		output: BedrockInvokeOutput{
			Body: []byte(`{
				"content":[{"type":"text","text":"Bedrock reply"}],
				"usage":{"input_tokens":13,"output_tokens":8}
			}`),
		},
	}
	provider := NewBedrockProvider(
		AWSRuntimeCredentials{RegionName: "us-east-1"},
		client,
	)

	result, err := provider.Generate(context.Background(), providerTestRequest())
	requireNoError(t, err)
	requireEqual(t, models.ChatProviderBedrock, result.Provider)
	requireEqual(t, "Bedrock reply", result.Text)
	requireEqual(t, 21, result.TokenUsage["total_tokens"])
	if len(client.calls) == 0 {
		t.Fatalf("expected invoke_model call")
	}
}

func TestOpenAIHealthcheckRequiresAPIKey(t *testing.T) {
	provider := NewOpenAIProvider("", defaultOpenAIBaseURL, nil)

	_, err := provider.Healthcheck()
	requireProviderErrorType[*ProviderAuthError](t, err)
}

func TestAnthropicHealthcheckRequiresAPIKey(t *testing.T) {
	provider := NewAnthropicProvider("", defaultAnthropicBaseURL, defaultAnthropicVersion, nil)

	_, err := provider.Healthcheck()
	requireProviderErrorType[*ProviderAuthError](t, err)
}

func TestBedrockHealthcheckRequiresRegion(t *testing.T) {
	provider := NewBedrockProvider(AWSRuntimeCredentials{RegionName: ""}, nil)

	_, err := provider.Healthcheck()
	requireProviderErrorType[*ProviderAuthError](t, err)
}

func TestBedrockProviderReportsMissingCredentials(t *testing.T) {
	provider := NewBedrockProvider(
		AWSRuntimeCredentials{RegionName: "us-east-1"},
		&dummyBedrockClient{err: ErrNoAWSCredentials},
	)

	_, err := provider.Generate(context.Background(), providerTestRequest())
	requireProviderErrorType[*ProviderAuthError](t, err)
	if !strings.Contains(err.Error(), "credentials not found") {
		t.Fatalf("expected missing credentials message, got %q", err.Error())
	}
}

func TestBedrockProviderMapsClientErrors(t *testing.T) {
	testCases := []bedrockClientErrorCase{
		{
			name:         "validation",
			errorCode:    "ValidationException",
			message:      "Invocation of model ID test-model requires inference profile.",
			expectedType: "request",
			expectedDetail: []string{
				"ValidationException",
				"inference profile",
			},
		},
		{
			name:         "throttled",
			errorCode:    "ThrottlingException",
			message:      "Rate exceeded",
			expectedType: "rate_limit",
			expectedDetail: []string{
				"ThrottlingException",
			},
		},
		{
			name:         "auth",
			errorCode:    "ExpiredTokenException",
			message:      "The security token included in the request is expired",
			expectedType: "auth",
			expectedDetail: []string{
				"ExpiredTokenException",
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			assertBedrockClientErrorCase(t, testCase)
		})
	}
}

type bedrockClientErrorCase struct {
	name           string
	errorCode      string
	message        string
	expectedType   string
	expectedDetail []string
}

func assertBedrockClientErrorCase(t *testing.T, testCase bedrockClientErrorCase) {
	t.Helper()
	provider := NewBedrockProvider(
		AWSRuntimeCredentials{RegionName: "us-east-1"},
		&dummyBedrockClient{
			err: fakeBedrockClientError{
				code:    testCase.errorCode,
				message: testCase.message,
			},
		},
	)
	_, err := provider.Generate(context.Background(), providerTestRequest())
	if err == nil {
		t.Fatalf("expected provider error")
	}
	if !matchesProviderErrorType(err, testCase.expectedType) {
		t.Fatalf("expected provider error type %q, got %T", testCase.expectedType, err)
	}
	assertErrorContainsDetails(t, err, testCase.expectedDetail)
}

func assertErrorContainsDetails(t *testing.T, err error, details []string) {
	t.Helper()
	for _, detail := range details {
		if !strings.Contains(err.Error(), detail) {
			t.Fatalf("expected detail %q in %q", detail, err.Error())
		}
	}
}

func matchesProviderErrorType(err error, expectedType string) bool {
	switch expectedType {
	case "request":
		var target *ProviderRequestError
		return errors.As(err, &target)
	case "rate_limit":
		var target *ProviderRateLimitError
		return errors.As(err, &target)
	case "auth":
		var target *ProviderAuthError
		return errors.As(err, &target)
	default:
		return false
	}
}

func TestBedrockProviderExtractTextIgnoresNonTextContent(t *testing.T) {
	client := &dummyBedrockClient{
		output: BedrockInvokeOutput{
			Body: []byte(`{
				"content":[{"type":"tool_use","name":"x"},{"type":"text","text":7}],
				"usage":{"input_tokens":1,"output_tokens":1}
			}`),
		},
	}
	provider := NewBedrockProvider(
		AWSRuntimeCredentials{RegionName: "us-east-1"},
		client,
	)

	result, err := provider.Generate(context.Background(), providerTestRequest())
	requireNoError(t, err)
	requireEqual(t, "", result.Text)
}

func providerTestRequest() ProviderGenerateRequest {
	return ProviderGenerateRequest{
		ModelID: "test-model",
		Messages: []ProviderMessage{
			{Role: "user", Content: "Hello"},
		},
		SystemPrompt: "You are helpful.",
		Temperature:  0.2,
		MaxTokens:    128,
	}
}

func buildJSONResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func requireEqual[T comparable](t *testing.T, expected T, actual T) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func requireProviderErrorType[T error](t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected provider error")
	}
	var target T
	if !errors.As(err, &target) {
		t.Fatalf("expected error type %T, got %T", target, err)
	}
}

func TestOpenAIProviderBuildsRequestPayloadWithSystemPrompt(t *testing.T) {
	payload := buildOpenAIPayload(providerTestRequest())
	encoded, err := json.Marshal(payload)
	requireNoError(t, err)
	jsonText := string(encoded)
	if !strings.Contains(jsonText, `"role":"system"`) {
		t.Fatalf("expected system role in payload")
	}
	if !strings.Contains(jsonText, `"model":"test-model"`) {
		t.Fatalf("expected model in payload")
	}
}
