package add

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/time/rate"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/tui"
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
	telemetry       func(telemetry.CommandTelemetry)
	runTea          func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error)
}

type fetcher func(models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error)

type downloader func(url string, filepath string, client httpClient.Doer, program httpClient.Sender, filesystem ...afero.Fs) error

var errAborted = errors.New("add aborted")

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <platform> <id>",
		Short: i18n.T("cmd.add.short"),
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			details := perf.PerformanceDetails{
				"platform": args[0],
				"id":       args[1],
			}
			region := perf.StartRegionWithDetails("app.command.add", &details)
			defer func() {
				details["success"] = err == nil
				region.End()
			}()

			configPath, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			quiet, err := cmd.Flags().GetBool("quiet")
			if err != nil {
				return err
			}
			debug, err := cmd.Flags().GetBool("debug")
			if err != nil {
				return err
			}
			version, err := cmd.Flags().GetString("version")
			if err != nil {
				return err
			}
			allowFallback, err := cmd.Flags().GetBool("allow-version-fallback")
			if err != nil {
				return err
			}

			log := logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), quiet, debug)

			limiter := rate.NewLimiter(rate.Inf, 0)

			err = runAdd(cmd, addOptions{
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
				telemetry:       telemetry.CaptureCommand,
				runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
					return tea.NewProgram(model, options...).Run()
				},
			})
			return err
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

func runAdd(cmd *cobra.Command, opts addOptions, deps addDeps) error {
	meta := config.NewMetadata(opts.ConfigPath)

	prepareDetails := perf.PerformanceDetails{
		"config_path": opts.ConfigPath,
	}
	prepareRegion := perf.StartRegionWithDetails("app.command.add.stage.prepare", &prepareDetails)
	cfg, lock, err := ensureConfigAndLock(deps.fs, meta, opts.Quiet, deps.minecraftClient)
	prepareDetails["success"] = err == nil
	prepareRegion.End()
	if err != nil {
		return err
	}

	useTUI := tui.ShouldUseTUI(opts.Quiet, cmd.InOrStdin(), cmd.OutOrStdout())

	platformValue := normalizePlatform(opts.Platform)
	projectID := opts.ProjectID

	if modExists(cfg, platformValue, projectID) {
		perf.Mark("app.command.add.outcome.already_exists", &perf.PerformanceDetails{
			"platform":   platformValue,
			"project_id": projectID,
		})
		deps.logger.Debug(i18n.T("cmd.add.debug.already_exists", i18n.Tvars{
			Data: &i18n.TData{
				"id":       projectID,
				"platform": platformValue,
			},
		}))
		deps.telemetry(telemetry.CommandTelemetry{
			Command: "add",
			Success: true,
			Arguments: map[string]interface{}{
				"platform": platformValue,
				"id":       projectID,
				"version":  opts.Version,
				"fallback": opts.AllowVersionFallback,
			},
			Extra: map[string]interface{}{
				"flag": "already-exists",
			},
		})
		return nil
	}

	start := time.Now()

	resolveDetails := perf.PerformanceDetails{
		"platform":   platformValue,
		"project_id": projectID,
		"use_tui":    useTUI,
		"quiet":      opts.Quiet,
	}
	resolveRegion := perf.StartRegionWithDetails("app.command.add.stage.resolve", &resolveDetails)
	remoteMod, resolvedPlatform, resolvedID, fetchErr := resolveRemoteMod(cfg, opts, platformValue, projectID, deps, useTUI, cmd.InOrStdin(), cmd.OutOrStdout())
	resolveDetails["success"] = fetchErr == nil
	resolveDetails["resolved_platform"] = resolvedPlatform
	resolveDetails["resolved_project_id"] = resolvedID
	resolveRegion.End()
	if fetchErr != nil {
		deps.telemetry(telemetry.CommandTelemetry{
			Command: "add",
			Success: false,
			Config:  &cfg,
			Error:   fetchErr,
			Arguments: map[string]interface{}{
				"platform": platformValue,
				"id":       projectID,
				"version":  opts.Version,
				"fallback": opts.AllowVersionFallback,
			},
		})
		if errors.Is(fetchErr, errAborted) {
			return nil
		}
		return fetchErr
	}

	downloadDetails := perf.PerformanceDetails{
		"url":        remoteMod.DownloadURL,
		"platform":   resolvedPlatform,
		"project_id": resolvedID,
		"file_name":  remoteMod.FileName,
	}
	downloadRegion := perf.StartRegionWithDetails("app.command.add.stage.download", &downloadDetails)

	if err := deps.fs.MkdirAll(meta.ModsFolderPath(cfg), 0755); err != nil {
		downloadDetails["success"] = false
		downloadRegion.End()
		return err
	}

	destination := filepath.Join(meta.ModsFolderPath(cfg), remoteMod.FileName)
	if err := deps.downloader(remoteMod.DownloadURL, destination, downloadClient(deps.clients), &noopSender{}, deps.fs); err != nil {
		downloadDetails["success"] = false
		downloadRegion.End()
		return err
	}
	downloadDetails["success"] = true
	downloadDetails["path"] = destination
	downloadRegion.End()

	cfg.Mods = append(cfg.Mods, models.Mod{
		Type:                 resolvedPlatform,
		ID:                   resolvedID,
		Name:                 remoteMod.Name,
		AllowVersionFallback: optionalBool(opts.AllowVersionFallback),
		Version:              optionalString(opts.Version),
	})

	lock = append(lock, models.ModInstall{
		Type:        resolvedPlatform,
		Id:          resolvedID,
		Name:        remoteMod.Name,
		FileName:    remoteMod.FileName,
		ReleasedOn:  remoteMod.ReleaseDate,
		Hash:        remoteMod.Hash,
		DownloadUrl: remoteMod.DownloadURL,
	})

	persistDetails := perf.PerformanceDetails{
		"config_path": opts.ConfigPath,
		"platform":    resolvedPlatform,
		"project_id":  resolvedID,
	}
	persistRegion := perf.StartRegionWithDetails("app.command.add.stage.persist", &persistDetails)
	if err := config.WriteConfig(deps.fs, meta, cfg); err != nil {
		persistDetails["success"] = false
		persistRegion.End()
		return err
	}
	if err := config.WriteLock(deps.fs, meta, lock); err != nil {
		persistDetails["success"] = false
		persistRegion.End()
		return err
	}
	persistDetails["success"] = true
	persistRegion.End()

	deps.logger.Log(i18n.T("cmd.add.success", i18n.Tvars{
		Data: &i18n.TData{
			"name":     remoteMod.Name,
			"id":       resolvedID,
			"platform": resolvedPlatform,
		},
	}), true)

	deps.telemetry(telemetry.CommandTelemetry{
		Command:  "add",
		Success:  true,
		Duration: time.Since(start),
		Arguments: map[string]interface{}{
			"platform": resolvedPlatform,
			"id":       resolvedID,
			"version":  opts.Version,
			"fallback": opts.AllowVersionFallback,
		},
	})

	return nil
}

