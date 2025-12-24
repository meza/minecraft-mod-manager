package update

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/cmd/mmm/install"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/modpath"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/tui"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/time/rate"
)

type updateOptions struct {
	ConfigPath string
	Quiet      bool
	Debug      bool
}

type updateDeps struct {
	fs         afero.Fs
	logger     *logger.Logger
	clients    platform.Clients
	fetchMod   fetcher
	downloader downloader
	install    installer
	telemetry  func(telemetry.CommandTelemetry)
}

type fetcher func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error)

type downloader func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error

type installer func(context.Context, *cobra.Command, string, bool, bool) (install.Result, error)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update",
		Aliases: []string{"u"},
		Short:   i18n.T("cmd.update.short"),
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUpdateCommand(cmd)
		},
	}

	return cmd
}

func runUpdateCommand(cmd *cobra.Command) error {
	ctx, span := perf.StartSpan(cmd.Context(), "app.command.update")

	opts, err := updateOptionsFromFlags(cmd)
	if err != nil {
		span.SetAttributes(attribute.Bool("success", false))
		span.End()
		return err
	}

	deps := defaultUpdateDeps(cmd, opts)
	updated, failed, err := runUpdate(ctx, cmd, opts, deps)
	span.SetAttributes(attribute.Bool("success", err == nil))
	span.End()

	if err != nil {
		cmd.SilenceUsage = true
	}

	recordUpdateTelemetry(deps.telemetry, updated, failed, err)
	return err
}

func updateOptionsFromFlags(cmd *cobra.Command) (updateOptions, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return updateOptions{}, err
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return updateOptions{}, err
	}
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return updateOptions{}, err
	}

	return updateOptions{
		ConfigPath: configPath,
		Quiet:      quiet,
		Debug:      debug,
	}, nil
}

func defaultUpdateDeps(cmd *cobra.Command, opts updateOptions) updateDeps {
	log := logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), opts.Quiet, opts.Debug)
	limiter := rate.NewLimiter(rate.Inf, 0)

	return updateDeps{
		fs:         afero.NewOsFs(),
		logger:     log,
		clients:    platform.DefaultClients(limiter),
		fetchMod:   platform.FetchMod,
		downloader: httpclient.DownloadFile,
		install:    install.Run,
		telemetry:  telemetry.RecordCommand,
	}
}

func recordUpdateTelemetry(telemetryRecorder func(telemetry.CommandTelemetry), updated int, failed int, err error) {
	payload := telemetry.CommandTelemetry{
		Command:     "update",
		Success:     err == nil,
		Error:       err,
		ExitCode:    0,
		Interactive: false,
		Extra: map[string]interface{}{
			"updatedMods": updated,
			"failedMods":  failed,
		},
	}
	if err != nil {
		payload.ExitCode = 1
	}
	telemetryRecorder(payload)
}

type modUpdateCandidate struct {
	ConfigIndex int
	Mod         models.Mod
}

type modUpdateOutcome struct {
	ConfigIndex int
	LockIndex   int
	NewName     string
	NewInstall  models.ModInstall
	LogEvents   []logEvent
	Updated     bool
	Error       error
}

type logEventKind int

const (
	logEventKindLog logEventKind = iota
	logEventKindError
	logEventKindDebug
)

type logEvent struct {
	Kind      logEventKind
	Message   string
	ForceShow bool
}

//nolint:gocognit,gocyclo,funlen // Update flow mirrors spec stages; splitting obscures behavior.
func runUpdate(ctx context.Context, cmd *cobra.Command, opts updateOptions, deps updateDeps) (int, int, error) {
	installResult, err := deps.install(ctx, cmd, opts.ConfigPath, opts.Quiet, opts.Debug)
	if err != nil {
		return 0, 0, err
	}
	if installResult.UnmanagedFound {
		deps.logger.Error(i18n.T("cmd.update.error.unmanaged_found"))
		return 0, 0, errUnmanagedFiles
	}

	meta := config.NewMetadata(opts.ConfigPath)

	cfg, err := config.ReadConfig(ctx, deps.fs, meta)
	if err != nil {
		return 0, 0, err
	}

	lock, err := config.ReadLock(ctx, deps.fs, meta)
	if err != nil {
		return 0, 0, err
	}

	colorize := tui.IsTerminalWriter(cmd.OutOrStdout())

	candidates := updateCandidates(cfg)
	outcomes := processCandidates(ctx, meta, cfg, lock, candidates, deps, colorize)
	updatedCount, failedCount := applyUpdateOutcomes(deps.logger, outcomes, &cfg, lock)

	if updatedCount == 0 && failedCount == 0 {
		deps.logger.Log(messageWithIcon(tui.SuccessIcon(colorize), i18n.T("cmd.update.no_updates")), true)
	}

	if err := config.WriteLock(ctx, deps.fs, meta, lock); err != nil {
		return updatedCount, failedCount, err
	}
	if err := config.WriteConfig(ctx, deps.fs, meta, cfg); err != nil {
		return updatedCount, failedCount, err
	}

	if failedCount > 0 {
		return updatedCount, failedCount, errUpdateFailures
	}

	return updatedCount, failedCount, nil
}

