package remove

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type removeOptions struct {
	ConfigPath string
	Unattended bool
	Quiet      bool
	Debug      bool
	DryRun     bool
	Lookups    []string
}

type initRequest struct {
	configPath string
}

type initRunner func(context.Context, *cobra.Command, initRequest) error

type removeDeps struct {
	fs        afero.Fs
	logger    *logger.Logger
	telemetry func(telemetry.CommandTelemetry)
	runTea    func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error)
	runInit   initRunner
}

type removeRunState struct {
	meta           config.Metadata
	cfg            models.ModsJSON
	lock           []models.ModInstall
	mode           interaction.ExecutionMode
	shouldContinue bool
}

var runInteractiveInit = initCmd.RunInteractiveInit

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <mods...>",
		Short: i18n.T("cmd.remove.short", nil),
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemoveCommand(cmd, args)
		},
	}

	cmd.Flags().BoolP("dry-run", "n", false, i18n.T("cmd.remove.flag.dry_run", nil))

	return cmd
}

func runRemoveCommand(cmd *cobra.Command, args []string) error {
	ctx, span := perf.StartSpan(cmd.Context(), "app.command.remove")

	opts, err := removeOptionsFromFlags(cmd, args)
	if err != nil {
		span.SetAttributes(attribute.Bool("success", false))
		span.End()
		return err
	}

	deps := defaultRemoveDeps(cmd, opts)
	mode := interaction.ResolveExecutionMode(interaction.ExecutionModeInput{
		Unattended: opts.Unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
	})

	removedCount, _, runErr := runRemove(ctx, cmd, opts, deps)
	span.SetAttributes(attribute.Bool("success", runErr == nil))
	span.End()

	recordRemoveTelemetry(deps.telemetry, opts, removedCount, mode, runErr)
	applyRemoveCommandErrorPolicy(cmd, runErr)
	return runErr
}

func applyRemoveCommandErrorPolicy(cmd *cobra.Command, err error) {
	if err == nil {
		return
	}
	if clierrors.IsHandled(err) {
		cmd.SilenceErrors = true
	}
	cmd.SilenceUsage = true
}

func removeOptionsFromFlags(cmd *cobra.Command, args []string) (removeOptions, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return removeOptions{}, err
	}
	unattended, err := cmd.Flags().GetBool("unattended")
	if err != nil {
		return removeOptions{}, err
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return removeOptions{}, err
	}
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return removeOptions{}, err
	}
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return removeOptions{}, err
	}

	return removeOptions{
		ConfigPath: configPath,
		Unattended: unattended,
		Quiet:      quiet,
		Debug:      debug,
		DryRun:     dryRun,
		Lookups:    args,
	}, nil
}

func defaultRemoveDeps(cmd *cobra.Command, options removeOptions) removeDeps {
	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
		Quiet: options.Quiet,
		Debug: options.Debug,
	})

	return removeDeps{
		fs:        common.FS,
		logger:    common.Logger,
		telemetry: telemetry.RecordCommand,
		runTea:    defaultRunTea,
		runInit: func(ctx context.Context, command *cobra.Command, request initRequest) error {
			return runInteractiveInit(ctx, command, initCmd.InteractiveInitDeps{
				FS:              common.FS,
				Output:          common.Output,
				Logger:          common.Logger,
				MinecraftClient: common.MinecraftClient,
				RunTea:          defaultRunTea,
			}, initCmd.InteractiveInitOptions{
				ConfigPath: request.configPath,
				Quiet:      options.Quiet,
				Debug:      options.Debug,
			})
		},
	}
}

func defaultRunTea(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
	program := tea.NewProgram(model, options...)
	return program.Run()
}

var runTeaProgram = defaultRunTea

func recordRemoveTelemetry(telemetryRecorder func(telemetry.CommandTelemetry), opts removeOptions, removedCount int, mode interaction.ExecutionMode, err error) {
	payload := telemetry.CommandTelemetry{
		Command:       "remove",
		Success:       err == nil,
		Error:         err,
		ExitCode:      0,
		Interactive:   mode.IsInteractive(),
		ExecutionMode: mode.String(),
		Arguments: map[string]interface{}{
			"dryRun": opts.DryRun,
			"mods":   opts.Lookups,
		},
	}
	if err != nil {
		payload.ExitCode = 1
	} else {
		payload.Extra = map[string]interface{}{
			"numberOfMods": removedCount,
		}
	}
	telemetryRecorder(payload)
}

