package update

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/cmd/mmm/install"
	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/locksync"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestPrepareUpdateExecutionReturnsInstallError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods:       []models.Mod{{ID: "alpha", Type: models.MODRINTH}},
	}
	lock := []models.ModInstall{{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0o755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)

	state, err := prepareUpdateExecution(context.Background(), cmd, updateOptions{
		ConfigPath: meta.ConfigPath,
	}, updateDeps{
		fs: fs,
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			return install.Result{}, errors.New("install failed")
		},
	}, interaction.ExecutionModeNonTTY)
	assert.True(t, clierrors.IsHandled(err))
	assert.True(t, state.shouldContinue)
}

func TestRunUpdateStopsWhenLockSyncCanceled(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods:       []models.Mod{{ID: "alpha", Type: models.MODRINTH}},
	}
	lock := []models.ModInstall{{ID: "beta", Type: models.MODRINTH, FileName: "beta.jar"}}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0o755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTY{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTY{Buffer: bytes.NewBuffer(nil)})

	counts, err := runUpdate(context.Background(), cmd, updateOptions{
		ConfigPath: meta.ConfigPath,
	}, updateDeps{
		fs: fs,
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			return install.Result{}, nil
		},
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
			return updated, nil
		},
	})
	require.NoError(t, err)
	assert.Equal(t, updateCounts{}, counts)
}

func TestPrepareUpdateExecutionUsesLockSyncPolicyAndSkipsInstallLockSync(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods:       []models.Mod{{ID: "alpha", Type: models.MODRINTH}},
	}
	lock := []models.ModInstall{{ID: "beta", Type: models.MODRINTH, FileName: "beta.jar"}}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0o755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(io.Discard)

	var installOptions install.RunOptions
	installCalled := false

	state, err := prepareUpdateExecution(context.Background(), cmd, updateOptions{
		ConfigPath: meta.ConfigPath,
		Unattended: true,
		LockSync:   locksync.PolicyFlags{Skip: true},
	}, updateDeps{
		fs: fs,
		install: func(_ context.Context, _ *cobra.Command, options install.RunOptions) (install.Result, error) {
			installCalled = true
			installOptions = options
			return install.Result{}, nil
		},
	}, interaction.ExecutionModeUnattended)
	require.NoError(t, err)
	assert.True(t, state.shouldContinue)
	assert.True(t, installCalled)
	assert.True(t, installOptions.Unattended)
	assert.True(t, installOptions.SkipLockSync)
	assert.Equal(t, meta.ConfigPath, installOptions.ConfigPath)
}

func TestRunUpdateReturnsPolicyError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods:       []models.Mod{},
	}
	lock := []models.ModInstall{{ID: "beta", Type: models.MODRINTH, FileName: "beta.jar"}}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0o755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)

	_, err := runUpdate(context.Background(), cmd, updateOptions{
		ConfigPath: meta.ConfigPath,
		LockSync:   locksync.PolicyFlags{Add: true, Delete: true},
	}, updateDeps{
		fs: fs,
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			return install.Result{}, nil
		},
		runTea: runTeaProgram,
	})
	assert.Error(t, err)
}

func TestPrepareUpdateExecutionReturnsReadErrorAfterInstall(t *testing.T) {
	readErr := errors.New("read failed")
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods:       []models.Mod{{ID: "alpha", Type: models.MODRINTH}},
	}
	lock := []models.ModInstall{{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}}
	require.NoError(t, baseFs.MkdirAll(meta.Dir(), 0o755))
	require.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), baseFs, meta, lock))

	fs := &toggleOpenErrorFs{
		Fs:       baseFs,
		failPath: meta.LockPath(),
		err:      readErr,
	}

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)

	_, err := prepareUpdateExecution(context.Background(), cmd, updateOptions{
		ConfigPath: meta.ConfigPath,
	}, updateDeps{
		fs: fs,
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			fs.fail = true
			return install.Result{}, nil
		},
	}, interaction.ExecutionModeNonTTY)
	assert.ErrorIs(t, err, readErr)
}

type toggleOpenErrorFs struct {
	afero.Fs
	failPath string
	err      error
	fail     bool
}

func (filesystem *toggleOpenErrorFs) Open(name string) (afero.File, error) {
	if filesystem.fail && filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return nil, filesystem.err
	}
	return filesystem.Fs.Open(name)
}
