package prune

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/tui"
)

type errorWriter struct {
	err error
}

func (writer errorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

type countingErrorWriter struct {
	failOn int
	count  int
	err    error
}

func (writer *countingErrorWriter) Write(payload []byte) (int, error) {
	writer.count++
	if writer.count == writer.failOn {
		return 0, writer.err
	}
	return len(payload), nil
}

type errorReader struct {
	err error
}

func (reader errorReader) Read([]byte) (int, error) {
	return 0, reader.err
}

type errorTerminalReader struct {
	err error
}

func (reader errorTerminalReader) Read([]byte) (int, error) {
	return 0, reader.err
}

func (reader errorTerminalReader) Fd() uintptr {
	return 0
}

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

func TestApplyPruneCommandErrorPolicy(t *testing.T) {
	cmd := &cobra.Command{}
	applyPruneCommandErrorPolicy(cmd, nil)
	assert.False(t, cmd.SilenceErrors)
	assert.False(t, cmd.SilenceUsage)

	handledErr := clierrors.MarkHandled(errors.New("handled"))
	cmd = &cobra.Command{}
	applyPruneCommandErrorPolicy(cmd, handledErr)
	assert.True(t, cmd.SilenceErrors)
	assert.True(t, cmd.SilenceUsage)

	cmd = &cobra.Command{}
	applyPruneCommandErrorPolicy(cmd, errors.New("unhandled"))
	assert.False(t, cmd.SilenceErrors)
	assert.True(t, cmd.SilenceUsage)
}

func TestRecordPruneTelemetryExitCodes(t *testing.T) {
	var payloads []telemetry.CommandTelemetry
	recorder := func(payload telemetry.CommandTelemetry) {
		payloads = append(payloads, payload)
	}

	recordPruneTelemetry(recorder, pruneOptions{Force: true}, 2, nil)
	recordPruneTelemetry(recorder, pruneOptions{}, 0, errors.New("boom"))

	require.Len(t, payloads, 2)
	assert.Equal(t, 0, payloads[0].ExitCode)
	assert.Equal(t, 1, payloads[1].ExitCode)
	assert.Equal(t, 2, payloads[0].Extra["deletedCount"])
}

func TestRunPruneNoUnmanagedReturnsOutputError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	outErr := errors.New("write failed")
	_, err := runPrune(context.Background(), cmd, pruneOptions{ConfigPath: meta.ConfigPath}, pruneDeps{
		fs:     fs,
		output: output.New(errorWriter{err: outErr}, io.Discard, false),
		telemetry: func(telemetry.CommandTelemetry) {
		},
	})

	assert.ErrorIs(t, err, outErr)
}

func TestShouldDeleteUnmanagedReturnsListError(t *testing.T) {
	restoreTerminal := tui.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(io.Discard)

	outErr := errors.New("list failed")
	result, err := shouldDeleteUnmanaged(cmd, pruneOptions{}, pruneDeps{
		output: output.New(errorWriter{err: outErr}, io.Discard, false),
	}, tui.ColorDisabled, []string{"/mods/unmanaged.jar"})

	assert.False(t, result)
	assert.ErrorIs(t, err, outErr)
}

func TestShouldDeleteUnmanagedUnattendedReturnsListError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(io.Discard)

	outErr := errors.New("list failed")
	result, err := shouldDeleteUnmanaged(cmd, pruneOptions{Unattended: true}, pruneDeps{
		output: output.New(errorWriter{err: outErr}, io.Discard, false),
	}, tui.ColorDisabled, []string{"/mods/unmanaged.jar"})

	assert.False(t, result)
	assert.ErrorIs(t, err, outErr)
}

func TestShouldDeleteUnmanagedUnattendedReturnsWarningError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(io.Discard)

	outErr := errors.New("warn failed")
	result, err := shouldDeleteUnmanaged(cmd, pruneOptions{Unattended: true}, pruneDeps{
		output: output.New(io.Discard, errorWriter{err: outErr}, false),
	}, tui.ColorDisabled, []string{"/mods/unmanaged.jar"})

	assert.False(t, result)
	assert.ErrorIs(t, err, outErr)
}

func TestShouldDeleteUnmanagedReturnsConfirmError(t *testing.T) {
	restoreTerminal := tui.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	readErr := errors.New("read failed")
	cmd := &cobra.Command{}
	cmd.SetIn(errorTerminalReader{err: readErr})
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(io.Discard)

	_, err := shouldDeleteUnmanaged(cmd, pruneOptions{}, pruneDeps{
		output: output.New(io.Discard, io.Discard, false),
	}, tui.ColorEnabled, []string{"/mods/unmanaged.jar"})

	assert.ErrorIs(t, err, readErr)
}

func TestReadLockRequiredReturnsErrorOnInvalidJSON(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, afero.WriteFile(fs, meta.LockPath(), []byte("not-json"), 0644))

	_, err := readLockRequired(context.Background(), fs, meta)
	assert.Error(t, err)
}

func TestRunPruneConfigReadError(t *testing.T) {
	fs := afero.NewMemMapFs()

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := runPrune(context.Background(), cmd, pruneOptions{ConfigPath: "/cfg/missing.json"}, pruneDeps{
		fs:     fs,
		output: output.New(io.Discard, io.Discard, false),
		telemetry: func(telemetry.CommandTelemetry) {
		},
	})

	assert.Error(t, err)
}

func TestRunPruneListUnmanagedError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := runPrune(context.Background(), cmd, pruneOptions{
		ConfigPath: meta.ConfigPath,
		Force:      true,
	}, pruneDeps{
		fs:     fs,
		output: output.New(io.Discard, io.Discard, false),
		telemetry: func(telemetry.CommandTelemetry) {
		},
	})

	assert.Error(t, err)
}

