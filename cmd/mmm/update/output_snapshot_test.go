package update

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

	"github.com/meza/minecraft-mod-manager/cmd/mmm/install"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
)

func TestUpdateNonTTYInstallHeaderSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Loader:      models.FABRIC,
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "mod-a", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	lock := []models.ModInstall{
		{
			Type:        models.MODRINTH,
			ID:          "mod-a",
			Name:        "Alpha",
			FileName:    "alpha.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "hash",
			DownloadURL: "https://example.invalid/alpha.jar",
		},
	}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "alpha.jar"), []byte("data"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			_, writeErr := cmd.OutOrStdout().Write([]byte("install output\n"))
			require.NoError(t, writeErr)
			return install.Result{}, nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Alpha",
				FileName:    "alpha.jar",
				ReleaseDate: "2024-01-01T00:00:00Z",
				Hash:        "hash",
				DownloadURL: "https://example.invalid/alpha.jar",
			}, nil
		},
		clients:   platform.Clients{Modrinth: noopDoer{}},
		telemetry: func(telemetry.CommandTelemetry) {},
	})
	require.NoError(t, err)

	snaps.MatchSnapshot(t, strings.TrimSpace(out.String()))
}
