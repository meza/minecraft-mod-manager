package locksync

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type statErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (fs statErrorFs) Stat(name string) (os.FileInfo, error) {
	if name == fs.failPath {
		return nil, fs.err
	}
	return fs.Fs.Stat(name)
}

type removeErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (fs removeErrorFs) Remove(name string) error {
	if name == fs.failPath {
		return fs.err
	}
	return fs.Fs.Remove(name)
}

func TestRunLockSyncGate_NoExtrasReturnsUnchanged(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods: []models.Mod{{
			Type: models.MODRINTH,
			ID:   "abc",
			Name: "Alpha",
		}},
	}
	lock := []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "abc",
		Name:     "Alpha",
		FileName: "alpha.jar",
	}}

	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	outcome, err := RunLockSyncGate(GateInput{
		Ctx:         context.Background(),
		Fs:          fs,
		Meta:        meta,
		Config:      cfg,
		Lock:        lock,
		Mode:        interaction.ExecutionModeNonTTY,
		ColorMode:   view.ColorDisabled,
		In:          bytes.NewBuffer(nil),
		Out:         &bytes.Buffer{},
		RunTea:      func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) { return model, nil },
		PolicyFlags: PolicyFlags{},
	})
	require.NoError(t, err)
	assert.False(t, outcome.Applied)
	assert.Equal(t, cfg, outcome.Config)
	assert.Equal(t, lock, outcome.Lock)
	assert.True(t, outcome.ShouldContinue)
}

func TestRunLockSyncGate_PromptCanceledStops(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "abc",
		Name:     "Alpha",
		FileName: "alpha.jar",
	}}

	outcome, err := RunLockSyncGate(GateInput{
		Ctx:       context.Background(),
		Fs:        fs,
		Meta:      meta,
		Config:    cfg,
		Lock:      lock,
		Mode:      interaction.ExecutionModeInteractive,
		ColorMode: view.ColorDisabled,
		In:        bytes.NewBuffer(nil),
		Out:       &bytes.Buffer{},
		RunTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return lockSyncPromptModel{canceled: true}, nil
		},
		PolicyFlags: PolicyFlags{},
	})
	require.NoError(t, err)
	assert.False(t, outcome.ShouldContinue)
	assert.False(t, outcome.Applied)
	assert.Equal(t, PolicyUnknown, outcome.Policy)
}

func TestRunLockSyncGate_PromptErrorReturnsError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "abc",
		Name:     "Alpha",
		FileName: "alpha.jar",
	}}
	expectedErr := errors.New("prompt failed")

	_, err := RunLockSyncGate(GateInput{
		Ctx:       context.Background(),
		Fs:        fs,
		Meta:      meta,
		Config:    cfg,
		Lock:      lock,
		Mode:      interaction.ExecutionModeInteractive,
		ColorMode: view.ColorDisabled,
		In:        bytes.NewBuffer(nil),
		Out:       &bytes.Buffer{},
		RunTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, expectedErr
		},
		PolicyFlags: PolicyFlags{},
	})
	assert.ErrorIs(t, err, expectedErr)
}

func TestRunLockSyncGate_PolicyMultipleReturnsError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "abc",
		Name:     "Alpha",
		FileName: "alpha.jar",
	}}

	_, err := RunLockSyncGate(GateInput{
		Ctx:       context.Background(),
		Fs:        fs,
		Meta:      meta,
		Config:    cfg,
		Lock:      lock,
		Mode:      interaction.ExecutionModeNonTTY,
		ColorMode: view.ColorDisabled,
		In:        bytes.NewBuffer(nil),
		Out:       &bytes.Buffer{},
		RunTea:    func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) { return model, nil },
		PolicyFlags: PolicyFlags{
			Add:    true,
			Delete: true,
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cmd.lock_sync.error.policy_multiple")
}

func TestRunLockSyncGate_NonInteractiveDefaultsAdd(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "abc",
		Name:     "Alpha",
		FileName: "alpha.jar",
	}}
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	outcome, err := RunLockSyncGate(GateInput{
		Ctx:       context.Background(),
		Fs:        fs,
		Meta:      meta,
		Config:    cfg,
		Lock:      lock,
		Mode:      interaction.ExecutionModeNonTTY,
		ColorMode: view.ColorDisabled,
		In:        bytes.NewBuffer(nil),
		Out:       &bytes.Buffer{},
		RunTea:    func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) { return model, nil },
	})
	require.NoError(t, err)
	assert.True(t, outcome.Applied)
	assert.Len(t, outcome.Config.Mods, 1)
	assert.Equal(t, "abc", outcome.Config.Mods[0].ID)
	assert.Equal(t, PolicyAdd, outcome.Policy)

	readCfg, readErr := config.ReadConfig(context.Background(), fs, meta)
	require.NoError(t, readErr)
	assert.Len(t, readCfg.Mods, 1)
}

