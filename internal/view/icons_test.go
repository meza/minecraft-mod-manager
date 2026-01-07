package view

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSuccessIconIsUnstyledWhenNotColorized(t *testing.T) {
	restore := SetUnicodeSupportFuncForTesting(func() bool { return true })
	defer restore()

	assert.Equal(t, "\u2705", SuccessIcon(ColorDisabled))
}

func TestErrorIconIsUnstyledWhenNotColorized(t *testing.T) {
	restore := SetUnicodeSupportFuncForTesting(func() bool { return true })
	defer restore()

	assert.Equal(t, "\u274C", ErrorIcon(ColorDisabled))
}

func TestFinalErrorIconIsUnstyledWhenNotColorized(t *testing.T) {
	restore := SetUnicodeSupportFuncForTesting(func() bool { return true })
	defer restore()

	assert.Equal(t, "\u203C\uFE0F", FinalErrorIcon(ColorDisabled))
}

func TestSuccessIconIsStyledWhenColorized(t *testing.T) {
	restore := SetUnicodeSupportFuncForTesting(func() bool { return true })
	defer restore()

	assert.Equal(t, QuestionStyle.Render("\u2705"), SuccessIcon(ColorEnabled))
}

func TestErrorIconIsStyledWhenColorized(t *testing.T) {
	restore := SetUnicodeSupportFuncForTesting(func() bool { return true })
	defer restore()

	assert.Equal(t, ErrorStyle.Render("\u274C"), ErrorIcon(ColorEnabled))
}

func TestFinalErrorIconIsStyledWhenColorized(t *testing.T) {
	restore := SetUnicodeSupportFuncForTesting(func() bool { return true })
	defer restore()

	assert.Equal(t, ErrorStyle.Render("\u203C\uFE0F"), FinalErrorIcon(ColorEnabled))
}

func TestIconsFallbackToAsciiWhenUnicodeUnsupported(t *testing.T) {
	restore := SetUnicodeSupportFuncForTesting(func() bool { return false })
	defer restore()

	assert.Equal(t, "V", SuccessIcon(ColorDisabled))
	assert.Equal(t, "X", ErrorIcon(ColorDisabled))
	assert.Equal(t, "!!", FinalErrorIcon(ColorDisabled))
	assert.Equal(t, "[~]", PendingIcon(ColorDisabled))
	assert.Equal(t, "->", DownloadIcon(ColorDisabled))
}

func TestPendingIconUsesUnicodeWhenAvailable(t *testing.T) {
	restore := SetUnicodeSupportFuncForTesting(func() bool { return true })
	defer restore()

	assert.Equal(t, QuestionStyle.Render("\u23F3"), PendingIcon(ColorEnabled))
}

func TestDownloadIconUsesUnicodeWhenAvailable(t *testing.T) {
	restore := SetUnicodeSupportFuncForTesting(func() bool { return true })
	defer restore()

	assert.Equal(t, QuestionStyle.Render("\u2B07\uFE0F"), DownloadIcon(ColorEnabled))
}
