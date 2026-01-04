package scan

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

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
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	tui "github.com/meza/minecraft-mod-manager/internal/view"
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
	output          *output.Output
	prompter        prompter
	telemetry       func(telemetry.CommandTelemetry)
	runInit         initRunner

	curseforgeFingerprint      func(string) uint32
	modrinthVersionForSha      func(context.Context, string, httpclient.Doer) (*modrinth.Version, error)
	modrinthProjectTitle       func(context.Context, string, httpclient.Doer) (string, error)
	curseforgeFingerprintMatch func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error)
	curseforgeProjectName      func(context.Context, string, httpclient.Doer) (string, error)
}

type prompter interface {
	ConfirmAdd() (bool, error)
	ConfirmInit(configPath string) (bool, error)
}

type terminalPrompter struct {
	in  io.Reader
	out io.Writer
}

type noopPrompter struct{}

func (prompter noopPrompter) ConfirmAdd() (bool, error) {
	return false, nil
}

func (prompter noopPrompter) ConfirmInit(string) (bool, error) {
	return false, nil
}

func (prompter terminalPrompter) ConfirmAdd() (bool, error) {
	return tui.RunConfirmPrompt(prompter.in, prompter.out, tui.ConfirmPrompt{
		Question:    i18n.T("cmd.scan.confirm_add", nil),
		DefaultHint: "y/N",
	})
}

func (prompter terminalPrompter) ConfirmInit(configPath string) (bool, error) {
	if _, err := fmt.Fprintln(prompter.out, i18n.T("cmd.scan.config_missing", &i18n.Tvars{
		Data: &i18n.TData{"configPath": configPath},
	})); err != nil {
		return false, err
	}
	return tui.RunConfirmPrompt(prompter.in, prompter.out, tui.ConfirmPrompt{
		Question:    i18n.T("cmd.scan.confirm_init", nil),
		DefaultHint: "y/N",
	})
}

