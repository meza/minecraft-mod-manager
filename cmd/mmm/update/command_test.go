package update

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"testing"

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

func TestCommandMissingQuietFlagErrors(t *testing.T) {
	runE := Command().RunE
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().StringP("config", "c", "modlist.json", "config")
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, []string{}))
}

func TestCommandMissingDebugFlagErrors(t *testing.T) {
	runE := Command().RunE
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().StringP("config", "c", "modlist.json", "config")
	cmd.Flags().BoolP("quiet", "q", false, "quiet")
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, []string{}))
}

func TestCommandSuccess(t *testing.T) {
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

func TestCommandSetsSilenceUsageOnError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cmd := Command()
	addPersistentFlagsForTesting(cmd)
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{"--config", filepath.Join(t.TempDir(), "missing.json"), "--quiet"})

	assert.Error(t, cmd.Execute())
	assert.True(t, cmd.SilenceUsage)
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
