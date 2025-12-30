package list

import (
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/output"
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

	cfg := models.ModsJSON{
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

	installedContents := []byte("installed")
	installedHash := fmt.Sprintf("%x", sha1.Sum(installedContents))

	lock := []models.ModInstall{
		{ID: "mod-a", Type: models.MODRINTH, FileName: "mod-a.jar", Hash: installedHash},
		{ID: "mod-b", Type: models.CURSEFORGE, FileName: "mod-b.jar", Hash: "missing"},
	}
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar"), installedContents, 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, _, err := runList(context.Background(), cmd, meta.ConfigPath, runListOptions{quiet: false}, listDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		output:    output.New(out, errOut, false),
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	expected := "cmd.list.header\n" +
		"V cmd.list.entry.installed, Arg 1: {Count: 0, Data: &map[id:mod-a name:Mod A]}\n" +
		"X cmd.list.entry.missing, Arg 1: {Count: 0, Data: &map[id:mod-b name:Mod B]}\n"
	assert.Equal(t, expected, out.String())
	assert.Empty(t, errOut.String())
}

func TestRunListShowsHashMismatch(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
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

	installedContents := []byte("installed")
	installedHash := fmt.Sprintf("%x", sha1.Sum(installedContents))
	otherHash := fmt.Sprintf("%x", sha1.Sum([]byte("different")))

	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar"), installedContents, 0644))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{
		{ID: "mod-a", Type: models.MODRINTH, FileName: "mod-a.jar", Hash: otherHash},
	}))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, _, err := runList(context.Background(), cmd, meta.ConfigPath, runListOptions{quiet: false}, listDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		output:    output.New(out, errOut, false),
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	expected := "cmd.list.header\n" +
		"X cmd.list.entry.hash_mismatch, Arg 1: {Count: 0, Data: &map[file:mod-a.jar fix:cmd.list.entry.hash_mismatch.fix, Arg 1: {Count: 0, Data: &map[fix_command:mmm install]} id:mod-a name:Mod A platform:modrinth]}\n"
	assert.Equal(t, expected, out.String())
	assert.Empty(t, errOut.String())
	assert.NotEqual(t, installedHash, otherHash)
}

func TestEntryStatusReturnsMissingWhenHashEmpty(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "mod-a", Type: models.MODRINTH}

	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar"), []byte("installed"), 0644))

	lock := []models.ModInstall{{ID: "mod-a", Type: models.MODRINTH, FileName: "mod-a.jar", Hash: ""}}
	status := entryStatus(mod, lock, meta, cfg, fs)

	assert.Equal(t, listEntryMissing, status.Status)
}

func TestEntryStatusReturnsMissingWhenHashReadFails(t *testing.T) {
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "mod-a", Type: models.MODRINTH}

	fs := openErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar"),
		err:      errors.New("open failed"),
	}

	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar"), []byte("installed"), 0644))

	lock := []models.ModInstall{{ID: "mod-a", Type: models.MODRINTH, FileName: "mod-a.jar", Hash: "expected"}}
	status := entryStatus(mod, lock, meta, cfg, fs)

	assert.Equal(t, listEntryMissing, status.Status)
}

func TestSha1ForFileReturnsErrorOnOpenFailure(t *testing.T) {
	fs := openErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.FromSlash("/mods/mod-a.jar"),
		err:      errors.New("open failed"),
	}

	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/mods"), 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.FromSlash("/mods/mod-a.jar"), []byte("installed"), 0644))

	_, err := sha1ForFile(fs, filepath.FromSlash("/mods/mod-a.jar"))
	assert.Error(t, err)
}

func TestSha1ForFileReturnsErrorOnReadFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	filePath := filepath.FromSlash("/mods/mod-a.jar")
	assert.NoError(t, baseFs.MkdirAll(filepath.FromSlash("/mods"), 0755))
	assert.NoError(t, afero.WriteFile(baseFs, filePath, []byte("installed"), 0644))

	fs := readErrorFs{
		Fs:       baseFs,
		failPath: filePath,
		err:      errors.New("read failed"),
	}

	_, err := sha1ForFile(fs, filePath)
	assert.Error(t, err)
}

func TestSha1ForFileReturnsErrorOnCloseFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	filePath := filepath.FromSlash("/mods/mod-a.jar")
	assert.NoError(t, baseFs.MkdirAll(filepath.FromSlash("/mods"), 0755))
	assert.NoError(t, afero.WriteFile(baseFs, filePath, []byte("installed"), 0644))

	fs := closeErrorFs{
		Fs:       baseFs,
		failPath: filePath,
		err:      errors.New("close failed"),
	}

	_, err := sha1ForFile(fs, filePath)
	assert.Error(t, err)
}

func TestRunListLogsInvalidLockFileName(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
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
		{ID: "mod-a", Name: " ", Type: models.MODRINTH, FileName: "mods/mod-a.jar"},
	}))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, _, err := runList(context.Background(), cmd, meta.ConfigPath, runListOptions{quiet: false}, listDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		output:    output.New(out, errOut, false),
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Contains(t, errOut.String(), "cmd.list.error.invalid_filename_lock")
}

