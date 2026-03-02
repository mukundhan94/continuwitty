package evalops

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadSuiteRun reads a run artifact from disk.
func ReadSuiteRun(path string) (*SuiteRun, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil, nil
	}
	body, err := os.ReadFile(trimmed)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(body) == 0 {
		return nil, nil
	}
	var run SuiteRun
	if err := json.Unmarshal(body, &run); err != nil {
		return nil, fmt.Errorf("parse suite run %s: %w", trimmed, err)
	}
	return &run, nil
}

// WriteSuiteRun writes one run artifact as pretty JSON.
func WriteSuiteRun(path string, run SuiteRun) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil
	}
	if err := ensureParentDir(trimmed); err != nil {
		return err
	}
	body, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal suite run: %w", err)
	}
	body = append(body, '\n')
	return os.WriteFile(trimmed, body, 0o644)
}

// LoadHistory reads historical runs from JSONL.
func LoadHistory(path string) ([]SuiteRun, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return []SuiteRun{}, nil
	}
	file, err := os.Open(trimmed)
	if os.IsNotExist(err) {
		return []SuiteRun{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	runs := make([]SuiteRun, 0)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var run SuiteRun
		if err := json.Unmarshal([]byte(line), &run); err != nil {
			return nil, fmt.Errorf("parse history line: %w", err)
		}
		runs = append(runs, run)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return runs, nil
}

// AppendHistory appends a run and keeps only the most recent maxEntries when maxEntries > 0.
func AppendHistory(path string, run SuiteRun, maxEntries int) ([]SuiteRun, error) {
	history, err := LoadHistory(path)
	if err != nil {
		return nil, err
	}
	history = append(history, run)
	if maxEntries > 0 && len(history) > maxEntries {
		history = history[len(history)-maxEntries:]
	}
	if err := WriteHistory(path, history); err != nil {
		return nil, err
	}
	return history, nil
}

// WriteHistory stores JSONL history.
func WriteHistory(path string, history []SuiteRun) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil
	}
	if err := ensureParentDir(trimmed); err != nil {
		return err
	}
	lines := make([]string, 0, len(history))
	for _, run := range history {
		body, err := json.Marshal(run)
		if err != nil {
			return fmt.Errorf("marshal history run: %w", err)
		}
		lines = append(lines, string(body))
	}
	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	return os.WriteFile(trimmed, []byte(content), 0o644)
}

// LastRun returns the most recent run in history.
func LastRun(history []SuiteRun) *SuiteRun {
	if len(history) == 0 {
		return nil
	}
	latest := history[len(history)-1]
	return &latest
}

func ensureParentDir(path string) error {
	directory := filepath.Dir(path)
	if directory == "" || directory == "." {
		return nil
	}
	return os.MkdirAll(directory, 0o755)
}
