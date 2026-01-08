package scan

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modsetup"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/cobra"
)

func scanFailureTelemetry(err error) telemetry.CommandTelemetry {
	return telemetry.CommandTelemetry{Command: "scan", Success: false, ExitCode: 1, Error: err}
}

func scanSuccessTelemetry(preferPlatform models.Platform, add bool) telemetry.CommandTelemetry {
	return telemetry.CommandTelemetry{
		Command:  "scan",
		Success:  true,
		ExitCode: 0,
		Arguments: map[string]interface{}{
			"prefer": preferPlatform,
			"add":    add,
		},
	}
}

func scanSuccessTelemetryWithoutArgs() telemetry.CommandTelemetry {
	return telemetry.CommandTelemetry{
		Command:  "scan",
		Success:  true,
		ExitCode: 0,
	}
}

func persistScanMatches(ctx context.Context, cmd *cobra.Command, input scanExecutionInput, matches []scanMatch) ([]scanMatch, error) {
	if len(matches) == 0 {
		return nil, nil
	}
	if input.setupCoordinator == nil {
		return nil, errors.New("missing setup coordinator")
	}

	cfg := input.cfg
	lock := input.lock
	changedConfig := false
	changedLock := false
	added := make([]scanMatch, 0, len(matches))
	failed := make([]string, 0)

	for _, match := range matches {
		outcome, err := input.setupCoordinator.UpsertConfigAndLock(cfg, lock, match.Platform, match.ProjectID, platformRemoteMod(match), modsetup.EnsurePersistOptions{})
		if err != nil {
			failed = append(failed, renderScanPersistFailureLine(colorModeForOutput(cmd.OutOrStdout()), match.FileName))
			continue
		}
		cfg = outcome.Config
		lock = outcome.Lock
		if outcome.Result.ConfigAdded || outcome.Result.ConfigUpdated {
			changedConfig = true
		}
		if outcome.Result.LockAdded || outcome.Result.LockUpdated {
			changedLock = true
		}
		added = append(added, match)
	}

	if changedConfig {
		if err := config.WriteConfig(ctx, input.deps.fs, input.meta, cfg); err != nil {
			return added, err
		}
	}
	if changedLock {
		if err := config.WriteLock(ctx, input.deps.fs, input.meta, lock); err != nil {
			return added, err
		}
	}

	if len(failed) > 0 {
		combined := strings.Join(failed, "\n")
		if outputErr := runOutputLines(cmd, input.deps, cmd.OutOrStdout(), []string{combined}); outputErr != nil {
			return added, outputErr
		}
	}

	return added, nil
}

func platformRemoteMod(match scanMatch) platform.RemoteMod {
	return platform.RemoteMod{
		Name:        match.Name,
		FileName:    match.FileName,
		Hash:        match.Hash,
		ReleaseDate: match.ReleaseDate,
		DownloadURL: match.DownloadURL,
	}
}

func renderScanPersistFailureLine(colorMode view.ColorMode, fileName string) string {
	return fmt.Sprintf("%s %s", view.ErrorIcon(colorMode), i18n.T("cmd.scan.persist_failed", &i18n.Tvars{
		Data: &i18n.TData{"file": fileName},
	}))
}
