package test

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
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
	"golang.org/x/sync/errgroup"
)

const defaultTestMaxConcurrency = 4

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

type latestVersionFetcher func(context.Context, httpclient.Doer) (string, error)

type versionValidator func(context.Context, string, httpclient.Doer) (bool, error)

var errInvalidVersion = errors.New("invalid minecraft version")
var errLatestVersionRequired = errors.New("could not determine latest version: please provide an explicit version")
var errVersionValidationUnavailable = errors.New("could not verify minecraft version")

// exitCodeError is a private error type that carries a specific exit code.
// Used for the "same version" case (exit code 2) where we need a non-standard exit code
// but the condition is not a failure.
type exitCodeError struct {
	code int
}

func (exitError *exitCodeError) Error() string {
	return fmt.Sprintf("exit code %d", exitError.code)
}

func (exitError *exitCodeError) ExitCode() int {
	return exitError.code
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTestCommand(cmd, args, runner)
		},
	}

	return cmd
}

func runTestCommand(cmd *cobra.Command, args []string, runner testRunner) error {
	ctx, span := perf.StartSpan(cmd.Context(), "app.command.test")

	opts, err := testOptionsFromFlags(cmd, args)
	if err != nil {
		span.SetAttributes(attribute.Bool("success", false))
		span.End()
		return err
	}

	log := logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), opts.Quiet, opts.Debug)
	deps := defaultTestDeps(log)

	exitCode, err := runner(ctx, cmd, opts, deps)
	span.SetAttributes(attribute.Bool("success", err == nil))
	span.End()

	handleTestCommandError(cmd, err)
	recordTestTelemetry(deps.telemetry, opts.GameVersion, exitCode, err)
	return err
}

func testOptionsFromFlags(cmd *cobra.Command, args []string) (testOptions, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return testOptions{}, err
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return testOptions{}, err
	}
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return testOptions{}, err
	}

	return testOptions{
		ConfigPath:  configPath,
		GameVersion: resolveGameVersion(args),
		Quiet:       quiet,
		Debug:       debug,
	}, nil
}

func resolveGameVersion(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return "latest"
}

func defaultTestDeps(log *logger.Logger) testDeps {
	limiter := httpclient.DefaultLimiter()
	return testDeps{
		fs:             afero.NewOsFs(),
		logger:         log,
		clients:        platform.DefaultClients(limiter),
		fetchMod:       platform.FetchMod,
		latestVersion:  minecraft.GetLatestVersion,
		isValidVersion: minecraft.IsValidVersion,
		telemetry:      telemetry.RecordCommand,
	}
}

func handleTestCommandError(cmd *cobra.Command, err error) {
	if err == nil {
		return
	}
	cmd.SilenceUsage = true
	// Suppress cobra's error printing for errors that we have already
	// logged or that should not produce additional output.
	var exitErr *exitCodeError
	if errors.As(err, &exitErr) {
		// Exit code errors (like same-version) are already logged; suppress cobra output
		cmd.SilenceErrors = true
	} else if errors.Is(err, errLatestVersionRequired) || errors.Is(err, errInvalidVersion) || errors.Is(err, errVersionValidationUnavailable) {
		// These errors are already logged via deps.logger.Error(); suppress cobra output
		cmd.SilenceErrors = true
	}
}

func recordTestTelemetry(telemetryRecorder func(telemetry.CommandTelemetry), gameVersion string, exitCode int, err error) {
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
	telemetryRecorder(payload)
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

	targetVersion, exitCode, err := resolveTargetVersion(ctx, cfg, opts, deps)
	if err != nil {
		return exitCode, err
	}

	if len(cfg.Mods) == 0 {
		deps.logger.Log(i18n.T("cmd.test.success", i18n.Tvars{
			Data: &i18n.TData{"version": targetVersion},
		}), logger.LogQuiet)
		return 0, nil
	}

	colorize := tui.IsTerminalWriter(cmd.OutOrStdout())
	colorMode := tui.ColorDisabled
	if colorize {
		colorMode = tui.ColorEnabled
	}

	outcomes, err := collectOutcomes(ctx, cfg, targetVersion, deps)
	if err != nil {
		return 0, err
	}
	return evaluateTestOutcomes(targetVersion, outcomes, deps, colorMode)
}

