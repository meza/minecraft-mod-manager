package install

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	curseforgeFingerprint "github.com/meza/curseforge-fingerprint-go"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/time/rate"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/mmmignore"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/modpath"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/tui"
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
	curseforgeFingerprintMatch func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error)
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

func Command() *cobra.Command {
	return commandWithRunner(runInstall)
}

func commandWithRunner(runner installRunner) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "install",
		Aliases: []string{"i"},
		Short:   i18n.T("cmd.install.short"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInstallCommand(cmd, runner)
		},
	}

	return cmd
}

// Run executes the install consistency check without emitting install telemetry.
// It is used by other commands (for example `update`) that need install semantics
// as a prerequisite.
func Run(ctx context.Context, cmd *cobra.Command, configPath string, quiet bool, debug bool) (Result, error) {
	log := logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), quiet, debug)
	limiter := rate.NewLimiter(rate.Inf, 0)

	opts := installOptions{
		ConfigPath: configPath,
		Quiet:      quiet,
		Debug:      debug,
	}

	return runInstall(ctx, cmd, opts, defaultInstallDeps(log, limiter, func(telemetry.CommandTelemetry) {}))
}

func runInstallCommand(cmd *cobra.Command, runner installRunner) error {
	ctx, span := perf.StartSpan(cmd.Context(), "app.command.install")

	opts, err := installOptionsFromFlags(cmd)
	if err != nil {
		span.SetAttributes(attribute.Bool("success", false))
		span.End()
		return err
	}

	log := logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), opts.Quiet, opts.Debug)
	limiter := rate.NewLimiter(rate.Inf, 0)
	deps := defaultInstallDeps(log, limiter, telemetry.RecordCommand)

	result, err := runner(ctx, cmd, opts, deps)
	span.SetAttributes(attribute.Bool("success", err == nil))
	span.End()

	recordInstallTelemetry(deps.telemetry, result, err)
	return err
}

func installOptionsFromFlags(cmd *cobra.Command) (installOptions, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return installOptions{}, err
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return installOptions{}, err
	}
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return installOptions{}, err
	}

	return installOptions{
		ConfigPath: configPath,
		Quiet:      quiet,
		Debug:      debug,
	}, nil
}

func defaultInstallDeps(log *logger.Logger, limiter *rate.Limiter, telemetryRecorder func(telemetry.CommandTelemetry)) installDeps {
	return installDeps{
		fs:         afero.NewOsFs(),
		logger:     log,
		clients:    platform.DefaultClients(limiter),
		downloader: httpclient.DownloadFile,
		fetchMod:   platform.FetchMod,
		telemetry:  telemetryRecorder,

		curseforgeFingerprint:      curseforgeFingerprint.GetFingerprintFor,
		modrinthVersionForSha:      defaultModrinthVersionForSha,
		modrinthProjectTitle:       defaultModrinthProjectTitle,
		curseforgeFingerprintMatch: defaultCurseforgeFingerprintMatch,
		curseforgeProjectName:      defaultCurseforgeProjectName,
	}
}

func recordInstallTelemetry(telemetryRecorder func(telemetry.CommandTelemetry), result Result, err error) {
	payload := telemetry.CommandTelemetry{
		Command:     "install",
		Success:     err == nil,
		Error:       err,
		ExitCode:    0,
		Interactive: false,
		Extra: map[string]interface{}{
			"numberOfMods": result.InstalledCount,
		},
	}
	if err != nil {
		payload.ExitCode = 1
	}
	telemetryRecorder(payload)
}

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

var errUnresolvedFiles = errors.New("unresolved files in mods folder")
var errInstallFailures = errors.New("one or more mods failed to install")

