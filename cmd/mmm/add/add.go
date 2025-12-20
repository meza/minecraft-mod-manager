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
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
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
	minecraftClient httpClient.Doer
	logger          *logger.Logger
	fetchMod        fetcher
	downloader      downloader
	runTea          func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error)
}

type fetcher func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error)

type downloader func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error

var errAborted = errors.New("add aborted")

type addRunner func(context.Context, *perf.Span, *cobra.Command, addOptions, addDeps) (telemetry.CommandTelemetry, error)

func Command() *cobra.Command {
	return commandWithRunner(runAdd)
}

func commandWithRunner(runner addRunner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <platform> <id>",
		Short: i18n.T("cmd.add.short"),
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			ctx, span := perf.StartSpan(cmd.Context(), "app.command.add",
				perf.WithAttributes(
					attribute.String("platform", args[0]),
					attribute.String("id", args[1]),
				),
			)

			configPath, err := cmd.Flags().GetString("config")
			if err != nil {
				span.SetAttributes(attribute.Bool("success", false))
				span.End()
				return err
			}
			quiet, err := cmd.Flags().GetBool("quiet")
			if err != nil {
				span.SetAttributes(attribute.Bool("success", false))
				span.End()
				return err
			}
			debug, err := cmd.Flags().GetBool("debug")
			if err != nil {
				span.SetAttributes(attribute.Bool("success", false))
				span.End()
				return err
			}
			version, err := cmd.Flags().GetString("version")
			if err != nil {
				span.SetAttributes(attribute.Bool("success", false))
				span.End()
				return err
			}
			allowFallback, err := cmd.Flags().GetBool("allow-version-fallback")
			if err != nil {
				span.SetAttributes(attribute.Bool("success", false))
				span.End()
				return err
			}

			log := logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), quiet, debug)

			limiter := rate.NewLimiter(rate.Inf, 0)

			telemetryPayload, err := runner(ctx, span, cmd, addOptions{
				Platform:             args[0],
				ProjectID:            args[1],
				ConfigPath:           configPath,
				Quiet:                quiet,
				Debug:                debug,
				Version:              version,
				AllowVersionFallback: allowFallback,
			}, addDeps{
				fs:              afero.NewOsFs(),
				clients:         platform.DefaultClients(limiter),
				minecraftClient: httpClient.NewRLClient(limiter),
				logger:          log,
				fetchMod:        platform.FetchMod,
				downloader:      httpClient.DownloadFile,
				runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
					return tea.NewProgram(model, options...).Run()
				},
			})

			errToReturn := err
			if errors.Is(err, errAborted) {
				errToReturn = nil
			}

			span.SetAttributes(attribute.Bool("success", errToReturn == nil))
			span.End()

			telemetryPayload.Success = errToReturn == nil
			if telemetryPayload.Success {
				telemetryPayload.Error = nil
				telemetryPayload.ExitCode = 0
			} else {
				telemetryPayload.ExitCode = 1
			}
			telemetry.RecordCommand(telemetryPayload)

			return errToReturn
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

