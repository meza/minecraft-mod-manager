package list

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/tui"
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

	_, _, err := runList(context.Background(), cmd, meta.ConfigPath, false, listDeps{
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

func TestRunListLogsInvalidLockFileName(t *testing.T) {
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
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{
		{Id: "mod-a", Name: " ", Type: models.MODRINTH, FileName: "mods/mod-a.jar"},
	}))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, _, err := runList(context.Background(), cmd, meta.ConfigPath, false, listDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Contains(t, errOut.String(), "cmd.list.error.invalid_filename_lock")
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

	_, _, err := runList(context.Background(), cmd, meta.ConfigPath, false, listDeps{
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

	_, _, err := runList(context.Background(), cmd, meta.ConfigPath, false, listDeps{
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

	_, _, err := runList(context.Background(), cmd, meta.ConfigPath, false, listDeps{
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

	_, _, err := runList(context.Background(), cmd, meta.ConfigPath, false, listDeps{
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

	_, _, err := runList(context.Background(), cmd, meta.ConfigPath, true, listDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, true, false),
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	expected := "cmd.list.header\n" +
		"✗ cmd.list.entry.missing, Arg 1: {Count: 0, Data: &map[id:mod-a name:Mod A]}\n"
	assert.Equal(t, expected, out.String())
}

func TestReadLockOrEmptyReturnsErrorOnStatFailure(t *testing.T) {
	fs := statErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.FromSlash("/cfg/modlist-lock.json"),
		err:      errors.New("stat failed"),
	}
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	_, err := readLockOrEmpty(context.Background(), fs, meta)
	assert.Error(t, err)
}

func TestBuildEntriesUsesIDWhenNameBlank(t *testing.T) {
	cfg := models.ModsJson{
		Mods: []models.Mod{
			{ID: "mod-a", Name: " ", Type: models.MODRINTH},
		},
	}

	entries := buildEntries(cfg, nil, config.NewMetadata("modlist.json"), afero.NewMemMapFs())
	if assert.Len(t, entries, 1) {
		assert.Equal(t, "mod-a", entries[0].DisplayName)
	}
}

func TestIsInstalledReturnsFalseWhenFileNameMissing(t *testing.T) {
	mod := models.Mod{ID: "mod-a", Type: models.MODRINTH}
	lock := []models.ModInstall{{Id: "mod-a", Type: models.MODRINTH, FileName: ""}}

	assert.False(t, isInstalled(mod, lock, config.NewMetadata("modlist.json"), models.ModsJson{}, afero.NewMemMapFs()))
}

func TestIsInstalledReturnsFalseWhenFileNameInvalid(t *testing.T) {
	mod := models.Mod{ID: "mod-a", Type: models.MODRINTH}
	lock := []models.ModInstall{{Id: "mod-a", Type: models.MODRINTH, FileName: "mods/mod-a.jar"}}

	assert.False(t, isInstalled(mod, lock, config.NewMetadata("modlist.json"), models.ModsJson{}, afero.NewMemMapFs()))
}

func TestIsInstalledReturnsFalseWhenStatFails(t *testing.T) {
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{ModsFolder: "mods"}
	mod := models.Mod{ID: "mod-a", Type: models.MODRINTH}
	lock := []models.ModInstall{{Id: "mod-a", Type: models.MODRINTH, FileName: "mod-a.jar"}}

	failPath := filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar")
	fs := statErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: failPath,
		err:      errors.New("stat failed"),
	}

	assert.False(t, isInstalled(mod, lock, meta, cfg, fs))
}

func TestRunListTuiProgramRunnerError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restore := tui.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

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
	cmd.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetErr(errOut)

	_, usedTUI, err := runList(context.Background(), cmd, meta.ConfigPath, false, listDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
		programRunner: func(tea.Model, ...tea.ProgramOption) error {
			return errors.New("tui failed")
		},
	})

	assert.True(t, usedTUI)
	assert.Error(t, err)
}

func TestRunListTuiLogsEmptyView(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restore := tui.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

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
	cmd.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetErr(errOut)

	entriesCount, usedTUI, err := runList(context.Background(), cmd, meta.ConfigPath, false, listDeps{
		fs:            fs,
		logger:        logger.New(out, errOut, false, false),
		telemetry:     func(telemetry.CommandTelemetry) {},
		programRunner: defaultProgramRunner,
	})

	assert.NoError(t, err)
	assert.True(t, usedTUI)
	assert.Equal(t, 0, entriesCount)
	assert.Contains(t, out.String(), "cmd.list.empty")
}

func TestRunListUsesDefaultProgramRunner(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restore := tui.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

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

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetErr(&bytes.Buffer{})

	out := &bytes.Buffer{}

	_, usedTUI, err := runList(context.Background(), cmd, meta.ConfigPath, false, listDeps{
		fs:            fs,
		logger:        logger.New(out, &bytes.Buffer{}, false, false),
		telemetry:     func(telemetry.CommandTelemetry) {},
		programRunner: defaultProgramRunner,
	})

	assert.True(t, usedTUI)
	assert.NoError(t, err)
	assert.Contains(t, out.String(), "cmd.list.empty")
}

type statErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (s statErrorFs) Stat(name string) (os.FileInfo, error) {
	if filepath.Clean(name) == filepath.Clean(s.failPath) {
		return nil, s.err
	}
	return s.Fs.Stat(name)
}

type fakeTTY struct {
	*bytes.Buffer
}

func (f fakeTTY) Fd() uintptr { return 0 }
