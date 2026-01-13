package remove

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/testutil/terminal"
	terminalpty "github.com/meza/minecraft-mod-manager/testutil/terminal/pty"
)

func TestRemoveCommandInteractivePTYIncludesSummary(t *testing.T) {
	terminal.ApplyFixtures(t)

	originalRunRemoveProgram := runRemoveProgram
	runRemoveProgram = defaultRunRemoveProgram
	t.Cleanup(func() { runRemoveProgram = originalRunRemoveProgram })

	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: 120, Rows: 40}))
	require.NotNil(t, session)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "modlist.json")
	meta := config.NewMetadata(configPath)
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "mod-a", Name: "Mod A", Type: models.MODRINTH},
		},
	}
	lock := []models.ModInstall{
		{ID: "mod-a", Type: models.MODRINTH, FileName: "mod-a.jar"},
	}

	fs := afero.NewOsFs()
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar"), []byte("mod"), 0644))

	cmd := Command()
	addPersistentFlagsForTesting(cmd)
	cmd.SetIn(session.Input())
	cmd.SetOut(session.Output())
	cmd.SetErr(session.Output())
	cmd.SetArgs([]string{"--config", configPath, "mod-a"})

	execErr := cmd.Execute()
	require.NoError(t, execErr)
	session.WaitForOutputAndClose(t, func(output []byte) bool {
		return strings.Contains(string(output), "cmd.remove.summary.success")
	}, terminalpty.WithWaitDuration(2*time.Second))

	normalized := terminal.NormalizeOutput(session.OutputString(), terminal.NormalizeOptions{StripControlSequences: true})
	require.Contains(t, normalized, "cmd.remove.summary.success")
}
