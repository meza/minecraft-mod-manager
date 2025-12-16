package list

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

func TestRunListPrintsInstalledAndMissing(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "mod-a", Name: "Mod A", Type: models.MODRINTH},
			{ID: "mod-b", Name: "Mod B", Type: models.CURSEFORGE},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	lock := []models.ModInstall{
		{Id: "mod-a", Type: models.MODRINTH, FileName: "mod-a.jar"},
		{Id: "mod-b", Type: models.CURSEFORGE, FileName: "mod-b.jar"},
	}
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar"), []byte("installed"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runList(context.Background(), cmd, meta.ConfigPath, false, listDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	expected := "cmd.list.header\n" +
		"✓ cmd.list.entry.installed, Arg 1: {Count: 0, Data: &map[id:mod-a name:Mod A]}\n" +
		"✗ cmd.list.entry.missing, Arg 1: {Count: 0, Data: &map[id:mod-b name:Mod B]}\n"
	assert.Equal(t, expected, out.String())
	assert.Empty(t, errOut.String())
}

func TestRunListMissingLockTreatsAllAsNotInstalled(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "mod-a", Name: "Mod A", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runList(context.Background(), cmd, meta.ConfigPath, false, listDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	expected := "cmd.list.header\n" +
		"✗ cmd.list.entry.missing, Arg 1: {Count: 0, Data: &map[id:mod-a name:Mod A]}\n"
	assert.Equal(t, expected, out.String())
}

func TestRunListInvalidLockErrors(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "mod-a", Name: "Mod A", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	lockPath := meta.LockPath()
	assert.NoError(t, afero.WriteFile(fs, lockPath, []byte("{invalid"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runList(context.Background(), cmd, meta.ConfigPath, false, listDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.Error(t, err)
}

func TestRunListInvalidConfigErrors(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, afero.WriteFile(fs, meta.ConfigPath, []byte("{invalid"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runList(context.Background(), cmd, meta.ConfigPath, false, listDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.Error(t, err)
}

func TestRunListShowsEmptyMessageWhenNoMods(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runList(context.Background(), cmd, meta.ConfigPath, false, listDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	expected := "cmd.list.empty\n"
	assert.Equal(t, expected, out.String())
}

func TestRunListQuietStillPrints(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "mod-a", Name: "Mod A", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runList(context.Background(), cmd, meta.ConfigPath, true, listDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, true, false),
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	expected := "cmd.list.header\n" +
		"✗ cmd.list.entry.missing, Arg 1: {Count: 0, Data: &map[id:mod-a name:Mod A]}\n"
	assert.Equal(t, expected, out.String())
}
