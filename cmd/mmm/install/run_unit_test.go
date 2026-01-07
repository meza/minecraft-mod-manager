package install

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestRunInteractiveInstallReturnsErrorWhenRunnerMissing(t *testing.T) {
	restore := runInstallProgram
	runInstallProgram = nil
	t.Cleanup(func() { runInstallProgram = restore })

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	items, indexByKey := buildInstallItems(models.ModsJSON{})
	_, err := runInteractiveInstall(context.Background(), cmd, installExecutionInput{
		meta:       config.NewMetadata("modlist.json"),
		cfg:        models.ModsJSON{},
		lock:       nil,
		deps:       installDeps{},
		items:      items,
		indexByKey: indexByKey,
	}, Result{})
	assert.ErrorContains(t, err, "missing bubble tea runner")
}

func TestRunInteractiveInstallReturnsHandledErrorFromOutcome(t *testing.T) {
	restore := runInstallProgram
	runInstallProgram = func(model *installModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = installExecutionOutcome{err: errors.New("boom"), errType: installExecutionErrorDownload}
		return model, nil
	}
	t.Cleanup(func() { runInstallProgram = restore })

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	items, indexByKey := buildInstallItems(models.ModsJSON{})
	_, err := runInteractiveInstall(context.Background(), cmd, installExecutionInput{
		meta:       config.NewMetadata("modlist.json"),
		cfg:        models.ModsJSON{},
		lock:       nil,
		deps:       installDeps{},
		items:      items,
		indexByKey: indexByKey,
	}, Result{})

	assert.ErrorContains(t, err, "boom")
	assert.True(t, clierrors.IsHandled(err))
}

func TestRunInteractiveInstallSuccess(t *testing.T) {
	restore := runInstallProgram
	runInstallProgram = func(model *installModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = installExecutionOutcome{err: nil, errType: installExecutionErrorNone}
		return model, nil
	}
	t.Cleanup(func() { runInstallProgram = restore })

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	items, indexByKey := buildInstallItems(models.ModsJSON{})
	_, err := runInteractiveInstall(context.Background(), cmd, installExecutionInput{
		meta:       config.NewMetadata("modlist.json"),
		cfg:        models.ModsJSON{},
		lock:       nil,
		deps:       installDeps{},
		items:      items,
		indexByKey: indexByKey,
	}, Result{})
	assert.NoError(t, err)
}

func TestRunInteractiveInstallExecutesRunner(t *testing.T) {
	restore := runInstallProgram
	runInstallProgram = func(model *installModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.bindSender(func(tea.Msg) {})
		cmd := model.Init()
		if cmd != nil {
			msg := cmd()
			_, _ = model.Update(msg)
		}
		return model, nil
	}
	t.Cleanup(func() { runInstallProgram = restore })

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	items, indexByKey := buildInstallItems(cfg)
	_, err := runInteractiveInstall(context.Background(), cmd, installExecutionInput{
		meta:       meta,
		cfg:        cfg,
		lock:       nil,
		deps:       installDeps{fs: fs},
		items:      items,
		indexByKey: indexByKey,
	}, Result{})
	assert.NoError(t, err)
}

func TestRunInteractiveInstallReturnsRunnerError(t *testing.T) {
	restore := runInstallProgram
	runInstallProgram = func(*installModel, ...tea.ProgramOption) (tea.Model, error) {
		return nil, errors.New("runner failed")
	}
	t.Cleanup(func() { runInstallProgram = restore })

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	items, indexByKey := buildInstallItems(models.ModsJSON{})
	_, err := runInteractiveInstall(context.Background(), cmd, installExecutionInput{
		meta:       config.NewMetadata("modlist.json"),
		cfg:        models.ModsJSON{},
		lock:       nil,
		deps:       installDeps{},
		items:      items,
		indexByKey: indexByKey,
	}, Result{})
	assert.ErrorContains(t, err, "runner failed")
}

func TestRunInstallWorkerReturnsSuccessWithoutState(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	lock := []models.ModInstall{{
		Type:        models.MODRINTH,
		ID:          "alpha",
		Name:        "Alpha",
		FileName:    "alpha.jar",
		Hash:        sha1Hex("data"),
		DownloadURL: "https://example.invalid/alpha.jar",
	}}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "alpha.jar"), []byte("data"), 0644))

	input := installConfiguredInputs{
		ctx:  context.Background(),
		meta: meta,
		cfg:  cfg,
		lock: lock,
		deps: installDeps{
			fs:     fs,
			logger: logger.New(io.Discard, io.Discard, false, false),
		},
	}

	failedCount := int64(0)
	err := runInstallWorker(context.Background(), input, cfg.Mods[0], cfg, 0, nil, nil, &failedCount)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), failedCount)
}

func TestRunInstallWorkerRecordsFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	lock := []models.ModInstall{{
		Type:        models.MODRINTH,
		ID:          "alpha",
		Name:        "Alpha",
		FileName:    "alpha.jar",
		Hash:        "",
		DownloadURL: "https://example.invalid/alpha.jar",
	}}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	items, indexByKey := buildInstallItems(cfg)
	state := &installExecutionState{items: items, indexByKey: indexByKey}
	input := installConfiguredInputs{
		ctx:  context.Background(),
		meta: meta,
		cfg:  cfg,
		lock: lock,
		deps: installDeps{
			fs:     fs,
			logger: logger.New(io.Discard, io.Discard, false, false),
		},
	}

	failedCount := int64(0)
	err := runInstallWorker(context.Background(), input, cfg.Mods[0], cfg, 0, state, nil, &failedCount)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), failedCount)
	assert.Equal(t, installItemFailed, state.items[indexByKey[installModKey(cfg.Mods[0])]].Status)
}

func TestRunNonTTYInstallReturnsErrorWhenRunnerMissing(t *testing.T) {
	restore := runInstallTranscriptProgram
	runInstallTranscriptProgram = nil
	t.Cleanup(func() { runInstallTranscriptProgram = restore })

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	items, indexByKey := buildInstallItems(models.ModsJSON{})
	_, err := runNonTTYInstall(context.Background(), cmd, installExecutionInput{
		meta:       config.NewMetadata("modlist.json"),
		cfg:        models.ModsJSON{},
		lock:       nil,
		deps:       installDeps{},
		items:      items,
		indexByKey: indexByKey,
	}, Result{})
	assert.ErrorContains(t, err, "missing bubble tea runner")
}

func TestRunNonTTYInstallReturnsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	restore := runInstallTranscriptProgram
	runInstallTranscriptProgram = func(model *installTranscriptModel, _ ...tea.ProgramOption) (tea.Model, error) {
		updated, cmd := model.Update(outputLineErrorMsg{Err: writeErr})
		runTeaCmd(cmd)
		return updated, nil
	}
	t.Cleanup(func() { runInstallTranscriptProgram = restore })

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	items, indexByKey := buildInstallItems(models.ModsJSON{})
	_, err := runNonTTYInstall(context.Background(), cmd, installExecutionInput{
		meta:       config.NewMetadata("modlist.json"),
		cfg:        models.ModsJSON{},
		lock:       nil,
		deps:       installDeps{},
		items:      items,
		indexByKey: indexByKey,
	}, Result{})
	assert.ErrorIs(t, err, writeErr)
}

func TestRunQuietInstallReturnsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	executionInput := installExecutionInput{
		meta: meta,
		cfg:  cfg,
		lock: []models.ModInstall{{
			Type:        models.MODRINTH,
			ID:          "alpha",
			Name:        "Alpha",
			FileName:    "alpha.jar",
			Hash:        "",
			DownloadURL: "https://example.invalid/alpha.jar",
		}},
		deps: installDeps{
			fs:     fs,
			logger: logger.New(io.Discard, io.Discard, false, false),
			output: output.New(io.Discard, io.Discard, false),
			runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
				return view.OutputLinesModel{Err: writeErr}, nil
			},
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		},
		items:      nil,
		indexByKey: nil,
	}
	executionInput.items, executionInput.indexByKey = buildInstallItems(cfg)

	cmd := &cobra.Command{}
	cmd.SetOut(errorWriter{err: writeErr})

	_, err := runQuietInstall(context.Background(), cmd, executionInput.deps, executionInput, Result{})
	assert.ErrorIs(t, err, writeErr)
}

func TestRenderInstallQuietFailureVariants(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	items := []installItem{
		{DisplayName: "Alpha", Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}, Status: installItemFailed, FailureReason: "reason"},
	}

	lines := renderInstallQuietFailure(view.ColorDisabled, installExecutionOutcome{errType: installExecutionErrorDownload, items: items})
	assert.Contains(t, strings.Join(lines, "\n"), "cmd.install.quiet.download_failed")

	lines = renderInstallQuietFailure(view.ColorDisabled, installExecutionOutcome{errType: installExecutionErrorWriteLock, lockPath: "/lock"})
	assert.Contains(t, strings.Join(lines, "\n"), "cmd.install.summary.write_failed_abort")

	lines = renderInstallQuietFailure(view.ColorDisabled, installExecutionOutcome{errType: installExecutionErrorWriteConfig, configPath: "/config"})
	assert.Contains(t, strings.Join(lines, "\n"), "cmd.install.summary.write_failed_abort")

	lines = renderInstallQuietFailure(view.ColorDisabled, installExecutionOutcome{errType: installExecutionErrorCanceled})
	assert.Contains(t, strings.Join(lines, "\n"), "cmd.install.summary.canceled")

	lines = renderInstallQuietFailure(view.ColorDisabled, installExecutionOutcome{errType: installExecutionErrorNone, err: errors.New("boom")})
	assert.Contains(t, strings.Join(lines, "\n"), "cmd.install.error.failed")
}

func TestRenderInstallQuietDownloadFailureWithoutFailedItems(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	lines := renderInstallQuietDownloadFailure(view.ColorDisabled, []installItem{
		{DisplayName: "Alpha", Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}, Status: installItemSuccess},
	})
	assert.Len(t, lines, 1)
	assert.Contains(t, lines[0], "cmd.install.quiet.download_failed")
}
