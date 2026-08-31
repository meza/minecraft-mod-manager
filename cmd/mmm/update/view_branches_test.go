package update

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/cmd/mmm/install"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestUpdateErrorViewsRenderHints(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	installView := renderUpdateInstallFailureView(view.ColorEnabled)
	assert.Contains(t, installView, "cmd.update.error.install_failed")
	assert.Contains(t, installView, "cmd.update.error.install_failed_hint")

	lockView := renderUpdateWriteLockFailureView(updateErrorViewInput{
		colorMode: view.ColorEnabled,
		lockPath:  "/lock",
	})
	assert.Contains(t, lockView, "cmd.update.error.write_lock")
	assert.Contains(t, lockView, "cmd.update.error.write_hint")

	configView := renderUpdateWriteConfigFailureView(updateErrorViewInput{
		colorMode:  view.ColorEnabled,
		configPath: "/config",
	})
	assert.Contains(t, configView, "cmd.update.error.write_config")
	assert.Contains(t, configView, "cmd.update.error.write_hint")
}

func TestRenderFinalErrorLineColorModes(t *testing.T) {
	line := renderFinalErrorLine(view.ColorDisabled, "message")
	assert.Contains(t, line, "message")

	line = renderFinalErrorLine(view.ColorEnabled, "message")
	assert.Contains(t, line, "message")
}

func TestRenderUpdateSummarySectionsFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	lines := renderUpdateSummarySections(updateResultsViewInput{
		items: []updateItem{
			{ConfigIndex: 0, Status: updateItemStatusFailed, FailReason: "boom"},
		},
		colorMode: view.ColorEnabled,
	})
	assert.Equal(t, 2, len(lines))
	assert.Contains(t, lines[0], "cmd.update.summary.incomplete")
	assert.Contains(t, lines[1], "cmd.update.summary.incomplete_hint")
}

func TestRenderUpdateItemIconBranches(t *testing.T) {
	input := updateRunningViewInput{
		colorMode:    view.ColorDisabled,
		spinnerFrame: "⠋",
	}

	assert.Equal(t, view.SuccessIcon(view.ColorDisabled), renderUpdateItemIcon(input, updateItem{Status: updateItemStatusUpdated}))
	assert.Equal(t, view.SuccessIcon(view.ColorDisabled), renderUpdateItemIcon(input, updateItem{Status: updateItemStatusUpToDate}))
	assert.Equal(t, view.ErrorIcon(view.ColorDisabled), renderUpdateItemIcon(input, updateItem{Status: updateItemStatusFailed}))
	assert.Equal(t, view.PinnedIcon(view.ColorDisabled), renderUpdateItemIcon(input, updateItem{Status: updateItemStatusSkipped}))
	assert.Equal(t, view.DownloadIcon(view.ColorDisabled), renderUpdateItemIcon(input, updateItem{Status: updateItemStatusDownloading}))
	assert.Equal(t, input.spinnerFrame, renderUpdateItemIcon(input, updateItem{Status: updateItemStatusUpdating}))
	assert.Equal(t, view.PendingIcon(view.ColorDisabled), renderUpdateItemIcon(input, updateItem{Status: updateItemStatusPending}))

	input.spinnerFrame = ""
	assert.Equal(t, view.PendingIcon(view.ColorDisabled), renderUpdateItemIcon(input, updateItem{Status: updateItemStatusUpdating}))
	assert.Equal(t, view.PendingIcon(view.ColorDisabled), renderUpdateItemIcon(input, updateItem{Status: updateItemStatus(99)}))
}

func TestRenderUpdateSummarySectionsSuccess(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	lines := renderUpdateSummarySections(updateResultsViewInput{
		items:     []updateItem{{ConfigIndex: 0, Status: updateItemStatusUpToDate}},
		colorMode: view.ColorDisabled,
	})
	assert.Equal(t, 1, len(lines))
	assert.Contains(t, lines[0], "cmd.update.summary.success")
}

func TestPrependUpdateHeaderWhenSectionsEmpty(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	sections := prependUpdateHeader(view.ColorDisabled, nil)
	if assert.Len(t, sections, 1) {
		assert.Contains(t, sections[0], "cmd.update.header")
	}
}

