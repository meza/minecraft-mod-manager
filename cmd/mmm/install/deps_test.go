package install

import (
	"context"
	"io"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
)

func TestNewInstallDepsWiresInitRunner(t *testing.T) {
	restore := runInteractiveInit
	var captured initCmd.InteractiveInitOptions
	runInteractiveInit = func(ctx context.Context, cmd *cobra.Command, initDeps initCmd.InteractiveInitDeps, opts initCmd.InteractiveInitOptions) error {
		captured = opts
		return nil
	}
	t.Cleanup(func() { runInteractiveInit = restore })

	common := cmddeps.CommonDeps{
		FS:              afero.NewMemMapFs(),
		Output:          output.New(io.Discard, io.Discard, false),
		Logger:          logger.New(io.Discard, io.Discard, false, false),
		Clients:         platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		MinecraftClient: httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)),
	}

	deps := newInstallDeps(common, installOptions{Quiet: true, Debug: true}, func(telemetry.CommandTelemetry) {})
	err := deps.runInit(context.Background(), &cobra.Command{}, initRequest{configPath: "/cfg/modlist.json"})
	assert.NoError(t, err)
	assert.Equal(t, "/cfg/modlist.json", captured.ConfigPath)
	assert.True(t, captured.Quiet)
	assert.True(t, captured.Debug)
}
