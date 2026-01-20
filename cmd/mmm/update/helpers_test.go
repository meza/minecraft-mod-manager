package update

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/meza/minecraft-mod-manager/cmd/mmm/install"
	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadClientPrefersCurseforge(t *testing.T) {
	curseforgeClient := noopDoer{}
	modrinthClient := noopDoer{}

	chosen := platform.PreferredDownloadClient(platform.Clients{
		Curseforge: curseforgeClient,
		Modrinth:   modrinthClient,
	})
	assert.Equal(t, curseforgeClient, chosen)
}

func TestDownloadClientFallsBackToModrinth(t *testing.T) {
	modrinthClient := noopDoer{}

	chosen := platform.PreferredDownloadClient(platform.Clients{
		Modrinth: modrinthClient,
	})
	assert.Equal(t, modrinthClient, chosen)
}

func TestEffectiveAllowedReleaseTypesUsesOverrides(t *testing.T) {
	mod := models.Mod{AllowedReleaseTypes: []models.ReleaseType{models.Beta}}
	cfg := models.ModsJSON{DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release}}

	assert.Equal(t, mod.AllowedReleaseTypes, models.EffectiveAllowedReleaseTypes(mod, cfg))
}

func TestLockIndexForModMatchesTypeAndID(t *testing.T) {
	mod := models.Mod{Type: models.CURSEFORGE, ID: "123"}
	lock := []models.ModInstall{
		{Type: models.MODRINTH, ID: "abc"},
		{Type: models.CURSEFORGE, ID: "123"},
	}

	assert.Equal(t, 1, models.LockIndexForMod(mod, lock))
}

func TestNextBackupPathSkipsExisting(t *testing.T) {
	fs := afero.NewMemMapFs()
	target := filepath.FromSlash("/mods/mod.jar")
	base := target + ".mmm.bak"

	assert.NoError(t, afero.WriteFile(fs, base, []byte("backup"), 0644))
	next, err := nextBackupPath(fs, target)
	assert.NoError(t, err)
	assert.Equal(t, base+".1", next)
}

func TestNextBackupPathReturnsErrorOnStatFailure(t *testing.T) {
	fs := statErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.FromSlash("/mods/mod.jar.mmm.bak"),
		err:      errors.New("stat failed"),
	}

	_, err := nextBackupPath(fs, filepath.FromSlash("/mods/mod.jar"))
	assert.Error(t, err)
}

func TestNextBackupPathReturnsErrorAfterExhaustion(t *testing.T) {
	fs := afero.NewMemMapFs()
	target := filepath.FromSlash("/mods/mod.jar")
	base := target + ".mmm.bak"

	assert.NoError(t, afero.WriteFile(fs, base, []byte("backup"), 0644))
	for i := 1; i < 100; i++ {
		assert.NoError(t, afero.WriteFile(fs, base+"."+strconv.Itoa(i), []byte("backup"), 0644))
	}

	_, err := nextBackupPath(fs, target)
	assert.Error(t, err)
}

func TestReplaceExistingFileRenamesWhenDestinationMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	source := filepath.FromSlash("/mods/source.jar")
	destination := filepath.FromSlash("/mods/dest.jar")

	assert.NoError(t, afero.WriteFile(fs, source, []byte("source"), 0644))
	assert.NoError(t, replaceExistingFile(fs, logger.New(io.Discard, io.Discard, false, false), source, destination))

	exists, err := afero.Exists(fs, destination)
	assert.NoError(t, err)
	assert.True(t, exists)
	exists, err = afero.Exists(fs, source)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestReplaceExistingFileReturnsErrorOnStatFailure(t *testing.T) {
	fs := statErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.FromSlash("/mods/dest.jar"),
		err:      errors.New("stat failed"),
	}

	err := replaceExistingFile(fs, logger.New(io.Discard, io.Discard, false, false), filepath.FromSlash("/mods/source.jar"), filepath.FromSlash("/mods/dest.jar"))
	assert.Error(t, err)
}

func TestReplaceExistingFileReturnsErrorOnBackupRenameFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	destination := filepath.FromSlash("/mods/dest.jar")
	source := filepath.FromSlash("/mods/source.jar")
	backup := destination + ".mmm.bak"

	assert.NoError(t, afero.WriteFile(baseFs, destination, []byte("old"), 0644))
	assert.NoError(t, afero.WriteFile(baseFs, source, []byte("new"), 0644))

	fs := renameErrorFs{Fs: baseFs, failOld: destination, failNew: backup, err: errors.New("rename failed")}
	err := replaceExistingFile(fs, logger.New(io.Discard, io.Discard, false, false), source, destination)
	assert.Error(t, err)
}

