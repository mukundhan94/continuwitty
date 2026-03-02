package mcp

import (
	"context"
	"reflect"
	"strings"
)

type streamChatSendMessageRouteInput struct {
	requestID  any
	toolName   string
	actor      Actor
	params     map[string]any
	asToolCall bool
}

func (service *CompatibilityService) emitStreamCallFrames(
	ctx context.Context,
	request StreamCallRequest,
	frames chan<- Frame,
) bool {
	streamFrames, handled := service.streamCallFrames(ctx, request)
	if !handled {
		return false
	}
	for _, frame := range streamFrames {
		if !emitFrame(ctx, frames, frame) {
			return true
		}
	}
	return true
}

func (service *CompatibilityService) streamCallFrames(
	ctx context.Context,
	request StreamCallRequest,
) ([]Frame, bool) {
	if frames, handled := service.streamDirectChatSendMessageFrames(ctx, request); handled {
		return frames, true
	}
	if frames, handled := service.streamToolsCallChatSendMessageFrames(ctx, request); handled {
		return frames, true
	}
	return nil, false
}

func (service *CompatibilityService) streamDirectChatSendMessageFrames(
	ctx context.Context,
	request StreamCallRequest,
) ([]Frame, bool) {
	method := strings.TrimSpace(request.Request.Method)
	if method == "" || method == "tools/call" {
		return nil, false
	}
	dottedMethod := toDottedToolName(toolIdentifier(method))
	if canonicalToolName(dottedMethod) != "chat.send_message" {
		return nil, false
	}
	if policyError := authorizeToolCall(dottedMethod, request.TokenAuth); policyError != nil {
		return []Frame{
			errorFrame(request.Request.ID, policyError.code, policyError.message, policyError.data),
		}, true
	}
	return service.streamChatSendMessageFrames(
		ctx,
		streamChatSendMessageRouteInput{
			requestID:  request.Request.ID,
			toolName:   method,
			actor:      request.Actor,
			params:     request.Request.Params,
			asToolCall: false,
		},
	), true
}

func (service *CompatibilityService) streamToolsCallChatSendMessageFrames(
	ctx context.Context,
	request StreamCallRequest,
) ([]Frame, bool) {
	if request.Request.Method != "tools/call" {
		return nil, false
	}
	name, ok := requiredToolName(request.Request.Params)
	if !ok {
		return nil, false
	}
	arguments, ok := toolCallArguments(request.Request.Params)
	if !ok {
		return nil, false
	}
	dottedName := toDottedToolName(toolIdentifier(name))
	if canonicalToolName(dottedName) != "chat.send_message" {
		return nil, false
	}
	if policyError := authorizeToolCall(dottedName, request.TokenAuth); policyError != nil {
		return []Frame{
			errorFrame(request.Request.ID, policyError.code, policyError.message, policyError.data),
		}, true
	}
	if !streamEnabled(arguments) {
		return nil, false
	}
	return service.streamChatSendMessageFrames(
		ctx,
		streamChatSendMessageRouteInput{
			requestID:  request.Request.ID,
			toolName:   name,
			actor:      request.Actor,
			params:     arguments,
			asToolCall: true,
		},
	), true
}

func (service *CompatibilityService) streamChatSendMessageFrames(
	ctx context.Context,
	input streamChatSendMessageRouteInput,
) []Frame {
	if !streamEnabled(input.params) {
		return service.nonStreamChatSendMessageFrames(ctx, input)
	}
	if service.messageStream == nil {
		return service.nonStreamChatSendMessageFrames(ctx, input)
	}
	sendRequest, dispatchErr := parseSessionMessageSendRequest(input.actor, input.params)
	if dispatchErr != nil {
		return []Frame{errorFrame(input.requestID, dispatchErr.code, dispatchErr.message, dispatchErr.data)}
	}
	events, err := service.messageStream.StreamMessageEvents(
		ctx,
		sendRequest,
	)
	if err != nil {
		return []Frame{chatSendErrorFrame(input.requestID, err)}
	}
	return buildChatSendStreamFrames(
		input.requestID,
		input.toolName,
		input.asToolCall,
		events,
	)
}

