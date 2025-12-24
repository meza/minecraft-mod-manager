package remove

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
)

type removeOptions struct {
	ConfigPath string
	Quiet      bool
	Debug      bool
	DryRun     bool
	Lookups    []string
}

type removeDeps struct {
	fs        afero.Fs
	logger    *logger.Logger
	telemetry func(telemetry.CommandTelemetry)
}

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <mods...>",
		Short: i18n.T("cmd.remove.short"),
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			ctx, span := perf.StartSpan(cmd.Context(), "app.command.remove")

			configPath, err := cmd.Flags().GetString("config")
			if err != nil {
				span.SetAttributes(attribute.Bool("success", false))
				span.End()
				return err
			}
			quiet, err := cmd.Flags().GetBool("quiet")
			if err != nil {
				span.SetAttributes(attribute.Bool("success", false))
				span.End()
				return err
			}
			debug, err := cmd.Flags().GetBool("debug")
			if err != nil {
				span.SetAttributes(attribute.Bool("success", false))
				span.End()
				return err
			}
			dryRun, err := cmd.Flags().GetBool("dry-run")
			if err != nil {
				span.SetAttributes(attribute.Bool("success", false))
				span.End()
				return err
			}

			log := logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), quiet, debug)

			deps := removeDeps{
				fs:        afero.NewOsFs(),
				logger:    log,
				telemetry: telemetry.RecordCommand,
			}

			removedCount, err := runRemove(ctx, removeOptions{
				ConfigPath: configPath,
				Quiet:      quiet,
				Debug:      debug,
				DryRun:     dryRun,
				Lookups:    args,
			}, deps)
			span.SetAttributes(attribute.Bool("success", err == nil))
			span.End()

			payload := telemetry.CommandTelemetry{
				Command:     "remove",
				Success:     err == nil,
				Error:       err,
				ExitCode:    0,
				Interactive: false,
				Arguments: map[string]interface{}{
					"dryRun": dryRun,
					"mods":   args,
				},
			}
			if err != nil {
				payload.ExitCode = 1
			} else {
				payload.Extra = map[string]interface{}{
					"numberOfMods": removedCount,
				}
			}
			deps.telemetry(payload)

			return err
		},
	}

	cmd.Flags().BoolP("dry-run", "n", false, i18n.T("cmd.remove.flag.dry_run"))

	return cmd
}

func runRemove(ctx context.Context, opts removeOptions, deps removeDeps) (int, error) {
	meta := config.NewMetadata(opts.ConfigPath)

	cfg, err := config.ReadConfig(ctx, deps.fs, meta)
	if err != nil {
		return 0, err
	}

	lock, err := readLockForRemove(ctx, deps.fs, meta, opts.DryRun)
	if err != nil {
		return 0, err
	}

	matches, err := resolveModsToRemove(opts.Lookups, cfg)
	if err != nil {
		return 0, err
	}

	if opts.DryRun {
		deps.logger.Log("Running in dry-run mode. Nothing will actually be removed.", false)
	}

	removedCount := 0
	for _, mod := range matches {
		removed, err := removeMod(ctx, meta, &cfg, &lock, mod, opts, deps)
		if err != nil {
			return removedCount, err
		}
		if removed {
			removedCount++
		}
	}

	return removedCount, nil
}

func removeMod(ctx context.Context, meta config.Metadata, cfg *models.ModsJSON, lock *[]models.ModInstall, mod models.Mod, opts removeOptions, deps removeDeps) (bool, error) {
	if opts.DryRun {
		deps.logger.Log(fmt.Sprintf("Would have removed %s", mod.Name), false)
		return false, nil
	}

	if err := removeLockEntry(ctx, meta, cfg, lock, mod, deps); err != nil {
		return false, err
	}
	if err := removeConfigEntry(ctx, meta, cfg, mod, deps); err != nil {
		return false, err
	}

	deps.logger.Log(fmt.Sprintf("✅ Removed %s", mod.Name), false)
	return true, nil
}

func removeLockEntry(ctx context.Context, meta config.Metadata, cfg *models.ModsJSON, lock *[]models.ModInstall, mod models.Mod, deps removeDeps) error {
	lockIndex := lockIndexFor(mod, *lock)
	if lockIndex < 0 {
		return nil
	}

	normalizedFileName, err := modfilename.Normalize((*lock)[lockIndex].FileName)
	if err != nil {
		deps.logger.Error(i18n.T("cmd.remove.error.invalid_filename_lock", i18n.Tvars{
			Data: &i18n.TData{
				"name": mod.Name,
				"file": modfilename.Display((*lock)[lockIndex].FileName),
			},
		}))
	} else {
		installedPath := filepath.Join(meta.ModsFolderPath(*cfg), normalizedFileName)
		if err := removeFileForce(deps.fs, installedPath); err != nil {
			return err
		}
	}

	*lock = append((*lock)[:lockIndex], (*lock)[lockIndex+1:]...)
	return config.WriteLock(ctx, deps.fs, meta, *lock)
}

func removeConfigEntry(ctx context.Context, meta config.Metadata, cfg *models.ModsJSON, mod models.Mod, deps removeDeps) error {
	cfgIndex := configIndexFor(mod, cfg.Mods)
	if cfgIndex < 0 {
		return nil
	}

	cfg.Mods = append(cfg.Mods[:cfgIndex], cfg.Mods[cfgIndex+1:]...)
	return config.WriteConfig(ctx, deps.fs, meta, *cfg)
}

func readLockForRemove(ctx context.Context, fs afero.Fs, meta config.Metadata, dryRun bool) ([]models.ModInstall, error) {
	if !dryRun {
		return config.EnsureLock(ctx, fs, meta)
	}

	lockPath := meta.LockPath()
	exists, err := afero.Exists(fs, lockPath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return []models.ModInstall{}, nil
	}
	return config.ReadLock(ctx, fs, meta)
}

func resolveModsToRemove(lookups []string, cfg models.ModsJSON) ([]models.Mod, error) {
	matches := make([]models.Mod, 0)
	seen := make(map[string]bool)

	for _, lookup := range lookups {
		pattern := strings.ToLower(strings.TrimSpace(lookup))
		if pattern == "" {
			continue
		}

		if _, err := filepath.Match(pattern, ""); err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", lookup, err)
		}

		for _, mod := range cfg.Mods {
			key := string(mod.Type) + ":" + mod.ID
			if seen[key] {
				continue
			}

			ok := globMatches(pattern, strings.ToLower(mod.ID))
			if ok {
				seen[key] = true
				matches = append(matches, mod)
				continue
			}

			ok = globMatches(pattern, strings.ToLower(mod.Name))
			if ok {
				seen[key] = true
				matches = append(matches, mod)
				continue
			}
		}
	}

	return matches, nil
}

func globMatches(pattern string, value string) bool {
	ok, err := filepath.Match(pattern, value)
	if err != nil {
		return false
	}
	return ok
}

func lockIndexFor(mod models.Mod, lock []models.ModInstall) int {
	for i := range lock {
		if lock[i].Type == mod.Type && lock[i].ID == mod.ID {
			return i
		}
	}
	return -1
}

func configIndexFor(mod models.Mod, mods []models.Mod) int {
	for i := range mods {
		if mods[i].Type == mod.Type && mods[i].ID == mod.ID {
			return i
		}
	}
	return -1
}

func removeFileForce(fs afero.Fs, path string) error {
	if err := fs.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}
