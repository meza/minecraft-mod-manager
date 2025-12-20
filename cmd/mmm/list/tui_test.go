package list

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/perf"
)

func TestModelInitReturnsQuit(t *testing.T) {
	model := newModel("view", nil)
	cmd := model.Init()
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestModelUpdateNonKeyMessageQuits(t *testing.T) {
	model := newModel("view", nil)
	_, cmd := model.Update(struct{}{})
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestModelUpdateKeyMessageQuitsWithoutSpan(t *testing.T) {
	model := newModel("view", nil)
	_, cmd := model.Update(tea.KeyMsg{})
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestModelUpdateKeyMessageQuitsWithSpan(t *testing.T) {
	perf.Reset()
	t.Cleanup(perf.Reset)

	ctx := context.Background()
	_, span := perf.StartSpan(ctx, "tui.list.test")
	t.Cleanup(span.End)

	model := newModel("view", span)
	_, cmd := model.Update(tea.KeyMsg{})
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestModelUpdateNonKeyMessageQuitsWithSpan(t *testing.T) {
	perf.Reset()
	t.Cleanup(perf.Reset)

	ctx := context.Background()
	_, span := perf.StartSpan(ctx, "tui.list.test")
	t.Cleanup(span.End)

	model := newModel("view", span)
	_, cmd := model.Update(struct{}{})
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}
