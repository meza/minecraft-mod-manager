package update

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/cmd/mmm/install"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/locksync"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

type updateOptions struct {
	ConfigPath string
	Unattended bool
	Quiet      bool
	Debug      bool
	LockSync   locksync.PolicyFlags
}

type updateDeps struct {
	fs         afero.Fs
	logger     *logger.Logger
	output     *output.Output
	clients    platform.Clients
	fetchMod   fetcher
	downloader downloader
	install    installer
	telemetry  func(telemetry.CommandTelemetry)
	runTea     teaRunner
	runInit    initRunner
}

type fetcher func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error)

type downloader func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error

type installer func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error)

type teaRunner func(tea.Model, ...tea.ProgramOption) (tea.Model, error)

type initRequest struct {
	ConfigPath string
}

type initRunner func(context.Context, *cobra.Command, initRequest) error

type modUpdateCandidate struct {
	ConfigIndex int
	Mod         models.Mod
}

type modUpdateOutcome struct {
	ConfigIndex int
	LockIndex   int
	NewName     string
	NewInstall  models.ModInstall
	Result      updateOutcomeResult
	FailReason  string
}

type updateCounts struct {
	updated int
	failed  int
}

type updateContext struct {
	meta           config.Metadata
	cfg            models.ModsJSON
	lock           []models.ModInstall
	colorMode      view.ColorMode
	shouldContinue bool
}

type updateItemStatus int

const (
	updateItemStatusPending updateItemStatus = iota
	updateItemStatusUpdating
	updateItemStatusDownloading
	updateItemStatusUpToDate
	updateItemStatusUpdated
	updateItemStatusSkipped
	updateItemStatusFailed
)

type updateProgress struct {
	ratio      float64
	downloaded int64
	total      int64
}

type updateItem struct {
	ConfigIndex int
	LockIndex   int
	Mod         models.Mod
	DisplayName string
	Status      updateItemStatus
	FailReason  string
	Progress    *updateProgress
}

type updateOutcomeResult int

const (
	updateOutcomeUpToDate updateOutcomeResult = iota
	updateOutcomeUpdated
	updateOutcomeSkipped
	updateOutcomeFailed
)

type updateExecutionErrorType int

const (
	updateExecutionErrorNone updateExecutionErrorType = iota
	updateExecutionErrorWriteLock
	updateExecutionErrorWriteConfig
	updateExecutionErrorCanceled
	updateExecutionErrorUnknown
)

type updateExecutionOutcome struct {
	items      []updateItem
	err        error
	errType    updateExecutionErrorType
	lockPath   string
	configPath string
}
