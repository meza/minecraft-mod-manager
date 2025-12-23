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

type installRunner func(context.Context, *cobra.Command, installOptions, installDeps) (Result, error)

func Command() *cobra.Command {
	return commandWithRunner(runInstall)
}

func commandWithRunner(runner installRunner) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "install",
		Aliases: []string{"i"},
		Short:   i18n.T("cmd.install.short"),
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			ctx, span := perf.StartSpan(cmd.Context(), "app.command.install")

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

			log := logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), quiet, debug)
			limiter := rate.NewLimiter(rate.Inf, 0)

			deps := installDeps{
				fs:         afero.NewOsFs(),
				logger:     log,
				clients:    platform.DefaultClients(limiter),
				downloader: httpclient.DownloadFile,
				fetchMod:   platform.FetchMod,
				telemetry:  telemetry.RecordCommand,

				curseforgeFingerprint:      curseforgeFingerprint.GetFingerprintFor,
				modrinthVersionForSha:      defaultModrinthVersionForSha,
				modrinthProjectTitle:       defaultModrinthProjectTitle,
				curseforgeFingerprintMatch: defaultCurseforgeFingerprintMatch,
				curseforgeProjectName:      defaultCurseforgeProjectName,
			}

			opts := installOptions{
				ConfigPath: configPath,
				Quiet:      quiet,
				Debug:      debug,
			}

			result, err := runner(ctx, cmd, opts, deps)
			span.SetAttributes(attribute.Bool("success", err == nil))
			span.End()

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
			deps.telemetry(payload)

			return err
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

	deps := installDeps{
		fs:         afero.NewOsFs(),
		logger:     log,
		clients:    platform.DefaultClients(limiter),
		downloader: httpclient.DownloadFile,
		fetchMod:   platform.FetchMod,
		telemetry:  func(telemetry.CommandTelemetry) {},

		curseforgeFingerprint:      curseforgeFingerprint.GetFingerprintFor,
		modrinthVersionForSha:      defaultModrinthVersionForSha,
		modrinthProjectTitle:       defaultModrinthProjectTitle,
		curseforgeFingerprintMatch: defaultCurseforgeFingerprintMatch,
		curseforgeProjectName:      defaultCurseforgeProjectName,
	}

	opts := installOptions{
		ConfigPath: configPath,
		Quiet:      quiet,
		Debug:      debug,
	}

	return runInstall(ctx, cmd, opts, deps)
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

	cfg, err := config.ReadConfig(ctx, deps.fs, meta)
	if err != nil {
		return Result{}, err
	}

	lock, err := config.EnsureLock(ctx, deps.fs, meta)
	if err != nil {
		return Result{}, err
	}

	colorize := tui.IsTerminalWriter(cmd.OutOrStdout())

	unresolved, unmanagedFound, err := preflightUnknownFiles(ctx, meta, cfg, lock, deps, colorize)
	if err != nil {
		return Result{}, err
	}
	if unresolved {
		deps.logger.Error(i18n.T("cmd.install.error.unresolved"))
		return Result{}, errUnresolvedFiles
	}

	if err := deps.fs.MkdirAll(meta.ModsFolderPath(cfg), 0755); err != nil {
		return Result{}, err
	}

	failedCount := 0

	for i := range cfg.Mods {
		mod := cfg.Mods[i]

		version := "latest"
		if mod.Version != nil && strings.TrimSpace(*mod.Version) != "" {
			version = strings.TrimSpace(*mod.Version)
		}
		deps.logger.Debug(i18n.T("cmd.install.debug.checking", i18n.Tvars{
			Data: &i18n.TData{
				"name":     mod.Name,
				"version":  version,
				"platform": mod.Type,
			},
		}))

		lockIndex := lockIndexFor(mod, lock)
		if lockIndex >= 0 {
			normalizedFileName, err := modfilename.Normalize(lock[lockIndex].FileName)
			if err != nil {
				deps.logger.Error(i18n.T("cmd.install.error.invalid_filename_lock", i18n.Tvars{
					Data: &i18n.TData{
						"name": mod.Name,
						"file": modfilename.Display(lock[lockIndex].FileName),
					},
				}))
				failedCount++
				continue
			}
			installEntry := lock[lockIndex]
			installEntry.FileName = normalizedFileName
			if err := ensureLockInstall(ctx, meta, cfg, mod, installEntry, deps); err != nil {
				if message, handled := integrityErrorMessage(err, mod.Name); handled {
					deps.logger.Error(message)
					failedCount++
					continue
				}
				return Result{}, err
			}
			continue
		}

		remote, fetchErr := deps.fetchMod(ctx, mod.Type, mod.ID, platform.FetchOptions{
			AllowedReleaseTypes: effectiveAllowedReleaseTypes(mod, cfg),
			GameVersion:         cfg.GameVersion,
			Loader:              cfg.Loader,
			AllowFallback:       mod.AllowVersionFallback != nil && *mod.AllowVersionFallback,
			FixedVersion:        optionalStringValue(mod.Version),
		}, deps.clients)
		if fetchErr != nil {
			if handleExpectedFetchError(fetchErr, mod, deps, colorize) {
				continue
			}
			return Result{}, fetchErr
		}

		normalizedFileName, err := modfilename.Normalize(remote.FileName)
		if err != nil {
			deps.logger.Error(i18n.T("cmd.install.error.invalid_filename_remote", i18n.Tvars{
				Data: &i18n.TData{
					"name": mod.Name,
					"file": modfilename.Display(remote.FileName),
				},
			}))
			failedCount++
			continue
		}
		remote.FileName = normalizedFileName

		if strings.TrimSpace(remote.Hash) == "" {
			deps.logger.Error(i18n.T("cmd.install.error.missing_hash_remote", i18n.Tvars{
				Data: &i18n.TData{"name": mod.Name},
			}))
			failedCount++
			continue
		}

		deps.logger.Log(i18n.T("cmd.install.download.missing", i18n.Tvars{
			Data: &i18n.TData{
				"name":     mod.Name,
				"platform": mod.Type,
			},
		}), true)

		destination := filepath.Join(meta.ModsFolderPath(cfg), remote.FileName)
		resolvedDestination, err := modpath.ResolveWritablePath(deps.fs, meta.ModsFolderPath(cfg), destination)
		if err != nil {
			if message, handled := integrityErrorMessage(err, mod.Name); handled {
				deps.logger.Error(message)
				failedCount++
				continue
			}
			return Result{}, err
		}
		installer := modinstall.NewInstaller(deps.fs, modinstall.Downloader(deps.downloader))
		if err := installer.DownloadAndVerify(ctx, remote.DownloadURL, resolvedDestination, remote.Hash, downloadClient(deps.clients), &noopSender{}); err != nil {
			if message, handled := integrityErrorMessage(err, mod.Name); handled {
				deps.logger.Error(message)
				failedCount++
				continue
			}
			return Result{}, err
		}

		cfg.Mods[i].Name = remote.Name
		lock = append(lock, models.ModInstall{
			Type:        mod.Type,
			ID:          mod.ID,
			Name:        remote.Name,
			FileName:    remote.FileName,
			ReleasedOn:  remote.ReleaseDate,
			Hash:        remote.Hash,
			DownloadURL: remote.DownloadURL,
		})
	}

	if err := config.WriteLock(ctx, deps.fs, meta, lock); err != nil {
		return Result{}, err
	}
	if err := config.WriteConfig(ctx, deps.fs, meta, cfg); err != nil {
		return Result{}, err
	}

	if failedCount > 0 {
		return Result{InstalledCount: len(cfg.Mods), UnmanagedFound: unmanagedFound}, errInstallFailures
	}

	deps.logger.Log(messageWithIcon(tui.SuccessIcon(colorize), i18n.T("cmd.install.success")), true)
	return Result{InstalledCount: len(cfg.Mods), UnmanagedFound: unmanagedFound}, nil
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
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

