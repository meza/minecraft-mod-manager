package remove

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

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/locksync"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/meza/minecraft-mod-manager/testutil/terminal"
)

type errorWriter struct {
	err error
}

func (writer errorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

func TestResolveMatchesForRemovePrefersLockEntries(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "sodium", Name: "Config Sodium"},
		},
	}
	lock := []models.ModInstall{{
		Type: models.MODRINTH,
		ID:   "sodium",
		Name: "Lock Sodium",
	}}

	matches, err := resolveMatchesForRemove([]string{"lock*"}, cfg, lock)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, "sodium", matches[0].mod.ID)
	assert.Equal(t, "Lock Sodium", matches[0].mod.Name)
	assert.True(t, matches[0].hasLockEntry)
}

func TestResolveMatchesForRemoveUsesConfigNameWhenLockNameMissing(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Config Name"}},
	}
	lock := []models.ModInstall{{Type: models.MODRINTH, ID: "sodium"}}

	matches, err := resolveMatchesForRemove([]string{"sod*"}, cfg, lock)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, "Config Name", matches[0].mod.Name)
}

func TestResolveMatchesForRemoveSkipsConfigMatchWhenLockEntryExists(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Config Name"}},
	}
	lock := []models.ModInstall{{Type: models.MODRINTH, ID: "sodium", Name: "Lock Name"}}

	matches, err := resolveMatchesForRemove([]string{"config*"}, cfg, lock)
	require.NoError(t, err)
	assert.Empty(t, matches)
}

func TestResolveMatchesForRemoveFallsBackToConfigWhenLockMissing(t *testing.T) {
	cfg := models.ModsJSON{Mods: []models.Mod{{Type: models.MODRINTH, ID: "x", Name: "y"}}}
	lock := []models.ModInstall{{Type: models.MODRINTH, ID: "other", Name: "Other"}}

	matches, err := resolveMatchesForRemove([]string{"y"}, cfg, lock)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, "x", matches[0].mod.ID)
	assert.False(t, matches[0].hasLockEntry)
}

func TestResolveMatchesForRemoveIncludesConfigEntriesWithoutLockWhenPatternMatchesBoth(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
			{Type: models.MODRINTH, ID: "iris", Name: "Iris"},
		},
	}
	lock := []models.ModInstall{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}}

	matches, err := resolveMatchesForRemove([]string{"*"}, cfg, lock)
	require.NoError(t, err)
	require.Len(t, matches, 2)
	assert.True(t, matches[0].hasLockEntry || matches[1].hasLockEntry)
	assert.True(t, matches[0].hasLockEntry != matches[1].hasLockEntry)
}

func TestResolveMatchesForRemoveErrorsOnInvalidPattern(t *testing.T) {
	cfg := models.ModsJSON{Mods: []models.Mod{{Type: models.MODRINTH, ID: "x", Name: "y"}}}
	_, err := resolveMatchesForRemove([]string{"["}, cfg, nil)
	assert.Error(t, err)
}

func TestResolveMatchesForRemoveSkipsBlanksAndDedupes(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
		},
	}
	lock := []models.ModInstall{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}}

	matches, err := resolveMatchesForRemove([]string{"", "sod*", "SOD*"}, cfg, lock)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, "sodium", matches[0].mod.ID)
	assert.True(t, matches[0].hasLockEntry)
}

func TestGlobMatchesReturnsFalseOnInvalidPattern(t *testing.T) {
	assert.False(t, globMatches("[", "value"))
}

func TestBuildRemoveItemsSortsByNameThenPlatformThenID(t *testing.T) {
	matches := []removeMatch{
		{mod: models.Mod{Type: models.CURSEFORGE, ID: "b", Name: "Same"}},
		{mod: models.Mod{Type: models.MODRINTH, ID: "a", Name: "Same"}},
		{mod: models.Mod{Type: models.MODRINTH, ID: "c", Name: "Alpha"}},
	}

	items := buildRemoveItems(matches)
	require.Len(t, items, 3)
	assert.Equal(t, "Alpha", items[0].Mod.Name)
	assert.Equal(t, "b", items[1].Mod.ID)
	assert.Equal(t, "a", items[2].Mod.ID)
}