func updateCandidates(cfg models.ModsJSON) []modUpdateCandidate {
	candidates := make([]modUpdateCandidate, 0, len(cfg.Mods))
	for i := range cfg.Mods {
		candidates = append(candidates, modUpdateCandidate{
			ConfigIndex: i,
			Mod:         cfg.Mods[i],
		})
	}
	return candidates
}

func processCandidates(ctx context.Context, meta config.Metadata, cfg models.ModsJSON, lock []models.ModInstall, candidates []modUpdateCandidate, deps updateDeps, colorize bool) []modUpdateOutcome {
	results := make(chan modUpdateOutcome, len(candidates))
	var waitGroup sync.WaitGroup

	for _, candidate := range candidates {
		candidate := candidate
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			results <- processMod(ctx, meta, cfg, lock, candidate, deps, colorize)
		}()
	}

	waitGroup.Wait()
	close(results)

	outcomes := make([]modUpdateOutcome, len(candidates))
	for outcome := range results {
		outcomes[outcome.ConfigIndex] = outcome
	}
	return outcomes
}

func applyUpdateOutcomes(log *logger.Logger, outcomes []modUpdateOutcome, cfg *models.ModsJSON, lock []models.ModInstall) (int, int) {
	updatedCount := 0
	failedCount := 0

	for _, outcome := range outcomes {
		for _, event := range outcome.LogEvents {
			switch event.Kind {
			case logEventKindLog:
				log.Log(event.Message, event.ForceShow)
			case logEventKindError:
				log.Error(event.Message)
			case logEventKindDebug:
				log.Debug(event.Message)
			}
		}

		if strings.TrimSpace(outcome.NewName) != "" && outcome.ConfigIndex >= 0 && outcome.ConfigIndex < len(cfg.Mods) {
			cfg.Mods[outcome.ConfigIndex].Name = outcome.NewName
		}

		if outcome.Error != nil {
			failedCount++
			continue
		}

		if outcome.Updated {
			lock[outcome.LockIndex] = outcome.NewInstall
			updatedCount++
		}
	}

	return updatedCount, failedCount
}

var errUpdateFailures = errors.New("one or more mods failed to update")
var errUnmanagedFiles = errors.New("unmanaged files in mods folder")

