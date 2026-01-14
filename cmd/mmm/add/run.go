package add

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/modsetup"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
)

type downloadFailureError struct {
	err error
}

func (err downloadFailureError) Error() string {
	return err.err.Error()
}

func (err downloadFailureError) Unwrap() error {
	return err.err
}

func runAdd(ctx context.Context, commandSpan *perf.Span, cmd *cobra.Command, opts addOptions, deps addDeps) (telemetry.CommandTelemetry, error) {
	runState, err := prepareAddRunState(ctx, cmd, opts, deps)
	if err != nil {
		return addFailureTelemetryWithoutArgs(runState.mode.String(), runState.mode.IsInteractive(), err), err
	}
	if !runState.shouldContinue {
		return telemetry.CommandTelemetry{
			Command:       "add",
			Interactive:   runState.mode.IsInteractive(),
			ExecutionMode: runState.mode.String(),
		}, nil
	}

	colorMode := colorModeForOutput(cmd.OutOrStdout())
	platformValue, projectID := normalizedAddIdentifiers(opts)
	identifiers := addIdentifiers{platformValue: platformValue, projectID: projectID}

	for {
		outcome, attemptErr := runAddAttempt(ctx, commandSpan, cmd, opts, deps, runState, colorMode, identifiers)
		if attemptErr != nil {
			return addFailureTelemetry(outcome.platformValue, outcome.projectID, opts, runState.mode.String(), runState.mode.IsInteractive(), attemptErr), attemptErr
		}
		if outcome.recovered {
			identifiers = addIdentifiers{platformValue: outcome.platformValue, projectID: outcome.projectID}
			continue
		}
		return outcome.telemetry, nil
	}
}

func runAddAttempt(
	ctx context.Context,
	commandSpan *perf.Span,
	cmd *cobra.Command,
	opts addOptions,
	deps addDeps,
	runState addRunState,
	colorMode view.ColorMode,
	identifiers addIdentifiers,
) (addAttemptOutcome, error) {
	existingOutcome, handled, existingInstallErr := runExistingInstallStep(ctx, commandSpan, cmd, opts, deps, runState, identifiers)
	if handled {
		return existingOutcome, existingInstallErr
	}
	return runResolvedAdd(ctx, commandSpan, cmd, opts, deps, runState, colorMode, identifiers)
}

func runExistingInstallStep(
	ctx context.Context,
	commandSpan *perf.Span,
	cmd *cobra.Command,
	opts addOptions,
	deps addDeps,
	runState addRunState,
	identifiers addIdentifiers,
) (addAttemptOutcome, bool, error) {
	telemetryResult, handled, existingInstallErr := handleExistingInstallIfPresent(existingInstallCheckInput{
		ctx:           ctx,
		commandSpan:   commandSpan,
		meta:          runState.meta,
		cfg:           runState.cfg,
		lock:          runState.lock,
		platformValue: identifiers.platformValue,
		projectID:     identifiers.projectID,
		opts:          opts,
		deps:          deps,
		mode:          runState.mode,
		in:            cmd.InOrStdin(),
		out:           cmd.OutOrStdout(),
	})
	if !handled {
		return addAttemptOutcome{}, false, nil
	}
	return addAttemptOutcome{
		platformValue: identifiers.platformValue,
		projectID:     identifiers.projectID,
		telemetry:     telemetryResult,
	}, true, existingInstallErr
}