func TestRunRemoveNonTTYRequiresForce(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	t.Setenv("LANG", "en_GB.UTF-8")
	terminal.ApplyFixtures(t, terminal.WithoutMMMTestEnv())

	fs := afero.NewMemMapFs()
	log := logger.New(io.Discard, io.Discard, false, false)

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
		},
	}

	meta := config.NewMetadata("modlist.json")
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{{Type: models.MODRINTH, ID: "sodium", FileName: "sodium.jar"}}))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "sodium.jar"), []byte("mod"), 0644))

	cmd := &cobra.Command{}
	input := terminal.NewDevice()
	output := terminal.NewDevice()
	terminal.ApplyTerminalDetection(t, input, output, terminal.NonTTYCapabilities())
	cmd.SetIn(input)
	cmd.SetOut(output)

	deps := removeDeps{
		fs:     fs,
		logger: log,
		runTea: defaultRunTea,
	}

	_, usedInteractive, err := runRemove(context.Background(), cmd, removeOptions{
		ConfigPath: meta.ConfigPath,
		Lookups:    []string{"*"},
	}, deps)
	assert.Error(t, err)
	assert.False(t, usedInteractive)
	assert.Contains(t, output.String(), "Non-interactive terminal detected")
	assert.Contains(t, output.String(), "--force")

	exists, err := afero.Exists(fs, filepath.Join(meta.ModsFolderPath(cfg), "sodium.jar"))
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestRunRemoveCancelDoesNotCreateLockFile(t *testing.T) {
	terminal.ApplyFixtures(t)

	fs := afero.NewMemMapFs()
	output := terminal.NewDevice()
	input := terminal.NewDevice()
	terminal.ApplyTerminalDetection(t, input, output, terminal.TTYCapabilities())
	log := logger.New(io.Discard, io.Discard, false, false)

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}},
	}
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	originalConfig, err := afero.ReadFile(fs, meta.ConfigPath)
	require.NoError(t, err)

	cmd := &cobra.Command{}
	cmd.SetIn(input)
	cmd.SetOut(output)

	deps := removeDeps{
		fs:     fs,
		logger: log,
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return removeConfirmModel{confirmed: false}, nil
		},
	}

	removed, usedInteractive, err := runRemove(context.Background(), cmd, removeOptions{
		ConfigPath: meta.ConfigPath,
		Lookups:    []string{"sod*"},
	}, deps)
	require.NoError(t, err)
	assert.Equal(t, 0, removed)
	assert.True(t, usedInteractive)

	exists, err := afero.Exists(fs, meta.LockPath())
	require.NoError(t, err)
	assert.False(t, exists)

	updatedConfig, err := afero.ReadFile(fs, meta.ConfigPath)
	require.NoError(t, err)
	assert.Equal(t, originalConfig, updatedConfig)
}

func TestRunRemoveNonTTYRefusalDoesNotCreateLockFile(t *testing.T) {
	terminal.ApplyFixtures(t)

	fs := afero.NewMemMapFs()
	output := terminal.NewDevice()
	input := terminal.NewDevice()
	terminal.ApplyTerminalDetection(t, input, output, terminal.NonTTYCapabilities())
	log := logger.New(io.Discard, io.Discard, false, false)

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}},
	}
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	originalConfig, err := afero.ReadFile(fs, meta.ConfigPath)
	require.NoError(t, err)

	cmd := &cobra.Command{}
	cmd.SetIn(input)
	cmd.SetOut(output)

	deps := removeDeps{fs: fs, logger: log, runTea: defaultRunTea}

	_, usedInteractive, err := runRemove(context.Background(), cmd, removeOptions{
		ConfigPath: meta.ConfigPath,
		Lookups:    []string{"sod*"},
	}, deps)
	assert.Error(t, err)
	assert.False(t, usedInteractive)

	exists, err := afero.Exists(fs, meta.LockPath())
	require.NoError(t, err)
	assert.False(t, exists)

	updatedConfig, err := afero.ReadFile(fs, meta.ConfigPath)
	require.NoError(t, err)
	assert.Equal(t, originalConfig, updatedConfig)
}

