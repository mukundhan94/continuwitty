package export

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"
)

var errNotZIPArchive = errors.New("payload is not a zip archive")

func parseProjectExportBundle(fileBytes []byte) (ProjectExportBundle, error) {
	textPayload, err := decodeBundleText(fileBytes)
	if err != nil {
		return ProjectExportBundle{}, err
	}
	var bundle ProjectExportBundle
	if err := json.Unmarshal([]byte(textPayload), &bundle); err != nil {
		return ProjectExportBundle{}, ErrInvalidExportBundle
	}
	return bundle, nil
}

func decodeBundleText(fileBytes []byte) (string, error) {
	if len(fileBytes) == 0 {
		return "", ErrImportFileEmpty
	}
	text, err := readExportJSONFromZIP(fileBytes)
	if err == nil {
		return text, nil
	}
	if !errors.Is(err, errNotZIPArchive) {
		return "", err
	}
	if !utf8.Valid(fileBytes) {
		return "", ErrUnsupportedImportFileFormat
	}
	return string(fileBytes), nil
}

func readExportJSONFromZIP(fileBytes []byte) (string, error) {
	archive, err := zip.NewReader(bytes.NewReader(fileBytes), int64(len(fileBytes)))
	if err != nil {
		return "", errNotZIPArchive
	}
	file := findExportJSONFile(archive)
	if file == nil {
		return "", ErrImportZipMissingExportJSON
	}
	reader, err := file.Open()
	if err != nil {
		return "", ErrUnsupportedImportFileFormat
	}
	defer reader.Close()
	payload, err := io.ReadAll(reader)
	if err != nil {
		return "", ErrUnsupportedImportFileFormat
	}
	if !utf8.Valid(payload) {
		return "", ErrUnsupportedImportFileFormat
	}
	return string(payload), nil
}

func findExportJSONFile(archive *zip.Reader) *zip.File {
	for _, file := range archive.File {
		if file.Name == "export.json" {
			return file
		}
	}
	return nil
}
