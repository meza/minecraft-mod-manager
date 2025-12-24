// Package add implements the add command.
package add

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/time/rate"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/modpath"
	"github.com/meza/minecraft-mod-manager/internal/modsetup"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/tui"
	"go.opentelemetry.io/otel/attribute"
)

type addOptions struct {
	Platform             string
	ProjectID            string
	ConfigPath           string
	Quiet                bool
	Debug                bool
	Version              string
	AllowVersionFallback bool
}

type addDeps struct {
	fs              afero.Fs
	clients         platform.Clients
	minecraftClient httpclient.Doer
	logger          *logger.Logger
	fetchMod        fetcher
	downloader      downloader
	runTea          func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error)
}

type fetcher func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error)

type downloader func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error

var errAborted = errors.New("add aborted")

type addRunner func(context.Context, *perf.Span, *cobra.Command, addOptions, addDeps) (telemetry.CommandTelemetry, error)

// Command builds the add command.
func Command() *cobra.Command {
	return commandWithRunner(runAdd)
}

func commandWithRunner(runner addRunner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <platform> <id>",
		Short: i18n.T("cmd.add.short"),
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddCommand(cmd, args, runner)
		},
		Aliases:       []string{"a"},
		SilenceUsage:  false,
		SilenceErrors: false,
	}

	cmd.Flags().String("version", "", i18n.T("cmd.add.flag.version"))
	cmd.Flags().Bool("allow-version-fallback", false, i18n.T("cmd.add.flag.allow_version_fallback"))

	cmd.ValidArgsFunction = func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return []string{string(models.CURSEFORGE), string(models.MODRINTH)}, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

func runAddCommand(cmd *cobra.Command, args []string, runner addRunner) error {
	ctx, span := startAddSpan(cmd.Context(), args)

	options, err := readAddOptions(cmd, args)
	if err != nil {
		span.SetAttributes(attribute.Bool("success", false))
		span.End()
		return err
	}

	log := logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), options.Quiet, options.Debug)
	deps := defaultAddDeps(log, rate.NewLimiter(rate.Inf, 0))

	telemetryPayload, err := runner(ctx, span, cmd, options, deps)
	errToReturn := normalizeAddError(err)
	endAddSpan(span, errToReturn)
	recordAddTelemetry(telemetryPayload, errToReturn)
	return errToReturn
}

func startAddSpan(ctx context.Context, args []string) (context.Context, *perf.Span) {
	return perf.StartSpan(ctx, "app.command.add",
		perf.WithAttributes(
			attribute.String("platform", args[0]),
			attribute.String("id", args[1]),
		),
	)
}

func normalizeAddError(err error) error {
	if errors.Is(err, errAborted) {
		return nil
	}
	return err
}

func endAddSpan(span *perf.Span, err error) {
	span.SetAttributes(attribute.Bool("success", err == nil))
	span.End()
}

func recordAddTelemetry(telemetryPayload telemetry.CommandTelemetry, err error) {
	telemetryPayload.Success = err == nil
	if telemetryPayload.Success {
		telemetryPayload.Error = nil
		telemetryPayload.ExitCode = 0
	} else {
		telemetryPayload.ExitCode = 1
	}
	telemetry.RecordCommand(telemetryPayload)
}

