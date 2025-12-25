package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSuccessIconIsUnstyledWhenNotColorized(t *testing.T) {
	assert.Equal(t, "✅", SuccessIcon(ColorDisabled))
}

func TestErrorIconIsUnstyledWhenNotColorized(t *testing.T) {
	assert.Equal(t, "❌", ErrorIcon(ColorDisabled))
}

func TestSuccessIconIsStyledWhenColorized(t *testing.T) {
	assert.Equal(t, QuestionStyle.Render("✅"), SuccessIcon(ColorEnabled))
}

func TestErrorIconIsStyledWhenColorized(t *testing.T) {
	assert.Equal(t, ErrorStyle.Render("❌"), ErrorIcon(ColorEnabled))
}