func handleExpectedFetchError(err error, mod models.Mod, deps installDeps, colorize bool) bool {
	var notFound *platform.ModNotFoundError
	if errors.As(err, &notFound) {
		deps.logger.Log(messageWithIcon(tui.ErrorIcon(colorize), i18n.T("cmd.install.error.mod_not_found", i18n.Tvars{
			Data: &i18n.TData{
				"name":     mod.Name,
				"id":       mod.ID,
				"platform": mod.Type,
			},
		})), true)
		return true
	}

	var noFile *platform.NoCompatibleFileError
	if errors.As(err, &noFile) {
		deps.logger.Log(messageWithIcon(tui.ErrorIcon(colorize), i18n.T("cmd.install.error.no_file", i18n.Tvars{
			Data: &i18n.TData{
				"name":     mod.Name,
				"id":       mod.ID,
				"platform": mod.Type,
			},
		})), true)
		return true
	}

	return false
}

func messageWithIcon(icon string, message string) string {
	return fmt.Sprintf("%s %s", icon, message)
}

func preflightUnknownFiles(ctx context.Context, meta config.Metadata, cfg models.ModsJSON, lock []models.ModInstall, deps installDeps, colorize bool) (bool, bool, error) {
	files, err := listModFiles(deps.fs, meta, cfg)
	if err != nil {
		return false, false, err
	}

	nonManaged := make([]string, 0)
	for _, file := range files {
		if !fileIsManaged(file, lock) {
			nonManaged = append(nonManaged, file)
		}
	}

	if len(nonManaged) == 0 {
		return false, false, nil
	}

	scanned, err := scanFiles(ctx, nonManaged, deps)
	if err != nil {
		return false, false, err
	}

	return reportScanResults(scanned, cfg, lock, deps, colorize)
}

