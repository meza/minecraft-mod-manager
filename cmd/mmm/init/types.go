package init

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

type initRunner func(context.Context, *cobra.Command, initOptions, initDeps, config.Metadata) error

type initOptions struct {
	ConfigPath   string
	Unattended   bool
	Quiet        bool
	Debug        bool
	Force        bool
	Loader       models.Loader
	GameVersion  string
	ReleaseTypes []models.ReleaseType
	ModsFolder   string
	Provided     providedFlags
}

type providedFlags struct {
	Loader       bool
	GameVersion  bool
	ReleaseTypes bool
	ModsFolder   bool
}

type initDeps struct {
	fs              afero.Fs
	minecraftClient httpclient.Doer
	logger          *logger.Logger
	output          *output.Output
	telemetry       func(telemetry.CommandTelemetry)
	runTea          func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error)
}

type loaderFlag struct {
	value models.Loader
}

type gameVersionNormalizationMode int

const (
	gameVersionUnattended gameVersionNormalizationMode = iota
	gameVersionInteractive
)
