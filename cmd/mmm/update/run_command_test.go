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
	tui "github.com/meza/minecraft-mod-manager/internal/view"
)

func TestRunUpdateCommandReturnsFlagError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	setCommandOutputForTesting(cmd)

	assert.Error(t, runUpdateCommand(cmd))
}

func TestRunUpdateCommandUsesLogProgramWhenInteractive(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restore := tui.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restore)

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

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	addFlagsForUpdateCommandTesting(cmd)
	cmd.SetIn(terminalReader{Reader: bytes.NewBuffer(nil)})
	cmd.SetOut(&terminalWriter{})
	cmd.SetErr(io.Discard)
	require.NoError(t, cmd.Flags().Set("config", configPath))
	require.NoError(t, cmd.Flags().Set("quiet", "true"))

	assert.NoError(t, runUpdateCommand(cmd))
}

func TestRunUpdateCommandSkipsLogProgramWhenUnattended(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restore := tui.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restore)

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

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	addFlagsForUpdateCommandTesting(cmd)
	cmd.SetIn(terminalReader{Reader: bytes.NewBuffer(nil)})
	cmd.SetOut(&terminalWriter{})
	cmd.SetErr(io.Discard)
	require.NoError(t, cmd.Flags().Set("config", configPath))
	require.NoError(t, cmd.Flags().Set("unattended", "true"))

	assert.NoError(t, runUpdateCommand(cmd))
}

func addFlagsForUpdateCommandTesting(cmd *cobra.Command) {
	cmd.Flags().StringP("config", "c", "./modlist.json", "An alternative JSON file containing the configuration")
	cmd.Flags().Bool("unattended", false, "Disable prompts and fail fast when required inputs are missing")
	cmd.Flags().BoolP("quiet", "q", false, "Suppress non-essential output (errors and required results still print)")
	cmd.Flags().BoolP("debug", "d", false, "Enable debug messages")
}
