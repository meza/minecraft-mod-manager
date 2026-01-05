package change

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestBuildChangeSectionsSuccessWithSkipped(t *testing.T) {
	items := []changeItem{
		{Mod: modFixture("alpha"), DisplayName: "Alpha", Skipped: true},
		{Mod: modFixture("beta"), DisplayName: "Beta"},
	}
	sections := buildChangeSections(changeViewInput{
		stage:     changeStageSuccess,
		target:    "1.19.4",
		items:     items,
		colorMode: view.ColorDisabled,
	})
	output := view.RenderViewSections(sections, view.SectionSeparatorParagraph)
	assert.Contains(t, output, "Skipped unsupported mods")
	assert.Contains(t, output, "Now targeting 1.19.4")
}

func TestBuildChangeSectionsCompatibilityFailed(t *testing.T) {
	items := []changeItem{
		{Mod: modFixture("alpha"), DisplayName: "Alpha", CompatStatus: changeCompatUnsupported},
	}
	sections := buildChangeSections(changeViewInput{
		stage:     changeStageCompatibilityFailed,
		target:    "1.19.4",
		items:     items,
		colorMode: view.ColorDisabled,
	})
	output := view.RenderViewSections(sections, view.SectionSeparatorParagraph)
	assert.Contains(t, output, "Compatibility")
	assert.NotContains(t, output, "Downloading")
	assert.NotContains(t, output, "Switching")
	assert.Contains(t, output, "Compatibility check failed for 1.19.4")
}

func TestBuildChangeSectionsCompatibilityFailureRunning(t *testing.T) {
	items := []changeItem{
		{Mod: modFixture("alpha"), DisplayName: "Alpha", CompatStatus: changeCompatUnsupported},
		{Mod: modFixture("beta"), DisplayName: "Beta", CompatStatus: changeCompatChecking},
		{Mod: modFixture("gamma"), DisplayName: "Gamma", CompatStatus: changeCompatSupported},
	}
	sections := buildChangeSections(changeViewInput{
		stage:     changeStageCompatibilityFailureDetected,
		target:    "1.19.4",
		items:     items,
		colorMode: view.ColorDisabled,
	})
	output := view.RenderViewSections(sections, view.SectionSeparatorParagraph)
	assert.Contains(t, output, "Change Minecraft version to 1.19.4")
	assert.Contains(t, output, "Compatibility")
	assert.Contains(t, output, "Downloading: (cancelled due to incompatibility)")
	assert.Contains(t, output, "Switching: (cancelled due to incompatibility)")
	assert.NotContains(t, output, "Gamma (gamma)")
}

func TestBuildChangeSectionsDownloadFailed(t *testing.T) {
	items := []changeItem{
		{Mod: modFixture("alpha"), DisplayName: "Alpha", CompatStatus: changeCompatSupported, DownloadStatus: changeDownloadFailed, ErrorReason: "boom"},
	}
	sections := buildChangeSections(changeViewInput{
		stage:     changeStageDownloadFailed,
		target:    "1.19.4",
		items:     items,
		colorMode: view.ColorDisabled,
	})
	output := view.RenderViewSections(sections, view.SectionSeparatorParagraph)
	assert.Contains(t, output, "Downloads failed")
	assert.Contains(t, output, "download failed: boom")
}

func TestBuildChangeSectionsSwitchFailed(t *testing.T) {
	items := []changeItem{
		{
			Mod:            modFixture("alpha"),
			DisplayName:    "Alpha",
			CompatStatus:   changeCompatSupported,
			DownloadStatus: changeDownloadSucceeded,
			SwitchStatus:   changeSwitchFailed,
			ErrorReason:    "oops",
		},
	}
	sections := buildChangeSections(changeViewInput{
		stage:     changeStageSwitchFailed,
		target:    "1.19.4",
		items:     items,
		colorMode: view.ColorDisabled,
	})
	output := view.RenderViewSections(sections, view.SectionSeparatorParagraph)
	assert.Contains(t, output, "Switching failed")
	assert.Contains(t, output, "switch failed: oops")
}

func TestBuildQuietChangeLinesSuccessNoSkipped(t *testing.T) {
	lines := buildQuietChangeLines(changeOutcome{Stage: changeStageSuccess}, "1.19.4", view.ColorDisabled)
	assert.Empty(t, lines)
}

func TestBuildQuietChangeLinesCompatibilityFailed(t *testing.T) {
	lines := buildQuietChangeLines(changeOutcome{
		Stage: changeStageCompatibilityFailed,
		Items: []changeItem{{Mod: modFixture("alpha"), DisplayName: "Alpha", CompatStatus: changeCompatUnsupported}},
	}, "1.19.4", view.ColorDisabled)
	assert.Len(t, lines, 2)
	assert.Contains(t, lines[0], "Compatibility failed for 1.19.4")
}