func runInstall(ctx context.Context, cmd *cobra.Command, opts installOptions, deps installDeps) (Result, error) {
	meta := config.NewMetadata(opts.ConfigPath)

	cfg, lock, err := loadInstallConfig(ctx, deps, meta)
	if err != nil {
		return Result{}, err
	}

	colorize := tui.IsTerminalWriter(cmd.OutOrStdout())

	preflight, err := preflightInstall(ctx, meta, cfg, lock, deps, colorize)
	if err != nil {
		return Result{}, err
	}

	if mkdirErr := deps.fs.MkdirAll(meta.ModsFolderPath(cfg), 0755); mkdirErr != nil {
		return Result{}, mkdirErr
	}

	configured, err := installConfiguredMods(installConfiguredInputs{
		ctx:      ctx,
		meta:     meta,
		cfg:      cfg,
		lock:     lock,
		deps:     deps,
		colorize: colorize,
	})
	if err != nil {
		return Result{}, err
	}

	if err := persistInstallConfig(ctx, deps, meta, configured); err != nil {
		return Result{}, err
	}

	if configured.failedCount > 0 {
		return Result{InstalledCount: len(configured.cfg.Mods), UnmanagedFound: preflight.unmanagedFound}, errInstallFailures
	}

	deps.logger.Log(messageWithIcon(tui.SuccessIcon(colorize), i18n.T("cmd.install.success")), true)
	return Result{InstalledCount: len(configured.cfg.Mods), UnmanagedFound: preflight.unmanagedFound}, nil
}

func loadInstallConfig(ctx context.Context, deps installDeps, meta config.Metadata) (models.ModsJSON, []models.ModInstall, error) {
	cfg, err := config.ReadConfig(ctx, deps.fs, meta)
	if err != nil {
		return models.ModsJSON{}, nil, err
	}

	lock, err := config.EnsureLock(ctx, deps.fs, meta)
	if err != nil {
		return models.ModsJSON{}, nil, err
	}

	return cfg, lock, nil
}

func preflightInstall(ctx context.Context, meta config.Metadata, cfg models.ModsJSON, lock []models.ModInstall, deps installDeps, colorize bool) (scanReportOutcome, error) {
	preflight, err := preflightUnknownFiles(preflightInputs{
		ctx:      ctx,
		meta:     meta,
		cfg:      cfg,
		lock:     lock,
		deps:     deps,
		colorize: colorize,
	})
	if err != nil {
		return scanReportOutcome{}, err
	}
	if preflight.unresolved {
		deps.logger.Error(i18n.T("cmd.install.error.unresolved"))
		return scanReportOutcome{}, errUnresolvedFiles
	}
	return preflight, nil
}

func persistInstallConfig(ctx context.Context, deps installDeps, meta config.Metadata, configured installConfiguredOutcome) error {
	if err := config.WriteLock(ctx, deps.fs, meta, configured.lock); err != nil {
		return err
	}
	if err := config.WriteConfig(ctx, deps.fs, meta, configured.cfg); err != nil {
		return err
	}
	return nil
}

func installConfiguredMods(input installConfiguredInputs) (installConfiguredOutcome, error) {
	failedCount := 0
	cfg := input.cfg
	lock := input.lock

	for i := range cfg.Mods {
		mod := cfg.Mods[i]

		version := modVersionLabel(mod)
		input.deps.logger.Debug(i18n.T("cmd.install.debug.checking", i18n.Tvars{
			Data: &i18n.TData{
				"name":     mod.Name,
				"version":  version,
				"platform": mod.Type,
			},
		}))

		outcome, err := installMod(installModInputs{
			ctx:      input.ctx,
			meta:     input.meta,
			cfg:      cfg,
			lock:     lock,
			mod:      mod,
			deps:     input.deps,
			colorize: input.colorize,
		})
		if err != nil {
			return installConfiguredOutcome{}, err
		}
		if outcome.failed {
			failedCount++
			continue
		}
		if outcome.newName != "" {
			cfg.Mods[i].Name = outcome.newName
		}
		if outcome.lockEntry != nil {
			lock = append(lock, *outcome.lockEntry)
		}
	}

	return installConfiguredOutcome{
		cfg:         cfg,
		lock:        lock,
		failedCount: failedCount,
	}, nil
}