func runRemove(ctx context.Context, cmd *cobra.Command, opts removeOptions, deps removeDeps) (int, bool, error) {
	runState, err := prepareRemoveRunState(ctx, cmd, opts, deps)
	if err != nil {
		return 0, runState.mode.IsInteractive(), err
	}
	if !runState.shouldContinue {
		return 0, runState.mode.IsInteractive(), nil
	}

	matches, err := resolveMatchesForRemove(opts.Lookups, runState.cfg, runState.lock)
	if err != nil {
		return 0, runState.mode.IsInteractive(), err
	}

	return runRemoveWithMatches(ctx, cmd, opts, deps, runState, matches)
}

func runRemoveWithMatches(
	ctx context.Context,
	cmd *cobra.Command,
	opts removeOptions,
	deps removeDeps,
	runState removeRunState,
	matches []removeMatch,
) (int, bool, error) {
	interactive := runState.mode.IsInteractive()
	if len(matches) == 0 {
		if err := handleRemoveNoMatches(cmd, deps, opts); err != nil {
			return 0, interactive, err
		}
		return 0, interactive, nil
	}

	items := buildRemoveItems(matches)
	indexByKey := removeIndexByKey(items)
	colorMode := colorModeForWriter(cmd)

	if opts.DryRun {
		if err := runRemoveDryRun(cmd, deps, opts, colorMode, items); err != nil {
			return 0, interactive, err
		}
		return 0, interactive, nil
	}

	execInput := removeExecutionInput{
		meta:  runState.meta,
		cfg:   runState.cfg,
		lock:  runState.lock,
		items: items,
		deps:  deps,
	}

	if opts.Quiet {
		count, err := runRemoveQuiet(ctx, cmd, execInput, colorMode)
		return count, interactive, err
	}

	if shouldUseTranscript(cmd) {
		count, err := runRemoveTranscriptOutput(ctx, cmd, execInput, colorMode, indexByKey)
		return count, interactive, err
	}

	count, err := runRemoveInteractiveOutput(ctx, cmd, execInput, colorMode, indexByKey)
	return count, interactive, err
}

func handleRemoveNoMatches(cmd *cobra.Command, deps removeDeps, opts removeOptions) error {
	if opts.Quiet {
		return nil
	}
	if outputErr := runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{renderRemoveNoMatchesLine()}); outputErr != nil {
		return outputErr
	}
	return nil
}

func runRemoveDryRun(cmd *cobra.Command, deps removeDeps, opts removeOptions, colorMode view.ColorMode, items []removeItem) error {
	if opts.Quiet {
		return nil
	}
	lines := []string{renderRemoveDryRunSection(colorMode, items)}
	if outputErr := runOutputLines(cmd, deps, cmd.OutOrStdout(), lines); outputErr != nil {
		return outputErr
	}
	return nil
}

func runRemoveQuiet(ctx context.Context, cmd *cobra.Command, execInput removeExecutionInput, colorMode view.ColorMode) (int, error) {
	outcome := runRemoveExecution(ctx, execInput, removeExecSender{})
	if outcome.err == nil {
		return countRemoveSuccess(outcome.items), nil
	}
	lines := renderRemoveQuietFailure(colorMode, outcome)
	if outputErr := runOutputLines(cmd, execInput.deps, cmd.OutOrStdout(), lines); outputErr != nil {
		return 0, outputErr
	}
	return countRemoveSuccess(outcome.items), clierrors.MarkHandled(outcome.err)
}

func runRemoveTranscriptOutput(
	ctx context.Context,
	cmd *cobra.Command,
	execInput removeExecutionInput,
	colorMode view.ColorMode,
	indexByKey map[string]int,
) (int, error) {
	outcome, transcriptErr := runRemoveTranscript(ctx, cmd, execInput, colorMode, indexByKey)
	if transcriptErr != nil {
		return countRemoveSuccess(outcome.items), transcriptErr
	}
	if outcome.err != nil {
		return countRemoveSuccess(outcome.items), clierrors.MarkHandled(outcome.err)
	}
	return countRemoveSuccess(outcome.items), nil
}

