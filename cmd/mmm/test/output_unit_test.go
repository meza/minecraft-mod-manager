package test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestBuildTestSectionsSuccessSummary(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	sections := buildTestSections(testViewInput{
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items: []testItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, Status: testItemStatusSupported},
		},
	})

	output := view.RenderViewSections(sections, view.SectionSeparatorParagraph)
	assert.Contains(t, output, "cmd.test.header")
	assert.Contains(t, output, "cmd.test.section.compatible")
	assert.Contains(t, output, "cmd.test.success")
	assert.NotContains(t, output, "cmd.test.summary.inconclusive")
}

func TestBuildTestSectionsUnsupportedSummary(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	sections := buildTestSections(testViewInput{
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items: []testItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, Status: testItemStatusUnsupported},
		},
	})

	output := view.RenderViewSections(sections, view.SectionSeparatorParagraph)
	assert.Contains(t, output, "cmd.test.section.not_compatible")
	assert.Contains(t, output, "cmd.test.summary.unsupported")
}

func TestBuildTestSectionsInconclusiveSummary(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	sections := buildTestSections(testViewInput{
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items: []testItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, Status: testItemStatusInconclusive},
		},
	})

	output := view.RenderViewSections(sections, view.SectionSeparatorParagraph)
	assert.Contains(t, output, "cmd.test.section.inconclusive")
	assert.Contains(t, output, "cmd.test.summary.inconclusive")
	assert.Contains(t, output, "cmd.test.summary.inconclusive_hint")
}

func TestBuildTestSectionsInconclusiveSummaryColorEnabled(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	sections := buildTestSections(testViewInput{
		targetVersion: "1.20.1",
		colorMode:     view.ColorEnabled,
		items: []testItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, Status: testItemStatusInconclusive},
		},
	})

	output := view.RenderViewSections(sections, view.SectionSeparatorParagraph)
	assert.Contains(t, output, "cmd.test.summary.inconclusive_hint")
}

func TestRenderTestRunningViewIncludesSpinner(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	spin := view.NewSpinner()
	frame := spin.Frame()
	output := renderTestRunningView(testViewInput{
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		spinner:       &spin,
		items: []testItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, Status: testItemStatusChecking},
		},
	})

	assert.Contains(t, output, "cmd.compatibility.section")
	if frame != "" {
		assert.Contains(t, output, frame+" Alpha (alpha) [modrinth]")
	}
}

func TestBuildQuietTestLinesOmitsReason(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	lines := buildQuietTestLines([]testItem{
		{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, Status: testItemStatusUnsupported},
		{Mod: models.Mod{ID: "beta", Name: "Beta", Type: models.MODRINTH}, Status: testItemStatusInconclusive, Reason: "timeout"},
	}, view.ColorDisabled)

	assert.Equal(t, []string{
		"❌ Alpha (alpha) [modrinth]",
		"❔ Beta (beta) [modrinth]",
	}, lines)
}

func TestRenderTestRunningItemLineCoversStatuses(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	spin := view.NewSpinner()
	input := testViewInput{
		colorMode: view.ColorDisabled,
		spinner:   &spin,
	}
	mod := models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}

	output := renderTestRunningItemLine(input, testItem{Mod: mod, Status: testItemStatusSupported})
	assert.Contains(t, output, "Alpha (alpha) [modrinth]")

	output = renderTestRunningItemLine(input, testItem{Mod: mod, Status: testItemStatusUnsupported})
	assert.Contains(t, output, "Alpha (alpha) [modrinth]")

	output = renderTestRunningItemLine(input, testItem{Mod: mod, Status: testItemStatusInconclusive})
	assert.Contains(t, output, "Alpha (alpha) [modrinth]")

	output = renderTestRunningItemLine(input, testItem{Mod: mod, Status: testItemStatusChecking})
	frame := spin.Frame()
	if frame != "" {
		assert.Contains(t, output, frame+" Alpha (alpha) [modrinth]")
	}
}

func TestRenderTestItemLinePendingAndUnsupported(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	mod := models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}

	output := renderTestItemLine(view.ColorDisabled, testItem{Mod: mod, Status: testItemStatusChecking})
	assert.Contains(t, output, "Alpha (alpha) [modrinth]")

	output = renderTestItemLine(view.ColorDisabled, testItem{Mod: mod, Status: testItemStatusUnsupported})
	assert.Contains(t, output, "Alpha (alpha) [modrinth]")
}

func TestRenderTestItemLineWithReasonFallsBack(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	mod := models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}
	item := testItem{Mod: mod, Status: testItemStatusInconclusive}

	output := renderTestItemLineWithReason(view.ColorDisabled, item)
	assert.Equal(t, renderTestItemLine(view.ColorDisabled, item), output)
}

func TestRenderFinalErrorLineColorEnabled(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	output := renderFinalErrorLine(view.ColorEnabled, "boom")
	assert.Contains(t, output, "boom")
}

func TestPruneEmptySectionsRemovesBlankEntries(t *testing.T) {
	filtered := pruneEmptySections([]string{"", "Alpha", "  ", "Beta"})
	assert.Equal(t, []string{"Alpha", "Beta"}, filtered)
}
