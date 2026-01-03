package init

import (
	"context"
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/tui"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
)

func runInitCommand(ctx context.Context, cmd *cobra.Command, options initOptions, deps initDeps, meta config.Metadata) error {
	finalOptions, didUseTUI, err := runInit(ctx, cmd, options, deps, meta)
	if errors.Is(err, ErrInitCanceled) {
		err = nil
	}

	if deps.telemetry != nil {
		deps.telemetry(buildTelemetryPayload(finalOptions, didUseTUI, err))
	}

	return err
}

func runInit(ctx context.Context, cmd *cobra.Command, options initOptions, deps initDeps, meta config.Metadata) (initOptions, bool, error) {
	promptMode := tui.PromptEnabled
	if options.Unattended {
		promptMode = tui.PromptDisabled
	}
	shouldUseTUI := tui.ShouldUseTUI(promptMode, cmd.InOrStdin(), cmd.OutOrStdout())
	didUseTUI := false

	gameVersionMode := gameVersionUnattended
	if shouldUseTUI {
		gameVersionMode = gameVersionInteractive
	}
	options, err := normalizeGameVersion(ctx, options, deps, gameVersionMode)
	if err != nil {
		return options, didUseTUI, err
	}

	if shouldUseTUI {
		updated, launched, runErr := runInteractiveInitWithLaunchFlag(ctx, cmd, options, deps, meta)
		if runErr != nil {
			return options, launched, normalizeInitCancelError(runErr)
		}
		options = updated
		didUseTUI = launched
	}

	_, err = initWithDeps(ctx, options, deps)
	return options, didUseTUI, err
}

func runInteractiveInitWithLaunchFlag(ctx context.Context, cmd *cobra.Command, options initOptions, deps initDeps, meta config.Metadata) (initOptions, bool, error) {
	sessionCtx, sessionSpan := perf.StartSpan(ctx, "tui.init.session",
		perf.WithAttributes(
			attribute.Bool("provided_loader", options.Provided.Loader),
			attribute.Bool("provided_game_version", options.Provided.GameVersion),
			attribute.Bool("provided_release_types", options.Provided.ReleaseTypes),
			attribute.Bool("provided_mods_folder", options.Provided.ModsFolder),
		),
	)
	defer sessionSpan.End()

	model := NewModel(sessionCtx, sessionSpan, options, deps, meta)
	if model.state == done {
		return model.result, false, nil
	}

	result, err := deps.runTea(model, tui.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
	if err != nil {
		return options, true, err
	}

	finalResult, err := finalizeInteractiveResult(result)
	if err != nil {
		return options, true, err
	}

	return finalResult, true, nil
}

func finalizeInteractiveResult(result tea.Model) (initOptions, error) {
	var finalModel CommandModel
	switch typed := result.(type) {
	case *CommandModel:
		finalModel = *typed
	case CommandModel:
		finalModel = typed
	default:
		return initOptions{}, errors.New(i18n.T("cmd.init.error.interactive.failed", nil))
	}

	if finalModel.err != nil {
		return initOptions{}, finalModel.err
	}

	if finalModel.state != done {
		return initOptions{}, ErrInitCanceled
	}

	return finalModel.result, nil
}

func normalizeInitCancelError(err error) error {
	if errors.Is(err, ErrInitCanceled) {
		return ErrInitCanceled
	}
	return err
}
