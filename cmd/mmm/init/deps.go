package init

import (
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/spf13/cobra"
)

func newInitDeps(cmd *cobra.Command, common cmddeps.CommonDeps) initDeps {
	return initDeps{
		fs:              common.FS,
		minecraftClient: common.MinecraftClient,
		prompter: terminalPrompter{
			in:  cmd.InOrStdin(),
			out: cmd.OutOrStdout(),
		},
		logger:    common.Logger,
		output:    common.Output,
		telemetry: telemetry.RecordCommand,
		runTea:    defaultRunTea,
	}
}
