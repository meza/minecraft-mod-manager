package change

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestConfigInitModelCancel(t *testing.T) {
	model := newConfigInitModel("headline", "question")
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.True(t, updated.(configInitModel).canceled)
	assert.NotNil(t, cmd)
}

func TestConfigInitModelEscCancel(t *testing.T) {
	model := newConfigInitModel("headline", "question")
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.True(t, updated.(configInitModel).canceled)
	assert.NotNil(t, cmd)
}

func TestConfigInitModelInitReturnsNil(t *testing.T) {
	model := newConfigInitModel("headline", "question")
	assert.Nil(t, model.Init())
}

func TestConfigInitModelConfirm(t *testing.T) {
	model := newConfigInitModel("", "question")
	updated, cmd := model.Update(confirmSelectedMessage{confirmed: true})
	assert.True(t, updated.(configInitModel).confirmed)
	assert.NotNil(t, cmd)
}

func TestConfigInitModelForwardsPromptUpdates(t *testing.T) {
	model := newConfigInitModel("", "question")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	assert.False(t, updated.(configInitModel).confirmed)
	assert.False(t, updated.(configInitModel).canceled)
}

func TestConfigInitModelViewRendersHeadline(t *testing.T) {
	model := newConfigInitModel("headline", "question")
	viewText := model.View()
	assert.Contains(t, viewText, "headline")
}

func TestConfigInitModelViewWithoutHeadline(t *testing.T) {
	model := newConfigInitModel("", "question")
	viewText := model.View()
	assert.NotContains(t, viewText, "headline")
	assert.Contains(t, viewText, "question")
}

func TestRunConfigInitPromptUsesRunner(t *testing.T) {
	cmd := &cobra.Command{}
	meta := config.NewMetadata("/cfg/modlist.json")
	deps := changeDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
	}

	confirmed, canceled, err := runConfigInitPrompt(cmd, deps, meta)
	assert.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

func TestRunConfigInitPromptErrorsWithoutRunner(t *testing.T) {
	cmd := &cobra.Command{}
	meta := config.NewMetadata("/cfg/modlist.json")
	deps := changeDeps{}

	_, _, err := runConfigInitPrompt(cmd, deps, meta)
	assert.Error(t, err)
}

func TestRunConfigInitPromptPropagatesRunnerError(t *testing.T) {
	cmd := &cobra.Command{}
	meta := config.NewMetadata("/cfg/modlist.json")
	deps := changeDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, errors.New("boom")
		},
	}

	_, _, err := runConfigInitPrompt(cmd, deps, meta)
	assert.Error(t, err)
}

func TestRunConfigInitPromptColorEnabled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	restoreProfile := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreTerminal)
	t.Cleanup(restoreProfile)

	cmd := &cobra.Command{}
	cmd.SetOut(fdWriter{fd: 1})
	meta := config.NewMetadata("/cfg/modlist.json")
	deps := changeDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
	}

	confirmed, canceled, err := runConfigInitPrompt(cmd, deps, meta)
	assert.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

func TestConfigInitResultUnexpectedModel(t *testing.T) {
	_, _, err := configInitResult(fakePromptModel{})
	assert.Error(t, err)
}

func TestConfigInitResultWithPointer(t *testing.T) {
	confirmed, canceled, err := configInitResult(&configInitModel{confirmed: true})
	assert.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

type fakePromptModel struct{}

func (fakePromptModel) Init() tea.Cmd {
	return nil
}

func (fakePromptModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return fakePromptModel{}, nil
}

func (fakePromptModel) View() string {
	return ""
}

func TestRunInteractiveInitUsesOverride(t *testing.T) {
	original := runInitInteractive
	defer func() { runInitInteractive = original }()

	called := false
	runInitInteractive = func(_ context.Context, _ *cobra.Command, _ initCmd.InteractiveInitDeps, _ initCmd.InteractiveInitOptions) error {
		called = true
		return nil
	}

	err := runInteractiveInit(context.Background(), &cobra.Command{}, changeDeps{}, changeOptions{}, config.NewMetadata("modlist.json"))
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestRunInteractiveInitPropagatesError(t *testing.T) {
	original := runInitInteractive
	defer func() { runInitInteractive = original }()

	runInitInteractive = func(_ context.Context, _ *cobra.Command, _ initCmd.InteractiveInitDeps, _ initCmd.InteractiveInitOptions) error {
		return errors.New("boom")
	}

	err := runInteractiveInit(context.Background(), &cobra.Command{}, changeDeps{}, changeOptions{}, config.NewMetadata("modlist.json"))
	assert.Error(t, err)
}
