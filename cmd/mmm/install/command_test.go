package install

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/locksync"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
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

func TestInstallOptionsFromFlagsReturnsLockSyncFlagError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("unattended", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")
	cmd.Flags().String(locksync.FlagAdd, "", "")

	_, err := installOptionsFromFlags(cmd)
	assert.Error(t, err)
}

func TestCommandWithRunner_HandledErrorSilencesCobra(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cmd := commandWithRunner(func(_ context.Context, _ *cobra.Command, _ installOptions, _ installDeps) (Result, error) {
		return Result{}, clierrors.MarkHandled(assert.AnError)
	})
	addPersistentFlagsForTesting(cmd)
	setCommandOutputForTesting(cmd)

	cmd.SetArgs([]string{})
	assert.Error(t, cmd.Execute())
	assert.True(t, cmd.SilenceErrors)
	assert.True(t, cmd.SilenceUsage)
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

func TestCommandWithRunnerMissingUnattendedFlagErrors(t *testing.T) {
	runE := commandWithRunner(func(context.Context, *cobra.Command, installOptions, installDeps) (Result, error) {
		return Result{}, nil
	}).RunE

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().StringP("config", "c", "modlist.json", "config")
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
	cmd.Flags().Bool("unattended", false, "unattended")
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
	cmd.Flags().Bool("unattended", false, "unattended")
	cmd.Flags().BoolP("quiet", "q", false, "quiet")
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, nil))
}

func TestCommandWithRunnerUsesOutputWhenTerminal(t *testing.T) {
	outputWriter := &terminalWriter{}
	cmd := commandWithRunner(func(_ context.Context, _ *cobra.Command, _ installOptions, deps installDeps) (Result, error) {
		return Result{InstalledCount: 1}, deps.output.Log("installed", output.LogForce)
	})
	addPersistentFlagsForTesting(cmd)
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(outputWriter)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{})

	assert.NoError(t, cmd.Execute())
	assert.Contains(t, outputWriter.String(), "installed")
}

func TestCommandWithRunnerQuietSuppressesOutput(t *testing.T) {
	outputWriter := &terminalWriter{}
	cmd := commandWithRunner(func(_ context.Context, _ *cobra.Command, _ installOptions, deps installDeps) (Result, error) {
		return Result{InstalledCount: 1}, deps.output.Log("quiet", output.LogQuiet)
	})
	addPersistentFlagsForTesting(cmd)
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(outputWriter)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--unattended", "--quiet"})

	assert.NoError(t, cmd.Execute())
	assert.Empty(t, outputWriter.String())
}

func TestRun_ReturnsErrorWhenConfigMissing(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := Run(context.Background(), cmd, RunOptions{
		ConfigPath: filepath.Join(t.TempDir(), "missing.json"),
	})
	assert.Error(t, err)
}

func TestRun_ReturnsZeroWhenNoMods(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewOsFs()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "modlist.json")
	meta := config.NewMetadata(configPath)

	cfg := models.ModsJSON{
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

	result, err := Run(context.Background(), cmd, RunOptions{
		ConfigPath: configPath,
		Quiet:      true,
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, result.InstalledCount)
}

func addPersistentFlagsForTesting(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("config", "c", "./modlist.json", "An alternative JSON file containing the configuration")
	cmd.PersistentFlags().Bool("unattended", false, "Disable prompts and fail fast when required inputs are missing")
	cmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress non-essential output (errors and required results still print)")
	cmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug messages")
}

func setCommandOutputForTesting(cmd *cobra.Command) {
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
}

type terminalReader struct {
	io.Reader
}

func (terminalReader) Fd() uintptr {
	return 0
}

type terminalWriter struct {
	bytes.Buffer
}

func (writer *terminalWriter) Fd() uintptr {
	return 1
}
