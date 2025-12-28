package update

import (
	"context"
	"errors"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var errUpdateFailures = errors.New("one or more mods failed to update")
var errUnmanagedFiles = errors.New("unmanaged files in mods folder")

func runUpdate(ctx context.Context, cmd *cobra.Command, opts updateOptions, deps updateDeps) (updateCounts, error) {
	if err := ensureInstallForUpdate(ctx, cmd, opts, deps); err != nil {
		return updateCounts{}, err
	}

	updateContext, err := loadUpdateContext(ctx, cmd, opts, deps)
	if err != nil {
		return updateCounts{}, err
	}

	candidates := updateCandidates(updateContext.cfg)
	outcomes, err := processCandidates(
		ctx,
		updateContext.meta,
		updateContext.cfg,
		updateContext.lock,
		candidates,
		deps,
		updateContext.colorMode,
	)
	if err != nil && outcomes == nil {
		return updateCounts{}, err
	}
	counts := applyUpdateOutcomes(deps.logger, outcomes, &updateContext.cfg, updateContext.lock)

	reportNoUpdatesIfNeeded(deps.logger, counts, updateContext.colorMode)

	persistContext := ctx
	if isContextCancellation(err) {
		persistContext = context.WithoutCancel(ctx)
	}
	if persistErr := persistUpdateConfig(persistContext, deps, updateContext); persistErr != nil {
		return counts, persistErr
	}

	if err != nil {
		return counts, err
	}

	if counts.failed > 0 {
		return counts, errUpdateFailures
	}

	return counts, nil
}

func ensureInstallForUpdate(ctx context.Context, cmd *cobra.Command, opts updateOptions, deps updateDeps) error {
	installResult, err := deps.install(ctx, cmd, opts.ConfigPath, opts.Quiet, opts.Debug)
	if err != nil {
		return err
	}
	if installResult.UnmanagedFound {
		deps.logger.Error(i18n.T("cmd.update.error.unmanaged_found"))
		return errUnmanagedFiles
	}
	return nil
}

func loadUpdateContext(ctx context.Context, cmd *cobra.Command, opts updateOptions, deps updateDeps) (updateContext, error) {
	meta := config.NewMetadata(opts.ConfigPath)

	cfg, err := config.ReadConfig(ctx, deps.fs, meta)
	if err != nil {
		return updateContext{}, err
	}

	lock, err := config.ReadLock(ctx, deps.fs, meta)
	if err != nil {
		return updateContext{}, err
	}

	colorMode := tui.ColorDisabled
	if tui.IsTerminalWriter(cmd.OutOrStdout()) {
		colorMode = tui.ColorEnabled
	}

	return updateContext{
		meta:      meta,
		cfg:       cfg,
		lock:      lock,
		colorMode: colorMode,
	}, nil
}

func reportNoUpdatesIfNeeded(log *logger.Logger, counts updateCounts, colorMode tui.ColorMode) {
	if counts.updated == 0 && counts.failed == 0 {
		log.Log(messageWithIcon(tui.SuccessIcon(colorMode), i18n.T("cmd.update.no_updates")), logger.LogForce)
	}
}

func persistUpdateConfig(ctx context.Context, deps updateDeps, updateContext updateContext) error {
	if err := config.WriteLock(ctx, deps.fs, updateContext.meta, updateContext.lock); err != nil {
		return err
	}
	if err := config.WriteConfig(ctx, deps.fs, updateContext.meta, updateContext.cfg); err != nil {
		return err
	}
	return nil
}

func updateCandidates(cfg models.ModsJSON) []modUpdateCandidate {
	candidates := make([]modUpdateCandidate, 0, len(cfg.Mods))
	for i := range cfg.Mods {
		candidates = append(candidates, modUpdateCandidate{
			ConfigIndex: i,
			Mod:         cfg.Mods[i],
		})
	}
	return candidates
}

func processCandidates(
	ctx context.Context,
	meta config.Metadata,
	cfg models.ModsJSON,
	lock []models.ModInstall,
	candidates []modUpdateCandidate,
	deps updateDeps,
	colorMode tui.ColorMode,
) ([]modUpdateOutcome, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	outcomes := make([]modUpdateOutcome, len(candidates))
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(defaultUpdateMaxConcurrency)

	for _, candidate := range candidates {
		candidate := candidate
		group.Go(func() error {
			outcome := processMod(groupCtx, meta, cfg, lock, candidate, deps, colorMode)
			outcomes[outcome.ConfigIndex] = outcome
			if err := groupCtx.Err(); err != nil {
				return err
			}
			return nil
		})
	}

	err := group.Wait()
	return outcomes, err
}

func applyUpdateOutcomes(log *logger.Logger, outcomes []modUpdateOutcome, cfg *models.ModsJSON, lock []models.ModInstall) updateCounts {
	counts := updateCounts{}

	for _, outcome := range outcomes {
		for _, event := range outcome.LogEvents {
			switch event.Kind {
			case logEventKindLog:
				visibility := logger.LogQuiet
				if event.ForceShow {
					visibility = logger.LogForce
				}
				log.Log(event.Message, visibility)
			case logEventKindError:
				log.Error(event.Message)
			case logEventKindDebug:
				log.Debug(event.Message)
			}
		}

		if strings.TrimSpace(outcome.NewName) != "" && outcome.ConfigIndex >= 0 && outcome.ConfigIndex < len(cfg.Mods) {
			cfg.Mods[outcome.ConfigIndex].Name = outcome.NewName
		}

		if outcome.Error != nil {
			counts.failed++
			continue
		}

		if outcome.Updated {
			lock[outcome.LockIndex] = outcome.NewInstall
			counts.updated++
		}
	}

	return counts
}

func isContextCancellation(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
