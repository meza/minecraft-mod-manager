package scan

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	curseforgeFingerprint "github.com/meza/curseforge-fingerprint-go"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"github.com/meza/minecraft-mod-manager/internal/modsetup"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/tui"
)

type scanOptions struct {
	ConfigPath string
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
	prompter        prompter
	telemetry       func(telemetry.CommandTelemetry)

	curseforgeFingerprint      func(string) uint32
	modrinthVersionForSha      func(context.Context, string, httpclient.Doer) (*modrinth.Version, error)
	modrinthProjectTitle       func(context.Context, string, httpclient.Doer) (string, error)
	curseforgeFingerprintMatch func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error)
	curseforgeProjectName      func(context.Context, string, httpclient.Doer) (string, error)
}

type prompter interface {
	ConfirmAdd() (bool, error)
}

type terminalPrompter struct {
	in  io.Reader
	out io.Writer
}

func (prompter terminalPrompter) ConfirmAdd() (bool, error) {
	if _, err := fmt.Fprintf(prompter.out, "%s (y/N): ", i18n.T("cmd.scan.confirm_add")); err != nil {
		return false, err
	}
	answer, err := readLine(prompter.in)
	if err != nil {
		return false, err
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes", nil
}

func readLine(reader io.Reader) (string, error) {
	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return scanner.Text(), nil
}

func messageWithIcon(icon string, message string) string {
	return fmt.Sprintf("%s %s", icon, message)
}

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: i18n.T("cmd.scan.short"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runScanCommand(cmd)
		},
	}

	cmd.Flags().StringP("prefer", "p", string(models.MODRINTH), i18n.T("cmd.scan.flag.prefer"))
	cmd.Flags().BoolP("add", "a", false, i18n.T("cmd.scan.flag.add"))

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
	return err
}

func scanOptionsFromFlags(cmd *cobra.Command) (scanOptions, error) {
	configPath, err := cmd.Flags().GetString("config")
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
		Quiet:      quiet,
		Debug:      debug,
		Prefer:     prefer,
		Add:        add,
	}, nil
}

func defaultScanDeps(cmd *cobra.Command, opts scanOptions) scanDeps {
	log := logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), opts.Quiet, opts.Debug)
	limiter := httpclient.DefaultLimiter()

	return scanDeps{
		fs:              afero.NewOsFs(),
		clients:         platform.DefaultClients(limiter),
		minecraftClient: httpclient.NewRLClient(limiter),
		logger:          log,
		prompter:        terminalPrompter{in: cmd.InOrStdin(), out: cmd.OutOrStdout()},
		telemetry:       telemetry.RecordCommand,

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

func runScan(ctx context.Context, cmd *cobra.Command, opts scanOptions, deps scanDeps) (telemetry.CommandTelemetry, error) {
	meta := config.NewMetadata(opts.ConfigPath)
	setupCoordinator := modsetup.NewSetupCoordinator(deps.fs, deps.minecraftClient, nil)

	cfg, lock, err := setupCoordinator.EnsureConfigAndLock(ctx, meta, modsetup.EnsureConfigOptions{Quiet: opts.Quiet})
	if err != nil {
		return scanFailureTelemetry(err), err
	}

	preferPlatform, err := resolvePreferredPlatform(opts.Prefer, deps.logger)
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
		deps.logger.Log(messageWithIcon(tui.SuccessIcon(colorMode), i18n.T("cmd.scan.all_managed")), logger.LogQuiet)
		return scanSuccessTelemetryWithoutArgs(), nil
	}

	candidates, err := sha1Candidates(ctx, deps.fs, unmanaged)
	if err != nil {
		return scanFailureTelemetry(err), err
	}

	matches, unknown, unsure := identifyCandidates(ctx, candidates, preferPlatform, deps)
	printResults(deps.logger, cmd.OutOrStdout(), preferPlatform, matches, unknown, unsure)

	return persistScanMatchesIfRequested(persistScanRequest{
		Context:          ctx,
		Command:          cmd,
		Options:          opts,
		Dependencies:     deps,
		Metadata:         meta,
		SetupCoordinator: setupCoordinator,
		Matches:          matches,
		Unsure:           unsure,
		Config:           cfg,
		Lock:             lock,
		PreferPlatform:   preferPlatform,
		ColorMode:        colorMode,
	})
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
			deps.logger.Log(tui.ErrorIcon(colorMode)+i18n.T("cmd.scan.persist_failed", i18n.Tvars{
				Data: &i18n.TData{"file": match.FileName},
			}), logger.LogQuiet)
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

func identifyCandidates(ctx context.Context, candidates []scanCandidate, prefer models.Platform, deps scanDeps) ([]scanMatch, []string, []scanUnsure) {
	preferredMatches, preferredMisses, preferUnsure := lookupOnPlatform(ctx, candidates, prefer, deps)

	fallback := alternatePlatform(prefer)
	fallbackMatches, fallbackMisses, fallbackUnsure := lookupOnPlatform(ctx, preferredMisses, fallback, deps)

	matches := make([]scanMatch, 0, len(preferredMatches)+len(fallbackMatches))
	matches = append(matches, preferredMatches...)
	matches = append(matches, fallbackMatches...)

	for path, err := range fallbackUnsure {
		preferUnsure[path] = err
	}
	for _, match := range matches {
		delete(preferUnsure, match.Path)
	}

	unknown := make([]string, 0, len(fallbackMisses))
	for _, miss := range fallbackMisses {
		if _, isUnsure := preferUnsure[miss.Path]; isUnsure {
			continue
		}
		unknown = append(unknown, miss.Path)
	}
	sort.Strings(unknown)

	unsure := make([]scanUnsure, 0, len(preferUnsure))
	for path, err := range preferUnsure {
		unsure = append(unsure, scanUnsure{Path: path, Error: err})
	}
	sort.SliceStable(unsure, func(i, j int) bool { return unsure[i].Path < unsure[j].Path })

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Platform != matches[j].Platform {
			return matches[i].Platform == prefer
		}
		if matches[i].Name != matches[j].Name {
			return matches[i].Name < matches[j].Name
		}
		return matches[i].FileName < matches[j].FileName
	})

	return matches, unknown, unsure
}