func TestLogInvalidLockEntriesReturnsOutputError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	writeErr := errors.New("write failed")
	outWriter := output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false)

	err := logInvalidLockEntries([]models.ModInstall{
		{ID: "mod-a", Name: "Mod A", FileName: "mods/mod-a.jar"},
	}, outWriter)
	assert.ErrorIs(t, err, writeErr)
}

func TestRenderListReturnsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	cmd := &cobra.Command{}

	err := renderList(context.Background(), cmd, nil, "view", listDeps{
		output: output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false),
	}, listDisplayCLI)
	assert.ErrorIs(t, err, writeErr)
}

func TestRenderListReturnsOutputErrorWhenTuiEmpty(t *testing.T) {
	writeErr := errors.New("write failed")
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	err := renderList(context.Background(), cmd, nil, "view", listDeps{
		output: output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false),
		programRunner: func(tea.Model, ...tea.ProgramOption) error {
			return nil
		},
	}, listDisplayTUI)
	assert.ErrorIs(t, err, writeErr)
}

func TestRenderListLogsWhenTuiEmpty(t *testing.T) {
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&out)

	err := renderList(context.Background(), cmd, nil, "view", listDeps{
		output: output.New(&out, &out, false),
		programRunner: func(tea.Model, ...tea.ProgramOption) error {
			return nil
		},
	}, listDisplayTUI)
	assert.NoError(t, err)
	assert.Contains(t, out.String(), "view")
}

func TestRenderListReturnsProgramRunnerError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	err := renderList(context.Background(), cmd, []listEntry{{ID: "mod-a", DisplayName: "Mod A"}}, "view", listDeps{
		output: output.New(io.Discard, io.Discard, false),
		programRunner: func(tea.Model, ...tea.ProgramOption) error {
			return errors.New("boom")
		},
	}, listDisplayTUI)
	assert.ErrorContains(t, err, "boom")
}

func TestRenderListReturnsNilWhenTuiHasEntries(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	err := renderList(context.Background(), cmd, []listEntry{{ID: "mod-a", DisplayName: "Mod A"}}, "view", listDeps{
		output: output.New(io.Discard, io.Discard, false),
		programRunner: func(tea.Model, ...tea.ProgramOption) error {
			return nil
		},
	}, listDisplayTUI)
	assert.NoError(t, err)
}

func TestRenderListViewReturnsEmptyOnWriteError(t *testing.T) {
	originalWriteString := listWriteString
	t.Cleanup(func() {
		listWriteString = originalWriteString
	})
	listWriteString = func(*strings.Builder, string) error {
		return errors.New("write failed")
	}

	output := renderListView([]listEntry{
		{ID: "mod-a", DisplayName: "Mod A"},
	}, tui.ColorDisabled)

	assert.Equal(t, "", output)
}

func TestRenderListViewReturnsEmptyOnNewlineWriteError(t *testing.T) {
	originalWriteString := listWriteString
	t.Cleanup(func() {
		listWriteString = originalWriteString
	})
	callCount := 0
	listWriteString = func(*strings.Builder, string) error {
		callCount++
		if callCount == 2 {
			return errors.New("write failed")
		}
		return nil
	}

	output := renderListView([]listEntry{
		{ID: "mod-a", DisplayName: "Mod A"},
	}, tui.ColorDisabled)

	assert.Equal(t, "", output)
}

func TestRenderListViewReturnsEmptyOnEntryWriteError(t *testing.T) {
	originalWriteString := listWriteString
	t.Cleanup(func() {
		listWriteString = originalWriteString
	})
	callCount := 0
	listWriteString = func(*strings.Builder, string) error {
		callCount++
		if callCount == 3 {
			return errors.New("write failed")
		}
		return nil
	}

	output := renderListView([]listEntry{
		{ID: "mod-a", DisplayName: "Mod A"},
	}, tui.ColorDisabled)

	assert.Equal(t, "", output)
}

func TestRunListMissingLockTreatsAllAsNotInstalled(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
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

	_, _, err := runList(context.Background(), cmd, meta.ConfigPath, runListOptions{quiet: false}, listDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		output:    output.New(out, errOut, false),
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	expected := "cmd.list.header\n" +
		"X cmd.list.entry.missing, Arg 1: {Count: 0, Data: &map[id:mod-a name:Mod A]}\n"
	assert.Equal(t, expected, out.String())
}

func TestRunListInvalidLockErrors(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
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

	_, _, err := runList(context.Background(), cmd, meta.ConfigPath, runListOptions{quiet: false}, listDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		output:    output.New(out, errOut, false),
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.Error(t, err)
}

func TestRunListReturnsErrorOnInvalidLockOutputFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
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
		{ID: "mod-a", Name: "Mod A", Type: models.MODRINTH, FileName: "mods/mod-a.jar"},
	}))

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	writeErr := errors.New("write failed")
	_, _, err := runList(context.Background(), cmd, meta.ConfigPath, runListOptions{quiet: false}, listDeps{
		fs:        fs,
		logger:    logger.New(io.Discard, io.Discard, false, false),
		output:    output.New(io.Discard, errorWriter{err: writeErr}, false),
		telemetry: func(telemetry.CommandTelemetry) {},
	})
	assert.ErrorIs(t, err, writeErr)
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

	_, _, err := runList(context.Background(), cmd, meta.ConfigPath, runListOptions{quiet: false}, listDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		output:    output.New(out, errOut, false),
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.Error(t, err)
}

