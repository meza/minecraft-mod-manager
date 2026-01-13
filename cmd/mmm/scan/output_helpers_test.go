package scan

import (
	"bytes"
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestRenderScanFailureLineColorEnabled(t *testing.T) {
	line := renderScanFailureLine(view.ColorEnabled, errors.New("boom"))
	assert.Contains(t, line, "boom")
}

func TestRenderScanRunningItemLineStatuses(t *testing.T) {
	spin := view.NewSpinner()
	input := scanRunningViewInput{
		colorMode: view.ColorDisabled,
		spinner:   &spin,
	}

	scanning := renderScanRunningItemLine(input, scanItem{FileName: "alpha.jar", Status: scanItemStatusScanning})
	assert.Contains(t, scanning, "alpha.jar")

	pending := renderScanRunningItemLine(input, scanItem{FileName: "beta.jar", Status: scanItemStatusPending})
	assert.Contains(t, pending, "beta.jar")

	defaultLine := renderScanRunningItemLine(input, scanItem{FileName: "gamma.jar", Status: scanItemStatusUnknown})
	assert.Contains(t, defaultLine, "gamma.jar")
}

func TestSortScanItemsByFileCaseInsensitive(t *testing.T) {
	items := []scanItem{
		{FileName: "beta.jar"},
		{FileName: "Alpha.jar"},
		{FileName: "alpha.jar"},
	}
	sortScanItemsByFile(items)
	assert.Equal(t, "Alpha.jar", items[0].FileName)
	assert.Equal(t, "alpha.jar", items[1].FileName)
	assert.Equal(t, "beta.jar", items[2].FileName)
}

func TestSortScanCandidatesByFileCaseInsensitive(t *testing.T) {
	candidates := []scanCandidate{
		{FileName: "beta.jar"},
		{FileName: "Alpha.jar"},
		{FileName: "alpha.jar"},
	}
	sortScanCandidatesByFile(candidates)
	assert.Equal(t, "Alpha.jar", candidates[0].FileName)
	assert.Equal(t, "alpha.jar", candidates[1].FileName)
	assert.Equal(t, "beta.jar", candidates[2].FileName)
}

func TestSortScanMatchesByNameBreaksTies(t *testing.T) {
	items := []scanMatch{
		{Name: "Alpha", Platform: models.MODRINTH, ProjectID: "b", FileName: "b.jar"},
		{Name: "Alpha", Platform: models.CURSEFORGE, ProjectID: "a", FileName: "a.jar"},
		{Name: "alpha", Platform: models.MODRINTH, ProjectID: "a", FileName: "c.jar"},
		{Name: "alpha", Platform: models.MODRINTH, ProjectID: "a", FileName: "a.jar"},
	}
	sortScanMatchesByName(items)
	assert.Equal(t, models.CURSEFORGE, items[0].Platform)
	assert.Equal(t, "a", items[0].ProjectID)
	assert.Equal(t, "a.jar", items[1].FileName)
	assert.Equal(t, "c.jar", items[2].FileName)
}

func TestSortScanMatchesByFileCaseInsensitive(t *testing.T) {
	items := []scanMatch{
		{FileName: "beta.jar"},
		{FileName: "Alpha.jar"},
		{FileName: "alpha.jar"},
	}
	sortScanMatchesByFile(items)
	assert.Equal(t, "Alpha.jar", items[0].FileName)
	assert.Equal(t, "alpha.jar", items[1].FileName)
	assert.Equal(t, "beta.jar", items[2].FileName)
}

func TestSortScanUnsureByPathCaseInsensitive(t *testing.T) {
	items := []scanUnsure{
		{Path: "beta.jar"},
		{Path: "Alpha.jar"},
		{Path: "alpha.jar"},
	}
	sortScanUnsureByPath(items)
	assert.Equal(t, "Alpha.jar", items[0].Path)
	assert.Equal(t, "alpha.jar", items[1].Path)
	assert.Equal(t, "beta.jar", items[2].Path)
}

func TestSortStringsCaseInsensitive(t *testing.T) {
	items := []string{"beta.jar", "Alpha.jar", "alpha.jar"}
	sortStringsCaseInsensitive(items)
	assert.Equal(t, "Alpha.jar", items[0])
	assert.Equal(t, "alpha.jar", items[1])
	assert.Equal(t, "beta.jar", items[2])
}

func TestFilterScanItemsWithNoStatusesReturnsNil(t *testing.T) {
	items := []scanItem{{FileName: "alpha.jar"}}
	assert.Nil(t, filterScanItems(items))
}

func TestFilterScanItemsByStatus(t *testing.T) {
	items := []scanItem{
		{FileName: "alpha.jar", Status: scanItemStatusUnknown},
		{FileName: "beta.jar", Status: scanItemStatusRecognized},
		{FileName: "gamma.jar", Status: scanItemStatusPending},
	}
	filtered := filterScanItems(items, scanItemStatusUnknown, scanItemStatusRecognized)
	assert.Len(t, filtered, 2)
}

func TestPruneEmptySections(t *testing.T) {
	sections := []string{"one", "", "  ", "two"}
	assert.Equal(t, []string{"one", "two"}, pruneEmptySections(sections))
}

func TestRenderScanTranscriptLine(t *testing.T) {
	match := scanMatch{Name: "Alpha", ProjectID: "alpha", Platform: models.MODRINTH}
	recognized := renderScanTranscriptLine(view.ColorDisabled, scanItem{Status: scanItemStatusRecognized, Match: match})
	assert.Contains(t, recognized, "Alpha")

	unknown := renderScanTranscriptLine(view.ColorDisabled, scanItem{Status: scanItemStatusUnknown, FileName: "beta.jar"})
	assert.Contains(t, unknown, "beta.jar")

	unsure := renderScanTranscriptLine(view.ColorDisabled, scanItem{Status: scanItemStatusUnsure, FileName: "gamma.jar"})
	assert.Contains(t, unsure, "gamma.jar")

	pending := renderScanTranscriptLine(view.ColorDisabled, scanItem{Status: scanItemStatusPending})
	assert.Equal(t, "", pending)
}

func TestOutputProgramOptionsUsesCmdOutput(t *testing.T) {
	cmd := &cobra.Command{}
	writer := &bytes.Buffer{}
	cmd.SetOut(writer)

	options := outputProgramOptions(cmd, nil)
	assert.Len(t, options, 3)
}

func TestOutputProgramOptionsUsesWriterWhenProvided(t *testing.T) {
	cmd := &cobra.Command{}
	writer := &bytes.Buffer{}

	options := outputProgramOptions(cmd, writer)
	assert.Len(t, options, 3)
}