func TestBuildQuietChangeLinesDownloadFailed(t *testing.T) {
	lines := buildQuietChangeLines(changeOutcome{
		Stage: changeStageDownloadFailed,
		Items: []changeItem{{Mod: modFixture("alpha"), DisplayName: "Alpha", DownloadStatus: changeDownloadFailed, ErrorReason: "boom"}},
	}, "1.19.4", view.ColorDisabled)
	assert.Len(t, lines, 2)
	assert.Contains(t, lines[0], "Downloads failed")
	assert.Contains(t, lines[1], "download failed: boom")
}

func TestBuildQuietChangeLinesSwitchFailed(t *testing.T) {
	lines := buildQuietChangeLines(changeOutcome{
		Stage: changeStageSwitchFailed,
		Items: []changeItem{{Mod: modFixture("alpha"), DisplayName: "Alpha", SwitchStatus: changeSwitchFailed, ErrorReason: "oops"}},
	}, "1.19.4", view.ColorDisabled)
	assert.Len(t, lines, 2)
	assert.Contains(t, lines[0], "Switching failed")
	assert.Contains(t, lines[1], "switch failed: oops")
}

func TestRenderQuietFailuresEmpty(t *testing.T) {
	empty := renderQuietFailures(nil, view.ColorDisabled, func(changeItem) bool { return true }, func(changeItem) string {
		return "suffix"
	})
	assert.Equal(t, "", empty)
}

func TestRenderQuietFailuresSkipsItems(t *testing.T) {
	items := []changeItem{{Mod: modFixture("alpha"), DisplayName: "Alpha"}}
	rendered := renderQuietFailures(items, view.ColorDisabled, func(changeItem) bool { return false }, func(changeItem) string {
		return "suffix"
	})
	assert.Equal(t, "", rendered)
}

func TestBuildQuietChangeLinesSuccessWithSkipped(t *testing.T) {
	lines := buildQuietChangeLines(changeOutcome{
		Stage: changeStageSuccess,
		Items: []changeItem{{Mod: modFixture("alpha"), DisplayName: "Alpha", Skipped: true}},
	}, "1.19.4", view.ColorDisabled)
	assert.Len(t, lines, 1)
	assert.Contains(t, lines[0], "Skipped unsupported mods")
	assert.Contains(t, lines[0], "unsupported for 1.19.4")
}

func TestRenderQuietSkippedSectionSkipsNonSkippedItems(t *testing.T) {
	rendered := renderQuietSkippedSection([]changeItem{{Mod: modFixture("alpha"), DisplayName: "Alpha"}}, "1.19.4", view.ColorDisabled)
	assert.Equal(t, "Skipped unsupported mods:", rendered)
}

func TestBuildQuietChangeLinesDefaultStage(t *testing.T) {
	lines := buildQuietChangeLines(changeOutcome{Stage: changeStageRunning}, "1.19.4", view.ColorDisabled)
	assert.Empty(t, lines)
}

func TestBuildNonTTYCompatibilityFailureSectionsEmpty(t *testing.T) {
	sections := buildNonTTYCompatibilityFailureSections(changeViewInput{
		target:    "1.19.4",
		items:     nil,
		colorMode: view.ColorDisabled,
	})
	output := view.RenderViewSections(sections, view.SectionSeparatorParagraph)
	assert.Contains(t, output, "Compatibility check failed for 1.19.4")
	assert.Contains(t, output, "No changes were made.")
}

func TestRenderNonTTYCompatibilityLineVariants(t *testing.T) {
	input := changeViewInput{
		target:    "1.19.4",
		colorMode: view.ColorDisabled,
	}

	supported := renderNonTTYCompatibilityLine(input, changeItem{
		Mod:          modFixture("alpha"),
		DisplayName:  "Alpha",
		CompatStatus: changeCompatSupported,
	})
	assert.Contains(t, supported, view.PendingIcon(view.ColorDisabled))

	unsupported := renderNonTTYCompatibilityLine(input, changeItem{
		Mod:          modFixture("beta"),
		DisplayName:  "Beta",
		CompatStatus: changeCompatUnsupported,
	})
	assert.Contains(t, unsupported, view.ErrorIcon(view.ColorDisabled))
	assert.Contains(t, unsupported, "unsupported for 1.19.4")

	pending := renderNonTTYCompatibilityLine(input, changeItem{
		Mod:         modFixture("gamma"),
		DisplayName: "Gamma",
	})
	assert.Contains(t, pending, view.PendingIcon(view.ColorDisabled))
	assert.NotContains(t, pending, "unsupported for 1.19.4")
}

func TestWriteChangeOutcomeUsesOutputLines(t *testing.T) {
	output := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(output)

	deps := changeDeps{runTea: defaultRunTea}

	outcome := changeOutcome{
		Stage: changeStageSuccess,
		Items: []changeItem{{Mod: modFixture("alpha"), DisplayName: "Alpha"}},
	}

	assert.NoError(t, writeChangeOutcome(cmd, deps, outcome, "1.19.4", interaction.ExecutionModeUnattended))
}

