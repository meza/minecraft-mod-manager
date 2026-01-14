// Package modlistfixture provides test helpers for setting up modlist files.
package modlistfixture

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/spf13/afero"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/lifecycle"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

var registerLifecycle = lifecycle.Register
var unregisterLifecycle = lifecycle.Unregister
var reportCleanupError = func(test testing.TB, cleanupErr error) {
	test.Errorf("modlistfixture cleanup failed: %v", cleanupErr)
}

// Fixture provides access to a temporary modlist layout for tests.
//
// Contract:
// - Filesystem and paths are created by New; callers should not mutate RootDir directly.
// - Cleanup removes RootDir and unregisters any lifecycle hook (idempotent).
// - When created with WithOSFilesystem, Cleanup also runs on SIGINT/SIGTERM.
type Fixture struct {
	Filesystem afero.Fs
	RootDir    string
	ConfigPath string
	LockPath   string
	Meta       config.Metadata

	lifecycleHandlerID lifecycle.HandlerID
	cleanupOnce        sync.Once
	cleanupErr         error
}

type fixtureOptions struct {
	filesystem        afero.Fs
	baseDir           string
	registerLifecycle bool
}

// Option configures fixture creation.
type Option func(options *fixtureOptions) error

// WithFS overrides the filesystem used by the fixture.
//
// Contract:
// - Disables lifecycle cleanup registration unless WithOSFilesystem is applied later.
// - Returns an error when the filesystem is nil.
func WithFS(filesystem afero.Fs) Option {
	return func(options *fixtureOptions) error {
		if filesystem == nil {
			return errors.New("filesystem is nil")
		}
		options.filesystem = filesystem
		options.registerLifecycle = false
		return nil
	}
}

// WithOSFilesystem uses the real OS filesystem and registers a lifecycle cleanup hook.
//
// Contract:
// - Sets the filesystem to afero.NewOsFs().
// - Enables cleanup on SIGINT/SIGTERM via internal/lifecycle.
func WithOSFilesystem() Option {
	return func(options *fixtureOptions) error {
		options.filesystem = afero.NewOsFs()
		options.registerLifecycle = true
		return nil
	}
}

// WithBaseDir sets the parent directory for temp fixture creation.
//
// Contract:
// - The directory must exist when using the OS filesystem.
// - When empty, the OS or afero TempDir default is used.
func WithBaseDir(path string) Option {
	return func(options *fixtureOptions) error {
		options.baseDir = path
		return nil
	}
}

// New creates a fixture with a temporary directory and registers cleanup.
//
// Contract:
// - Default filesystem is in-memory (afero.NewMemMapFs).
// - ConfigPath is RootDir/modlist.json and LockPath is RootDir/modlist-lock.json.
// - Cleanup is registered with testing.TB and is safe to call multiple times.
// - Lifecycle cleanup is registered only when WithOSFilesystem is used.
func New(test testing.TB, configuredOptions ...Option) (*Fixture, error) {
	if test == nil {
		return nil, errors.New("test is nil")
	}
	test.Helper()

	fixtureSettings, err := applyOptions(configuredOptions)
	if err != nil {
		return nil, err
	}
	if fixtureSettings.filesystem == nil {
		return nil, errors.New("filesystem is nil")
	}

	rootDir, err := afero.TempDir(fixtureSettings.filesystem, fixtureSettings.baseDir, "mmm-test-")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	configPath := filepath.Join(rootDir, "modlist.json")
	meta := config.NewMetadata(configPath)
	fixture := &Fixture{
		Filesystem: fixtureSettings.filesystem,
		RootDir:    rootDir,
		ConfigPath: configPath,
		LockPath:   meta.LockPath(),
		Meta:       meta,
	}

	if fixtureSettings.registerLifecycle {
		registerLifecycleCleanup(fixture)
	}
	registerTestCleanup(test, fixture)

	return fixture, nil
}

func applyOptions(configuredOptions []Option) (fixtureOptions, error) {
	fixtureSettings := fixtureOptions{
		filesystem:        afero.NewMemMapFs(),
		registerLifecycle: false,
	}
	for _, option := range configuredOptions {
		if option == nil {
			continue
		}
		if err := option(&fixtureSettings); err != nil {
			return fixtureOptions{}, err
		}
	}
	return fixtureSettings, nil
}

func registerLifecycleCleanup(fixture *Fixture) {
	fixture.lifecycleHandlerID = registerLifecycle(func(sig os.Signal) {
		_ = sig
		ran, cleanupErr := fixture.runCleanup()
		if ran && cleanupErr != nil {
			log.Printf("modlistfixture cleanup failed: %v", cleanupErr)
		}
	})
}

func registerTestCleanup(test testing.TB, fixture *Fixture) {
	test.Cleanup(func() {
		ran, cleanupErr := fixture.runCleanup()
		if ran && cleanupErr != nil {
			reportCleanupError(test, cleanupErr)
		}
	})
}

// Cleanup removes the temporary directory and unregisters any lifecycle hook.
//
// Contract:
// - Safe to call multiple times.
// - Returns the first cleanup error, if any.
func (fixture *Fixture) Cleanup() error {
	if fixture == nil {
		return nil
	}
	_, cleanupErr := fixture.runCleanup()
	return cleanupErr
}

func (fixture *Fixture) runCleanup() (bool, error) {
	ran := false
	fixture.cleanupOnce.Do(func() {
		ran = true
		if fixture.lifecycleHandlerID != 0 {
			unregisterLifecycle(fixture.lifecycleHandlerID)
		}
		fixture.cleanupErr = fixture.Filesystem.RemoveAll(fixture.RootDir)
	})
	return ran, fixture.cleanupErr
}

// WriteConfig writes the provided config to modlist.json.
//
// Contract:
// - Returns an error when the fixture is nil or the write fails.
func (fixture *Fixture) WriteConfig(ctx context.Context, configFile models.ModsJSON) error {
	if fixture == nil {
		return errors.New("fixture is nil")
	}
	return config.WriteConfig(ctx, fixture.Filesystem, fixture.Meta, configFile)
}

// WriteLock writes the provided lock entries to modlist-lock.json.
//
// Contract:
// - Returns an error when the fixture is nil or the write fails.
func (fixture *Fixture) WriteLock(ctx context.Context, lock []models.ModInstall) error {
	if fixture == nil {
		return errors.New("fixture is nil")
	}
	return config.WriteLock(ctx, fixture.Filesystem, fixture.Meta, lock)
}

// EnsureModsFolder creates the mods folder for the provided config.
//
// Contract:
// - Returns the resolved mods folder path.
// - Returns an error when the fixture is nil or the folder cannot be created.
func (fixture *Fixture) EnsureModsFolder(configFile models.ModsJSON) (string, error) {
	if fixture == nil {
		return "", errors.New("fixture is nil")
	}
	modsFolder := fixture.Meta.ModsFolderPath(configFile)
	if err := fixture.Filesystem.MkdirAll(modsFolder, 0o755); err != nil {
		return "", fmt.Errorf("create mods folder: %w", err)
	}
	return modsFolder, nil
}