func TestRunRemoveNoMatchesOutputsMessage(t *testing.T) {
	fs := afero.NewMemMapFs()
	var out bytes.Buffer
	log := logger.New(&out, &out, false, false)

	cfg := models.ModsJSON{
		Mods: []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}},
	}
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)

	deps := removeDeps{fs: fs, logger: log, runTea: defaultRunTea}

	removed, _, err := runRemove(context.Background(), cmd, removeOptions{
		ConfigPath: meta.ConfigPath,
		Lookups:    []string{"none"},
	}, deps)
	require.NoError(t, err)
	assert.Equal(t, 0, removed)
	assert.Equal(t, "No matching mods found.\n", out.String())
}

func TestRunRemoveSuccessRemovesConfigAndLockAndDeletesFile(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	fs := afero.NewMemMapFs()
	var out bytes.Buffer
	log := logger.New(&out, &out, false, false)

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}},
	}
	meta := config.NewMetadata("config/modlist.json")
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	modsDir := meta.ModsFolderPath(cfg)
	require.NoError(t, fs.MkdirAll(modsDir, 0755))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "sodium",
		Name:     "Sodium",
		FileName: "sodium.jar",
	}}))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(modsDir, "sodium.jar"), []byte("x"), 0644))

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)

	deps := removeDeps{fs: fs, logger: log, runTea: defaultRunTea}

	removed, _, err := runRemove(context.Background(), cmd, removeOptions{
		ConfigPath: meta.ConfigPath,
		Force:      true,
		Lookups:    []string{"sod*"},
	}, deps)
	require.NoError(t, err)
	assert.Equal(t, 1, removed)

	exists, err := afero.Exists(fs, filepath.Join(modsDir, "sodium.jar"))
	require.NoError(t, err)
	assert.False(t, exists)

	updatedLock, err := config.ReadLock(context.Background(), fs, meta)
	require.NoError(t, err)
	assert.Empty(t, updatedLock)

	updatedCfg, err := config.ReadConfig(context.Background(), fs, meta)
	require.NoError(t, err)
	assert.Empty(t, updatedCfg.Mods)

	assert.Equal(t, "V Sodium (sodium)\n\nV Remove complete.\n", out.String())
}

func TestRunRemoveUsesLockNameAndRemovesConfigByID(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	fs := afero.NewMemMapFs()
	var out bytes.Buffer
	log := logger.New(&out, &out, false, false)

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Config Name"}},
	}
	meta := config.NewMetadata("config/modlist.json")
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	modsDir := meta.ModsFolderPath(cfg)
	require.NoError(t, fs.MkdirAll(modsDir, 0755))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "sodium",
		Name:     "Lock Name",
		FileName: "sodium.jar",
	}}))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(modsDir, "sodium.jar"), []byte("x"), 0644))

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)

	deps := removeDeps{fs: fs, logger: log, runTea: defaultRunTea}

	removed, _, err := runRemove(context.Background(), cmd, removeOptions{
		ConfigPath: meta.ConfigPath,
		Force:      true,
		Lookups:    []string{"lock*"},
	}, deps)
	require.NoError(t, err)
	assert.Equal(t, 1, removed)

	updatedCfg, err := config.ReadConfig(context.Background(), fs, meta)
	require.NoError(t, err)
	assert.Empty(t, updatedCfg.Mods)

	updatedLock, err := config.ReadLock(context.Background(), fs, meta)
	require.NoError(t, err)
	assert.Empty(t, updatedLock)

	assert.Contains(t, out.String(), "Lock Name (sodium)")
}

func TestRunRemoveRemovesConfigWhenLockEntryMissing(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	fs := afero.NewMemMapFs()
	var out bytes.Buffer
	log := logger.New(&out, &out, false, false)

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}},
	}
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)

	deps := removeDeps{fs: fs, logger: log, runTea: defaultRunTea}

	removed, _, err := runRemove(context.Background(), cmd, removeOptions{
		ConfigPath: meta.ConfigPath,
		Force:      true,
		Lookups:    []string{"sod*"},
	}, deps)
	require.NoError(t, err)
	assert.Equal(t, 1, removed)

	updatedCfg, err := config.ReadConfig(context.Background(), fs, meta)
	require.NoError(t, err)
	assert.Empty(t, updatedCfg.Mods)
}

