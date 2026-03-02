package export

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestParseProjectExportBundleValidationErrors(t *testing.T) {
	_, err := parseProjectExportBundle(nil)
	if !errors.Is(err, ErrImportFileEmpty) {
		t.Fatalf("expected ErrImportFileEmpty, got %v", err)
	}

	zipBytes := buildZIPPayload(t, map[string]string{"other.json": "{}"})
	_, err = parseProjectExportBundle(zipBytes)
	if !errors.Is(err, ErrImportZipMissingExportJSON) {
		t.Fatalf("expected ErrImportZipMissingExportJSON, got %v", err)
	}

	_, err = parseProjectExportBundle([]byte{0xff, 0xfe, 0xfd})
	if !errors.Is(err, ErrUnsupportedImportFileFormat) {
		t.Fatalf("expected ErrUnsupportedImportFileFormat, got %v", err)
	}

	_, err = parseProjectExportBundle([]byte(`{"schema_version":"1.0","project":`))
	if !errors.Is(err, ErrInvalidExportBundle) {
		t.Fatalf("expected ErrInvalidExportBundle, got %v", err)
	}
}

func TestParseProjectExportBundleFromZIP(t *testing.T) {
	zipBytes := buildZIPPayload(
		t,
		map[string]string{
			"export.json": `{"schema_version":"1.0","project":{"project_id":"alpha"}}`,
		},
	)
	bundle, err := parseProjectExportBundle(zipBytes)
	requireNoError(t, err)
	requireEqual(t, "1.0", bundle.SchemaVersion)
	requireEqual(t, "alpha", bundle.Project.ProjectID)
}

func buildZIPPayload(t *testing.T, files map[string]string) []byte {
	t.Helper()
	buffer := &bytes.Buffer{}
	writer := zip.NewWriter(buffer)
	for name, content := range files {
		fileWriter, err := writer.Create(name)
		requireNoError(t, err)
		_, err = io.WriteString(fileWriter, content)
		requireNoError(t, err)
	}
	requireNoError(t, writer.Close())
	return buffer.Bytes()
}
