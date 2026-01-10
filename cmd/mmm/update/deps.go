package update

import (
	"context"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/cmd/mmm/install"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/spf13/cobra"
)

var runInteractiveInit = initCmd.RunInteractiveInit

func newUpdateDeps(common cmddeps.CommonDeps, command *cobra.Command, opts updateOptions) updateDeps {
	return updateDeps{
		fs:         common.FS,
		logger:     common.Logger,
		output:     common.Output,
		clients:    common.Clients,
		fetchMod:   platform.FetchMod,
		downloader: httpclient.DownloadFile,
		install: func(ctx context.Context, _ *cobra.Command, configPath string, quiet bool, debug bool) (install.Result, error) {
			return install.Run(ctx, command, configPath, quiet, debug)
		},
		telemetry: telemetry.RecordCommand,
		runTea:    runTeaProgram,
		runInit: func(ctx context.Context, command *cobra.Command, request initRequest) error {
			return runInteractiveInit(ctx, command, initCmd.InteractiveInitDeps{
				FS:              common.FS,
				Output:          common.Output,
				Logger:          common.Logger,
				MinecraftClient: common.MinecraftClient,
				RunTea:          runTeaProgram,
			}, initCmd.InteractiveInitOptions{
				ConfigPath: request.ConfigPath,
				Quiet:      opts.Quiet,
				Debug:      opts.Debug,
			})
		},
	}
}
