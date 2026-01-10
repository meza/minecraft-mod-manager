package update

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
	"github.com/muesli/termenv"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/cmd/mmm/install"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestColorModeForOutput(t *testing.T) {
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.ANSI })
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreColor)
	t.Cleanup(restoreTerminal)

	writer := &terminalWriter{}
	assert.Equal(t, view.ColorEnabled, colorModeForOutput(writer))

	restoreColorDisabled := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColorDisabled)
	assert.Equal(t, view.ColorDisabled, colorModeForOutput(writer))
}

func TestUpdateDisplayNameFallsBackToID(t *testing.T) {
	assert.Equal(t, "mod-id", updateDisplayName(models.Mod{ID: "mod-id"}))
}

func TestIsTerminalUpdateStatus(t *testing.T) {
	assert.False(t, isTerminalUpdateStatus(updateItemStatusPending))
	assert.False(t, isTerminalUpdateStatus(updateItemStatusUpdating))
	assert.False(t, isTerminalUpdateStatus(updateItemStatusDownloading))
	assert.True(t, isTerminalUpdateStatus(updateItemStatusUpToDate))
	assert.True(t, isTerminalUpdateStatus(updateItemStatusUpdated))
	assert.True(t, isTerminalUpdateStatus(updateItemStatusSkipped))
	assert.True(t, isTerminalUpdateStatus(updateItemStatusFailed))
}

func TestUpdateExecSenderSend(t *testing.T) {
	var received tea.Msg
	sender := updateExecSender{send: func(msg tea.Msg) { received = msg }}
	sender.Send(updateItemStatusMsg{index: 1})
	assert.IsType(t, updateItemStatusMsg{}, received)

	updateExecSender{}.Send(updateItemStatusMsg{index: 2})
}

func TestUpdateProgressSenderSendsProgress(t *testing.T) {
	var received tea.Msg
	sender := updateExecSender{send: func(msg tea.Msg) { received = msg }}
	progressSender := updateProgressSender{index: 2, sender: sender}

	progressSender.Send(httpclient.DownloadProgressMsg{
		Downloaded: 50,
		Total:      100,
		Ratio:      0.5,
	})

	typed, ok := received.(updateItemProgressMsg)
	if assert.True(t, ok) {
		assert.Equal(t, 2, typed.index)
		assert.Equal(t, 0.5, typed.progress.ratio)
		assert.Equal(t, int64(50), typed.progress.downloaded)
		assert.Equal(t, int64(100), typed.progress.total)
	}
}

func TestUpdateProgressSenderIgnoresUnknownMessages(t *testing.T) {
	sender := updateProgressSender{index: 1, sender: updateExecSender{}}
	sender.Send(struct{}{})
}

func TestSendUpdateDownloadStart(t *testing.T) {
	received := make(chan updateItemStatusMsg, 1)
	sender := updateExecSender{
		send: func(msg tea.Msg) {
			typed, ok := msg.(updateItemStatusMsg)
			if ok {
				received <- typed
			}
		},
	}

	sendUpdateDownloadStart(modUpdateCandidate{ConfigIndex: 3, Mod: models.Mod{ID: "mod"}}, sender)

	msg := <-received
	assert.Equal(t, 3, msg.index)
	assert.Equal(t, updateItemStatusDownloading, msg.status)
}

func TestSendUpdateDownloadStartSkipsPinnedOrNilSender(t *testing.T) {
	pinnedVersion := "1.0.0"
	sender := updateExecSender{send: func(tea.Msg) { t.Fatal("unexpected message") }}

	sendUpdateDownloadStart(modUpdateCandidate{
		ConfigIndex: 0,
		Mod:         models.Mod{ID: "mod", Version: &pinnedVersion},
	}, sender)

	sendUpdateDownloadStart(modUpdateCandidate{ConfigIndex: 1, Mod: models.Mod{ID: "mod"}}, updateExecSender{})
}

