package change

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/meza/minecraft-mod-manager/cmd/mmm/test"
	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

func changeOptionsFromFlags(cmd *cobra.Command, args []string) (changeOptions, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return changeOptions{}, err
	}
	nonInteractive, err := cmd.Flags().GetBool("non-interactive")
	if err != nil {
		return changeOptions{}, err
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return changeOptions{}, err
	}
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return changeOptions{}, err
	}
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return changeOptions{}, err
	}

	return changeOptions{
		ConfigPath:     configPath,
		GameVersion:    resolveGameVersion(args),
		NonInteractive: nonInteractive,
		Quiet:          quiet,
		Debug:          debug,
		Force:          force,
	}, nil
}

func resolveGameVersion(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return "latest"
}

func applyChangeCommandErrorPolicy(cmd *cobra.Command, err error) {
	if err == nil {
		return
	}
	if clierrors.IsHandled(err) {
		cmd.SilenceErrors = true
	}
	cmd.SilenceUsage = true
}

func runChange(ctx context.Context, cmd *cobra.Command, opts changeOptions, deps changeDeps) (changeResult, error) {
	meta := config.NewMetadata(opts.ConfigPath)

	cfg, err := deps.readConfig(ctx, deps.fs, meta)
	if err != nil {
		return changeResult{ExitCode: 1}, err
	}

	testResult, err := runVersionCheck(ctx, opts, deps)
	if err != nil || testResult.ExitCode != 0 {
		return changeResult{
			TargetVersion:   testResult.TargetVersion,
			UnsupportedMods: testResult.UnsupportedMods,
			ExitCode:        testResult.ExitCode,
		}, err
	}

	if err := applyVersionChange(ctx, meta, cfg, testResult.TargetVersion, deps); err != nil {
		return changeResult{TargetVersion: testResult.TargetVersion, ExitCode: 1}, err
	}

	installResult, installErr := deps.installRunner(ctx, deps.installCmd, opts.ConfigPath, opts.Quiet, opts.Debug)

	return changeResult{
		TargetVersion:   testResult.TargetVersion,
		UnsupportedMods: testResult.UnsupportedMods,
		InstallResult:   installResult,
		ExitCode:        exitCodeForError(installErr),
	}, installErr
}

func removeInstalledMods(fs afero.Fs, meta config.Metadata, cfg models.ModsJSON, lock []models.ModInstall, remover func(afero.Fs, string) error) error {
	if len(cfg.Mods) == 0 || len(lock) == 0 {
		return nil
	}

	lockIndex := indexLockByMod(lock)
	modsDir := meta.ModsFolderPath(cfg)

	for _, mod := range cfg.Mods {
		lockEntry, ok := lockIndex[modKey(mod)]
		if !ok {
			continue
		}
		path := filepath.Join(modsDir, lockEntry.FileName)
		exists, err := afero.Exists(fs, path)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		if err := remover(fs, path); err != nil {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}

	return nil
}

func indexLockByMod(lock []models.ModInstall) map[string]models.ModInstall {
	index := make(map[string]models.ModInstall, len(lock))
	for _, entry := range lock {
		index[fmt.Sprintf("%s:%s", entry.Type, entry.ID)] = entry
	}
	return index
}

func modKey(mod models.Mod) string {
	return fmt.Sprintf("%s:%s", mod.Type, mod.ID)
}

func exitCodeForError(err error) int {
	if err == nil {
		return 0
	}
	if exitCoder, ok := err.(interface{ ExitCode() int }); ok {
		return exitCoder.ExitCode()
	}
	return 1
}

func runVersionCheck(ctx context.Context, opts changeOptions, deps changeDeps) (test.Result, error) {
	testOpts := test.Options{
		ConfigPath:     opts.ConfigPath,
		GameVersion:    opts.GameVersion,
		NonInteractive: opts.NonInteractive,
		Quiet:          opts.Quiet,
		Debug:          opts.Debug,
		Force:          opts.Force,
	}

	testResult, err := deps.testRunner(ctx, deps.testCmd, testOpts, deps.testDeps, deps.colorMode)
	if err == nil && testResult.ExitCode == 0 {
		return testResult, nil
	}

	exitCode := testResult.ExitCode
	if exitCode == 0 {
		exitCode = exitCodeForError(err)
	}
	testResult.ExitCode = exitCode
	return testResult, err
}

func applyVersionChange(ctx context.Context, meta config.Metadata, cfg models.ModsJSON, targetVersion string, deps changeDeps) error {
	lock, err := deps.ensureLock(ctx, deps.fs, meta)
	if err != nil {
		return err
	}

	if err := removeInstalledMods(deps.fs, meta, cfg, lock, deps.removeFile); err != nil {
		return err
	}

	cfg.GameVersion = targetVersion

	if err := deps.writeLock(ctx, deps.fs, meta, []models.ModInstall{}); err != nil {
		return err
	}

	return deps.writeConfig(ctx, deps.fs, meta, cfg)
}
