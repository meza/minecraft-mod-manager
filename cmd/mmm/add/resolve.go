package add

import (
	"context"
	"errors"
	"fmt"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
)

func resolveRemoteModWithSpan(inputs addResolveInputs) (resolvedRemoteMod, error) {
	resolveCtx, resolveSpan := perf.StartSpan(inputs.ctx, "app.command.add.stage.resolve",
		perf.WithAttributes(
			attribute.String("platform", string(inputs.platformValue)),
			attribute.String("project_id", inputs.projectID),
			attribute.String("execution_mode", inputs.mode.String()),
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
	return resolvedRemoteMod{
		platform:  inputs.platformValue,
		projectID: inputs.projectID,
	}, err
}

func fetchRemoteModOnce(ctx context.Context, inputs addResolveInputs) (platform.RemoteMod, error) {
	attemptCtx, attemptSpan := perf.StartSpan(ctx, "app.command.add.resolve.attempt",
		perf.WithAttributes(
			attribute.Int("attempt", 0),
			attribute.String("source", "cli"),
			attribute.String("platform", string(inputs.platformValue)),
			attribute.String("project_id", inputs.projectID),
			attribute.String("execution_mode", inputs.mode.String()),
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

func normalizeRemoteModFileName(remoteMod platform.RemoteMod) (platform.RemoteMod, error) {
	normalizedFileName, err := modfilename.Normalize(remoteMod.FileName)
	if err != nil {
		message := i18n.T("cmd.add.error.invalid_filename_remote", &i18n.Tvars{
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

func handleResolveFailure(cmd *cobra.Command, runState addRunState, deps addDeps, platformValue models.Platform, projectID string, err error) recoveryOutcome {
	var notFound *platform.ModNotFoundError
	var noCompatible *platform.NoCompatibleFileError

	switch {
	case errors.As(err, &notFound):
		return handleRecoveryPrompt(cmd, runState, deps, recoveryReasonNotFound, platformValue, projectID, err)
	case errors.As(err, &noCompatible):
		return handleRecoveryPrompt(cmd, runState, deps, recoveryReasonNoCompatible, platformValue, projectID, err)
	default:
		return recoveryOutcome{platformValue: platformValue, projectID: projectID, recovered: false, err: err}
	}
}

func handleRecoveryPrompt(cmd *cobra.Command, runState addRunState, deps addDeps, reason recoveryReason, platformValue models.Platform, projectID string, err error) recoveryOutcome {
	if !runState.mode.IsInteractive() {
		if outputErr := writeUnattendedResolveFailure(cmd, runState, deps, reason, platformValue, projectID); outputErr != nil {
			return recoveryOutcome{platformValue: platformValue, projectID: projectID, recovered: false, err: outputErr}
		}
		return recoveryOutcome{platformValue: platformValue, projectID: projectID, recovered: false, err: clierrors.MarkHandled(err)}
	}

	result, promptErr := runRecoveryFlow(recoveryFlowInput{
		reason:      reason,
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

func writeUnattendedResolveFailure(cmd *cobra.Command, runState addRunState, deps addDeps, reason recoveryReason, platformValue models.Platform, projectID string) error {
	colorMode := colorModeForOutput(cmd.OutOrStdout())
	var lines []string
	switch reason {
	case recoveryReasonNotFound:
		summary := renderFinalErrorLine(colorMode, projectNotFoundUnattendedSummary(platformValue, projectID))
		hint := projectNotFoundHint(platformValue, projectID)
		lines = []string{summary, hint}
	case recoveryReasonNoCompatible:
		summary := renderFinalErrorLine(colorMode, noCompatibleSummary(runState.cfg.Loader.String(), runState.cfg.GameVersion))
		lines = []string{summary}
	default:
		return errors.New("unsupported add recovery reason")
	}
	return runOutputLines(cmd, deps, cmd.OutOrStdout(), lines)
}

func retryCountForClient(client httpclient.Doer) int {
	retrying, ok := client.(*httpclient.RLHTTPClient)
	if !ok {
		return 0
	}
	config := retrying.RetryConfig
	if config == nil {
		return httpclient.RetryConfig{MaxRetries: 3}.MaxRetries
	}
	return config.MaxRetries
}