func TestRunRemoveSkipsMissingFileStillUpdatesConfigAndLock(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	fs := afero.NewMemMapFs()
	var out bytes.Buffer
	log := logger.New(&out, &out, false, false)

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}},
	}
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "sodium",
		Name:     "Sodium",
		FileName: "missing.jar",
	}}))

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)

	deps := removeDeps{fs: fs, logger: log, runTea: defaultRunTea}

	removed, _, err := runRemove(context.Background(), cmd, removeOptions{
		ConfigPath: meta.ConfigPath,
		Force:      true,
		Lookups:    []string{"sod*"},
	}, deps)
	require.NoError(t, err)
	assert.Equal(t, 1, removed)

	updatedLock, err := config.ReadLock(context.Background(), fs, meta)
	require.NoError(t, err)
	assert.Empty(t, updatedLock)

	updatedCfg, err := config.ReadConfig(context.Background(), fs, meta)
	require.NoError(t, err)
	assert.Empty(t, updatedCfg.Mods)

	assert.Equal(t, "V Sodium (sodium)\n\nV Remove complete.\n", out.String())
}

func TestRunRemoveDeleteFailureKeepsEntries(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	baseFs := afero.NewMemMapFs()
	var out bytes.Buffer
	log := logger.New(&out, &out, false, false)

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}},
	}
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	require.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), baseFs, meta, []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "sodium",
		FileName: "bad.jar",
	}}))
	require.NoError(t, afero.WriteFile(baseFs, filepath.Join(meta.ModsFolderPath(cfg), "bad.jar"), []byte("x"), 0644))

	fs := removeErrorFs{
		Fs:       baseFs,
		failPath: filepath.Join(meta.ModsFolderPath(cfg), "bad.jar"),
		err:      errors.New("remove failed"),
	}

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)

	deps := removeDeps{fs: fs, logger: log, runTea: defaultRunTea}

	_, _, err := runRemove(context.Background(), cmd, removeOptions{
		ConfigPath: meta.ConfigPath,
		Force:      true,
		Lookups:    []string{"sod*"},
	}, deps)
	assert.Error(t, err)

	updatedLock, readErr := config.ReadLock(context.Background(), baseFs, meta)
	require.NoError(t, readErr)
	require.Len(t, updatedLock, 1)

	updatedCfg, readErr := config.ReadConfig(context.Background(), baseFs, meta)
	require.NoError(t, readErr)
	require.Len(t, updatedCfg.Mods, 1)

	expected := "X Sodium (sodium) delete failed: remove failed\n\n!! Remove incomplete.\nFix the reason and rerun mmm remove.\n"
	assert.Equal(t, expected, out.String())
}

func TestRunRemoveInvalidLockFilenameKeepsEntries(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	fs := afero.NewMemMapFs()
	var out bytes.Buffer
	log := logger.New(&out, &out, false, false)

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}},
	}
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "sodium",
		FileName: "",
	}}))

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)

	deps := removeDeps{fs: fs, logger: log, runTea: defaultRunTea}

	_, _, err := runRemove(context.Background(), cmd, removeOptions{
		ConfigPath: meta.ConfigPath,
		Force:      true,
		Lookups:    []string{"sod*"},
	}, deps)
	assert.Error(t, err)

	updatedLock, readErr := config.ReadLock(context.Background(), fs, meta)
	require.NoError(t, readErr)
	require.Len(t, updatedLock, 1)

	updatedCfg, readErr := config.ReadConfig(context.Background(), fs, meta)
	require.NoError(t, readErr)
	require.Len(t, updatedCfg.Mods, 1)

	expected := "X Sodium (sodium) delete failed: lock file filename is unsafe ((empty))\n\n!! Remove incomplete.\nFix the reason and rerun mmm remove.\n"
	assert.Equal(t, expected, out.String())
}