func runAdd(ctx context.Context, commandSpan *perf.Span, cmd *cobra.Command, opts addOptions, deps addDeps) (telemetry.CommandTelemetry, error) {
	meta := config.NewMetadata(opts.ConfigPath)
	useTUI := tui.ShouldUseTUI(opts.Quiet, cmd.InOrStdin(), cmd.OutOrStdout())
	setupCoordinator := modsetup.NewSetupCoordinator(deps.fs, deps.minecraftClient, modsetup.Downloader(deps.downloader))

	cfg, lock, err := prepareAddConfig(ctx, opts, meta, setupCoordinator)
	if err != nil {
		return addFailureTelemetryWithoutArgs(useTUI, err), err
	}

	platformValue := normalizePlatform(opts.Platform)
	projectID := opts.ProjectID

	install, installFound := findLockInstall(lock, platformValue, projectID)
	if modsetup.ModExists(cfg, platformValue, projectID) && installFound {
		return handleExistingInstall(ctx, commandSpan, meta, cfg, install, platformValue, projectID, opts, deps, useTUI)
	}

	remoteMod, resolvedPlatform, resolvedID, fetchErr := resolveRemoteModWithSpan(ctx, commandSpan, cfg, opts, platformValue, projectID, deps, useTUI, cmd.InOrStdin(), cmd.OutOrStdout())
	if fetchErr != nil {
		return addFailureTelemetry(platformValue, projectID, opts, useTUI, fetchErr), fetchErr
	}

	remoteMod, err = normalizeRemoteModFileName(remoteMod)
	if err != nil {
		return addFailureTelemetry(resolvedPlatform, resolvedID, opts, useTUI, err), err
	}

	_, err = ensureRemoteMod(ctx, meta, cfg, remoteMod, resolvedPlatform, resolvedID, deps)
	if err != nil {
		return addFailureTelemetry(resolvedPlatform, resolvedID, opts, useTUI, err), err
	}

	if err := persistAdd(ctx, meta, cfg, lock, remoteMod, resolvedPlatform, resolvedID, opts, setupCoordinator); err != nil {
		return addFailureTelemetry(resolvedPlatform, resolvedID, opts, useTUI, err), err
	}

	logAddSuccess(deps.logger, remoteMod.Name, resolvedID, resolvedPlatform)
	return addSuccessTelemetry(resolvedPlatform, resolvedID, opts, useTUI), nil
}

func prepareAddConfig(ctx context.Context, opts addOptions, meta config.Metadata, setupCoordinator *modsetup.SetupCoordinator) (models.ModsJSON, []models.ModInstall, error) {
	prepareCtx, prepareSpan := perf.StartSpan(ctx, "app.command.add.stage.prepare", perf.WithAttributes(attribute.String("config_path", opts.ConfigPath)))
	cfg, lock, err := setupCoordinator.EnsureConfigAndLock(prepareCtx, meta, opts.Quiet)
	prepareSpan.SetAttributes(attribute.Bool("success", err == nil))
	prepareSpan.End()
	return cfg, lock, err
}

func resolveRemoteModWithSpan(ctx context.Context, commandSpan *perf.Span, cfg models.ModsJSON, opts addOptions, platformValue models.Platform, projectID string, deps addDeps, useTUI bool, in io.Reader, out io.Writer) (platform.RemoteMod, models.Platform, string, error) {
	resolveCtx, resolveSpan := perf.StartSpan(ctx, "app.command.add.stage.resolve",
		perf.WithAttributes(
			attribute.String("platform", string(platformValue)),
			attribute.String("project_id", projectID),
			attribute.Bool("use_tui", useTUI),
			attribute.Bool("quiet", opts.Quiet),
		),
	)
	remoteMod, resolvedPlatform, resolvedID, fetchErr := resolveRemoteMod(resolveCtx, commandSpan, cfg, opts, platformValue, projectID, deps, useTUI, in, out)
	resolveSpan.SetAttributes(
		attribute.Bool("success", fetchErr == nil),
		attribute.String("resolved_platform", string(resolvedPlatform)),
		attribute.String("resolved_project_id", resolvedID),
	)
	resolveSpan.End()
	return remoteMod, resolvedPlatform, resolvedID, fetchErr
}

func normalizeRemoteModFileName(remoteMod platform.RemoteMod) (platform.RemoteMod, error) {
	normalizedFileName, err := modfilename.Normalize(remoteMod.FileName)
	if err != nil {
		message := i18n.T("cmd.add.error.invalid_filename_remote", i18n.Tvars{
			Data: &i18n.TData{
				"name": remoteMod.Name,
				"file": modfilename.Display(remoteMod.FileName),
			},
		})
		return platform.RemoteMod{}, errors.New(message)
	}
	remoteMod.FileName = normalizedFileName
	return remoteMod, nil
}