func scanFiles(ctx context.Context, files []string, deps installDeps) ([]scannedFile, error) {
	results := make([]scannedFile, 0, len(files))

	fingerprints := make([]int, 0, len(files))
	fingerprintToIndices := make(map[int][]int, len(files))
	for i, file := range files {
		sha, err := sha1ForFile(deps.fs, file)
		if err != nil {
			return nil, err
		}

		fingerprint := int(deps.curseforgeFingerprint(file))
		fingerprints = append(fingerprints, fingerprint)
		fingerprintToIndices[fingerprint] = append(fingerprintToIndices[fingerprint], i)

		results = append(results, scannedFile{
			Path: file,
			Sha1: sha,
		})
	}

	sort.Ints(fingerprints)

	curseforgeByFingerprint, err := curseforgeMatchesByFingerprint(ctx, fingerprints, deps)
	if err != nil {
		return nil, err
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

	for i := range results {
		version, err := deps.modrinthVersionForSha(ctx, results[i].Sha1, deps.clients.Modrinth)
		if err != nil {
			var notFound *modrinth.VersionNotFoundError
			if errors.As(err, &notFound) {
				continue
			}
			return nil, err
		}

		name, err := deps.modrinthProjectTitle(ctx, version.ProjectID, deps.clients.Modrinth)
		if err != nil {
			return nil, err
		}

		results[i].Hits = append(results[i].Hits, scanHit{
			Platform: models.MODRINTH,
			Project:  version.ProjectID,
			Name:     name,
		})
	}

	for i := range results {
		results[i].Hits = sortHitsPreferModrinth(results[i].Hits)
	}

	return results, nil
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

func reportScanResults(scanned []scannedFile, cfg models.ModsJSON, lock []models.ModInstall, deps installDeps, colorize bool) (bool, bool, error) {
	unresolved := false
	unmanagedFound := false

	for _, item := range scanned {
		if len(item.Hits) == 0 {
			continue
		}

		matchedModIndex := findConfiguredModIndex(cfg, item.Hits)
		if matchedModIndex < 0 {
			unmanagedFound = true
			name := item.Hits[0].Name
			if colorize {
				name = tui.TitleStyle.Copy().Bold(true).Render(name)
			}
			deps.logger.Log(tui.SuccessIcon(colorize)+i18n.T("cmd.install.unmanaged.found", i18n.Tvars{
				Data: &i18n.TData{"name": name},
			}), true)
			continue
		}

		mod := cfg.Mods[matchedModIndex]
		lockIndex := lockIndexFor(mod, lock)
		if lockIndex < 0 {
			deps.logger.Log(messageWithIcon(tui.ErrorIcon(colorize), i18n.T("cmd.install.unsure.lock_missing", i18n.Tvars{
				Data: &i18n.TData{"name": item.Hits[0].Name},
			})), true)
			unresolved = true
			continue
		}

		if !strings.EqualFold(lock[lockIndex].Hash, item.Sha1) {
			deps.logger.Log(messageWithIcon(tui.ErrorIcon(colorize), i18n.T("cmd.install.unsure.hash_mismatch", i18n.Tvars{
				Data: &i18n.TData{"name": item.Hits[0].Name},
			})), true)
			unresolved = true
		}
	}

	return unresolved, unmanagedFound, nil
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

func (n *noopSender) Send(msg tea.Msg) { _ = msg }

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