func TestRunLockSyncGate_ForceSkips(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "abc",
		Name:     "Alpha",
		FileName: "alpha.jar",
	}}

	outcome, err := RunLockSyncGate(GateInput{
		Ctx:       context.Background(),
		Fs:        fs,
		Meta:      meta,
		Config:    cfg,
		Lock:      lock,
		Mode:      interaction.ExecutionModeNonTTY,
		ColorMode: view.ColorDisabled,
		In:        bytes.NewBuffer(nil),
		Out:       &bytes.Buffer{},
		RunTea:    func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) { return model, nil },
		Force:     true,
	})
	require.NoError(t, err)
	assert.Equal(t, PolicySkip, outcome.Policy)
	assert.False(t, outcome.Applied)
}

func TestRunLockSyncGate_WritesResolutionAfterPrompt(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "abc",
		Name:     "Alpha",
		FileName: "alpha.jar",
	}}

	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	outputLines := []string{}
	runTea := func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		if outputModel, ok := model.(view.OutputLinesModel); ok {
			outputLines = append(outputLines, outputModel.Lines...)
			return model, nil
		}
		return lockSyncPromptModel{policy: PolicyAdd}, nil
	}

	outcome, err := RunLockSyncGate(GateInput{
		Ctx:         context.Background(),
		Fs:          fs,
		Meta:        meta,
		Config:      cfg,
		Lock:        lock,
		Mode:        interaction.ExecutionModeInteractive,
		CommandName: "install",
		ColorMode:   view.ColorDisabled,
		In:          bytes.NewBuffer(nil),
		Out:         &bytes.Buffer{},
		RunTea:      runTea,
		PolicyFlags: PolicyFlags{},
	})
	require.NoError(t, err)
	assert.True(t, outcome.ShouldContinue)
	assert.NotEmpty(t, outputLines)
	assert.Contains(t, strings.Join(outputLines, "\n"), "cmd.lock_sync.result.add")
	assert.Contains(t, strings.Join(outputLines, "\n"), "cmd.lock_sync.continue")
}

func TestRunLockSyncGate_ReturnsResolutionOutputError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "abc",
		Name:     "Alpha",
		FileName: "alpha.jar",
	}}

	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	runTea := func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		if _, ok := model.(view.OutputLinesModel); ok {
			return model, errors.New("write failed")
		}
		return lockSyncPromptModel{policy: PolicyAdd}, nil
	}

	_, err := RunLockSyncGate(GateInput{
		Ctx:         context.Background(),
		Fs:          fs,
		Meta:        meta,
		Config:      cfg,
		Lock:        lock,
		Mode:        interaction.ExecutionModeInteractive,
		CommandName: "install",
		ColorMode:   view.ColorDisabled,
		In:          bytes.NewBuffer(nil),
		Out:         &bytes.Buffer{},
		RunTea:      runTea,
		PolicyFlags: PolicyFlags{},
	})
	assert.Error(t, err)
}

func TestWriteLockSyncResolutionReturnsErrorWhenRunnerMissing(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	err := writeLockSyncResolution(GateInput{
		CommandName: "install",
		ColorMode:   view.ColorDisabled,
		In:          bytes.NewBuffer(nil),
		Out:         &bytes.Buffer{},
		RunTea:      nil,
	}, PolicyAdd)
	assert.Error(t, err)
}

func TestWriteLockSyncResolutionSkipsWhenNoLines(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	runTeaCalled := false
	err := writeLockSyncResolution(GateInput{
		CommandName: "install",
		ColorMode:   view.ColorDisabled,
		In:          bytes.NewBuffer(nil),
		Out:         &bytes.Buffer{},
		RunTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			runTeaCalled = true
			return model, nil
		},
	}, PolicyUnknown)
	require.NoError(t, err)
	assert.False(t, runTeaCalled)
}