func runRemoveInteractiveOutput(
	ctx context.Context,
	cmd *cobra.Command,
	execInput removeExecutionInput,
	colorMode view.ColorMode,
	indexByKey map[string]int,
) (int, error) {
	outcome, interactiveErr := runRemoveInteractive(ctx, cmd, execInput, colorMode, indexByKey)
	if interactiveErr != nil {
		return countRemoveSuccess(outcome.items), interactiveErr
	}
	if outcome.err != nil {
		return countRemoveSuccess(outcome.items), clierrors.MarkHandled(outcome.err)
	}
	return countRemoveSuccess(outcome.items), nil
}

func shouldUseTranscript(cmd *cobra.Command) bool {
	if cmd == nil {
		return true
	}
	return !view.SupportsPrompting(cmd.InOrStdin(), cmd.OutOrStdout())
}

type removeExecutionInput struct {
	meta  config.Metadata
	cfg   models.ModsJSON
	lock  []models.ModInstall
	items []removeItem
	deps  removeDeps
}

func runRemoveInteractive(
	ctx context.Context,
	cmd *cobra.Command,
	execInput removeExecutionInput,
	colorMode view.ColorMode,
	indexByKey map[string]int,
) (removeExecutionOutcome, error) {
	model := newRemoveModel(ctx, colorMode, execInput.items, indexByKey, func(ctx context.Context, sender removeExecSender) removeExecutionOutcome {
		return runRemoveExecution(ctx, execInput, sender)
	})

	result, err := runRemoveProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
	if err != nil {
		return removeExecutionOutcome{}, err
	}
	return removeOutcomeFromModel(result)
}

func runRemoveTranscript(
	ctx context.Context,
	cmd *cobra.Command,
	execInput removeExecutionInput,
	colorMode view.ColorMode,
	indexByKey map[string]int,
) (removeExecutionOutcome, error) {
	model := newRemoveTranscriptModel(ctx, colorMode, execInput.items, indexByKey, cmd.OutOrStdout(), func(ctx context.Context, sender removeExecSender) removeExecutionOutcome {
		return runRemoveExecution(ctx, execInput, sender)
	})

	_, err := runRemoveTranscriptProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
	if err != nil {
		return removeExecutionOutcome{}, err
	}
	return model.outcome, nil
}

func countRemoveSuccess(items []removeItem) int {
	count := 0
	for _, item := range items {
		if item.Status == removeItemSuccess {
			count++
		}
	}
	return count
}

func renderRemoveQuietFailure(colorMode view.ColorMode, outcome removeExecutionOutcome) []string {
	switch outcome.errType {
	case removeExecutionErrorDelete:
		return renderRemoveQuietDeleteFailure(colorMode, outcome.items)
	case removeExecutionErrorWriteConfig, removeExecutionErrorWriteLock, removeExecutionErrorUnknown:
		if outcome.err != nil {
			return []string{renderRemoveFailureLine(colorMode, outcome.err)}
		}
		return nil
	default:
		return nil
	}
}

func renderRemoveQuietDeleteFailure(colorMode view.ColorMode, items []removeItem) []string {
	failed := filterRemoveItems(items, removeItemFailed)
	if len(failed) == 0 {
		return []string{renderRemoveFailureSummary(colorMode)}
	}
	return []string{strings.Join(renderRemoveItems(colorMode, failed, ""), "\n"), renderRemoveFailureSummary(colorMode)}
}

