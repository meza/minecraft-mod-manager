package perf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultExportFilename = "mmm-perf.json"

type exportEntry struct {
	Name           string              `json:"name"`
	Type           EntryType           `json:"type"`
	Timestamp      *time.Time          `json:"timestamp,omitempty"`
	StartTimestamp *time.Time          `json:"start_timestamp,omitempty"`
	DurationNS     int64               `json:"duration_ns,omitempty"`
	Details        *PerformanceDetails `json:"details,omitempty"`
}

// ExportToFile writes the supplied performance log as JSON to
// <outDir>/mmm-perf.json. Any absolute filesystem paths in known detail keys
// are rewritten to be relative to baseDir so output stays portable.
//
// This is intended to be used as a best-effort diagnostic artifact; callers
// should treat any returned error as non-fatal.
func ExportToFile(outDir string, baseDir string, log PerformanceLog) (string, error) {
	if outDir == "" {
		outDir = "."
	}

	normalized := normalizeForExport(log, baseDir)
	exported := exportLog(normalized)

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", err
	}

	path := filepath.Join(outDir, defaultExportFilename)
	data, err := json.MarshalIndent(exported, "", "  ")
	if err != nil {
		return "", err
	}

	return path, os.WriteFile(path, data, 0644)
}

func normalizeForExport(log PerformanceLog, baseDir string) PerformanceLog {
	if len(log) == 0 {
		return log
	}

	normalized := make(PerformanceLog, 0, len(log))
	for _, entry := range log {
		normalized = append(normalized, Entry{
			Name:      entry.Name,
			Type:      entry.Type,
			StartTime: entry.StartTime,
			Duration:  entry.Duration,
			Details:   normalizeDetails(entry.Details, baseDir),
		})
	}
	return normalized
}

func exportLog(log PerformanceLog) []exportEntry {
	if len(log) == 0 {
		return nil
	}

	exported := make([]exportEntry, 0, len(log))
	for _, entry := range log {
		exported = append(exported, exportEntry{
			Name:           entry.Name,
			Type:           entry.Type,
			Timestamp:      markTimestamp(entry),
			StartTimestamp: measureStartTimestamp(entry),
			DurationNS:     measureDuration(entry),
			Details:        entry.Details,
		})
	}
	return exported
}

func markTimestamp(entry Entry) *time.Time {
	if entry.Type != MarkType {
		return nil
	}
	return &entry.StartTime
}

func measureStartTimestamp(entry Entry) *time.Time {
	if entry.Type != MeasureType {
		return nil
	}
	return &entry.StartTime
}

func measureDuration(entry Entry) int64 {
	if entry.Type != MeasureType {
		return 0
	}
	return entry.Duration.Nanoseconds()
}

func normalizeDetails(details *PerformanceDetails, baseDir string) *PerformanceDetails {
	if details == nil || len(*details) == 0 {
		return details
	}

	normalized := make(PerformanceDetails, len(*details))
	for key, value := range *details {
		normalized[key] = normalizeValue(key, value, baseDir)
	}
	return &normalized
}

func normalizeValue(key string, value interface{}, baseDir string) interface{} {
	stringValue, ok := value.(string)
	if !ok {
		return value
	}

	if key == "config_path" {
		if baseDir != "" && filepath.IsAbs(stringValue) {
			rel, err := filepath.Rel(baseDir, stringValue)
			if err == nil {
				return exportPath(rel)
			}
		}
		return exportPath(stringValue)
	}

	if !looksLikePathKey(key) {
		return value
	}

	if baseDir == "" {
		return exportPath(stringValue)
	}

	if filepath.IsAbs(stringValue) {
		rel, err := filepath.Rel(baseDir, stringValue)
		if err == nil {
			return exportPath(rel)
		}
	}

	return exportPath(stringValue)
}

func looksLikePathKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	return key == "path" || strings.HasSuffix(key, "_path") || strings.HasSuffix(key, "path")
}

func trimLeadingDot(value string) string {
	if value == "." {
		return value
	}
	value = strings.TrimPrefix(value, "./")
	return value
}

func exportPath(value string) string {
	cleaned := trimLeadingDot(filepath.Clean(value))
	if cleaned == "." {
		return cleaned
	}
	return filepath.ToSlash(cleaned)
}