func ensureRemoteMod(ctx context.Context, meta config.Metadata, cfg models.ModsJSON, remoteMod platform.RemoteMod, resolvedPlatform models.Platform, resolvedID string, deps addDeps) (modinstall.EnsureResult, error) {
	downloadCtx, downloadSpan := perf.StartSpan(ctx, "app.command.add.stage.download",
		perf.WithAttributes(
			attribute.String("url", remoteMod.DownloadURL),
			attribute.String("platform", string(resolvedPlatform)),
			attribute.String("project_id", resolvedID),
			attribute.String("file_name", remoteMod.FileName),
		),
	)

	// For idempotency, avoid re-downloading when the remote file already exists locally and the SHA-1 matches.
	ensureInstaller := modinstall.NewInstaller(deps.fs, modinstall.Downloader(deps.downloader))
	destination := filepath.Join(meta.ModsFolderPath(cfg), remoteMod.FileName)
	ensureResult, err := ensureInstaller.EnsureLockedFile(downloadCtx, meta, cfg, models.ModInstall{
		FileName:    remoteMod.FileName,
		Hash:        remoteMod.Hash,
		DownloadURL: remoteMod.DownloadURL,
	}, downloadClient(deps.clients), nil)
	if err != nil {
		if message, handled := integrityErrorMessage(err, remoteMod.Name); handled {
			err = errors.New(message)
		}
		downloadSpan.SetAttributes(attribute.Bool("success", false))
		downloadSpan.End()
		return modinstall.EnsureResult{}, err
	}
	downloadSpan.SetAttributes(
		attribute.Bool("success", true),
		attribute.String("path", destination),
		attribute.String("reason", string(ensureResult.Reason)),
	)
	downloadSpan.End()
	return ensureResult, nil
}

func persistAdd(ctx context.Context, meta config.Metadata, cfg models.ModsJSON, lock []models.ModInstall, remoteMod platform.RemoteMod, resolvedPlatform models.Platform, resolvedID string, opts addOptions, setupCoordinator *modsetup.SetupCoordinator) error {
	_, persistSpan := perf.StartSpan(ctx, "app.command.add.stage.persist",
		perf.WithAttributes(
			attribute.String("config_path", opts.ConfigPath),
			attribute.String("platform", string(resolvedPlatform)),
			attribute.String("project_id", resolvedID),
		),
	)
	_, _, _, err := setupCoordinator.EnsurePersisted(ctx, meta, cfg, lock, resolvedPlatform, resolvedID, remoteMod, modsetup.EnsurePersistOptions{
		Version:              opts.Version,
		AllowVersionFallback: opts.AllowVersionFallback,
	})
	if err != nil {
		persistSpan.SetAttributes(attribute.Bool("success", false))
		persistSpan.End()
		return err
	}
	persistSpan.SetAttributes(attribute.Bool("success", true))
	persistSpan.End()
	return nil
}

func logAddSuccess(log *logger.Logger, modName string, resolvedID string, resolvedPlatform models.Platform) {
	log.Log(i18n.T("cmd.add.success", i18n.Tvars{
		Data: &i18n.TData{
			"name":     modName,
			"id":       resolvedID,
			"platform": resolvedPlatform,
		},
	}), true)
}

func addSuccessTelemetry(platformValue models.Platform, projectID string, opts addOptions, useTUI bool) telemetry.CommandTelemetry {
	return telemetry.CommandTelemetry{
		Command:     "add",
		Success:     true,
		ExitCode:    0,
		Interactive: useTUI,
		Arguments:   addTelemetryArgs(platformValue, projectID, opts),
	}
}

func integrityErrorMessage(err error, modName string) (string, bool) {
	var missingHash modinstall.MissingHashError
	if errors.As(err, &missingHash) {
		return i18n.T("cmd.add.error.missing_hash_remote", i18n.Tvars{
			Data: &i18n.TData{"name": modName},
		}), true
	}

	var hashMismatch modinstall.HashMismatchError
	if errors.As(err, &hashMismatch) {
		return i18n.T("cmd.add.error.hash_mismatch", i18n.Tvars{
			Data: &i18n.TData{"name": modName},
		}), true
	}

	var outsideRoot modpath.OutsideRootError
	if errors.As(err, &outsideRoot) {
		return i18n.T("cmd.add.error.symlink_outside_mods", i18n.Tvars{
			Data: &i18n.TData{
				"name": modName,
				"path": outsideRoot.ResolvedPath,
				"root": outsideRoot.Root,
			},
		}), true
	}

	return "", false
}