func TestWriteChangeOutcomeNonTTYUsesPendingIconForQueued(t *testing.T) {
	output := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(output)

	deps := changeDeps{runTea: defaultRunTea}

	outcome := changeOutcome{
		Stage: changeStageDownloadFailed,
		Items: []changeItem{{
			Mod:            modFixture("alpha"),
			DisplayName:    "Alpha",
			CompatStatus:   changeCompatSupported,
			DownloadStatus: changeDownloadQueued,
		}},
	}

	assert.NoError(t, writeChangeOutcome(cmd, deps, outcome, "1.19.4", interaction.ExecutionModeNonTTY))
	assert.Contains(t, output.String(), "[~] Alpha (alpha) [modrinth]")
}

func TestWriteChangeOutcomeUnattendedCompatibilityFailedFiltersSupported(t *testing.T) {
	output := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(output)

	deps := changeDeps{runTea: defaultRunTea}

	outcome := changeOutcome{
		Stage: changeStageCompatibilityFailed,
		Items: []changeItem{
			{Mod: modFixture("alpha"), DisplayName: "Alpha", CompatStatus: changeCompatUnsupported},
			{Mod: modFixture("beta"), DisplayName: "Beta", CompatStatus: changeCompatSupported},
		},
	}

	assert.NoError(t, writeChangeOutcome(cmd, deps, outcome, "1.19.4", interaction.ExecutionModeUnattended))
	rendered := output.String()
	assert.Contains(t, rendered, "Alpha (alpha)")
	assert.NotContains(t, rendered, "Beta (beta)")
}

func TestWriteChangeOutcomeNonTTYCompatibilityFailedUsesFlatList(t *testing.T) {
	output := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(output)

	deps := changeDeps{runTea: defaultRunTea}

	items := []changeItem{
		{Mod: modFixture("alpha"), DisplayName: "Alpha", CompatStatus: changeCompatUnsupported},
		{Mod: modFixture("beta"), DisplayName: "Beta", CompatStatus: changeCompatPending},
	}
	outcome := changeOutcome{
		Stage: changeStageCompatibilityFailed,
		Items: items,
	}

	assert.NoError(t, writeChangeOutcome(cmd, deps, outcome, "1.19.4", interaction.ExecutionModeNonTTY))
	rendered := output.String()
	assert.NotContains(t, rendered, "Compatibility:")
	assert.NotContains(t, rendered, "Downloading:")
	assert.NotContains(t, rendered, "Switching:")
	assert.Contains(t, rendered, "unsupported for 1.19.4")
	assert.NotContains(t, rendered, "Beta (beta)")
	assert.Contains(t, rendered, "Compatibility check failed for 1.19.4")
	assert.Contains(t, rendered, "No changes were made.")
}

func TestWriteChangeOutcomeUnattendedCompatibilityFailedKeepsSections(t *testing.T) {
	output := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(output)

	deps := changeDeps{runTea: defaultRunTea}

	items := []changeItem{
		{Mod: modFixture("alpha"), DisplayName: "Alpha", CompatStatus: changeCompatUnsupported},
	}
	outcome := changeOutcome{
		Stage: changeStageCompatibilityFailed,
		Items: items,
	}

	assert.NoError(t, writeChangeOutcome(cmd, deps, outcome, "1.19.4", interaction.ExecutionModeUnattended))
	rendered := output.String()
	assert.Contains(t, rendered, "Compatibility:")
	assert.NotContains(t, rendered, "Downloading:")
	assert.NotContains(t, rendered, "Switching:")
}

func TestDefaultSpinnerFrameReturnsEmptyWhenMissingFrames(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	originalFrames := spinner.Line.Frames
	t.Cleanup(func() { spinner.Line.Frames = originalFrames })
	spinner.Line.Frames = []string{}

	assert.Equal(t, "", defaultSpinnerFrame())
}

func TestWriteQuietChangeOutcomeUsesOutputLines(t *testing.T) {
	output := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(output)

	deps := changeDeps{runTea: defaultRunTea}

	outcome := changeOutcome{
		Stage: changeStageDownloadFailed,
		Items: []changeItem{{Mod: modFixture("alpha"), DisplayName: "Alpha", DownloadStatus: changeDownloadFailed, ErrorReason: "boom"}},
	}

	assert.NoError(t, writeQuietChangeOutcome(cmd, deps, outcome, "1.19.4"))
}

func TestDefaultSpinnerFrameUsesUnicodeWhenAvailable(t *testing.T) {
	restore := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restore)

	frame := defaultSpinnerFrame()
	assert.NotEmpty(t, frame)
}

func TestDefaultSpinnerFrameUsesAsciiWhenUnicodeUnavailable(t *testing.T) {
	restore := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restore)

	frame := defaultSpinnerFrame()
	assert.NotEmpty(t, frame)
}

func modFixture(id string) models.Mod {
	name := strings.ToUpper(id)
	return models.Mod{ID: id, Name: name, Type: models.MODRINTH}
}
