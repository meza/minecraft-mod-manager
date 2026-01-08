package test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
)

func TestTestCommandDefaultOutputSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "MissingMod", Type: models.MODRINTH},
			{ID: "proj-2", Name: "NoFileMod", Type: models.MODRINTH},
			{ID: "proj-3", Name: "PlatformErrorMod", Type: models.CURSEFORGE},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	result, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
		clients: platform.Clients{
			Modrinth:   noopDoer{},
			Curseforge: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			switch id {
			case "proj-1":
				return platform.RemoteMod{}, &platform.ModNotFoundError{Platform: p, ProjectID: id}
			case "proj-2":
				return platform.RemoteMod{}, &platform.NoCompatibleFileError{Platform: p, ProjectID: id}
			case "proj-3":
				return platform.RemoteMod{}, errors.New("platform timeout")
			default:
				return platform.RemoteMod{Name: "SupportedMod"}, nil
			}
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, errUnsupportedMods)
	assert.Equal(t, 1, result.ExitCode)

	snapshot := fmt.Sprintf(
		"stdout:\n%s\nstderr:\n%s",
		strings.TrimSpace(out.String()),
		strings.TrimSpace(errOut.String()),
	)
	snaps.MatchSnapshot(t, snapshot)
}