func resolveRemoteMod(ctx context.Context, commandSpan *perf.Span, cfg models.ModsJSON, opts addOptions, platformValue models.Platform, projectID string, deps addDeps, useTUI bool, in io.Reader, out io.Writer) (platform.RemoteMod, models.Platform, string, error) {
	deps.logger.Debug(fmt.Sprintf("fetching %s/%s (loader=%s, gameVersion=%s, fallback=%t, fixedVersion=%s)", platformValue, projectID, cfg.Loader, cfg.GameVersion, opts.AllowVersionFallback, opts.Version))

	remote, err := fetchRemoteModOnce(ctx, cfg, opts, platformValue, projectID, deps, useTUI)
	if err == nil {
		return remote, platformValue, projectID, nil
	}

	logFetchFailure(deps.logger, platformValue, projectID, err)
	return resolveRemoteModFromError(ctx, commandSpan, cfg, opts, platformValue, projectID, deps, useTUI, in, out, err)
}

func fetchRemoteModOnce(ctx context.Context, cfg models.ModsJSON, opts addOptions, platformValue models.Platform, projectID string, deps addDeps, useTUI bool) (platform.RemoteMod, error) {
	attemptCtx, attemptSpan := perf.StartSpan(ctx, "app.command.add.resolve.attempt",
		perf.WithAttributes(
			attribute.Int("attempt", 0),
			attribute.String("source", "cli"),
			attribute.String("platform", string(platformValue)),
			attribute.String("project_id", projectID),
			attribute.Bool("use_tui", useTUI),
			attribute.Bool("quiet", opts.Quiet),
		),
	)
	remote, err := deps.fetchMod(attemptCtx, platformValue, projectID, platform.FetchOptions{
		AllowedReleaseTypes: cfg.DefaultAllowedReleaseTypes,
		GameVersion:         cfg.GameVersion,
		Loader:              cfg.Loader,
		AllowFallback:       opts.AllowVersionFallback,
		FixedVersion:        opts.Version,
	}, deps.clients)
	attemptSpan.SetAttributes(attribute.Bool("success", err == nil))
	if err != nil {
		attemptSpan.SetAttributes(attribute.String("error_type", fmt.Sprintf("%T", err)))
	}
	attemptSpan.End()
	return remote, err
}

func logFetchFailure(log *logger.Logger, platformValue models.Platform, projectID string, err error) {
	log.Debug(fmt.Sprintf("fetch failed for %s/%s: %v", platformValue, projectID, err))
	if inner := errors.Unwrap(err); inner != nil {
		log.Debug(fmt.Sprintf("fetch failure detail: %v", inner))
	}
}

func resolveRemoteModFromError(ctx context.Context, commandSpan *perf.Span, cfg models.ModsJSON, opts addOptions, platformValue models.Platform, projectID string, deps addDeps, useTUI bool, in io.Reader, out io.Writer, err error) (platform.RemoteMod, models.Platform, string, error) {
	var unknownPlatformError *platform.UnknownPlatformError
	if errors.As(err, &unknownPlatformError) {
		return resolveUnknownPlatform(ctx, commandSpan, cfg, opts, platformValue, projectID, deps, useTUI, in, out, unknownPlatformError)
	}

	var modNotFoundError *platform.ModNotFoundError
	if errors.As(err, &modNotFoundError) {
		return resolveModNotFound(ctx, commandSpan, cfg, opts, platformValue, projectID, deps, useTUI, in, out, err)
	}

	var noCompatibleFileError *platform.NoCompatibleFileError
	if errors.As(err, &noCompatibleFileError) {
		return resolveNoCompatibleFile(ctx, commandSpan, cfg, opts, platformValue, projectID, deps, useTUI, in, out, err)
	}

	return platform.RemoteMod{}, platformValue, projectID, err
}

