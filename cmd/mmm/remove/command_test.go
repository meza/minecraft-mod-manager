package remove

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

	assert.Error(t, runE(cmd, []string{"mod"}))
}

func TestCommandMissingQuietFlagErrors(t *testing.T) {
	runE := Command().RunE
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().StringP("config", "c", "modlist.json", "config")
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, []string{"mod"}))
}

func TestCommandMissingDebugFlagErrors(t *testing.T) {
	runE := Command().RunE
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().StringP("config", "c", "modlist.json", "config")
	cmd.Flags().BoolP("quiet", "q", false, "quiet")
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, []string{"mod"}))
}

func TestCommandMissingDryRunFlagErrors(t *testing.T) {
	runE := Command().RunE
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().StringP("config", "c", "modlist.json", "config")
	cmd.Flags().BoolP("quiet", "q", false, "quiet")
	cmd.Flags().BoolP("debug", "d", false, "debug")
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, []string{"mod"}))
}

func TestCommandSuccessRemovesFiles(t *testing.T) {
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
		Mods: []models.Mod{
			{ID: "mod-a", Name: "Mod A", Type: models.MODRINTH},
		},
	}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	lock := []models.ModInstall{
		{Id: "mod-a", Type: models.MODRINTH, FileName: "mod-a.jar"},
	}
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar"), []byte("mod"), 0644))

	cmd := Command()
	addPersistentFlagsForTesting(cmd)
	output := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(output)
	cmd.SetErr(errOut)
	cmd.SetArgs([]string{"--config", configPath, "mod-a"})

	assert.NoError(t, cmd.Execute())

	exists, err := afero.Exists(fs, filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar"))
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestCommandReturnsErrorWhenRemoveFails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "missing.json")

	cmd := Command()
	addPersistentFlagsForTesting(cmd)
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--config", configPath, "mod-a"})

	assert.Error(t, cmd.Execute())
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