func TestReplaceExistingFileReturnsErrorOnNextBackupPathFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	destination := filepath.FromSlash("/mods/dest.jar")
	source := filepath.FromSlash("/mods/source.jar")
	backup := destination + ".mmm.bak"

	assert.NoError(t, afero.WriteFile(baseFs, destination, []byte("old"), 0644))
	assert.NoError(t, afero.WriteFile(baseFs, source, []byte("new"), 0644))

	fs := statErrorFs{Fs: baseFs, failPath: backup, err: errors.New("stat failed")}
	err := replaceExistingFile(fs, logger.New(io.Discard, io.Discard, false, false), source, destination)
	assert.Error(t, err)
}

func TestReplaceExistingFileRestoresBackupOnRenameFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	destination := filepath.FromSlash("/mods/dest.jar")
	source := filepath.FromSlash("/mods/source.jar")

	assert.NoError(t, afero.WriteFile(baseFs, destination, []byte("old"), 0644))
	assert.NoError(t, afero.WriteFile(baseFs, source, []byte("new"), 0644))

	fs := renameErrorFs{Fs: baseFs, failOld: source, failNew: destination, err: errors.New("rename failed")}
	err := replaceExistingFile(fs, logger.New(io.Discard, io.Discard, false, false), source, destination)
	assert.Error(t, err)

	content, readErr := afero.ReadFile(fs, destination)
	assert.NoError(t, readErr)
	assert.Equal(t, []byte("old"), content)
}

func TestReplaceExistingFileLogsWhenBackupCleanupFails(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	destination := filepath.FromSlash("/mods/dest.jar")
	source := filepath.FromSlash("/mods/source.jar")
	backup := destination + ".mmm.bak"

	assert.NoError(t, afero.WriteFile(baseFs, destination, []byte("old"), 0644))
	assert.NoError(t, afero.WriteFile(baseFs, source, []byte("new"), 0644))

	fs := removeErrorFs{Fs: baseFs, failPath: backup, err: errors.New("remove failed")}
	log := logger.New(io.Discard, io.Discard, false, true)

	assert.NoError(t, replaceExistingFile(fs, log, source, destination))
}

func TestDownloadAndSwapRemovesNewFileWhenOldRemovalFails(t *testing.T) {
	fs := removeErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.FromSlash("/mods/old.jar"),
		err:      errors.New("remove failed"),
	}

	oldPath := filepath.FromSlash("/mods/old.jar")
	newPath := filepath.FromSlash("/mods/new.jar")
	assert.NoError(t, afero.WriteFile(fs, oldPath, []byte("old"), 0644))

	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("new"), 0644)
		},
	}

	assert.Error(t, downloadAndSwap(context.Background(), deps, oldPath, newPath, filepath.FromSlash("/mods"), "https://example.invalid/new.jar", sha1Hex("new"), nil))

	exists, err := afero.Exists(fs, newPath)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestDownloadAndSwapCleansTempOnDownloaderFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/same.jar")
	assert.NoError(t, afero.WriteFile(fs, path, []byte("old"), 0644))

	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			return errors.New("download failed")
		},
		logger: logger.New(io.Discard, io.Discard, false, false),
		output: output.New(io.Discard, io.Discard, false),
	}

	assert.Error(t, downloadAndSwap(context.Background(), deps, path, path, filepath.FromSlash("/mods"), "https://example.invalid/same.jar", sha1Hex("new"), nil))

	entries, entriesErr := afero.ReadDir(fs, filepath.FromSlash("/mods"))
	assert.NoError(t, entriesErr)
	for _, entry := range entries {
		assert.False(t, strings.HasSuffix(entry.Name(), ".tmp"))
	}
}

