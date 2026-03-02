package mcp

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

var (
	backupToolNamePattern  = regexp.MustCompile(`"name"\s*:\s*"([a-z_]+\.[a-z0-9_]+)"`)
	backupToolValuePattern = regexp.MustCompile(`"([a-z_]+\.[a-z0-9_]+)"`)
	backupAliasPattern     = regexp.MustCompile(`"([a-z_]+\.[a-z0-9_]+)"\s*:\s*"([a-z_]+\.[a-z0-9_]+)"`)
)

func TestGoCatalogIncludesAllBackupTools(t *testing.T) {
	catalogText := readBackupCatalog(t)
	backupNames := extractPatternMatches(catalogText, backupToolNamePattern)

	goOrder := mapFromToolCatalogSlice(toolCatalogOrder)
	goMetadata := mapFromKeys(toolCatalogEntries)

	if missing := missingKeys(backupNames, goOrder); len(missing) > 0 {
		t.Fatalf("toolCatalogOrder missing backup tools: %v", missing)
	}
	if missing := missingKeys(backupNames, goMetadata); len(missing) > 0 {
		t.Fatalf("toolCatalogEntries missing backup tools: %v", missing)
	}
}

func TestGoReadWriteSetsIncludeAllBackupTools(t *testing.T) {
	catalogText := readBackupCatalog(t)

	backupRead := extractSetBlockValues(catalogText, "_READ_TOOL_NAMES")
	backupWrite := extractSetBlockValues(catalogText, "_WRITE_TOOL_NAMES")

	if missing := missingKeys(backupRead, mapFromToolSet(readToolNames)); len(missing) > 0 {
		t.Fatalf("readToolNames missing backup read tools: %v", missing)
	}
	if missing := missingKeys(backupWrite, mapFromToolSet(writeToolNames)); len(missing) > 0 {
		t.Fatalf("writeToolNames missing backup write tools: %v", missing)
	}
}

func TestGoAliasesIncludeAllBackupAliases(t *testing.T) {
	catalogText := readBackupCatalog(t)
	backupAliasBlock := extractNamedBlock(t, catalogText, "_TOOL_ALIASES")
	pairs := backupAliasPattern.FindAllStringSubmatch(backupAliasBlock, -1)

	for _, pair := range pairs {
		if len(pair) < 3 {
			continue
		}
		alias := pair[1]
		target := pair[2]
		mapped, ok := toolAliases[toolIdentifier(alias)]
		if !ok {
			t.Fatalf("missing backup alias %q", alias)
		}
		if mapped.String() != target {
			t.Fatalf("alias mismatch for %q: got %q want %q", alias, mapped, target)
		}
	}
}

func readBackupCatalog(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve runtime caller")
	}
	catalogPath := filepath.Clean(
		filepath.Join(filepath.Dir(file), "..", "..", "backup", "app", "mcp", "catalog.py"),
	)
	content, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatalf("read backup catalog %q: %v", catalogPath, err)
	}
	return string(content)
}

func extractPatternMatches(content string, pattern *regexp.Regexp) map[string]struct{} {
	matches := pattern.FindAllStringSubmatch(content, -1)
	result := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		result[match[1]] = struct{}{}
	}
	return result
}

func extractSetBlockValues(content string, blockName string) map[string]struct{} {
	block := extractNamedBlock(nil, content, blockName)
	values := backupToolValuePattern.FindAllStringSubmatch(block, -1)
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		if len(value) < 2 {
			continue
		}
		result[value[1]] = struct{}{}
	}
	return result
}

func extractNamedBlock(t *testing.T, content string, blockName string) string {
	pattern := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(blockName) + `\s*=\s*\{(.*?)\}`)
	match := pattern.FindStringSubmatch(content)
	if len(match) >= 2 {
		return match[1]
	}
	if t != nil {
		t.Fatalf("block %q not found in backup catalog", blockName)
	}
	return ""
}

func mapFromToolCatalogSlice(values []toolIdentifier) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value.String()] = struct{}{}
	}
	return result
}

func mapFromToolSet(values map[toolIdentifier]struct{}) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for value := range values {
		result[value.String()] = struct{}{}
	}
	return result
}

func mapFromKeys[T any](items map[string]T) map[string]struct{} {
	result := make(map[string]struct{}, len(items))
	for key := range items {
		result[key] = struct{}{}
	}
	return result
}

func missingKeys(required map[string]struct{}, available map[string]struct{}) []string {
	missing := make([]string, 0)
	for key := range required {
		if _, ok := available[key]; !ok {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	return missing
}

func TestBackupCatalogIsPresentAndNonEmpty(t *testing.T) {
	content := readBackupCatalog(t)
	if strings.TrimSpace(content) == "" {
		t.Fatal("backup catalog is empty")
	}
}
