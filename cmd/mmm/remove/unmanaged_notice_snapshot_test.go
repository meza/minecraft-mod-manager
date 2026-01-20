package remove

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/locksync"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/testutil/terminal"
)

func TestRemoveUnmanagedNoticeSnapshotShortHeight(t *testing.T) {
	runRemoveUnmanagedNoticeSnapshot(t, 25)
}

func TestRemoveUnmanagedNoticeSnapshotTallHeight(t *testing.T) {
	runRemoveUnmanagedNoticeSnapshot(t, tallSnapshotRows())
}

func runRemoveUnmanagedNoticeSnapshot(t *testing.T, rows uint16) {
	terminal.ApplyFixtures(t)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{{
			ID:   "mod-a",
			Name: "Mod A",
			Type: models.MODRINTH,
		}},
	}
	lock := []models.ModInstall{{
		ID:       "mod-a",
		Type:     models.MODRINTH,
		FileName: "mod-a.jar",
	}}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar"), []byte("mod"), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar"), []byte("extra"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, _, err := runRemove(context.Background(), cmd, removeOptions{
		ConfigPath: meta.ConfigPath,
		Unattended: true,
		Force:      true,
		Lookups:    []string{"mod-a"},
		LockSync:   locksync.PolicyFlags{Skip: true},
	}, removeDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
	})
	require.ErrorIs(t, err, interaction.ErrUnmanagedFiles)
	require.True(t, clierrors.IsHandled(err))

	snapshot := formatNoticeSnapshot(out.String(), errOut.String(), rows)
	snaps.MatchSnapshot(t, snapshot)
}

func formatNoticeSnapshot(stdout string, stderr string, rows uint16) string {
	out := terminal.NormalizeOutput(stdout, terminal.NormalizeOptions{
		StripControlSequences:  true,
		TrimTrailingWhitespace: true,
		TrimTrailingEmptyLines: true,
		TrimSpace:              true,
		RowLimit:               int(rows),
	})
	return "stdout:\n" + out + "\nstderr:\n" + strings.TrimSpace(stderr)
}
