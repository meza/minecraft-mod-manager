package install

import (
	"context"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

type installDeps struct {
	fs         afero.Fs
	logger     *logger.Logger
	clients    platform.Clients
	downloader downloader
	fetchMod   fetcher
	telemetry  func(telemetry.CommandTelemetry)

	curseforgeFingerprint      func(string) uint32
	modrinthVersionForSha      func(context.Context, string, httpclient.Doer) (*modrinth.Version, error)
	modrinthProjectTitle       func(context.Context, string, httpclient.Doer) (string, error)
	curseforgeFingerprintMatch func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error)
	curseforgeProjectName      func(context.Context, string, httpclient.Doer) (string, error)
}

type fetcher func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error)

type downloader func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error

type installOptions struct {
	ConfigPath string
	Quiet      bool
	Debug      bool
}

type Result struct {
	InstalledCount int
	UnmanagedFound bool
}

type installConfiguredInputs struct {
	ctx      context.Context
	meta     config.Metadata
	cfg      models.ModsJSON
	lock     []models.ModInstall
	deps     installDeps
	colorize bool
}

type installConfiguredOutcome struct {
	cfg         models.ModsJSON
	lock        []models.ModInstall
	failedCount int
}

type installModInputs struct {
	ctx      context.Context
	meta     config.Metadata
	cfg      models.ModsJSON
	lock     []models.ModInstall
	mod      models.Mod
	deps     installDeps
	colorize bool
}

type preflightInputs struct {
	ctx      context.Context
	meta     config.Metadata
	cfg      models.ModsJSON
	lock     []models.ModInstall
	deps     installDeps
	colorize bool
}

type scanReportInputs struct {
	scanned  []scannedFile
	cfg      models.ModsJSON
	lock     []models.ModInstall
	deps     installDeps
	colorize bool
}

type scanReportOutcome struct {
	unresolved     bool
	unmanagedFound bool
}

type installRunner func(context.Context, *cobra.Command, installOptions, installDeps) (Result, error)

type scanHit struct {
	Platform models.Platform
	Project  string
	Name     string
}

type scannedFile struct {
	Path string
	Sha1 string
	Hits []scanHit
}

type modInstallOutcome struct {
	failed    bool
	newName   string
	lockEntry *models.ModInstall
}

type scanCandidates struct {
	results              []scannedFile
	fingerprints         []uint32
	fingerprintToIndices map[uint32][]int
}

type platformLookupFailure struct {
	Platform     models.Platform
	Files        []string
	Reason       string
	DebugDetails string
}