func runResolvedAdd(
	ctx context.Context,
	commandSpan *perf.Span,
	cmd *cobra.Command,
	opts addOptions,
	deps addDeps,
	runState addRunState,
	colorMode view.ColorMode,
	identifiers addIdentifiers,
) (addAttemptOutcome, error) {
	resolveOutcome, resolveErr := resolveAddMod(ctx, commandSpan, cmd, opts, deps, runState, identifiers)
	if resolveErr != nil {
		return addAttemptOutcome{
			platformValue: resolveOutcome.platformValue,
			projectID:     resolveOutcome.projectID,
		}, resolveErr
	}
	if resolveOutcome.recovered {
		return addAttemptOutcome{
			platformValue: resolveOutcome.platformValue,
			projectID:     resolveOutcome.projectID,
			recovered:     true,
		}, nil
	}

	downloadOutcome := runDownloadStep(ctx, cmd, opts, deps, runState, colorMode, identifiers, resolveOutcome)
	if downloadOutcome.err != nil {
		return addAttemptOutcome{
			platformValue: downloadOutcome.platformValue,
			projectID:     downloadOutcome.projectID,
		}, downloadOutcome.err
	}
	if downloadOutcome.recovered {
		return addAttemptOutcome{
			platformValue: downloadOutcome.platformValue,
			projectID:     downloadOutcome.projectID,
			recovered:     true,
		}, nil
	}

	telemetryPayload, err := finishAdd(ctx, cmd, opts, deps, runState, colorMode, resolveOutcome)
	if err != nil {
		return addAttemptOutcome{
			platformValue: resolveOutcome.resolved.platform,
			projectID:     resolveOutcome.resolved.projectID,
		}, err
	}
	return addAttemptOutcome{
		platformValue: resolveOutcome.resolved.platform,
		projectID:     resolveOutcome.resolved.projectID,
		telemetry:     telemetryPayload,
	}, nil
}

func runDownloadStep(
	ctx context.Context,
	cmd *cobra.Command,
	opts addOptions,
	deps addDeps,
	runState addRunState,
	colorMode view.ColorMode,
	identifiers addIdentifiers,
	resolveOutcome resolveStepOutcome,
) recoveryOutcome {
	_, downloadErr := ensureRemoteMod(ensureRemoteModInput{
		ctx:              ctx,
		meta:             runState.meta,
		cfg:              runState.cfg,
		remoteMod:        resolveOutcome.remoteMod,
		resolvedPlatform: resolveOutcome.resolved.platform,
		resolvedID:       resolveOutcome.resolved.projectID,
		deps:             deps,
		mode:             runState.mode,
		colorMode:        colorMode,
		input:            cmd.InOrStdin(),
		output:           cmd.OutOrStdout(),
		quiet:            opts.Quiet,
	})
	if downloadErr == nil {
		return recoveryOutcome{
			platformValue: resolveOutcome.resolved.platform,
			projectID:     resolveOutcome.resolved.projectID,
		}
	}

	var downloadFailure downloadFailureError
	if errors.As(downloadErr, &downloadFailure) {
		downloadOutcome := handleDownloadFailure(cmd, runState, deps, identifiers.platformValue, identifiers.projectID, downloadFailure)
		if downloadOutcome.recovered || downloadOutcome.err != nil {
			return downloadOutcome
		}
	}
	return recoveryOutcome{
		platformValue: resolveOutcome.resolved.platform,
		projectID:     resolveOutcome.resolved.projectID,
		err:           downloadErr,
	}
}

func finishAdd(
	ctx context.Context,
	cmd *cobra.Command,
	opts addOptions,
	deps addDeps,
	runState addRunState,
	colorMode view.ColorMode,
	resolveOutcome resolveStepOutcome,
) (telemetry.CommandTelemetry, error) {
	if err := persistAdd(addPersistInput{
		ctx:              ctx,
		meta:             runState.meta,
		cfg:              runState.cfg,
		lock:             runState.lock,
		remoteMod:        resolveOutcome.remoteMod,
		resolvedPlatform: resolveOutcome.resolved.platform,
		resolvedID:       resolveOutcome.resolved.projectID,
		opts:             opts,
		setupCoordinator: runState.setupCoordinator,
	}); err != nil {
		return telemetry.CommandTelemetry{}, err
	}

	if opts.Quiet {
		return addSuccessTelemetry(resolveOutcome.resolved.platform, resolveOutcome.resolved.projectID, opts, runState.mode.String(), runState.mode.IsInteractive()), nil
	}

	if outputErr := runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{renderAddSuccessLine(colorMode, resolveOutcome.remoteMod.Name, resolveOutcome.resolved.projectID, resolveOutcome.resolved.platform)}); outputErr != nil {
		return telemetry.CommandTelemetry{}, outputErr
	}
	return addSuccessTelemetry(resolveOutcome.resolved.platform, resolveOutcome.resolved.projectID, opts, runState.mode.String(), runState.mode.IsInteractive()), nil
}