func checkMod(
	ctx context.Context,
	cfg models.ModsJSON,
	candidate modCheckCandidate,
	targetVersion string,
	deps testDeps,
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
		outcome.LogEvents = append(outcome.LogEvents, fetchFailureUserEvent(fetchErr, mod))

		if debugEvent, ok := fetchFailureDebugEvent(fetchErr, mod, cfg, targetVersion, fetchOpts); ok {
			outcome.LogEvents = append(outcome.LogEvents, debugEvent)
		}
	}

	return outcome
}

func effectiveAllowedReleaseTypes(mod models.Mod, cfg models.ModsJSON) []models.ReleaseType {
	if len(mod.AllowedReleaseTypes) > 0 {
		return mod.AllowedReleaseTypes
	}
	return cfg.DefaultAllowedReleaseTypes
}

func fetchFailureDetails(mod models.Mod, cfg models.ModsJSON, targetVersion string, opts platform.FetchOptions) string {
	fixedVersion := strings.TrimSpace(opts.FixedVersion)
	if fixedVersion == "" {
		fixedVersion = "none"
	}
	return fmt.Sprintf("id=%s version=%s loader=%s releases=%s fixedVersion=%s allowFallback=%t",
		mod.ID,
		targetVersion,
		cfg.Loader,
		formatReleaseTypes(opts.AllowedReleaseTypes),
		fixedVersion,
		opts.AllowFallback,
	)
}

func formatReleaseTypes(releaseTypes []models.ReleaseType) string {
	if len(releaseTypes) == 0 {
		return "none"
	}
	entries := make([]string, 0, len(releaseTypes))
	for _, releaseType := range releaseTypes {
		entries = append(entries, string(releaseType))
	}
	return strings.Join(entries, ",")
}

func fetchFailureUserEvent(fetchErr error, mod models.Mod) logEvent {
	var notFound *platform.ModNotFoundError
	if errors.As(fetchErr, &notFound) {
		return logEvent{
			Kind: logEventKindError,
			Message: i18n.T("cmd.test.error.mod_not_found", i18n.Tvars{
				Data: &i18n.TData{
					"name":     mod.Name,
					"id":       mod.ID,
					"platform": mod.Type,
				},
			}),
		}
	}

	var noFile *platform.NoCompatibleFileError
	if errors.As(fetchErr, &noFile) {
		return logEvent{
			Kind: logEventKindError,
			Message: i18n.T("cmd.test.error.no_file", i18n.Tvars{
				Data: &i18n.TData{
					"name":     mod.Name,
					"id":       mod.ID,
					"platform": mod.Type,
				},
			}),
		}
	}

	summary, ok := clierrors.SummarizePlatformError(fetchErr, mod.Type)
	reason := i18n.T("cmd.platform.error.reason.unknown")
	if ok && strings.TrimSpace(summary.Reason) != "" {
		reason = summary.Reason
	}

	return logEvent{
		Kind: logEventKindError,
		Message: i18n.T("cmd.test.error.platform", i18n.Tvars{
			Data: &i18n.TData{
				"name":     mod.Name,
				"platform": mod.Type,
				"reason":   reason,
			},
		}),
	}
}

func fetchFailureDebugEvent(fetchErr error, mod models.Mod, cfg models.ModsJSON, targetVersion string, opts platform.FetchOptions) (logEvent, bool) {
	details := fetchFailureDetails(mod, cfg, targetVersion, opts)
	debugError := fetchErr.Error()
	if summary, ok := clierrors.SummarizePlatformError(fetchErr, mod.Type); ok && strings.TrimSpace(summary.DebugDetails) != "" {
		debugError = summary.DebugDetails
	}
	return logEvent{
		Kind: logEventKindDebug,
		Message: i18n.T("cmd.test.debug.platform_error", i18n.Tvars{
			Data: &i18n.TData{
				"name":     mod.Name,
				"platform": mod.Type,
				"error":    debugError,
				"details":  details,
			},
		}),
	}, true
}

