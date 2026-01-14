package prune

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestDefaultPruneDepsSetsRunners(t *testing.T) {
	originalInit := runInteractiveInit
	t.Cleanup(func() {
		runInteractiveInit = originalInit
	})
	seenConfigPath := ""
	runInteractiveInit = func(_ context.Context, _ *cobra.Command, _ initCmd.InteractiveInitDeps, options initCmd.InteractiveInitOptions) error {
		seenConfigPath = options.ConfigPath
		return nil
	}

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	deps := defaultPruneDeps(cmd, pruneOptions{})
	assert.NotNil(t, deps.runTea)
	assert.NotNil(t, deps.runInit)

	require.NoError(t, deps.runInit(context.Background(), cmd, initRequest{configPath: "/cfg/modlist.json"}))
	assert.Equal(t, "/cfg/modlist.json", seenConfigPath)
}

func TestSortUnmanagedFilesOrdersByBase(t *testing.T) {
	files := []string{
		filepath.FromSlash("/mods/B.jar"),
		filepath.FromSlash("/mods/a.jar"),
		filepath.FromSlash("/mods/a-2.jar"),
	}

	sorted := sortUnmanagedFiles(files)
	assert.Equal(t, filepath.FromSlash("/mods/a-2.jar"), sorted[0])
	assert.Equal(t, filepath.FromSlash("/mods/a.jar"), sorted[1])
	assert.Equal(t, filepath.FromSlash("/mods/B.jar"), sorted[2])
}

func TestSortUnmanagedFilesTiebreaksByFullPath(t *testing.T) {
	files := []string{
		filepath.FromSlash("/mods/Alpha.jar"),
		filepath.FromSlash("/mods/alpha.jar"),
	}

	sorted := sortUnmanagedFiles(files)
	assert.Equal(t, filepath.FromSlash("/mods/Alpha.jar"), sorted[0])
	assert.Equal(t, filepath.FromSlash("/mods/alpha.jar"), sorted[1])
}

func TestConfirmPruneQuietReturnsOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	writeErr := errors.New("write failed")
	shouldDelete, err := confirmPrune(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{Err: writeErr}, nil
		},
	}, pruneOptions{Quiet: true}, interaction.ExecutionModeUnattended, view.ColorDisabled, []string{"file.jar"})

	assert.False(t, shouldDelete)
	assert.ErrorIs(t, err, writeErr)
}

func TestRunForceInteractiveDeletionUsesFallbackRunner(t *testing.T) {
	originalRunner := runTeaProgram
	t.Cleanup(func() {
		runTeaProgram = originalRunner
	})

	ran := false
	runTeaProgram = func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
		ran = true
		return pruneForceModel{
			results: []pruneFileResult{{Path: "/mods/file.jar", Status: pruneFileStatusDeleted}},
			done:    true,
		}, nil
	}

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	results, deleteErr, runErr := runForceInteractiveDeletion(cmd, pruneDeps{}, view.ColorDisabled, []string{"file.jar"})
	assert.True(t, ran)
	assert.NoError(t, runErr)
	assert.NoError(t, deleteErr)
	assert.Len(t, results, 1)
}

func TestRunForceInteractiveDeletionReturnsResultError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	results, deleteErr, runErr := runForceInteractiveDeletion(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return unexpectedModel{}, nil
		},
	}, view.ColorDisabled, []string{"file.jar"})

	assert.Error(t, runErr)
	assert.Empty(t, results)
	assert.NoError(t, deleteErr)
}

func TestRenderPruneFileLineStatusVariants(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	pending := renderPruneFileLine(view.ColorDisabled, pruneFileResult{Path: "file.jar", Status: pruneFileStatusDeleting})
	assert.Contains(t, pending, "\u23F3")

	success := renderPruneFileLine(view.ColorDisabled, pruneFileResult{Path: "file.jar", Status: pruneFileStatusDeleted})
	assert.Contains(t, success, "\u2705")

	failed := renderPruneFileLine(view.ColorDisabled, pruneFileResult{Path: "file.jar", Status: pruneFileStatusFailed, Err: errors.New("boom")})
	assert.Contains(t, failed, "\u274C")
	assert.Contains(t, failed, i18n.T("cmd.file.delete_failed", &i18n.Tvars{
		Data: &i18n.TData{"reason": "boom"},
	}))
}

func TestRenderDeletingListIncludesHeader(t *testing.T) {
	list := renderDeletingList(view.ColorDisabled, []string{filepath.FromSlash("/mods/file.jar")})
	assert.Contains(t, list, i18n.T("cmd.prune.header.deleting", nil))
	assert.Contains(t, list, "file.jar")
}

