package add

import (
	"context"
	"errors"
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/tui"
	"go.opentelemetry.io/otel/attribute"
)

func resolveRemoteModWithSpan(inputs addResolveInputs) (resolvedRemoteMod, error) {
	resolveCtx, resolveSpan := perf.StartSpan(inputs.ctx, "app.command.add.stage.resolve",
		perf.WithAttributes(
			attribute.String("platform", string(inputs.platformValue)),
			attribute.String("project_id", inputs.projectID),
			attribute.Bool("use_tui", inputs.useTUI),
			attribute.Bool("quiet", inputs.opts.Quiet),
		),
	)
	resolved, fetchErr := resolveRemoteMod(resolveCtx, inputs)
	resolveSpan.SetAttributes(
		attribute.Bool("success", fetchErr == nil),
		attribute.String("resolved_platform", string(resolved.platform)),
		attribute.String("resolved_project_id", resolved.projectID),
	)
	resolveSpan.End()
	return resolved, fetchErr
}

func resolveAndEnsureRemoteMod(inputs resolveAndEnsureInputs) (resolvedRemoteMod, platform.RemoteMod, error) {
	resolved, fetchErr := resolveRemoteModWithSpan(addResolveInputs{
		ctx:           inputs.ctx,
		commandSpan:   inputs.commandSpan,
		cfg:           inputs.cfg,
		opts:          inputs.opts,
		platformValue: inputs.platformValue,
		projectID:     inputs.projectID,
		deps:          inputs.deps,
		useTUI:        inputs.useTUI,
		in:            inputs.in,
		out:           inputs.out,
	})
	if fetchErr != nil {
		return resolved, platform.RemoteMod{}, fetchErr
	}

	remoteMod := resolved.remoteMod
	remoteMod, err := normalizeRemoteModFileName(remoteMod)
	if err != nil {
		return resolved, platform.RemoteMod{}, err
	}

	_, err = ensureRemoteMod(inputs.ctx, inputs.meta, inputs.cfg, remoteMod, resolved.platform, resolved.projectID, inputs.deps)
	if err != nil {
		return resolved, platform.RemoteMod{}, err
	}

	return resolved, remoteMod, nil
}

func normalizeRemoteModFileName(remoteMod platform.RemoteMod) (platform.RemoteMod, error) {
	normalizedFileName, err := modfilename.Normalize(remoteMod.FileName)
	if err != nil {
		message := i18n.T("cmd.add.error.invalid_filename_remote", i18n.Tvars{
			Data: &i18n.TData{
				"name": remoteMod.Name,
				"file": modfilename.Display(remoteMod.FileName),
			},
		})
		return platform.RemoteMod{}, errors.New(message)
	}
	remoteMod.FileName = normalizedFileName
	return remoteMod, nil
}

func resolveRemoteMod(ctx context.Context, inputs addResolveInputs) (resolvedRemoteMod, error) {
	if err := inputs.deps.logger.Debug(fmt.Sprintf(
		"fetching %s/%s (loader=%s, gameVersion=%s, fallback=%t, fixedVersion=%s)",
		inputs.platformValue,
		inputs.projectID,
		inputs.cfg.Loader,
		inputs.cfg.GameVersion,
		inputs.opts.AllowVersionFallback,
		inputs.opts.Version,
	)); err != nil {
		return resolvedRemoteMod{
			platform:  inputs.platformValue,
			projectID: inputs.projectID,
		}, err
	}

	remote, err := fetchRemoteModOnce(ctx, inputs)
	if err == nil {
		return resolvedRemoteMod{
			remoteMod: remote,
			platform:  inputs.platformValue,
			projectID: inputs.projectID,
		}, nil
	}

	if logErr := logFetchFailure(inputs.deps.logger, inputs.platformValue, inputs.projectID, err); logErr != nil {
		return resolvedRemoteMod{
			platform:  inputs.platformValue,
			projectID: inputs.projectID,
		}, logErr
	}
	return resolveRemoteModFromError(ctx, inputs, err)
}