func ensureConfigAndLock(fs afero.Fs, meta config.Metadata, quiet bool, minecraftClient httpClient.Doer) (models.ModsJson, []models.ModInstall, error) {
	cfg, err := config.ReadConfig(fs, meta)
	if err != nil {
		var notFound *config.ConfigFileNotFoundException
		if errors.As(err, &notFound) {
			if quiet {
				return models.ModsJson{}, nil, err
			}
			cfg, err = config.InitConfig(fs, meta, minecraftClient)
			if err != nil {
				return models.ModsJson{}, nil, err
			}
		} else {
			return models.ModsJson{}, nil, err
		}
	}

	lock, err := config.EnsureLock(fs, meta)
	if err != nil {
		return models.ModsJson{}, nil, err
	}

	return cfg, lock, nil
}

func resolveRemoteMod(cfg models.ModsJson, opts addOptions, platformValue models.Platform, projectID string, deps addDeps, useTUI bool, in io.Reader, out io.Writer) (platform.RemoteMod, models.Platform, string, error) {
	deps.logger.Debug(fmt.Sprintf("fetching %s/%s (loader=%s, gameVersion=%s, fallback=%t, fixedVersion=%s)", platformValue, projectID, cfg.Loader, cfg.GameVersion, opts.AllowVersionFallback, opts.Version))

	attemptDetails := perf.PerformanceDetails{
		"attempt":    0,
		"source":     "cli",
		"platform":   platformValue,
		"project_id": projectID,
		"use_tui":    useTUI,
		"quiet":      opts.Quiet,
	}
	attemptRegion := perf.StartRegionWithDetails("app.command.add.resolve.attempt", &attemptDetails)
	remote, err := deps.fetchMod(platformValue, projectID, platform.FetchOptions{
		AllowedReleaseTypes: cfg.DefaultAllowedReleaseTypes,
		GameVersion:         cfg.GameVersion,
		Loader:              cfg.Loader,
		AllowFallback:       opts.AllowVersionFallback,
		FixedVersion:        opts.Version,
	}, deps.clients)
	attemptDetails["success"] = err == nil
	if err != nil {
		attemptDetails["error_type"] = fmt.Sprintf("%T", err)
	}
	attemptRegion.End()
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
		return resolveRemoteModWithTUI(addTUIStateUnknownPlatformSelect, cfg, opts, platformValue, projectID, deps, in, out)
	case *platform.ModNotFoundError:
		if opts.Quiet || !useTUI {
			deps.logger.Error(errorMessageForModNotFound(projectID, platformValue))
			return platform.RemoteMod{}, platformValue, projectID, err
		}
		return resolveRemoteModWithTUI(addTUIStateModNotFoundConfirm, cfg, opts, platformValue, projectID, deps, in, out)
	case *platform.NoCompatibleFileError:
		if opts.Quiet || !useTUI {
			deps.logger.Error(errorMessageForNoFile(projectID, platformValue))
			return platform.RemoteMod{}, platformValue, projectID, err
		}
		return resolveRemoteModWithTUI(addTUIStateNoFileConfirm, cfg, opts, platformValue, projectID, deps, in, out)
	default:
		return platform.RemoteMod{}, platformValue, projectID, err
	}
}

