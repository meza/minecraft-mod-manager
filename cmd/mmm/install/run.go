package install

import (
	"context"
	"errors"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/tui"
	"github.com/spf13/cobra"
)

var errUnresolvedFiles = errors.New("unresolved files in mods folder")
var errInstallFailures = errors.New("one or more mods failed to install")

func runInstall(ctx context.Context, cmd *cobra.Command, opts installOptions, deps installDeps) (Result, error) {
	meta := config.NewMetadata(opts.ConfigPath)

	cfg, lock, err := loadInstallConfig(ctx, deps, meta)
	if err != nil {
		return Result{}, err
	}

	colorize := tui.IsTerminalWriter(cmd.OutOrStdout())
	colorMode := tui.ColorDisabled
	if colorize {
		colorMode = tui.ColorEnabled
	}

	preflight, err := preflightInstall(ctx, meta, cfg, lock, deps, colorize)
	if err != nil {
		return Result{}, err
	}

	if mkdirErr := deps.fs.MkdirAll(meta.ModsFolderPath(cfg), 0755); mkdirErr != nil {
		return Result{}, mkdirErr
	}

	configured, err := installConfiguredMods(installConfiguredInputs{
		ctx:      ctx,
		meta:     meta,
		cfg:      cfg,
		lock:     lock,
		deps:     deps,
		colorize: colorize,
	})
	if err != nil {
		return Result{}, err
	}

	if err := persistInstallConfig(ctx, deps, meta, configured); err != nil {
		return Result{}, err
	}

	if configured.failedCount > 0 {
		return Result{InstalledCount: len(configured.cfg.Mods), UnmanagedFound: preflight.unmanagedFound}, errInstallFailures
	}

	if err := deps.output.Log(messageWithIcon(tui.SuccessIcon(colorMode), i18n.T("cmd.install.success")), output.LogForce); err != nil {
		return Result{}, err
	}
	return Result{InstalledCount: len(configured.cfg.Mods), UnmanagedFound: preflight.unmanagedFound}, nil
}

func loadInstallConfig(ctx context.Context, deps installDeps, meta config.Metadata) (models.ModsJSON, []models.ModInstall, error) {
	cfg, err := config.ReadConfig(ctx, deps.fs, meta)
	if err != nil {
		return models.ModsJSON{}, nil, err
	}
	lock, err := config.EnsureLock(ctx, deps.fs, meta)
	if err != nil {
		return models.ModsJSON{}, nil, err
	}
	return cfg, lock, nil
}

func persistInstallConfig(ctx context.Context, deps installDeps, meta config.Metadata, configured installConfiguredOutcome) error {
	if err := config.WriteLock(ctx, deps.fs, meta, configured.lock); err != nil {
		return err
	}
	if err := config.WriteConfig(ctx, deps.fs, meta, configured.cfg); err != nil {
		return err
	}
	return nil
}

func installConfiguredMods(input installConfiguredInputs) (installConfiguredOutcome, error) {
	failedCount := 0
	cfg := input.cfg
	lock := input.lock

	for i := range cfg.Mods {
		mod := cfg.Mods[i]
		version := modVersionLabel(mod)
		if err := input.deps.logger.Debug(i18n.T("cmd.install.debug.checking", i18n.Tvars{
			Data: &i18n.TData{
				"name":     mod.Name,
				"version":  version,
				"platform": mod.Type,
			},
		})); err != nil {
			return installConfiguredOutcome{}, err
		}

		outcome, err := installMod(installModInputs{
			ctx:      input.ctx,
			meta:     input.meta,
			cfg:      cfg,
			lock:     lock,
			mod:      mod,
			deps:     input.deps,
			colorize: input.colorize,
		})
		if err != nil {
			return installConfiguredOutcome{}, err
		}
		if outcome.failed {
			failedCount++
			continue
		}
		if outcome.newName != "" {
			cfg.Mods[i].Name = outcome.newName
		}
		if outcome.lockEntry != nil {
			lock = append(lock, *outcome.lockEntry)
		}
	}

	return installConfiguredOutcome{
		cfg:         cfg,
		lock:        lock,
		failedCount: failedCount,
	}, nil
}
