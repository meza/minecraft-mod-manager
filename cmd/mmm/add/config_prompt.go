package add

import (
	"context"
	"errors"

	tea "github.com/charmbracelet/bubbletea"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/cobra"
)

var runInitInteractive = initCmd.RunInteractiveInit

type configInitModel struct {
	headline  string
	prompt    confirmPromptModel
	canceled  bool
	confirmed bool
}

func newConfigInitModel(headline string, question string) configInitModel {
	return configInitModel{
		headline: headline,
		prompt:   newConfirmPromptModel(question, func(confirmed bool) tea.Msg { return confirmSelectedMessage{confirmed: confirmed} }),
	}
}

func (model configInitModel) Init() tea.Cmd {
	return nil
}

func (model configInitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		if typed.String() == "ctrl+c" || typed.String() == "esc" {
			model.canceled = true
			return model, tea.Quit
		}
	case confirmSelectedMessage:
		model.confirmed = typed.confirmed
		return model, tea.Quit
	}

	updatedPrompt, cmd := model.prompt.Update(msg)
	model.prompt = updatedPrompt
	return model, cmd
}

func (model configInitModel) View() string {
	if model.headline == "" {
		return model.prompt.View()
	}
	return model.headline + "\n" + model.prompt.View()
}

func runConfigInitPrompt(cmd *cobra.Command, deps addDeps, meta config.Metadata) (confirmed bool, canceled bool, err error) {
	colorMode := colorModeForOutput(cmd.OutOrStdout())
	headline := messageWithIcon(view.FinalErrorIcon(colorMode), i18n.T("cmd.config.error.missing", &i18n.Tvars{
		Data: &i18n.TData{"configPath": meta.ConfigPath},
	}))
	if colorMode.Enabled() {
		headline = view.ErrorStyle.Render(headline)
	}

	question := i18n.T("cmd.config.confirm_init_short", nil)
	model := newConfigInitModel(headline, question)

	runTea := deps.runTea
	if runTea == nil {
		return false, false, errors.New("missing bubble tea runner")
	}

	result, err := runTea(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
	if err != nil {
		return false, false, err
	}
	return configInitResult(result)
}

func configInitResult(result tea.Model) (confirmed bool, canceled bool, err error) {
	switch typed := result.(type) {
	case configInitModel:
		return typed.confirmed, typed.canceled, nil
	case *configInitModel:
		return typed.confirmed, typed.canceled, nil
	default:
		return false, false, errors.New("unexpected prompt model")
	}
}

func runInteractiveInit(
	ctx context.Context,
	cmd *cobra.Command,
	deps addDeps,
	options addOptions,
	meta config.Metadata,
) error {
	return runInitInteractive(ctx, cmd, initCmd.InteractiveInitDeps{
		FS:              deps.fs,
		Output:          deps.output,
		Logger:          deps.logger,
		MinecraftClient: deps.minecraftClient,
		RunTea:          deps.runTea,
	}, initCmd.InteractiveInitOptions{
		ConfigPath: meta.ConfigPath,
		Quiet:      options.Quiet,
		Debug:      options.Debug,
	})
}
