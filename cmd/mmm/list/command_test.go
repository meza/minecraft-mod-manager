package list

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
	cmd := Command()
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{})

	assert.Error(t, cmd.Execute())
}

func TestCommandMissingQuietFlagErrors(t *testing.T) {
	cmd := Command()
	cmd.PersistentFlags().StringP("config", "c", "modlist.json", "config")
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{"--config", "modlist.json"})

	assert.Error(t, cmd.Execute())
}

func TestCommandMissingDebugFlagErrors(t *testing.T) {
	cmd := Command()
	cmd.PersistentFlags().StringP("config", "c", "modlist.json", "config")
	cmd.PersistentFlags().BoolP("quiet", "q", false, "quiet")
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{"--config", "modlist.json"})

	assert.Error(t, cmd.Execute())
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
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	cmd := Command()
	addPersistentFlagsForTesting(cmd)

	output := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(output)
	cmd.SetErr(errOut)
	cmd.SetArgs([]string{"--config", configPath, "--quiet"})

	assert.NoError(t, cmd.Execute())
	assert.NotEmpty(t, output.String())
}

func TestCommandErrorFromRunList(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewOsFs()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "modlist.json")

	require.NoError(t, fs.MkdirAll(tempDir, 0755))
	require.NoError(t, afero.WriteFile(fs, configPath, []byte("{invalid"), 0644))

	cmd := Command()
	addPersistentFlagsForTesting(cmd)

	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--config", configPath, "--quiet"})

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