func prepareAddRunState(ctx context.Context, cmd *cobra.Command, opts addOptions, deps addDeps) (addRunState, error) {
	mode := interaction.ResolveExecutionMode(interaction.ExecutionModeInput{
		Unattended: opts.Unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
	})
	runState := addRunState{
		meta:             config.NewMetadata(opts.ConfigPath),
		mode:             mode,
		shouldContinue:   true,
		setupCoordinator: modsetup.NewSetupCoordinator(deps.fs, deps.minecraftClient, modsetup.Downloader(deps.downloader)),
	}

	configState, err := ensureAddConfig(ctx, cmd, opts, deps, runState)
	if err != nil {
		return runState, err
	}
	runState.cfg = configState.cfg
	runState.lock = configState.lock
	runState.shouldContinue = configState.shouldContinue
	return runState, nil
}

func ensureAddConfig(ctx context.Context, cmd *cobra.Command, opts addOptions, deps addDeps, runState addRunState) (addConfigState, error) {
	configState, err := loadAddConfig(ctx, deps, runState.meta)
	if err == nil {
		return configState, nil
	}

	var notFound *config.ConfigFileNotFoundException
	if !errors.As(err, &notFound) {
		return addConfigState{}, handleAddFailure(cmd, deps, err)
	}

	return handleMissingAddConfig(ctx, cmd, opts, deps, runState)
}

func loadAddConfig(ctx context.Context, deps addDeps, meta config.Metadata) (addConfigState, error) {
	cfg, err := config.ReadConfig(ctx, deps.fs, meta)
	if err != nil {
		return addConfigState{}, err
	}
	lock, lockErr := config.EnsureLock(ctx, deps.fs, meta)
	if lockErr != nil {
		return addConfigState{}, lockErr
	}
	return addConfigState{
		cfg:            cfg,
		lock:           lock,
		shouldContinue: true,
	}, nil
}

func handleMissingAddConfig(ctx context.Context, cmd *cobra.Command, opts addOptions, deps addDeps, runState addRunState) (addConfigState, error) {
	promptErr := configMissingPromptError(opts, cmd, runState.meta)
	if promptErr != nil {
		if outputErr := writeConfigMissingOutput(cmd, deps, runState.meta); outputErr != nil {
			return addConfigState{}, outputErr
		}
		return addConfigState{}, clierrors.MarkHandled(promptErr)
	}

	confirmed, canceled, err := runConfigInitPrompt(cmd, deps, runState.meta)
	if err != nil {
		return addConfigState{}, err
	}
	if canceled || !confirmed {
		return addConfigState{shouldContinue: false}, nil
	}

	if initErr := runInteractiveInit(ctx, cmd, deps, opts, runState.meta); initErr != nil {
		if errors.Is(initErr, initCmd.ErrInitCanceled) {
			return addConfigState{shouldContinue: false}, nil
		}
		return addConfigState{}, initErr
	}

	configState, err := loadAddConfig(ctx, deps, runState.meta)
	if err != nil {
		return addConfigState{}, handleAddFailure(cmd, deps, err)
	}
	return configState, nil
}

