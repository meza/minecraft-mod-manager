package install

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

type fakePromptModel struct{}

func (fakePromptModel) Init() tea.Cmd { return nil }

func (fakePromptModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return fakePromptModel{}, nil
}

func (fakePromptModel) View() string { return "" }

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

func TestConfirmPromptModelAcceptsNo(t *testing.T) {
	prompt := newConfirmPromptModel("Question?")
	prompt.input.SetValue(prompt.noOption.short)
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

func TestConfigInitModelInitReturnsNil(t *testing.T) {
	model := newConfigInitModel("headline", "Question?")
	assert.Nil(t, model.Init())
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

func TestConfigInitModelViewIncludesHeadline(t *testing.T) {
	model := newConfigInitModel("headline", "Question?")
	viewOutput := model.View()
	assert.Contains(t, viewOutput, "headline")
}

func TestConfigInitModelViewWithEmptyHeadline(t *testing.T) {
	model := newConfigInitModel("", "Question?")
	viewOutput := model.View()
	assert.Contains(t, viewOutput, "Question?")
}

func TestConfigInitResultReturnsErrorForUnexpectedModel(t *testing.T) {
	_, _, err := configInitResult(fakePromptModel{})
	assert.Error(t, err)
}

func TestRunConfigInitPromptUsesRunTeaResult(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return false })
	t.Cleanup(restore)

	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})

	deps := installDeps{
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
	}

	confirmed, canceled, err := runConfigInitPrompt(command, deps, config.NewMetadata("/cfg/modlist.json"))
	require.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

func TestConfigInitResultHandlesPointerModel(t *testing.T) {
	model := &configInitModel{confirmed: true, canceled: false}
	confirmed, canceled, err := configInitResult(model)
	require.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

func TestRunConfigInitPromptReturnsErrorWhenRunTeaFails(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return false })
	t.Cleanup(restore)

	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})

	runErr := errors.New("run tea failed")
	deps := installDeps{
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			return nil, runErr
		},
	}

	_, _, err := runConfigInitPrompt(command, deps, config.NewMetadata("/cfg/modlist.json"))
	assert.ErrorIs(t, err, runErr)
}

func TestRunConfigInitPromptReturnsErrorForUnexpectedModel(t *testing.T) {
	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})

	deps := installDeps{
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			return fakePromptModel{}, nil
		},
	}

	_, _, err := runConfigInitPrompt(command, deps, config.NewMetadata("/cfg/modlist.json"))
	assert.Error(t, err)
}

func TestRunConfigInitPromptUsesColorStyles(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreColor)

	outputBuffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	command.SetOut(fakeTTY{Buffer: outputBuffer})

	deps := installDeps{
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
	}

	confirmed, canceled, err := runConfigInitPrompt(command, deps, config.NewMetadata("/cfg/modlist.json"))
	require.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

func TestRunConfigInitPromptReturnsErrorWhenMissingRunner(t *testing.T) {
	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})

	_, _, err := runConfigInitPrompt(command, installDeps{}, config.NewMetadata("/cfg/modlist.json"))
	assert.Error(t, err)
}

type fakeTTY struct {
	*bytes.Buffer
}

func (tty fakeTTY) Fd() uintptr { return 1 }
