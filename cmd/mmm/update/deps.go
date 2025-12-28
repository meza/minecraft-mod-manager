package update

import (
	"github.com/meza/minecraft-mod-manager/cmd/mmm/install"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func defaultUpdateDeps(cmd *cobra.Command, opts updateOptions) updateDeps {
	quietForOutput := opts.Quiet && !opts.Debug
	out := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), quietForOutput)
	log := logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, opts.Debug)
	limiter := httpclient.DefaultLimiter()

	return updateDeps{
		fs:         afero.NewOsFs(),
		logger:     log,
		output:     out,
		clients:    platform.DefaultClients(limiter),
		fetchMod:   platform.FetchMod,
		downloader: httpclient.DownloadFile,
		install:    install.Run,
		telemetry:  telemetry.RecordCommand,
	}
}
