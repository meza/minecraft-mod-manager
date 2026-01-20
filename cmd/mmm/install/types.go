package install

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/locksync"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

type installDeps struct {
	fs         afero.Fs
	logger     *logger.Logger
	output     *output.Output
	clients    platform.Clients
	downloader downloader
	fetchMod   fetcher
	telemetry  func(telemetry.CommandTelemetry)
	runTea     func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error)
	runInit    initRunner
}

type fetcher func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error)

type downloader func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error

type installOptions struct {
	ConfigPath   string
	Unattended   bool
	Quiet        bool
	Debug        bool
	LockSync     locksync.PolicyFlags
	SkipLockSync bool
}

type Result struct {
	InstalledCount int
}

type installConfiguredInputs struct {
	ctx        context.Context
	meta       config.Metadata
	cfg        models.ModsJSON
	lock       []models.ModInstall
	deps       installDeps
	colorize   bool
	state      *installExecutionState
	writeState *installWriteState
}

type installConfiguredOutcome struct {
	failedCount int
	items       []installItem
}

type installModInputs struct {
	ctx      context.Context
	meta     config.Metadata
	cfg      models.ModsJSON
	lock     []models.ModInstall
	mod      models.Mod
	deps     installDeps
	colorize bool
	state    *installExecutionState
}

type installRunner func(context.Context, *cobra.Command, installOptions, installDeps) (Result, error)

type initRequest struct {
	configPath string
}

type initRunner func(context.Context, *cobra.Command, initRequest) error

type modInstallOutcome struct {
	failed        bool
	failureReason string
	newName       string
	lockEntry     *models.ModInstall
}
