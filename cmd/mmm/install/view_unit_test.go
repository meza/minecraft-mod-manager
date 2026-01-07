package install

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/muesli/termenv"
)

func TestRenderFinalErrorLineRespectsColorMode(t *testing.T) {
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(func() {
		restoreColor()
		restoreTerminal()
	})

	line := renderFinalErrorLine(view.ColorEnabled, "boom")
	assert.Contains(t, line, "boom")

	line = renderFinalErrorLine(view.ColorDisabled, "boom")
	assert.Contains(t, line, "!!")
}

func TestRenderInstallExecutionFailureSummaryIncludesReasonAndHint(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	lines := renderInstallExecutionFailureSummary(view.ColorDisabled, errors.New("boom"))
	combined := strings.Join(lines, "\n")
	assert.Contains(t, combined, "cmd.install.error.failed")
	assert.Contains(t, combined, "cmd.install.error.failed_hint")
}

func TestRenderInstallExecutionFailureSummaryColorizesHint(t *testing.T) {
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(func() {
		restoreColor()
		restoreTerminal()
	})

	t.Setenv("MMM_TEST", "true")
	lines := renderInstallExecutionFailureSummary(view.ColorEnabled, errors.New("boom"))
	assert.Contains(t, lines[1], "cmd.install.error.failed_hint")
}

func TestRenderInstallSummaryHintsUseCtaStyleWhenColorEnabled(t *testing.T) {
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(func() {
		restoreColor()
		restoreTerminal()
	})

	t.Setenv("MMM_TEST", "true")

	lines := renderInstallDownloadFailureSummaryWithHint(view.ColorEnabled)
	assert.Contains(t, lines[len(lines)-1], "cmd.install.summary.download_failed_hint")

	lines = renderInstallWriteLockSummary(view.ColorEnabled, "/cfg/modlist-lock.json")
	assert.Contains(t, lines[len(lines)-1], "cmd.install.summary.write_failed_hint")

	lines = renderInstallWriteConfigSummary(view.ColorEnabled, "/cfg/modlist.json")
	assert.Contains(t, lines[len(lines)-1], "cmd.install.summary.write_failed_hint")
}
