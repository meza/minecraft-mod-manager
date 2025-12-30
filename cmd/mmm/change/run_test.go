package change

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/meza/minecraft-mod-manager/cmd/mmm/install"
	"github.com/meza/minecraft-mod-manager/cmd/mmm/test"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/tui"
)

func TestRunChangeUpdatesConfigAndRunsInstall(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/workspace/modlist.json")

	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
			{ID: "beta", Name: "Beta", Type: models.CURSEFORGE},
		},
	}
	lock := []models.ModInstall{
		{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"},
		{ID: "beta", Type: models.CURSEFORGE, FileName: "beta.jar"},
	}

	require.NoError(t, fs.MkdirAll(filepath.Join(meta.Dir(), "mods"), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), "mods", "alpha.jar"), []byte("alpha"), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), "mods", "beta.jar"), []byte("beta"), 0644))

	removed := make([]string, 0)
	installCalls := 0

	deps := changeDeps{
		fs:         fs,
		testDeps:   test.Deps{},
		testCmd:    &cobra.Command{},
		installCmd: &cobra.Command{},
		colorMode:  tui.ColorDisabled,
		testRunner: func(_ context.Context, _ *cobra.Command, _ test.Options, _ test.Deps, _ tui.ColorMode) (test.Result, error) {
			return test.Result{
				TargetVersion: "1.21.1",
				ExitCode:      0,
			}, nil
		},
		installRunner: func(_ context.Context, _ *cobra.Command, _ string, _ bool, _ bool) (install.Result, error) {
			installCalls++
			return install.Result{InstalledCount: len(cfg.Mods)}, nil
		},
		readConfig:  config.ReadConfig,
		ensureLock:  func(_ context.Context, _ afero.Fs, _ config.Metadata) ([]models.ModInstall, error) { return lock, nil },
		writeConfig: config.WriteConfig,
		writeLock:   config.WriteLock,
		removeFile: func(fs afero.Fs, path string) error {
			removed = append(removed, path)
			return fs.Remove(path)
		},
	}

	result, err := runChange(context.Background(), &cobra.Command{}, changeOptions{
		ConfigPath:  "/workspace/modlist.json",
		GameVersion: "latest",
	}, deps)
	require.NoError(t, err)

	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "1.21.1", result.TargetVersion)
	assert.Equal(t, 1, installCalls)

	updatedCfg, err := config.ReadConfig(context.Background(), fs, meta)
	require.NoError(t, err)
	assert.Equal(t, "1.21.1", updatedCfg.GameVersion)

	updatedLock, err := config.ReadLock(context.Background(), fs, meta)
	require.NoError(t, err)
	assert.Empty(t, updatedLock)

	assert.Contains(t, removed, filepath.Join(meta.Dir(), "mods", "alpha.jar"))
	assert.Contains(t, removed, filepath.Join(meta.Dir(), "mods", "beta.jar"))
}

func TestRunChangeContinuesWithForceWhenUnsupported(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/workspace/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "gamma", Name: "Gamma", Type: models.MODRINTH},
		},
	}
	lock := []models.ModInstall{
		{ID: "gamma", Type: models.MODRINTH, FileName: "gamma.jar"},
	}

	require.NoError(t, fs.MkdirAll(filepath.Join(meta.Dir(), "mods"), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), "mods", "gamma.jar"), []byte("gamma"), 0644))

	var installCalled bool
	deps := changeDeps{
		fs:         fs,
		testDeps:   test.Deps{},
		testCmd:    &cobra.Command{},
		installCmd: &cobra.Command{},
		colorMode:  tui.ColorDisabled,
		testRunner: func(_ context.Context, _ *cobra.Command, opts test.Options, _ test.Deps, _ tui.ColorMode) (test.Result, error) {
			assert.True(t, opts.Force)
			return test.Result{
				TargetVersion:   "1.21.0",
				ExitCode:        0,
				UnsupportedMods: []models.Mod{{ID: "gamma", Name: "Gamma", Type: models.MODRINTH}},
			}, nil
		},
		installRunner: func(_ context.Context, _ *cobra.Command, _ string, _ bool, _ bool) (install.Result, error) {
			installCalled = true
			return install.Result{}, nil
		},
		readConfig:  config.ReadConfig,
		ensureLock:  func(_ context.Context, _ afero.Fs, _ config.Metadata) ([]models.ModInstall, error) { return lock, nil },
		writeConfig: config.WriteConfig,
		writeLock:   config.WriteLock,
		removeFile: func(fs afero.Fs, path string) error {
			return fs.Remove(path)
		},
	}

	result, err := runChange(context.Background(), &cobra.Command{}, changeOptions{
		ConfigPath:  "/workspace/modlist.json",
		GameVersion: "1.21.0",
		Force:       true,
	}, deps)
	require.NoError(t, err)

	assert.True(t, installCalled)
	assert.Equal(t, "1.21.0", result.TargetVersion)
	assert.Equal(t, []models.Mod{{ID: "gamma", Name: "Gamma", Type: models.MODRINTH}}, result.UnsupportedMods)
}

