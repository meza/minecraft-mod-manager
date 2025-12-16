package perf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExportToFile_WritesJSONAndNormalizesPaths(t *testing.T) {
	tempDir := t.TempDir()
	baseDir := filepath.Join(tempDir, "cfg")
	outDir := filepath.Join(tempDir, "out")

	absConfig := filepath.Join(baseDir, "modlist.json")
	absJar := filepath.Join(baseDir, "mods", "a.jar")

	log := PerformanceLog{
		{
			Name:      "example",
			Type:      MarkType,
			StartTime: time.Unix(1, 0),
			Details: &PerformanceDetails{
				"config_path": absConfig,
				"path":        absJar,
			},
		},
		{
			Name:      "example-duration",
			Type:      MeasureType,
			StartTime: time.Unix(1, 0),
			Duration:  3 * time.Second,
		},
	}

	written, err := ExportToFile(outDir, baseDir, log)
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(outDir, defaultExportFilename), written)

	raw, err := os.ReadFile(written)
	assert.NoError(t, err)

	var decoded []exportEntry
	assert.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Len(t, decoded, 2)

	assert.NotNil(t, decoded[0].Details)
	assert.Equal(t, "modlist.json", (*decoded[0].Details)["config_path"])
	assert.Equal(t, "mods/a.jar", (*decoded[0].Details)["path"])

	assert.NotNil(t, decoded[0].Timestamp)
	assert.Nil(t, decoded[0].StartTimestamp)
	assert.Equal(t, int64(0), decoded[0].DurationNS)

	assert.Nil(t, decoded[1].Timestamp)
	assert.NotNil(t, decoded[1].StartTimestamp)
	assert.Equal(t, int64((3 * time.Second).Nanoseconds()), decoded[1].DurationNS)
}

func TestNormalizeForExport_LeavesNonPathDetailsUntouched(t *testing.T) {
	log := PerformanceLog{
		{
			Name:      "example",
			Type:      MarkType,
			StartTime: time.Unix(1, 0),
			Details: &PerformanceDetails{
				"status":  200,
				"url":     "https://example.com",
				"attempt": 1,
			},
		},
	}

	normalized := normalizeForExport(log, "/base")
	assert.Len(t, normalized, 1)
	assert.NotNil(t, normalized[0].Details)
	assert.Equal(t, 200, (*normalized[0].Details)["status"])
	assert.Equal(t, "https://example.com", (*normalized[0].Details)["url"])
	assert.Equal(t, 1, (*normalized[0].Details)["attempt"])
}

func TestExportLog_EmptyReturnsNil(t *testing.T) {
	assert.Nil(t, exportLog(nil))
	assert.Nil(t, exportLog(PerformanceLog{}))
}

func TestExportLog_SetsTimestampsByEntryType(t *testing.T) {
	now := time.Unix(2, 0)
	log := PerformanceLog{
		{Name: "mark", Type: MarkType, StartTime: now},
		{Name: "measure", Type: MeasureType, StartTime: now, Duration: time.Second},
	}

	exported := exportLog(log)
	assert.Len(t, exported, 2)
	assert.NotNil(t, exported[0].Timestamp)
	assert.Nil(t, exported[0].StartTimestamp)
	assert.NotNil(t, exported[1].StartTimestamp)
	assert.Nil(t, exported[1].Timestamp)
	assert.Equal(t, int64(time.Second.Nanoseconds()), exported[1].DurationNS)
}

func TestExportToFile_DefaultsOutDirWhenEmpty(t *testing.T) {
	perfDir := t.TempDir()
	previous, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(previous) })
	assert.NoError(t, os.Chdir(perfDir))

	written, err := ExportToFile("", perfDir, PerformanceLog{})
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(".", defaultExportFilename), written)
}

func TestExportToFile_ReturnsErrorWhenOutDirIsAFile(t *testing.T) {
	tempDir := t.TempDir()
	outFile := filepath.Join(tempDir, "not-a-dir")
	assert.NoError(t, os.WriteFile(outFile, []byte("x"), 0644))

	_, err := ExportToFile(outFile, tempDir, PerformanceLog{})
	assert.Error(t, err)
}

func TestExportToFile_ReturnsErrorWhenLogCannotMarshal(t *testing.T) {
	tempDir := t.TempDir()
	log := PerformanceLog{
		{
			Name: "bad",
			Type: MarkType,
			Details: &PerformanceDetails{
				"bad": make(chan int),
			},
		},
	}

	_, err := ExportToFile(tempDir, tempDir, log)
	assert.Error(t, err)
}

func TestNormalizeForExport_HandlesNilAndEmptyDetails(t *testing.T) {
	log := PerformanceLog{
		{Name: "nil", Type: MarkType},
		{Name: "empty", Type: MarkType, Details: &PerformanceDetails{}},
	}

	normalized := normalizeForExport(log, "/base")
	assert.Len(t, normalized, 2)
	assert.Nil(t, normalized[0].Details)
	assert.NotNil(t, normalized[1].Details)
}

func TestNormalizeValue_CleansRelativePaths(t *testing.T) {
	assert.Equal(t, ".", trimLeadingDot("."))
	assert.Equal(t, "mods/a.jar", normalizeValue("path", "./mods/a.jar", "").(string))
	assert.Equal(t, "mods/a.jar", normalizeValue("path", "mods/a.jar", "/base").(string))
	assert.Equal(t, "modlist.json", normalizeValue("config_path", "./modlist.json", "").(string))
	assert.Equal(t, ".", normalizeValue("path", ".", "").(string))
	assert.Equal(t, "not-a-path", normalizeValue("url", "not-a-path", "/base").(string))
}

func TestLooksLikePathKey(t *testing.T) {
	assert.True(t, looksLikePathKey("path"))
	assert.True(t, looksLikePathKey("config_path"))
	assert.True(t, looksLikePathKey("outputPath"))
	assert.False(t, looksLikePathKey("url"))
}
