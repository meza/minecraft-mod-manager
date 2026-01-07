package prune

import (
	"bytes"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type unexpectedModel struct{}

func (unexpectedModel) Init() tea.Cmd { return nil }

func (unexpectedModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return unexpectedModel{}, nil }

func (unexpectedModel) View() string { return "" }

func TestConfirmPromptModelDefaultsToNo(t *testing.T) {
	prompt := newConfirmPromptModel("Question?")
	updated, cmd := prompt.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	selected, ok := msg.(confirmSelectedMessage)
	if !assert.True(t, ok) {
		return
	}
	assert.False(t, selected.confirmed)
	assert.NotEmpty(t, updated.value)
}

func TestConfirmPromptModelInitReturnsNil(t *testing.T) {
	prompt := newConfirmPromptModel("Question?")
	assert.Nil(t, prompt.Init())
}

func TestConfirmPromptModelAcceptsYes(t *testing.T) {
	prompt := newConfirmPromptModel("Question?")
	prompt.input.SetValue(prompt.yesOption.short)
	_, cmd := prompt.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	selected, ok := msg.(confirmSelectedMessage)
	if !assert.True(t, ok) {
		return
	}
	assert.True(t, selected.confirmed)
}

func TestConfirmPromptModelAcceptsNoLabel(t *testing.T) {
	prompt := newConfirmPromptModel("Question?")
	prompt.input.SetValue(prompt.noOption.label)
	_, cmd := prompt.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	selected, ok := msg.(confirmSelectedMessage)
	if !assert.True(t, ok) {
		return
	}
	assert.False(t, selected.confirmed)
}

func TestConfirmPromptModelInvalidChoiceSetsError(t *testing.T) {
	prompt := newConfirmPromptModel("Question?")
	prompt.input.SetValue("maybe")
	updated, cmd := prompt.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.Error(t, updated.error)
}

func TestConfirmPromptModelClearsErrorOnInput(t *testing.T) {
	prompt := newConfirmPromptModel("Question?")
	prompt.error = errors.New("previous error")
	updated, _ := prompt.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.NoError(t, updated.error)
}

func TestConfirmPromptModelViewShowsValue(t *testing.T) {
	prompt := newConfirmPromptModel("Question?")
	prompt.value = "n"
	viewOutput := prompt.View()
	assert.Contains(t, viewOutput, "n")
}

func TestConfirmPromptModelViewShowsError(t *testing.T) {
	prompt := newConfirmPromptModel("Question?")
	prompt.error = errors.New("invalid choice")
	viewOutput := prompt.View()
	assert.Contains(t, viewOutput, "invalid choice")
}

func TestConfigInitModelCancelsOnEsc(t *testing.T) {
	model := newConfigInitModel("headline", "Question?")
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	_, ok := cmd().(tea.QuitMsg)
	assert.True(t, ok)
	assert.True(t, updated.(configInitModel).canceled)
}

func TestConfigInitModelCancelsOnCtrlC(t *testing.T) {
	model := newConfigInitModel("headline", "Question?")
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	_, ok := cmd().(tea.QuitMsg)
	assert.True(t, ok)
	assert.True(t, updated.(configInitModel).canceled)
}

func TestConfigInitModelAcceptsConfirmMessage(t *testing.T) {
	model := newConfigInitModel("headline", "Question?")
	updated, cmd := model.Update(confirmSelectedMessage{confirmed: true})
	_, ok := cmd().(tea.QuitMsg)
	assert.True(t, ok)
	assert.True(t, updated.(configInitModel).confirmed)
}

func TestConfigInitModelUpdateForwardsMessages(t *testing.T) {
	model := newConfigInitModel("headline", "Question?")
	updated, cmd := model.Update(struct{}{})
	assert.Nil(t, cmd)
	assert.Equal(t, "headline", updated.(configInitModel).headline)
}

func TestConfigInitModelInitReturnsNil(t *testing.T) {
	model := newConfigInitModel("headline", "Question?")
	assert.Nil(t, model.Init())
}

func TestConfigInitModelViewUsesHeadline(t *testing.T) {
	model := newConfigInitModel("headline", "Question?")
	viewOutput := model.View()
	assert.Contains(t, viewOutput, "headline")
}

func TestConfigInitModelViewWithoutHeadlineShowsPrompt(t *testing.T) {
	model := newConfigInitModel("", "Question?")
	viewOutput := model.View()
	assert.Equal(t, model.prompt.View(), viewOutput)
}

func TestRunConfigInitPromptUsesDefaultRunner(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return false })
	t.Cleanup(restoreTerminal)
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreColor)

	originalRunner := runTeaProgram
	t.Cleanup(func() {
		runTeaProgram = originalRunner
	})
	runTeaProgram = func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		return configInitModel{confirmed: false}, nil
	}

	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})

	confirmed, canceled, err := runConfigInitPrompt(command, pruneDeps{}, config.NewMetadata("/cfg/modlist.json"))
	require.NoError(t, err)
	assert.False(t, confirmed)
	assert.False(t, canceled)
}

