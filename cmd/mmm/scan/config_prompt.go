package scan

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type scanConfigInitModel struct {
	headline  string
	prompt    scanConfirmPromptModel
	canceled  bool
	confirmed bool
}

func newScanConfigInitModel(headline string, question string) scanConfigInitModel {
	return scanConfigInitModel{
		headline: headline,
		prompt:   newScanConfirmPromptModel(question),
	}
}

func (model scanConfigInitModel) Init() tea.Cmd {
	return nil
}

func (model scanConfigInitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		if typed.String() == "ctrl+c" || typed.String() == "esc" {
			model.canceled = true
			return model, tea.Quit
		}
	case scanConfirmSelectedMessage:
		model.confirmed = typed.confirmed
		return model, tea.Quit
	}

	updatedPrompt, cmd := model.prompt.Update(msg)
	model.prompt = updatedPrompt
	return model, cmd
}

func (model scanConfigInitModel) View() string {
	if model.headline == "" {
		return model.prompt.View()
	}
	return model.headline + "\n\n" + model.prompt.View()
}

func runConfigInitPrompt(cmd *cobra.Command, deps scanDeps, meta config.Metadata) (confirmed bool, canceled bool, err error) {
	colorMode := colorModeForOutput(cmd.OutOrStdout())
	headline := messageWithIcon(view.FinalErrorIcon(colorMode), i18n.T("cmd.scan.error.config_missing", &i18n.Tvars{
		Data: &i18n.TData{"configPath": meta.ConfigPath},
	}))
	if colorMode.Enabled() {
		headline = view.ErrorStyle.Render(headline)
	}

	question := i18n.T("cmd.scan.confirm_init", nil)
	model := newScanConfigInitModel(headline, question)

	runTea := deps.runTea
	if runTea == nil {
		runTea = runTeaProgram
	}

	result, err := runTea(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
	if err != nil {
		return false, false, err
	}
	return scanConfigInitResult(result)
}

func scanConfigInitResult(result tea.Model) (confirmed bool, canceled bool, err error) {
	switch typed := result.(type) {
	case scanConfigInitModel:
		return typed.confirmed, typed.canceled, nil
	case *scanConfigInitModel:
		return typed.confirmed, typed.canceled, nil
	default:
		return false, false, errors.New("unexpected prompt model")
	}
}
