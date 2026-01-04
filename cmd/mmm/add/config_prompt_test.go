package add

import (
	"bytes"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestConfigInitModelCancelOnEsc(t *testing.T) {
	model := newConfigInitModel("headline", "question")
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.True(t, updated.(configInitModel).canceled)
	assert.NotNil(t, cmd)
}

func TestConfigInitModelInitReturnsNil(t *testing.T) {
	model := newConfigInitModel("headline", "question")
	assert.Nil(t, model.Init())
}

func TestConfigInitModelUpdateConfirmSelected(t *testing.T) {
	model := newConfigInitModel("headline", "question")
	updated, cmd := model.Update(confirmSelectedMessage{confirmed: true})
	typed := updated.(configInitModel)
	assert.True(t, typed.confirmed)
	assert.NotNil(t, cmd)
}

func TestConfigInitModelViewIncludesHeadline(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newConfigInitModel("headline", "question")
	view := model.View()
	assert.Contains(t, view, "headline")
	assert.Contains(t, view, "question")
}

func TestConfigInitModelViewWithoutHeadline(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newConfigInitModel("", "question")
	view := model.View()
	assert.Contains(t, view, "question")
	assert.NotContains(t, view, "headline")
}

func TestConfigInitModelMessageBuilder(t *testing.T) {
	model := newConfigInitModel("headline", "question")
	cmd := model.prompt.confirmSelected(true)
	msg := cmd()
	typed, ok := msg.(confirmSelectedMessage)
	assert.True(t, ok)
	assert.True(t, typed.confirmed)
}

func TestConfigInitModelUpdatePassesThroughPrompt(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newConfigInitModel("headline", "question")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	typed := updated.(configInitModel)
	assert.Contains(t, typed.prompt.input.Value(), "y")
}

func TestConfigInitResultUnexpectedModel(t *testing.T) {
	_, _, err := configInitResult(fakePromptModel{})
	assert.Error(t, err)
}

func TestConfigInitResultPointerModel(t *testing.T) {
	confirmed, canceled, err := configInitResult(&configInitModel{confirmed: true, canceled: true})
	assert.NoError(t, err)
	assert.True(t, confirmed)
	assert.True(t, canceled)
}

func TestRunConfigInitPromptMissingRunner(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	_, _, err := runConfigInitPrompt(cmd, addDeps{}, config.NewMetadata("modlist.json"))
	assert.Error(t, err)
}

func TestRunConfigInitPromptReturnsRunnerError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	runErr := errors.New("run tea failed")
	deps := addDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, runErr
		},
	}

	_, _, err := runConfigInitPrompt(cmd, deps, config.NewMetadata("modlist.json"))
	assert.ErrorIs(t, err, runErr)
}

func TestRunConfigInitPromptReturnsConfirmed(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	deps := addDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			if typed, ok := model.(configInitModel); ok {
				typed.confirmed = true
				return typed, nil
			}
			return model, nil
		},
	}

	confirmed, canceled, err := runConfigInitPrompt(cmd, deps, config.NewMetadata("modlist.json"))
	assert.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

func TestRunConfigInitPromptColorEnabled(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.ANSI })
	t.Cleanup(restoreColor)

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTYReader{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTYWriter{Buffer: bytes.NewBuffer(nil)})
	cmd.SetErr(bytes.NewBuffer(nil))

	deps := addDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			if typed, ok := model.(configInitModel); ok {
				typed.confirmed = true
				return typed, nil
			}
			return model, nil
		},
	}

	confirmed, canceled, err := runConfigInitPrompt(cmd, deps, config.NewMetadata("modlist.json"))
	assert.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

type fakePromptModel struct{}

func (model fakePromptModel) Init() tea.Cmd {
	return nil
}

func (model fakePromptModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return model, nil
}

func (model fakePromptModel) View() string {
	return ""
}