func resolveUnknownPlatform(ctx context.Context, commandSpan *perf.Span, cfg models.ModsJSON, opts addOptions, platformValue models.Platform, projectID string, deps addDeps, useTUI bool, in io.Reader, out io.Writer, unknownPlatformError *platform.UnknownPlatformError) (platform.RemoteMod, models.Platform, string, error) {
	if opts.Quiet || !useTUI {
		message := errorMessageForUnknownPlatform(unknownPlatformError.Platform)
		deps.logger.Error(message)
		return platform.RemoteMod{}, platformValue, projectID, errors.New(message)
	}
	return resolveRemoteModWithTUI(ctx, commandSpan, addTUIStateUnknownPlatformSelect, cfg, opts, platformValue, projectID, deps, in, out)
}

func resolveModNotFound(ctx context.Context, commandSpan *perf.Span, cfg models.ModsJSON, opts addOptions, platformValue models.Platform, projectID string, deps addDeps, useTUI bool, in io.Reader, out io.Writer, err error) (platform.RemoteMod, models.Platform, string, error) {
	if opts.Quiet || !useTUI {
		deps.logger.Error(errorMessageForModNotFound(projectID, platformValue))
		return platform.RemoteMod{}, platformValue, projectID, err
	}
	return resolveRemoteModWithTUI(ctx, commandSpan, addTUIStateModNotFoundConfirm, cfg, opts, platformValue, projectID, deps, in, out)
}

func resolveNoCompatibleFile(ctx context.Context, commandSpan *perf.Span, cfg models.ModsJSON, opts addOptions, platformValue models.Platform, projectID string, deps addDeps, useTUI bool, in io.Reader, out io.Writer, err error) (platform.RemoteMod, models.Platform, string, error) {
	if opts.Quiet || !useTUI {
		deps.logger.Error(errorMessageForNoFile(projectID, platformValue))
		return platform.RemoteMod{}, platformValue, projectID, err
	}
	return resolveRemoteModWithTUI(ctx, commandSpan, addTUIStateNoFileConfirm, cfg, opts, platformValue, projectID, deps, in, out)
}

func findLockInstall(lock []models.ModInstall, platformValue models.Platform, projectID string) (models.ModInstall, bool) {
	for i := range lock {
		if lock[i].Type == platformValue && lock[i].ID == projectID {
			return lock[i], true
		}
	}
	return models.ModInstall{}, false
}

func modNameForConfig(cfg models.ModsJSON, platformValue models.Platform, projectID string) string {
	for i := range cfg.Mods {
		if cfg.Mods[i].Type == platformValue && cfg.Mods[i].ID == projectID {
			return cfg.Mods[i].Name
		}
	}
	return projectID
}

func normalizePlatform(value string) models.Platform {
	switch strings.ToLower(value) {
	case string(models.CURSEFORGE):
		return models.CURSEFORGE
	case string(models.MODRINTH):
		return models.MODRINTH
	default:
		return models.Platform(strings.ToLower(value))
	}
}

func alternatePlatform(platform models.Platform) models.Platform {
	if platform == models.CURSEFORGE {
		return models.MODRINTH
	}
	return models.CURSEFORGE
}

func downloadClient(clients platform.Clients) httpclient.Doer {
	if clients.Curseforge != nil {
		return clients.Curseforge
	}
	return clients.Modrinth
}

func resolveRemoteModWithTUI(ctx context.Context, commandSpan *perf.Span, initialState addTUIState, cfg models.ModsJSON, opts addOptions, platformValue models.Platform, projectID string, deps addDeps, in io.Reader, out io.Writer) (platform.RemoteMod, models.Platform, string, error) {
	if commandSpan != nil {
		commandSpan.AddEvent("app.command.add.tui.open", perf.WithEventAttributes(
			attribute.Int("initial_state", int(initialState)),
			attribute.String("platform", string(platformValue)),
			attribute.String("project_id", projectID),
		))
	}

	tuiCtx, tuiSpan := perf.StartSpan(ctx, "tui.add.session",
		perf.WithAttributes(
			attribute.String("platform", string(platformValue)),
			attribute.String("project_id", projectID),
			attribute.Int("initial_state", int(initialState)),
		),
	)
	attempt := 0
	model := newAddTUIModel(tuiCtx, tuiSpan, initialState, platformValue, projectID, cfg, buildAddTUIFetchCmd(tuiCtx, cfg, opts, platformValue, projectID, deps, &attempt))

	if deps.runTea == nil {
		return platform.RemoteMod{}, platformValue, projectID, errors.New("missing add dependencies: runTea")
	}

	result, err := runAddTUIProgram(deps.runTea, model, tuiSpan, in, out)
	if err != nil {
		return platform.RemoteMod{}, platformValue, projectID, err
	}

	typed, ok := result.(addTUIModel)
	if !ok {
		return platform.RemoteMod{}, platformValue, projectID, errors.New("unexpected add TUI result model")
	}

	return typed.result()
}