func TestRunChangeStopsWhenVersionCheckFails(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/workspace/modlist.json")

	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
	}
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	testErr := stubExitError{code: 2}
	deps := changeDeps{
		fs:         fs,
		testDeps:   test.Deps{},
		testCmd:    &cobra.Command{},
		installCmd: &cobra.Command{},
		colorMode:  tui.ColorDisabled,
		testRunner: func(_ context.Context, _ *cobra.Command, _ test.Options, _ test.Deps, _ tui.ColorMode) (test.Result, error) {
			return test.Result{
				TargetVersion: "1.20.1",
				ExitCode:      2,
			}, testErr
		},
		installRunner: func(_ context.Context, _ *cobra.Command, _ string, _ bool, _ bool) (install.Result, error) {
			t.Fatal("installRunner should not be called")
			return install.Result{}, nil
		},
		readConfig:  config.ReadConfig,
		ensureLock:  config.EnsureLock,
		writeConfig: config.WriteConfig,
		writeLock:   config.WriteLock,
		removeFile:  func(_ afero.Fs, _ string) error { return nil },
	}

	result, err := runChange(context.Background(), &cobra.Command{}, changeOptions{
		ConfigPath:  "/workspace/modlist.json",
		GameVersion: "1.20.1",
	}, deps)

	assert.Equal(t, testErr, err)
	assert.Equal(t, 2, result.ExitCode)

	updatedCfg, readErr := config.ReadConfig(context.Background(), fs, meta)
	require.NoError(t, readErr)
	assert.Equal(t, "1.20.1", updatedCfg.GameVersion)
}

func TestExitCodeForError(t *testing.T) {
	assert.Equal(t, 0, exitCodeForError(nil))
	assert.Equal(t, 3, exitCodeForError(stubExitError{code: 3}))
	assert.Equal(t, 1, exitCodeForError(assert.AnError))
}

func TestRemoveInstalledModsSkipsMissingFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/workspace/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "missing", Name: "Missing", Type: models.MODRINTH},
		},
	}

	lock := []models.ModInstall{
		{ID: "missing", Type: models.MODRINTH, FileName: "missing.jar"},
	}

	err := removeInstalledMods(fs, meta, cfg, lock, func(_ afero.Fs, _ string) error {
		t.Fatal("remover should not be called")
		return nil
	})
	assert.NoError(t, err)
}

func TestRemoveInstalledModsReturnsErrorOnRemoval(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/workspace/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	require.NoError(t, fs.MkdirAll(filepath.Join(meta.Dir(), "mods"), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), "mods", "alpha.jar"), []byte("data"), 0644))

	lock := []models.ModInstall{
		{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"},
	}

	err := removeInstalledMods(fs, meta, cfg, lock, func(_ afero.Fs, _ string) error {
		return assert.AnError
	})
	assert.Error(t, err)
}

func TestRemoveInstalledModsSkipsWhenNoMods(t *testing.T) {
	err := removeInstalledMods(afero.NewMemMapFs(), config.NewMetadata("/workspace/modlist.json"), models.ModsJSON{}, []models.ModInstall{}, func(afero.Fs, string) error {
		t.Fatal("remover should not run")
		return nil
	})
	assert.NoError(t, err)
}

