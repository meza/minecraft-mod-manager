package init

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/privacy"
)

func TestBuildTelemetryPayloadExitCode(t *testing.T) {
	payload := buildTelemetryPayload(initOptions{
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
	}, true, nil)
	assert.Equal(t, 0, payload.ExitCode)
	assert.True(t, payload.Success)

	payload = buildTelemetryPayload(initOptions{}, false, errors.New("boom"))
	assert.Equal(t, 1, payload.ExitCode)
	assert.False(t, payload.Success)
}

func TestNormalizeGameVersionEmptyNoop(t *testing.T) {
	opts, err := normalizeGameVersion(context.Background(), initOptions{}, initDeps{output: output.New(io.Discard, io.Discard, true)}, gameVersionUnattended)
	assert.NoError(t, err)
	assert.Equal(t, "", opts.GameVersion)
}

func TestNormalizeGameVersionInteractiveLatestClearsValue(t *testing.T) {
	opts, err := normalizeGameVersion(context.Background(), initOptions{
		GameVersion: "latest",
		Provided: providedFlags{
			GameVersion: true,
		},
	}, initDeps{output: output.New(io.Discard, io.Discard, true)}, gameVersionInteractive)
	assert.NoError(t, err)
	assert.Equal(t, "", opts.GameVersion)
	assert.False(t, opts.Provided.GameVersion)
}

func TestNormalizeGameVersionUnattendedResolvesLatest(t *testing.T) {
	opts, err := normalizeGameVersion(context.Background(), initOptions{
		GameVersion: "latest",
	}, initDeps{
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, gameVersionUnattended)
	assert.NoError(t, err)
	assert.Equal(t, "1.21.1", opts.GameVersion)
}

func TestNormalizeGameVersionInteractive(t *testing.T) {
	options := normalizeGameVersionInteractive(initOptions{
		GameVersion: "latest",
		Provided: providedFlags{
			GameVersion: true,
		},
	})
	assert.Empty(t, options.GameVersion)
	assert.False(t, options.Provided.GameVersion)

	options = normalizeGameVersionInteractive(initOptions{
		GameVersion: "1.21.1",
		Provided: providedFlags{
			GameVersion: true,
		},
	})
	assert.Equal(t, "1.21.1", options.GameVersion)
	assert.True(t, options.Provided.GameVersion)

	options = normalizeGameVersionInteractive(initOptions{})
	assert.Empty(t, options.GameVersion)
}

func TestValidateModsFolderErrors(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	err := validateModsFolder(fs, meta, "")
	assert.ErrorContains(t, err, "cannot be empty")

	badFs := statErrorFs{Fs: fs, err: errors.New("stat failed")}
	err = validateModsFolder(badFs, meta, "mods")
	assert.ErrorContains(t, err, "stat failed")

	mkdirErr := fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755)
	if mkdirErr != nil {
		t.Fatalf("mkdir failed: %v", mkdirErr)
	}
	info, err := fs.Stat(filepath.FromSlash("/cfg/mods"))
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	sequenceFs := &statSequenceFs{Fs: fs, err: errors.New("stat second call"), info: info}
	err = validateModsFolder(sequenceFs, meta, "mods")
	assert.ErrorContains(t, err, "stat second call")
}

func TestValidateModsFolderMissingAndNotDirectory(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	err := validateModsFolder(fs, meta, "mods")
	assert.ErrorContains(t, err, "cmd.init.error.mods-folder.missing")

	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg"), 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.FromSlash("/cfg/mods"), []byte("file"), 0644))
	err = validateModsFolder(fs, meta, "mods")
	assert.ErrorContains(t, err, "cmd.init.error.mods-folder.not-directory")

	assert.NoError(t, fs.Remove(filepath.FromSlash("/cfg/mods")))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))
	err = validateModsFolder(fs, meta, "mods")
	assert.NoError(t, err)
}

func TestValidateModsFolderInteractiveMissingAndNotDirectory(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	err := validateModsFolderInteractive(fs, meta, "")
	assert.ErrorContains(t, err, "cmd.init.error.mods-folder.empty")

	err = validateModsFolderInteractive(fs, meta, "mods")
	assert.ErrorContains(t, err, "cmd.init.prompt.mods-folder.missing")

	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg"), 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.FromSlash("/cfg/mods"), []byte("file"), 0644))
	err = validateModsFolderInteractive(fs, meta, "mods")
	assert.ErrorContains(t, err, "cmd.init.prompt.mods-folder.not-directory")

	assert.NoError(t, fs.Remove(filepath.FromSlash("/cfg/mods")))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))
	err = validateModsFolderInteractive(fs, meta, "mods")
	assert.NoError(t, err)
}