//nolint:gocyclo,funlen // Keeps per-mod update flow readable.
func processMod(
	ctx context.Context,
	meta config.Metadata,
	cfg models.ModsJSON,
	lock []models.ModInstall,
	candidate modUpdateCandidate,
	deps updateDeps,
	colorize bool,
) modUpdateOutcome {
	mod := candidate.Mod

	outcome := modUpdateOutcome{
		ConfigIndex: candidate.ConfigIndex,
	}

	lockIndex := lockIndexFor(mod, lock)
	if lockIndex < 0 {
		outcome.LogEvents = append(outcome.LogEvents, logEvent{
			Kind: logEventKindError,
			Message: i18n.T("cmd.update.error.missing_lock_entry", i18n.Tvars{
				Data: &i18n.TData{
					"name": mod.Name,
					"id":   mod.ID,
				},
			}),
		})
		outcome.Error = errUpdateFailures
		return outcome
	}
	outcome.LockIndex = lockIndex

	if isPinned(mod) {
		outcome.NewName = lock[lockIndex].Name
		return outcome
	}

	outcome.LogEvents = append(outcome.LogEvents, logEvent{
		Kind: logEventKindDebug,
		Message: i18n.T("cmd.update.debug.checking", i18n.Tvars{
			Data: &i18n.TData{
				"name":     mod.Name,
				"platform": mod.Type,
			},
		}),
	})

	remote, fetchErr := deps.fetchMod(ctx, mod.Type, mod.ID, platform.FetchOptions{
		AllowedReleaseTypes: effectiveAllowedReleaseTypes(mod, cfg),
		GameVersion:         cfg.GameVersion,
		Loader:              cfg.Loader,
		AllowFallback:       mod.AllowVersionFallback != nil && *mod.AllowVersionFallback,
		FixedVersion:        "",
	}, deps.clients)
	if fetchErr != nil {
		if event, handled := expectedFetchErrorEvent(fetchErr, mod, colorize); handled {
			outcome.LogEvents = append(outcome.LogEvents, event)
			outcome.Error = errUpdateFailures
			return outcome
		}
		outcome.LogEvents = append(outcome.LogEvents, logEvent{Kind: logEventKindError, Message: fetchErr.Error()})
		outcome.Error = errUpdateFailures
		return outcome
	}

	normalizedRemoteFileName, err := modfilename.Normalize(remote.FileName)
	if err != nil {
		outcome.LogEvents = append(outcome.LogEvents, logEvent{
			Kind: logEventKindError,
			Message: i18n.T("cmd.update.error.invalid_filename_remote", i18n.Tvars{
				Data: &i18n.TData{
					"name": mod.Name,
					"file": modfilename.Display(remote.FileName),
				},
			}),
		})
		outcome.Error = errUpdateFailures
		return outcome
	}
	remote.FileName = normalizedRemoteFileName

	installed := lock[lockIndex]
	normalizedInstalledFileName, err := modfilename.Normalize(installed.FileName)
	if err != nil {
		outcome.LogEvents = append(outcome.LogEvents, logEvent{
			Kind: logEventKindError,
			Message: i18n.T("cmd.update.error.invalid_filename_lock", i18n.Tvars{
				Data: &i18n.TData{
					"name": mod.Name,
					"file": modfilename.Display(installed.FileName),
				},
			}),
		})
		outcome.Error = errUpdateFailures
		return outcome
	}
	oldPath := filepath.Join(meta.ModsFolderPath(cfg), normalizedInstalledFileName)
	exists, err := afero.Exists(deps.fs, oldPath)
	if err != nil {
		outcome.LogEvents = append(outcome.LogEvents, logEvent{Kind: logEventKindError, Message: err.Error()})
		outcome.Error = errUpdateFailures
		return outcome
	}
	if !exists {
		outcome.LogEvents = append(outcome.LogEvents, logEvent{
			Kind: logEventKindError,
			Message: i18n.T("cmd.update.error.locked_file_missing", i18n.Tvars{
				Data: &i18n.TData{
					"name": mod.Name,
					"id":   mod.ID,
					"path": oldPath,
				},
			}),
		})
		outcome.Error = errUpdateFailures
		return outcome
	}

	installedDate, err := parseRFC3339(installed.ReleasedOn)
	if err != nil {
		outcome.LogEvents = append(outcome.LogEvents, logEvent{Kind: logEventKindError, Message: err.Error()})
		outcome.Error = errUpdateFailures
		return outcome
	}
	remoteDate, err := parseRFC3339(remote.ReleaseDate)
	if err != nil {
		outcome.LogEvents = append(outcome.LogEvents, logEvent{Kind: logEventKindError, Message: err.Error()})
		outcome.Error = errUpdateFailures
		return outcome
	}

	outcome.NewName = remote.Name

	if !remoteDate.After(installedDate) {
		return outcome
	}

	if strings.TrimSpace(installed.Hash) == "" {
		outcome.LogEvents = append(outcome.LogEvents, logEvent{
			Kind: logEventKindError,
			Message: i18n.T("cmd.update.error.missing_hash_lock", i18n.Tvars{
				Data: &i18n.TData{"name": mod.Name},
			}),
		})
		outcome.Error = errUpdateFailures
		return outcome
	}

	if strings.TrimSpace(remote.Hash) == "" {
		outcome.LogEvents = append(outcome.LogEvents, logEvent{
			Kind: logEventKindError,
			Message: i18n.T("cmd.update.error.missing_hash_remote", i18n.Tvars{
				Data: &i18n.TData{"name": mod.Name},
			}),
		})
		outcome.Error = errUpdateFailures
		return outcome
	}

	if strings.EqualFold(strings.TrimSpace(remote.Hash), strings.TrimSpace(installed.Hash)) {
		return outcome
	}

	outcome.LogEvents = append(outcome.LogEvents, logEvent{
		Kind:      logEventKindLog,
		ForceShow: true,
		Message: i18n.T("cmd.update.has_update", i18n.Tvars{
			Data: &i18n.TData{"name": mod.Name},
		}),
	})

	newPath := filepath.Join(meta.ModsFolderPath(cfg), remote.FileName)

	if err := downloadAndSwap(ctx, deps, oldPath, newPath, meta.ModsFolderPath(cfg), remote.DownloadURL, remote.Hash); err != nil {
		if message, handled := integrityErrorMessage(err, mod.Name); handled {
			outcome.LogEvents = append(outcome.LogEvents, logEvent{Kind: logEventKindError, Message: message})
		} else {
			outcome.LogEvents = append(outcome.LogEvents, logEvent{Kind: logEventKindError, Message: err.Error()})
		}
		outcome.Error = errUpdateFailures
		return outcome
	}

	updatedInstall := models.ModInstall{
		Type:        mod.Type,
		ID:          mod.ID,
		Name:        remote.Name,
		FileName:    remote.FileName,
		ReleasedOn:  remote.ReleaseDate,
		Hash:        remote.Hash,
		DownloadURL: remote.DownloadURL,
	}

	outcome.NewInstall = updatedInstall
	outcome.Updated = true
	return outcome
}

