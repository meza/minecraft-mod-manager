package test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/tui"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/time/rate"
)

type testOptions struct {
	ConfigPath  string
	GameVersion string
	Quiet       bool
	Debug       bool
}

type testDeps struct {
	fs             afero.Fs
	logger         *logger.Logger
	clients        platform.Clients
	fetchMod       fetcher
	latestVersion  latestVersionFetcher
	isValidVersion versionValidator
	telemetry      func(telemetry.CommandTelemetry)
}

type fetcher func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error)

type latestVersionFetcher func(context.Context, httpClient.Doer) (string, error)

type versionValidator func(context.Context, string, httpClient.Doer) bool

var errInvalidVersion = errors.New("invalid minecraft version")
var errLatestVersionRequired = errors.New("could not determine latest version: please provide an explicit version")

// exitCodeError is a private error type that carries a specific exit code.
// Used for the "same version" case (exit code 2) where we need a non-standard exit code
// but the condition is not a failure.
type exitCodeError struct {
	code int
}

func (e *exitCodeError) Error() string {
	return fmt.Sprintf("exit code %d", e.code)
}

func (e *exitCodeError) ExitCode() int {
	return e.code
}

// errSameVersion signals that the target version matches the current config version.
// This is a special case that returns exit code 2 per spec.
var errSameVersion = &exitCodeError{code: 2}
var errUnsupportedMods = &exitCodeError{code: 1}

type testRunner func(context.Context, *cobra.Command, testOptions, testDeps) (int, error)

func Command() *cobra.Command {
	return commandWithRunner(runTest)
}

func commandWithRunner(runner testRunner) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "test [game_version]",
		Aliases: []string{"t"},
		Short:   i18n.T("cmd.test.short"),
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			ctx, span := perf.StartSpan(cmd.Context(), "app.command.test")

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

			gameVersion := "latest"
			if len(args) > 0 {
				gameVersion = args[0]
			}

			log := logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), quiet, debug)
			limiter := rate.NewLimiter(rate.Inf, 0)

			deps := testDeps{
				fs:             afero.NewOsFs(),
				logger:         log,
				clients:        platform.DefaultClients(limiter),
				fetchMod:       platform.FetchMod,
				latestVersion:  minecraft.GetLatestVersion,
				isValidVersion: minecraft.IsValidVersion,
				telemetry:      telemetry.RecordCommand,
			}

			exitCode, err := runner(ctx, cmd, testOptions{
				ConfigPath:  configPath,
				GameVersion: gameVersion,
				Quiet:       quiet,
				Debug:       debug,
			}, deps)

			span.SetAttributes(attribute.Bool("success", err == nil))
			span.End()

			if err != nil {
				cmd.SilenceUsage = true
				// Suppress cobra's error printing for errors that we have already
				// logged or that should not produce additional output.
				var exitErr *exitCodeError
				if errors.As(err, &exitErr) {
					// Exit code errors (like same-version) are already logged; suppress cobra output
					cmd.SilenceErrors = true
				} else if errors.Is(err, errLatestVersionRequired) || errors.Is(err, errInvalidVersion) {
					// These errors are already logged via deps.logger.Error(); suppress cobra output
					cmd.SilenceErrors = true
				}
			}

			payload := telemetry.CommandTelemetry{
				Command:     "test",
				Success:     err == nil && exitCode == 0,
				Error:       err,
				ExitCode:    exitCode,
				Interactive: false,
				Extra: map[string]interface{}{
					"targetVersion": gameVersion,
					"exitCode":      exitCode,
				},
			}
			deps.telemetry(payload)

			return err
		},
	}

	return cmd
}

type modCheckCandidate struct {
	ConfigIndex int
	Mod         models.Mod
}

type modCheckOutcome struct {
	ConfigIndex int
	Mod         models.Mod
	Supported   bool
	LogEvents   []logEvent
}

type logEventKind int

const (
	logEventKindError logEventKind = iota
	logEventKindDebug
)

type logEvent struct {
	Kind      logEventKind
	Message   string
	ForceShow bool
}