func buildAddTUIFetchCmd(tuiCtx context.Context, cfg models.ModsJSON, opts addOptions, platformValue models.Platform, projectID string, deps addDeps, attempt *int) func(models.Platform, string) tea.Cmd {
	return func(platformValue models.Platform, projectID string) tea.Cmd {
		return func() tea.Msg {
			*attempt += 1
			attemptNumber := *attempt
			attemptCtx, attemptSpan := perf.StartSpan(tuiCtx, "app.command.add.resolve.attempt",
				perf.WithAttributes(
					attribute.Int("attempt", attemptNumber),
					attribute.String("source", "tui"),
					attribute.String("platform", string(platformValue)),
					attribute.String("project_id", projectID),
					attribute.Bool("quiet", opts.Quiet),
				),
			)
			remote, err := deps.fetchMod(attemptCtx, platformValue, projectID, platform.FetchOptions{
				AllowedReleaseTypes: cfg.DefaultAllowedReleaseTypes,
				GameVersion:         cfg.GameVersion,
				Loader:              cfg.Loader,
				AllowFallback:       opts.AllowVersionFallback,
				FixedVersion:        opts.Version,
			}, deps.clients)
			attemptSpan.SetAttributes(attribute.Bool("success", err == nil))
			if err != nil {
				attemptSpan.SetAttributes(attribute.String("error_type", fmt.Sprintf("%T", err)))
			}
			attemptSpan.End()
			return addTUIFetchResultMsg{
				platform:  platformValue,
				projectID: projectID,
				remote:    remote,
				err:       err,
			}
		}
	}
}

func runAddTUIProgram(runTea func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error), model tea.Model, tuiSpan *perf.Span, in io.Reader, out io.Writer) (tea.Model, error) {
	result, err := runTea(model, tui.ProgramOptions(in, out)...)
	if err != nil {
		tuiSpan.SetAttributes(attribute.Bool("success", false))
		tuiSpan.End()
		return nil, err
	}
	tuiSpan.SetAttributes(attribute.Bool("success", true))
	tuiSpan.End()
	return result, nil
}

func readAddOptions(cmd *cobra.Command, args []string) (addOptions, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return addOptions{}, err
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return addOptions{}, err
	}
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return addOptions{}, err
	}
	version, err := cmd.Flags().GetString("version")
	if err != nil {
		return addOptions{}, err
	}
	allowFallback, err := cmd.Flags().GetBool("allow-version-fallback")
	if err != nil {
		return addOptions{}, err
	}

	return addOptions{
		Platform:             args[0],
		ProjectID:            args[1],
		ConfigPath:           configPath,
		Quiet:                quiet,
		Debug:                debug,
		Version:              version,
		AllowVersionFallback: allowFallback,
	}, nil
}

func defaultAddDeps(log *logger.Logger, limiter *rate.Limiter) addDeps {
	return addDeps{
		fs:              afero.NewOsFs(),
		clients:         platform.DefaultClients(limiter),
		minecraftClient: httpclient.NewRLClient(limiter),
		logger:          log,
		fetchMod:        platform.FetchMod,
		downloader:      httpclient.DownloadFile,
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			return tea.NewProgram(model, options...).Run()
		},
	}
}