//nolint:gocognit,gocyclo // Sequential swap steps are clearer without extra indirection.
func downloadAndSwap(ctx context.Context, deps updateDeps, oldPath string, newPath string, modsFolder string, downloadURL string, expectedHash string) error {
	if strings.TrimSpace(expectedHash) == "" {
		return modinstall.MissingHashError{FileName: filepath.Base(newPath)}
	}

	resolvedNewPath, err := modpath.ResolveWritablePath(deps.fs, modsFolder, newPath)
	if err != nil {
		return err
	}

	tempPath, err := createTempDownloadPath(deps.fs, resolvedNewPath)
	if err != nil {
		return err
	}

	downloadErr := deps.downloader(ctx, downloadURL, tempPath, downloadClient(deps.clients), &noopSender{}, deps.fs)
	if downloadErr != nil {
		return removeTempFile(deps.fs, tempPath, downloadErr)
	}

	if err := verifyDownloadedHash(deps.fs, tempPath, expectedHash, newPath); err != nil {
		return removeTempFile(deps.fs, tempPath, err)
	}

	if err := replaceExistingFile(deps.fs, deps.logger, tempPath, resolvedNewPath); err != nil {
		return removeTempFile(deps.fs, tempPath, err)
	}

	if filepath.Clean(oldPath) == filepath.Clean(newPath) {
		return nil
	}

	if err := deps.fs.Remove(oldPath); err != nil {
		removeErr := deps.fs.Remove(resolvedNewPath)
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return errors.Join(err, fmt.Errorf("failed to remove new file %s: %w", resolvedNewPath, removeErr))
		}
		return err
	}

	return nil
}

func createTempDownloadPath(fs afero.Fs, resolvedNewPath string) (string, error) {
	tempFile, err := afero.TempFile(fs, filepath.Dir(resolvedNewPath), filepath.Base(resolvedNewPath)+".mmm.*.tmp")
	if err != nil {
		return "", err
	}
	tempPath := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		return "", removeTempFile(fs, tempPath, err)
	}
	return tempPath, nil
}

func removeTempFile(fs afero.Fs, tempPath string, err error) error {
	removeErr := fs.Remove(tempPath)
	if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return errors.Join(err, fmt.Errorf("failed to remove temp file %s: %w", tempPath, removeErr))
	}
	return err
}

func verifyDownloadedHash(fs afero.Fs, tempPath string, expectedHash string, newPath string) error {
	actualHash, err := sha1ForFile(fs, tempPath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(expectedHash), actualHash) {
		return modinstall.HashMismatchError{FileName: filepath.Base(newPath), Expected: expectedHash, Actual: actualHash}
	}
	return nil
}

