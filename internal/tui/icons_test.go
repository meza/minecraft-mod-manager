package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSuccessIconIsUnstyledWhenNotColorized(t *testing.T) {
	assert.Equal(t, "✅", SuccessIcon(false))
}

func TestErrorIconIsUnstyledWhenNotColorized(t *testing.T) {
	assert.Equal(t, "❌", ErrorIcon(false))
}

func TestSuccessIconIsStyledWhenColorized(t *testing.T) {
	assert.Equal(t, QuestionStyle.Render("✅"), SuccessIcon(true))
}

func TestErrorIconIsStyledWhenColorized(t *testing.T) {
	assert.Equal(t, ErrorStyle.Render("❌"), ErrorIcon(true))
}
