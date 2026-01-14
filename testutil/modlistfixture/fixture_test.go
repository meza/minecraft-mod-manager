package modlistfixture

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/lifecycle"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

func TestNewWithNilTestReturnsError(t *testing.T) {
	fixture, err := New(nil)
	assert.Nil(t, fixture)
	assert.Error(t, err)
}

func TestNewWithNilFilesystemReturnsError(t *testing.T) {
	fixture, err := New(t, WithFS(nil))
	assert.Nil(t, fixture)
	assert.Error(t, err)
}

func TestNewReturnsErrorWhenTempDirFails(t *testing.T) {
	readOnlyFilesystem := afero.NewReadOnlyFs(afero.NewMemMapFs())

	fixture, err := New(t, WithFS(readOnlyFilesystem))
	assert.Nil(t, fixture)
	assert.Error(t, err)
}

func TestNewReturnsErrorWhenFilesystemIsNilAfterOptions(t *testing.T) {
	fixture, err := New(t, withNilFilesystem())
	assert.Nil(t, fixture)
	assert.Error(t, err)
}

func TestNewCreatesTempDirAndPaths(t *testing.T) {
	fixture, err := New(t)
	require.NoError(t, err)

	exists, err := afero.Exists(fixture.Filesystem, fixture.RootDir)
	require.NoError(t, err)
	assert.True(t, exists)

	assert.Equal(t, filepath.Join(fixture.RootDir, "modlist.json"), fixture.ConfigPath)
	assert.Equal(t, filepath.Join(fixture.RootDir, "modlist-lock.json"), fixture.LockPath)
	assert.Equal(t, fixture.ConfigPath, fixture.Meta.ConfigPath)

	require.NoError(t, fixture.Cleanup())

	exists, err = afero.Exists(fixture.Filesystem, fixture.RootDir)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestNewRegistersLifecycleCleanupForOSFilesystem(t *testing.T) {
	originalRegister := registerLifecycle
	originalUnregister := unregisterLifecycle
	t.Cleanup(func() {
		registerLifecycle = originalRegister
		unregisterLifecycle = originalUnregister
	})

	var registered bool
	var unregistered lifecycle.HandlerID
	var storedHandler lifecycle.Handler
	registerLifecycle = func(handler lifecycle.Handler) lifecycle.HandlerID {
		registered = true
		require.NotNil(t, handler)
		storedHandler = handler
		return 42
	}
	unregisterLifecycle = func(id lifecycle.HandlerID) {
		unregistered = id
	}

	fixture, err := New(t, WithOSFilesystem())
	require.NoError(t, err)
	assert.True(t, registered)

	require.NotNil(t, storedHandler)
	storedHandler(os.Interrupt)

	require.NoError(t, fixture.Cleanup())
	assert.Equal(t, lifecycle.HandlerID(42), unregistered)

	exists, err := afero.Exists(fixture.Filesystem, fixture.RootDir)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestWriteConfigAndLockAndModsFolder(t *testing.T) {
	fixture, err := New(t)
	require.NoError(t, err)

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Name: "Example Mod", ID: "mod-id", Type: models.MODRINTH},
		},
	}

	err = fixture.WriteConfig(context.Background(), cfg)
	require.NoError(t, err)

	lock := []models.ModInstall{
		{Name: "Example Mod", ID: "mod-id", Type: models.MODRINTH},
	}
	err = fixture.WriteLock(context.Background(), lock)
	require.NoError(t, err)

	modsFolder, err := fixture.EnsureModsFolder(cfg)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(fixture.RootDir, "mods"), modsFolder)

	_, err = fixture.Filesystem.Stat(filepath.Join(fixture.RootDir, "modlist.json"))
	assert.NoError(t, err)
	_, err = fixture.Filesystem.Stat(filepath.Join(fixture.RootDir, "modlist-lock.json"))
	assert.NoError(t, err)
	_, err = fixture.Filesystem.Stat(modsFolder)
	assert.NoError(t, err)
}

func TestRegisterLifecycleCleanupLogsOnCleanupError(t *testing.T) {
	originalRegister := registerLifecycle
	t.Cleanup(func() {
		registerLifecycle = originalRegister
	})

	var storedHandler lifecycle.Handler
	registerLifecycle = func(handler lifecycle.Handler) lifecycle.HandlerID {
		storedHandler = handler
		return 10
	}

	fixture := &Fixture{
		Filesystem: removeAllErrorFilesystem{
			Fs:  afero.NewMemMapFs(),
			err: errors.New("remove failed"),
		},
		RootDir: "root",
	}

	registerLifecycleCleanup(fixture)
	require.NotNil(t, storedHandler)
	storedHandler(os.Interrupt)
}