func messageWithIcon(icon string, message string) string {
	return fmt.Sprintf("%s %s", icon, message)
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
		output:          common.Output,
		prompter:        pickPrompter(opts, cmd.InOrStdin(), cmd.OutOrStdout()),
		telemetry:       telemetry.RecordCommand,
		runInit: func(ctx context.Context, cmd *cobra.Command, request initRequest) error {
			return runInteractiveInit(ctx, cmd, initCmd.InteractiveInitDeps{
				FS:              common.FS,
				Output:          common.Output,
				Logger:          common.Logger,
				MinecraftClient: common.MinecraftClient,
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

func pickPrompter(options scanOptions, in io.Reader, out io.Writer) prompter {
	if options.Unattended || !tui.SupportsPrompting(in, out) {
		return noopPrompter{}
	}
	return terminalPrompter{in: in, out: out}
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

type candidateIdentification struct {
	matches []scanMatch
	unknown []string
	unsure  []scanUnsure
}

func runScan(ctx context.Context, cmd *cobra.Command, opts scanOptions, deps scanDeps) (telemetry.CommandTelemetry, error) {
	meta := config.NewMetadata(opts.ConfigPath)
	setupCoordinator := modsetup.NewSetupCoordinator(deps.fs, deps.minecraftClient, nil)

	configState, err := ensureScanConfig(ctx, cmd, opts, deps, meta)
	if err != nil {
		return scanFailureTelemetry(err), err
	}
	if !configState.ShouldContinue {
		return scanSuccessTelemetryWithoutArgs(), nil
	}
	return runScanWithConfig(ctx, cmd, opts, deps, meta, setupCoordinator, configState.Config, configState.Lock)
}

func runScanWithConfig(ctx context.Context, cmd *cobra.Command, opts scanOptions, deps scanDeps, meta config.Metadata, setupCoordinator *modsetup.SetupCoordinator, cfg models.ModsJSON, lock []models.ModInstall) (telemetry.CommandTelemetry, error) {
	preferPlatform, err := resolvePreferredPlatform(opts.Prefer, deps.output)
	if err != nil {
		return scanFailureTelemetry(err), err
	}

	files, err := listJarFiles(deps.fs, meta, cfg)
	if err != nil {
		return scanFailureTelemetry(err), err
	}

	unmanaged := unmanagedFiles(files, lock)
	colorMode := colorModeForOutput(cmd.OutOrStdout())

	if len(unmanaged) == 0 {
		return reportAllManaged(deps.output, colorMode)
	}

	candidates, err := sha1Candidates(ctx, deps.fs, unmanaged)
	if err != nil {
		return scanFailureTelemetry(err), err
	}

	identification, err := identifyAndPrintCandidates(ctx, deps.output, cmd.OutOrStdout(), preferPlatform, candidates, deps)
	if err != nil {
		return scanFailureTelemetry(err), err
	}

	return persistScanMatchesIfRequested(persistScanRequest{
		Context:          ctx,
		Command:          cmd,
		Options:          opts,
		Dependencies:     deps,
		Metadata:         meta,
		SetupCoordinator: setupCoordinator,
		Matches:          identification.matches,
		Unsure:           identification.unsure,
		Config:           cfg,
		Lock:             lock,
		PreferPlatform:   preferPlatform,
		ColorMode:        colorMode,
	})
}

type initRequest struct {
	ConfigPath string
}

type initRunner func(context.Context, *cobra.Command, initRequest) error

type scanConfigState struct {
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
		return scanConfigState{Config: cfg, Lock: lock, ShouldContinue: true}, nil
	}

	var notFound *config.ConfigFileNotFoundException
	if !errors.As(err, &notFound) {
		return scanConfigState{}, err
	}

	if promptErr := configMissingPromptError(opts, cmd, meta); promptErr != nil {
		return scanConfigState{}, promptErr
	}

	confirmInit, err := deps.prompter.ConfirmInit(meta.ConfigPath)
	if err != nil {
		return scanConfigState{}, err
	}
	if !confirmInit {
		return scanConfigState{ShouldContinue: false}, nil
	}
	if deps.runInit == nil {
		return scanConfigState{}, errors.New("missing init runner")
	}
	if runErr := deps.runInit(ctx, cmd, initRequest{ConfigPath: meta.ConfigPath}); runErr != nil {
		if errors.Is(runErr, initCmd.ErrInitCanceled) {
			return scanConfigState{ShouldContinue: false}, nil
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
	return scanConfigState{Config: cfg, Lock: lock, ShouldContinue: true}, nil
}

func configMissingPromptError(opts scanOptions, cmd *cobra.Command, meta config.Metadata) error {
	return interaction.CheckConfigInitGate(meta, interaction.ConfigInitGate{
		Unattended: opts.Unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
		UnattendedError: func(meta config.Metadata) error {
			return configMissingUnattendedError(meta)
		},
		NoTTYError: func(meta config.Metadata) error {
			return configMissingNoTTYError(meta)
		},
	})
}

func configMissingUnattendedError(meta config.Metadata) error {
	return errors.New(i18n.T("cmd.scan.error.config_missing_noninteractive", &i18n.Tvars{
		Data: &i18n.TData{"configPath": meta.ConfigPath},
	}))
}

func configMissingNoTTYError(meta config.Metadata) error {
	return errors.New(i18n.T("cmd.scan.error.config_missing_no_tty", &i18n.Tvars{
		Data: &i18n.TData{"configPath": meta.ConfigPath},
	}))
}

func reportAllManaged(out *output.Output, colorMode tui.ColorMode) (telemetry.CommandTelemetry, error) {
	if outputErr := out.Log(messageWithIcon(tui.SuccessIcon(colorMode), i18n.T("cmd.scan.all_managed", nil)), output.LogQuiet); outputErr != nil {
		return scanFailureTelemetry(outputErr), outputErr
	}
	return scanSuccessTelemetryWithoutArgs(), nil
}

func identifyAndPrintCandidates(ctx context.Context, out *output.Output, writer io.Writer, preferPlatform models.Platform, candidates []scanCandidate, deps scanDeps) (candidateIdentification, error) {
	identification, err := identifyCandidates(ctx, candidates, preferPlatform, deps)
	if err != nil {
		return candidateIdentification{}, err
	}
	if err := printResults(out, writer, preferPlatform, identification.matches, identification.unknown, identification.unsure); err != nil {
		return candidateIdentification{}, err
	}
	return identification, nil
}

func persistScanMatches(ctx context.Context, cmd *cobra.Command, meta config.Metadata, setupCoordinator *modsetup.SetupCoordinator, deps scanDeps, matches []scanMatch, cfg models.ModsJSON, lock []models.ModInstall) (bool, error) {
	changedConfig := false
	changedLock := false
	colorMode := tui.ColorDisabled
	if tui.IsTerminalWriter(cmd.OutOrStdout()) {
		colorMode = tui.ColorEnabled
	}

	for _, match := range matches {
		outcome, err := setupCoordinator.UpsertConfigAndLock(cfg, lock, match.Platform, match.ProjectID, platform.RemoteMod{
			Name:        match.Name,
			FileName:    match.FileName,
			Hash:        match.Hash,
			ReleaseDate: match.ReleaseDate,
			DownloadURL: match.DownloadURL,
		}, modsetup.EnsurePersistOptions{})
		if err != nil {
			if outputErr := deps.output.Log(tui.ErrorIcon(colorMode)+i18n.T("cmd.scan.persist_failed", &i18n.Tvars{
				Data: &i18n.TData{"file": match.FileName},
			}), output.LogQuiet); outputErr != nil {
				return false, outputErr
			}
			continue
		}

		if outcome.Result.ConfigAdded || outcome.Result.ConfigUpdated {
			changedConfig = true
		}
		if outcome.Result.LockAdded || outcome.Result.LockUpdated {
			changedLock = true
		}

		cfg = outcome.Config
		lock = outcome.Lock
	}

	if changedConfig {
		if err := config.WriteConfig(ctx, deps.fs, meta, cfg); err != nil {
			return false, err
		}
	}
	if changedLock {
		if err := config.WriteLock(ctx, deps.fs, meta, lock); err != nil {
			return false, err
		}
	}

	return changedConfig || changedLock, nil
}

func identifyCandidates(ctx context.Context, candidates []scanCandidate, prefer models.Platform, deps scanDeps) (candidateIdentification, error) {
	preferredOutcome, err := lookupOnPlatform(ctx, candidates, prefer, deps)
	if err != nil {
		return candidateIdentification{}, err
	}

	fallback := alternatePlatform(prefer)
	fallbackOutcome, err := lookupOnPlatform(ctx, preferredOutcome.misses, fallback, deps)
	if err != nil {
		return candidateIdentification{}, err
	}

	matches := combineMatches(preferredOutcome.matches, fallbackOutcome.matches)
	unsureByPath := mergeUnsure(preferredOutcome.unsure, fallbackOutcome.unsure, matches)
	unknown := collectUnknownPaths(fallbackOutcome.misses, unsureByPath)
	unsure := buildUnsureList(unsureByPath)
	sortMatchesByPreference(matches, prefer)

	return candidateIdentification{
		matches: matches,
		unknown: unknown,
		unsure:  unsure,
	}, nil
}

func combineMatches(preferred []scanMatch, fallback []scanMatch) []scanMatch {
	matches := make([]scanMatch, 0, len(preferred)+len(fallback))
	matches = append(matches, preferred...)
	matches = append(matches, fallback...)
	return matches
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

func collectUnknownPaths(misses []scanCandidate, unsureByPath map[string]error) []string {
	unknown := make([]string, 0, len(misses))
	for _, miss := range misses {
		if _, isUnsure := unsureByPath[miss.Path]; isUnsure {
			continue
		}
		unknown = append(unknown, miss.Path)
	}
	sort.Strings(unknown)
	return unknown
}

func buildUnsureList(unsureByPath map[string]error) []scanUnsure {
	unsure := make([]scanUnsure, 0, len(unsureByPath))
	for path, outcomeErr := range unsureByPath {
		unsure = append(unsure, scanUnsure{Path: path, Error: outcomeErr})
	}
	sort.SliceStable(unsure, func(i, j int) bool { return unsure[i].Path < unsure[j].Path })
	return unsure
}

func sortMatchesByPreference(matches []scanMatch, prefer models.Platform) {
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Platform != matches[j].Platform {
			return matches[i].Platform == prefer
		}
		if matches[i].Name != matches[j].Name {
			return matches[i].Name < matches[j].Name
		}
		return matches[i].FileName < matches[j].FileName
	})
}

func lookupOnPlatform(ctx context.Context, candidates []scanCandidate, platformValue models.Platform, deps scanDeps) (platformLookupOutcome, error) {
	switch platformValue {
	case models.MODRINTH:
		return lookupModrinth(ctx, candidates, deps)
	case models.CURSEFORGE:
		return lookupCurseforge(ctx, candidates, deps)
	default:
		return platformLookupOutcome{
			matches: nil,
			misses:  candidates,
			unsure:  map[string]error{},
		}, nil
	}
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

func printResults(out *output.Output, writer io.Writer, _ models.Platform, matches []scanMatch, unknown []string, unsure []scanUnsure) error {
	colorMode := tui.ColorDisabled
	if tui.IsTerminalWriter(writer) {
		colorMode = tui.ColorEnabled
	}

	if len(matches) > 0 {
		if err := printMatchResults(out, colorMode, matches); err != nil {
			return err
		}
	}

	if len(unknown) > 0 {
		if err := printUnknownResults(out, colorMode, unknown); err != nil {
			return err
		}
	}

	if len(unsure) > 0 {
		if err := printUnsureResults(out, colorMode, unsure); err != nil {
			return err
		}
	}

	if len(matches) == 0 && len(unknown) == 0 && len(unsure) == 0 {
		if err := out.Log(i18n.T("cmd.scan.no_results", nil), output.LogQuiet); err != nil {
			return err
		}
	}
	return nil
}

func printMatchResults(out *output.Output, colorMode tui.ColorMode, matches []scanMatch) error {
	if err := out.Log(i18n.T("cmd.scan.recognized.header", nil), output.LogQuiet); err != nil {
		return err
	}
	for _, match := range matches {
		name := tui.RenderIfColorEnabled(colorMode, tui.TitleStyle.Bold(true), match.Name)
		if err := out.Log(messageWithIcon(tui.SuccessIcon(colorMode), i18n.T("cmd.scan.recognized.entry", &i18n.Tvars{
			Data: &i18n.TData{
				"name":     name,
				"platform": string(match.Platform),
				"id":       match.ProjectID,
				"file":     match.FileName,
			},
		})), output.LogQuiet); err != nil {
			return err
		}
	}
	return nil
}

func printUnknownResults(out *output.Output, colorMode tui.ColorMode, unknown []string) error {
	if err := out.Log(i18n.T("cmd.scan.unknown.header", nil), output.LogQuiet); err != nil {
		return err
	}
	for _, file := range unknown {
		if err := out.Log(messageWithIcon(tui.ErrorIcon(colorMode), i18n.T("cmd.scan.unknown.entry", &i18n.Tvars{
			Data: &i18n.TData{"file": filepath.Base(file)},
		})), output.LogQuiet); err != nil {
			return err
		}
	}
	return nil
}

func printUnsureResults(out *output.Output, colorMode tui.ColorMode, unsure []scanUnsure) error {
	if err := out.Log(i18n.T("cmd.scan.unsure.header", nil), output.LogQuiet); err != nil {
		return err
	}
	for _, item := range unsure {
		if err := out.Log(messageWithIcon(tui.ErrorIcon(colorMode), i18n.T("cmd.scan.unsure.entry_with_reason", &i18n.Tvars{
			Data: &i18n.TData{
				"file":   filepath.Base(item.Path),
				"reason": unsureReason(item.Error),
			},
		})), output.LogQuiet); err != nil {
			return err
		}
	}
	return nil
}

func unsureReason(err error) string {
	if err != nil {
		return err.Error()
	}
	return "unknown error"
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
