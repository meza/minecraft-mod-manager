package scan

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modsetup"
)

func TestPersistScanMatchesEmpty(t *testing.T) {
	added, err := persistScanMatches(context.Background(), &cobra.Command{}, scanExecutionInput{}, nil)
	assert.NoError(t, err)
	assert.Nil(t, added)
}

func TestPersistScanMatchesMissingSetupCoordinator(t *testing.T) {
	added, err := persistScanMatches(context.Background(), &cobra.Command{}, scanExecutionInput{}, []scanMatch{scanMatchFixture()})
	assert.Error(t, err)
	assert.Nil(t, added)
}

func TestPersistScanMatchesConfigWriteError(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	require.NoError(t, baseFs.MkdirAll("/cfg", 0o755))
	readOnlyFs := afero.NewReadOnlyFs(baseFs)

	meta := config.NewMetadata("/cfg/modlist.json")
	input := scanExecutionInput{
		meta:             meta,
		cfg:              models.ModsJSON{ModsFolder: "mods"},
		lock:             []models.ModInstall{},
		setupCoordinator: modsetup.NewSetupCoordinator(baseFs, nil, nil),
		deps: scanDeps{
			fs:     readOnlyFs,
			runTea: runTeaStub(runTeaScenario{}),
		},
	}

	_, err := persistScanMatches(context.Background(), &cobra.Command{}, input, []scanMatch{scanMatchFixture()})
	assert.Error(t, err)
}

func TestPersistScanMatchesLockWriteError(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	require.NoError(t, baseFs.MkdirAll("/cfg", 0o755))
	readOnlyFs := afero.NewReadOnlyFs(baseFs)

	meta := config.NewMetadata("/cfg/modlist.json")
	existing := scanMatchFixture()
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods: []models.Mod{
			{
				Type: existing.Platform,
				ID:   existing.ProjectID,
				Name: existing.Name,
			},
		},
	}

	input := scanExecutionInput{
		meta:             meta,
		cfg:              cfg,
		lock:             []models.ModInstall{},
		setupCoordinator: modsetup.NewSetupCoordinator(baseFs, nil, nil),
		deps: scanDeps{
			fs:     readOnlyFs,
			runTea: runTeaStub(runTeaScenario{}),
		},
	}

	_, err := persistScanMatches(context.Background(), &cobra.Command{}, input, []scanMatch{existing})
	assert.Error(t, err)
}

func TestPersistScanMatchesOutputError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/cfg", 0o755))
	meta := config.NewMetadata("/cfg/modlist.json")

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	input := scanExecutionInput{
		meta:             meta,
		cfg:              models.ModsJSON{ModsFolder: "mods"},
		lock:             []models.ModInstall{},
		setupCoordinator: modsetup.NewSetupCoordinator(fs, nil, nil),
		deps: scanDeps{
			fs:     fs,
			runTea: runTeaStub(runTeaScenario{err: errors.New("write failed")}),
		},
	}

	_, err := persistScanMatches(context.Background(), cmd, input, []scanMatch{{FileName: "alpha.jar"}})
	assert.Error(t, err)
}
