package scan

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestScanConfirmPromptInitReturnsNil(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanConfirmPromptModel("Question?")
	assert.Nil(t, model.Init())
}

func TestScanConfirmPromptAcceptsDefaultNo(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanConfirmPromptModel("Question?")
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, model.noOption.short, updated.Value)
	assert.NotNil(t, cmd)

	msg := cmd()
	selected, ok := msg.(scanConfirmSelectedMessage)
	assert.True(t, ok)
	assert.False(t, selected.confirmed)
}

func TestScanConfirmPromptEscAndCtrlCQuit(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanConfirmPromptModel("Question?")
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.NotNil(t, cmd)
	assert.Equal(t, model.Value, updated.Value)

	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)

	model = newScanConfirmPromptModel("Question?")
	_, cmd = model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.NotNil(t, cmd)

	msg = cmd()
	_, ok = msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestScanConfirmPromptAcceptsYesAndNo(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanConfirmPromptModel("Question?")
	model.input.SetValue(model.yesOption.short)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, model.yesOption.short, updated.Value)
	assert.NotNil(t, cmd)

	msg := cmd()
	selected, ok := msg.(scanConfirmSelectedMessage)
	assert.True(t, ok)
	assert.True(t, selected.confirmed)

	model = newScanConfirmPromptModel("Question?")
	model.input.SetValue(model.noOption.label)
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, model.noOption.short, updated.Value)
	assert.NotNil(t, cmd)

	msg = cmd()
	selected, ok = msg.(scanConfirmSelectedMessage)
	assert.True(t, ok)
	assert.False(t, selected.confirmed)
}

func TestScanConfirmPromptInvalidChoiceSetsError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanConfirmPromptModel("Question?")
	model.input.SetValue("maybe")
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.Error(t, updated.error)
}

func TestScanConfirmPromptUpdateClearsErrorOnInput(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanConfirmPromptModel("Question?")
	model.error = errors.New("boom")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.NoError(t, updated.error)
}

func TestScanConfirmPromptViewShowsSelectedValue(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanConfirmPromptModel("Question?")
	model.Value = model.yesOption.short
	viewText := model.View()
	assert.Contains(t, viewText, model.Value)
}

func TestScanConfirmPromptViewShowsError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanConfirmPromptModel("Question?")
	model.error = errors.New("invalid")
	viewText := model.View()
	assert.Contains(t, viewText, "invalid")
}

func TestScanConfirmPromptConfirmSelected(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanConfirmPromptModel("Question?")
	msg := model.confirmSelected(true)()
	selected, ok := msg.(scanConfirmSelectedMessage)
	assert.True(t, ok)
	assert.True(t, selected.confirmed)
}

func TestScanConfirmPromptMatchesOption(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanConfirmPromptModel("Question?")
	assert.True(t, model.matchesOption(strings.ToUpper(model.yesOption.short), model.yesOption))
	assert.True(t, model.matchesOption(model.yesOption.label, model.yesOption))
	assert.False(t, model.matchesOption("nope", model.yesOption))
}

func TestScanConfirmPromptHandleKeyMsgBlurredKeepsError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanConfirmPromptModel("Question?")
	model.error = errors.New("boom")
	model.input.Blur()

	updated, cmd, handled := model.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.False(t, handled)
	assert.Nil(t, cmd)
	assert.EqualError(t, updated.error, "boom")
}

func TestScanConfirmKeyMapHelp(t *testing.T) {
	keymap := scanConfirmKeyMap{}
	shortHelp := keymap.ShortHelp()
	assert.Len(t, shortHelp, 2)

	fullHelp := keymap.FullHelp()
	assert.Len(t, fullHelp, 1)
	assert.Len(t, fullHelp[0], 2)
}

func TestScanAdoptionPromptInitReturnsNil(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanAdoptionPromptModel([]string{"one"}, "Question?")
	assert.Nil(t, model.Init())
}