func TestRunListShowsEmptyMessageWhenNoMods(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
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

	_, _, err := runList(context.Background(), cmd, meta.ConfigPath, runListOptions{quiet: false}, listDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		output:    output.New(out, errOut, false),
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

	cfg := models.ModsJSON{
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

	_, _, err := runList(context.Background(), cmd, meta.ConfigPath, runListOptions{quiet: true}, listDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, true, false),
		output:    output.New(out, errOut, true),
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	expected := "cmd.list.header\n" +
		"X cmd.list.entry.missing, Arg 1: {Count: 0, Data: &map[id:mod-a name:Mod A]}\n"
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
	cfg := models.ModsJSON{
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
	lock := []models.ModInstall{{ID: "mod-a", Type: models.MODRINTH, FileName: ""}}

	assert.False(t, isInstalled(mod, lock, config.NewMetadata("modlist.json"), models.ModsJSON{}, afero.NewMemMapFs()))
}

func TestIsInstalledReturnsFalseWhenFileNameInvalid(t *testing.T) {
	mod := models.Mod{ID: "mod-a", Type: models.MODRINTH}
	lock := []models.ModInstall{{ID: "mod-a", Type: models.MODRINTH, FileName: "mods/mod-a.jar"}}

	assert.False(t, isInstalled(mod, lock, config.NewMetadata("modlist.json"), models.ModsJSON{}, afero.NewMemMapFs()))
}

func TestIsInstalledReturnsFalseWhenStatFails(t *testing.T) {
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "mod-a", Type: models.MODRINTH}
	lock := []models.ModInstall{{ID: "mod-a", Type: models.MODRINTH, FileName: "mod-a.jar"}}

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
	cfg := models.ModsJSON{
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

	_, usedTUI, err := runList(context.Background(), cmd, meta.ConfigPath, runListOptions{quiet: false}, listDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		output:    output.New(out, errOut, false),
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
	cfg := models.ModsJSON{
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

	entriesCount, usedTUI, err := runList(context.Background(), cmd, meta.ConfigPath, runListOptions{quiet: false}, listDeps{
		fs:            fs,
		logger:        logger.New(out, errOut, false, false),
		output:        output.New(out, errOut, false),
		telemetry:     func(telemetry.CommandTelemetry) {},
		programRunner: defaultProgramRunner,
	})

	assert.NoError(t, err)
	assert.True(t, usedTUI)
	assert.Equal(t, 0, entriesCount)
	assert.Contains(t, out.String(), "cmd.list.empty")
}

func TestRunListNonInteractiveDisablesTUI(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restore := tui.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
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

	_, usedTUI, err := runList(context.Background(), cmd, meta.ConfigPath, runListOptions{
		nonInteractive: true,
		quiet:          false,
	}, listDeps{
		fs:            fs,
		logger:        logger.New(out, errOut, false, false),
		output:        output.New(out, errOut, false),
		telemetry:     func(telemetry.CommandTelemetry) {},
		programRunner: defaultProgramRunner,
	})

	assert.NoError(t, err)
	assert.False(t, usedTUI)
}

func TestRunListUsesDefaultProgramRunner(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restore := tui.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
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

	_, usedTUI, err := runList(context.Background(), cmd, meta.ConfigPath, runListOptions{quiet: false}, listDeps{
		fs:            fs,
		logger:        logger.New(out, &bytes.Buffer{}, false, false),
		output:        output.New(out, &bytes.Buffer{}, false),
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

type openErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

type readErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

type closeErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

type errorWriter struct {
	err error
}

func (writer errorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

func (filesystem statErrorFs) Stat(name string) (os.FileInfo, error) {
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return nil, filesystem.err
	}
	return filesystem.Fs.Stat(name)
}

func (filesystem openErrorFs) Open(name string) (afero.File, error) {
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return nil, filesystem.err
	}
	return filesystem.Fs.Open(name)
}

type readErrorFile struct {
	afero.File
	err error
}

func (file readErrorFile) Read([]byte) (int, error) {
	return 0, file.err
}

func (filesystem readErrorFs) Open(name string) (afero.File, error) {
	file, err := filesystem.Fs.Open(name)
	if err != nil {
		return nil, err
	}
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return readErrorFile{File: file, err: filesystem.err}, nil
	}
	return file, nil
}

type closeErrorFile struct {
	afero.File
	err error
}

func (file closeErrorFile) Close() error {
	return file.err
}

func (filesystem closeErrorFs) Open(name string) (afero.File, error) {
	file, err := filesystem.Fs.Open(name)
	if err != nil {
		return nil, err
	}
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return closeErrorFile{File: file, err: filesystem.err}, nil
	}
	return file, nil
}

type fakeTTY struct {
	*bytes.Buffer
}

func (tty fakeTTY) Fd() uintptr { return 0 }