func fetchRemoteModOnce(ctx context.Context, inputs addResolveInputs) (platform.RemoteMod, error) {
	attemptCtx, attemptSpan := perf.StartSpan(ctx, "app.command.add.resolve.attempt",
		perf.WithAttributes(
			attribute.Int("attempt", 0),
			attribute.String("source", "cli"),
			attribute.String("platform", string(inputs.platformValue)),
			attribute.String("project_id", inputs.projectID),
			attribute.Bool("use_tui", inputs.useTUI),
			attribute.Bool("quiet", inputs.opts.Quiet),
		),
	)
	remote, err := inputs.deps.fetchMod(attemptCtx, inputs.platformValue, inputs.projectID, platform.FetchOptions{
		AllowedReleaseTypes: inputs.cfg.DefaultAllowedReleaseTypes,
		GameVersion:         inputs.cfg.GameVersion,
		Loader:              inputs.cfg.Loader,
		AllowFallback:       inputs.opts.AllowVersionFallback,
		FixedVersion:        inputs.opts.Version,
	}, inputs.deps.clients)
	attemptSpan.SetAttributes(attribute.Bool("success", err == nil))
	if err != nil {
		attemptSpan.SetAttributes(attribute.String("error_type", fmt.Sprintf("%T", err)))
	}
	attemptSpan.End()
	return remote, err
}

func logFetchFailure(log *logger.Logger, platformValue models.Platform, projectID string, err error) error {
	if logErr := log.Debug(fmt.Sprintf("fetch failed for %s/%s: %v", platformValue, projectID, err)); logErr != nil {
		return logErr
	}
	if inner := errors.Unwrap(err); inner != nil {
		if logErr := log.Debug(fmt.Sprintf("fetch failure detail: %v", inner)); logErr != nil {
			return logErr
		}
	}
	return nil
}

func resolveRemoteModFromError(ctx context.Context, inputs addResolveInputs, err error) (resolvedRemoteMod, error) {
	var unknownPlatformError *platform.UnknownPlatformError
	if errors.As(err, &unknownPlatformError) {
		return resolveUnknownPlatform(ctx, inputs, unknownPlatformError)
	}

	var modNotFoundError *platform.ModNotFoundError
	if errors.As(err, &modNotFoundError) {
		return resolveModNotFound(ctx, inputs, err)
	}

	var noCompatibleFileError *platform.NoCompatibleFileError
	if errors.As(err, &noCompatibleFileError) {
		return resolveNoCompatibleFile(ctx, inputs, err)
	}

	return resolvedRemoteMod{
		platform:  inputs.platformValue,
		projectID: inputs.projectID,
	}, err
}

func resolveUnknownPlatform(ctx context.Context, inputs addResolveInputs, unknownPlatformError *platform.UnknownPlatformError) (resolvedRemoteMod, error) {
	if inputs.opts.Quiet || !inputs.useTUI {
		message := errorMessageForUnknownPlatform(unknownPlatformError.Platform)
		if err := inputs.deps.output.Error(message); err != nil {
			return resolvedRemoteMod{
				platform:  inputs.platformValue,
				projectID: inputs.projectID,
			}, err
		}
		return resolvedRemoteMod{
			platform:  inputs.platformValue,
			projectID: inputs.projectID,
		}, errors.New(message)
	}
	return resolveRemoteModWithTUI(ctx, inputs, addTUIStateUnknownPlatformSelect)
}

func resolveModNotFound(ctx context.Context, inputs addResolveInputs, err error) (resolvedRemoteMod, error) {
	if inputs.opts.Quiet || !inputs.useTUI {
		if outputErr := inputs.deps.output.Error(errorMessageForModNotFound(inputs.projectID, inputs.platformValue)); outputErr != nil {
			return resolvedRemoteMod{
				platform:  inputs.platformValue,
				projectID: inputs.projectID,
			}, outputErr
		}
		return resolvedRemoteMod{
			platform:  inputs.platformValue,
			projectID: inputs.projectID,
		}, err
	}
	return resolveRemoteModWithTUI(ctx, inputs, addTUIStateModNotFoundConfirm)
}

