package add

import (
	"context"
	"errors"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/locksync"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/modsetup"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

type addOptions struct {
	Platform             string
	ProjectID            string
	ConfigPath           string
	Unattended           bool
	Quiet                bool
	Debug                bool
	Version              string
	AllowVersionFallback bool
	LockSync             locksync.PolicyFlags
}

type addDeps struct {
	fs              afero.Fs
	clients         platform.Clients
	minecraftClient httpclient.Doer
	logger          *logger.Logger
	output          *output.Output
	fetchMod        fetcher
	downloader      downloader
	runTea          func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error)
}

type fetcher func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error)

type downloader func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error

var errAborted = errors.New("add aborted")

type addRunner func(context.Context, *perf.Span, *cobra.Command, addOptions, addDeps) (telemetry.CommandTelemetry, error)

type addResolveInputs struct {
	ctx           context.Context
	commandSpan   *perf.Span
	cfg           models.ModsJSON
	opts          addOptions
	platformValue models.Platform
	projectID     string
	deps          addDeps
	mode          interaction.ExecutionMode
	in            io.Reader
	out           io.Writer
}

type resolvedRemoteMod struct {
	remoteMod platform.RemoteMod
	platform  models.Platform
	projectID string
}

type addPersistInput struct {
	ctx              context.Context
	meta             config.Metadata
	cfg              models.ModsJSON
	lock             []models.ModInstall
	remoteMod        platform.RemoteMod
	resolvedPlatform models.Platform
	resolvedID       string
	opts             addOptions
	setupCoordinator *modsetup.SetupCoordinator
}

type existingInstallInput struct {
	ctx           context.Context
	commandSpan   *perf.Span
	meta          config.Metadata
	cfg           models.ModsJSON
	install       models.ModInstall
	platformValue models.Platform
	projectID     string
	opts          addOptions
	deps          addDeps
	mode          interaction.ExecutionMode
	in            io.Reader
	out           io.Writer
}

type existingInstallCheckInput struct {
	ctx           context.Context
	commandSpan   *perf.Span
	meta          config.Metadata
	cfg           models.ModsJSON
	lock          []models.ModInstall
	platformValue models.Platform
	projectID     string
	opts          addOptions
	deps          addDeps
	mode          interaction.ExecutionMode
	in            io.Reader
	out           io.Writer
}

type addRunState struct {
	meta             config.Metadata
	cfg              models.ModsJSON
	lock             []models.ModInstall
	mode             interaction.ExecutionMode
	shouldContinue   bool
	setupCoordinator *modsetup.SetupCoordinator
}

type addConfigState struct {
	cfg            models.ModsJSON
	lock           []models.ModInstall
	shouldContinue bool
}

type addIdentifiers struct {
	platformValue models.Platform
	projectID     string
}

type addAttemptOutcome struct {
	platformValue models.Platform
	projectID     string
	recovered     bool
	telemetry     telemetry.CommandTelemetry
}

type recoveryOutcome struct {
	platformValue models.Platform
	projectID     string
	recovered     bool
	err           error
}

type resolveStepOutcome struct {
	resolved      resolvedRemoteMod
	remoteMod     platform.RemoteMod
	recovered     bool
	platformValue models.Platform
	projectID     string
}

type ensureRemoteModInput struct {
	ctx              context.Context
	meta             config.Metadata
	cfg              models.ModsJSON
	remoteMod        platform.RemoteMod
	resolvedPlatform models.Platform
	resolvedID       string
	deps             addDeps
	mode             interaction.ExecutionMode
	colorMode        view.ColorMode
	input            io.Reader
	output           io.Writer
	quiet            bool
}

type ensureRemoteFileInput struct {
	ctx              context.Context
	installer        *modinstall.Installer
	meta             config.Metadata
	cfg              models.ModsJSON
	remoteMod        platform.RemoteMod
	resolvedPlatform models.Platform
	resolvedID       string
	deps             addDeps
	colorMode        view.ColorMode
	input            io.Reader
	output           io.Writer
}