func TestDownloadAndSwapCleansTempOnReplaceFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/same.jar")
	assert.NoError(t, afero.WriteFile(baseFs, path, []byte("old"), 0644))

	fs := renameErrorFs{Fs: baseFs, failOldContains: ".mmm.", failNew: path, err: errors.New("rename failed")}

	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("new"), 0644)
		},
		logger: logger.New(io.Discard, io.Discard, false, false),
		output: output.New(io.Discard, io.Discard, false),
	}

	assert.Error(t, downloadAndSwap(context.Background(), deps, path, path, filepath.FromSlash("/mods"), "https://example.invalid/same.jar", sha1Hex("new"), nil))

	entries, entriesErr := afero.ReadDir(fs, filepath.FromSlash("/mods"))
	assert.NoError(t, entriesErr)
	for _, entry := range entries {
		assert.False(t, strings.HasSuffix(entry.Name(), ".tmp"))
	}
}

func TestDownloadAndSwapReturnsMissingHashError(t *testing.T) {
	fs := afero.NewMemMapFs()
	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
	}

	err := downloadAndSwap(context.Background(), deps, filepath.FromSlash("/mods/old.jar"), filepath.FromSlash("/mods/new.jar"), filepath.FromSlash("/mods"), "https://example.invalid/new.jar", "", nil)
	var missingHash modinstall.MissingHashError
	assert.ErrorAs(t, err, &missingHash)
}

func TestDownloadAndSwapReturnsErrorOnHashReadFailure(t *testing.T) {
	base := afero.NewMemMapFs()
	newPath := filepath.FromSlash("/mods/new.jar")
	fs := openErrorFs{Fs: base, failContains: ".mmm.", err: errors.New("open failed")}

	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("data"), 0644)
		},
		logger: logger.New(io.Discard, io.Discard, false, false),
		output: output.New(io.Discard, io.Discard, false),
	}

	err := downloadAndSwap(context.Background(), deps, filepath.FromSlash("/mods/old.jar"), newPath, filepath.FromSlash("/mods"), "https://example.invalid/new.jar", sha1Hex("data"), nil)
	assert.Error(t, err)
}

func TestDownloadAndSwapReturnsErrorWhenResolveWritablePathFails(t *testing.T) {
	root := t.TempDir()
	modsRoot := filepath.Join(root, "mods")
	assert.NoError(t, os.MkdirAll(modsRoot, 0755))

	outside := t.TempDir()
	target := filepath.Join(outside, "target.jar")
	assert.NoError(t, os.WriteFile(target, []byte("data"), 0644))

	newPath := filepath.Join(modsRoot, "link.jar")
	fs := symlinkStubFs{
		Fs:       afero.NewOsFs(),
		symlinks: map[string]string{newPath: target},
	}

	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
	}

	err := downloadAndSwap(context.Background(), deps, filepath.Join(modsRoot, "old.jar"), newPath, modsRoot, "https://example.invalid/new.jar", sha1Hex("data"), nil)
	assert.Error(t, err)
}

func TestDownloadAndSwapReturnsErrorOnTempFileFailure(t *testing.T) {
	base := afero.NewMemMapFs()
	assert.NoError(t, base.MkdirAll(filepath.FromSlash("/mods"), 0755))
	fs := openFileErrorFs{Fs: base, err: errors.New("open failed")}

	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
	}

	err := downloadAndSwap(context.Background(), deps, filepath.FromSlash("/mods/old.jar"), filepath.FromSlash("/mods/new.jar"), filepath.FromSlash("/mods"), "https://example.invalid/new.jar", sha1Hex("data"), nil)
	assert.Error(t, err)
}

func TestSha1ForFileReturnsErrorOnOpenFailure(t *testing.T) {
	fs := openErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.FromSlash("/mods/missing.jar"),
		err:      errors.New("open failed"),
	}

	_, err := sha1ForFile(fs, filepath.FromSlash("/mods/missing.jar"))
	assert.Error(t, err)
}

func TestSha1ForFileReturnsErrorOnReadFailure(t *testing.T) {
	base := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/data.jar")
	assert.NoError(t, afero.WriteFile(base, path, []byte("data"), 0644))

	fs := readErrorFs{Fs: base, failPath: path, err: errors.New("read failed")}
	_, err := sha1ForFile(fs, path)
	assert.Error(t, err)
}

func TestIntegrityErrorMessageReturnsFalseForUnknownError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	message, ok := integrityErrorMessage(errors.New("unknown"))
	assert.False(t, ok)
	assert.Empty(t, message)
}

func TestIntegrityErrorMessageHandlesMissingHash(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	message, ok := integrityErrorMessage(modinstall.MissingHashError{FileName: "x.jar"})
	assert.True(t, ok)
	assert.Contains(t, message, "cmd.update.error.missing_hash")
}