func replaceExistingFile(fs afero.Fs, log *logger.Logger, sourcePath string, destinationPath string) error {
	exists, err := afero.Exists(fs, destinationPath)
	if err != nil {
		return err
	}

	if !exists {
		return fs.Rename(sourcePath, destinationPath)
	}

	backupPath, err := nextBackupPath(fs, destinationPath)
	if err != nil {
		return err
	}

	if err := fs.Rename(destinationPath, backupPath); err != nil {
		return err
	}

	if err := fs.Rename(sourcePath, destinationPath); err != nil {
		rollbackErr := fs.Rename(backupPath, destinationPath)
		if rollbackErr != nil {
			return errors.Join(err, fmt.Errorf("failed to restore backup %s: %w", backupPath, rollbackErr))
		}
		return err
	}

	if removeErr := fs.Remove(backupPath); removeErr != nil {
		log.Debug(i18n.T("cmd.update.debug.backup_cleanup_failed", i18n.Tvars{
			Data: &i18n.TData{
				"path": backupPath,
				"err":  removeErr.Error(),
			},
		}))
	}
	return nil
}

func nextBackupPath(fs afero.Fs, destinationPath string) (string, error) {
	base := destinationPath + ".mmm.bak"

	backup := base
	for i := 0; i < 100; i++ {
		exists, err := afero.Exists(fs, backup)
		if err != nil {
			return "", err
		}
		if !exists {
			return backup, nil
		}
		backup = fmt.Sprintf("%s.%d", base, i+1)
	}
	return "", errors.New("cannot allocate backup path")
}

func sha1ForFile(fs afero.Fs, path string) (hash string, returnErr error) {
	file, err := fs.Open(path)
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && returnErr == nil {
			returnErr = closeErr
		}
	}()

	hasher := sha1.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func integrityErrorMessage(err error, modName string) (string, bool) {
	var missingHash modinstall.MissingHashError
	if errors.As(err, &missingHash) {
		return i18n.T("cmd.update.error.missing_hash_lock", i18n.Tvars{
			Data: &i18n.TData{"name": modName},
		}), true
	}

	var hashMismatch modinstall.HashMismatchError
	if errors.As(err, &hashMismatch) {
		return i18n.T("cmd.update.error.hash_mismatch", i18n.Tvars{
			Data: &i18n.TData{"name": modName},
		}), true
	}

	var outsideRoot modpath.OutsideRootError
	if errors.As(err, &outsideRoot) {
		return i18n.T("cmd.update.error.symlink_outside_mods", i18n.Tvars{
			Data: &i18n.TData{
				"name": modName,
				"path": outsideRoot.ResolvedPath,
				"root": outsideRoot.Root,
			},
		}), true
	}

	return "", false
}

func isPinned(mod models.Mod) bool {
	if mod.Version == nil {
		return false
	}
	return strings.TrimSpace(*mod.Version) != ""
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

func parseRFC3339(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, errors.New(i18n.T("cmd.update.error.invalid_timestamp", i18n.Tvars{
			Data: &i18n.TData{"value": value},
		}))
	}
	return parsed, nil
}

type noopSender struct{}

func (sender *noopSender) Send(msg tea.Msg) { _ = msg }

func downloadClient(clients platform.Clients) httpclient.Doer {
	if clients.Curseforge != nil {
		return clients.Curseforge
	}
	return clients.Modrinth
}

func expectedFetchErrorEvent(err error, mod models.Mod, colorize bool) (logEvent, bool) {
	var notFound *platform.ModNotFoundError
	if errors.As(err, &notFound) {
		return logEvent{
			Kind:      logEventKindLog,
			ForceShow: true,
			Message: messageWithIcon(tui.ErrorIcon(colorize), i18n.T("cmd.update.error.mod_not_found", i18n.Tvars{
				Data: &i18n.TData{
					"name":     mod.Name,
					"id":       mod.ID,
					"platform": mod.Type,
				},
			})),
		}, true
	}

	var noFile *platform.NoCompatibleFileError
	if errors.As(err, &noFile) {
		return logEvent{
			Kind:      logEventKindLog,
			ForceShow: true,
			Message: messageWithIcon(tui.ErrorIcon(colorize), i18n.T("cmd.update.error.no_file", i18n.Tvars{
				Data: &i18n.TData{
					"name":     mod.Name,
					"id":       mod.ID,
					"platform": mod.Type,
				},
			})),
		}, true
	}

	return logEvent{}, false
}

func messageWithIcon(icon string, message string) string {
	return fmt.Sprintf("%s %s", icon, message)
}
