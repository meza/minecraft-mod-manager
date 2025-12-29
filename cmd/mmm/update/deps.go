package update

import (
	"context"

	"github.com/meza/minecraft-mod-manager/cmd/mmm/install"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/spf13/cobra"
)

func newUpdateDeps(common cmddeps.CommonDeps, installCommand *cobra.Command) updateDeps {
	return updateDeps{
		fs:         common.FS,
		logger:     common.Logger,
		output:     common.Output,
		clients:    common.Clients,
		fetchMod:   platform.FetchMod,
		downloader: httpclient.DownloadFile,
		install: func(ctx context.Context, _ *cobra.Command, configPath string, quiet bool, debug bool) (install.Result, error) {
			return install.Run(ctx, installCommand, configPath, quiet, debug)
		},
		telemetry: telemetry.RecordCommand,
	}
}
