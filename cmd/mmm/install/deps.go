package install

import (
	"context"

	curseforgeFingerprint "github.com/meza/curseforge-fingerprint-go"
	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/spf13/cobra"
)

func newInstallDeps(common cmddeps.CommonDeps, options installOptions, telemetryRecorder func(telemetry.CommandTelemetry)) installDeps {
	return installDeps{
		fs:         common.FS,
		logger:     common.Logger,
		output:     common.Output,
		clients:    common.Clients,
		downloader: httpclient.DownloadFile,
		fetchMod:   platform.FetchMod,
		telemetry:  telemetryRecorder,
		runTea:     defaultRunTea,

		curseforgeFingerprint:      curseforgeFingerprint.GetFingerprintFor,
		modrinthVersionForSha:      defaultModrinthVersionForSha,
		modrinthProjectTitle:       defaultModrinthProjectTitle,
		curseforgeFingerprintMatch: defaultCurseforgeFingerprintMatch,
		curseforgeProjectName:      defaultCurseforgeProjectName,
		runInit: func(ctx context.Context, command *cobra.Command, request initRequest) error {
			return runInteractiveInit(ctx, command, initCmd.InteractiveInitDeps{
				FS:              common.FS,
				Output:          common.Output,
				Logger:          common.Logger,
				MinecraftClient: common.MinecraftClient,
				RunTea:          defaultRunTea,
			}, initCmd.InteractiveInitOptions{
				ConfigPath: request.configPath,
				Quiet:      options.Quiet,
				Debug:      options.Debug,
			})
		},
	}
}
