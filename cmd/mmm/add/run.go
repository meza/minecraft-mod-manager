package add

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/modsetup"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/tui"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
)

func runAdd(ctx context.Context, commandSpan *perf.Span, cmd *cobra.Command, opts addOptions, deps addDeps) (telemetry.CommandTelemetry, error) {
	runState, err := prepareAddRunState(ctx, cmd, opts, deps)
	if err != nil {
		return addFailureTelemetryWithoutArgs(runState.useTUI, err), err
	}
	colorMode := colorModeForOutput(cmd.OutOrStdout())
	platformValue, projectID := normalizedAddIdentifiers(opts)
	if telemetryResult, handled, existingInstallErr := handleExistingInstallIfPresent(existingInstallCheckInput{
		ctx:           ctx,
		commandSpan:   commandSpan,
		meta:          runState.meta,
		cfg:           runState.cfg,
		lock:          runState.lock,
		platformValue: platformValue,
		projectID:     projectID,
		opts:          opts,
		deps:          deps,
		useTUI:        runState.useTUI,
	}); handled {
		return telemetryResult, existingInstallErr
	}
	resolved, remoteMod, err := resolveAndEnsureRemoteMod(resolveAndEnsureInputs{
		ctx:           ctx,
		commandSpan:   commandSpan,
		meta:          runState.meta,
		cfg:           runState.cfg,
		opts:          opts,
		platformValue: platformValue,
		projectID:     projectID,
		deps:          deps,
		useTUI:        runState.useTUI,
		in:            cmd.InOrStdin(),
		out:           cmd.OutOrStdout(),
	})
	if err != nil {
		return addFailureTelemetry(resolved.platform, resolved.projectID, opts, runState.useTUI, err), err
	}
	return finalizeAddWithResolved(ctx, runState, resolved, remoteMod, opts, deps, colorMode)
}

func finalizeAddWithResolved(ctx context.Context, runState addRunState, resolved resolvedRemoteMod, remoteMod platform.RemoteMod, opts addOptions, deps addDeps, colorMode tui.ColorMode) (telemetry.CommandTelemetry, error) {
	return finalizeAdd(finalizeAddInput{
		ctx:              ctx,
		meta:             runState.meta,
		cfg:              runState.cfg,
		lock:             runState.lock,
		remoteMod:        remoteMod,
		resolvedPlatform: resolved.platform,
		resolvedID:       resolved.projectID,
		opts:             opts,
		setupCoordinator: runState.setupCoordinator,
		logger:           deps.logger,
		output:           deps.output,
		useTUI:           runState.useTUI,
		colorMode:        colorMode,
	})
}

func prepareAddConfig(ctx context.Context, opts addOptions, meta config.Metadata, setupCoordinator *modsetup.SetupCoordinator) (models.ModsJSON, []models.ModInstall, error) {
	prepareCtx, prepareSpan := perf.StartSpan(ctx, "app.command.add.stage.prepare", perf.WithAttributes(attribute.String("config_path", opts.ConfigPath)))
	cfg, lock, err := setupCoordinator.EnsureConfigAndLock(prepareCtx, meta, modsetup.EnsureConfigOptions{Quiet: opts.Quiet})
	prepareSpan.SetAttributes(attribute.Bool("success", err == nil))
	prepareSpan.End()
	return cfg, lock, err
}

func prepareAddRunState(ctx context.Context, cmd *cobra.Command, opts addOptions, deps addDeps) (addRunState, error) {
	quietMode := tui.QuietDisabled
	if opts.Quiet {
		quietMode = tui.QuietEnabled
	}
	runState := addRunState{
		meta:             config.NewMetadata(opts.ConfigPath),
		useTUI:           tui.ShouldUseTUI(quietMode, cmd.InOrStdin(), cmd.OutOrStdout()),
		setupCoordinator: modsetup.NewSetupCoordinator(deps.fs, deps.minecraftClient, modsetup.Downloader(deps.downloader)),
	}

	cfg, lock, err := prepareAddConfig(ctx, opts, runState.meta, runState.setupCoordinator)
	if err != nil {
		return runState, err
	}

	runState.cfg = cfg
	runState.lock = lock
	return runState, nil
}