func configMissingPromptError(opts addOptions, cmd *cobra.Command, meta config.Metadata) error {
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

func handleAddFailure(cmd *cobra.Command, deps addDeps, err error) error {
	if outputErr := writeAddFailureOutput(cmd, deps, err); outputErr != nil {
		return outputErr
	}
	return clierrors.MarkHandled(err)
}

func writeAddFailureOutput(cmd *cobra.Command, deps addDeps, err error) error {
	colorMode := colorModeForOutput(cmd.OutOrStdout())
	headline := renderFinalErrorLine(colorMode, i18n.T("cmd.add.error.failed", &i18n.Tvars{
		Data: &i18n.TData{"reason": err.Error()},
	}))

	hint := i18n.T("cmd.add.error.failed_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}

	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{headline, hint})
}

func writeConfigMissingOutput(cmd *cobra.Command, deps addDeps, meta config.Metadata) error {
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

func resolveAddMod(
	ctx context.Context,
	commandSpan *perf.Span,
	cmd *cobra.Command,
	opts addOptions,
	deps addDeps,
	runState addRunState,
	identifiers addIdentifiers,
) (resolveStepOutcome, error) {
	resolved, remoteMod, err := resolveAndNormalizeRemoteMod(addResolveInputs{
		ctx:           ctx,
		commandSpan:   commandSpan,
		cfg:           runState.cfg,
		opts:          opts,
		platformValue: identifiers.platformValue,
		projectID:     identifiers.projectID,
		deps:          deps,
		mode:          runState.mode,
		in:            cmd.InOrStdin(),
		out:           cmd.OutOrStdout(),
	})
	if err == nil {
		return resolveStepOutcome{
			resolved:      resolved,
			remoteMod:     remoteMod,
			recovered:     false,
			platformValue: resolved.platform,
			projectID:     resolved.projectID,
		}, nil
	}

	resolveRecovery := handleResolveFailure(cmd, runState, deps, identifiers.platformValue, identifiers.projectID, err)
	if resolveRecovery.recovered {
		return resolveStepOutcome{
			recovered:     true,
			platformValue: resolveRecovery.platformValue,
			projectID:     resolveRecovery.projectID,
		}, nil
	}
	return resolveStepOutcome{
		platformValue: resolveRecovery.platformValue,
		projectID:     resolveRecovery.projectID,
	}, resolveRecovery.err
}

func ensureRemoteMod(input ensureRemoteModInput) (modinstall.EnsureResult, error) {
	downloadCtx, downloadSpan := perf.StartSpan(input.ctx, "app.command.add.stage.download",
		perf.WithAttributes(
			attribute.String("url", input.remoteMod.DownloadURL),
			attribute.String("platform", string(input.resolvedPlatform)),
			attribute.String("project_id", input.resolvedID),
			attribute.String("file_name", input.remoteMod.FileName),
		),
	)

	destination := filepath.Join(input.meta.ModsFolderPath(input.cfg), input.remoteMod.FileName)
	installer := modinstall.NewInstaller(input.deps.fs, modinstall.Downloader(input.deps.downloader))
	fileInput := ensureRemoteFileInput{
		ctx:              downloadCtx,
		installer:        installer,
		meta:             input.meta,
		cfg:              input.cfg,
		remoteMod:        input.remoteMod,
		resolvedPlatform: input.resolvedPlatform,
		resolvedID:       input.resolvedID,
		deps:             input.deps,
		colorMode:        input.colorMode,
		input:            input.input,
		output:           input.output,
	}
	var (
		ensureResult modinstall.EnsureResult
		err          error
	)
	if input.quiet || !input.mode.IsInteractive() {
		ensureResult, err = ensureRemoteFileSilent(fileInput)
	} else {
		ensureResult, err = ensureRemoteFileInteractive(fileInput)
	}
	if err != nil {
		if message, handled := integrityErrorMessage(err, input.remoteMod.Name); handled {
			err = errors.New(message)
		}
		downloadSpan.SetAttributes(attribute.Bool("success", false))
		downloadSpan.End()
		return modinstall.EnsureResult{}, err
	}

	downloadSpan.SetAttributes(
		attribute.Bool("success", true),
		attribute.String("path", destination),
		attribute.String("reason", string(ensureResult.Reason)),
	)
	downloadSpan.End()
	return ensureResult, nil
}

func ensureRemoteFileSilent(input ensureRemoteFileInput) (modinstall.EnsureResult, error) {
	install := models.ModInstall{
		FileName:    input.remoteMod.FileName,
		Hash:        input.remoteMod.Hash,
		DownloadURL: input.remoteMod.DownloadURL,
	}
	client := platform.PreferredDownloadClient(input.deps.clients)
	result, err := input.installer.EnsureLockedFile(input.ctx, input.meta, input.cfg, install, client, nil)
	if err != nil {
		return modinstall.EnsureResult{}, downloadFailureError{err: err}
	}
	return result, nil
}

func ensureRemoteFileInteractive(input ensureRemoteFileInput) (modinstall.EnsureResult, error) {
	install := models.ModInstall{
		FileName:    input.remoteMod.FileName,
		Hash:        input.remoteMod.Hash,
		DownloadURL: input.remoteMod.DownloadURL,
	}
	client := platform.PreferredDownloadClient(input.deps.clients)
	model := newDownloadProgressModel(input.remoteMod.Name, input.resolvedID, input.resolvedPlatform, input.colorMode, func(sender httpclient.Sender) (modinstall.EnsureResult, error) {
		return input.installer.EnsureLockedFile(input.ctx, input.meta, input.cfg, install, client, sender)
	})

	result, err := runProgressProgram(model, view.ProgramOptions(input.input, input.output)...)
	if err != nil {
		return modinstall.EnsureResult{}, downloadFailureError{err: err}
	}
	progressModel, ok := result.(*downloadProgressModel)
	if !ok {
		return modinstall.EnsureResult{}, errors.New("unexpected download progress model")
	}
	if progressModel.err != nil {
		return modinstall.EnsureResult{}, downloadFailureError{err: progressModel.err}
	}
	return progressModel.result, nil
}

func resolveAndNormalizeRemoteMod(inputs addResolveInputs) (resolvedRemoteMod, platform.RemoteMod, error) {
	resolved, fetchErr := resolveRemoteModWithSpan(inputs)
	if fetchErr != nil {
		return resolved, platform.RemoteMod{}, fetchErr
	}

	remoteMod := resolved.remoteMod
	remoteMod, err := normalizeRemoteModFileName(remoteMod)
	if err != nil {
		return resolved, platform.RemoteMod{}, err
	}

	return resolved, remoteMod, nil
}

func persistAdd(input addPersistInput) error {
	persistCtx, persistSpan := perf.StartSpan(input.ctx, "app.command.add.stage.persist",
		perf.WithAttributes(
			attribute.String("platform", string(input.resolvedPlatform)),
			attribute.String("project_id", input.resolvedID),
		),
	)

	_, err := input.setupCoordinator.EnsurePersisted(persistCtx, input.meta, input.cfg, input.lock, input.resolvedPlatform, input.resolvedID, input.remoteMod, modsetup.EnsurePersistOptions{
		Version:              input.opts.Version,
		AllowVersionFallback: input.opts.AllowVersionFallback,
	})
	persistSpan.SetAttributes(attribute.Bool("success", err == nil))
	persistSpan.End()
	return err
}

func handleDownloadFailure(cmd *cobra.Command, runState addRunState, deps addDeps, platformValue models.Platform, projectID string, failure downloadFailureError) recoveryOutcome {
	if !runState.mode.IsInteractive() {
		if outputErr := writeDownloadFailureOutput(cmd, deps, platformValue, projectID); outputErr != nil {
			return recoveryOutcome{platformValue: platformValue, projectID: projectID, recovered: false, err: outputErr}
		}
		return recoveryOutcome{platformValue: platformValue, projectID: projectID, recovered: false, err: clierrors.MarkHandled(failure)}
	}

	result, promptErr := runRecoveryFlow(recoveryFlowInput{
		reason:      recoveryReasonDownloadFailed,
		platform:    platformValue,
		projectID:   projectID,
		loader:      runState.cfg.Loader.String(),
		gameVersion: runState.cfg.GameVersion,
		retryCount:  retryCountForClient(platform.PreferredDownloadClient(deps.clients)),
		colorMode:   colorModeForOutput(cmd.OutOrStdout()),
		in:          cmd.InOrStdin(),
		out:         cmd.OutOrStdout(),
		runTea:      deps.runTea,
	})
	if promptErr != nil {
		return recoveryOutcome{platformValue: platformValue, projectID: projectID, recovered: false, err: clierrors.MarkHandled(promptErr)}
	}
	return recoveryOutcome{platformValue: result.platform, projectID: result.projectID, recovered: true, err: nil}
}

func writeDownloadFailureOutput(cmd *cobra.Command, deps addDeps, platformValue models.Platform, projectID string) error {
	colorMode := colorModeForOutput(cmd.OutOrStdout())
	retryCount := retryCountForClient(platform.PreferredDownloadClient(deps.clients))
	headline := renderFinalErrorLine(colorMode, downloadFailedSummary(platformValue, projectID))
	details := downloadFailedDetails(platformValue, retryCount)
	combined := fmt.Sprintf("%s\n\n%s", headline, details)
	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{combined})
}