func TestIntegrityErrorMessageHandlesHashMismatch(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	message, ok := integrityErrorMessage(modinstall.HashMismatchError{FileName: "x.jar", Expected: "a", Actual: "b"})
	assert.True(t, ok)
	assert.Contains(t, message, "cmd.update.error.hash_mismatch")
}

type statErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (filesystem statErrorFs) Stat(name string) (os.FileInfo, error) {
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return nil, filesystem.err
	}
	return filesystem.Fs.Stat(name)
}

type renameErrorFs struct {
	afero.Fs
	failOld         string
	failNew         string
	failOldContains string
	err             error
}

func (filesystem renameErrorFs) Rename(oldname, newname string) error {
	if filesystem.failOldContains != "" && strings.Contains(filepath.Clean(oldname), filesystem.failOldContains) && filepath.Clean(newname) == filepath.Clean(filesystem.failNew) {
		return filesystem.err
	}
	if filesystem.failOld != "" && filepath.Clean(oldname) == filepath.Clean(filesystem.failOld) && filepath.Clean(newname) == filepath.Clean(filesystem.failNew) {
		return filesystem.err
	}
	return filesystem.Fs.Rename(oldname, newname)
}

type removeErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (filesystem removeErrorFs) Remove(name string) error {
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return filesystem.err
	}
	return filesystem.Fs.Remove(name)
}

func sha1Hex(data string) string {
	sum := sha1.Sum([]byte(data))
	return hex.EncodeToString(sum[:])
}

type readErrorFs struct {
	afero.Fs
	failPath string
	err      error
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

type readErrorFile struct {
	afero.File
	err error
}

func (file readErrorFile) Read([]byte) (int, error) {
	return 0, file.err
}

type openErrorFs struct {
	afero.Fs
	failPath     string
	failContains string
	err          error
}

func (filesystem openErrorFs) Open(name string) (afero.File, error) {
	if filesystem.failPath != "" && filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return nil, filesystem.err
	}
	if filesystem.failContains != "" && strings.Contains(filepath.Clean(name), filesystem.failContains) {
		return nil, filesystem.err
	}
	return filesystem.Fs.Open(name)
}

type openFileErrorFs struct {
	afero.Fs
	err error
}

func (filesystem openFileErrorFs) OpenFile(string, int, os.FileMode) (afero.File, error) {
	return nil, filesystem.err
}

type symlinkStubFs struct {
	afero.Fs
	symlinks map[string]string
}

func (filesystem symlinkStubFs) LstatIfPossible(path string) (os.FileInfo, bool, error) {
	cleanPath := filepath.Clean(path)
	if _, ok := filesystem.symlinks[cleanPath]; ok {
		return fakeFileInfo{name: filepath.Base(cleanPath), mode: os.ModeSymlink}, true, nil
	}
	return nil, true, os.ErrNotExist
}

func (filesystem symlinkStubFs) ReadlinkIfPossible(path string) (string, error) {
	cleanPath := filepath.Clean(path)
	target, ok := filesystem.symlinks[cleanPath]
	if !ok {
		return "", os.ErrNotExist
	}
	return target, nil
}

type fakeFileInfo struct {
	name string
	mode os.FileMode
}

func (info fakeFileInfo) Name() string       { return info.name }
func (info fakeFileInfo) Size() int64        { return 0 }
func (info fakeFileInfo) Mode() os.FileMode  { return info.mode }
func (info fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (info fakeFileInfo) IsDir() bool        { return false }
func (info fakeFileInfo) Sys() interface{}   { return nil }

func TestFetchErrorReasonForPlatformError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	mod := models.Mod{Name: "Example", ID: "proj-1", Type: models.MODRINTH}
	reason := fetchErrorReason(&httpclient.ResponseError{StatusCode: http.StatusForbidden}, mod, logger.New(io.Discard, io.Discard, false, true))
	assert.Contains(t, reason, "cmd.update.error.platform")
	assert.Contains(t, reason, "cmd.platform.error.reason.auth")
}

func TestFetchErrorReasonForExpectedFetchError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	mod := models.Mod{Name: "Example", ID: "proj-1", Type: models.MODRINTH}
	reason := fetchErrorReason(&platform.ModNotFoundError{Platform: models.MODRINTH, ProjectID: "proj-1"}, mod, nil)
	assert.Contains(t, reason, "cmd.update.error.mod_not_found")
}

