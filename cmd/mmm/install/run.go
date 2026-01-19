package install

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"golang.org/x/sync/errgroup"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/locksync"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/cobra"
)

var errUnresolvedFiles = errors.New("unresolved files in mods folder")
var errInstallFailures = errors.New("one or more mods failed to install")

type installRunState struct {
	meta           config.Metadata
	cfg            models.ModsJSON
	lock           []models.ModInstall
	mode           interaction.ExecutionMode
	shouldContinue bool
}

func runInstall(ctx context.Context, cmd *cobra.Command, opts installOptions, deps installDeps) (Result, error) {
	runState, err := prepareInstallRunState(ctx, cmd, opts, deps)
	if err != nil {
		return Result{}, err
	}
	if !runState.shouldContinue {
		return Result{}, nil
	}

	colorMode := colorModeForOutput(cmd.OutOrStdout())
	preflight, err := preflightInstall(ctx, runState.meta, runState.cfg, runState.lock, deps, colorMode.Enabled())
	if err != nil {
		return handleInstallPreflightError(cmd, deps, preflight, err)
	}
	if outputErr := writeInstallPreflightOutput(cmd, deps, preflight.lines); outputErr != nil {
		return Result{}, outputErr
	}

	items, indexByKey := buildInstallItems(runState.cfg)
	executionInput := installExecutionInput{
		meta:       runState.meta,
		cfg:        runState.cfg,
		lock:       runState.lock,
		deps:       deps,
		items:      items,
		indexByKey: indexByKey,
	}

	result := Result{
		InstalledCount: len(items),
		UnmanagedFound: preflight.unmanagedFound,
	}

	if opts.Quiet {
		return runQuietInstall(ctx, cmd, deps, executionInput, result)
	}
	if runState.mode == interaction.ExecutionModeNonTTY {
		return runNonTTYInstall(ctx, cmd, executionInput, result)
	}
	return runInteractiveInstall(ctx, cmd, executionInput, result)
}

func handleInstallPreflightError(cmd *cobra.Command, deps installDeps, preflight scanReportOutcome, err error) (Result, error) {
	if !errors.Is(err, errUnresolvedFiles) {
		return Result{}, handleInstallFailure(cmd, deps, err)
	}
	if outputErr := writeInstallUnresolvedOutput(cmd, deps, preflight.lines); outputErr != nil {
		return Result{}, outputErr
	}
	return Result{UnmanagedFound: preflight.unmanagedFound}, clierrors.MarkHandled(err)
}

func prepareInstallRunState(ctx context.Context, cmd *cobra.Command, opts installOptions, deps installDeps) (installRunState, error) {
	runState := newInstallRunState(cmd, opts)

	configState, err := readConfigAndLock(ctx, cmd, opts, deps, runState)
	if err == nil {
		runState.cfg = configState.cfg
		runState.lock = configState.lock
		runState.shouldContinue = configState.shouldContinue
		return runState, nil
	}

	var notFound *config.ConfigFileNotFoundException
	if !errors.As(err, &notFound) {
		return runState, handleInstallFailure(cmd, deps, err)
	}

	return resolveInstallConfigAfterPrompt(ctx, cmd, opts, deps, runState)
}

func newInstallRunState(cmd *cobra.Command, opts installOptions) installRunState {
	mode := interaction.ResolveExecutionMode(interaction.ExecutionModeInput{
		Unattended: opts.Unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
	})

	return installRunState{
		meta:           config.NewMetadata(opts.ConfigPath),
		mode:           mode,
		shouldContinue: true,
	}
}

type installConfigState struct {
	cfg            models.ModsJSON
	lock           []models.ModInstall
	shouldContinue bool
}

func readConfigAndLock(ctx context.Context, cmd *cobra.Command, opts installOptions, deps installDeps, runState installRunState) (installConfigState, error) {
	cfg, err := config.ReadConfig(ctx, deps.fs, runState.meta)
	if err != nil {
		return installConfigState{}, err
	}
	lock, err := config.EnsureLock(ctx, deps.fs, runState.meta)
	if err != nil {
		return installConfigState{}, err
	}
	if opts.SkipLockSync {
		return installConfigState{
			cfg:            cfg,
			lock:           lock,
			shouldContinue: true,
		}, nil
	}
	syncOutcome, syncErr := locksync.RunLockSyncGate(locksync.GateInput{
		Ctx:         ctx,
		Fs:          deps.fs,
		Meta:        runState.meta,
		Config:      cfg,
		Lock:        lock,
		Mode:        runState.mode,
		CommandName: cmd.Name(),
		ColorMode:   colorModeForOutput(cmd.OutOrStdout()),
		In:          cmd.InOrStdin(),
		Out:         cmd.OutOrStdout(),
		RunTea:      deps.runTea,
		PolicyFlags: opts.LockSync,
	})
	if syncErr != nil {
		return installConfigState{}, syncErr
	}
	return installConfigState{
		cfg:            syncOutcome.Config,
		lock:           syncOutcome.Lock,
		shouldContinue: syncOutcome.ShouldContinue,
	}, nil
}

