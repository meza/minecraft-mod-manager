package update

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/locksync"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

func TestCommandMissingConfigFlagErrors(t *testing.T) {
	runE := Command().RunE
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, []string{}))
}

func TestCommandMissingUnattendedFlagErrors(t *testing.T) {
	runE := Command().RunE
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().StringP("config", "c", "modlist.json", "config")
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, []string{}))
}

func TestCommandMissingQuietFlagErrors(t *testing.T) {
	runE := Command().RunE
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().StringP("config", "c", "modlist.json", "config")
	cmd.Flags().Bool("unattended", false, "unattended")
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, []string{}))
}

func TestCommandMissingDebugFlagErrors(t *testing.T) {
	runE := Command().RunE
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().StringP("config", "c", "modlist.json", "config")
	cmd.Flags().Bool("unattended", false, "unattended")
	cmd.Flags().BoolP("quiet", "q", false, "quiet")
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, []string{}))
}

func TestApplyUpdateCommandErrorPolicyHandledError(t *testing.T) {
	cmd := &cobra.Command{}

	applyUpdateCommandErrorPolicy(cmd, clierrors.MarkHandled(assert.AnError))

	assert.True(t, cmd.SilenceErrors)
	assert.True(t, cmd.SilenceUsage)
}

func TestApplyUpdateCommandErrorPolicyUnhandledError(t *testing.T) {
	cmd := &cobra.Command{}

	applyUpdateCommandErrorPolicy(cmd, errors.New("boom"))

	assert.True(t, cmd.SilenceUsage)
	assert.False(t, cmd.SilenceErrors)
}

func TestApplyUpdateCommandErrorPolicyNil(t *testing.T) {
	cmd := &cobra.Command{}

	applyUpdateCommandErrorPolicy(cmd, nil)

	assert.False(t, cmd.SilenceUsage)
	assert.False(t, cmd.SilenceErrors)
}

func TestCommandSuccess(t *testing.T) {
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

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	cmd := Command()
	addPersistentFlagsForTesting(cmd)
	output := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(output)
	cmd.SetErr(errOut)
	cmd.SetArgs([]string{"--config", configPath, "--quiet"})

	assert.NoError(t, cmd.Execute())
}

func TestCommandSuccessUnattended(t *testing.T) {
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

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	cmd := Command()
	addPersistentFlagsForTesting(cmd)
	output := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(output)
	cmd.SetErr(errOut)
	cmd.SetArgs([]string{"--config", configPath, "--unattended"})

	assert.NoError(t, cmd.Execute())
}

func TestCommandSetsSilenceUsageOnError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cmd := Command()
	addPersistentFlagsForTesting(cmd)
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{"--config", filepath.Join(t.TempDir(), "missing.json"), "--quiet"})

	assert.Error(t, cmd.Execute())
	assert.True(t, cmd.SilenceUsage)
}

func TestCommandOutputsResults(t *testing.T) {
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

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	cmd := Command()
	addPersistentFlagsForTesting(cmd)
	output := &bytes.Buffer{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(output)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--config", configPath})

	assert.NoError(t, cmd.Execute())
	assert.Contains(t, output.String(), "cmd.list.empty")
}

func TestUpdateOptionsFromFlagsReturnsLockSyncFlagError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("unattended", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")
	cmd.Flags().String(locksync.FlagAdd, "", "")

	_, err := updateOptionsFromFlags(cmd)
	assert.Error(t, err)
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