func TestRunPruneDeleteError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	unmanagedPath := filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar")
	require.NoError(t, afero.WriteFile(fs, unmanagedPath, []byte("data"), 0644))

	wrapped := removeErrorFs{Fs: fs, failPath: unmanagedPath, err: errors.New("remove failed")}

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := runPrune(context.Background(), cmd, pruneOptions{
		ConfigPath: meta.ConfigPath,
		Force:      true,
	}, pruneDeps{
		fs:     wrapped,
		output: output.New(io.Discard, io.Discard, false),
		telemetry: func(telemetry.CommandTelemetry) {
		},
	})

	assert.Error(t, err)
}

func TestHandleLockReadErrorReturnsOriginalError(t *testing.T) {
	original := errors.New("original")
	err := handleLockReadError(original, output.New(io.Discard, io.Discard, false))
	assert.Equal(t, original, err)
}

func TestHandleLockReadErrorReturnsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	err := handleLockReadError(&lockMissingError{message: "lock missing"}, output.New(io.Discard, errorWriter{err: writeErr}, false))
	assert.ErrorIs(t, err, writeErr)
}

func TestListUnmanagedFilesReturnsErrorOnMissingModsDir(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))

	_, err := listUnmanagedFiles(fs, meta, cfg, []models.ModInstall{})
	assert.Error(t, err)
}

func TestListJarFilesSkipsDirsNonJarAndIgnores(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, fs.MkdirAll(filepath.Join(meta.ModsFolderPath(cfg), "nested"), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "note.txt"), []byte("data"), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "good.jar"), []byte("data"), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "ignored.jar"), []byte("data"), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), ".mmmignore"), []byte("ignored.jar\n"), 0644))

	files, err := listJarFiles(fs, meta, cfg)
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, filepath.Join(meta.ModsFolderPath(cfg), "good.jar"), files[0])
}

func TestListJarFilesReturnsAbsError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	originalAbs := absPath
	absPath = func(string) (string, error) {
		return "", errors.New("abs failed")
	}
	t.Cleanup(func() {
		absPath = originalAbs
	})

	_, err := listJarFiles(fs, meta, cfg)
	assert.Error(t, err)
}

func TestListJarFilesReturnsIgnorePatternError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "good.jar"), []byte("data"), 0644))

	wrapped := statErrorFs{
		Fs:       fs,
		failPath: filepath.Join(meta.Dir(), ".mmmignore"),
		err:      errors.New("stat failed"),
	}

	_, err := listJarFiles(wrapped, meta, cfg)
	assert.Error(t, err)
}

func TestPrintUnmanagedListHeaderError(t *testing.T) {
	writeErr := errors.New("write failed")
	err := printUnmanagedList(output.New(errorWriter{err: writeErr}, io.Discard, false), tui.ColorDisabled, []string{"file.jar"})
	assert.ErrorIs(t, err, writeErr)
}

func TestPrintUnmanagedListEntryError(t *testing.T) {
	writer := &countingErrorWriter{failOn: 2, err: errors.New("write failed")}
	err := printUnmanagedList(output.New(writer, io.Discard, false), tui.ColorDisabled, []string{"file-1.jar", "file-2.jar"})
	assert.ErrorIs(t, err, writer.err)
}

func TestConfirmDeletionWriteError(t *testing.T) {
	writeErr := errors.New("write failed")
	_, err := confirmDeletion(strings.NewReader("y\n"), errorWriter{err: writeErr}, tui.ColorDisabled)
	assert.ErrorIs(t, err, writeErr)
}

func TestConfirmDeletionReadError(t *testing.T) {
	readErr := errors.New("read failed")
	_, err := confirmDeletion(errorReader{err: readErr}, io.Discard, tui.ColorDisabled)
	assert.ErrorIs(t, err, readErr)
}

func TestReportPromptDisabledFirstWriteError(t *testing.T) {
	writeErr := errors.New("write failed")
	err := reportPromptDisabled(output.New(io.Discard, errorWriter{err: writeErr}, false))
	assert.ErrorIs(t, err, writeErr)
}

func TestReportPromptDisabledSecondWriteError(t *testing.T) {
	writer := &countingErrorWriter{failOn: 2, err: errors.New("write failed")}
	err := reportPromptDisabled(output.New(io.Discard, writer, false))
	assert.ErrorIs(t, err, writer.err)
}

func TestDeleteUnmanagedFilesRemoveError(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/unmanaged.jar")
	require.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, afero.WriteFile(fs, path, []byte("data"), 0644))

	wrapped := removeErrorFs{Fs: fs, failPath: path, err: errors.New("remove failed")}
	_, err := deleteUnmanagedFiles(pruneDeps{
		fs:     wrapped,
		output: output.New(io.Discard, io.Discard, false),
	}, []string{path})

	assert.Error(t, err)
}

func TestDeleteUnmanagedFilesOutputError(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/unmanaged.jar")
	require.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, afero.WriteFile(fs, path, []byte("data"), 0644))

	writeErr := errors.New("write failed")
	_, err := deleteUnmanagedFiles(pruneDeps{
		fs:     fs,
		output: output.New(errorWriter{err: writeErr}, io.Discard, false),
	}, []string{path})

	assert.ErrorIs(t, err, writeErr)
}

func TestRemoveFileForceMissingFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	missingPath := filepath.FromSlash("/mods/missing.jar")
	assert.NoError(t, removeFileForce(fs, missingPath))
}
