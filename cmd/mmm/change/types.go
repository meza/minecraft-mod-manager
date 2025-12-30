package change

import (
	"context"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/meza/minecraft-mod-manager/cmd/mmm/install"
	"github.com/meza/minecraft-mod-manager/cmd/mmm/test"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/tui"
)

type changeOptions struct {
	ConfigPath     string
	GameVersion    string
	NonInteractive bool
	Quiet          bool
	Debug          bool
	Force          bool
}

type changeResult struct {
	TargetVersion   string
	UnsupportedMods []models.Mod
	InstallResult   install.Result
	ExitCode        int
}

type changeRunner func(context.Context, *cobra.Command, changeOptions, changeDeps) (changeResult, error)

type changeTestRunner func(context.Context, *cobra.Command, test.Options, test.Deps, tui.ColorMode) (test.Result, error)

type changeInstallRunner func(context.Context, *cobra.Command, string, bool, bool) (install.Result, error)

type changeDeps struct {
	fs            afero.Fs
	testDeps      test.Deps
	testCmd       *cobra.Command
	installCmd    *cobra.Command
	colorMode     tui.ColorMode
	testRunner    changeTestRunner
	installRunner changeInstallRunner
	readConfig    func(context.Context, afero.Fs, config.Metadata) (models.ModsJSON, error)
	ensureLock    func(context.Context, afero.Fs, config.Metadata) ([]models.ModInstall, error)
	writeConfig   func(context.Context, afero.Fs, config.Metadata, models.ModsJSON) error
	writeLock     func(context.Context, afero.Fs, config.Metadata, []models.ModInstall) error
	removeFile    func(afero.Fs, string) error
	telemetry     func(telemetry.CommandTelemetry)
}