func TestPruneConfirmModelCancelsOnEsc(t *testing.T) {
	model := newPruneConfirmModel("list", "Question?")
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	_, ok := cmd().(tea.QuitMsg)
	assert.True(t, ok)
	assert.True(t, updated.(pruneConfirmModel).canceled)
}

func TestPruneConfirmModelAcceptsConfirmMessage(t *testing.T) {
	model := newPruneConfirmModel("list", "Question?")
	updated, cmd := model.Update(confirmSelectedMessage{confirmed: true})
	_, ok := cmd().(tea.QuitMsg)
	assert.True(t, ok)
	assert.True(t, updated.(pruneConfirmModel).confirmed)
}

func TestPruneConfirmModelUpdateForwardsMessages(t *testing.T) {
	model := newPruneConfirmModel("list", "Question?")
	updated, cmd := model.Update(struct{}{})
	assert.Nil(t, cmd)
	assert.Equal(t, "list", updated.(pruneConfirmModel).listView)
}

func TestPruneConfirmModelInitReturnsNil(t *testing.T) {
	model := newPruneConfirmModel("list", "Question?")
	assert.Nil(t, model.Init())
}

func TestPruneConfirmModelViewUsesList(t *testing.T) {
	model := newPruneConfirmModel("list", "Question?")
	viewOutput := model.View()
	assert.Contains(t, viewOutput, "list")
}

func TestPruneConfirmModelViewWithoutListUsesPrompt(t *testing.T) {
	model := newPruneConfirmModel("", "Question?")
	viewOutput := model.View()
	assert.Equal(t, model.prompt.View(), viewOutput)
}

func TestRunDeletePromptUsesDefaultRunner(t *testing.T) {
	originalRunner := runTeaProgram
	t.Cleanup(func() {
		runTeaProgram = originalRunner
	})
	runTeaProgram = func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		return pruneConfirmModel{confirmed: true}, nil
	}

	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})

	confirmed, canceled, err := runDeletePrompt(command, pruneDeps{}, view.ColorDisabled, []string{"/mods/unmanaged.jar"})
	require.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

func TestPruneConfirmResultHandlesPointerModel(t *testing.T) {
	model := &pruneConfirmModel{confirmed: true, canceled: false}
	confirmed, canceled, err := pruneConfirmResult(model)
	require.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

func TestPruneConfirmResultReturnsErrorForUnexpectedModel(t *testing.T) {
	_, _, err := pruneConfirmResult(unexpectedModel{})
	assert.Error(t, err)
}

func TestConfigInitResultReturnsErrorForUnexpectedModel(t *testing.T) {
	_, _, err := configInitResult(unexpectedModel{})
	assert.Error(t, err)
}
