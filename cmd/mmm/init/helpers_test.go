package init

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}

func TestTerminalPrompterConfirmOverwrite(t *testing.T) {
	prompter := terminalPrompter{
		in:  bytes.NewBufferString("y\n"),
		out: io.Discard,
	}
	confirmed, err := prompter.ConfirmOverwrite("config.json")
	assert.NoError(t, err)
	assert.True(t, confirmed)

	prompter = terminalPrompter{
		in:  bytes.NewBufferString("no\n"),
		out: io.Discard,
	}
	confirmed, err = prompter.ConfirmOverwrite("config.json")
	assert.NoError(t, err)
	assert.False(t, confirmed)
}

func TestTerminalPrompterRequestNewConfigPath(t *testing.T) {
	prompter := terminalPrompter{
		in:  bytes.NewBufferString("\n"),
		out: io.Discard,
	}
	_, err := prompter.RequestNewConfigPath("config.json")
	assert.ErrorContains(t, err, "cannot be empty")

	prompter = terminalPrompter{
		in:  bytes.NewBufferString("/tmp/modlist.json\n"),
		out: io.Discard,
	}
	path, err := prompter.RequestNewConfigPath("config.json")
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/modlist.json", path)
}

func TestTerminalPrompterRequestNewConfigPathReadError(t *testing.T) {
	prompter := terminalPrompter{
		in:  errorReader{err: errors.New("read failed")},
		out: io.Discard,
	}
	_, err := prompter.RequestNewConfigPath("config.json")
	assert.ErrorContains(t, err, "read failed")
}

func TestReadLineEOFAndError(t *testing.T) {
	_, err := readLine(bytes.NewBuffer(nil))
	assert.ErrorIs(t, err, io.EOF)

	_, err = readLine(errorReader{err: errors.New("boom")})
	assert.ErrorContains(t, err, "boom")
}

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
	opts, err := normalizeGameVersion(context.Background(), initOptions{}, initDeps{}, false)
	assert.NoError(t, err)
	assert.Equal(t, "", opts.GameVersion)
}

func TestValidateModsFolderErrors(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	_, err := validateModsFolder(fs, meta, "")
	assert.ErrorContains(t, err, "cannot be empty")

	badFs := statErrorFs{Fs: fs, err: errors.New("stat failed")}
	_, err = validateModsFolder(badFs, meta, "mods")
	assert.ErrorContains(t, err, "stat failed")

	if err := fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	info, err := fs.Stat(filepath.FromSlash("/cfg/mods"))
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	sequenceFs := &statSequenceFs{Fs: fs, err: errors.New("stat second call"), info: info}
	_, err = validateModsFolder(sequenceFs, meta, "mods")
	assert.ErrorContains(t, err, "stat second call")
}

func TestGetCurrentWorkingDirectoryError(t *testing.T) {
	assert.Equal(t, "", getCurrentWorkingDirectoryWith(func() (string, error) {
		return "", errors.New("boom")
	}))
}

func TestInitWithDepsPrompterErrors(t *testing.T) {
	minecraft.ClearManifestCache()
	fs := afero.NewMemMapFs()
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.FromSlash("/cfg/modlist.json"), []byte(`{"existing":true}`), 0644))

	_, err := initWithDeps(context.Background(), initOptions{
		ConfigPath:   filepath.FromSlash("/cfg/modlist.json"),
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
	}, initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		prompter:        fakePrompter{confirmErr: errors.New("confirm failed")},
	})
	assert.ErrorContains(t, err, "confirm failed")

	_, err = initWithDeps(context.Background(), initOptions{
		ConfigPath:   filepath.FromSlash("/cfg/modlist.json"),
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
	}, initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		prompter:        fakePrompter{overwrite: false, newPathErr: errors.New("new path failed")},
	})
	assert.ErrorContains(t, err, "new path failed")
}

func TestInitWithDepsLogsWhenLoggerProvided(t *testing.T) {
	minecraft.ClearManifestCache()
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	logBuffer := &bytes.Buffer{}
	_, err := initWithDeps(context.Background(), initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
	}, initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		logger:          logger.New(logBuffer, io.Discard, false, false),
	})
	assert.NoError(t, err)
	assert.Contains(t, logBuffer.String(), "Initialized configuration")
}

func TestInitWithDepsLatestVersionError(t *testing.T) {
	minecraft.ClearManifestCache()
	fs := afero.NewMemMapFs()
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	_, err := initWithDeps(context.Background(), initOptions{
		ConfigPath:   filepath.FromSlash("/cfg/modlist.json"),
		Loader:       models.FABRIC,
		GameVersion:  "",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
	}, initDeps{
		fs: fs,
		minecraftClient: doerFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("offline")
		}),
	})
	assert.ErrorContains(t, err, "could not determine latest minecraft version")
}

func TestInitWithDepsMkdirAllError(t *testing.T) {
	minecraft.ClearManifestCache()
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, baseFs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	fs := mkdirErrorFs{Fs: baseFs, failPath: meta.Dir(), err: errors.New("mkdir failed")}
	_, err := initWithDeps(context.Background(), initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
	}, initDeps{
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

	_, err := initWithDeps(context.Background(), initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
	}, initDeps{
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

	_, err := initWithDeps(context.Background(), initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
	}, initDeps{
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