func TestRemoveInstalledModsContinuesOnMissingLockEntry(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/workspace/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	lock := []models.ModInstall{
		{ID: "beta", Type: models.MODRINTH, FileName: "beta.jar"},
	}

	err := removeInstalledMods(fs, meta, cfg, lock, func(afero.Fs, string) error {
		t.Fatal("remover should not run")
		return nil
	})
	assert.NoError(t, err)
}

func TestRemoveInstalledModsReturnsStatError(t *testing.T) {
	fs := errorStatFs{Fs: afero.NewMemMapFs()}
	meta := config.NewMetadata("/workspace/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	lock := []models.ModInstall{
		{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"},
	}

	err := removeInstalledMods(fs, meta, cfg, lock, func(afero.Fs, string) error {
		return nil
	})
	assert.Error(t, err)
}

func TestApplyVersionChangeReturnsError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/workspace/modlist.json")
	cfg := models.ModsJSON{GameVersion: "1.20.1", ModsFolder: "mods"}

	err := applyVersionChange(context.Background(), meta, cfg, "1.21.0", changeDeps{
		fs: fs,
		ensureLock: func(context.Context, afero.Fs, config.Metadata) ([]models.ModInstall, error) {
			return nil, assert.AnError
		},
	})
	assert.Error(t, err)
}

func TestApplyVersionChangeWriteLockError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/workspace/modlist.json")
	cfg := models.ModsJSON{GameVersion: "1.20.1", ModsFolder: "mods"}

	err := applyVersionChange(context.Background(), meta, cfg, "1.21.0", changeDeps{
		fs: fs,
		ensureLock: func(context.Context, afero.Fs, config.Metadata) ([]models.ModInstall, error) {
			return []models.ModInstall{}, nil
		},
		writeLock: func(context.Context, afero.Fs, config.Metadata, []models.ModInstall) error {
			return assert.AnError
		},
	})
	assert.Error(t, err)
}

func TestApplyVersionChangeRemoveError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/workspace/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "mod", Name: "Mod", Type: models.MODRINTH},
		},
	}
	require.NoError(t, fs.MkdirAll(filepath.Join(meta.Dir(), "mods"), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), "mods", "mod.jar"), []byte("data"), 0644))

	err := applyVersionChange(context.Background(), meta, cfg, "1.21.0", changeDeps{
		fs: fs,
		ensureLock: func(context.Context, afero.Fs, config.Metadata) ([]models.ModInstall, error) {
			return []models.ModInstall{{ID: "mod", Type: models.MODRINTH, FileName: "mod.jar"}}, nil
		},
		removeFile: func(afero.Fs, string) error {
			return assert.AnError
		},
		writeLock: func(context.Context, afero.Fs, config.Metadata, []models.ModInstall) error { return nil },
		writeConfig: func(context.Context, afero.Fs, config.Metadata, models.ModsJSON) error {
			return nil
		},
	})
	assert.Error(t, err)
}

func TestRunChangeReturnsInstallError(t *testing.T) {
	deps := changeDeps{
		fs:         afero.NewMemMapFs(),
		testDeps:   test.Deps{},
		testCmd:    &cobra.Command{},
		installCmd: &cobra.Command{},
		colorMode:  tui.ColorDisabled,
		testRunner: func(context.Context, *cobra.Command, test.Options, test.Deps, tui.ColorMode) (test.Result, error) {
			return test.Result{TargetVersion: "1.21.0", ExitCode: 0}, nil
		},
		installRunner: func(context.Context, *cobra.Command, string, bool, bool) (install.Result, error) {
			return install.Result{}, stubExitError{code: 7}
		},
		readConfig: func(context.Context, afero.Fs, config.Metadata) (models.ModsJSON, error) {
			return models.ModsJSON{GameVersion: "1.20.1"}, nil
		},
		ensureLock: func(context.Context, afero.Fs, config.Metadata) ([]models.ModInstall, error) {
			return []models.ModInstall{}, nil
		},
		writeConfig: func(context.Context, afero.Fs, config.Metadata, models.ModsJSON) error { return nil },
		writeLock:   func(context.Context, afero.Fs, config.Metadata, []models.ModInstall) error { return nil },
		removeFile:  func(afero.Fs, string) error { return nil },
	}

	result, err := runChange(context.Background(), &cobra.Command{}, changeOptions{
		ConfigPath:  "/workspace/modlist.json",
		GameVersion: "1.21.0",
	}, deps)

	assert.Error(t, err)
	assert.Equal(t, 7, result.ExitCode)
}

