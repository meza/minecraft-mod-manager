package install

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestWithInstallProgramOptionsReturnsOriginalContextWhenEmpty(t *testing.T) {
	ctx := context.Background()
	result := WithInstallProgramOptions(ctx)
	assert.Equal(t, ctx, result)
}

func TestInstallProgramOptionsFromContextReturnsNilWhenMissing(t *testing.T) {
	assert.Nil(t, installProgramOptionsFromContext(context.Background()))
}

func TestInstallProgramOptionsFromContextReturnsOptions(t *testing.T) {
	marker := false
	option := func(*tea.Program) {
		marker = true
	}
	ctx := WithInstallProgramOptions(context.Background(), option)

	options := installProgramOptionsFromContext(ctx)
	if assert.Len(t, options, 1) {
		options[0](nil)
		assert.True(t, marker)
	}
}