func modExists(cfg models.ModsJson, platform models.Platform, projectID string) bool {
	for _, mod := range cfg.Mods {
		if mod.ID == projectID && mod.Type == platform {
			return true
		}
	}
	return false
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

func optionalBool(value bool) *bool {
	if !value {
		return nil
	}
	return &value
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

type noopSender struct{}

func (n *noopSender) Send(msg tea.Msg) {}

func downloadClient(clients platform.Clients) httpClient.Doer {
	if clients.Curseforge != nil {
		return clients.Curseforge
	}
	return clients.Modrinth
}

func resolveRemoteModWithTUI(initialState addTUIState, cfg models.ModsJson, opts addOptions, platformValue models.Platform, projectID string, deps addDeps, in io.Reader, out io.Writer) (platform.RemoteMod, models.Platform, string, error) {
	perf.Mark("app.command.add.tui.open", &perf.PerformanceDetails{
		"initial_state": int(initialState),
		"platform":      platformValue,
		"project_id":    projectID,
	})

	var attempt int
	model := newAddTUIModel(initialState, platformValue, projectID, cfg, func(platformValue models.Platform, projectID string) tea.Cmd {
		return func() tea.Msg {
			attempt++
			attemptDetails := perf.PerformanceDetails{
				"attempt":    attempt,
				"source":     "tui",
				"platform":   platformValue,
				"project_id": projectID,
				"quiet":      opts.Quiet,
			}
			attemptRegion := perf.StartRegionWithDetails("app.command.add.resolve.attempt", &attemptDetails)
			remote, err := deps.fetchMod(platformValue, projectID, platform.FetchOptions{
				AllowedReleaseTypes: cfg.DefaultAllowedReleaseTypes,
				GameVersion:         cfg.GameVersion,
				Loader:              cfg.Loader,
				AllowFallback:       opts.AllowVersionFallback,
				FixedVersion:        opts.Version,
			}, deps.clients)
			attemptDetails["success"] = err == nil
			if err != nil {
				attemptDetails["error_type"] = fmt.Sprintf("%T", err)
			}
			attemptRegion.End()
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
		return platform.RemoteMod{}, platformValue, projectID, err
	}

	typed, ok := result.(addTUIModel)
	if !ok {
		return platform.RemoteMod{}, platformValue, projectID, errors.New("unexpected add TUI result model")
	}

	return typed.result()
}