func lookupOnPlatform(ctx context.Context, candidates []scanCandidate, platformValue models.Platform, deps scanDeps) ([]scanMatch, []scanCandidate, map[string]error) {
	switch platformValue {
	case models.MODRINTH:
		return lookupModrinth(ctx, candidates, deps)
	case models.CURSEFORGE:
		return lookupCurseforge(ctx, candidates, deps)
	default:
		return nil, candidates, map[string]error{}
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

func printResults(log *logger.Logger, out io.Writer, _ models.Platform, matches []scanMatch, unknown []string, unsure []scanUnsure) {
	colorMode := tui.ColorDisabled
	if tui.IsTerminalWriter(out) {
		colorMode = tui.ColorEnabled
	}

	if len(matches) > 0 {
		log.Log(i18n.T("cmd.scan.recognized.header"), logger.LogQuiet)
		for _, match := range matches {
			name := tui.RenderIfColorEnabled(colorMode, tui.TitleStyle.Bold(true), match.Name)
			log.Log(messageWithIcon(tui.SuccessIcon(colorMode), i18n.T("cmd.scan.recognized.entry", i18n.Tvars{
				Data: &i18n.TData{
					"name":     name,
					"platform": match.Platform,
					"id":       match.ProjectID,
					"file":     match.FileName,
				},
			})), logger.LogQuiet)
		}
	}

	if len(unknown) > 0 {
		log.Log(i18n.T("cmd.scan.unknown.header"), logger.LogQuiet)
		for _, file := range unknown {
			log.Log(messageWithIcon(tui.ErrorIcon(colorMode), i18n.T("cmd.scan.unknown.entry", i18n.Tvars{
				Data: &i18n.TData{"file": filepath.Base(file)},
			})), logger.LogQuiet)
		}
	}

	if len(unsure) > 0 {
		log.Log(i18n.T("cmd.scan.unsure.header"), logger.LogQuiet)
		for _, item := range unsure {
			reason := "unknown error"
			if item.Error != nil {
				reason = item.Error.Error()
			}
			log.Log(messageWithIcon(tui.ErrorIcon(colorMode), i18n.T("cmd.scan.unsure.entry_with_reason", i18n.Tvars{
				Data: &i18n.TData{
					"file":   filepath.Base(item.Path),
					"reason": reason,
				},
			})), logger.LogQuiet)
		}
	}

	if len(matches) == 0 && len(unknown) == 0 && len(unsure) == 0 {
		log.Log(i18n.T("cmd.scan.no_results"), logger.LogQuiet)
	}
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