func runTest(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (int, error) {
	meta := config.NewMetadata(opts.ConfigPath)

	cfg, err := config.ReadConfig(ctx, deps.fs, meta)
	if err != nil {
		return 0, err
	}

	targetVersion := opts.GameVersion

	if strings.EqualFold(targetVersion, "latest") {
		latest, err := deps.latestVersion(ctx, deps.clients.Modrinth)
		if err != nil {
			// Per ADR 0006: when manifest fails, we cannot determine "latest" in non-interactive mode.
			// The user must provide an explicit version. Interactive prompting is for TUI only.
			deps.logger.Error(i18n.T("cmd.test.error.latest_unavailable", i18n.Tvars{}))
			return 0, errLatestVersionRequired
		}
		targetVersion = latest
	}

	// Per ADR 0006: if version validation fails (manifest unavailable), assume valid.
	// The isValidVersion function should already handle this per ADR 0006.
	if !deps.isValidVersion(ctx, targetVersion, deps.clients.Modrinth) {
		deps.logger.Error(i18n.T("cmd.test.error.invalid_version", i18n.Tvars{
			Data: &i18n.TData{"version": targetVersion},
		}))
		return 0, errInvalidVersion
	}

	if targetVersion == cfg.GameVersion {
		deps.logger.Log(i18n.T("cmd.test.same_version", i18n.Tvars{
			Data: &i18n.TData{"version": targetVersion},
		}), true)
		// Return exit code 2 via errSameVersion so it propagates through main.go
		return 2, errSameVersion
	}

	if len(cfg.Mods) == 0 {
		deps.logger.Log(i18n.T("cmd.test.success", i18n.Tvars{
			Data: &i18n.TData{"version": targetVersion},
		}), false)
		return 0, nil
	}

	colorize := tui.IsTerminalWriter(cmd.OutOrStdout())

	candidates := make([]modCheckCandidate, 0, len(cfg.Mods))
	for i := range cfg.Mods {
		candidates = append(candidates, modCheckCandidate{
			ConfigIndex: i,
			Mod:         cfg.Mods[i],
		})
	}

	results := make(chan modCheckOutcome, len(candidates))
	var waitGroup sync.WaitGroup

	for _, candidate := range candidates {
		candidate := candidate
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			results <- checkMod(ctx, cfg, candidate, targetVersion, deps, colorize)
		}()
	}

	waitGroup.Wait()
	close(results)

	outcomes := make([]modCheckOutcome, len(candidates))
	for outcome := range results {
		if outcome.ConfigIndex >= 0 && outcome.ConfigIndex < len(outcomes) {
			outcomes[outcome.ConfigIndex] = outcome
		}
	}

	unsupportedMods := make([]modCheckOutcome, 0)
	for _, outcome := range outcomes {
		for _, event := range outcome.LogEvents {
			switch event.Kind {
			case logEventKindError:
				deps.logger.Error(event.Message)
			case logEventKindDebug:
				deps.logger.Debug(event.Message)
			}
		}

		if !outcome.Supported {
			unsupportedMods = append(unsupportedMods, outcome)
		}
	}

	if len(unsupportedMods) > 0 {
		deps.logger.Log(i18n.T("cmd.test.missing_support_header", i18n.Tvars{
			Data: &i18n.TData{"version": targetVersion},
		}), true)

		for _, unsupported := range unsupportedMods {
			modEntry := formatMissingModEntry(unsupported.Mod, colorize)
			deps.logger.Log(modEntry, true)
		}

		deps.logger.Log(i18n.T("cmd.test.cannot_upgrade", i18n.Tvars{
			Data: &i18n.TData{"version": targetVersion},
		}), true)

		return 1, errUnsupportedMods
	}

	deps.logger.Log(i18n.T("cmd.test.success", i18n.Tvars{
		Data: &i18n.TData{"version": targetVersion},
	}), false)
	return 0, nil
}

func checkMod(
	ctx context.Context,
	cfg models.ModsJson,
	candidate modCheckCandidate,
	targetVersion string,
	deps testDeps,
	colorize bool,
) modCheckOutcome {
	mod := candidate.Mod

	outcome := modCheckOutcome{
		ConfigIndex: candidate.ConfigIndex,
		Mod:         mod,
		Supported:   true,
	}

	outcome.LogEvents = append(outcome.LogEvents, logEvent{
		Kind: logEventKindDebug,
		Message: i18n.T("cmd.test.debug.checking", i18n.Tvars{
			Data: &i18n.TData{
				"name":     mod.Name,
				"platform": mod.Type,
				"version":  targetVersion,
			},
		}),
	})

	fetchOpts := platform.FetchOptions{
		AllowedReleaseTypes: effectiveAllowedReleaseTypes(mod, cfg),
		GameVersion:         targetVersion,
		Loader:              cfg.Loader,
		AllowFallback:       mod.AllowVersionFallback != nil && *mod.AllowVersionFallback,
	}

	if mod.Version != nil && strings.TrimSpace(*mod.Version) != "" {
		fetchOpts.FixedVersion = *mod.Version
	}

	_, fetchErr := deps.fetchMod(ctx, mod.Type, mod.ID, fetchOpts, deps.clients)
	if fetchErr != nil {
		outcome.Supported = false

		var notFound *platform.ModNotFoundError
		var noFile *platform.NoCompatibleFileError

		if errors.As(fetchErr, &notFound) || errors.As(fetchErr, &noFile) {
			// No log event needed here - unsupported mods are reported in the summary
		} else {
			outcome.LogEvents = append(outcome.LogEvents, logEvent{
				Kind:    logEventKindError,
				Message: fetchErr.Error(),
			})
		}
	}

	return outcome
}

func effectiveAllowedReleaseTypes(mod models.Mod, cfg models.ModsJson) []models.ReleaseType {
	if len(mod.AllowedReleaseTypes) > 0 {
		return mod.AllowedReleaseTypes
	}
	return cfg.DefaultAllowedReleaseTypes
}

func formatMissingModEntry(mod models.Mod, colorize bool) string {
	icon := tui.ErrorIcon(colorize)
	name := mod.Name
	id := mod.ID

	if colorize {
		// Use PlaceholderStyle color without padding to allow explicit spacing control
		grayStyle := tui.PlaceholderStyle.UnsetPaddingLeft()
		idPart := grayStyle.Render(fmt.Sprintf("(%s)", id))
		return fmt.Sprintf("%s %s %s", icon, name, idPart)
	}

	return fmt.Sprintf("%s %s (%s)", icon, name, id)
}