func TestWriteDeletingOutputReturnsErrorOnOutputFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := writeDeletingOutput(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{Err: writeErr}, nil
		},
	}, view.ColorDisabled, []string{"file.jar"})
	assert.ErrorIs(t, err, writeErr)
}

func TestShouldDeleteUnmanagedInteractiveConfirmed(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	confirmed, err := shouldDeleteUnmanaged(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return pruneConfirmModel{confirmed: true}, nil
		},
	}, interaction.ExecutionModeInteractive, view.ColorDisabled, []string{"/mods/unmanaged.jar"})

	assert.True(t, confirmed)
	assert.NoError(t, err)
}

func TestShouldDeleteUnmanagedInteractiveCanceled(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	confirmed, err := shouldDeleteUnmanaged(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return pruneConfirmModel{canceled: true}, nil
		},
	}, interaction.ExecutionModeInteractive, view.ColorDisabled, []string{"/mods/unmanaged.jar"})

	assert.False(t, confirmed)
	assert.NoError(t, err)
}

func TestShouldDeleteUnmanagedNonInteractiveReturnsHandled(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	confirmed, err := shouldDeleteUnmanaged(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{}, nil
		},
	}, interaction.ExecutionModeUnattended, view.ColorDisabled, []string{"/mods/unmanaged.jar"})

	assert.False(t, confirmed)
	assert.True(t, clierrors.IsHandled(err))
}

func TestHandlePruneFailureMarksHandled(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := handlePruneFailure(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{}, nil
		},
	}, errors.New("boom"))
	assert.True(t, clierrors.IsHandled(err))
}

func TestHandlePruneFailureReturnsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := handlePruneFailure(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{Err: writeErr}, nil
		},
	}, errors.New("boom"))
	assert.ErrorIs(t, err, writeErr)
}

func TestWritePruneFailureOutputReturnsErrorOnOutputFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := writePruneFailureOutput(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{Err: writeErr}, nil
		},
	}, errors.New("boom"))
	assert.ErrorIs(t, err, writeErr)
}

func TestWritePruneFailureOutputUsesColorStyles(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreColor)

	cmd := &cobra.Command{}
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(&bytes.Buffer{})

	err := writePruneFailureOutput(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{}, nil
		},
	}, errors.New("boom"))
	assert.NoError(t, err)
}

func TestWriteConfigMissingOutputReturnsErrorOnOutputFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := writeConfigMissingOutput(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{Err: writeErr}, nil
		},
	}, config.NewMetadata("/cfg/modlist.json"))
	assert.ErrorIs(t, err, writeErr)
}

func TestWriteConfigMissingOutputUsesColorStyles(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreColor)

	cmd := &cobra.Command{}
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(&bytes.Buffer{})

	err := writeConfigMissingOutput(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{}, nil
		},
	}, config.NewMetadata("/cfg/modlist.json"))
	assert.NoError(t, err)
}

func TestWriteLockMissingOutputReturnsErrorOnOutputFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := writeLockMissingOutput(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{Err: writeErr}, nil
		},
	}, &lockMissingError{message: "lock missing"})
	assert.ErrorIs(t, err, writeErr)
}

func TestWriteLockMissingOutputUsesColorStyles(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreColor)

	cmd := &cobra.Command{}
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(&bytes.Buffer{})

	err := writeLockMissingOutput(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{}, nil
		},
	}, &lockMissingError{message: "lock missing"})
	assert.NoError(t, err)
}

func TestWriteNoUnmanagedOutputReturnsErrorOnOutputFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := writeNoUnmanagedOutput(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{Err: writeErr}, nil
		},
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestWriteNoUnmanagedOutputUsesColorStyles(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreColor)

	cmd := &cobra.Command{}
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(&bytes.Buffer{})

	err := writeNoUnmanagedOutput(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{}, nil
		},
	})
	assert.NoError(t, err)
}

func TestColorModeForWriter(t *testing.T) {
	assert.Equal(t, view.ColorDisabled, colorModeForWriter(nil))

	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreColor)

	cmd := &cobra.Command{}
	cmd.SetOut(&fakeTerminalWriter{})
	assert.Equal(t, view.ColorEnabled, colorModeForWriter(cmd))

	cmd = &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	assert.Equal(t, view.ColorDisabled, colorModeForWriter(cmd))
}

func TestConfigMissingPromptErrorInteractiveAllowsPrompt(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(&fakeTerminalWriter{})

	err := configMissingPromptError(pruneOptions{}, cmd, config.NewMetadata("/cfg/modlist.json"))
	assert.NoError(t, err)
}