func TestRunRemoveQuietSuccessSuppressesOutput(t *testing.T) {
	fs := afero.NewMemMapFs()
	var out bytes.Buffer
	log := logger.New(&out, &out, true, false)

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}},
	}
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "sodium",
		FileName: "missing.jar",
	}}))

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)

	deps := removeDeps{fs: fs, logger: log, runTea: defaultRunTea}

	removed, _, err := runRemove(context.Background(), cmd, removeOptions{
		ConfigPath: meta.ConfigPath,
		Quiet:      true,
		Force:      true,
		Lookups:    []string{"sod*"},
	}, deps)
	require.NoError(t, err)
	assert.Equal(t, 1, removed)
	assert.Equal(t, "", out.String())
}

func TestRunRemoveQuietFailureOutputsOnlyFailed(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	baseFs := afero.NewMemMapFs()
	var out bytes.Buffer
	log := logger.New(&out, &out, true, false)

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}},
	}
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	require.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), baseFs, meta, []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "sodium",
		FileName: "bad.jar",
	}}))
	require.NoError(t, afero.WriteFile(baseFs, filepath.Join(meta.ModsFolderPath(cfg), "bad.jar"), []byte("x"), 0644))

	fs := removeErrorFs{
		Fs:       baseFs,
		failPath: filepath.Join(meta.ModsFolderPath(cfg), "bad.jar"),
		err:      errors.New("remove failed"),
	}

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)

	deps := removeDeps{fs: fs, logger: log, runTea: defaultRunTea}

	_, _, err := runRemove(context.Background(), cmd, removeOptions{
		ConfigPath: meta.ConfigPath,
		Quiet:      true,
		Force:      true,
		Lookups:    []string{"sod*"},
	}, deps)
	assert.Error(t, err)

	expected := "X Sodium (sodium) delete failed: remove failed\n\n!! Remove incomplete.\nFix the reason and rerun mmm remove.\n"
	assert.Equal(t, expected, out.String())
}

func TestRunRemoveUnattendedConfigMissingOutputsError(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	fs := afero.NewMemMapFs()
	var out bytes.Buffer
	log := logger.New(&out, &out, true, false)

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)

	deps := removeDeps{fs: fs, logger: log, runTea: defaultRunTea}

	_, _, err := runRemove(context.Background(), cmd, removeOptions{
		ConfigPath: "missing.json",
		Unattended: true,
		Lookups:    []string{"mod"},
	}, deps)
	assert.Error(t, err)
	assert.True(t, clierrors.IsHandled(err))
	assert.Contains(t, out.String(), "No configuration file found")
	assert.Contains(t, out.String(), "Run `mmm init` to create one.")
}

func TestRunRemoveWriteLockFailureOutputsError(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	baseFs := afero.NewMemMapFs()
	var out bytes.Buffer
	log := logger.New(&out, &out, false, false)

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}},
	}
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), baseFs, meta, []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "sodium",
		FileName: "missing.jar",
	}}))

	fs := renameErrorFs{Fs: baseFs, failNew: meta.LockPath(), err: errors.New("rename failed")}

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)

	deps := removeDeps{fs: fs, logger: log, runTea: defaultRunTea}

	_, _, err := runRemove(context.Background(), cmd, removeOptions{
		ConfigPath: meta.ConfigPath,
		Force:      true,
		Lookups:    []string{"sod*"},
	}, deps)
	assert.Error(t, err)
	assert.True(t, clierrors.IsHandled(err))
	assert.Contains(t, out.String(), "Remove failed")

	updatedCfg, err := config.ReadConfig(context.Background(), baseFs, meta)
	require.NoError(t, err)
	assert.Equal(t, cfg.Mods, updatedCfg.Mods)

	updatedLock, err := config.ReadLock(context.Background(), baseFs, meta)
	require.NoError(t, err)
	assert.Len(t, updatedLock, 1)
	assert.Equal(t, "sodium", updatedLock[0].ID)
}