func TestFetchErrorReasonReturnsEmptyForNilError(t *testing.T) {
	reason := fetchErrorReason(nil, models.Mod{Type: models.MODRINTH}, nil)
	assert.Equal(t, "", reason)
}

func TestFetchErrorReasonSkipsDebugWhenDetailsEmpty(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	mod := models.Mod{Name: "Example", ID: "proj-1", Type: models.MODRINTH}
	reason := fetchErrorReason(emptyError{}, mod, logger.New(io.Discard, io.Discard, false, true))
	assert.Contains(t, reason, "cmd.update.error.platform")
}

func TestFetchErrorReasonIgnoresDebugLogErrors(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	mod := models.Mod{Name: "Example", ID: "proj-1", Type: models.MODRINTH}
	writeErr := errors.New("write failed")
	log := logger.New(errorWriter{err: writeErr}, io.Discard, false, true)
	reason := fetchErrorReason(&httpclient.ResponseError{StatusCode: http.StatusForbidden}, mod, log)
	assert.Contains(t, reason, "cmd.update.error.platform")
}

type emptyError struct{}

func (emptyError) Error() string {
	return ""
}

func TestReplaceExistingFileReturnsLoggerErrorOnBackupCleanupFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	baseFs := afero.NewMemMapFs()
	source := filepath.FromSlash("/mods/source.jar")
	destination := filepath.FromSlash("/mods/dest.jar")
	assert.NoError(t, afero.WriteFile(baseFs, destination, []byte("old"), 0644))
	assert.NoError(t, afero.WriteFile(baseFs, source, []byte("new"), 0644))

	fs := removeErrorFs{
		Fs:       baseFs,
		failPath: destination + ".mmm.bak",
		err:      errors.New("remove failed"),
	}

	err := replaceExistingFile(fs, logger.New(errorWriter{err: writeErr}, io.Discard, false, true), source, destination)
	assert.ErrorIs(t, err, writeErr)
}

func TestEnsureInstallForUpdateReturnsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	cmd := &cobra.Command{}
	cmd.SetOut(errorWriter{err: writeErr})

	err := ensureInstallForUpdate(context.Background(), cmd, updateOptions{}, updateDeps{
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			_, outputErr := cmd.OutOrStdout().Write([]byte("install output\n"))
			assert.ErrorIs(t, outputErr, writeErr)
			return install.Result{}, clierrors.MarkHandled(interaction.ErrUnmanagedFiles)
		},
	}, interaction.ExecutionModeNonTTY)
	assert.ErrorIs(t, err, writeErr)
}

func TestEnsureInstallForUpdateReturnsHeaderWriteError(t *testing.T) {
	writeErr := errors.New("write failed")
	cmd := &cobra.Command{}
	cmd.SetOut(errorWriter{err: writeErr})

	err := ensureInstallForUpdate(context.Background(), cmd, updateOptions{}, updateDeps{
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			_, outputErr := cmd.OutOrStdout().Write([]byte("install output\n"))
			assert.ErrorIs(t, outputErr, writeErr)
			return install.Result{}, nil
		},
	}, interaction.ExecutionModeNonTTY)
	assert.ErrorIs(t, err, writeErr)
}

func TestEnsureInstallForUpdatePassesThroughInstallOutputOnFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	out := &terminalWriter{}
	cmd := &cobra.Command{}
	cmd.SetIn(terminalReader{Reader: bytes.NewBuffer(nil)})
	cmd.SetOut(out)

	installErr := errors.New("install failed")
	err := ensureInstallForUpdate(context.Background(), cmd, updateOptions{}, updateDeps{
		install: func(ctx context.Context, _ *cobra.Command, _ install.RunOptions) (install.Result, error) {
			install.NotifyInstallViewObserver(ctx, "install output")
			return install.Result{}, installErr
		},
	}, interaction.ExecutionModeInteractive)

	assert.Error(t, err)
	output := out.String()
	assert.Contains(t, output, "install output")
	assert.Contains(t, output, "cmd.update.error.install_failed")
	installIndex := strings.Index(output, "install output")
	updateIndex := strings.Index(output, "cmd.update.error.install_failed")
	assert.Greater(t, updateIndex, installIndex)
}

