package test

import (
	"context"
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/cobra"
)

type testVersionPromptInput struct {
	headline     string
	question     string
	initialValue string
	initialError string
}

type testVersionPromptModel struct {
	headline string
	prompt   initCmd.GameVersionModel
	canceled bool
	selected string
}

func newTestVersionPromptModel(ctx context.Context, client httpclient.Doer, input testVersionPromptInput) testVersionPromptModel {
	prompt := initCmd.NewGameVersionPromptModel(ctx, client, input.initialValue, initCmd.GameVersionPromptOptions{
		Question: input.question,
		Messages: initCmd.GameVersionPromptMessages{
			Empty:             i18n.T("cmd.init.prompt.game-version.error", nil),
			Invalid:           i18n.T("cmd.test.prompt.version.invalid", nil),
			Unavailable:       i18n.T("cmd.test.prompt.version.unavailable", nil),
			LatestUnavailable: i18n.T("cmd.test.prompt.version.latest_unavailable", nil),
		},
		InitialError: input.initialError,
	})

	return testVersionPromptModel{
		headline: input.headline,
		prompt:   prompt,
	}
}

func (model testVersionPromptModel) Init() tea.Cmd {
	return nil
}

func (model testVersionPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		if typed.String() == "ctrl+c" || typed.String() == "esc" {
			model.canceled = true
			return model, tea.Quit
		}
	case initCmd.GameVersionSelectedMessage:
		model.selected = typed.GameVersion
		return model, tea.Quit
	}

	updated, cmd := model.prompt.Update(msg)
	model.prompt = updated
	return model, cmd
}

func (model testVersionPromptModel) View() string {
	if model.headline == "" {
		return model.prompt.View()
	}
	return model.headline + "\n\n" + model.prompt.View()
}

func runTestVersionPrompt(
	ctx context.Context,
	cmd *cobra.Command,
	deps testDeps,
	input testVersionPromptInput,
) (string, bool, error) {
	model := newTestVersionPromptModel(ctx, deps.minecraftClient, input)

	runTea := deps.runTea
	if runTea == nil {
		runTea = runTeaProgram
	}

	result, err := runTea(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
	if err != nil {
		return "", false, err
	}
	return testVersionPromptResult(result)
}

func testVersionPromptResult(result tea.Model) (string, bool, error) {
	switch typed := result.(type) {
	case testVersionPromptModel:
		return typed.selected, typed.canceled, nil
	case *testVersionPromptModel:
		return typed.selected, typed.canceled, nil
	default:
		return "", false, errors.New("unexpected version prompt model")
	}
}
