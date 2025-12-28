package add

import (
	"errors"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/modsetup"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
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
			useTUI:        input.useTUI,
		})
		return telemetryResult, true, err
	}
	return telemetry.CommandTelemetry{}, false, nil
}

func handleExistingInstall(input existingInstallInput) (telemetry.CommandTelemetry, error) {
	install, err := normalizeExistingInstallFileName(input)
	if err != nil {
		return addFailureTelemetry(input.platformValue, input.projectID, input.opts, input.useTUI, err), err
	}

	ensureResult, ensureErr := ensureExistingInstall(input, install)
	if ensureErr != nil {
		return addFailureTelemetry(input.platformValue, input.projectID, input.opts, input.useTUI, ensureErr), ensureErr
	}

	logEnsureResult(input.deps.logger, ensureResult.Reason, input.cfg, input.platformValue, input.projectID)
	return recordExistingInstallTelemetry(input, ensureResult.Reason), nil
}

func normalizeExistingInstallFileName(input existingInstallInput) (models.ModInstall, error) {
	normalizedFileName, err := modfilename.Normalize(input.install.FileName)
	if err != nil {
		message := i18n.T("cmd.add.error.invalid_filename_lock", i18n.Tvars{
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
	return installer.EnsureLockedFile(input.ctx, input.meta, input.cfg, install, platform.PreferredDownloadClient(input.deps.clients), nil)
}

func recordExistingInstallTelemetry(input existingInstallInput, reason modinstall.EnsureReason) telemetry.CommandTelemetry {
	if input.commandSpan != nil {
		input.commandSpan.AddEvent("app.command.add.outcome.already_exists", perf.WithEventAttributes(
			attribute.String("platform", string(input.platformValue)),
			attribute.String("project_id", input.projectID),
		))
	}
	input.deps.logger.Debug(i18n.T("cmd.add.debug.already_exists", i18n.Tvars{
		Data: &i18n.TData{
			"id":       input.projectID,
			"platform": input.platformValue,
		},
	}))
	return addExistingInstallTelemetry(input.platformValue, input.projectID, input.opts, input.useTUI, reason)
}

func logEnsureResult(log *logger.Logger, reason modinstall.EnsureReason, cfg models.ModsJSON, platformValue models.Platform, projectID string) {
	switch reason {
	case modinstall.EnsureReasonMissing:
		log.Log(i18n.T("cmd.install.download.missing", i18n.Tvars{
			Data: &i18n.TData{
				"name":     modNameForConfig(cfg, platformValue, projectID),
				"platform": platformValue,
			},
		}), logger.LogForce)
	case modinstall.EnsureReasonHashMismatch:
		log.Log(i18n.T("cmd.install.download.hash_mismatch", i18n.Tvars{
			Data: &i18n.TData{"name": modNameForConfig(cfg, platformValue, projectID)},
		}), logger.LogForce)
	}
}
