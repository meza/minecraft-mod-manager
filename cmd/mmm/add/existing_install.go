package add

import (
	"errors"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/modsetup"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"go.opentelemetry.io/otel/attribute"
)

func findLockInstall(lock []models.ModInstall, platformValue models.Platform, projectID string) (models.ModInstall, bool) {
	for i := range lock {
		if lock[i].Type == platformValue && lock[i].ID == projectID {
			return lock[i], true
		}
	}
	return models.ModInstall{}, false
}

func handleExistingInstallIfPresent(input existingInstallCheckInput) (telemetry.CommandTelemetry, bool, error) {
	install, installFound := findLockInstall(input.lock, input.platformValue, input.projectID)
	if modsetup.ModExists(input.cfg, input.platformValue, input.projectID) && installFound {
		telemetryResult, err := handleExistingInstall(existingInstallInput{
			ctx:           input.ctx,
			commandSpan:   input.commandSpan,
			meta:          input.meta,
			cfg:           input.cfg,
			install:       install,
			platformValue: input.platformValue,
			projectID:     input.projectID,
			opts:          input.opts,
			deps:          input.deps,
			mode:          input.mode,
			in:            input.in,
			out:           input.out,
		})
		return telemetryResult, true, err
	}
	return telemetry.CommandTelemetry{}, false, nil
}

func handleExistingInstall(input existingInstallInput) (telemetry.CommandTelemetry, error) {
	install, err := normalizeExistingInstallFileName(input)
	if err != nil {
		return addFailureTelemetry(input.platformValue, input.projectID, input.opts, input.mode.String(), input.mode.IsInteractive(), err), err
	}

	ensureResult, ensureErr := ensureExistingInstall(input, install)
	if ensureErr != nil {
		return addFailureTelemetry(input.platformValue, input.projectID, input.opts, input.mode.String(), input.mode.IsInteractive(), ensureErr), ensureErr
	}

	if !input.opts.Quiet {
		colorMode := colorModeForOutput(input.out)
		message := renderAddSuccessLine(colorMode, modNameForConfig(input.cfg, input.platformValue, input.projectID), input.projectID, input.platformValue)
		if outputErr := runOutputLines(nil, input.deps, input.out, []string{message}); outputErr != nil {
			return addFailureTelemetry(input.platformValue, input.projectID, input.opts, input.mode.String(), input.mode.IsInteractive(), outputErr), outputErr
		}
	}

	telemetryPayload, err := recordExistingInstallTelemetry(input, ensureResult.Reason)
	if err != nil {
		return addFailureTelemetry(input.platformValue, input.projectID, input.opts, input.mode.String(), input.mode.IsInteractive(), err), err
	}
	return telemetryPayload, nil
}

func normalizeExistingInstallFileName(input existingInstallInput) (models.ModInstall, error) {
	normalizedFileName, err := modfilename.Normalize(input.install.FileName)
	if err != nil {
		message := i18n.T("cmd.add.error.invalid_filename_lock", &i18n.Tvars{
			Data: &i18n.TData{
				"name": modNameForConfig(input.cfg, input.platformValue, input.projectID),
				"file": modfilename.Display(input.install.FileName),
			},
		})
		return models.ModInstall{}, errors.New(message)
	}

	install := input.install
	install.FileName = normalizedFileName
	return install, nil
}

func ensureExistingInstall(input existingInstallInput, install models.ModInstall) (modinstall.EnsureResult, error) {
	installer := modinstall.NewInstaller(input.deps.fs, modinstall.Downloader(input.deps.downloader))
	client := platform.PreferredDownloadClient(input.deps.clients)
	if input.opts.Quiet || !input.mode.IsInteractive() {
		return installer.EnsureLockedFile(input.ctx, input.meta, input.cfg, install, client, nil)
	}

	colorMode := colorModeForOutput(input.out)
	model := newDownloadProgressModel(modNameForConfig(input.cfg, input.platformValue, input.projectID), input.projectID, input.platformValue, colorMode, func(sender httpclient.Sender) (modinstall.EnsureResult, error) {
		return installer.EnsureLockedFile(input.ctx, input.meta, input.cfg, install, client, sender)
	})

	result, err := runProgressProgram(model, view.ProgramOptions(input.in, input.out)...)
	if err != nil {
		return modinstall.EnsureResult{}, err
	}
	progressModel, ok := result.(*downloadProgressModel)
	if !ok {
		return modinstall.EnsureResult{}, errors.New("unexpected download progress model")
	}
	if progressModel.err != nil {
		return modinstall.EnsureResult{}, progressModel.err
	}
	return progressModel.result, nil
}

func recordExistingInstallTelemetry(input existingInstallInput, reason modinstall.EnsureReason) (telemetry.CommandTelemetry, error) {
	if input.commandSpan != nil {
		input.commandSpan.AddEvent("app.command.add.outcome.already_exists", perf.WithEventAttributes(
			attribute.String("platform", string(input.platformValue)),
			attribute.String("project_id", input.projectID),
		))
	}
	if err := input.deps.logger.Debug(i18n.T("cmd.add.debug.already_exists", &i18n.Tvars{
		Data: &i18n.TData{
			"id":       input.projectID,
			"platform": string(input.platformValue),
		},
	})); err != nil {
		return telemetry.CommandTelemetry{}, err
	}
	return addExistingInstallTelemetry(input.platformValue, input.projectID, input.opts, input.mode.String(), input.mode.IsInteractive(), reason), nil
}
