package install

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

func TestInstallPlatformErrorOutputSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		ModsFolder:                 "mods",
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}

	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	badPath := filepath.Join(meta.ModsFolderPath(cfg), "bad.jar")
	assert.NoError(t, afero.WriteFile(fs, badPath, []byte("data"), 0644))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	deps := installDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
		clients: platform.Clients{
			Modrinth:   errorDoer{err: errors.New("unused")},
			Curseforge: errorDoer{err: errors.New("unused")},
		},
		curseforgeFingerprint: func(string) uint32 { return 123 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return nil, &httpclient.ResponseError{
				Method:     http.MethodPost,
				URL:        "https://example.invalid",
				StatusCode: http.StatusForbidden,
			}
		},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup("missing", modrinth.SHA1)}
		},
	}

	outcome, err := preflightUnknownFiles(preflightInputs{
		ctx:      context.Background(),
		meta:     meta,
		cfg:      cfg,
		lock:     nil,
		deps:     deps,
		colorize: false,
	})

	assert.NoError(t, err)
	assert.True(t, outcome.unresolved)

	snapshot := fmt.Sprintf(
		"stdout:\n%s\nstderr:\n%s",
		strings.TrimSpace(out.String()),
		strings.TrimSpace(errOut.String()),
	)
	snaps.MatchSnapshot(t, snapshot)
}
