package change

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestConfirmPromptDefaultsToNo(t *testing.T) {
	model := newConfirmPromptModel("Question?")
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	selected, ok := msg.(confirmSelectedMessage)
	assert.True(t, ok)
	assert.False(t, selected.confirmed)
	assert.NotEmpty(t, updated.value)
}

func TestConfirmPromptInitReturnsNil(t *testing.T) {
	model := newConfirmPromptModel("Question?")
	assert.Nil(t, model.Init())
}

func TestConfirmPromptAcceptsYesShort(t *testing.T) {
	model := newConfirmPromptModel("Question?")
	model.input.SetValue("y")
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	selected, ok := msg.(confirmSelectedMessage)
	assert.True(t, ok)
	assert.True(t, selected.confirmed)
	assert.Equal(t, updated.yesOption.short, updated.value)
}

func TestConfirmPromptAcceptsNoLabel(t *testing.T) {
	model := newConfirmPromptModel("Question?")
	model.input.SetValue(model.noOption.label)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	selected, ok := msg.(confirmSelectedMessage)
	assert.True(t, ok)
	assert.False(t, selected.confirmed)
	assert.Equal(t, updated.noOption.short, updated.value)
}

func TestConfirmPromptInvalidChoiceSetsError(t *testing.T) {
	model := newConfirmPromptModel("Question?")
	model.input.SetValue("maybe")
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.Error(t, updated.error)
}

func TestConfirmPromptViewShowsSelectedValue(t *testing.T) {
	model := newConfirmPromptModel("Question?")
	model.value = "y"
	viewText := model.View()
	assert.Contains(t, viewText, "y")
}

func TestConfirmPromptViewShowsError(t *testing.T) {
	model := newConfirmPromptModel("Question?")
	model.error = errors.New("bad choice")
	viewText := model.View()
	assert.Contains(t, viewText, "bad choice")
}

func TestConfirmPromptClearsErrorOnInput(t *testing.T) {
	model := newConfirmPromptModel("Question?")
	model.error = errors.New("bad choice")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.Nil(t, updated.error)
}

func TestBuildConfirmPromptUsesTemplate(t *testing.T) {
	prompt := buildConfirmPrompt("Question?", "y", "n")
	assert.Contains(t, prompt, "(y/n)")
}

func TestBuildConfirmPromptUsesTestSuffix(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	prompt := buildConfirmPrompt("Question?", "y", "n")
	assert.Contains(t, prompt, "cmd.init.prompt.confirm.suffix")
}
