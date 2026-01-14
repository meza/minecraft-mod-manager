package add

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestRenderAddSuccessLine(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)
	line := renderAddSuccessLine(view.ColorDisabled, "Example", "abc", models.MODRINTH)
	assert.Contains(t, line, "cmd.mod.display")
	assert.Contains(t, line, "\u2705")
}

func TestRenderFinalErrorLine(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)
	line := renderFinalErrorLine(view.ColorDisabled, "boom")
	assert.True(t, strings.HasPrefix(line, "\u203C\uFE0F"))
	assert.Contains(t, line, "boom")
}

func TestRenderFinalErrorLineColorEnabled(t *testing.T) {
	line := renderFinalErrorLine(view.ColorEnabled, "boom")
	assert.Contains(t, line, "boom")
}

func TestSummaryMessages(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	assert.Contains(t, projectNotFoundSummary(models.MODRINTH, "abc"), "cmd.add.error.not_found")
	assert.Contains(t, projectNotFoundUnattendedSummary(models.MODRINTH, "abc"), "cmd.add.error.not_found_unattended")
	assert.Contains(t, projectNotFoundHint(models.MODRINTH, "abc"), "cmd.add.error.not_found_hint")
	assert.Contains(t, noCompatibleSummary("fabric", "1.20.1"), "cmd.add.error.no_compatible")
	assert.Contains(t, downloadFailedSummary(models.MODRINTH, "abc"), "cmd.add.error.download_failed")
}

func TestDownloadFailedDetails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	details := downloadFailedDetails(models.MODRINTH, 3)
	assert.Contains(t, details, "cmd.add.error.download_failed_retries")
	assert.Contains(t, details, "cmd.add.error.download_failed_platform")
	assert.Contains(t, details, "cmd.add.error.download_failed_network")
}