func resolveNoCompatibleFile(ctx context.Context, inputs addResolveInputs, err error) (resolvedRemoteMod, error) {
	if inputs.opts.Quiet || !inputs.useTUI {
		if outputErr := inputs.deps.output.Error(errorMessageForNoFile(inputs.projectID, inputs.platformValue)); outputErr != nil {
			return resolvedRemoteMod{
				platform:  inputs.platformValue,
				projectID: inputs.projectID,
			}, outputErr
		}
		return resolvedRemoteMod{
			platform:  inputs.platformValue,
			projectID: inputs.projectID,
		}, err
	}
	return resolveRemoteModWithTUI(ctx, inputs, addTUIStateNoFileConfirm)
}

func resolveRemoteModWithTUI(ctx context.Context, inputs addResolveInputs, initialState addTUIState) (resolvedRemoteMod, error) {
	baseResult := resolvedRemoteMod{
		platform:  inputs.platformValue,
		projectID: inputs.projectID,
	}
	if inputs.commandSpan != nil {
		inputs.commandSpan.AddEvent("app.command.add.tui.open", perf.WithEventAttributes(
			attribute.Int("initial_state", int(initialState)),
			attribute.String("platform", string(inputs.platformValue)),
			attribute.String("project_id", inputs.projectID),
		))
	}

	tuiCtx, tuiSpan := perf.StartSpan(ctx, "tui.add.session",
		perf.WithAttributes(
			attribute.String("platform", string(inputs.platformValue)),
			attribute.String("project_id", inputs.projectID),
			attribute.Int("initial_state", int(initialState)),
		),
	)
	attempt := 0
	model := newAddTUIModel(tuiCtx, tuiSpan, initialState, inputs.platformValue, inputs.projectID, inputs.cfg, buildAddTUIFetchCmd(tuiCtx, inputs, &attempt))

	if inputs.deps.runTea == nil {
		return baseResult, errors.New("missing add dependencies: runTea")
	}

	programResult, err := runAddTUIProgram(inputs.deps.runTea, model, tuiSpan, inputs.in, inputs.out)
	if err != nil {
		return baseResult, err
	}

	typed, ok := programResult.(addTUIModel)
	if !ok {
		return baseResult, errors.New("unexpected add TUI result model")
	}

	addResult, err := typed.result()
	if err != nil {
		return baseResult, err
	}
	return resolvedRemoteMod(addResult), nil
}

func buildAddTUIFetchCmd(tuiCtx context.Context, inputs addResolveInputs, attempt *int) func(models.Platform, string) tea.Cmd {
	return func(platformValue models.Platform, projectID string) tea.Cmd {
		return func() tea.Msg {
			*attempt += 1
			attemptNumber := *attempt
			attemptCtx, attemptSpan := perf.StartSpan(tuiCtx, "app.command.add.resolve.attempt",
				perf.WithAttributes(
					attribute.Int("attempt", attemptNumber),
					attribute.String("source", "tui"),
					attribute.String("platform", string(platformValue)),
					attribute.String("project_id", projectID),
					attribute.Bool("quiet", inputs.opts.Quiet),
				),
			)
			remote, err := inputs.deps.fetchMod(attemptCtx, platformValue, projectID, platform.FetchOptions{
				AllowedReleaseTypes: inputs.cfg.DefaultAllowedReleaseTypes,
				GameVersion:         inputs.cfg.GameVersion,
				Loader:              inputs.cfg.Loader,
				AllowFallback:       inputs.opts.AllowVersionFallback,
				FixedVersion:        inputs.opts.Version,
			}, inputs.deps.clients)
			attemptSpan.SetAttributes(attribute.Bool("success", err == nil))
			if err != nil {
				attemptSpan.SetAttributes(attribute.String("error_type", fmt.Sprintf("%T", err)))
			}
			attemptSpan.End()
			return addTUIFetchResultMsg{
				platform:  platformValue,
				projectID: projectID,
				remote:    remote,
				err:       err,
			}
		}
	}
}

func runAddTUIProgram(runTea func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error), model tea.Model, tuiSpan *perf.Span, in io.Reader, out io.Writer) (tea.Model, error) {
	result, err := runTea(model, tui.ProgramOptions(in, out)...)
	if err != nil {
		tuiSpan.SetAttributes(attribute.Bool("success", false))
		tuiSpan.End()
		return nil, err
	}
	tuiSpan.SetAttributes(attribute.Bool("success", true))
	tuiSpan.End()
	return result, nil
}