func TestWriteInteractiveUpdateTranscriptSuccess(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cmd := &cobra.Command{}
	buffer := &bytes.Buffer{}
	cmd.SetOut(buffer)

	model := &updateModel{
		items: []updateItem{
			{ConfigIndex: 0, DisplayName: "Alpha", Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}, Status: updateItemStatusUpdated},
			{ConfigIndex: 1, DisplayName: "Pinned", Mod: models.Mod{ID: "pinned", Type: models.MODRINTH}, Status: updateItemStatusSkipped},
		},
		colorMode: view.ColorDisabled,
		outcome: updateExecutionOutcome{
			items:   []updateItem{{Status: updateItemStatusUpdated}},
			errType: updateExecutionErrorNone,
		},
	}

	require.NoError(t, writeInteractiveUpdateTranscript(cmd, model))
	assert.Contains(t, buffer.String(), "cmd.update.section.updated")
	assert.Contains(t, buffer.String(), "cmd.update.section.skipped")
	assert.Contains(t, buffer.String(), "cmd.update.summary.success")
}

func TestWriteInteractiveUpdateTranscriptWriteLockFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cmd := &cobra.Command{}
	buffer := &bytes.Buffer{}
	cmd.SetOut(buffer)

	model := &updateModel{
		colorMode: view.ColorDisabled,
		outcome: updateExecutionOutcome{
			errType:  updateExecutionErrorWriteLock,
			lockPath: "/tmp/lock",
		},
	}

	require.NoError(t, writeInteractiveUpdateTranscript(cmd, model))
	assert.Contains(t, buffer.String(), "cmd.update.error.write_lock")
}

func TestWriteInteractiveUpdateTranscriptWriteConfigFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cmd := &cobra.Command{}
	buffer := &bytes.Buffer{}
	cmd.SetOut(buffer)

	model := &updateModel{
		colorMode: view.ColorDisabled,
		outcome: updateExecutionOutcome{
			errType:    updateExecutionErrorWriteConfig,
			configPath: "/tmp/modlist.json",
		},
	}

	require.NoError(t, writeInteractiveUpdateTranscript(cmd, model))
	assert.Contains(t, buffer.String(), "cmd.update.error.write_config")
}

func TestWriteInteractiveUpdateTranscriptUnknownFailure(t *testing.T) {
	cmd := &cobra.Command{}
	buffer := &bytes.Buffer{}
	cmd.SetOut(buffer)

	model := &updateModel{
		colorMode: view.ColorDisabled,
		outcome: updateExecutionOutcome{
			errType: updateExecutionErrorUnknown,
			err:     errors.New("boom"),
		},
	}

	require.NoError(t, writeInteractiveUpdateTranscript(cmd, model))
	assert.Contains(t, buffer.String(), "boom")
}

func TestWriteInteractiveUpdateTranscriptCanceled(t *testing.T) {
	cmd := &cobra.Command{}
	buffer := &bytes.Buffer{}
	cmd.SetOut(buffer)

	model := &updateModel{
		outcome: updateExecutionOutcome{errType: updateExecutionErrorCanceled},
	}

	require.NoError(t, writeInteractiveUpdateTranscript(cmd, model))
	assert.Equal(t, "", buffer.String())
}

func TestWriteInteractiveUpdateTranscriptNilModel(t *testing.T) {
	cmd := &cobra.Command{}
	buffer := &bytes.Buffer{}
	cmd.SetOut(buffer)

	require.NoError(t, writeInteractiveUpdateTranscript(cmd, nil))
	assert.Equal(t, "", buffer.String())
}

func TestNotifySkippedItems(t *testing.T) {
	items := []updateItem{
		{ConfigIndex: 0, Status: updateItemStatusSkipped},
		{ConfigIndex: 1, Status: updateItemStatusUpdating},
	}
	received := make(chan updateItemStatusMsg, 1)
	notifySkippedItems(items, updateExecSender{send: func(msg tea.Msg) {
		typed, ok := msg.(updateItemStatusMsg)
		if ok {
			received <- typed
		}
	}})

	msg := <-received
	assert.Equal(t, 0, msg.index)
	assert.Equal(t, updateItemStatusSkipped, msg.status)
}

