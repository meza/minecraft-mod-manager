package scan

import (
	"context"
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	curseforgeFingerprint "github.com/meza/curseforge-fingerprint-go"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"github.com/meza/minecraft-mod-manager/internal/modsetup"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
)

var runInteractiveInit = initCmd.RunInteractiveInit

type scanOptions struct {
	ConfigPath string
	Unattended bool
	Quiet      bool
	Debug      bool
	Prefer     string
	Add        bool
}

type scanDeps struct {
	fs              afero.Fs
	clients         platform.Clients
	minecraftClient httpclient.Doer
	logger          *logger.Logger
	telemetry       func(telemetry.CommandTelemetry)
	runTea          func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error)
	runInit         initRunner

	curseforgeFingerprint      func(string) uint32
	modrinthVersionForSha      func(context.Context, string, httpclient.Doer) (*modrinth.Version, error)
	modrinthProjectTitle       func(context.Context, string, httpclient.Doer) (string, error)
	curseforgeFingerprintMatch func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error)
	curseforgeProjectName      func(context.Context, string, httpclient.Doer) (string, error)
}

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: i18n.T("cmd.scan.short", nil),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runScanCommand(cmd)
		},
	}

	cmd.Flags().StringP("prefer", "p", string(models.MODRINTH), i18n.T("cmd.scan.flag.prefer", nil))
	cmd.Flags().BoolP("add", "a", false, i18n.T("cmd.scan.flag.add", nil))

	return cmd
}

func runScanCommand(cmd *cobra.Command) error {
	ctx, span := perf.StartSpan(cmd.Context(), "app.command.scan")
	defer span.End()

	opts, err := scanOptionsFromFlags(cmd)
	if err != nil {
		span.SetAttributes(attribute.Bool("success", false))
		return err
	}

	deps := defaultScanDeps(cmd, opts)
	payload, err := runScan(ctx, cmd, opts, deps)
	span.SetAttributes(attribute.Bool("success", err == nil))

	if deps.telemetry != nil {
		deps.telemetry(payload)
	}
	applyScanCommandErrorPolicy(cmd, err)
	return err
}

func applyScanCommandErrorPolicy(cmd *cobra.Command, err error) {
	if err == nil {
		return
	}
	if clierrors.IsHandled(err) {
		cmd.SilenceErrors = true
	}
	cmd.SilenceUsage = true
}

func scanOptionsFromFlags(cmd *cobra.Command) (scanOptions, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return scanOptions{}, err
	}
	unattended, err := cmd.Flags().GetBool("unattended")
	if err != nil {
		return scanOptions{}, err
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return scanOptions{}, err
	}
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return scanOptions{}, err
	}
	prefer, err := cmd.Flags().GetString("prefer")
	if err != nil {
		return scanOptions{}, err
	}
	add, err := cmd.Flags().GetBool("add")
	if err != nil {
		return scanOptions{}, err
	}

	return scanOptions{
		ConfigPath: configPath,
		Unattended: unattended,
		Quiet:      quiet,
		Debug:      debug,
		Prefer:     prefer,
		Add:        add,
	}, nil
}

func defaultScanDeps(cmd *cobra.Command, opts scanOptions) scanDeps {
	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
		Quiet: opts.Quiet,
		Debug: opts.Debug,
	})
	return scanDeps{
		fs:              common.FS,
		clients:         common.Clients,
		minecraftClient: common.MinecraftClient,
		logger:          common.Logger,
		telemetry:       telemetry.RecordCommand,
		runTea:          runTeaProgram,
		runInit: func(ctx context.Context, cmd *cobra.Command, request initRequest) error {
			return runInteractiveInit(ctx, cmd, initCmd.InteractiveInitDeps{
				FS:              common.FS,
				Logger:          common.Logger,
				MinecraftClient: common.MinecraftClient,
				RunTea:          runTeaProgram,
			}, initCmd.InteractiveInitOptions{
				ConfigPath: request.ConfigPath,
				Quiet:      opts.Quiet,
				Debug:      opts.Debug,
			})
		},

		curseforgeFingerprint:      curseforgeFingerprint.GetFingerprintFor,
		modrinthVersionForSha:      defaultModrinthVersionForSha,
		modrinthProjectTitle:       defaultModrinthProjectTitle,
		curseforgeFingerprintMatch: defaultCurseforgeFingerprintMatch,
		curseforgeProjectName:      defaultCurseforgeProjectName,
	}
}

