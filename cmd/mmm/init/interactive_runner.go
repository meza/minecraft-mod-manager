package init

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/tui"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// InteractiveInitDeps supplies dependencies for RunInteractiveInit.
// Provide any dependencies that should be shared with the caller (filesystem, output, logging).
type InteractiveInitDeps struct {
	FS              afero.Fs
	Output          *output.Output
	Logger          *logger.Logger
	MinecraftClient httpclient.Doer
	RunTea          func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error)
}

// InteractiveInitOptions controls the init run invoked by RunInteractiveInit.
type InteractiveInitOptions struct {
	ConfigPath string
	Quiet      bool
	Debug      bool
}

// RunInteractiveInit runs the init TUI with default values and persists config/lock.
// Use this when another command needs to bootstrap configuration through the same
// interactive flow as `mmm init`. It does not prompt for flags; all prompts are
// driven by the TUI.
func RunInteractiveInit(ctx context.Context, cmd *cobra.Command, deps InteractiveInitDeps, options InteractiveInitOptions) error {
	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
		FS:              deps.FS,
		Output:          deps.Output,
		Logger:          deps.Logger,
		MinecraftClient: deps.MinecraftClient,
		Quiet:           options.Quiet,
		Debug:           options.Debug,
	})
	promptAllowed := tui.ShouldPrompt(tui.PromptEnabled, cmd.InOrStdin(), cmd.OutOrStdout())
	initDependencies := newInitDeps(cmd, common, promptAllowed)
	if deps.RunTea != nil {
		initDependencies.runTea = deps.RunTea
	}

	optionsForInit := initOptions{
		ConfigPath:     options.ConfigPath,
		NonInteractive: false,
		Quiet:          options.Quiet,
		Debug:          options.Debug,
		Loader:         "",
		GameVersion:    "latest",
		ReleaseTypes:   []models.ReleaseType{models.Release},
		ModsFolder:     "mods",
		Provided:       providedFlags{},
	}

	_, _, err := runInit(ctx, cmd, optionsForInit, initDependencies, config.NewMetadata(options.ConfigPath))
	return err
}
