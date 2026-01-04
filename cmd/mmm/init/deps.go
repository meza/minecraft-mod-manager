package init

import (
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
)

func newInitDeps(common cmddeps.CommonDeps) initDeps {
	return initDeps{
		fs:              common.FS,
		minecraftClient: common.MinecraftClient,
		logger:          common.Logger,
		output:          common.Output,
		telemetry:       telemetry.RecordCommand,
		runTea:          defaultRunTea,
	}
}
