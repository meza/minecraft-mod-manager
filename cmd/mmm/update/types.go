package update

import (
	"context"

	"github.com/meza/minecraft-mod-manager/cmd/mmm/install"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/tui"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

type updateOptions struct {
	ConfigPath     string
	NonInteractive bool
	Quiet          bool
	Debug          bool
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
}

type fetcher func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error)

type downloader func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error

type installer func(context.Context, *cobra.Command, string, bool, bool) (install.Result, error)

type modUpdateCandidate struct {
	ConfigIndex int
	Mod         models.Mod
}

type modUpdateOutcome struct {
	ConfigIndex int
	LockIndex   int
	NewName     string
	NewInstall  models.ModInstall
	LogEvents   []logEvent
	Updated     bool
	Error       error
}

type logEventKind int

const (
	logEventKindLog logEventKind = iota
	logEventKindError
	logEventKindDebug
)

type logEvent struct {
	Kind      logEventKind
	Message   string
	ForceShow bool
}

type updateCounts struct {
	updated int
	failed  int
}

type updateContext struct {
	meta      config.Metadata
	cfg       models.ModsJSON
	lock      []models.ModInstall
	colorMode tui.ColorMode
}
