package change

import (
	"context"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/time/rate"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
)

type changeOptions struct {
	ConfigPath  string
	GameVersion string
	Unattended  bool
	Quiet       bool
	Debug       bool
	Force       bool
}

type changeResult struct {
	TargetVersion  string
	TotalMods      int
	SkippedMods    int
	DownloadedMods int
	ExitCode       int
	Interactive    bool
}

type changeRunner func(context.Context, *cobra.Command, changeOptions, changeDeps) (changeResult, error)

type changeDeps struct {
	fs              afero.Fs
	output          *output.Output
	logger          *logger.Logger
	clients         platform.Clients
	downloadClient  httpclient.Doer
	minecraftClient httpclient.Doer
	limiter         *rate.Limiter
	downloader      downloader
	fetchMod        fetcher
	latestVersion   latestVersionFetcher
	isValidVersion  versionValidator
	runTea          func(tea.Model, ...tea.ProgramOption) (tea.Model, error)

	readConfig  func(context.Context, afero.Fs, config.Metadata) (models.ModsJSON, error)
	ensureLock  func(context.Context, afero.Fs, config.Metadata) ([]models.ModInstall, error)
	writeConfig func(context.Context, afero.Fs, config.Metadata, models.ModsJSON) error
	writeLock   func(context.Context, afero.Fs, config.Metadata, []models.ModInstall) error
	removeFile  func(afero.Fs, string) error
	renameFile  func(afero.Fs, string, string) error
	removeAll   func(afero.Fs, string) error
	mkdirAll    func(afero.Fs, string, os.FileMode) error

	telemetry func(telemetry.CommandTelemetry)
}

type fetcher func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error)

type downloader func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error

type latestVersionFetcher func(context.Context, httpclient.Doer) (string, error)

type versionValidator func(context.Context, string, httpclient.Doer) (bool, error)