func TestRunChangeEnsureLockError(t *testing.T) {
	deps := changeDeps{
		fs:         afero.NewMemMapFs(),
		testDeps:   test.Deps{},
		testCmd:    &cobra.Command{},
		installCmd: &cobra.Command{},
		colorMode:  tui.ColorDisabled,
		testRunner: func(context.Context, *cobra.Command, test.Options, test.Deps, tui.ColorMode) (test.Result, error) {
			return test.Result{TargetVersion: "1.21.0", ExitCode: 0}, nil
		},
		readConfig: func(context.Context, afero.Fs, config.Metadata) (models.ModsJSON, error) {
			return models.ModsJSON{GameVersion: "1.20.1"}, nil
		},
		ensureLock: func(context.Context, afero.Fs, config.Metadata) ([]models.ModInstall, error) {
			return nil, assert.AnError
		},
		writeConfig: func(context.Context, afero.Fs, config.Metadata, models.ModsJSON) error { return nil },
		writeLock:   func(context.Context, afero.Fs, config.Metadata, []models.ModInstall) error { return nil },
		removeFile:  func(afero.Fs, string) error { return nil },
	}

	result, err := runChange(context.Background(), &cobra.Command{}, changeOptions{
		ConfigPath:  "/workspace/modlist.json",
		GameVersion: "1.21.0",
	}, deps)

	assert.Error(t, err)
	assert.Equal(t, 1, result.ExitCode)
}

func TestRunVersionCheckSetsExitCodeOnError(t *testing.T) {
	deps := changeDeps{
		testDeps:  test.Deps{},
		testCmd:   &cobra.Command{},
		colorMode: tui.ColorDisabled,
		testRunner: func(context.Context, *cobra.Command, test.Options, test.Deps, tui.ColorMode) (test.Result, error) {
			return test.Result{TargetVersion: "1.21.0"}, assert.AnError
		},
	}

	result, err := runVersionCheck(context.Background(), changeOptions{
		ConfigPath:  "/workspace/modlist.json",
		GameVersion: "1.21.0",
	}, deps)

	assert.Equal(t, 1, result.ExitCode)
	assert.Equal(t, assert.AnError, err)
}

func TestRunChangeReadConfigError(t *testing.T) {
	deps := changeDeps{
		fs:         afero.NewMemMapFs(),
		testDeps:   test.Deps{},
		testCmd:    &cobra.Command{},
		installCmd: &cobra.Command{},
		colorMode:  tui.ColorDisabled,
		testRunner: func(context.Context, *cobra.Command, test.Options, test.Deps, tui.ColorMode) (test.Result, error) {
			t.Fatal("testRunner should not run")
			return test.Result{}, nil
		},
		readConfig: func(context.Context, afero.Fs, config.Metadata) (models.ModsJSON, error) {
			return models.ModsJSON{}, assert.AnError
		},
	}

	result, err := runChange(context.Background(), &cobra.Command{}, changeOptions{
		ConfigPath: "/workspace/modlist.json",
	}, deps)

	assert.Equal(t, assert.AnError, err)
	assert.Equal(t, 1, result.ExitCode)
}

type stubExitError struct {
	code int
}

func (err stubExitError) Error() string {
	return "stub"
}

func (err stubExitError) ExitCode() int {
	return err.code
}

type errorStatFs struct {
	afero.Fs
}

func (fs errorStatFs) Stat(name string) (os.FileInfo, error) {
	return nil, assert.AnError
}
