package update

import (
	"context"
	"errors"
	"io"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/cmd/mmm/install"
	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/locksync"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var errUpdateFailures = errors.New("one or more mods failed to update")

type updateExecutionInput struct {
	meta       config.Metadata
	cfg        *models.ModsJSON
	lock       []models.ModInstall
	deps       updateDeps
	items      []updateItem
	indexByKey map[int]int
	colorMode  view.ColorMode
}

type updateConfigState struct {
	cfg            models.ModsJSON
	shouldContinue bool
}

func runUpdate(ctx context.Context, cmd *cobra.Command, opts updateOptions, deps updateDeps) (updateCounts, error) {
	meta := config.NewMetadata(opts.ConfigPath)
	configState, err := ensureUpdateConfig(ctx, cmd, opts, deps, meta)
	if err != nil {
		return updateCounts{}, err
	}
	if !configState.shouldContinue {
		return updateCounts{}, nil
	}

	mode := interaction.ResolveExecutionMode(interaction.ExecutionModeInput{
		Unattended: opts.Unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
	})
	lockSyncContext, err := loadUpdateContextWithConfig(ctx, cmd, opts, deps, mode, meta, configState.cfg, updateLockSyncPhase)
	if err != nil {
		return updateCounts{}, err
	}
	if !lockSyncContext.shouldContinue {
		return updateCounts{}, nil
	}
	if handled, noModsErr := handleUpdateNoModsConfigured(cmd, opts, deps, meta, lockSyncContext); handled {
		return updateCounts{}, noModsErr
	}

	execState, err := prepareUpdateExecution(ctx, cmd, opts, deps, mode)
	if err != nil {
		return updateCounts{}, err
	}

	if opts.Quiet {
		outcome := runUpdateExecution(ctx, execState.input, updateExecSender{})
		return handleQuietUpdateResult(cmd, outcome)
	}

	outcome, err := runUpdateWithMode(ctx, cmd, execState.input, execState.mode)
	if err != nil {
		return updateCounts{}, err
	}
	return handleUpdateOutcome(outcome)
}

func handleUpdateNoModsConfigured(
	cmd *cobra.Command,
	opts updateOptions,
	deps updateDeps,
	meta config.Metadata,
	lockSyncContext updateContext,
) (bool, error) {
	if len(lockSyncContext.cfg.Mods) != 0 {
		return false, nil
	}
	if unmanagedErr := requireNoUnmanagedForUpdateNoMods(cmd, deps, meta, lockSyncContext); unmanagedErr != nil {
		return true, unmanagedErr
	}
	if opts.Quiet {
		return true, nil
	}
	if err := reportNoModsConfigured(cmd); err != nil {
		return true, err
	}
	return true, nil
}

func requireNoUnmanagedForUpdateNoMods(cmd *cobra.Command, deps updateDeps, meta config.Metadata, lockSyncContext updateContext) error {
	_, unmanagedErr := interaction.RequireNoUnmanagedFiles(interaction.UnmanagedGateInput{
		Fs:                     deps.fs,
		Meta:                   meta,
		Config:                 lockSyncContext.cfg,
		Lock:                   lockSyncContext.lock,
		ColorMode:              lockSyncContext.colorMode,
		Write:                  func(lines []string) error { return runOutputLines(cmd, cmd.OutOrStdout(), lines) },
		AllowMissingModsFolder: true,
	})
	if unmanagedErr == nil {
		return nil
	}
	if errors.Is(unmanagedErr, interaction.ErrUnmanagedFiles) {
		return clierrors.MarkHandled(unmanagedErr)
	}
	line := renderFinalErrorLine(lockSyncContext.colorMode, unmanagedErr.Error())
	if outputErr := runOutputLines(cmd, cmd.OutOrStdout(), []string{line}); outputErr != nil {
		return outputErr
	}
	return clierrors.MarkHandled(unmanagedErr)
}

func reportNoModsConfigured(cmd *cobra.Command) error {
	if outputErr := runOutputLines(cmd, cmd.OutOrStdout(), []string{i18n.T("cmd.list.empty", nil)}); outputErr != nil {
		return outputErr
	}
	return nil
}

type updateExecutionState struct {
	input          updateExecutionInput
	mode           interaction.ExecutionMode
	shouldContinue bool
}

func prepareUpdateExecution(ctx context.Context, cmd *cobra.Command, opts updateOptions, deps updateDeps, mode interaction.ExecutionMode) (updateExecutionState, error) {
	if installErr := ensureInstallForUpdate(ctx, cmd, opts, deps, mode); installErr != nil {
		return updateExecutionState{mode: mode, shouldContinue: true}, installErr
	}

	loadedContext, err := loadUpdateContext(ctx, cmd, opts, deps, mode, updateReadPhase)
	if err != nil {
		return updateExecutionState{mode: interaction.ExecutionModeNonTTY, shouldContinue: true}, err
	}

	items, indexByKey := buildUpdateItems(loadedContext.cfg, loadedContext.lock)
	executionInput := updateExecutionInput{
		meta:       loadedContext.meta,
		cfg:        &loadedContext.cfg,
		lock:       loadedContext.lock,
		deps:       deps,
		items:      items,
		indexByKey: indexByKey,
		colorMode:  loadedContext.colorMode,
	}

	return updateExecutionState{
		input:          executionInput,
		mode:           mode,
		shouldContinue: true,
	}, nil
}

func ensureUpdateConfig(ctx context.Context, cmd *cobra.Command, opts updateOptions, deps updateDeps, meta config.Metadata) (updateConfigState, error) {
	cfg, err := config.ReadConfig(ctx, deps.fs, meta)
	if err == nil {
		return updateConfigState{cfg: cfg, shouldContinue: true}, nil
	}

	var notFound *config.ConfigFileNotFoundException
	if !errors.As(err, &notFound) {
		return updateConfigState{}, err
	}

	if promptErr := configMissingPromptError(opts, cmd, meta); promptErr != nil {
		if outputErr := writeConfigMissingOutput(cmd, meta); outputErr != nil {
			return updateConfigState{}, outputErr
		}
		return updateConfigState{}, clierrors.MarkHandled(promptErr)
	}

	confirmed, canceled, err := runConfigInitPrompt(cmd, deps, meta)
	if err != nil {
		return updateConfigState{}, err
	}
	if canceled || !confirmed {
		return updateConfigState{shouldContinue: false}, nil
	}
	if deps.runInit == nil {
		return updateConfigState{}, errors.New("missing init runner")
	}
	if runErr := deps.runInit(ctx, cmd, initRequest{ConfigPath: meta.ConfigPath}); runErr != nil {
		if errors.Is(runErr, initCmd.ErrInitCanceled) {
			return updateConfigState{shouldContinue: false}, nil
		}
		return updateConfigState{}, runErr
	}

	cfg, err = config.ReadConfig(ctx, deps.fs, meta)
	if err != nil {
		return updateConfigState{}, err
	}
	return updateConfigState{cfg: cfg, shouldContinue: true}, nil
}

func runUpdateWithMode(ctx context.Context, cmd *cobra.Command, input updateExecutionInput, mode interaction.ExecutionMode) (updateExecutionOutcome, error) {
	if mode == interaction.ExecutionModeNonTTY {
		return runUpdateTranscript(ctx, cmd, input)
	}
	return runUpdateInteractive(ctx, cmd, input)
}

func ensureInstallForUpdate(ctx context.Context, cmd *cobra.Command, opts updateOptions, deps updateDeps, mode interaction.ExecutionMode) error {
	installCtx := install.WithRunningFooter(ctx, install.RunningFooter{Render: updateInstallRunningFooter})
	installView := ""
	if mode != interaction.ExecutionModeNonTTY {
		installCtx = install.WithInstallViewObserver(installCtx, func(view string) {
			installView = view
		})
		installCtx = install.WithInstallProgramOptions(installCtx, tea.WithAltScreen())
	}
	var originalOut io.Writer
	var headerWriter *updateInstallHeaderWriter
	if mode == interaction.ExecutionModeNonTTY {
		originalOut = cmd.OutOrStdout()
		headerWriter = &updateInstallHeaderWriter{
			out:    originalOut,
			header: i18n.T("cmd.update.header.installing_potentially_missing", nil),
		}
		cmd.SetOut(headerWriter)
	}
	_, err := deps.install(installCtx, cmd, install.RunOptions{
		ConfigPath:   opts.ConfigPath,
		Unattended:   opts.Unattended,
		Quiet:        opts.Quiet,
		Debug:        opts.Debug,
		LockSync:     opts.LockSync,
		SkipLockSync: true,
	})
	if mode == interaction.ExecutionModeNonTTY {
		cmd.SetOut(originalOut)
		if headerWriter.writeErr != nil {
			return headerWriter.writeErr
		}
	}
	if err != nil {
		if errors.Is(err, interaction.ErrUnmanagedFiles) {
			return err
		}
		return handleUpdateInstallFailure(cmd, err, installView, mode)
	}
	return nil
}

type updateInstallHeaderWriter struct {
	out         io.Writer
	header      string
	wroteHeader bool
	writeErr    error
}

func (writer *updateInstallHeaderWriter) Write(value []byte) (int, error) {
	if writer.writeErr != nil {
		return 0, writer.writeErr
	}
	if !writer.wroteHeader {
		if _, err := io.WriteString(writer.out, writer.header+"\n\n"); err != nil {
			writer.writeErr = err
			return 0, err
		}
		writer.wroteHeader = true
	}
	written, err := writer.out.Write(value)
	if err != nil {
		writer.writeErr = err
	}
	return written, err
}

func configMissingPromptError(opts updateOptions, cmd *cobra.Command, meta config.Metadata) error {
	return interaction.CheckConfigInitGate(meta, interaction.ConfigInitGate{
		Unattended: opts.Unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
		UnattendedError: func(meta config.Metadata) error {
			return errors.New(i18n.T("cmd.config.error.missing", &i18n.Tvars{
				Data: &i18n.TData{"configPath": meta.ConfigPath},
			}))
		},
		NoTTYError: func(meta config.Metadata) error {
			return errors.New(i18n.T("cmd.config.error.missing", &i18n.Tvars{
				Data: &i18n.TData{"configPath": meta.ConfigPath},
			}))
		},
	})
}

func writeConfigMissingOutput(cmd *cobra.Command, meta config.Metadata) error {
	colorMode := colorModeForOutput(cmd.OutOrStdout())
	headline := renderFinalErrorLine(colorMode, i18n.T("cmd.config.error.missing", &i18n.Tvars{
		Data: &i18n.TData{"configPath": meta.ConfigPath},
	}))
	hint := i18n.T("cmd.config.error.missing_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}
	return runOutputLines(cmd, cmd.OutOrStdout(), []string{headline, hint})
}

func handleUpdateInstallFailure(cmd *cobra.Command, err error, installView string, mode interaction.ExecutionMode) error {
	if isContextCancellation(err) {
		return clierrors.MarkHandled(err)
	}

	installView = strings.TrimRight(installView, "\n")
	sections := []string{renderUpdateInstallFailureView(colorModeForOutput(cmd.OutOrStdout()))}
	if mode != interaction.ExecutionModeNonTTY && strings.TrimSpace(installView) != "" {
		sections = append([]string{installView}, sections...)
	}
	if outputErr := runOutputLines(cmd, cmd.OutOrStdout(), sections); outputErr != nil {
		return outputErr
	}
	return clierrors.MarkHandled(err)
}

func loadUpdateContext(ctx context.Context, cmd *cobra.Command, opts updateOptions, deps updateDeps, mode interaction.ExecutionMode, phase updateLoadPhase) (updateContext, error) {
	meta := config.NewMetadata(opts.ConfigPath)

	cfg, err := config.ReadConfig(ctx, deps.fs, meta)
	if err != nil {
		return updateContext{}, err
	}

	return loadUpdateContextWithConfig(ctx, cmd, opts, deps, mode, meta, cfg, phase)
}

type updateLoadPhase int

const (
	updateLockSyncPhase updateLoadPhase = iota
	updateReadPhase
)

func loadUpdateContextWithConfig(
	ctx context.Context,
	cmd *cobra.Command,
	opts updateOptions,
	deps updateDeps,
	mode interaction.ExecutionMode,
	meta config.Metadata,
	cfg models.ModsJSON,
	phase updateLoadPhase,
) (updateContext, error) {
	var lock []models.ModInstall
	var err error
	if phase == updateLockSyncPhase {
		lock, err = config.EnsureLock(ctx, deps.fs, meta)
	} else {
		lock, err = config.ReadLock(ctx, deps.fs, meta)
	}
	if err != nil {
		return updateContext{}, err
	}

	colorMode := colorModeForOutput(cmd.OutOrStdout())

	if phase != updateLockSyncPhase {
		return updateContext{
			meta:           meta,
			cfg:            cfg,
			lock:           lock,
			colorMode:      colorMode,
			shouldContinue: true,
		}, nil
	}

	syncOutcome, syncErr := locksync.RunLockSyncGate(locksync.GateInput{
		Ctx:         ctx,
		Fs:          deps.fs,
		Meta:        meta,
		Config:      cfg,
		Lock:        lock,
		Mode:        mode,
		CommandName: cmd.Name(),
		ColorMode:   colorMode,
		In:          cmd.InOrStdin(),
		Out:         cmd.OutOrStdout(),
		RunTea:      deps.runTea,
		PolicyFlags: opts.LockSync,
	})
	if syncErr != nil {
		return updateContext{}, syncErr
	}

	return updateContext{
		meta:           meta,
		cfg:            syncOutcome.Config,
		lock:           syncOutcome.Lock,
		colorMode:      colorMode,
		shouldContinue: syncOutcome.ShouldContinue,
	}, nil
}

func buildUpdateItems(cfg models.ModsJSON, lock []models.ModInstall) ([]updateItem, map[int]int) {
	items := make([]updateItem, 0, len(cfg.Mods))

	for index, mod := range cfg.Mods {
		displayName := updateDisplayName(mod)
		status := updateItemStatusPending
		if isPinned(mod) {
			status = updateItemStatusSkipped
		}
		lockIndex := models.LockIndexForMod(mod, lock)
		item := updateItem{
			ConfigIndex: index,
			LockIndex:   lockIndex,
			Mod:         mod,
			DisplayName: displayName,
			Status:      status,
		}
		items = append(items, item)
	}

	sort.SliceStable(items, func(leftIndex int, rightIndex int) bool {
		left := strings.ToLower(items[leftIndex].DisplayName)
		right := strings.ToLower(items[rightIndex].DisplayName)
		if left != right {
			return left < right
		}
		leftPlatform := strings.ToLower(string(items[leftIndex].Mod.Type))
		rightPlatform := strings.ToLower(string(items[rightIndex].Mod.Type))
		if leftPlatform != rightPlatform {
			return leftPlatform < rightPlatform
		}
		return strings.ToLower(items[leftIndex].Mod.ID) < strings.ToLower(items[rightIndex].Mod.ID)
	})

	indexByKey := make(map[int]int, len(cfg.Mods))
	for index, item := range items {
		indexByKey[item.ConfigIndex] = index
	}
	return items, indexByKey
}

func updateDisplayName(mod models.Mod) string {
	name := strings.TrimSpace(mod.Name)
	if name == "" {
		return mod.ID
	}
	return name
}

func runUpdateInteractive(ctx context.Context, cmd *cobra.Command, input updateExecutionInput) (updateExecutionOutcome, error) {
	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	model := newUpdateModel(updateModelInput{
		ctx:           execCtx,
		colorMode:     input.colorMode,
		items:         input.items,
		indexByKey:    input.indexByKey,
		suppressFinal: true,
		execRunner: func(ctx context.Context, sender updateExecSender) updateExecutionOutcome {
			return runUpdateExecution(ctx, input, sender)
		},
	})

	if runUpdateProgram == nil {
		return updateExecutionOutcome{}, errors.New("missing bubble tea runner")
	}

	options := view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())
	options = append(options, tea.WithAltScreen())
	result, err := runUpdateProgram(model, options...)
	if err != nil {
		return updateExecutionOutcome{}, err
	}
	typed, ok := result.(*updateModel)
	if !ok {
		return updateExecutionOutcome{}, errors.New("unexpected update model")
	}
	if outputErr := writeInteractiveUpdateTranscript(cmd, typed); outputErr != nil {
		return typed.outcome, outputErr
	}
	return typed.outcome, nil
}

func runUpdateTranscript(ctx context.Context, cmd *cobra.Command, input updateExecutionInput) (updateExecutionOutcome, error) {
	model := newUpdateTranscriptModel(
		ctx,
		input.colorMode,
		input.items,
		input.indexByKey,
		cmd.OutOrStdout(),
		func(ctx context.Context, sender updateExecSender) updateExecutionOutcome {
			return runUpdateExecution(ctx, input, sender)
		},
	)

	if runUpdateTranscriptProgram == nil {
		return updateExecutionOutcome{}, errors.New("missing bubble tea runner")
	}

	result, err := runUpdateTranscriptProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
	if err != nil {
		return updateExecutionOutcome{}, err
	}
	return updateOutcomeFromModel(result)
}

func writeInteractiveUpdateTranscript(cmd *cobra.Command, model *updateModel) error {
	if model == nil {
		return nil
	}
	switch model.outcome.errType {
	case updateExecutionErrorWriteLock:
		line := renderUpdateWriteLockFailureView(updateErrorViewInput{
			colorMode: model.colorMode,
			lockPath:  model.outcome.lockPath,
		})
		return runOutputLines(cmd, cmd.OutOrStdout(), []string{line})
	case updateExecutionErrorWriteConfig:
		line := renderUpdateWriteConfigFailureView(updateErrorViewInput{
			colorMode:  model.colorMode,
			configPath: model.outcome.configPath,
		})
		return runOutputLines(cmd, cmd.OutOrStdout(), []string{line})
	case updateExecutionErrorUnknown:
		line := renderFinalErrorLine(model.colorMode, model.outcome.err.Error())
		return runOutputLines(cmd, cmd.OutOrStdout(), []string{line})
	case updateExecutionErrorCanceled:
		return nil
	default:
		sections := buildUpdateResultSections(updateResultsViewInput{
			items:     model.items,
			colorMode: model.colorMode,
		})
		return runOutputLines(cmd, cmd.OutOrStdout(), sections)
	}
}

func runUpdateExecution(ctx context.Context, input updateExecutionInput, sender updateExecSender) updateExecutionOutcome {
	items := cloneUpdateItems(input.items)
	notifySkippedItems(items, sender)

	candidates := updateCandidates(*input.cfg)
	outcomes, processErr := processCandidates(
		ctx,
		input.meta,
		*input.cfg,
		input.lock,
		candidates,
		input.deps,
		sender,
	)
	if processErr != nil && outcomes == nil {
		return updateExecutionOutcome{items: items, err: processErr, errType: errorTypeForUpdate(processErr)}
	}

	applyUpdateOutcomes(items, outcomes, input.cfg, input.lock, input.indexByKey)

	persistContext := ctx
	if isContextCancellation(processErr) {
		persistContext = context.WithoutCancel(ctx)
	}

	if err := config.WriteLock(persistContext, input.deps.fs, input.meta, input.lock); err != nil {
		return updateExecutionOutcome{
			items:    items,
			err:      err,
			errType:  updateExecutionErrorWriteLock,
			lockPath: input.meta.LockPath(),
		}
	}

	if err := config.WriteConfig(persistContext, input.deps.fs, input.meta, *input.cfg); err != nil {
		return updateExecutionOutcome{
			items:      items,
			err:        err,
			errType:    updateExecutionErrorWriteConfig,
			configPath: input.meta.ConfigPath,
		}
	}

	if processErr != nil {
		return updateExecutionOutcome{items: items, err: processErr, errType: errorTypeForUpdate(processErr)}
	}

	return updateExecutionOutcome{items: items, errType: updateExecutionErrorNone}
}

func notifySkippedItems(items []updateItem, sender updateExecSender) {
	if sender.send == nil {
		return
	}
	for _, item := range items {
		if item.Status != updateItemStatusSkipped {
			continue
		}
		sender.Send(updateItemStatusMsg{
			index:       item.ConfigIndex,
			status:      updateItemStatusSkipped,
			displayName: item.DisplayName,
		})
	}
}

func errorTypeForUpdate(err error) updateExecutionErrorType {
	if err == nil {
		return updateExecutionErrorNone
	}
	if isContextCancellation(err) {
		return updateExecutionErrorCanceled
	}
	return updateExecutionErrorUnknown
}

func handleQuietUpdateResult(cmd *cobra.Command, outcome updateExecutionOutcome) (updateCounts, error) {
	counts := countUpdateOutcome(outcome)

	if outcome.errType == updateExecutionErrorWriteLock {
		line := renderUpdateWriteLockFailureView(updateErrorViewInput{
			colorMode: colorModeForOutput(cmd.OutOrStdout()),
			lockPath:  outcome.lockPath,
		})
		if outputErr := runOutputLines(cmd, cmd.OutOrStdout(), []string{line}); outputErr != nil {
			return counts, outputErr
		}
		return counts, clierrors.MarkHandled(outcome.err)
	}
	if outcome.errType == updateExecutionErrorWriteConfig {
		line := renderUpdateWriteConfigFailureView(updateErrorViewInput{
			colorMode:  colorModeForOutput(cmd.OutOrStdout()),
			configPath: outcome.configPath,
		})
		if outputErr := runOutputLines(cmd, cmd.OutOrStdout(), []string{line}); outputErr != nil {
			return counts, outputErr
		}
		return counts, clierrors.MarkHandled(outcome.err)
	}
	if outcome.errType == updateExecutionErrorCanceled {
		return counts, clierrors.MarkHandled(outcome.err)
	}
	if outcome.errType == updateExecutionErrorUnknown {
		line := renderFinalErrorLine(colorModeForOutput(cmd.OutOrStdout()), outcome.err.Error())
		if outputErr := runOutputLines(cmd, cmd.OutOrStdout(), []string{line}); outputErr != nil {
			return counts, outputErr
		}
		return counts, clierrors.MarkHandled(outcome.err)
	}

	lines := renderUpdateQuietFailure(updateResultsViewInput{
		items:     outcome.items,
		colorMode: colorModeForOutput(cmd.OutOrStdout()),
	})
	if len(lines) == 0 {
		return counts, nil
	}
	if outputErr := runOutputLines(cmd, cmd.OutOrStdout(), lines); outputErr != nil {
		return counts, outputErr
	}
	return counts, clierrors.MarkHandled(errUpdateFailures)
}

func handleUpdateOutcome(outcome updateExecutionOutcome) (updateCounts, error) {
	counts := countUpdateOutcome(outcome)

	switch outcome.errType {
	case updateExecutionErrorWriteLock, updateExecutionErrorWriteConfig:
		return counts, clierrors.MarkHandled(outcome.err)
	case updateExecutionErrorCanceled:
		return counts, clierrors.MarkHandled(outcome.err)
	case updateExecutionErrorUnknown:
		return counts, clierrors.MarkHandled(outcome.err)
	}

	if counts.failed > 0 {
		return counts, clierrors.MarkHandled(errUpdateFailures)
	}
	return counts, nil
}

func renderUpdateQuietFailure(input updateResultsViewInput) []string {
	if !hasUpdateStatus(input.items, updateItemStatusFailed) {
		return nil
	}
	lines := make([]string, 0, len(input.items)+2)
	lines = append(lines, renderFinalErrorLine(input.colorMode, i18n.T("cmd.update.summary.incomplete", nil)))
	failed := filterUpdateItems(input.items, updateItemStatusFailed)
	if len(failed) > 0 {
		lines = append(lines, renderUpdateSection(i18n.T("cmd.update.section.failed", nil), updateRunningViewInput{
			items:     failed,
			colorMode: input.colorMode,
		}, failed))
	}
	return pruneEmptySections(lines)
}

func countUpdateOutcome(outcome updateExecutionOutcome) updateCounts {
	counts := updateCounts{}
	for _, item := range outcome.items {
		switch item.Status {
		case updateItemStatusUpdated:
			counts.updated++
		case updateItemStatusFailed:
			counts.failed++
		}
	}
	return counts
}

func updateCandidates(cfg models.ModsJSON) []modUpdateCandidate {
	candidates := make([]modUpdateCandidate, 0, len(cfg.Mods))
	for modIndex := range cfg.Mods {
		candidates = append(candidates, modUpdateCandidate{
			ConfigIndex: modIndex,
			Mod:         cfg.Mods[modIndex],
		})
	}
	return candidates
}

func processCandidates(
	ctx context.Context,
	meta config.Metadata,
	cfg models.ModsJSON,
	lock []models.ModInstall,
	candidates []modUpdateCandidate,
	deps updateDeps,
	sender updateExecSender,
) ([]modUpdateOutcome, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	outcomes := make([]modUpdateOutcome, 0, len(candidates))
	outcomeChan := make(chan modUpdateOutcome, len(candidates))
	errChan := make(chan error, 1)

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(defaultUpdateMaxConcurrency)

	go func() {
		for _, candidate := range candidates {
			if groupCtx.Err() != nil {
				break
			}
			candidate := candidate
			group.Go(func() error {
				sendUpdateStart(candidate, lock, sender)
				outcome := processMod(groupCtx, meta, cfg, lock, candidate, deps, sender)
				outcomeChan <- outcome
				if err := groupCtx.Err(); err != nil {
					return err
				}
				return nil
			})
		}
		errChan <- group.Wait()
		close(outcomeChan)
	}()

	for outcome := range outcomeChan {
		outcomes = append(outcomes, outcome)
		sendUpdateOutcome(outcome, sender)
	}

	return outcomes, <-errChan
}

func sendUpdateStart(candidate modUpdateCandidate, lock []models.ModInstall, sender updateExecSender) {
	if sender.send == nil {
		return
	}
	if isPinned(candidate.Mod) {
		return
	}
	lockIndex := models.LockIndexForMod(candidate.Mod, lock)
	if lockIndex < 0 {
		return
	}
	sender.Send(updateItemStatusMsg{
		index:  candidate.ConfigIndex,
		status: updateItemStatusUpdating,
	})
}

func sendUpdateDownloadStart(candidate modUpdateCandidate, sender updateExecSender) {
	if sender.send == nil {
		return
	}
	if isPinned(candidate.Mod) {
		return
	}
	sender.Send(updateItemStatusMsg{
		index:  candidate.ConfigIndex,
		status: updateItemStatusDownloading,
	})
}

func sendUpdateOutcome(outcome modUpdateOutcome, sender updateExecSender) {
	if sender.send == nil {
		return
	}
	status := statusForOutcome(outcome)
	sender.Send(updateItemStatusMsg{
		index:       outcome.ConfigIndex,
		status:      status,
		failReason:  outcome.FailReason,
		displayName: outcome.NewName,
	})
}

func statusForOutcome(outcome modUpdateOutcome) updateItemStatus {
	switch outcome.Result {
	case updateOutcomeUpdated:
		return updateItemStatusUpdated
	case updateOutcomeSkipped:
		return updateItemStatusSkipped
	case updateOutcomeFailed:
		return updateItemStatusFailed
	default:
		return updateItemStatusUpToDate
	}
}

func applyUpdateOutcomes(items []updateItem, outcomes []modUpdateOutcome, cfg *models.ModsJSON, lock []models.ModInstall, indexByKey map[int]int) {
	for _, outcome := range outcomes {
		if strings.TrimSpace(outcome.NewName) != "" && outcome.ConfigIndex >= 0 && outcome.ConfigIndex < len(cfg.Mods) {
			cfg.Mods[outcome.ConfigIndex].Name = outcome.NewName
		}

		if outcome.Result == updateOutcomeUpdated && outcome.LockIndex >= 0 && outcome.LockIndex < len(lock) {
			lock[outcome.LockIndex] = outcome.NewInstall
		}

		updateItemFromOutcome(items, outcome, indexByKey)
	}
}

func updateItemFromOutcome(items []updateItem, outcome modUpdateOutcome, indexByKey map[int]int) {
	itemIndex, ok := indexByKey[outcome.ConfigIndex]
	if !ok || itemIndex < 0 || itemIndex >= len(items) {
		return
	}
	item := items[itemIndex]
	item.Status = statusForOutcome(outcome)
	item.FailReason = outcome.FailReason
	item.Progress = nil
	if strings.TrimSpace(outcome.NewName) != "" {
		item.DisplayName = outcome.NewName
	}
	items[itemIndex] = item
}

func isContextCancellation(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