func (service *CompatibilityService) nonStreamChatSendMessageFrames(
	ctx context.Context,
	input streamChatSendMessageRouteInput,
) []Frame {
	if service.messageSend == nil {
		return []Frame{
			errorFrame(
				input.requestID,
				-32000,
				"Tool not implemented",
				map[string]any{"method": "chat.send_message"},
			),
		}
	}
	sendRequest, dispatchErr := parseSessionMessageSendRequest(input.actor, input.params)
	if dispatchErr != nil {
		return []Frame{errorFrame(input.requestID, dispatchErr.code, dispatchErr.message, dispatchErr.data)}
	}
	response, err := service.messageSend.SendMessage(
		ctx,
		sendRequest,
	)
	if err != nil {
		return []Frame{chatSendErrorFrame(input.requestID, err)}
	}
	if response == nil {
		return []Frame{errorFrame(input.requestID, -32603, "Internal error", nil)}
	}
	return []Frame{chatSendSuccessFrame(input.requestID, input.toolName, input.asToolCall, map[string]any{"message": *response})}
}

func streamEnabled(params map[string]any) bool {
	rawValue, exists := params["stream"]
	if !exists {
		return true
	}
	return truthyValue(rawValue)
}

func truthyValue(value any) bool {
	if value == nil {
		return false
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return typed != ""
	}
	reflectValue := reflect.ValueOf(value)
	switch reflectValue.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflectValue.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return reflectValue.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return reflectValue.Float() != 0
	case reflect.Slice, reflect.Array, reflect.Map:
		return reflectValue.Len() > 0
	default:
		return true
	}
}

func emitFrame(ctx context.Context, frames chan<- Frame, frame Frame) bool {
	select {
	case <-ctx.Done():
		return false
	case frames <- frame:
		return true
	}
}

func invalidParamErrorFrame(requestID any, field string) Frame {
	dispatchError := invalidParamError(field)
	return errorFrame(requestID, dispatchError.code, dispatchError.message, dispatchError.data)
}

func chatSendErrorFrame(requestID any, err error) Frame {
	dispatchError := mapChatSendError(err)
	return errorFrame(requestID, dispatchError.code, dispatchError.message, dispatchError.data)
}

func streamErrorEventFrame(requestID any, payload map[string]any) Frame {
	detail := "chat.send_message stream failed"
	if rawDetail, ok := payload["detail"].(string); ok {
		trimmed := strings.TrimSpace(rawDetail)
		if trimmed != "" {
			detail = trimmed
		}
	}
	return errorFrame(requestID, -32020, detail, payload)
}

func chatSendSuccessFrame(
	requestID any,
	toolName string,
	asToolCall bool,
	messagePayload map[string]any,
) Frame {
	if asToolCall {
		return successFrame(requestID, buildToolCallSuccessResult(toolName, messagePayload))
	}
	return successFrame(requestID, messagePayload)
}

func streamEventPayload(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{"value": nil}
	}
	return payload
}

func buildChatSendStreamFrames(
	requestID any,
	toolName string,
	asToolCall bool,
	events []MessageStreamEvent,
) []Frame {
	frames := make([]Frame, 0, len(events)+1)
	finalMessage, terminatedWithError := appendChatSendEventFrames(requestID, toolName, events, &frames)
	if terminatedWithError {
		return frames
	}
	if finalMessage == nil {
		return append(frames, errorFrame(requestID, -32021, "chat.send_message stream ended without completion", nil))
	}
	return append(
		frames,
		chatSendSuccessFrame(
			requestID,
			toolName,
			asToolCall,
			map[string]any{"message": finalMessage},
		),
	)
}

func appendChatSendEventFrames(
	requestID any,
	toolName string,
	events []MessageStreamEvent,
	frames *[]Frame,
) (map[string]any, bool) {
	var finalMessage map[string]any
	for _, event := range events {
		payload := streamEventPayload(event.Payload)
		*frames = append(*frames, mcpEventFrame(requestID, toolName, event.Type, payload))
		if event.Type == "error" {
			*frames = append(*frames, streamErrorEventFrame(requestID, payload))
			return nil, true
		}
		if event.Type == "done" {
			finalMessage = payload
		}
	}
	return finalMessage, false
}

func mcpEventFrame(requestID any, toolName string, eventName string, eventPayload map[string]any) Frame {
	return Frame{
		"jsonrpc": "2.0",
		"method":  "mcp.event",
		"params": map[string]any{
			"id":    requestID,
			"tool":  toolName,
			"event": eventName,
			"data":  eventPayload,
		},
	}
}