func TestRunLockSyncGate_SummaryErrorReturnsError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "abc",
		Name:     "Alpha",
		FileName: "alpha.jar",
	}}
	expectedErr := errors.New("summary failed")

	_, err := RunLockSyncGate(GateInput{
		Ctx:       context.Background(),
		Fs:        fs,
		Meta:      meta,
		Config:    cfg,
		Lock:      lock,
		Mode:      interaction.ExecutionModeNonTTY,
		ColorMode: view.ColorDisabled,
		In:        bytes.NewBuffer(nil),
		Out:       &bytes.Buffer{},
		RunTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, expectedErr
		},
	})
	assert.ErrorIs(t, err, expectedErr)
}

func TestRunLockSyncGate_IgnoreAddsMmmignoreAndRemovesLock(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "abc",
		Name:     "Alpha",
		FileName: "alpha.jar",
	}}
	require.NoError(t, fs.MkdirAll(filepath.Join(meta.Dir(), "mods"), 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), "mods", "alpha.jar"), []byte("ok"), 0o644))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	outcome, err := RunLockSyncGate(GateInput{
		Ctx:       context.Background(),
		Fs:        fs,
		Meta:      meta,
		Config:    cfg,
		Lock:      lock,
		Mode:      interaction.ExecutionModeNonTTY,
		ColorMode: view.ColorDisabled,
		In:        bytes.NewBuffer(nil),
		Out:       &bytes.Buffer{},
		RunTea:    func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) { return model, nil },
		PolicyFlags: PolicyFlags{
			Ignore: true,
		},
	})
	require.NoError(t, err)
	assert.True(t, outcome.Applied)
	assert.Equal(t, PolicyIgnore, outcome.Policy)

	ignoredData, readErr := afero.ReadFile(fs, filepath.Join(meta.Dir(), ".mmmignore"))
	require.NoError(t, readErr)
	assert.Contains(t, string(ignoredData), "alpha.jar")

	updatedLock, lockErr := config.ReadLock(context.Background(), fs, meta)
	require.NoError(t, lockErr)
	assert.Len(t, updatedLock, 0)
}

func TestRunLockSyncGate_IgnoreSkipsMissingFiles(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "abc",
		Name:     "Alpha",
		FileName: "alpha.jar",
	}}
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	outcome, err := RunLockSyncGate(GateInput{
		Ctx:       context.Background(),
		Fs:        fs,
		Meta:      meta,
		Config:    cfg,
		Lock:      lock,
		Mode:      interaction.ExecutionModeNonTTY,
		ColorMode: view.ColorDisabled,
		In:        bytes.NewBuffer(nil),
		Out:       &bytes.Buffer{},
		RunTea:    func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) { return model, nil },
		PolicyFlags: PolicyFlags{
			Ignore: true,
		},
	})
	require.NoError(t, err)
	assert.True(t, outcome.Applied)

	_, ignoreErr := afero.ReadFile(fs, filepath.Join(meta.Dir(), ".mmmignore"))
	assert.Error(t, ignoreErr)

	updatedLock, lockErr := config.ReadLock(context.Background(), fs, meta)
	require.NoError(t, lockErr)
	assert.Len(t, updatedLock, 0)
}

func TestRunLockSyncGate_IgnoreInvalidFilenameReturnsError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "abc",
		Name:     "Alpha",
		FileName: "mods/alpha.jar",
	}}
	_, err := RunLockSyncGate(GateInput{
		Ctx:       context.Background(),
		Fs:        fs,
		Meta:      meta,
		Config:    cfg,
		Lock:      lock,
		Mode:      interaction.ExecutionModeNonTTY,
		ColorMode: view.ColorDisabled,
		In:        bytes.NewBuffer(nil),
		Out:       &bytes.Buffer{},
		RunTea:    func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) { return model, nil },
		PolicyFlags: PolicyFlags{
			Ignore: true,
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cmd.lock_sync.error.invalid_filename_lock")
}

func TestRunLockSyncGate_DeleteRemovesFilesAndLock(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "abc",
		Name:     "Alpha",
		FileName: "alpha.jar",
	}}
	require.NoError(t, fs.MkdirAll(filepath.Join(meta.Dir(), "mods"), 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), "mods", "alpha.jar"), []byte("ok"), 0o644))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	outcome, err := RunLockSyncGate(GateInput{
		Ctx:       context.Background(),
		Fs:        fs,
		Meta:      meta,
		Config:    cfg,
		Lock:      lock,
		Mode:      interaction.ExecutionModeNonTTY,
		ColorMode: view.ColorDisabled,
		In:        bytes.NewBuffer(nil),
		Out:       &bytes.Buffer{},
		RunTea:    func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) { return model, nil },
		PolicyFlags: PolicyFlags{
			Delete: true,
		},
	})
	require.NoError(t, err)
	assert.True(t, outcome.Applied)

	exists, existsErr := afero.Exists(fs, filepath.Join(meta.Dir(), "mods", "alpha.jar"))
	require.NoError(t, existsErr)
	assert.False(t, exists)

	updatedLock, lockErr := config.ReadLock(context.Background(), fs, meta)
	require.NoError(t, lockErr)
	assert.Len(t, updatedLock, 0)
}

