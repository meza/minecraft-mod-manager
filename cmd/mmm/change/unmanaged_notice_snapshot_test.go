package change

import (
	"bytes"
	"context"
	"path/filepath"
	"runtime"
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
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/testutil/terminal"
)

func TestChangeUnmanagedNoticeSnapshotShortHeight(t *testing.T) {
	runChangeUnmanagedNoticeSnapshot(t, 25)
}

func TestChangeUnmanagedNoticeSnapshotTallHeight(t *testing.T) {
	runChangeUnmanagedNoticeSnapshot(t, tallSnapshotRows())
}

func runChangeUnmanagedNoticeSnapshot(t *testing.T, rows uint16) {
	terminal.ApplyFixtures(t)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
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
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar"), []byte("extra"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	result, err := runChange(context.Background(), cmd, changeOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: cfg.GameVersion,
		Unattended:  true,
		LockSync:    locksync.PolicyFlags{Skip: true},
	}, changeDeps{
		fs:         fs,
		readConfig: config.ReadConfig,
		ensureLock: config.EnsureLock,
	})
	require.ErrorIs(t, err, interaction.ErrUnmanagedFiles)
	require.True(t, clierrors.IsHandled(err))
	require.Equal(t, 1, result.ExitCode)

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

func tallSnapshotRows() uint16 {
	if runtime.GOOS == "windows" {
		return 40
	}
	return 80
}