type scanCandidate struct {
	Path     string
	FileName string
	Sha1     string
}

type scanMatch struct {
	Path        string
	Platform    models.Platform
	ProjectID   string
	Name        string
	FileName    string
	Hash        string
	ReleaseDate string
	DownloadURL string
}

type scanUnsure struct {
	Path  string
	Error error
}

type platformLookupOutcome struct {
	matches []scanMatch
	misses  []scanCandidate
	unsure  map[string]error
}

func runScan(ctx context.Context, cmd *cobra.Command, opts scanOptions, deps scanDeps) (telemetry.CommandTelemetry, error) {
	meta := config.NewMetadata(opts.ConfigPath)
	setupCoordinator := modsetup.NewSetupCoordinator(deps.fs, deps.minecraftClient, nil)
	mode := interaction.ResolveExecutionMode(interaction.ExecutionModeInput{
		Unattended: opts.Unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
	})

	configState, err := ensureScanConfig(ctx, cmd, opts, deps, meta)
	if err != nil {
		return scanFailureTelemetry(mode.String(), mode.IsInteractive(), err), handleScanFailure(cmd, deps, err)
	}
	if !configState.ShouldContinue {
		return scanSuccessTelemetryWithoutArgs(mode.String(), mode.IsInteractive()), nil
	}
	return runScanWithConfig(ctx, cmd, opts, deps, setupCoordinator, configState, mode)
}

func runScanWithConfig(ctx context.Context, cmd *cobra.Command, opts scanOptions, deps scanDeps, setupCoordinator *modsetup.SetupCoordinator, configState scanConfigState, mode interaction.ExecutionMode) (telemetry.CommandTelemetry, error) {
	preferPlatform, err := resolvePreferredPlatform(opts.Prefer)
	if err != nil {
		return scanFailureTelemetry(mode.String(), mode.IsInteractive(), err), handleScanFailure(cmd, deps, err)
	}

	files, err := listJarFiles(deps.fs, configState.Meta, configState.Config)
	if err != nil {
		return scanFailureTelemetry(mode.String(), mode.IsInteractive(), err), handleScanFailure(cmd, deps, err)
	}

	unmanaged := unmanagedFiles(files, configState.Lock)
	if len(unmanaged) == 0 {
		return handleAllManagedScan(cmd, deps, opts, mode)
	}

	candidates, err := sha1Candidates(ctx, deps.fs, unmanaged)
	if err != nil {
		return scanFailureTelemetry(mode.String(), mode.IsInteractive(), err), handleScanFailure(cmd, deps, err)
	}

	executionInput := scanExecutionInput{
		meta:             configState.Meta,
		cfg:              configState.Config,
		lock:             configState.Lock,
		setupCoordinator: setupCoordinator,
		candidates:       candidates,
		preferPlatform:   preferPlatform,
		deps:             deps,
	}

	_, err = runScanByMode(ctx, cmd, executionInput, mode, opts)
	if err != nil {
		return scanFailureTelemetry(mode.String(), mode.IsInteractive(), err), err
	}
	return scanSuccessTelemetry(preferPlatform, opts.Add, mode.String(), mode.IsInteractive()), nil
}

func handleAllManagedScan(cmd *cobra.Command, deps scanDeps, opts scanOptions, mode interaction.ExecutionMode) (telemetry.CommandTelemetry, error) {
	if opts.Quiet {
		return scanSuccessTelemetryWithoutArgs(mode.String(), mode.IsInteractive()), nil
	}
	if outputErr := writeScanAllManaged(cmd, deps); outputErr != nil {
		return scanFailureTelemetry(mode.String(), mode.IsInteractive(), outputErr), outputErr
	}
	return scanSuccessTelemetryWithoutArgs(mode.String(), mode.IsInteractive()), nil
}