func TestErrorTypeForUpdate(t *testing.T) {
	assert.Equal(t, updateExecutionErrorNone, errorTypeForUpdate(nil))
	assert.Equal(t, updateExecutionErrorCanceled, errorTypeForUpdate(context.Canceled))
	assert.Equal(t, updateExecutionErrorUnknown, errorTypeForUpdate(errors.New("boom")))
}

func TestUpdateItemFromOutcomeUpdatesFields(t *testing.T) {
	items := []updateItem{
		{ConfigIndex: 0, DisplayName: "Old", Status: updateItemStatusUpdating},
	}
	indexByKey := map[int]int{0: 0}
	updateItemFromOutcome(items, modUpdateOutcome{
		ConfigIndex: 0,
		Result:      updateOutcomeFailed,
		FailReason:  "boom",
		NewName:     "New Name",
	}, indexByKey)
	assert.Equal(t, updateItemStatusFailed, items[0].Status)
	assert.Equal(t, "boom", items[0].FailReason)
	assert.Equal(t, "New Name", items[0].DisplayName)

	updateItemFromOutcome(items, modUpdateOutcome{ConfigIndex: 10}, indexByKey)
}

func TestHandleQuietUpdateResultBranches(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	outputErr := errors.New("output failed")
	original := runTeaProgram
	runTeaProgram = func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		if linesModel, ok := model.(outputLinesModel); ok {
			linesModel.Err = outputErr
			return linesModel, nil
		}
		return model, nil
	}
	t.Cleanup(func() { runTeaProgram = original })

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)

	lockErr := errors.New("lock failed")
	_, err := handleQuietUpdateResult(cmd, updateExecutionOutcome{
		errType: updateExecutionErrorWriteLock,
		err:     lockErr,
	})
	assert.ErrorIs(t, err, outputErr)

	runTeaProgram = func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		return model, nil
	}

	configErr := errors.New("config failed")
	_, err = handleQuietUpdateResult(cmd, updateExecutionOutcome{
		errType: updateExecutionErrorWriteConfig,
		err:     configErr,
	})
	assert.ErrorIs(t, err, configErr)

	_, err = handleQuietUpdateResult(cmd, updateExecutionOutcome{
		errType: updateExecutionErrorCanceled,
		err:     context.Canceled,
	})
	assert.ErrorIs(t, err, context.Canceled)

	unknownErr := errors.New("unknown")
	_, err = handleQuietUpdateResult(cmd, updateExecutionOutcome{
		errType: updateExecutionErrorUnknown,
		err:     unknownErr,
	})
	assert.ErrorIs(t, err, unknownErr)

	_, err = handleQuietUpdateResult(cmd, updateExecutionOutcome{
		items: []updateItem{
			{ConfigIndex: 0, Status: updateItemStatusFailed, FailReason: "boom"},
		},
	})
	assert.ErrorIs(t, err, errUpdateFailures)

	_, err = handleQuietUpdateResult(cmd, updateExecutionOutcome{
		items: []updateItem{
			{ConfigIndex: 0, Status: updateItemStatusUpToDate},
		},
	})
	assert.NoError(t, err)
}

func TestLoadUpdateContextSuccess(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.ANSI })
	t.Cleanup(restoreTerminal)
	t.Cleanup(restoreColor)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{Mods: []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}}}
	lock := []models.ModInstall{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	cmd := &cobra.Command{}
	cmd.SetOut(&terminalWriter{})

	ctx, err := loadUpdateContext(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs: fs,
	})
	require.NoError(t, err)
	assert.Len(t, ctx.cfg.Mods, 1)
	assert.Len(t, ctx.lock, 1)
	assert.Equal(t, view.ColorEnabled, ctx.colorMode)
}

func TestLoadUpdateContextReturnsLockReadError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{Mods: []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}}}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	readErr := errors.New("read failed")
	failFs := openErrorFs{Fs: fs, failPath: meta.LockPath(), err: readErr}
	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)

	_, err := loadUpdateContext(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs: failFs,
	})
	assert.ErrorIs(t, err, readErr)
}