func TestUpdateInstallHeaderWriterEmitsHeaderOnce(t *testing.T) {
	output := &bytes.Buffer{}
	writer := &updateInstallHeaderWriter{
		out:    output,
		header: "Installing potentially missing mods:",
	}

	_, err := writer.Write([]byte("install output\n"))
	require.NoError(t, err)
	_, err = writer.Write([]byte("more output\n"))
	require.NoError(t, err)

	assert.Equal(t, "Installing potentially missing mods:\n\ninstall output\nmore output\n", output.String())
}

func TestUpdateInstallHeaderWriterReturnsHeaderError(t *testing.T) {
	writeErr := errors.New("write failed")
	writer := &updateInstallHeaderWriter{
		out:    errorWriter{err: writeErr},
		header: "Installing potentially missing mods:",
	}

	_, err := writer.Write([]byte("install output\n"))
	assert.ErrorIs(t, err, writeErr)
	_, err = writer.Write([]byte("more output\n"))
	assert.ErrorIs(t, err, writeErr)
}

func TestUpdateInstallHeaderWriterReturnsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	output := &failAfterWriter{err: writeErr}
	writer := &updateInstallHeaderWriter{
		out:    output,
		header: "Installing potentially missing mods:",
	}

	_, err := writer.Write([]byte("install output\n"))
	assert.ErrorIs(t, err, writeErr)
	assert.ErrorIs(t, writer.writeErr, writeErr)
	assert.Equal(t, "Installing potentially missing mods:\n\n", output.output.String())
}

func TestRunUpdateSkipsInstallFailureViewOnCanceledInstall(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Mods: []models.Mod{{ID: "mod-1", Name: "Mod One", Type: models.MODRINTH}},
	}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	_, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			return install.Result{}, context.Canceled
		},
	})

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, "", out.String())
}

func TestRunUpdateQuietSkipsInstallFailureViewOnCanceledInstall(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Mods: []models.Mod{{ID: "mod-1", Name: "Mod One", Type: models.MODRINTH}},
	}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	_, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath, Quiet: true}, updateDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			return install.Result{}, context.Canceled
		},
	})

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, "", out.String())
}

func TestRunUpdateReturnsOutputErrorWhenNoModsConfigured(t *testing.T) {
	writeErr := errors.New("write failed")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{Mods: []models.Mod{}}
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	cmd := &cobra.Command{}
	cmd.SetOut(errorWriter{err: writeErr})

	_, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			return install.Result{}, nil
		},
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestRunUpdateReturnsCanceledErrorOnCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Mods: []models.Mod{{ID: "abc", Name: "Example", Type: models.MODRINTH}},
	}
	lock := []models.ModInstall{{ID: "abc", Name: "Example", Type: models.MODRINTH, FileName: "mod.jar", Hash: "hash", DownloadURL: "https://example.invalid"}}
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	_, err := runUpdate(ctx, cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			return install.Result{}, nil
		},
	})
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRunUpdateReturnsErrorOnPersistFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Loader:      models.FABRIC,
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "Configured", Type: models.MODRINTH},
		},
	}
	lock := []models.ModInstall{
		{
			Type:        models.MODRINTH,
			ID:          "proj-1",
			Name:        "Configured",
			FileName:    "mod.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "hash",
			DownloadURL: "https://example.invalid/mod.jar",
		},
	}
	assert.NoError(t, baseFs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), baseFs, meta, lock))
	assert.NoError(t, afero.WriteFile(baseFs, filepath.Join(meta.ModsFolderPath(cfg), "mod.jar"), []byte("x"), 0644))

	fs := openFileErrorFs{Fs: baseFs, err: errors.New("open failed")}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	_, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
		output: output.New(io.Discard, io.Discard, false),
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			return install.Result{}, nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Configured",
				FileName:    "mod.jar",
				ReleaseDate: "2024-01-01T00:00:00Z",
				Hash:        "hash",
				DownloadURL: "https://example.invalid/mod.jar",
			}, nil
		},
	})
	assert.ErrorContains(t, err, "open failed")
}

type errorWriter struct {
	err error
}

func (writer errorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

type failAfterWriter struct {
	err    error
	writes int
	output bytes.Buffer
}

func (writer *failAfterWriter) Write(value []byte) (int, error) {
	writer.writes++
	if writer.writes == 1 {
		return writer.output.Write(value)
	}
	return 0, writer.err
}