func ensureRemoteMod(ctx context.Context, meta config.Metadata, cfg models.ModsJSON, remoteMod platform.RemoteMod, resolvedPlatform models.Platform, resolvedID string, deps addDeps) (modinstall.EnsureResult, error) {
	downloadCtx, downloadSpan := perf.StartSpan(ctx, "app.command.add.stage.download",
		perf.WithAttributes(
			attribute.String("url", remoteMod.DownloadURL),
			attribute.String("platform", string(resolvedPlatform)),
			attribute.String("project_id", resolvedID),
			attribute.String("file_name", remoteMod.FileName),
		),
	)

	// For idempotency, avoid re-downloading when the remote file already exists locally and the SHA-1 matches.
	ensureInstaller := modinstall.NewInstaller(deps.fs, modinstall.Downloader(deps.downloader))
	destination := filepath.Join(meta.ModsFolderPath(cfg), remoteMod.FileName)
	ensureResult, err := ensureInstaller.EnsureLockedFile(downloadCtx, meta, cfg, models.ModInstall{
		FileName:    remoteMod.FileName,
		Hash:        remoteMod.Hash,
		DownloadURL: remoteMod.DownloadURL,
	}, platform.PreferredDownloadClient(deps.clients), nil)
	if err != nil {
		if message, handled := integrityErrorMessage(err, remoteMod.Name); handled {
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

func persistAdd(input addPersistInput) error {
	_, persistSpan := perf.StartSpan(input.ctx, "app.command.add.stage.persist",
		perf.WithAttributes(
			attribute.String("config_path", input.opts.ConfigPath),
			attribute.String("platform", string(input.resolvedPlatform)),
			attribute.String("project_id", input.resolvedID),
		),
	)
	_, err := input.setupCoordinator.EnsurePersisted(input.ctx, input.meta, input.cfg, input.lock, input.resolvedPlatform, input.resolvedID, input.remoteMod, modsetup.EnsurePersistOptions{
		Version:              input.opts.Version,
		AllowVersionFallback: input.opts.AllowVersionFallback,
	})
	if err != nil {
		persistSpan.SetAttributes(attribute.Bool("success", false))
		persistSpan.End()
		return err
	}
	persistSpan.SetAttributes(attribute.Bool("success", true))
	persistSpan.End()
	return nil
}

func finalizeAdd(input finalizeAddInput) (telemetry.CommandTelemetry, error) {
	if err := persistAdd(addPersistInput{
		ctx:              input.ctx,
		meta:             input.meta,
		cfg:              input.cfg,
		lock:             input.lock,
		remoteMod:        input.remoteMod,
		resolvedPlatform: input.resolvedPlatform,
		resolvedID:       input.resolvedID,
		opts:             input.opts,
		setupCoordinator: input.setupCoordinator,
	}); err != nil {
		return addFailureTelemetry(input.resolvedPlatform, input.resolvedID, input.opts, input.useTUI, err), err
	}

	if err := logAddSuccess(input.output, input.colorMode, input.remoteMod.Name, input.resolvedID, input.resolvedPlatform); err != nil {
		return addFailureTelemetry(input.resolvedPlatform, input.resolvedID, input.opts, input.useTUI, err), err
	}
	return addSuccessTelemetry(input.resolvedPlatform, input.resolvedID, input.opts, input.useTUI), nil
}

func logAddSuccess(out *output.Output, colorMode tui.ColorMode, modName string, resolvedID string, resolvedPlatform models.Platform) error {
	return out.Log(fmt.Sprintf("%s %s", tui.SuccessIcon(colorMode), i18n.T("cmd.add.success", i18n.Tvars{
		Data: &i18n.TData{
			"name":     modName,
			"id":       resolvedID,
			"platform": resolvedPlatform,
		},
	})), output.LogForce)
}