func TestNewUpdateDepsWiresDefaults(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{})
	installCommand := install.Command()
	installCommand.SetOut(io.Discard)
	installCommand.SetErr(io.Discard)
	deps := newUpdateDeps(common, installCommand, updateOptions{})

	assert.NotNil(t, deps.fs)
	assert.NotNil(t, deps.logger)
	assert.NotNil(t, deps.output)
	assert.NotNil(t, deps.clients.Modrinth)
	assert.NotNil(t, deps.clients.Curseforge)
	assert.NotNil(t, deps.fetchMod)
	assert.NotNil(t, deps.downloader)
	assert.NotNil(t, deps.install)
	assert.NotNil(t, deps.telemetry)
	assert.NotNil(t, deps.runTea)
	assert.NotNil(t, deps.runInit)

	tempDir := t.TempDir()
	meta := config.NewMetadata(filepath.Join(tempDir, "modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods", Mods: []models.Mod{}}
	fs := afero.NewOsFs()
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "mods"), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	_, err := deps.install(context.Background(), installCommand, meta.ConfigPath, true, false)
	assert.NoError(t, err)
}

func TestNewUpdateDepsRunInitUsesInteractiveInit(t *testing.T) {
	originalRunInteractiveInit := runInteractiveInit
	t.Cleanup(func() { runInteractiveInit = originalRunInteractiveInit })

	called := false
	runInteractiveInit = func(ctx context.Context, cmd *cobra.Command, deps initCmd.InteractiveInitDeps, opts initCmd.InteractiveInitOptions) error {
		called = true
		assert.Equal(t, "/cfg/modlist.json", opts.ConfigPath)
		assert.True(t, opts.Quiet)
		assert.True(t, opts.Debug)
		return nil
	}

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{})
	deps := newUpdateDeps(common, cmd, updateOptions{Quiet: true, Debug: true})

	err := deps.runInit(context.Background(), cmd, initRequest{ConfigPath: "/cfg/modlist.json"})
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestConfigMissingPromptErrorUnattended(t *testing.T) {
	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})

	err := configMissingPromptError(updateOptions{Unattended: true}, command, config.NewMetadata("/cfg/modlist.json"))
	assert.Error(t, err)
}

func TestConfigMissingPromptErrorNoTTY(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return false })
	t.Cleanup(restoreTerminal)

	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})

	err := configMissingPromptError(updateOptions{}, command, config.NewMetadata("/cfg/modlist.json"))
	assert.Error(t, err)
}

func TestConfigMissingPromptErrorReturnsNilWhenPromptAllowed(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	command := &cobra.Command{}
	command.SetIn(terminalReader{Reader: bytes.NewBuffer(nil)})
	command.SetOut(&terminalWriter{})

	err := configMissingPromptError(updateOptions{}, command, config.NewMetadata("/cfg/modlist.json"))
	assert.NoError(t, err)
}

func TestWriteConfigMissingOutputColorEnabled(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreColor)

	originalRunTeaProgram := runTeaProgram
	runTeaProgram = func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		return view.OutputLinesModel{}, nil
	}
	t.Cleanup(func() { runTeaProgram = originalRunTeaProgram })

	command := &cobra.Command{}
	command.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	command.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})

	err := writeConfigMissingOutput(command, config.NewMetadata("/cfg/modlist.json"))
	assert.NoError(t, err)
}

func TestLoadUpdateContextReturnsConfigReadError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{Mods: []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}}}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	readErr := errors.New("read failed")
	failFs := openErrorFs{Fs: fs, failPath: meta.ConfigPath, err: readErr}
	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)

	_, err := loadUpdateContext(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs: failFs,
	})
	assert.ErrorIs(t, err, readErr)
}