func TestValidateModsFolderInteractiveStatErrors(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	statErrFs := statErrorFs{Fs: afero.NewMemMapFs(), err: errors.New("stat failed")}
	err := validateModsFolderInteractive(statErrFs, meta, "mods")
	assert.ErrorContains(t, err, "stat failed")

	baseFs := afero.NewMemMapFs()
	assert.NoError(t, baseFs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))
	info, err := baseFs.Stat(filepath.FromSlash("/cfg/mods"))
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	sequenceFs := &statSequenceFs{Fs: baseFs, err: errors.New("stat second call"), info: info}
	err = validateModsFolderInteractive(sequenceFs, meta, "mods")
	assert.ErrorContains(t, err, "stat second call")
}

func TestBuildTelemetryPayloadRedactsModsFolderError(t *testing.T) {
	t.Setenv("USER", "alice")
	t.Setenv("USERNAME", "")
	t.Setenv("LOGNAME", "")
	privacy.ResetForTesting()

	var pathWithUsername string
	var expectedRedacted string
	switch runtime.GOOS {
	case "windows":
		pathWithUsername = `C:\Users\alice\mods`
		expectedRedacted = `C:\Users\<user>\mods`
	case "darwin":
		pathWithUsername = "/Users/alice/mods"
		expectedRedacted = "/Users/<user>/mods"
	default:
		pathWithUsername = "/home/alice/mods"
		expectedRedacted = "/home/<user>/mods"
	}

	inputError := fmt.Errorf("mods folder does not exist: %s", pathWithUsername)
	payload := buildTelemetryPayload(initOptions{}, false, inputError)

	if assert.Error(t, payload.Error) {
		assert.Equal(t, fmt.Sprintf("mods folder does not exist: %s", expectedRedacted), payload.Error.Error())
		assert.ErrorIs(t, payload.Error, inputError)
	}
}

func TestInitWithDepsDoesNotWriteOutput(t *testing.T) {
	minecraft.ClearManifestCache()
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	logBuffer := &bytes.Buffer{}
	err := initWithDeps(context.Background(), initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
	}, initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		logger:          logger.New(logBuffer, io.Discard, false, false),
		output:          output.New(logBuffer, io.Discard, false),
	})
	assert.NoError(t, err)
	assert.Empty(t, logBuffer.String())
}

func TestInitWithDepsMkdirAllError(t *testing.T) {
	minecraft.ClearManifestCache()
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, baseFs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	fs := mkdirErrorFs{Fs: baseFs, failPath: meta.Dir(), err: errors.New("mkdir failed")}
	err := initWithDeps(context.Background(), initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	})
	assert.ErrorContains(t, err, "mkdir failed")
}

func TestInitWithDepsWriteConfigError(t *testing.T) {
	minecraft.ClearManifestCache()
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, baseFs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	fs := renameErrorFs{Fs: baseFs, failPath: meta.ConfigPath, err: errors.New("rename failed")}

	err := initWithDeps(context.Background(), initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	})
	assert.ErrorContains(t, err, "rename failed")
}

func TestInitWithDepsWriteLockError(t *testing.T) {
	minecraft.ClearManifestCache()
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, baseFs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	fs := renameErrorFs{Fs: baseFs, failPath: meta.LockPath(), err: errors.New("rename failed")}

	err := initWithDeps(context.Background(), initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	})
	assert.Error(t, err)
}

type statErrorFs struct {
	afero.Fs
	err error
}

func (fs statErrorFs) Stat(string) (os.FileInfo, error) {
	return nil, fs.err
}

type statSequenceFs struct {
	afero.Fs
	err   error
	info  os.FileInfo
	calls int
}

func (fs *statSequenceFs) Stat(name string) (os.FileInfo, error) {
	fs.calls++
	if fs.calls == 1 {
		return fs.info, nil
	}
	return nil, fs.err
}

type mkdirErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (fs mkdirErrorFs) MkdirAll(path string, perm os.FileMode) error {
	if path == fs.failPath {
		return fs.err
	}
	return fs.Fs.MkdirAll(path, perm)
}

type renameErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (fs renameErrorFs) Rename(oldname, newname string) error {
	if newname == fs.failPath {
		return fs.err
	}
	return fs.Fs.Rename(oldname, newname)
}