//nolint:funlen // Keeps existing-install remediation readable without altering behavior.
func handleExistingInstall(
	ctx context.Context,
	commandSpan *perf.Span,
	meta config.Metadata,
	cfg models.ModsJSON,
	install models.ModInstall,
	platformValue models.Platform,
	projectID string,
	opts addOptions,
	deps addDeps,
	useTUI bool,
) (telemetry.CommandTelemetry, error) {
	normalizedFileName, normalizeErr := modfilename.Normalize(install.FileName)
	if normalizeErr != nil {
		message := i18n.T("cmd.add.error.invalid_filename_lock", i18n.Tvars{
			Data: &i18n.TData{
				"name": modNameForConfig(cfg, platformValue, projectID),
				"file": modfilename.Display(install.FileName),
			},
		})
		err := errors.New(message)
		return addFailureTelemetry(platformValue, projectID, opts, useTUI, err), err
	}
	install.FileName = normalizedFileName

	installer := modinstall.NewInstaller(deps.fs, modinstall.Downloader(deps.downloader))
	ensureResult, ensureErr := installer.EnsureLockedFile(ctx, meta, cfg, install, downloadClient(deps.clients), nil)
	if ensureErr != nil {
		return addFailureTelemetry(platformValue, projectID, opts, useTUI, ensureErr), ensureErr
	}

	logEnsureResult(deps.logger, ensureResult.Reason, cfg, platformValue, projectID)

	if commandSpan != nil {
		commandSpan.AddEvent("app.command.add.outcome.already_exists", perf.WithEventAttributes(
			attribute.String("platform", string(platformValue)),
			attribute.String("project_id", projectID),
		))
	}
	deps.logger.Debug(i18n.T("cmd.add.debug.already_exists", i18n.Tvars{
		Data: &i18n.TData{
			"id":       projectID,
			"platform": platformValue,
		},
	}))
	return addExistingInstallTelemetry(platformValue, projectID, opts, useTUI, ensureResult.Reason), nil
}

func logEnsureResult(log *logger.Logger, reason modinstall.EnsureReason, cfg models.ModsJSON, platformValue models.Platform, projectID string) {
	switch reason {
	case modinstall.EnsureReasonMissing:
		log.Log(i18n.T("cmd.install.download.missing", i18n.Tvars{
			Data: &i18n.TData{
				"name":     modNameForConfig(cfg, platformValue, projectID),
				"platform": platformValue,
			},
		}), true)
	case modinstall.EnsureReasonHashMismatch:
		log.Log(i18n.T("cmd.install.download.hash_mismatch", i18n.Tvars{
			Data: &i18n.TData{"name": modNameForConfig(cfg, platformValue, projectID)},
		}), true)
	}
}

func addFailureTelemetry(platformValue models.Platform, projectID string, opts addOptions, useTUI bool, err error) telemetry.CommandTelemetry {
	return telemetry.CommandTelemetry{
		Command:     "add",
		Success:     false,
		Error:       err,
		ExitCode:    1,
		Interactive: useTUI,
		Arguments:   addTelemetryArgs(platformValue, projectID, opts),
	}
}

func addFailureTelemetryWithoutArgs(useTUI bool, err error) telemetry.CommandTelemetry {
	return telemetry.CommandTelemetry{
		Command:     "add",
		Success:     false,
		Error:       err,
		ExitCode:    1,
		Interactive: useTUI,
	}
}

func addExistingInstallTelemetry(platformValue models.Platform, projectID string, opts addOptions, useTUI bool, reason modinstall.EnsureReason) telemetry.CommandTelemetry {
	return telemetry.CommandTelemetry{
		Command:     "add",
		Success:     true,
		ExitCode:    0,
		Interactive: useTUI,
		Arguments:   addTelemetryArgs(platformValue, projectID, opts),
		Extra: map[string]interface{}{
			"flag":               "already-exists",
			"ensure_file_reason": string(reason),
		},
	}
}

func addTelemetryArgs(platformValue models.Platform, projectID string, opts addOptions) map[string]interface{} {
	return map[string]interface{}{
		"platform": platformValue,
		"id":       projectID,
		"version":  opts.Version,
		"fallback": opts.AllowVersionFallback,
	}
}
