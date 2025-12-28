package update

import (
	"context"

	"github.com/meza/minecraft-mod-manager/cmd/mmm/install"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func defaultUpdateDeps(out *output.Output, log *logger.Logger, installCommand *cobra.Command) updateDeps {
	limiter := httpclient.DefaultLimiter()

	return updateDeps{
		fs:         afero.NewOsFs(),
		logger:     log,
		output:     out,
		clients:    platform.DefaultClients(limiter),
		fetchMod:   platform.FetchMod,
		downloader: httpclient.DownloadFile,
		install: func(ctx context.Context, _ *cobra.Command, configPath string, quiet bool, debug bool) (install.Result, error) {
			return install.Run(ctx, installCommand, configPath, quiet, debug)
		},
		telemetry: telemetry.RecordCommand,
	}
}
