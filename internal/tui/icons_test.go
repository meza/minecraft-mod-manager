package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSuccessIconIsUnstyledWhenNotColorized(t *testing.T) {
	assert.Equal(t, "V", SuccessIcon(ColorDisabled))
}

func TestErrorIconIsUnstyledWhenNotColorized(t *testing.T) {
	assert.Equal(t, "X", ErrorIcon(ColorDisabled))
}

func TestSuccessIconIsStyledWhenColorized(t *testing.T) {
	assert.Equal(t, QuestionStyle.Render("\u2705"), SuccessIcon(ColorEnabled))
}

func TestErrorIconIsStyledWhenColorized(t *testing.T) {
	assert.Equal(t, ErrorStyle.Render("\u274C"), ErrorIcon(ColorEnabled))
}