func runAdd(ctx context.Context, commandSpan *perf.Span, cmd *cobra.Command, opts addOptions, deps addDeps) (telemetry.CommandTelemetry, error) {
	meta := config.NewMetadata(opts.ConfigPath)
	useTUI := tui.ShouldUseTUI(opts.Quiet, cmd.InOrStdin(), cmd.OutOrStdout())
	setupService := modsetup.NewService(deps.fs, deps.minecraftClient, modsetup.Downloader(deps.downloader))

	prepareCtx, prepareSpan := perf.StartSpan(ctx, "app.command.add.stage.prepare", perf.WithAttributes(attribute.String("config_path", opts.ConfigPath)))
	cfg, lock, err := setupService.EnsureConfigAndLock(prepareCtx, meta, opts.Quiet)
	prepareSpan.SetAttributes(attribute.Bool("success", err == nil))
	prepareSpan.End()
	if err != nil {
		return telemetry.CommandTelemetry{
			Command:     "add",
			Success:     false,
			Error:       err,
			ExitCode:    1,
			Interactive: useTUI,
		}, err
	}

	platformValue := normalizePlatform(opts.Platform)
	projectID := opts.ProjectID

	install, installFound := findLockInstall(lock, platformValue, projectID)
	if modsetup.ModExists(cfg, platformValue, projectID) && installFound {

		modInstall := modinstall.NewService(deps.fs, modinstall.Downloader(deps.downloader))
		ensureResult, err := modInstall.EnsureLockedFile(ctx, meta, cfg, install, downloadClient(deps.clients), nil)
		if err != nil {
			return telemetry.CommandTelemetry{
				Command:     "add",
				Success:     false,
				Error:       err,
				ExitCode:    1,
				Interactive: useTUI,
				Arguments: map[string]interface{}{
					"platform": platformValue,
					"id":       projectID,
					"version":  opts.Version,
					"fallback": opts.AllowVersionFallback,
				},
			}, err
		}

		switch ensureResult.Reason {
		case modinstall.EnsureReasonMissing:
			deps.logger.Log(i18n.T("cmd.install.download.missing", i18n.Tvars{
				Data: &i18n.TData{
					"name":     modNameForConfig(cfg, platformValue, projectID),
					"platform": platformValue,
				},
			}), true)
		case modinstall.EnsureReasonHashMismatch:
			deps.logger.Log(i18n.T("cmd.install.download.hash_mismatch", i18n.Tvars{
				Data: &i18n.TData{"name": modNameForConfig(cfg, platformValue, projectID)},
			}), true)
		}

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
		return telemetry.CommandTelemetry{
			Command:     "add",
			Success:     true,
			ExitCode:    0,
			Interactive: useTUI,
			Arguments: map[string]interface{}{
				"platform": platformValue,
				"id":       projectID,
				"version":  opts.Version,
				"fallback": opts.AllowVersionFallback,
			},
			Extra: map[string]interface{}{
				"flag":               "already-exists",
				"ensure_file_reason": string(ensureResult.Reason),
			},
		}, nil
	}

	resolveCtx, resolveSpan := perf.StartSpan(ctx, "app.command.add.stage.resolve",
		perf.WithAttributes(
			attribute.String("platform", string(platformValue)),
			attribute.String("project_id", projectID),
			attribute.Bool("use_tui", useTUI),
			attribute.Bool("quiet", opts.Quiet),
		),
	)
	remoteMod, resolvedPlatform, resolvedID, fetchErr := resolveRemoteMod(resolveCtx, commandSpan, cfg, opts, platformValue, projectID, deps, useTUI, cmd.InOrStdin(), cmd.OutOrStdout())
	resolveSpan.SetAttributes(
		attribute.Bool("success", fetchErr == nil),
		attribute.String("resolved_platform", string(resolvedPlatform)),
		attribute.String("resolved_project_id", resolvedID),
	)
	resolveSpan.End()
	if fetchErr != nil {
		payload := telemetry.CommandTelemetry{
			Command:     "add",
			Success:     false,
			Error:       fetchErr,
			ExitCode:    1,
			Interactive: useTUI,
			Arguments: map[string]interface{}{
				"platform": platformValue,
				"id":       projectID,
				"version":  opts.Version,
				"fallback": opts.AllowVersionFallback,
			},
		}
		return payload, fetchErr
	}

	downloadCtx, downloadSpan := perf.StartSpan(ctx, "app.command.add.stage.download",
		perf.WithAttributes(
			attribute.String("url", remoteMod.DownloadURL),
			attribute.String("platform", string(resolvedPlatform)),
			attribute.String("project_id", resolvedID),
			attribute.String("file_name", remoteMod.FileName),
		),
	)

	// For idempotency, avoid re-downloading when the remote file already exists locally and the SHA-1 matches.
	ensureRemote := modinstall.NewService(deps.fs, modinstall.Downloader(deps.downloader))
	destination := filepath.Join(meta.ModsFolderPath(cfg), remoteMod.FileName)
	ensureResult, err := ensureRemote.EnsureLockedFile(downloadCtx, meta, cfg, models.ModInstall{
		FileName:    remoteMod.FileName,
		Hash:        remoteMod.Hash,
		DownloadUrl: remoteMod.DownloadURL,
	}, downloadClient(deps.clients), nil)
	if err != nil {
		downloadSpan.SetAttributes(attribute.Bool("success", false))
		downloadSpan.End()
		return telemetry.CommandTelemetry{
			Command:     "add",
			Success:     false,
			Error:       err,
			ExitCode:    1,
			Interactive: useTUI,
			Arguments: map[string]interface{}{
				"platform": resolvedPlatform,
				"id":       resolvedID,
				"version":  opts.Version,
				"fallback": opts.AllowVersionFallback,
			},
		}, err
	}
	downloadSpan.SetAttributes(
		attribute.Bool("success", true),
		attribute.String("path", destination),
		attribute.String("reason", string(ensureResult.Reason)),
	)
	downloadSpan.End()

	_, persistSpan := perf.StartSpan(ctx, "app.command.add.stage.persist",
		perf.WithAttributes(
			attribute.String("config_path", opts.ConfigPath),
			attribute.String("platform", string(resolvedPlatform)),
			attribute.String("project_id", resolvedID),
		),
	)
	cfg, lock, _, err = setupService.EnsurePersisted(ctx, meta, cfg, lock, resolvedPlatform, resolvedID, remoteMod, modsetup.EnsurePersistOptions{
		Version:              opts.Version,
		AllowVersionFallback: opts.AllowVersionFallback,
	})
	if err != nil {
		persistSpan.SetAttributes(attribute.Bool("success", false))
		persistSpan.End()
		return telemetry.CommandTelemetry{
			Command:     "add",
			Success:     false,
			Error:       err,
			ExitCode:    1,
			Interactive: useTUI,
			Arguments: map[string]interface{}{
				"platform": resolvedPlatform,
				"id":       resolvedID,
				"version":  opts.Version,
				"fallback": opts.AllowVersionFallback,
			},
		}, err
	}
	persistSpan.SetAttributes(attribute.Bool("success", true))
	persistSpan.End()

	deps.logger.Log(i18n.T("cmd.add.success", i18n.Tvars{
		Data: &i18n.TData{
			"name":     remoteMod.Name,
			"id":       resolvedID,
			"platform": resolvedPlatform,
		},
	}), true)

	return telemetry.CommandTelemetry{
		Command:     "add",
		Success:     true,
		ExitCode:    0,
		Interactive: useTUI,
		Arguments: map[string]interface{}{
			"platform": resolvedPlatform,
			"id":       resolvedID,
			"version":  opts.Version,
			"fallback": opts.AllowVersionFallback,
		},
	}, nil
}

