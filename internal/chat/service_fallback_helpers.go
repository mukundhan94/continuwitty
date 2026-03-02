package chat

import (
	"context"

	"engram/internal/providers"
)

type fallbackIterationInput struct {
	TraceID        string
	Operation      string
	Prepared       PreparedGeneration
	Candidate      ProviderFallbackCandidate
	CandidateIndex int
	CandidateCount int
}

func (service *ChatService) generateWithFallback(
	ctx context.Context,
	traceID string,
	prepared PreparedGeneration,
) (providers.ProviderGenerateResult, ProviderFallbackCandidate, error) {
	candidates := service.providerFallback.Candidates(prepared.Session)
	for index, candidate := range candidates {
		iteration := fallbackIterationInput{
			TraceID:        traceID,
			Operation:      chatOperationSend,
			Prepared:       prepared,
			Candidate:      candidate,
			CandidateIndex: index,
			CandidateCount: len(candidates),
		}
		skip, terminalErr := service.checkCircuitForCandidate(iteration)
		if skip {
			if terminalErr != nil {
				return providers.ProviderGenerateResult{}, candidate, terminalErr
			}
			continue
		}
		adapter, err := service.resolveCandidateAdapter(ctx, iteration)
		if err != nil {
			return providers.ProviderGenerateResult{}, candidate, err
		}
		result, err := adapter.Generate(ctx, service.providerRequestWithCandidate(prepared, candidate))
		if err == nil {
			service.recordCandidateSuccess(iteration)
			result.Provider = candidate.Provider
			result.ModelID = candidate.ModelID
			return result, candidate, nil
		}
		providerErr, stop, mappedErr := service.mapCandidateError(iteration, err)
		if mappedErr != nil {
			return providers.ProviderGenerateResult{}, candidate, mappedErr
		}
		if stop {
			return providers.ProviderGenerateResult{}, candidate, providerErr
		}
	}
	return providers.ProviderGenerateResult{}, ProviderFallbackCandidate{}, NewChatProviderExecutionError(
		"provider execution failed",
		502,
		"provider_error",
	)
}

func (service *ChatService) streamWithFallback(
	ctx context.Context,
	traceID string,
	prepared PreparedGeneration,
) (StreamChunkResult, ProviderFallbackCandidate, error) {
	candidates := service.providerFallback.Candidates(prepared.Session)
	for index, candidate := range candidates {
		iteration := fallbackIterationInput{
			TraceID:        traceID,
			Operation:      chatOperationStream,
			Prepared:       prepared,
			Candidate:      candidate,
			CandidateIndex: index,
			CandidateCount: len(candidates),
		}
		skip, terminalErr := service.checkCircuitForCandidate(iteration)
		if skip {
			if terminalErr != nil {
				return StreamChunkResult{ProviderError: terminalErr}, candidate, nil
			}
			continue
		}
		adapter, err := service.resolveCandidateAdapter(ctx, iteration)
		if err != nil {
			return StreamChunkResult{}, candidate, err
		}
		candidatePrepared := prepared
		candidatePrepared.ProviderRequest = service.providerRequestWithCandidate(prepared, candidate)
		streamResult, err := service.runtime.YieldStreamChunks(
			ctx,
			adapter.StreamGenerate,
			candidatePrepared,
		)
		if err != nil {
			return StreamChunkResult{}, candidate, err
		}
		if streamResult.ProviderError == nil {
			service.recordCandidateSuccess(iteration)
			return streamResult, candidate, nil
		}
		if service.recordCandidateFailure(iteration, streamResult.ProviderError.ErrorCode()) {
			return streamResult, candidate, nil
		}
	}
	return StreamChunkResult{}, ProviderFallbackCandidate{}, NewChatProviderExecutionError(
		"provider execution failed",
		502,
		"provider_error",
	)
}

func (service *ChatService) checkCircuitForCandidate(
	input fallbackIterationInput,
) (bool, *ChatProviderExecutionError) {
	if service.circuitPolicy.Allow(input.Candidate.Provider) {
		return false, nil
	}
	providerErr := NewChatProviderExecutionError(
		"provider circuit is open",
		503,
		"provider_circuit_open",
	)
	service.recordProviderFailure(input.Candidate.Provider, input.Operation, providerErr.ErrorCode())
	service.recordLifecycleTrace(traceRecordInput{
		TraceID:   input.TraceID,
		Operation: input.Operation,
		Stage:     chatTraceProviderOpen,
		Prepared:  &input.Prepared,
		Provider:  input.Candidate.Provider,
		ErrorCode: providerErr.ErrorCode(),
	})
	if isLastFallbackCandidate(input) {
		return true, providerErr
	}
	return true, nil
}

func (service *ChatService) resolveCandidateAdapter(
	ctx context.Context,
	input fallbackIterationInput,
) (providers.ChatProviderAdapter, error) {
	adapter, err := service.resolveProvider(input.Candidate.Provider)
	if err != nil {
		return nil, err
	}
	service.recordLifecycleTrace(traceRecordInput{
		TraceID:   input.TraceID,
		Operation: input.Operation,
		Stage:     chatTraceProviderTry,
		Prepared:  &input.Prepared,
		Provider:  input.Candidate.Provider,
	})
	return adapter, nil
}

func (service *ChatService) mapCandidateError(
	input fallbackIterationInput,
	err error,
) (*ChatProviderExecutionError, bool, error) {
	mapped := MapProviderError(err)
	providerErr, ok := mapped.(*ChatProviderExecutionError)
	if !ok {
		return nil, false, mapped
	}
	return providerErr, service.recordCandidateFailure(input, providerErr.ErrorCode()), nil
}

func (service *ChatService) recordCandidateSuccess(input fallbackIterationInput) {
	service.circuitPolicy.RecordResult(input.Candidate.Provider, "")
	service.recordLifecycleTrace(traceRecordInput{
		TraceID:   input.TraceID,
		Operation: input.Operation,
		Stage:     chatTraceProviderOK,
		Prepared:  &input.Prepared,
		Provider:  input.Candidate.Provider,
	})
}

func (service *ChatService) recordCandidateFailure(
	input fallbackIterationInput,
	errorCode string,
) bool {
	service.circuitPolicy.RecordResult(input.Candidate.Provider, errorCode)
	service.recordProviderFailure(input.Candidate.Provider, input.Operation, errorCode)
	service.recordLifecycleTrace(traceRecordInput{
		TraceID:   input.TraceID,
		Operation: input.Operation,
		Stage:     chatTraceProviderFail,
		Prepared:  &input.Prepared,
		Provider:  input.Candidate.Provider,
		ErrorCode: errorCode,
	})
	return !isTransientProviderErrorCode(errorCode) || isLastFallbackCandidate(input)
}

func isLastFallbackCandidate(input fallbackIterationInput) bool {
	return input.CandidateIndex == input.CandidateCount-1
}

func (service *ChatService) providerRequestWithCandidate(
	prepared PreparedGeneration,
	candidate ProviderFallbackCandidate,
) providers.ProviderGenerateRequest {
	request := prepared.ProviderRequest
	request.ModelID = candidate.ModelID
	return request
}