func TestPrependUpdateHeaderWhenSectionsPresent(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	sections := prependUpdateHeader(view.ColorDisabled, []string{"section"})
	if assert.Len(t, sections, 1) {
		assert.Contains(t, sections[0], "cmd.update.header")
		assert.Contains(t, sections[0], "section")
	}
}

func TestRenderUpdateInstallWaitingLine(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	line := renderUpdateInstallWaitingLine(view.ColorEnabled, spinner.Dot.Frames[0])
	assert.Contains(t, line, "cmd.update.section.updating_label")
	assert.Contains(t, line, "cmd.update.section.updating_waiting")
	assert.Contains(t, line, spinner.Dot.Frames[0])
	assert.Contains(t, line, "(")
}

func TestRenderUpdateInstallWaitingLineNoUnicode(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	line := renderUpdateInstallWaitingLine(view.ColorDisabled, spinner.Line.Frames[0])
	assert.Contains(t, line, "cmd.update.section.updating_label")
	assert.Contains(t, line, "cmd.update.section.updating_waiting")
	assert.Contains(t, line, spinner.Line.Frames[0])
	assert.Contains(t, line, "(")
}

func TestUpdateInstallRunningFooter(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	line := updateInstallRunningFooter(install.RunningFooterInput{
		ColorMode:    view.ColorDisabled,
		SpinnerFrame: ".",
	})
	assert.Contains(t, line, "cmd.update.section.updating_label")
	assert.Contains(t, line, "cmd.update.section.updating_waiting")
	assert.Contains(t, line, "(")
}

func TestRenderUpdateTranscriptSummaryOmitsSections(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	lines := renderUpdateTranscriptSummary(view.ColorDisabled, []updateItem{
		{ConfigIndex: 0, Status: updateItemStatusFailed, FailReason: "boom"},
	})
	assert.Equal(t, 3, len(lines))
	assert.Equal(t, "", lines[1])
	assert.Contains(t, strings.Join(lines, "\n"), "cmd.update.summary.incomplete")
	assert.NotContains(t, strings.Join(lines, "\n"), "cmd.update.section.failed")
}

func TestRenderUpdateTranscriptSummarySuccess(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	lines := renderUpdateTranscriptSummary(view.ColorDisabled, []updateItem{
		{ConfigIndex: 0, Status: updateItemStatusUpToDate},
	})
	assert.Equal(t, 1, len(lines))
	assert.Contains(t, lines[0], "cmd.update.summary.success")
}

func TestRenderUpdateTranscriptSummaryFailureColorEnabled(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	lines := renderUpdateTranscriptSummary(view.ColorEnabled, []updateItem{
		{ConfigIndex: 0, Status: updateItemStatusFailed},
	})
	assert.Equal(t, 3, len(lines))
}

func TestUpdateFailureReasonsAvoidRepeatingModName(t *testing.T) {
	originalValue, hadValue := os.LookupEnv("MMM_TEST")
	require.NoError(t, os.Unsetenv("MMM_TEST"))
	t.Cleanup(func() {
		if hadValue {
			require.NoError(t, os.Setenv("MMM_TEST", originalValue))
		} else {
			require.NoError(t, os.Unsetenv("MMM_TEST"))
		}
	})

	t.Setenv("LANG", "en_GB")
	i18n.ResetForTesting()
	t.Cleanup(i18n.ResetForTesting)

	mod := models.Mod{ID: "mod-id", Name: "Armor Poser", Type: models.MODRINTH}
	reasons := []string{
		i18n.T("cmd.update.error.no_file", &i18n.Tvars{
			Data: &i18n.TData{"platform": string(mod.Type)},
		}),
		i18n.T("cmd.update.error.invalid_filename_lock", &i18n.Tvars{
			Data: &i18n.TData{"file": "bad.jar"},
		}),
	}

	for _, reason := range reasons {
		item := updateItem{
			ConfigIndex: 0,
			Mod:         mod,
			DisplayName: mod.Name,
			Status:      updateItemStatusFailed,
			FailReason:  reason,
		}
		line := renderUpdateItemLine(updateRunningViewInput{
			items:     []updateItem{item},
			colorMode: view.ColorDisabled,
		}, item)

		assert.Equal(t, 1, strings.Count(line, mod.Name))
		assert.NotContains(t, reason, "Skipping")
	}
}