func TestEnsureModsFolderReturnsErrorWhenCreateFails(t *testing.T) {
	fixture, err := New(t)
	require.NoError(t, err)

	expectedErr := errors.New("mkdir failed")
	fixture.Filesystem = mkdirAllErrorFilesystem{
		Fs:  fixture.Filesystem,
		err: expectedErr,
	}

	_, err = fixture.EnsureModsFolder(models.ModsJSON{ModsFolder: "mods"})
	assert.ErrorIs(t, err, expectedErr)
}

func TestCleanupIsIdempotent(t *testing.T) {
	fixture, err := New(t)
	require.NoError(t, err)

	err = fixture.Cleanup()
	require.NoError(t, err)

	err = fixture.Cleanup()
	assert.NoError(t, err)
}

func TestEnsureModsFolderReturnsErrorWhenFixtureIsNil(t *testing.T) {
	fixture := (*Fixture)(nil)
	_, err := fixture.EnsureModsFolder(models.ModsJSON{})
	assert.Error(t, err)
}

func TestCleanupReturnsNilWhenFixtureIsNil(t *testing.T) {
	fixture := (*Fixture)(nil)
	assert.NoError(t, fixture.Cleanup())
}

func TestRegisterTestCleanupReportsErrors(t *testing.T) {
	originalReporter := reportCleanupError
	t.Cleanup(func() {
		reportCleanupError = originalReporter
	})

	var reported []string
	reportCleanupError = func(test testing.TB, cleanupErr error) {
		reported = append(reported, cleanupErr.Error())
	}

	passed := t.Run("cleanup failure reports error", func(subtest *testing.T) {
		fixture := &Fixture{
			Filesystem: removeAllErrorFilesystem{
				Fs:  afero.NewMemMapFs(),
				err: errors.New("remove failed"),
			},
			RootDir: "root",
		}

		registerTestCleanup(subtest, fixture)
	})
	assert.True(t, passed)
	require.Len(t, reported, 1)
}

func TestWriteConfigReturnsErrorWhenFixtureIsNil(t *testing.T) {
	fixture := (*Fixture)(nil)
	err := fixture.WriteConfig(context.Background(), models.ModsJSON{})
	assert.Error(t, err)
}

func TestWriteLockReturnsErrorWhenFixtureIsNil(t *testing.T) {
	fixture := (*Fixture)(nil)
	err := fixture.WriteLock(context.Background(), nil)
	assert.Error(t, err)
}

func TestNewIgnoresNilOption(t *testing.T) {
	fixture, err := New(t, nil)
	require.NoError(t, err)
	require.NoError(t, fixture.Cleanup())
}

func TestWithBaseDirUsesProvidedParent(t *testing.T) {
	baseDir := t.TempDir()

	fixture, err := New(t, WithFS(afero.NewOsFs()), WithBaseDir(baseDir))
	require.NoError(t, err)

	assert.Equal(t, baseDir, filepath.Dir(fixture.RootDir))

	require.NoError(t, fixture.Cleanup())
}

func TestCleanupHandlesRemovalFailure(t *testing.T) {
	expectedErr := errors.New("remove failed")
	filesystem := removeAllErrorFilesystem{
		Fs:  afero.NewMemMapFs(),
		err: expectedErr,
	}

	fixture, err := New(t, WithFS(filesystem))
	require.NoError(t, err)

	err = fixture.Cleanup()
	assert.ErrorIs(t, err, expectedErr)
}

type removeAllErrorFilesystem struct {
	afero.Fs
	err error
}

func (filesystem removeAllErrorFilesystem) RemoveAll(name string) error {
	return filesystem.err
}

type mkdirAllErrorFilesystem struct {
	afero.Fs
	err error
}

func (filesystem mkdirAllErrorFilesystem) MkdirAll(path string, perm os.FileMode) error {
	return filesystem.err
}

func withNilFilesystem() Option {
	return func(options *fixtureOptions) error {
		options.filesystem = nil
		return nil
	}
}
