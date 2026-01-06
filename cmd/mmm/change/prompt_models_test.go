package change

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/view"
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

func TestChangePolicyPromptSelectsDefaultOnEnter(t *testing.T) {
	model := newChangePolicyPromptModel()
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	selected, ok := msg.(changePolicySelectedMsg)
	assert.True(t, ok)
	assert.Equal(t, changeForcePolicyKeepConfig, selected.policy)
	assert.NotNil(t, updated.list)
}

func TestChangePolicyPromptViewIncludesPrompt(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newChangePolicyPromptModel()
	assert.Contains(t, model.View(), "cmd.change.force_policy.prompt")
}

func TestChangePolicyPromptSetWidth(t *testing.T) {
	model := newChangePolicyPromptModel()
	model.SetWidth(42)
	assert.Equal(t, 42, model.list.Width())
}

func TestChangePolicyPromptSetWidthNil(t *testing.T) {
	var model *changePolicyPromptModel
	model.SetWidth(10)
}

func TestChangeForcePolicyItemFilterValue(t *testing.T) {
	item := changeForcePolicyItem{label: "keep", policy: changeForcePolicyKeepConfig}
	assert.Equal(t, "", item.FilterValue())
}

func TestForcePolicyLabel(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	label, ok := forcePolicyLabel(changeForcePolicyKeepConfig)
	assert.True(t, ok)
	assert.Contains(t, label, "cmd.change.force_policy.option.keep")

	label, ok = forcePolicyLabel(changeForcePolicyPruneConfig)
	assert.True(t, ok)
	assert.Contains(t, label, "cmd.change.force_policy.option.prune")

	label, ok = forcePolicyLabel(changeForcePolicyDisableSkipped)
	assert.True(t, ok)
	assert.Contains(t, label, "cmd.change.force_policy.option.disable")

	label, ok = forcePolicyLabel(changeForcePolicyUnset)
	assert.False(t, ok)
	assert.Equal(t, "", label)
}

func TestForcePolicyAnswerLine(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	line := forcePolicyAnswerLine(changeForcePolicyDisableSkipped)
	assert.Contains(t, line, "cmd.change.force_policy.prompt")
	assert.Contains(t, line, "cmd.change.force_policy.answer.disable")

	line = forcePolicyAnswerLine(changeForcePolicyUnset)
	assert.Contains(t, line, "cmd.change.force_policy.prompt")
	assert.NotContains(t, line, "cmd.change.force_policy.answer.disable")
}

func TestForcePolicyAnswerLabel(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	label, ok := forcePolicyAnswerLabel(changeForcePolicyKeepConfig)
	assert.True(t, ok)
	assert.Contains(t, label, "cmd.change.force_policy.answer.keep")

	label, ok = forcePolicyAnswerLabel(changeForcePolicyPruneConfig)
	assert.True(t, ok)
	assert.Contains(t, label, "cmd.change.force_policy.answer.prune")

	label, ok = forcePolicyAnswerLabel(changeForcePolicyDisableSkipped)
	assert.True(t, ok)
	assert.Contains(t, label, "cmd.change.force_policy.answer.disable")

	label, ok = forcePolicyAnswerLabel(changeForcePolicyUnset)
	assert.False(t, ok)
	assert.Equal(t, "", label)
}

func TestChangeForcePolicyDelegateRender(t *testing.T) {
	items := []list.Item{
		changeForcePolicyItem{label: "Keep", policy: changeForcePolicyKeepConfig},
		changeForcePolicyItem{label: "Prune", policy: changeForcePolicyPruneConfig},
	}
	listModel := list.New(items, changeForcePolicyDelegate{}, 20, 4)
	listModel.Select(0)

	var selected strings.Builder
	changeForcePolicyDelegate{}.Render(&selected, listModel, 0, items[0])
	assert.Contains(t, selected.String(), "Keep")

	var unselected strings.Builder
	changeForcePolicyDelegate{}.Render(&unselected, listModel, 1, items[1])
	assert.Contains(t, unselected.String(), "Prune")
}

func TestChangeForcePolicyDelegateRenderIgnoresUnknownItem(t *testing.T) {
	listModel := list.New(nil, changeForcePolicyDelegate{}, 10, 2)
	var buffer strings.Builder
	changeForcePolicyDelegate{}.Render(&buffer, listModel, 0, listItemStub{})
	assert.Equal(t, "", buffer.String())
}

func TestChangeForcePolicyDelegateRenderHandlesWriteError(t *testing.T) {
	items := []list.Item{changeForcePolicyItem{label: "Keep", policy: changeForcePolicyKeepConfig}}
	listModel := list.New(items, changeForcePolicyDelegate{}, 10, 2)
	listModel.Select(0)

	writer := failingWriter{}
	changeForcePolicyDelegate{}.Render(writer, listModel, 0, items[0])
}

func TestChangeForcePolicyDelegateRenderHandlesWriteErrorUnselected(t *testing.T) {
	items := []list.Item{
		changeForcePolicyItem{label: "Keep", policy: changeForcePolicyKeepConfig},
		changeForcePolicyItem{label: "Prune", policy: changeForcePolicyPruneConfig},
	}
	listModel := list.New(items, changeForcePolicyDelegate{}, 10, 2)
	listModel.Select(0)

	writer := failingWriter{}
	changeForcePolicyDelegate{}.Render(writer, listModel, 1, items[1])
}

func TestForcePolicyPointerUsesUnicodeWhenAvailable(t *testing.T) {
	restore := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	defer restore()
	assert.Equal(t, "\u276F ", forcePolicyPointer())
}

func TestForcePolicyPointerUsesAsciiFallback(t *testing.T) {
	restore := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	defer restore()
	assert.Equal(t, "> ", forcePolicyPointer())
}

type listItemStub struct{}

func (listItemStub) FilterValue() string { return "" }

type failingWriter struct{}

func (failingWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}