func TestRunLockSyncGate_DeleteFailsOnInvalidFilename(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "abc",
		Name:     "Alpha",
		FileName: "mods/alpha.jar",
	}}
	_, err := RunLockSyncGate(GateInput{
		Ctx:       context.Background(),
		Fs:        fs,
		Meta:      meta,
		Config:    cfg,
		Lock:      lock,
		Mode:      interaction.ExecutionModeNonTTY,
		ColorMode: view.ColorDisabled,
		In:        bytes.NewBuffer(nil),
		Out:       &bytes.Buffer{},
		RunTea:    func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) { return model, nil },
		PolicyFlags: PolicyFlags{
			Delete: true,
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cmd.lock_sync.error.invalid_filename_lock")
}

func TestRunLockSyncGate_DeleteRemoveErrorReturnsError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "abc",
		Name:     "Alpha",
		FileName: "alpha.jar",
	}}
	require.NoError(t, baseFs.MkdirAll(filepath.Join(meta.Dir(), "mods"), 0o755))
	require.NoError(t, afero.WriteFile(baseFs, filepath.Join(meta.Dir(), "mods", "alpha.jar"), []byte("ok"), 0o644))

	fs := removeErrorFs{Fs: baseFs, failPath: filepath.Join(meta.Dir(), "mods", "alpha.jar"), err: errors.New("remove failed")}

	_, err := RunLockSyncGate(GateInput{
		Ctx:       context.Background(),
		Fs:        fs,
		Meta:      meta,
		Config:    cfg,
		Lock:      lock,
		Mode:      interaction.ExecutionModeNonTTY,
		ColorMode: view.ColorDisabled,
		In:        bytes.NewBuffer(nil),
		Out:       &bytes.Buffer{},
		RunTea:    func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) { return model, nil },
		PolicyFlags: PolicyFlags{
			Delete: true,
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remove failed")
}

func TestRunLockSyncGate_ExistsCheckErrorReturnsError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "abc",
		Name:     "Alpha",
		FileName: "alpha.jar",
	}}
	require.NoError(t, baseFs.MkdirAll(filepath.Join(meta.Dir(), "mods"), 0o755))
	fs := statErrorFs{Fs: baseFs, failPath: filepath.Join(meta.Dir(), "mods", "alpha.jar"), err: errors.New("stat failed")}

	_, err := RunLockSyncGate(GateInput{
		Ctx:       context.Background(),
		Fs:        fs,
		Meta:      meta,
		Config:    cfg,
		Lock:      lock,
		Mode:      interaction.ExecutionModeNonTTY,
		ColorMode: view.ColorDisabled,
		In:        bytes.NewBuffer(nil),
		Out:       &bytes.Buffer{},
		RunTea:    func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) { return model, nil },
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stat failed")
}

func TestRemoveFileForceIgnoresMissingFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := removeFileForce(fs, filepath.FromSlash("/missing.jar"))
	assert.NoError(t, err)
}

func TestAddLockEntriesToConfigSkipsDuplicates(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	cfg := models.ModsJSON{
		Mods: []models.Mod{{
			Type: models.MODRINTH,
			ID:   "abc",
			Name: "Alpha",
		}},
	}
	extras := []extraLockEntry{{
		Install: models.ModInstall{
			Type: models.MODRINTH,
			ID:   "abc",
			Name: "Alpha",
		},
		DisplayName: "Alpha",
	}}
	updated, changed := addLockEntriesToConfig(cfg, extras)
	assert.False(t, changed)
	assert.Equal(t, cfg, updated)
}

func TestBuildExtraLockEntriesUsesIDWhenNameMissing(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "abc",
		FileName: "alpha.jar",
	}}
	extras, err := buildExtraLockEntries(fs, meta, cfg, lock)
	require.NoError(t, err)
	require.Len(t, extras, 1)
	assert.Equal(t, "abc", extras[0].DisplayName)
}

