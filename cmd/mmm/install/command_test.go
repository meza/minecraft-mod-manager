package install

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

func TestCommandWithRunner_ParsesFlags(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	var gotOpts installOptions
	cmd := commandWithRunner(func(_ context.Context, _ *cobra.Command, opts installOptions, _ installDeps) (Result, error) {
		gotOpts = opts
		return Result{InstalledCount: 1}, nil
	})
	addPersistentFlagsForTesting(cmd)
	setCommandOutputForTesting(cmd)

	cmd.SetArgs([]string{})
	assert.NoError(t, cmd.Execute())
	assert.Equal(t, "./modlist.json", gotOpts.ConfigPath)
	assert.False(t, gotOpts.Quiet)
	assert.False(t, gotOpts.Debug)
}

func TestCommandWithRunner_ErrorReturnsError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cmd := commandWithRunner(func(_ context.Context, _ *cobra.Command, _ installOptions, _ installDeps) (Result, error) {
		return Result{}, assert.AnError
	})
	addPersistentFlagsForTesting(cmd)
	setCommandOutputForTesting(cmd)

	cmd.SetArgs([]string{})
	assert.Error(t, cmd.Execute())
}

func TestCommandReturnsCommand(t *testing.T) {
	assert.NotNil(t, Command())
}

func TestCommandWithRunnerMissingConfigFlagErrors(t *testing.T) {
	runE := commandWithRunner(func(context.Context, *cobra.Command, installOptions, installDeps) (Result, error) {
		return Result{}, nil
	}).RunE

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, nil))
}

func TestCommandWithRunnerMissingQuietFlagErrors(t *testing.T) {
	runE := commandWithRunner(func(context.Context, *cobra.Command, installOptions, installDeps) (Result, error) {
		return Result{}, nil
	}).RunE

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().StringP("config", "c", "modlist.json", "config")
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, nil))
}

func TestCommandWithRunnerMissingDebugFlagErrors(t *testing.T) {
	runE := commandWithRunner(func(context.Context, *cobra.Command, installOptions, installDeps) (Result, error) {
		return Result{}, nil
	}).RunE

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().StringP("config", "c", "modlist.json", "config")
	cmd.Flags().BoolP("quiet", "q", false, "quiet")
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, nil))
}

func TestRun_ReturnsErrorWhenConfigMissing(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := Run(context.Background(), cmd, filepath.Join(t.TempDir(), "missing.json"), false, false)
	assert.Error(t, err)
}

func TestRun_ReturnsZeroWhenNoMods(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewOsFs()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "modlist.json")
	meta := config.NewMetadata(configPath)

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	result, err := Run(context.Background(), cmd, configPath, true, false)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.InstalledCount)
	assert.False(t, result.UnmanagedFound)
}

func addPersistentFlagsForTesting(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("config", "c", "./modlist.json", "An alternative JSON file containing the configuration")
	cmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress all output")
	cmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug messages")
}

func setCommandOutputForTesting(cmd *cobra.Command) {
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
}
