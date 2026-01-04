package add

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestConfirmPromptDefaultNoOnEmptyInput(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newConfirmPromptModel("Question?", nil)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, "cmd.init.prompt.option.no.short", updated.Value)
	assert.NotNil(t, cmd)
}

func TestConfirmKeyMapFullHelpReturnsBindings(t *testing.T) {
	help := confirmKeyMap{}.FullHelp()
	assert.NotEmpty(t, help)
}

func TestConfirmPromptInitReturnsNil(t *testing.T) {
	model := newConfirmPromptModel("Question?", nil)
	assert.Nil(t, model.Init())
}

func TestConfirmPromptAcceptsYesInput(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newConfirmPromptModel("Question?", nil)
	model.input.SetValue("cmd.init.prompt.option.yes.short")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, "cmd.init.prompt.option.yes.short", updated.Value)
	assert.NotNil(t, cmd)
}

func TestConfirmPromptAcceptsNoInput(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newConfirmPromptModel("Question?", nil)
	model.input.SetValue("cmd.init.prompt.option.no.short")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, "cmd.init.prompt.option.no.short", updated.Value)
	assert.NotNil(t, cmd)
}

func TestConfirmPromptUpdateClearsErrorOnNonEnter(t *testing.T) {
	model := newConfirmPromptModel("Question?", nil)
	model.error = errors.New("bad")

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	assert.Nil(t, updated.error)
}

func TestConfirmPromptRejectsInvalidInput(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newConfirmPromptModel("Question?", nil)
	model.input.SetValue("nope")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, updated.error)
	assert.Nil(t, cmd)
}

func TestConfirmPromptViewShowsError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newConfirmPromptModel("Question?", nil)
	model.error = errors.New("invalid")

	view := model.View()
	assert.Contains(t, view, "invalid")
}

func TestConfirmPromptViewAnsweredValue(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newConfirmPromptModel("Question?", nil)
	model.Value = "y"
	view := model.View()
	assert.Contains(t, view, "y")
}

type confirmTestMsg struct {
	confirmed bool
}

func TestConfirmPromptConfirmSelectedUsesMessageBuilder(t *testing.T) {
	model := newConfirmPromptModel("Question?", func(confirmed bool) tea.Msg {
		return confirmTestMsg{confirmed: confirmed}
	})
	cmd := model.confirmSelected(true)
	msg := cmd()
	typed, ok := msg.(confirmTestMsg)
	assert.True(t, ok)
	assert.True(t, typed.confirmed)
}

func TestConfirmPromptConfirmSelectedDefaultMessage(t *testing.T) {
	model := newConfirmPromptModel("Question?", nil)
	cmd := model.confirmSelected(false)
	msg := cmd()
	typed, ok := msg.(confirmSelectedMessage)
	assert.True(t, ok)
	assert.False(t, typed.confirmed)
}

func TestTextInputPromptViewRendersAnsweredValue(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newTextInputPromptModel("Question?", "abc")
	model.Value = "final"

	view := model.View()
	assert.Contains(t, view, "Question?")
	assert.Contains(t, view, "final")
}

func TestTextInputPromptInitReturnsNil(t *testing.T) {
	model := newTextInputPromptModel("Question?", "")
	assert.Nil(t, model.Init())
}

func TestTextInputPromptUpdateAcceptsInput(t *testing.T) {
	model := newTextInputPromptModel("Question?", "")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	assert.Contains(t, updated.input.Value(), "x")
}
