package init

import (
	"net/http"

	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func defaultInitDeps(cmd *cobra.Command, log *logger.Logger) initDeps {
	return initDeps{
		fs:              afero.NewOsFs(),
		minecraftClient: http.DefaultClient,
		prompter: terminalPrompter{
			in:  cmd.InOrStdin(),
			out: cmd.OutOrStdout(),
		},
		logger:    log,
		telemetry: telemetry.RecordCommand,
		runTea:    defaultRunTea,
	}
}