func resolveInstallConfigAfterPrompt(ctx context.Context, cmd *cobra.Command, opts installOptions, deps installDeps, runState installRunState) (installRunState, error) {
	promptErr := configMissingPromptError(opts, cmd, runState.meta)
	if promptErr != nil {
		if outputErr := writeConfigMissingOutput(cmd, deps, runState.meta); outputErr != nil {
			return runState, outputErr
		}
		return runState, clierrors.MarkHandled(promptErr)
	}

	confirmed, canceled, err := runConfigInitPrompt(cmd, deps, runState.meta)
	if err != nil {
		return runState, err
	}
	if canceled || !confirmed {
		runState.shouldContinue = false
		return runState, nil
	}

	err = runInstallInit(ctx, cmd, deps, &runState)
	if err != nil {
		return runState, err
	}
	if !runState.shouldContinue {
		return runState, nil
	}

	configState, err := readConfigAndLock(ctx, cmd, opts, deps, runState)
	if err != nil {
		return runState, handleInstallFailure(cmd, deps, err)
	}
	runState.cfg = configState.cfg
	runState.lock = configState.lock
	runState.shouldContinue = configState.shouldContinue
	return runState, nil
}

func runInstallInit(ctx context.Context, cmd *cobra.Command, deps installDeps, runState *installRunState) error {
	if deps.runInit == nil {
		return errors.New("missing init runner")
	}
	if runErr := deps.runInit(ctx, cmd, initRequest{configPath: runState.meta.ConfigPath}); runErr != nil {
		if errors.Is(runErr, initCmd.ErrInitCanceled) {
			runState.shouldContinue = false
			return nil
		}
		return runErr
	}
	return nil
}

func configMissingPromptError(opts installOptions, cmd *cobra.Command, meta config.Metadata) error {
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

func handleInstallFailure(cmd *cobra.Command, deps installDeps, err error) error {
	if outputErr := writeInstallFailureOutput(cmd, deps, err); outputErr != nil {
		return outputErr
	}
	return clierrors.MarkHandled(err)
}

func writeInstallFailureOutput(cmd *cobra.Command, deps installDeps, err error) error {
	colorMode := colorModeForOutput(cmd.OutOrStdout())
	headline := renderFinalErrorLine(colorMode, i18n.T("cmd.install.error.failed", &i18n.Tvars{
		Data: &i18n.TData{"reason": err.Error()},
	}))
	hint := i18n.T("cmd.install.error.failed_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}
	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{headline, hint})
}

func writeConfigMissingOutput(cmd *cobra.Command, deps installDeps, meta config.Metadata) error {
	colorMode := colorModeForOutput(cmd.OutOrStdout())
	headline := renderFinalErrorLine(colorMode, i18n.T("cmd.config.error.missing", &i18n.Tvars{
		Data: &i18n.TData{"configPath": meta.ConfigPath},
	}))
	hint := i18n.T("cmd.config.error.missing_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}
	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{headline, hint})
}

func writeInstallUnresolvedOutput(cmd *cobra.Command, deps installDeps, lines []string) error {
	if len(lines) == 0 {
		return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{i18n.T("cmd.install.error.unresolved", nil)})
	}
	section := strings.Join(lines, "\n")
	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{section, i18n.T("cmd.install.error.unresolved", nil)})
}

func writeInstallPreflightOutput(cmd *cobra.Command, deps installDeps, lines []string) error {
	if len(lines) == 0 {
		return nil
	}
	section := strings.Join(lines, "\n")
	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{section})
}

func installConfiguredMods(input installConfiguredInputs) (installConfiguredOutcome, error) {
	cfg := input.cfg
	executionState := input.state
	writeState := input.writeState

	group, groupCtx := errgroup.WithContext(input.ctx)
	group.SetLimit(installMaxConcurrency(len(cfg.Mods)))

	var failedCount int64

	for index := range cfg.Mods {
		mod := cfg.Mods[index]
		modIndex := index
		group.Go(func() error {
			return runInstallWorker(groupCtx, input, mod, cfg, modIndex, executionState, writeState, &failedCount)
		})
	}

	if err := group.Wait(); err != nil {
		return installConfiguredOutcome{}, err
	}

	return installConfiguredOutcome{
		failedCount: int(failedCount),
		items:       snapshotInstallItems(executionState, cfg),
	}, nil
}

func runInstallWorker(
	ctx context.Context,
	input installConfiguredInputs,
	mod models.Mod,
	cfg models.ModsJSON,
	modIndex int,
	state *installExecutionState,
	writeState *installWriteState,
	failedCount *int64,
) error {
	workerInput := input
	workerInput.ctx = ctx

	outcome, err := installConfiguredMod(workerInput, mod, cfg, input.lock, state)
	if err != nil {
		return err
	}
	if outcome.failed {
		atomic.AddInt64(failedCount, 1)
		if state != nil {
			state.setFailure(installModKey(mod), outcome.failureReason)
		}
		return nil
	}

	if state != nil {
		state.setSuccess(installModKey(mod), outcome.newName)
	}
	if writeState == nil {
		return nil
	}
	return writeState.recordDownloadSuccess(ctx, modIndex, outcome)
}