func resolveRemoteMod(ctx context.Context, commandSpan *perf.Span, cfg models.ModsJson, opts addOptions, platformValue models.Platform, projectID string, deps addDeps, useTUI bool, in io.Reader, out io.Writer) (platform.RemoteMod, models.Platform, string, error) {
	deps.logger.Debug(fmt.Sprintf("fetching %s/%s (loader=%s, gameVersion=%s, fallback=%t, fixedVersion=%s)", platformValue, projectID, cfg.Loader, cfg.GameVersion, opts.AllowVersionFallback, opts.Version))

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
	if err == nil {
		return remote, platformValue, projectID, nil
	}

	deps.logger.Debug(fmt.Sprintf("fetch failed for %s/%s: %v", platformValue, projectID, err))
	if inner := errors.Unwrap(err); inner != nil {
		deps.logger.Debug(fmt.Sprintf("fetch failure detail: %v", inner))
	}

	switch e := err.(type) {
	case *platform.UnknownPlatformError:
		if opts.Quiet || !useTUI {
			deps.logger.Error(errorMessageForUnknownPlatform(e.Platform))
			return platform.RemoteMod{}, platformValue, projectID, errors.New(errorMessageForUnknownPlatform(e.Platform))
		}
		return resolveRemoteModWithTUI(ctx, commandSpan, addTUIStateUnknownPlatformSelect, cfg, opts, platformValue, projectID, deps, in, out)
	case *platform.ModNotFoundError:
		if opts.Quiet || !useTUI {
			deps.logger.Error(errorMessageForModNotFound(projectID, platformValue))
			return platform.RemoteMod{}, platformValue, projectID, err
		}
		return resolveRemoteModWithTUI(ctx, commandSpan, addTUIStateModNotFoundConfirm, cfg, opts, platformValue, projectID, deps, in, out)
	case *platform.NoCompatibleFileError:
		if opts.Quiet || !useTUI {
			deps.logger.Error(errorMessageForNoFile(projectID, platformValue))
			return platform.RemoteMod{}, platformValue, projectID, err
		}
		return resolveRemoteModWithTUI(ctx, commandSpan, addTUIStateNoFileConfirm, cfg, opts, platformValue, projectID, deps, in, out)
	default:
		return platform.RemoteMod{}, platformValue, projectID, err
	}
}