func TestScanAdoptionPromptUpdatesWindowSize(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanAdoptionPromptModel([]string{"one"}, "Question?")
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 6})
	typed := updated.(*scanAdoptionPromptModel)
	assert.Equal(t, 120, typed.windowW)
	assert.Equal(t, 6, typed.windowH)
}

func TestScanAdoptionPromptCancel(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanAdoptionPromptModel([]string{"one"}, "Question?")
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.True(t, updated.(*scanAdoptionPromptModel).canceled)
	assert.NotNil(t, cmd)
}

func TestScanAdoptionPromptConfirmSelected(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanAdoptionPromptModel([]string{"one"}, "Question?")
	updated, cmd := model.Update(scanConfirmSelectedMessage{confirmed: true})
	assert.True(t, updated.(*scanAdoptionPromptModel).confirmed)
	assert.NotNil(t, cmd)
}

func TestScanAdoptionPromptForwardsPromptUpdates(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanAdoptionPromptModel([]string{"one"}, "Question?")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	assert.False(t, updated.(*scanAdoptionPromptModel).canceled)
	assert.False(t, updated.(*scanAdoptionPromptModel).confirmed)
}

func TestScanAdoptionPromptViewIncludesSections(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanAdoptionPromptModel([]string{"section"}, "Question?")
	viewText := model.View()
	assert.Contains(t, viewText, "section")
	assert.Contains(t, viewText, "Question?")
}

func TestScanAdoptionPromptViewUsesViewportHeight(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanAdoptionPromptModel([]string{"section", "second"}, "Question?")
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 3})
	viewText := updated.(*scanAdoptionPromptModel).View()
	assert.Contains(t, viewText, "Question?")
	assert.NotContains(t, viewText, "section")
}

func TestScanAdoptionPromptUpdateViewportUsesWindowHeight(t *testing.T) {
	model := newScanAdoptionPromptModel([]string{"one"}, "Question?")
	model.windowW = 40
	model.updateViewport("line-one", 10)
	assert.Equal(t, 10, model.viewport.Height)
	assert.Equal(t, 40, model.viewport.Width)
}

func TestScanAdoptionPromptUpdateViewportFocusBottomShortContent(t *testing.T) {
	model := newScanAdoptionPromptModel([]string{"one"}, "Question?")
	model.updateViewport("line-one", 10)
	assert.False(t, model.focusBottom)
	assert.Equal(t, 0, model.viewport.YOffset)
}

func TestScanAdoptionPromptUpdateViewportPreservesOffset(t *testing.T) {
	model := newScanAdoptionPromptModel([]string{"one"}, "Question?")
	model.focusBottom = false
	model.viewport.SetYOffset(1)
	model.updateViewport("line-one\nline-two\nline-three", 2)
	assert.False(t, model.focusBottom)
	assert.Equal(t, 2, model.viewport.Height)
}

func TestScanAdoptionPromptUpdateViewportNegativeHeight(t *testing.T) {
	model := newScanAdoptionPromptModel([]string{"one"}, "Question?")
	model.updateViewport("line-one\nline-two", -1)
	assert.Equal(t, 0, model.viewport.Height)
}

func TestScanAdoptionPromptUpdateViewportDefaultMessage(t *testing.T) {
	model := newScanAdoptionPromptModel([]string{"one"}, "Question?")
	cmd := model.updateViewportForMessage(scanFakePromptModel{})
	assert.Nil(t, cmd)
}

func TestScanAdoptionPromptMouseUpdatesViewport(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanAdoptionPromptModel([]string{"one", "two"}, "Question?")
	updated, _ := model.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
	typed := updated.(*scanAdoptionPromptModel)
	assert.False(t, typed.canceled)
	assert.False(t, typed.confirmed)
}

func TestScanAdoptionPromptResultUnexpectedModel(t *testing.T) {
	_, err := scanAdoptionPromptResult(scanFakePromptModel{})
	assert.Error(t, err)
}

func TestScanAdoptionPromptResultPointer(t *testing.T) {
	result, err := scanAdoptionPromptResult(&scanAdoptionPromptModel{confirmed: true})
	assert.NoError(t, err)
	assert.True(t, result.confirmed)
}
