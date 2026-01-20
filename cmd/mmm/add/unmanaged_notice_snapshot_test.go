package add

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
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
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/locksync"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/testutil/terminal"
)

func TestAddUnmanagedNoticeSnapshotShortHeight(t *testing.T) {
	runAddUnmanagedNoticeSnapshot(t, 25)
}

func TestAddUnmanagedNoticeSnapshotTallHeight(t *testing.T) {
	runAddUnmanagedNoticeSnapshot(t, tallSnapshotRows())
}

func runAddUnmanagedNoticeSnapshot(t *testing.T, rows uint16) {
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

	modData := []byte("mod")
	modHash := sha1Hex(string(modData))
	lock := []models.ModInstall{{
		ID:          "mod-a",
		Type:        models.MODRINTH,
		FileName:    "mod-a.jar",
		Hash:        modHash,
		DownloadURL: "https://example.invalid/mod-a.jar",
	}}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar"), modData, 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar"), []byte("extra"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runAdd(context.Background(), nil, cmd, addOptions{
		Platform:   string(models.MODRINTH),
		ProjectID:  "mod-a",
		ConfigPath: meta.ConfigPath,
		Unattended: true,
		LockSync:   locksync.PolicyFlags{Skip: true},
	}, addDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}, Curseforge: noopDoer{}},
		logger:  logger.New(io.Discard, io.Discard, false, false),
		output:  output.New(out, errOut, false),
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			return errors.New("download should not run")
		},
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

type noopDoer struct{}

func (noopDoer) Do(*http.Request) (*http.Response, error) {
	return nil, errors.New("noop doer")
}

func tallSnapshotRows() uint16 {
	if runtime.GOOS == "windows" {
		return 40
	}
	return 80
}
