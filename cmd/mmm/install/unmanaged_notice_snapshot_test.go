package install

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
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/testutil/terminal"
)

func TestInstallUnmanagedNoticeSnapshotShortHeight(t *testing.T) {
	runInstallUnmanagedNoticeSnapshot(t, 25)
}

func TestInstallUnmanagedNoticeSnapshotTallHeight(t *testing.T) {
	runInstallUnmanagedNoticeSnapshot(t, tallSnapshotRows())
}

func runInstallUnmanagedNoticeSnapshot(t *testing.T, rows uint16) {
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

	_, err := runInstall(context.Background(), cmd, installOptions{
		ConfigPath:   meta.ConfigPath,
		Unattended:   true,
		Quiet:        true,
		SkipLockSync: true,
		LockSync:     locksync.PolicyFlags{Skip: true},
	}, installDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
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

func tallSnapshotRows() uint16 {
	if runtime.GOOS == "windows" {
		return 40
	}
	return 80
}