func installConfiguredMod(input installConfiguredInputs, mod models.Mod, cfg models.ModsJSON, lock []models.ModInstall, state *installExecutionState) (modInstallOutcome, error) {
	version := modVersionLabel(mod)
	if err := input.deps.logger.Debug(i18n.T("cmd.install.debug.checking", &i18n.Tvars{
		Data: &i18n.TData{
			"name":     mod.Name,
			"version":  version,
			"platform": string(mod.Type),
		},
	})); err != nil {
		return modInstallOutcome{}, err
	}

	return installMod(installModInputs{
		ctx:      input.ctx,
		meta:     input.meta,
		cfg:      cfg,
		lock:     lock,
		mod:      mod,
		deps:     input.deps,
		colorize: input.colorize,
		state:    state,
	})
}

func runInteractiveInstall(
	ctx context.Context,
	cmd *cobra.Command,
	executionInput installExecutionInput,
	result Result,
) (Result, error) {
	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	runningFooter := runningFooterFromContext(ctx)
	model := newInstallModel(execCtx, colorModeForOutput(cmd.OutOrStdout()), executionInput.items, executionInput.indexByKey, cancel, func(ctx context.Context, sender httpclient.Sender) installExecutionOutcome {
		return runInstallExecution(ctx, executionInput, sender)
	}, runningFooter)

	if runInstallProgram == nil {
		return result, errors.New("missing bubble tea runner")
	}

	options := view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())
	options = append(options, installProgramOptionsFromContext(ctx)...)
	_, err := runInstallProgram(model, options...)
	if err != nil {
		return result, err
	}

	NotifyInstallViewObserver(ctx, model.View())

	outcome := model.outcome
	if outcome.err != nil {
		return result, clierrors.MarkHandled(outcome.err)
	}
	return result, nil
}

func runNonTTYInstall(
	ctx context.Context,
	cmd *cobra.Command,
	executionInput installExecutionInput,
	result Result,
) (Result, error) {
	model := newInstallTranscriptModel(ctx, colorModeForOutput(cmd.OutOrStdout()), executionInput.items, executionInput.indexByKey, cmd.OutOrStdout(), func(ctx context.Context, sender httpclient.Sender) installExecutionOutcome {
		return runInstallExecution(ctx, executionInput, sender)
	})

	if runInstallTranscriptProgram == nil {
		return result, errors.New("missing bubble tea runner")
	}

	options := view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())
	_, err := runInstallTranscriptProgram(model, options...)
	if err != nil {
		return result, err
	}

	outcome := model.outcome
	if outcome.err != nil {
		return result, clierrors.MarkHandled(outcome.err)
	}
	return result, nil
}

func runQuietInstall(
	ctx context.Context,
	cmd *cobra.Command,
	deps installDeps,
	executionInput installExecutionInput,
	result Result,
) (Result, error) {
	outcome := runInstallExecution(ctx, executionInput, nil)
	if outcome.err == nil {
		return result, nil
	}

	colorMode := colorModeForOutput(cmd.OutOrStdout())
	lines := renderInstallQuietFailure(colorMode, outcome)
	if outputErr := runOutputLines(cmd, deps, cmd.OutOrStdout(), lines); outputErr != nil {
		return result, outputErr
	}
	return result, clierrors.MarkHandled(outcome.err)
}

func renderInstallQuietFailure(colorMode view.ColorMode, outcome installExecutionOutcome) []string {
	switch outcome.errType {
	case installExecutionErrorDownload:
		return renderInstallQuietDownloadFailure(colorMode, outcome.items)
	case installExecutionErrorWriteLock:
		return renderInstallWriteLockSummary(colorMode, outcome.lockPath)
	case installExecutionErrorWriteConfig:
		return renderInstallWriteConfigSummary(colorMode, outcome.configPath)
	case installExecutionErrorCanceled:
		return renderInstallCanceledSummary(colorMode)
	default:
		return []string{renderFinalErrorLine(colorMode, i18n.T("cmd.install.error.failed", &i18n.Tvars{
			Data: &i18n.TData{"reason": outcome.err.Error()},
		}))}
	}
}

func renderInstallQuietDownloadFailure(colorMode view.ColorMode, items []installItem) []string {
	headline := renderFinalErrorLine(colorMode, i18n.T("cmd.install.quiet.download_failed", nil))
	failedItems := make([]installItem, 0, len(items))
	for _, item := range items {
		if item.Status != installItemFailed {
			continue
		}
		failedItems = append(failedItems, item)
	}
	if len(failedItems) == 0 {
		return []string{headline}
	}
	return []string{headline, strings.Join(renderInstallItemLines(colorMode, failedItems), "\n")}
}