func TestHandleQuietUpdateResultReturnsOutputErrorForFailureLines(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	outputErr := errors.New("output failed")
	original := runTeaProgram
	runTeaProgram = func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		if linesModel, ok := model.(outputLinesModel); ok {
			linesModel.Err = outputErr
			return linesModel, nil
		}
		return model, nil
	}
	t.Cleanup(func() { runTeaProgram = original })

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)

	_, err := handleQuietUpdateResult(cmd, updateExecutionOutcome{
		items: []updateItem{
			{ConfigIndex: 0, Status: updateItemStatusFailed, FailReason: "boom"},
		},
	})
	assert.ErrorIs(t, err, outputErr)
}

func TestHandleQuietUpdateResultReturnsOutputErrorForWriteConfig(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	outputErr := errors.New("output failed")
	original := runTeaProgram
	runTeaProgram = func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		if linesModel, ok := model.(outputLinesModel); ok {
			linesModel.Err = outputErr
			return linesModel, nil
		}
		return model, nil
	}
	t.Cleanup(func() { runTeaProgram = original })

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)

	_, err := handleQuietUpdateResult(cmd, updateExecutionOutcome{
		errType: updateExecutionErrorWriteConfig,
		err:     errors.New("config failed"),
	})
	assert.ErrorIs(t, err, outputErr)
}

func TestHandleQuietUpdateResultReturnsHandledErrorForWriteLock(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	original := runTeaProgram
	runTeaProgram = func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		return model, nil
	}
	t.Cleanup(func() { runTeaProgram = original })

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)

	lockErr := errors.New("lock failed")
	_, err := handleQuietUpdateResult(cmd, updateExecutionOutcome{
		errType: updateExecutionErrorWriteLock,
		err:     lockErr,
	})
	assert.ErrorIs(t, err, lockErr)
}

func TestHandleQuietUpdateResultReturnsOutputErrorForUnknown(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	outputErr := errors.New("output failed")
	original := runTeaProgram
	runTeaProgram = func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		if linesModel, ok := model.(outputLinesModel); ok {
			linesModel.Err = outputErr
			return linesModel, nil
		}
		return model, nil
	}
	t.Cleanup(func() { runTeaProgram = original })

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)

	_, err := handleQuietUpdateResult(cmd, updateExecutionOutcome{
		errType: updateExecutionErrorUnknown,
		err:     errors.New("unknown"),
	})
	assert.ErrorIs(t, err, outputErr)
}

func TestRenderUpdateQuietFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	lines := renderUpdateQuietFailure(updateResultsViewInput{
		items: []updateItem{
			{ConfigIndex: 0, Status: updateItemStatusFailed, FailReason: "boom"},
		},
		colorMode: view.ColorDisabled,
	})
	assert.Contains(t, strings.Join(lines, "\n"), "cmd.update.summary.incomplete")
	assert.Contains(t, strings.Join(lines, "\n"), "cmd.update.section.failed")
}

func TestRunUpdateInteractiveAndTranscript(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	meta := config.NewMetadata("modlist.json")
	filesystem := afero.NewMemMapFs()
	require.NoError(t, filesystem.MkdirAll(meta.Dir(), 0755))

	cfg := models.ModsJSON{Mods: []models.Mod{}}
	lock := []models.ModInstall{}
	items := []updateItem{}
	indexByKey := map[int]int{}

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)

	interactiveOutcome, err := runUpdateInteractive(context.Background(), cmd, updateExecutionInput{
		meta:       meta,
		cfg:        &cfg,
		lock:       lock,
		items:      items,
		indexByKey: indexByKey,
		colorMode:  view.ColorDisabled,
		deps: updateDeps{
			fs: filesystem,
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, updateExecutionErrorNone, interactiveOutcome.errType)

	transcriptOutcome, err := runUpdateTranscript(context.Background(), cmd, updateExecutionInput{
		meta:       meta,
		cfg:        &cfg,
		lock:       lock,
		items:      items,
		indexByKey: indexByKey,
		colorMode:  view.ColorDisabled,
		deps: updateDeps{
			fs: filesystem,
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, updateExecutionErrorNone, transcriptOutcome.errType)
}

func TestUpdateOutcomeFromModelUnexpectedType(t *testing.T) {
	_, err := updateOutcomeFromModel(outputLinesModel{})
	assert.Error(t, err)
}