func TestRunRemoveWriteConfigFailureOutputsError(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	baseFs := afero.NewMemMapFs()
	var out bytes.Buffer
	log := logger.New(&out, &out, false, false)

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}},
	}
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), baseFs, meta, []models.ModInstall{{
		Type:     models.MODRINTH,
		ID:       "sodium",
		FileName: "missing.jar",
	}}))

	fs := renameErrorFs{Fs: baseFs, failNew: meta.ConfigPath, err: errors.New("rename failed")}

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)

	deps := removeDeps{fs: fs, logger: log, runTea: defaultRunTea}

	_, _, err := runRemove(context.Background(), cmd, removeOptions{
		ConfigPath: meta.ConfigPath,
		Force:      true,
		Lookups:    []string{"sod*"},
	}, deps)
	assert.Error(t, err)
	assert.True(t, clierrors.IsHandled(err))
	assert.Contains(t, out.String(), "Remove failed")

	updatedCfg, err := config.ReadConfig(context.Background(), baseFs, meta)
	require.NoError(t, err)
	assert.Equal(t, cfg.Mods, updatedCfg.Mods)

	updatedLock, err := config.ReadLock(context.Background(), baseFs, meta)
	require.NoError(t, err)
	assert.Len(t, updatedLock, 1)
	assert.Equal(t, "sodium", updatedLock[0].ID)
}

func TestRemoveModelViewFinalFailureIncludesSummary(t *testing.T) {
	model := newRemoveModel(context.Background(), view.ColorDisabled, []removeItem{{
		Mod:           models.Mod{Name: "Sodium", ID: "sodium"},
		Status:        removeItemFailed,
		FailureReason: "oops",
	}}, map[string]int{"modrinth:sodium": 0}, nil)
	model.done = true
	model.outcome = removeExecutionOutcome{err: errors.New("fail"), errType: removeExecutionErrorDelete}

	viewText := model.View()
	assert.Contains(t, viewText, "Remove incomplete.")
}

func TestRemoveTranscriptModelHandlesOutputLineError(t *testing.T) {
	model := newRemoveTranscriptModel(context.Background(), view.ColorDisabled, nil, nil, io.Discard, nil)
	updated, _ := model.Update(outputLineErrorMsg{Err: errors.New("write failed")})
	updatedModel := updated.(*removeTranscriptModel)
	assert.Equal(t, removeExecutionErrorUnknown, updatedModel.outcome.errType)
}

func TestRunRemoveReturnsErrorWhenUnmanagedNoticeFails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Name: "Alpha", ID: "alpha", Type: models.MODRINTH},
		},
	}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	failPath := filepath.Join(meta.Dir(), ".mmmignore")
	wrapped := statErrorFs{Fs: fs, failPath: failPath, err: errors.New("stat failed")}

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	_, _, err := runRemove(context.Background(), cmd, removeOptions{
		ConfigPath: meta.ConfigPath,
		Lookups:    []string{"alpha"},
		LockSync:   locksync.PolicyFlags{Skip: true},
	}, removeDeps{
		fs: wrapped,
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	})

	assert.True(t, clierrors.IsHandled(err))
}

func TestRunRemoveReturnsOutputErrorWhenUnmanagedNoticeWriteFails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Name: "Alpha", ID: "alpha", Type: models.MODRINTH},
		},
	}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar"), []byte("data"), 0644))

	writeErr := errors.New("write failed")
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	_, _, err := runRemove(context.Background(), cmd, removeOptions{
		ConfigPath: meta.ConfigPath,
		Lookups:    []string{"alpha"},
		LockSync:   locksync.PolicyFlags{Skip: true},
	}, removeDeps{
		fs: fs,
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			if _, ok := model.(view.OutputLinesModel); ok {
				return view.OutputLinesModel{Err: writeErr}, nil
			}
			return model, nil
		},
	})

	assert.ErrorIs(t, err, writeErr)
}

type renameErrorFs struct {
	afero.Fs
	failNew string
	err     error
}

func (filesystem renameErrorFs) Rename(oldname, newname string) error {
	if filepath.Clean(newname) == filepath.Clean(filesystem.failNew) {
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