func TestRenderLockSyncListIncludesHeaderAndStatuses(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	extras := []extraLockEntry{
		{
			Install: models.ModInstall{
				Type:     models.MODRINTH,
				ID:       "alpha",
				Name:     "Alpha",
				FileName: "alpha.jar",
			},
			DisplayName: "Alpha",
			FileStatus:  fileStatusMissing,
		},
		{
			Install: models.ModInstall{
				Type:     models.MODRINTH,
				ID:       "beta",
				Name:     "Beta",
				FileName: "mods/beta.jar",
			},
			DisplayName: "Beta",
			FileStatus:  fileStatusInvalid,
		},
	}
	rendered := renderLockSyncList(extras, view.ColorDisabled)
	assert.Contains(t, rendered, "cmd.lock_sync.header")
	assert.Contains(t, rendered, "cmd.lock_sync.entry.missing_suffix")
	assert.Contains(t, rendered, "cmd.lock_sync.entry.invalid_suffix")
}

func TestRenderLockSyncListShowsPresentEntriesWithoutSuffix(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	extras := []extraLockEntry{{
		Install: models.ModInstall{
			Type:     models.MODRINTH,
			ID:       "alpha",
			Name:     "Alpha",
			FileName: "alpha.jar",
		},
		DisplayName: "Alpha",
		FileStatus:  fileStatusPresent,
	}}
	rendered := renderLockSyncList(extras, view.ColorDisabled)
	assert.Contains(t, rendered, "cmd.lock_sync.header")
	assert.NotContains(t, rendered, "cmd.lock_sync.entry.missing_suffix")
	assert.NotContains(t, rendered, "cmd.lock_sync.entry.invalid_suffix")
}

func TestRenderLockSyncListReturnsEmptyOnWriteFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	extras := []extraLockEntry{
		{
			Install: models.ModInstall{
				Type:     models.MODRINTH,
				ID:       "alpha",
				Name:     "Alpha",
				FileName: "alpha.jar",
			},
			DisplayName: "Alpha",
			FileStatus:  fileStatusMissing,
		},
		{
			Install: models.ModInstall{
				Type:     models.MODRINTH,
				ID:       "beta",
				Name:     "Beta",
				FileName: "beta.jar",
			},
			DisplayName: "Beta",
			FileStatus:  fileStatusMissing,
		},
	}

	cases := []struct {
		name   string
		target int
	}{
		{name: "header", target: 1},
		{name: "header-newline", target: 2},
		{name: "entry-line", target: 3},
		{name: "entry-separator", target: 4},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			restore := view.WriteString
			t.Cleanup(func() { view.WriteString = restore })
			callCount := 0
			view.WriteString = func(writer io.Writer, value string) error {
				callCount++
				if callCount == tc.target {
					return errors.New("write failed")
				}
				return restore(writer, value)
			}

			rendered := renderLockSyncList(extras, view.ColorDisabled)
			assert.Equal(t, "", rendered)
		})
	}
}

func TestWriteLockSyncSummaryReturnsErrorOnMissingRunner(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	err := writeLockSyncSummary(lockSyncSummaryInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing lock sync summary runner")
}

func TestWriteLockSyncSummaryWritesOutputLines(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	meta := config.NewMetadata("modlist.json")
	buffer := &bytes.Buffer{}
	extras := []extraLockEntry{{
		Install: models.ModInstall{
			Type:     models.MODRINTH,
			ID:       "alpha",
			Name:     "Alpha",
			FileName: "alpha.jar",
		},
		DisplayName: "Alpha",
		FileStatus:  fileStatusMissing,
	}}
	var captured tea.Model
	runTea := func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		captured = model
		return model, nil
	}

	err := writeLockSyncSummary(lockSyncSummaryInput{
		runTea:    runTea,
		in:        bytes.NewBuffer(nil),
		out:       buffer,
		colorMode: view.ColorDisabled,
		meta:      meta,
		extras:    extras,
		policy:    PolicySkip,
		command:   "install",
	})
	require.NoError(t, err)

	outputModel, ok := captured.(view.OutputLinesModel)
	require.True(t, ok)
	assert.Len(t, outputModel.Lines, 2)
	assert.Contains(t, outputModel.Lines[0], "cmd.lock_sync.header")
	assert.Contains(t, outputModel.Lines[1], "cmd.common.no_changes")
	assert.Contains(t, outputModel.Lines[1], "cmd.lock_sync.continue")
}