type modInstallOutcome struct {
	failed    bool
	newName   string
	lockEntry *models.ModInstall
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func modVersionLabel(mod models.Mod) string {
	if mod.Version != nil && strings.TrimSpace(*mod.Version) != "" {
		return strings.TrimSpace(*mod.Version)
	}
	return "latest"
}

func effectiveAllowedReleaseTypes(mod models.Mod, cfg models.ModsJSON) []models.ReleaseType {
	if len(mod.AllowedReleaseTypes) > 0 {
		return mod.AllowedReleaseTypes
	}
	return cfg.DefaultAllowedReleaseTypes
}

func lockIndexFor(mod models.Mod, lock []models.ModInstall) int {
	for i := range lock {
		if lock[i].Type == mod.Type && lock[i].ID == mod.ID {
			return i
		}
	}
	return -1
}

func installMod(input installModInputs) (modInstallOutcome, error) {
	lockIndex := lockIndexFor(input.mod, input.lock)
	if lockIndex >= 0 {
		return installFromLock(input.ctx, input.meta, input.cfg, input.mod, input.lock[lockIndex], input.deps)
	}
	return installFromRemote(input)
}

func installFromLock(ctx context.Context, meta config.Metadata, cfg models.ModsJSON, mod models.Mod, installEntry models.ModInstall, deps installDeps) (modInstallOutcome, error) {
	normalizedFileName, normalizeErr := modfilename.Normalize(installEntry.FileName)
	if normalizeErr != nil {
		deps.logger.Error(i18n.T("cmd.install.error.invalid_filename_lock", i18n.Tvars{
			Data: &i18n.TData{
				"name": mod.Name,
				"file": modfilename.Display(installEntry.FileName),
			},
		}))
		return modInstallOutcome{failed: true}, nil //nolint:nilerr // Keep install flow running after logging invalid lock entry.
	}
	installEntry.FileName = normalizedFileName
	if err := ensureLockInstall(ctx, meta, cfg, mod, installEntry, deps); err != nil {
		if message, handled := integrityErrorMessage(err, mod.Name); handled {
			deps.logger.Error(message)
			return modInstallOutcome{failed: true}, nil
		}
		return modInstallOutcome{}, err
	}
	return modInstallOutcome{}, nil
}

func installFromRemote(input installModInputs) (modInstallOutcome, error) {
	remote, fetchErr := fetchRemoteModForInstall(input.ctx, input.mod, input.cfg, input.deps)
	if fetchErr != nil {
		if handleExpectedFetchError(fetchErr, input) {
			return modInstallOutcome{}, nil
		}
		return modInstallOutcome{}, fetchErr
	}

	normalizedRemote, outcome := normalizeRemoteForInstall(remote, input.mod, input.deps)
	if outcome.failed {
		return outcome, nil
	}

	input.deps.logger.Log(i18n.T("cmd.install.download.missing", i18n.Tvars{
		Data: &i18n.TData{
			"name":     input.mod.Name,
			"platform": input.mod.Type,
		},
	}), true)

	resolvedDestination, outcome, err := resolveRemoteDestination(input.meta, input.cfg, normalizedRemote, input.mod, input.deps)
	if err != nil || outcome.failed {
		return outcome, err
	}

	if handled, err := downloadRemoteMod(input.ctx, normalizedRemote, resolvedDestination, input.mod, input.deps); err != nil {
		return modInstallOutcome{}, err
	} else if handled {
		return modInstallOutcome{failed: true}, nil
	}

	lockEntry := buildLockEntry(input.mod, normalizedRemote)
	return modInstallOutcome{newName: normalizedRemote.Name, lockEntry: &lockEntry}, nil
}

func fetchRemoteModForInstall(ctx context.Context, mod models.Mod, cfg models.ModsJSON, deps installDeps) (platform.RemoteMod, error) {
	return deps.fetchMod(ctx, mod.Type, mod.ID, platform.FetchOptions{
		AllowedReleaseTypes: effectiveAllowedReleaseTypes(mod, cfg),
		GameVersion:         cfg.GameVersion,
		Loader:              cfg.Loader,
		AllowFallback:       mod.AllowVersionFallback != nil && *mod.AllowVersionFallback,
		FixedVersion:        optionalStringValue(mod.Version),
	}, deps.clients)
}

func downloadRemoteMod(ctx context.Context, remote platform.RemoteMod, resolvedDestination string, mod models.Mod, deps installDeps) (bool, error) {
	installer := modinstall.NewInstaller(deps.fs, modinstall.Downloader(deps.downloader))
	if err := installer.DownloadAndVerify(ctx, remote.DownloadURL, resolvedDestination, remote.Hash, downloadClient(deps.clients), &noopSender{}); err != nil {
		if message, handled := integrityErrorMessage(err, mod.Name); handled {
			deps.logger.Error(message)
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func buildLockEntry(mod models.Mod, remote platform.RemoteMod) models.ModInstall {
	return models.ModInstall{
		Type:        mod.Type,
		ID:          mod.ID,
		Name:        remote.Name,
		FileName:    remote.FileName,
		ReleasedOn:  remote.ReleaseDate,
		Hash:        remote.Hash,
		DownloadURL: remote.DownloadURL,
	}
}

func normalizeRemoteForInstall(remote platform.RemoteMod, mod models.Mod, deps installDeps) (platform.RemoteMod, modInstallOutcome) {
	normalizedFileName, err := modfilename.Normalize(remote.FileName)
	if err != nil {
		deps.logger.Error(i18n.T("cmd.install.error.invalid_filename_remote", i18n.Tvars{
			Data: &i18n.TData{
				"name": mod.Name,
				"file": modfilename.Display(remote.FileName),
			},
		}))
		return platform.RemoteMod{}, modInstallOutcome{failed: true}
	}
	remote.FileName = normalizedFileName

	if strings.TrimSpace(remote.Hash) == "" {
		deps.logger.Error(i18n.T("cmd.install.error.missing_hash_remote", i18n.Tvars{
			Data: &i18n.TData{"name": mod.Name},
		}))
		return platform.RemoteMod{}, modInstallOutcome{failed: true}
	}

	return remote, modInstallOutcome{}
}

func resolveRemoteDestination(meta config.Metadata, cfg models.ModsJSON, remote platform.RemoteMod, mod models.Mod, deps installDeps) (string, modInstallOutcome, error) {
	destination := filepath.Join(meta.ModsFolderPath(cfg), remote.FileName)
	resolvedDestination, err := modpath.ResolveWritablePath(deps.fs, meta.ModsFolderPath(cfg), destination)
	if err != nil {
		if message, handled := integrityErrorMessage(err, mod.Name); handled {
			deps.logger.Error(message)
			return "", modInstallOutcome{failed: true}, nil
		}
		return "", modInstallOutcome{}, err
	}
	return resolvedDestination, modInstallOutcome{}, nil
}

func ensureLockInstall(ctx context.Context, meta config.Metadata, cfg models.ModsJSON, mod models.Mod, install models.ModInstall, deps installDeps) error {
	installer := modinstall.NewInstaller(deps.fs, modinstall.Downloader(deps.downloader))
	result, err := installer.EnsureLockedFile(ctx, meta, cfg, install, downloadClient(deps.clients), &noopSender{})
	if err != nil {
		return err
	}

	switch result.Reason {
	case modinstall.EnsureReasonMissing:
		deps.logger.Log(i18n.T("cmd.install.download.missing", i18n.Tvars{
			Data: &i18n.TData{
				"name":     mod.Name,
				"platform": install.Type,
			},
		}), true)
	case modinstall.EnsureReasonHashMismatch:
		deps.logger.Log(i18n.T("cmd.install.download.hash_mismatch", i18n.Tvars{
			Data: &i18n.TData{"name": mod.Name},
		}), true)
	}
	return nil
}

func integrityErrorMessage(err error, modName string) (string, bool) {
	var missingHash modinstall.MissingHashError
	if errors.As(err, &missingHash) {
		return i18n.T("cmd.install.error.missing_hash_lock", i18n.Tvars{
			Data: &i18n.TData{"name": modName},
		}), true
	}

	var hashMismatch modinstall.HashMismatchError
	if errors.As(err, &hashMismatch) {
		return i18n.T("cmd.install.error.hash_mismatch", i18n.Tvars{
			Data: &i18n.TData{"name": modName},
		}), true
	}

	var outsideRoot modpath.OutsideRootError
	if errors.As(err, &outsideRoot) {
		return i18n.T("cmd.install.error.symlink_outside_mods", i18n.Tvars{
			Data: &i18n.TData{
				"name": modName,
				"path": outsideRoot.ResolvedPath,
				"root": outsideRoot.Root,
			},
		}), true
	}

	return "", false
}

func handleExpectedFetchError(err error, input installModInputs) bool {
	var notFound *platform.ModNotFoundError
	if errors.As(err, &notFound) {
		input.deps.logger.Log(messageWithIcon(tui.ErrorIcon(input.colorize), i18n.T("cmd.install.error.mod_not_found", i18n.Tvars{
			Data: &i18n.TData{
				"name":     input.mod.Name,
				"id":       input.mod.ID,
				"platform": input.mod.Type,
			},
		})), true)
		return true
	}

	var noFile *platform.NoCompatibleFileError
	if errors.As(err, &noFile) {
		input.deps.logger.Log(messageWithIcon(tui.ErrorIcon(input.colorize), i18n.T("cmd.install.error.no_file", i18n.Tvars{
			Data: &i18n.TData{
				"name":     input.mod.Name,
				"id":       input.mod.ID,
				"platform": input.mod.Type,
			},
		})), true)
		return true
	}

	return false
}

func messageWithIcon(icon string, message string) string {
	return fmt.Sprintf("%s %s", icon, message)
}

func preflightUnknownFiles(input preflightInputs) (scanReportOutcome, error) {
	files, err := listModFiles(input.deps.fs, input.meta, input.cfg)
	if err != nil {
		return scanReportOutcome{}, err
	}

	nonManaged := make([]string, 0)
	for _, file := range files {
		if !fileIsManaged(file, input.lock) {
			nonManaged = append(nonManaged, file)
		}
	}

	if len(nonManaged) == 0 {
		return scanReportOutcome{}, nil
	}

	scanned, err := scanFiles(input.ctx, nonManaged, input.deps)
	if err != nil {
		var lookupFailure *platformLookupFailure
		if errors.As(err, &lookupFailure) {
			logPlatformLookupFailure(input.deps.logger, lookupFailure, input.colorize)
			return scanReportOutcome{unresolved: true}, nil
		}
		return scanReportOutcome{}, err
	}

	return reportScanResults(scanReportInputs{
		scanned:  scanned,
		cfg:      input.cfg,
		lock:     input.lock,
		deps:     input.deps,
		colorize: input.colorize,
	})
}

func scanFiles(ctx context.Context, files []string, deps installDeps) ([]scannedFile, error) {
	candidates, err := buildScanCandidates(files, deps)
	if err != nil {
		return nil, err
	}

	if err := applyCurseforgeHits(ctx, candidates.results, candidates.fingerprintToIndices, candidates.fingerprints, deps); err != nil {
		return nil, newPlatformLookupFailure(models.CURSEFORGE, files, err)
	}

	if err := applyModrinthHits(ctx, candidates.results, deps); err != nil {
		return nil, err
	}

	for i := range candidates.results {
		candidates.results[i].Hits = sortHitsPreferModrinth(candidates.results[i].Hits)
	}

	return candidates.results, nil
}

type scanCandidates struct {
	results              []scannedFile
	fingerprints         []int
	fingerprintToIndices map[int][]int
}

type platformLookupFailure struct {
	Platform     models.Platform
	Files        []string
	Reason       string
	DebugDetails string
}

func (failure *platformLookupFailure) Error() string {
	return failure.Reason
}

func newPlatformLookupFailure(platform models.Platform, files []string, err error) *platformLookupFailure {
	if err == nil {
		return &platformLookupFailure{
			Platform: platform,
			Files:    files,
			Reason:   i18n.T("cmd.platform.error.reason.unknown"),
		}
	}

	summary, _ := clierrors.SummarizePlatformError(err, platform)
	return &platformLookupFailure{
		Platform:     platform,
		Files:        files,
		Reason:       summary.Reason,
		DebugDetails: summary.DebugDetails,
	}
}

func logPlatformLookupFailure(log *logger.Logger, failure *platformLookupFailure, colorize bool) {
	if failure == nil || log == nil {
		return
	}
	if strings.TrimSpace(failure.DebugDetails) != "" {
		log.Debug(i18n.T("cmd.install.debug.platform_error", i18n.Tvars{
			Data: &i18n.TData{
				"platform": failure.Platform,
				"details":  failure.DebugDetails,
			},
		}))
	}
	for _, filePath := range failure.Files {
		log.Log(messageWithIcon(tui.ErrorIcon(colorize), i18n.T("cmd.install.unsure.platform_error", i18n.Tvars{
			Data: &i18n.TData{
				"file":     filepath.Base(filePath),
				"platform": failure.Platform,
				"reason":   failure.Reason,
			},
		})), true)
	}
}

func buildScanCandidates(files []string, deps installDeps) (scanCandidates, error) {
	results := make([]scannedFile, 0, len(files))
	fingerprints := make([]int, 0, len(files))
	fingerprintToIndices := make(map[int][]int, len(files))

	for index, filePath := range files {
		sha, err := sha1ForFile(deps.fs, filePath)
		if err != nil {
			return scanCandidates{}, err
		}

		fingerprint := int(deps.curseforgeFingerprint(filePath))
		fingerprints = append(fingerprints, fingerprint)
		fingerprintToIndices[fingerprint] = append(fingerprintToIndices[fingerprint], index)

		results = append(results, scannedFile{
			Path: filePath,
			Sha1: sha,
		})
	}

	sort.Ints(fingerprints)

	return scanCandidates{
		results:              results,
		fingerprints:         fingerprints,
		fingerprintToIndices: fingerprintToIndices,
	}, nil
}

func applyCurseforgeHits(ctx context.Context, results []scannedFile, fingerprintToIndices map[int][]int, fingerprints []int, deps installDeps) error {
	curseforgeByFingerprint, err := curseforgeMatchesByFingerprint(ctx, fingerprints, deps)
	if err != nil {
		return err
	}

	for fingerprint, hit := range curseforgeByFingerprint {
		indices, ok := fingerprintToIndices[fingerprint]
		if !ok || len(indices) == 0 {
			continue
		}
		for _, index := range indices {
			results[index].Hits = append(results[index].Hits, hit)
		}
	}

	return nil
}

func applyModrinthHits(ctx context.Context, results []scannedFile, deps installDeps) error {
	for i := range results {
		version, err := deps.modrinthVersionForSha(ctx, results[i].Sha1, deps.clients.Modrinth)
		if err != nil {
			var notFound *modrinth.VersionNotFoundError
			if errors.As(err, &notFound) {
				continue
			}
			return newPlatformLookupFailure(models.MODRINTH, []string{results[i].Path}, err)
		}

		name, err := deps.modrinthProjectTitle(ctx, version.ProjectID, deps.clients.Modrinth)
		if err != nil {
			return newPlatformLookupFailure(models.MODRINTH, []string{results[i].Path}, err)
		}

		results[i].Hits = append(results[i].Hits, scanHit{
			Platform: models.MODRINTH,
			Project:  version.ProjectID,
			Name:     name,
		})
	}

	return nil
}

func sortHitsPreferModrinth(hits []scanHit) []scanHit {
	if len(hits) <= 1 {
		return hits
	}

	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Platform == models.MODRINTH && hits[j].Platform != models.MODRINTH {
			return true
		}
		if hits[i].Platform != models.MODRINTH && hits[j].Platform == models.MODRINTH {
			return false
		}
		return hits[i].Platform < hits[j].Platform
	})

	return hits
}

func curseforgeMatchesByFingerprint(ctx context.Context, fingerprints []int, deps installDeps) (map[int]scanHit, error) {
	unique := uniqueInts(fingerprints)
	if len(unique) == 0 {
		return map[int]scanHit{}, nil
	}

	result, err := deps.curseforgeFingerprintMatch(ctx, unique, deps.clients.Curseforge)
	if err != nil {
		return nil, err
	}

	matches := make(map[int]scanHit, len(result.Matches))
	for _, file := range result.Matches {
		projectID := fmt.Sprintf("%d", file.ProjectID)
		name, err := deps.curseforgeProjectName(ctx, projectID, deps.clients.Curseforge)
		if err != nil {
			return nil, err
		}

		matches[file.Fingerprint] = scanHit{
			Platform: models.CURSEFORGE,
			Project:  projectID,
			Name:     name,
		}
	}
	return matches, nil
}

func uniqueInts(values []int) []int {
	seen := make(map[int]struct{}, len(values))
	result := make([]int, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func reportScanResults(input scanReportInputs) (scanReportOutcome, error) {
	outcome := scanReportOutcome{}

	for _, item := range input.scanned {
		if len(item.Hits) == 0 {
			continue
		}

		matchedModIndex := findConfiguredModIndex(input.cfg, item.Hits)
		if matchedModIndex < 0 {
			outcome.unmanagedFound = true
			name := item.Hits[0].Name
			if input.colorize {
				name = tui.TitleStyle.Copy().Bold(true).Render(name)
			}
			input.deps.logger.Log(tui.SuccessIcon(input.colorize)+i18n.T("cmd.install.unmanaged.found", i18n.Tvars{
				Data: &i18n.TData{"name": name},
			}), true)
			continue
		}

		mod := input.cfg.Mods[matchedModIndex]
		lockIndex := lockIndexFor(mod, input.lock)
		if lockIndex < 0 {
			input.deps.logger.Log(messageWithIcon(tui.ErrorIcon(input.colorize), i18n.T("cmd.install.unsure.lock_missing", i18n.Tvars{
				Data: &i18n.TData{"name": item.Hits[0].Name},
			})), true)
			outcome.unresolved = true
			continue
		}

		if !strings.EqualFold(input.lock[lockIndex].Hash, item.Sha1) {
			input.deps.logger.Log(messageWithIcon(tui.ErrorIcon(input.colorize), i18n.T("cmd.install.unsure.hash_mismatch", i18n.Tvars{
				Data: &i18n.TData{"name": item.Hits[0].Name},
			})), true)
			outcome.unresolved = true
		}
	}

	return outcome, nil
}

func findConfiguredModIndex(cfg models.ModsJSON, hits []scanHit) int {
	for _, hit := range hits {
		for i := range cfg.Mods {
			if cfg.Mods[i].Type == hit.Platform && cfg.Mods[i].ID == hit.Project {
				return i
			}
		}
	}
	return -1
}

func fileIsManaged(filePath string, installations []models.ModInstall) bool {
	filename := filepath.Base(filePath)
	for _, install := range installations {
		if install.FileName == filename {
			return true
		}
	}
	return false
}

func listModFiles(fs afero.Fs, meta config.Metadata, cfg models.ModsJSON) ([]string, error) {
	all, err := afero.ReadDir(fs, meta.ModsFolderPath(cfg))
	if err != nil {
		return nil, err
	}

	candidates := make([]string, 0, len(all))
	for _, entry := range all {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".jar") {
			continue
		}
		candidates = append(candidates, filepath.Join(meta.ModsFolderPath(cfg), entry.Name()))
	}

	patterns, err := mmmignore.ListPatterns(fs, meta.Dir())
	if err != nil {
		return nil, err
	}

	filtered := make([]string, 0, len(candidates))
	for _, path := range candidates {
		if mmmignore.IsIgnored(meta.Dir(), path, patterns) {
			continue
		}
		filtered = append(filtered, path)
	}
	return filtered, nil
}

func sha1ForFile(fs afero.Fs, path string) (string, error) {
	file, err := fs.Open(path)
	if err != nil {
		return "", err
	}

	h := sha1.New()
	if _, err := io.Copy(h, file); err != nil {
		closeErr := file.Close()
		if closeErr != nil {
			return "", errors.Join(err, closeErr)
		}
		return "", err
	}

	if err := file.Close(); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

type noopSender struct{}

func (sender *noopSender) Send(msg tea.Msg) { _ = msg }

func downloadClient(clients platform.Clients) httpclient.Doer {
	if clients.Curseforge != nil {
		return clients.Curseforge
	}
	return clients.Modrinth
}

func defaultModrinthVersionForSha(ctx context.Context, sha1 string, doer httpclient.Doer) (*modrinth.Version, error) {
	client := modrinth.NewClient(doer)
	return modrinth.GetVersionForHash(ctx, modrinth.NewVersionHashLookup(sha1, modrinth.SHA1), client)
}

func defaultModrinthProjectTitle(ctx context.Context, projectID string, doer httpclient.Doer) (string, error) {
	client := modrinth.NewClient(doer)
	project, err := modrinth.GetProject(ctx, projectID, client)
	if err != nil {
		return "", err
	}
	return project.Title, nil
}

func defaultCurseforgeFingerprintMatch(ctx context.Context, fingerprints []int, doer httpclient.Doer) (*curseforge.FingerprintResult, error) {
	client := curseforge.NewClient(doer)
	return curseforge.GetFingerprintsMatches(ctx, fingerprints, client)
}

func defaultCurseforgeProjectName(ctx context.Context, projectID string, doer httpclient.Doer) (string, error) {
	client := curseforge.NewClient(doer)
	project, err := curseforge.GetProject(ctx, projectID, client)
	if err != nil {
		return "", err
	}
	return project.Name, nil
}
