package scan

import (
	"context"
	"fmt"
	"io"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modsetup"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	tui "github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/cobra"
)

func colorModeForOutput(out io.Writer) tui.ColorMode {
	if tui.IsTerminalWriter(out) {
		return tui.ColorEnabled
	}
	return tui.ColorDisabled
}

type persistScanRequest struct {
	Context          context.Context
	Command          *cobra.Command
	Options          scanOptions
	Dependencies     scanDeps
	Metadata         config.Metadata
	SetupCoordinator *modsetup.SetupCoordinator
	Matches          []scanMatch
	Unsure           []scanUnsure
	Config           models.ModsJSON
	Lock             []models.ModInstall
	PreferPlatform   models.Platform
	ColorMode        tui.ColorMode
}

func persistScanMatchesIfRequested(request persistScanRequest) (telemetry.CommandTelemetry, error) {
	shouldPersist, err := confirmPersist(request.Options, request.Dependencies)
	if err != nil {
		return scanFailureTelemetry(err), err
	}
	if !shouldPersist {
		return scanSuccessTelemetry(request.PreferPlatform, request.Options.Add), nil
	}

	if len(request.Unsure) > 0 {
		if outputErr := request.Dependencies.output.Log(i18n.T("cmd.scan.persist_skipped_unsure", nil), output.LogQuiet); outputErr != nil {
			return scanFailureTelemetry(outputErr), outputErr
		}
		return scanSuccessTelemetry(request.PreferPlatform, request.Options.Add), nil
	}

	persisted, err := persistScanMatches(
		request.Context,
		request.Command,
		request.Metadata,
		request.SetupCoordinator,
		request.Dependencies,
		request.Matches,
		request.Config,
		request.Lock,
	)
	if err != nil {
		return scanFailureTelemetry(err), err
	}
	if persisted {
		if outputErr := request.Dependencies.output.Log(messageWithIcon(tui.SuccessIcon(request.ColorMode), i18n.T("cmd.scan.persisted", nil)), output.LogQuiet); outputErr != nil {
			return scanFailureTelemetry(outputErr), outputErr
		}
	}

	return scanSuccessTelemetry(request.PreferPlatform, request.Options.Add), nil
}

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

func resolvePreferredPlatform(value string, out *output.Output) (models.Platform, error) {
	preferPlatform := normalizePlatform(value)
	if preferPlatform != models.MODRINTH && preferPlatform != models.CURSEFORGE {
		platformErr := fmt.Errorf("unknown platform: %s", value)
		if err := out.Error(platformErr.Error()); err != nil {
			return "", err
		}
		return "", platformErr
	}
	return preferPlatform, nil
}

func unmanagedFiles(files []string, lock []models.ModInstall) []string {
	unmanaged := make([]string, 0, len(files))
	for _, file := range files {
		if fileIsManaged(file, lock) {
			continue
		}
		unmanaged = append(unmanaged, file)
	}
	return unmanaged
}

func confirmPersist(opts scanOptions, deps scanDeps) (bool, error) {
	shouldPersist := opts.Add
	if !shouldPersist {
		if deps.prompter == nil {
			return false, nil
		}
		return deps.prompter.ConfirmAdd()
	}
	return shouldPersist, nil
}