func findLockInstall(lock []models.ModInstall, platformValue models.Platform, projectID string) (models.ModInstall, bool) {
	for i := range lock {
		if lock[i].Type == platformValue && lock[i].Id == projectID {
			return lock[i], true
		}
	}
	return models.ModInstall{}, false
}

func modNameForConfig(cfg models.ModsJson, platformValue models.Platform, projectID string) string {
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

func downloadClient(clients platform.Clients) httpClient.Doer {
	if clients.Curseforge != nil {
		return clients.Curseforge
	}
	return clients.Modrinth
}

func resolveRemoteModWithTUI(ctx context.Context, commandSpan *perf.Span, initialState addTUIState, cfg models.ModsJson, opts addOptions, platformValue models.Platform, projectID string, deps addDeps, in io.Reader, out io.Writer) (platform.RemoteMod, models.Platform, string, error) {
	if commandSpan != nil {
		commandSpan.AddEvent("app.command.add.tui.open", perf.WithEventAttributes(
			attribute.Int("initial_state", int(initialState)),
			attribute.String("platform", string(platformValue)),
			attribute.String("project_id", projectID),
		))
	}

	var attempt int
	tuiCtx, tuiSpan := perf.StartSpan(ctx, "tui.add.session",
		perf.WithAttributes(
			attribute.String("platform", string(platformValue)),
			attribute.String("project_id", projectID),
			attribute.Int("initial_state", int(initialState)),
		),
	)
	model := newAddTUIModel(tuiCtx, tuiSpan, initialState, platformValue, projectID, cfg, func(platformValue models.Platform, projectID string) tea.Cmd {
		return func() tea.Msg {
			attempt++
			attemptCtx, attemptSpan := perf.StartSpan(tuiCtx, "app.command.add.resolve.attempt",
				perf.WithAttributes(
					attribute.Int("attempt", attempt),
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
	})

	if deps.runTea == nil {
		return platform.RemoteMod{}, platformValue, projectID, errors.New("missing add dependencies: runTea")
	}

	result, err := deps.runTea(model, tui.ProgramOptions(in, out)...)
	if err != nil {
		tuiSpan.SetAttributes(attribute.Bool("success", false))
		tuiSpan.End()
		return platform.RemoteMod{}, platformValue, projectID, err
	}
	tuiSpan.SetAttributes(attribute.Bool("success", true))
	tuiSpan.End()

	typed, ok := result.(addTUIModel)
	if !ok {
		return platform.RemoteMod{}, platformValue, projectID, errors.New("unexpected add TUI result model")
	}

	return typed.result()
}
