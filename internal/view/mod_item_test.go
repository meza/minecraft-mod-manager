package view

import (
	"testing"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/stretchr/testify/assert"
)

func TestRenderModLabel(t *testing.T) {
	label := RenderModLabel(ColorDisabled, "Example", "abc", "modrinth")
	assert.Equal(t, "Example (abc) [modrinth]", label)
}

func TestRenderModItemLineAddsSuffix(t *testing.T) {
	restore := SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restore)

	label := RenderModLabel(ColorDisabled, "Example", "abc", "modrinth")
	line := RenderModItemLine(ModItemLine{
		Label:  label,
		Suffix: "not installed",
		Status: ModItemStatusError,
	}, ColorDisabled)

	assert.Contains(t, line, "X Example (abc) [modrinth] not installed")
}

func TestRenderModItemLineAddsProgressLines(t *testing.T) {
	bar := progress.New(progress.WithWidth(10), progress.WithoutPercentage())
	line := RenderModItemLine(ModItemLine{
		Label:  "Example (abc) [modrinth]",
		Status: ModItemStatusDownloading,
		Progress: &ProgressDetails{
			Bar:        bar,
			Ratio:      0.5,
			Downloaded: 512,
			Total:      1024,
		},
	}, ColorDisabled)

	assert.Contains(t, line, "Example (abc) [modrinth]")
	assert.Contains(t, line, "50% (512 B / 1 KB)")
}

func TestRenderModItemLineUsesPendingIcon(t *testing.T) {
	restore := SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restore)

	line := RenderModItemLine(ModItemLine{
		Label:  "Example (abc) [modrinth]",
		Status: ModItemStatusPending,
	}, ColorDisabled)

	assert.Contains(t, line, "[~] Example (abc) [modrinth]")
}

func TestRenderModItemLineUsesDownloadIcon(t *testing.T) {
	restore := SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restore)

	line := RenderModItemLine(ModItemLine{
		Label:  "Example (abc) [modrinth]",
		Status: ModItemStatusDownloading,
	}, ColorDisabled)

	assert.Contains(t, line, "-> Example (abc) [modrinth]")
}

func TestRenderModItemLineUsesSpinnerFrame(t *testing.T) {
	line := RenderModItemLine(ModItemLine{
		Label:        "Example (abc) [modrinth]",
		Status:       ModItemStatusSpinning,
		SpinnerFrame: ".",
	}, ColorDisabled)

	assert.Contains(t, line, ". Example (abc) [modrinth]")
}

func TestRenderModItemLineUsesPendingIconWhenSpinnerEmpty(t *testing.T) {
	restore := SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restore)

	line := RenderModItemLine(ModItemLine{
		Label:        "Example (abc) [modrinth]",
		Status:       ModItemStatusSpinning,
		SpinnerFrame: " ",
	}, ColorDisabled)

	assert.Contains(t, line, "[~] Example (abc) [modrinth]")
}

func TestRenderModItemLineUsesErrorIconForSkipped(t *testing.T) {
	restore := SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restore)

	line := RenderModItemLine(ModItemLine{
		Label:  "Example (abc) [modrinth]",
		Status: ModItemStatusSkipped,
	}, ColorDisabled)

	assert.Contains(t, line, "X Example (abc) [modrinth]")
}

func TestRenderModItemLineUsesSuccessIcon(t *testing.T) {
	restore := SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restore)

	line := RenderModItemLine(ModItemLine{
		Label:  "Example (abc) [modrinth]",
		Status: ModItemStatusSuccess,
	}, ColorDisabled)

	assert.Contains(t, line, "V Example (abc) [modrinth]")
}