func TestEnsurePruneConfigMissingNoPrompt(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(io.Discard)

	state, err := ensurePruneConfig(context.Background(), cmd, pruneOptions{Unattended: true}, pruneDeps{
		fs: afero.NewMemMapFs(),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{}, nil
		},
	}, config.NewMetadata("/cfg/modlist.json"))

	assert.Equal(t, pruneConfigState{}, state)
	assert.True(t, clierrors.IsHandled(err))
}

func TestEnsurePruneConfigPromptDeclined(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(io.Discard)

	state, err := ensurePruneConfig(context.Background(), cmd, pruneOptions{}, pruneDeps{
		fs: afero.NewMemMapFs(),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: false}, nil
		},
	}, config.NewMetadata("/cfg/modlist.json"))

	assert.NoError(t, err)
	assert.False(t, state.ShouldContinue)
}

func TestEnsurePruneConfigRunInitCanceled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(io.Discard)

	state, err := ensurePruneConfig(context.Background(), cmd, pruneOptions{}, pruneDeps{
		fs: afero.NewMemMapFs(),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
		runInit: func(context.Context, *cobra.Command, initRequest) error {
			return initCmd.ErrInitCanceled
		},
	}, config.NewMetadata("/cfg/modlist.json"))

	assert.NoError(t, err)
	assert.False(t, state.ShouldContinue)
}

func TestEnsurePruneConfigRunInitMissingRunner(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(io.Discard)

	state, err := ensurePruneConfig(context.Background(), cmd, pruneOptions{}, pruneDeps{
		fs: afero.NewMemMapFs(),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
	}, config.NewMetadata("/cfg/modlist.json"))

	assert.Error(t, err)
	assert.Equal(t, pruneConfigState{}, state)
}

func TestEnsurePruneConfigConfigPresentLockMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(io.Discard)

	state, err := ensurePruneConfig(context.Background(), cmd, pruneOptions{}, pruneDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{}, nil
		},
	}, meta)

	assert.Equal(t, pruneConfigState{}, state)
	assert.True(t, clierrors.IsHandled(err))
}

func TestEnsurePruneConfigSuccessAfterInit(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(io.Discard)

	state, err := ensurePruneConfig(context.Background(), cmd, pruneOptions{}, pruneDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
		runInit: func(context.Context, *cobra.Command, initRequest) error {
			require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
			require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
			require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
			require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))
			return nil
		},
	}, meta)

	assert.NoError(t, err)
	assert.True(t, state.ShouldContinue)
}

func TestRunPruneForceInteractiveWritesDeleting(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "extra.jar"), []byte("data"), 0644))

	outputBuffer := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(io.Discard)

	runner := func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		if forceModel, ok := model.(pruneForceModel); ok {
			results, deleteErr := forceModel.deleteFunc()
			forceModel.results = results
			forceModel.deleteErr = deleteErr
			forceModel.done = true
			if _, writeErr := fmt.Fprint(outputBuffer, forceModel.View()); writeErr != nil {
				return forceModel, writeErr
			}
			return forceModel, nil
		}
		if outputModel, ok := model.(view.OutputLinesModel); ok {
			if _, writeErr := fmt.Fprint(outputBuffer, outputModel.View()); writeErr != nil {
				return outputModel, writeErr
			}
			return outputModel, nil
		}
		return model, nil
	}

	_, err := runPrune(context.Background(), cmd, pruneOptions{ConfigPath: meta.ConfigPath, Force: true}, pruneDeps{
		fs:     fs,
		runTea: runner,
	})
	assert.NoError(t, err)
	assert.Contains(t, outputBuffer.String(), i18n.T("cmd.prune.header.deleted", nil))
	assert.NotContains(t, outputBuffer.String(), i18n.T("cmd.prune.header.deleting", nil))
}

func TestRunPruneQuietSkipsSuccessOutput(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "extra.jar"), []byte("data"), 0644))

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(io.Discard)

	_, err := runPrune(context.Background(), cmd, pruneOptions{ConfigPath: meta.ConfigPath, Force: true, Quiet: true}, pruneDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			t.Fatal("unexpected output in quiet mode")
			return view.OutputLinesModel{}, nil
		},
	})
	assert.NoError(t, err)
}

func TestRenderDeleteSuccessSummaryUsesCtaStyle(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreColor)

	summary := renderDeleteSuccessSummary(view.ColorEnabled)
	assert.True(t, strings.Contains(summary, i18n.T("cmd.prune.summary.success_hint", nil)))
}