func filterRemoveItems(items []removeItem, status removeItemStatus) []removeItem {
	filtered := make([]removeItem, 0, len(items))
	for _, item := range items {
		if item.Status == status {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func prepareRemoveRunState(ctx context.Context, cmd *cobra.Command, opts removeOptions, deps removeDeps) (removeRunState, error) {
	mode := interaction.ResolveExecutionMode(interaction.ExecutionModeInput{
		Unattended: opts.Unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
	})

	runState := removeRunState{
		meta:           config.NewMetadata(opts.ConfigPath),
		mode:           mode,
		shouldContinue: true,
	}

	state, err := loadRemoveConfigState(ctx, deps, runState.meta, opts.DryRun)
	if err == nil {
		runState.cfg = state.cfg
		runState.lock = state.lock
		return runState, nil
	}

	var notFound *config.ConfigFileNotFoundException
	if !errors.As(err, &notFound) {
		return runState, handleRemoveFailure(cmd, deps, err)
	}

	return handleMissingRemoveConfig(ctx, cmd, opts, deps, runState)
}

type removeConfigState struct {
	cfg  models.ModsJSON
	lock []models.ModInstall
}

func loadRemoveConfigState(ctx context.Context, deps removeDeps, meta config.Metadata, dryRun bool) (removeConfigState, error) {
	cfg, lock, err := readRemoveConfig(ctx, deps, meta, dryRun)
	if err != nil {
		return removeConfigState{}, err
	}
	return removeConfigState{cfg: cfg, lock: lock}, nil
}

func handleMissingRemoveConfig(
	ctx context.Context,
	cmd *cobra.Command,
	opts removeOptions,
	deps removeDeps,
	runState removeRunState,
) (removeRunState, error) {
	promptErr := configMissingPromptError(opts, cmd, runState.meta)
	if promptErr != nil {
		if outputErr := writeConfigMissingOutput(cmd, deps, runState.meta); outputErr != nil {
			return runState, outputErr
		}
		return runState, clierrors.MarkHandled(promptErr)
	}

	confirmed, canceled, promptErr := runConfigInitPrompt(cmd, deps, runState.meta)
	if promptErr != nil {
		return runState, promptErr
	}
	if canceled || !confirmed {
		runState.shouldContinue = false
		return runState, nil
	}

	if deps.runInit == nil {
		return runState, errors.New("missing init runner")
	}
	if runErr := deps.runInit(ctx, cmd, initRequest{configPath: runState.meta.ConfigPath}); runErr != nil {
		if errors.Is(runErr, initCmd.ErrInitCanceled) {
			runState.shouldContinue = false
			return runState, nil
		}
		return runState, runErr
	}

	state, err := loadRemoveConfigState(ctx, deps, runState.meta, opts.DryRun)
	if err != nil {
		return runState, handleRemoveFailure(cmd, deps, err)
	}
	runState.cfg = state.cfg
	runState.lock = state.lock
	return runState, nil
}

func configMissingPromptError(opts removeOptions, cmd *cobra.Command, meta config.Metadata) error {
	return interaction.CheckConfigInitGate(meta, interaction.ConfigInitGate{
		Unattended: opts.Unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
		UnattendedError: func(meta config.Metadata) error {
			return errors.New(i18n.T("cmd.remove.error.config_missing", &i18n.Tvars{
				Data: &i18n.TData{"configPath": meta.ConfigPath},
			}))
		},
		NoTTYError: func(meta config.Metadata) error {
			return errors.New(i18n.T("cmd.remove.error.config_missing", &i18n.Tvars{
				Data: &i18n.TData{"configPath": meta.ConfigPath},
			}))
		},
	})
}

func writeConfigMissingOutput(cmd *cobra.Command, deps removeDeps, meta config.Metadata) error {
	colorMode := colorModeForWriter(cmd)
	headline := messageWithIcon(view.FinalErrorIcon(colorMode), i18n.T("cmd.remove.error.config_missing", &i18n.Tvars{
		Data: &i18n.TData{"configPath": meta.ConfigPath},
	}))
	if colorMode.Enabled() {
		headline = view.ErrorStyle.Render(headline)
	}

	hint := i18n.T("cmd.remove.error.config_missing_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}

	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{headline, hint})
}

func handleRemoveFailure(cmd *cobra.Command, deps removeDeps, err error) error {
	if outputErr := runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{renderRemoveFailureLine(colorModeForWriter(cmd), err)}); outputErr != nil {
		return outputErr
	}
	return clierrors.MarkHandled(err)
}

func readRemoveConfig(ctx context.Context, deps removeDeps, meta config.Metadata, dryRun bool) (models.ModsJSON, []models.ModInstall, error) {
	cfg, err := config.ReadConfig(ctx, deps.fs, meta)
	if err != nil {
		return models.ModsJSON{}, nil, err
	}
	lock, err := readLockForRemove(ctx, deps.fs, meta, removeLockOptions{dryRun: dryRun})
	if err != nil {
		return models.ModsJSON{}, nil, err
	}
	return cfg, lock, nil
}

func runRemoveExecution(ctx context.Context, input removeExecutionInput, sender removeExecSender) removeExecutionOutcome {
	results := make([]removeItem, len(input.items))
	copy(results, input.items)

	var mutex sync.Mutex
	var waitGroup sync.WaitGroup

	for index, item := range input.items {
		waitGroup.Add(1)
		go func(index int, item removeItem) {
			defer waitGroup.Done()
			result := removeSingleItem(input, item)
			mutex.Lock()
			results[index] = result
			mutex.Unlock()
			if result.Status == removeItemSuccess {
				sender.Send(removeItemSuccessMsg{key: removeItemKey(item.Mod)})
				return
			}
			if result.Status == removeItemFailed {
				sender.Send(removeItemFailureMsg{key: removeItemKey(item.Mod), reason: result.FailureReason})
			}
		}(index, item)
	}

	waitGroup.Wait()

	writeOutcome := applyRemoveUpdates(ctx, input, results)
	finalItems := results

	outcome := removeExecutionOutcome{items: finalItems}
	if writeOutcome.err != nil {
		outcome.err = writeOutcome.err
		outcome.errType = writeOutcome.errType
		return outcome
	}

	if hasRemoveFailures(finalItems) {
		outcome.err = errors.New("remove incomplete")
		outcome.errType = removeExecutionErrorDelete
		return outcome
	}

	outcome.errType = removeExecutionErrorNone
	return outcome
}

type removeWriteOutcome struct {
	err     error
	errType removeExecutionErrorType
}

func applyRemoveUpdates(ctx context.Context, input removeExecutionInput, results []removeItem) removeWriteOutcome {
	updatedConfig := input.cfg
	updatedLock := input.lock
	removedAny := false

	for _, item := range results {
		if item.Status != removeItemSuccess {
			continue
		}
		updatedConfig = removeConfigEntry(updatedConfig, item.Mod)
		updatedLock = removeLockEntry(updatedLock, item.Mod)
		removedAny = true
	}

	if !removedAny {
		return removeWriteOutcome{}
	}

	originalFiles, originalErr := readOriginalRemoveFiles(input.deps.fs, input.meta)
	if originalErr != nil {
		return removeWriteOutcome{err: originalErr, errType: removeExecutionErrorUnknown}
	}

	if err := config.WriteConfig(ctx, input.deps.fs, input.meta, updatedConfig); err != nil {
		restoreErr := restoreOriginalConfig(input.deps.fs, input.meta, originalFiles)
		if restoreErr != nil {
			writeErr := fmt.Errorf("failed to write config: %w", err)
			restoreWriteErr := fmt.Errorf("restore failed: %w", restoreErr)
			return removeWriteOutcome{
				err:     errors.Join(writeErr, restoreWriteErr),
				errType: removeExecutionErrorWriteConfig,
			}
		}
		return removeWriteOutcome{err: err, errType: removeExecutionErrorWriteConfig}
	}
	if err := config.WriteLock(ctx, input.deps.fs, input.meta, updatedLock); err != nil {
		restoreErr := restoreOriginalFiles(input.deps.fs, input.meta, originalFiles)
		if restoreErr != nil {
			writeErr := fmt.Errorf("failed to write lock: %w", err)
			restoreWriteErr := fmt.Errorf("restore failed: %w", restoreErr)
			return removeWriteOutcome{
				err:     errors.Join(writeErr, restoreWriteErr),
				errType: removeExecutionErrorWriteLock,
			}
		}
		return removeWriteOutcome{err: err, errType: removeExecutionErrorWriteLock}
	}
	return removeWriteOutcome{}
}

type removeOriginalFiles struct {
	configExists bool
	configBytes  []byte
	lockExists   bool
	lockBytes    []byte
}

func readOriginalRemoveFiles(fs afero.Fs, meta config.Metadata) (removeOriginalFiles, error) {
	configBytes, configExists, configErr := readExistingFile(fs, meta.ConfigPath)
	if configErr != nil {
		return removeOriginalFiles{}, configErr
	}
	lockBytes, lockExists, lockErr := readExistingFile(fs, meta.LockPath())
	if lockErr != nil {
		return removeOriginalFiles{}, lockErr
	}
	return removeOriginalFiles{
		configExists: configExists,
		configBytes:  configBytes,
		lockExists:   lockExists,
		lockBytes:    lockBytes,
	}, nil
}

func readExistingFile(fs afero.Fs, path string) ([]byte, bool, error) {
	exists, err := afero.Exists(fs, path)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

func restoreOriginalConfig(fs afero.Fs, meta config.Metadata, originalFiles removeOriginalFiles) error {
	if originalFiles.configExists {
		return restoreExistingFile(fs, meta.ConfigPath, originalFiles.configBytes)
	}
	return removeFileIfExists(fs, meta.ConfigPath)
}

func restoreOriginalFiles(fs afero.Fs, meta config.Metadata, originalFiles removeOriginalFiles) error {
	if originalFiles.configExists {
		if err := restoreExistingFile(fs, meta.ConfigPath, originalFiles.configBytes); err != nil {
			return err
		}
	} else if err := removeFileIfExists(fs, meta.ConfigPath); err != nil {
		return err
	}

	if originalFiles.lockExists {
		return restoreExistingFile(fs, meta.LockPath(), originalFiles.lockBytes)
	}
	return removeFileIfExists(fs, meta.LockPath())
}

func restoreExistingFile(fs afero.Fs, path string, data []byte) error {
	return afero.WriteFile(fs, path, data, 0644)
}

func removeFileIfExists(fs afero.Fs, path string) error {
	if err := fs.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func removeSingleItem(input removeExecutionInput, item removeItem) removeItem {
	item.Status = removeItemPending
	item.FailureReason = ""

	if !item.HasLockEntry {
		item.Status = removeItemSuccess
		return item
	}

	normalizedFileName, err := modfilename.Normalize(item.LockFileName)
	if err != nil {
		item.Status = removeItemFailed
		item.FailureReason = i18n.T("cmd.remove.error.invalid_filename_reason", &i18n.Tvars{
			Data: &i18n.TData{"file": modfilename.Display(item.LockFileName)},
		})
		return item
	}

	installedPath := filepath.Join(input.meta.ModsFolderPath(input.cfg), normalizedFileName)
	if err := removeFileForce(input.deps.fs, installedPath); err != nil {
		item.Status = removeItemFailed
		item.FailureReason = err.Error()
		return item
	}

	item.Status = removeItemSuccess
	return item
}

func hasRemoveFailures(items []removeItem) bool {
	for _, item := range items {
		if item.Status == removeItemFailed {
			return true
		}
	}
	return false
}

type removeMatch struct {
	mod          models.Mod
	hasLockEntry bool
	lockFileName string
}

func buildRemoveItems(matches []removeMatch) []removeItem {
	items := make([]removeItem, 0, len(matches))
	for _, match := range matches {
		items = append(items, removeItem{
			Mod:          match.mod,
			Status:       removeItemPending,
			HasLockEntry: match.hasLockEntry,
			LockFileName: match.lockFileName,
		})
	}

	sort.SliceStable(items, func(left int, right int) bool {
		leftName := strings.ToLower(items[left].Mod.Name)
		rightName := strings.ToLower(items[right].Mod.Name)
		if leftName == rightName {
			leftPlatform := string(items[left].Mod.Type)
			rightPlatform := string(items[right].Mod.Type)
			if leftPlatform == rightPlatform {
				return items[left].Mod.ID < items[right].Mod.ID
			}
			return leftPlatform < rightPlatform
		}
		return leftName < rightName
	})

	return items
}

func removeIndexByKey(items []removeItem) map[string]int {
	indexByKey := make(map[string]int, len(items))
	for index, item := range items {
		indexByKey[removeItemKey(item.Mod)] = index
	}
	return indexByKey
}

func removeItemKey(mod models.Mod) string {
	return string(mod.Type) + ":" + mod.ID
}

func removeLockEntry(lock []models.ModInstall, mod models.Mod) []models.ModInstall {
	index := models.LockIndexForMod(mod, lock)
	if index < 0 {
		return lock
	}
	return append(lock[:index], lock[index+1:]...)
}

func removeConfigEntry(cfg models.ModsJSON, mod models.Mod) models.ModsJSON {
	index := configIndexFor(mod, cfg.Mods)
	if index < 0 {
		return cfg
	}
	cfg.Mods = append(cfg.Mods[:index], cfg.Mods[index+1:]...)
	return cfg
}

func configIndexFor(mod models.Mod, mods []models.Mod) int {
	for index := range mods {
		if mods[index].Type == mod.Type && mods[index].ID == mod.ID {
			return index
		}
	}
	return -1
}

func readLockForRemove(ctx context.Context, fs afero.Fs, meta config.Metadata, options removeLockOptions) ([]models.ModInstall, error) {
	if !options.dryRun {
		return config.EnsureLock(ctx, fs, meta)
	}

	lockPath := meta.LockPath()
	exists, err := afero.Exists(fs, lockPath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return []models.ModInstall{}, nil
	}
	return config.ReadLock(ctx, fs, meta)
}

type removeLockOptions struct {
	dryRun bool
}

func resolveMatchesForRemove(lookups []string, cfg models.ModsJSON, lock []models.ModInstall) ([]removeMatch, error) {
	matches := make([]removeMatch, 0)
	seen := make(map[string]bool)
	lockByKey := make(map[string]models.ModInstall, len(lock))
	for _, entry := range lock {
		lockByKey[string(entry.Type)+":"+entry.ID] = entry
	}

	for _, lookup := range lookups {
		pattern := strings.ToLower(strings.TrimSpace(lookup))
		if pattern == "" {
			continue
		}

		if _, err := filepath.Match(pattern, ""); err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", lookup, err)
		}

		lockMatches := matchLockEntries(pattern, lock)
		for _, entry := range lockMatches {
			mod := modFromLockEntry(entry, cfg)
			key := removeItemKey(mod)
			if seen[key] {
				continue
			}
			seen[key] = true
			matches = append(matches, removeMatch{
				mod:          mod,
				hasLockEntry: true,
				lockFileName: entry.FileName,
			})
		}

		for _, mod := range matchConfigMods(pattern, cfg.Mods) {
			key := removeItemKey(mod)
			if seen[key] {
				continue
			}
			if _, ok := lockByKey[key]; ok {
				continue
			}
			seen[key] = true
			matches = append(matches, removeMatch{mod: mod})
		}
	}

	return matches, nil
}

func matchLockEntries(pattern string, lock []models.ModInstall) []models.ModInstall {
	matches := make([]models.ModInstall, 0)
	for _, entry := range lock {
		if globMatches(pattern, strings.ToLower(entry.ID)) || globMatches(pattern, strings.ToLower(entry.Name)) {
			matches = append(matches, entry)
		}
	}
	return matches
}

func matchConfigMods(pattern string, mods []models.Mod) []models.Mod {
	matches := make([]models.Mod, 0)
	for _, mod := range mods {
		if globMatches(pattern, strings.ToLower(mod.ID)) || globMatches(pattern, strings.ToLower(mod.Name)) {
			matches = append(matches, mod)
		}
	}
	return matches
}

func modFromLockEntry(entry models.ModInstall, cfg models.ModsJSON) models.Mod {
	mod := models.Mod{
		Type: entry.Type,
		ID:   entry.ID,
		Name: entry.Name,
	}
	if mod.Name != "" {
		return mod
	}
	for _, candidate := range cfg.Mods {
		if candidate.Type == entry.Type && candidate.ID == entry.ID {
			mod.Name = candidate.Name
			break
		}
	}
	return mod
}

func globMatches(pattern string, value string) bool {
	ok, err := filepath.Match(pattern, value)
	if err != nil {
		return false
	}
	return ok
}

func removeFileForce(fs afero.Fs, path string) error {
	if err := fs.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}

func colorModeForWriter(cmd *cobra.Command) view.ColorMode {
	if cmd == nil {
		return view.ColorDisabled
	}
	if !view.SupportsColor(cmd.OutOrStdout()) || !view.SupportsControlSequences(cmd.OutOrStdout()) {
		return view.ColorDisabled
	}
	return view.ColorEnabled
}
