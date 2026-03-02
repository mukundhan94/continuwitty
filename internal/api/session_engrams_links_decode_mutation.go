package api

import (
	"net/http"
	"strings"

	"engram/internal/models"

	"github.com/google/uuid"
)

func decodeCreateEngramLinkInput(
	writer http.ResponseWriter,
	request *http.Request,
	sourceEngramID uuid.UUID,
	actorUserID uuid.UUID,
) (SessionEngramLinkCreateInput, bool) {
	payload := createEngramLinkPayload{}
	if !decodeJSONAllowEmpty(writer, request, &payload) {
		return SessionEngramLinkCreateInput{}, false
	}
	if payload.TargetEngramID == uuid.Nil {
		writeInvalidParameter(writer, "target_engram_id")
		return SessionEngramLinkCreateInput{}, false
	}
	relationType, ok := parseOptionalStringEnum(
		writer,
		"relation_type",
		payload.RelationType,
		models.EngramLinkRelationRelatedTo,
		models.ParseEngramLinkRelationType,
	)
	if !ok {
		return SessionEngramLinkCreateInput{}, false
	}
	origin, ok := parseOptionalStringEnum(
		writer,
		"origin",
		payload.Origin,
		models.EngramLinkOriginManual,
		models.ParseEngramLinkOrigin,
	)
	if !ok {
		return SessionEngramLinkCreateInput{}, false
	}
	status, ok := parseOptionalStringEnum(
		writer,
		"status",
		payload.Status,
		models.EngramLinkStatusActive,
		models.ParseEngramLinkStatus,
	)
	if !ok {
		return SessionEngramLinkCreateInput{}, false
	}
	scores, ok := parseCreateEngramLinkScores(writer, payload)
	if !ok {
		return SessionEngramLinkCreateInput{}, false
	}
	return SessionEngramLinkCreateInput{
		SourceEngramID:   sourceEngramID,
		TargetEngramID:   payload.TargetEngramID,
		RelationType:     relationType,
		Weight:           scores.weight,
		TemporalWeight:   scores.temporalWeight,
		Confidence:       scores.confidence,
		Origin:           origin,
		Status:           status,
		EvidenceJSON:     emptyMap(payload.EvidenceJSON),
		CreatedByUserID:  actorUserID,
		ActorUserID:      actorUserID,
		LastReinforcedAt: payload.LastReinforcedAt,
	}, true
}

type createEngramLinkScores struct {
	weight         float64
	temporalWeight float64
	confidence     float64
}

func parseCreateEngramLinkScores(
	writer http.ResponseWriter,
	payload createEngramLinkPayload,
) (createEngramLinkScores, bool) {
	weight, ok := parseScoreWithDefault(writer, "weight", payload.Weight, 0.6)
	if !ok {
		return createEngramLinkScores{}, false
	}
	temporalWeight, ok := parseScoreWithDefault(writer, "temporal_weight", payload.TemporalWeight, 0.5)
	if !ok {
		return createEngramLinkScores{}, false
	}
	confidence, ok := parseScoreWithDefault(writer, "confidence", payload.Confidence, 0.5)
	if !ok {
		return createEngramLinkScores{}, false
	}
	return createEngramLinkScores{
		weight:         weight,
		temporalWeight: temporalWeight,
		confidence:     confidence,
	}, true
}

func decodeUpdateEngramLinkInput(
	writer http.ResponseWriter,
	request *http.Request,
	linkID uuid.UUID,
	actorUserID uuid.UUID,
) (SessionEngramLinkUpdateInput, bool) {
	payload := updateEngramLinkPayload{}
	if !decodeJSONAllowEmpty(writer, request, &payload) {
		return SessionEngramLinkUpdateInput{}, false
	}
	if !hasEngramLinkUpdateField(payload) {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "No update fields provided"})
		return SessionEngramLinkUpdateInput{}, false
	}
	if !validateOptionalScores(writer, payload) {
		return SessionEngramLinkUpdateInput{}, false
	}
	status, ok := parseUpdateLinkStatus(writer, payload.Status)
	if !ok {
		return SessionEngramLinkUpdateInput{}, false
	}
	return SessionEngramLinkUpdateInput{
		LinkID:           linkID,
		ActorUserID:      actorUserID,
		Weight:           payload.Weight,
		TemporalWeight:   payload.TemporalWeight,
		Confidence:       payload.Confidence,
		Status:           status,
		EvidenceJSON:     payload.EvidenceJSON,
		LastReinforcedAt: payload.LastReinforcedAt,
	}, true
}

func validateOptionalScores(writer http.ResponseWriter, payload updateEngramLinkPayload) bool {
	scoreFields := []struct {
		name  string
		value *float64
	}{
		{name: "weight", value: payload.Weight},
		{name: "temporal_weight", value: payload.TemporalWeight},
		{name: "confidence", value: payload.Confidence},
	}
	for _, field := range scoreFields {
		if !isNilOrScore(field.name, field.value, writer) {
			return false
		}
	}
	return true
}

func parseUpdateLinkStatus(writer http.ResponseWriter, rawValue *string) (*models.EngramLinkStatus, bool) {
	return parseOptionalStringEnumPointer(writer, "status", rawValue, models.ParseEngramLinkStatus)
}

func hasEngramLinkUpdateField(payload updateEngramLinkPayload) bool {
	return payload.Weight != nil ||
		payload.TemporalWeight != nil ||
		payload.Confidence != nil ||
		payload.Status != nil ||
		payload.EvidenceJSON != nil ||
		payload.LastReinforcedAt != nil
}

func parseScoreWithDefault(
	writer http.ResponseWriter,
	field string,
	value *float64,
	defaultValue float64,
) (float64, bool) {
	if value == nil {
		return defaultValue, true
	}
	if *value < 0 || *value > 1 {
		writeInvalidParameter(writer, field)
		return 0, false
	}
	return *value, true
}

type stringEnumParser[T ~string] func(string) (T, error)

func parseOptionalStringEnum[T ~string](
	writer http.ResponseWriter,
	field string,
	rawValue string,
	defaultValue T,
	parse stringEnumParser[T],
) (T, bool) {
	trimmed := strings.TrimSpace(rawValue)
	if trimmed == "" {
		return defaultValue, true
	}
	parsed, err := parse(trimmed)
	if err != nil {
		writeInvalidParameter(writer, field)
		return "", false
	}
	return parsed, true
}

func parseOptionalStringEnumPointer[T ~string](
	writer http.ResponseWriter,
	field string,
	rawValue *string,
	parse stringEnumParser[T],
) (*T, bool) {
	if rawValue == nil {
		return nil, true
	}
	parsed, ok := parseOptionalStringEnum(writer, field, *rawValue, "", parse)
	if !ok {
		return nil, false
	}
	return &parsed, true
}

func isNilOrScore(field string, value *float64, writer http.ResponseWriter) bool {
	if value == nil {
		return true
	}
	if *value < 0 || *value > 1 {
		writeInvalidParameter(writer, field)
		return false
	}
	return true
}

func emptyMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}
