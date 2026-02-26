package export

import (
	"engram/internal/models"

	"github.com/google/uuid"
)

func collectionIDsOf(collections []models.EngramCollectionRecord) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(collections))
	for _, collection := range collections {
		ids = append(ids, collection.CollectionID)
	}
	return ids
}

func cloneCollections(collections []models.EngramCollectionRecord) []models.EngramCollectionRecord {
	cloned := make([]models.EngramCollectionRecord, 0, len(collections))
	for _, collection := range collections {
		cloned = append(cloned, collection)
	}
	return cloned
}

func cloneSourceRecords(values []models.AdminEngramSourceRecord) []models.AdminEngramSourceRecord {
	cloned := make([]models.AdminEngramSourceRecord, 0, len(values))
	for _, value := range values {
		cloned = append(cloned, value)
	}
	return cloned
}

func copyStringSlice(values []string) []string {
	copied := make([]string, 0, len(values))
	for _, value := range values {
		copied = append(copied, value)
	}
	return copied
}

func copyUUIDSlice(values []uuid.UUID) []uuid.UUID {
	copied := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		copied = append(copied, value)
	}
	return copied
}

func toSourceInputs(values []models.AdminEngramSourceRecord) []models.AdminEngramSourceInput {
	inputs := make([]models.AdminEngramSourceInput, 0, len(values))
	for _, value := range values {
		inputs = append(
			inputs,
			models.AdminEngramSourceInput{
				CapturedAt:  value.CapturedAt,
				URL:         value.URL,
				Title:       value.Title,
				Snippet:     value.Snippet,
				ContentText: value.ContentText,
				ContentHash: value.ContentHash,
			},
		)
	}
	return inputs
}
