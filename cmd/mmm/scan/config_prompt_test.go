package scan

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

func TestScanConfigInitModelCancel(t *testing.T) {
	model := newScanConfigInitModel("headline", "question")
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.True(t, updated.(scanConfigInitModel).canceled)
	assert.NotNil(t, cmd)
}

func TestScanConfigInitModelEscCancel(t *testing.T) {
	model := newScanConfigInitModel("headline", "question")
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.True(t, updated.(scanConfigInitModel).canceled)
	assert.NotNil(t, cmd)
}

func TestScanConfigInitModelInitReturnsNil(t *testing.T) {
	model := newScanConfigInitModel("headline", "question")
	assert.Nil(t, model.Init())
}

func TestScanConfigInitModelConfirm(t *testing.T) {
	model := newScanConfigInitModel("", "question")
	updated, cmd := model.Update(scanConfirmSelectedMessage{confirmed: true})
	assert.True(t, updated.(scanConfigInitModel).confirmed)
	assert.NotNil(t, cmd)
}

func TestScanConfigInitModelForwardsPromptUpdates(t *testing.T) {
	model := newScanConfigInitModel("", "question")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	assert.False(t, updated.(scanConfigInitModel).confirmed)
	assert.False(t, updated.(scanConfigInitModel).canceled)
}

func TestScanConfigInitModelViewRendersHeadline(t *testing.T) {
	model := newScanConfigInitModel("headline", "question")
	viewText := model.View()
	assert.Contains(t, viewText, "headline")
}

func TestScanConfigInitModelViewWithoutHeadline(t *testing.T) {
	model := newScanConfigInitModel("", "question")
	viewText := model.View()
	assert.NotContains(t, viewText, "headline")
	assert.Contains(t, viewText, "question")
}

func TestRunScanConfigInitPromptUsesRunner(t *testing.T) {
	cmd := &cobra.Command{}
	meta := config.NewMetadata("/cfg/modlist.json")
	deps := scanDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return scanConfigInitModel{confirmed: true}, nil
		},
	}

	confirmed, canceled, err := runConfigInitPrompt(cmd, deps, meta)
	assert.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

func TestRunScanConfigInitPromptPropagatesRunnerError(t *testing.T) {
	cmd := &cobra.Command{}
	meta := config.NewMetadata("/cfg/modlist.json")
	deps := scanDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, errors.New("boom")
		},
	}

	_, _, err := runConfigInitPrompt(cmd, deps, meta)
	assert.Error(t, err)
}

func TestRunScanConfigInitPromptColorEnabled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	restoreProfile := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreTerminal)
	t.Cleanup(restoreProfile)

	cmd := &cobra.Command{}
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})
	meta := config.NewMetadata("/cfg/modlist.json")
	deps := scanDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return scanConfigInitModel{confirmed: true}, nil
		},
	}

	confirmed, canceled, err := runConfigInitPrompt(cmd, deps, meta)
	assert.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

func TestRunScanConfigInitPromptUsesDefaultRunner(t *testing.T) {
	originalRunTea := runTeaProgram
	t.Cleanup(func() { runTeaProgram = originalRunTea })

	runTeaProgram = func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
		return scanConfigInitModel{confirmed: true}, nil
	}

	cmd := &cobra.Command{}
	meta := config.NewMetadata("/cfg/modlist.json")
	deps := scanDeps{}

	confirmed, canceled, err := runConfigInitPrompt(cmd, deps, meta)
	assert.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

func TestScanConfigInitResultUnexpectedModel(t *testing.T) {
	_, _, err := scanConfigInitResult(scanFakePromptModel{})
	assert.Error(t, err)
}

func TestScanConfigInitResultWithPointer(t *testing.T) {
	confirmed, canceled, err := scanConfigInitResult(&scanConfigInitModel{confirmed: true})
	assert.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

type scanFakePromptModel struct{}

func (scanFakePromptModel) Init() tea.Cmd {
	return nil
}

func (scanFakePromptModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return scanFakePromptModel{}, nil
}

func (scanFakePromptModel) View() string {
	return ""
}