func formatMissingModEntry(mod models.Mod, colorMode tui.ColorMode) string {
	icon := tui.ErrorIcon(colorMode)
	name := mod.Name
	id := mod.ID

	// Use PlaceholderStyle color without padding to allow explicit spacing control
	grayStyle := tui.PlaceholderStyle.UnsetPaddingLeft()
	idPart := tui.RenderIfColorEnabled(colorMode, grayStyle, fmt.Sprintf("(%s)", id))
	return fmt.Sprintf("%s %s %s", icon, name, idPart)
}

func resolveTargetVersion(ctx context.Context, cfg models.ModsJSON, opts testOptions, deps testDeps) (string, int, error) {
	targetVersion := opts.GameVersion

	if strings.EqualFold(targetVersion, "latest") {
		latest, err := deps.latestVersion(ctx, deps.clients.Modrinth)
		if err != nil {
			// Per ADR 0006: when manifest fails, we cannot determine "latest" in non-interactive mode.
			// The user must provide an explicit version. Interactive prompting is for TUI only.
			deps.logger.Error(i18n.T("cmd.test.error.latest_unavailable", i18n.Tvars{}))
			return "", 0, errLatestVersionRequired
		}
		targetVersion = latest
	}

	valid, validationErr := deps.isValidVersion(ctx, targetVersion, deps.clients.Modrinth)
	if validationErr != nil {
		deps.logger.Error(i18n.T("cmd.test.error.version_unavailable", i18n.Tvars{}))
		return "", 0, errVersionValidationUnavailable
	}
	if !valid {
		deps.logger.Error(i18n.T("cmd.test.error.invalid_version", i18n.Tvars{
			Data: &i18n.TData{"version": targetVersion},
		}))
		return "", 0, errInvalidVersion
	}

	if targetVersion == cfg.GameVersion {
		deps.logger.Log(i18n.T("cmd.test.same_version", i18n.Tvars{
			Data: &i18n.TData{"version": targetVersion},
		}), logger.LogForce)
		// Return exit code 2 via errSameVersion so it propagates through main.go
		return targetVersion, 2, errSameVersion
	}

	return targetVersion, 0, nil
}

func collectOutcomes(ctx context.Context, cfg models.ModsJSON, targetVersion string, deps testDeps) ([]modCheckOutcome, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	candidates := make([]modCheckCandidate, 0, len(cfg.Mods))
	for i := range cfg.Mods {
		candidates = append(candidates, modCheckCandidate{
			ConfigIndex: i,
			Mod:         cfg.Mods[i],
		})
	}

	outcomes := make([]modCheckOutcome, len(candidates))
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(defaultTestMaxConcurrency)

	for _, candidate := range candidates {
		candidate := candidate
		group.Go(func() error {
			outcome := checkMod(groupCtx, cfg, candidate, targetVersion, deps)
			if outcome.ConfigIndex >= 0 && outcome.ConfigIndex < len(outcomes) {
				outcomes[outcome.ConfigIndex] = outcome
			}
			if err := groupCtx.Err(); err != nil {
				return err
			}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, err
	}

	return outcomes, nil
}

func logOutcomes(outcomes []modCheckOutcome, deps testDeps) []modCheckOutcome {
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
	return unsupportedMods
}

func evaluateTestOutcomes(targetVersion string, outcomes []modCheckOutcome, deps testDeps, colorMode tui.ColorMode) (int, error) {
	unsupportedMods := logOutcomes(outcomes, deps)
	if len(unsupportedMods) > 0 {
		deps.logger.Log(i18n.T("cmd.test.missing_support_header", i18n.Tvars{
			Data: &i18n.TData{"version": targetVersion},
		}), logger.LogForce)

		for _, unsupported := range unsupportedMods {
			modEntry := formatMissingModEntry(unsupported.Mod, colorMode)
			deps.logger.Log(modEntry, logger.LogForce)
		}

		deps.logger.Log(i18n.T("cmd.test.cannot_upgrade", i18n.Tvars{
			Data: &i18n.TData{"version": targetVersion},
		}), logger.LogForce)

		return 1, errUnsupportedMods
	}

	deps.logger.Log(i18n.T("cmd.test.success", i18n.Tvars{
		Data: &i18n.TData{"version": targetVersion},
	}), logger.LogQuiet)
	return 0, nil
}
