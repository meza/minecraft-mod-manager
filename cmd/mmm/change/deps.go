package change

import (
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/meza/minecraft-mod-manager/cmd/mmm/install"
	"github.com/meza/minecraft-mod-manager/cmd/mmm/test"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	tui "github.com/meza/minecraft-mod-manager/internal/view"
)

func newChangeDeps(common cmddeps.CommonDeps, testCmd *cobra.Command, installCmd *cobra.Command) changeDeps {
	colorMode := tui.ColorDisabled
	if tui.IsTerminalWriter(testCmd.OutOrStdout()) {
		colorMode = tui.ColorEnabled
	}

	return changeDeps{
		fs:            common.FS,
		output:        common.Output,
		testDeps:      test.NewDeps(common),
		testCmd:       testCmd,
		installCmd:    installCmd,
		colorMode:     colorMode,
		testRunner:    test.RunWithDeps,
		installRunner: install.Run,
		readConfig:    config.ReadConfig,
		ensureLock:    config.EnsureLock,
		writeConfig:   config.WriteConfig,
		writeLock:     config.WriteLock,
		removeFile: func(fs afero.Fs, path string) error {
			return fs.Remove(path)
		},
		telemetry: telemetry.RecordCommand,
	}
}