type initRequest struct {
	ConfigPath string
}

type initRunner func(context.Context, *cobra.Command, initRequest) error

type scanConfigState struct {
	Meta           config.Metadata
	Config         models.ModsJSON
	Lock           []models.ModInstall
	ShouldContinue bool
}

func ensureScanConfig(ctx context.Context, cmd *cobra.Command, opts scanOptions, deps scanDeps, meta config.Metadata) (scanConfigState, error) {
	cfg, err := config.ReadConfig(ctx, deps.fs, meta)
	if err == nil {
		lock, lockErr := config.EnsureLock(ctx, deps.fs, meta)
		if lockErr != nil {
			return scanConfigState{}, lockErr
		}
		return scanConfigState{Meta: meta, Config: cfg, Lock: lock, ShouldContinue: true}, nil
	}

	var notFound *config.ConfigFileNotFoundException
	if !errors.As(err, &notFound) {
		return scanConfigState{}, err
	}

	if promptErr := configMissingPromptError(opts, cmd, meta); promptErr != nil {
		if outputErr := writeConfigMissingOutput(cmd, deps, meta); outputErr != nil {
			return scanConfigState{}, outputErr
		}
		return scanConfigState{}, clierrors.MarkHandled(promptErr)
	}

	confirmed, canceled, err := runConfigInitPrompt(cmd, deps, meta)
	if err != nil {
		return scanConfigState{}, err
	}
	if canceled || !confirmed {
		return scanConfigState{Meta: meta, ShouldContinue: false}, nil
	}
	if deps.runInit == nil {
		return scanConfigState{}, errors.New("missing init runner")
	}
	if runErr := deps.runInit(ctx, cmd, initRequest{ConfigPath: meta.ConfigPath}); runErr != nil {
		if errors.Is(runErr, initCmd.ErrInitCanceled) {
			return scanConfigState{Meta: meta, ShouldContinue: false}, nil
		}
		return scanConfigState{}, runErr
	}

	cfg, err = config.ReadConfig(ctx, deps.fs, meta)
	if err != nil {
		return scanConfigState{}, err
	}
	lock, lockErr := config.EnsureLock(ctx, deps.fs, meta)
	if lockErr != nil {
		return scanConfigState{}, lockErr
	}
	return scanConfigState{Meta: meta, Config: cfg, Lock: lock, ShouldContinue: true}, nil
}

func configMissingPromptError(opts scanOptions, cmd *cobra.Command, meta config.Metadata) error {
	return interaction.CheckConfigInitGate(meta, interaction.ConfigInitGate{
		Unattended:      opts.Unattended,
		In:              cmd.InOrStdin(),
		Out:             cmd.OutOrStdout(),
		UnattendedError: configMissingError,
		NoTTYError:      configMissingError,
	})
}

func configMissingError(meta config.Metadata) error {
	return errors.New(i18n.T("cmd.scan.error.config_missing", &i18n.Tvars{
		Data: &i18n.TData{"configPath": meta.ConfigPath},
	}))
}

func mergeUnsure(preferred map[string]error, fallback map[string]error, matches []scanMatch) map[string]error {
	unsureByPath := make(map[string]error, len(preferred)+len(fallback))
	for path, outcomeErr := range preferred {
		unsureByPath[path] = outcomeErr
	}
	for path, outcomeErr := range fallback {
		unsureByPath[path] = outcomeErr
	}
	for _, match := range matches {
		delete(unsureByPath, match.Path)
	}
	return unsureByPath
}

func uniqueUint32s(values []uint32) []uint32 {
	seen := make(map[uint32]struct{}, len(values))
	result := make([]uint32, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func normalizePlatform(value string) models.Platform {
	switch strings.ToLower(strings.TrimSpace(value)) {
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

func defaultCurseforgeFingerprintMatch(ctx context.Context, fingerprints []uint32, doer httpclient.Doer) (*curseforge.FingerprintResult, error) {
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
