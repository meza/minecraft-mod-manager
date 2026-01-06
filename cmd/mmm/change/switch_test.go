package change

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

func TestSwitchToDownloadedModsMkdirError(t *testing.T) {
	input := changeSwitchInput{
		ctx:           context.Background(),
		deps:          changeDeps{fs: afero.NewMemMapFs(), mkdirAll: func(afero.Fs, string, os.FileMode) error { return errors.New("mkdir failed") }},
		meta:          config.NewMetadata("/cfg/modlist.json"),
		cfg:           models.ModsJSON{ModsFolder: "mods"},
		lock:          []models.ModInstall{},
		targetVersion: "1.20.1",
		items:         []changeItem{},
		lockEntries:   map[string]models.ModInstall{},
		stagedPaths:   map[string]string{},
		resolvedNames: map[string]string{},
		stagingBackup: "/staging/backup",
		changeState:   newChangeExecutionState(changeExecutionInput{items: []changeItem{}, indexByKey: map[string]int{}}, nil),
	}

	err := switchToDownloadedMods(input)
	assert.Error(t, err)
}

func TestApplySwitchForItemContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}
	state := newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil)

	input := changeSwitchInput{
		ctx:         ctx,
		deps:        changeDeps{fs: afero.NewMemMapFs()},
		meta:        config.NewMetadata("/cfg/modlist.json"),
		cfg:         models.ModsJSON{ModsFolder: "mods"},
		lock:        []models.ModInstall{},
		items:       items,
		lockEntries: map[string]models.ModInstall{},
		stagedPaths: map[string]string{},
		changeState: state,
	}

	err := applySwitchForItem(input, map[string]models.ModInstall{}, input.meta.ModsFolderPath(input.cfg), items[0], &switchState{})
	assert.Error(t, err)
	assert.Equal(t, changeSwitchInProgress, state.snapshot()[0].SwitchStatus)
}

func TestApplySwitchForItemSkipped(t *testing.T) {
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod, Skipped: true}}
	index := map[string]int{changeModKey(mod): 0}
	state := newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil)

	input := changeSwitchInput{
		ctx:         context.Background(),
		deps:        changeDeps{fs: afero.NewMemMapFs()},
		meta:        config.NewMetadata("/cfg/modlist.json"),
		cfg:         models.ModsJSON{ModsFolder: "mods"},
		items:       items,
		lockEntries: map[string]models.ModInstall{},
		stagedPaths: map[string]string{},
		changeState: state,
	}

	err := applySwitchForItem(input, map[string]models.ModInstall{}, input.meta.ModsFolderPath(input.cfg), items[0], &switchState{})
	assert.NoError(t, err)
	assert.Equal(t, changeSwitchSkipped, state.snapshot()[0].SwitchStatus)
}

func TestApplySwitchForItemSkippedDisable(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod, Skipped: true}}
	index := map[string]int{changeModKey(mod): 0}
	state := newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil)

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "alpha.jar"), []byte("old"), 0o644))

	lockEntry := models.ModInstall{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}
	input := changeSwitchInput{
		ctx:         context.Background(),
		deps:        changeDeps{fs: fs, renameFile: func(fs afero.Fs, from, to string) error { return fs.Rename(from, to) }},
		meta:        meta,
		cfg:         cfg,
		forcePolicy: changeForcePolicyDisableSkipped,
		items:       items,
		lockEntries: map[string]models.ModInstall{changeModKey(mod): lockEntry},
		stagedPaths: map[string]string{},
		changeState: state,
	}

	err := applySwitchForItem(input, map[string]models.ModInstall{changeModKey(mod): lockEntry}, meta.ModsFolderPath(cfg), items[0], &switchState{})
	assert.NoError(t, err)
	assert.Equal(t, changeSwitchSkipped, state.snapshot()[0].SwitchStatus)

	disabledPath := filepath.Join(meta.ModsFolderPath(cfg), "alpha.jar.disabled")
	exists, err := afero.Exists(fs, disabledPath)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestApplySwitchForItemSkippedDisableError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod, Skipped: true}}
	index := map[string]int{changeModKey(mod): 0}
	state := newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil)

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "alpha.jar"), []byte("old"), 0o644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "alpha.jar.disabled"), []byte("disabled"), 0o644))

	lockEntry := models.ModInstall{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}
	input := changeSwitchInput{
		ctx:         context.Background(),
		deps:        changeDeps{fs: fs, renameFile: func(fs afero.Fs, from, to string) error { return fs.Rename(from, to) }},
		meta:        meta,
		cfg:         cfg,
		forcePolicy: changeForcePolicyDisableSkipped,
		items:       items,
		lockEntries: map[string]models.ModInstall{changeModKey(mod): lockEntry},
		stagedPaths: map[string]string{},
		changeState: state,
	}

	err := applySwitchForItem(input, map[string]models.ModInstall{changeModKey(mod): lockEntry}, meta.ModsFolderPath(cfg), items[0], &switchState{})
	assert.Error(t, err)
	assert.Equal(t, changeSwitchFailed, state.snapshot()[0].SwitchStatus)
}

func TestApplySwitchForItemMissingLockEntry(t *testing.T) {
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}
	state := newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil)

	input := changeSwitchInput{
		ctx:         context.Background(),
		deps:        changeDeps{fs: afero.NewMemMapFs()},
		meta:        config.NewMetadata("/cfg/modlist.json"),
		cfg:         models.ModsJSON{ModsFolder: "mods"},
		items:       items,
		lockEntries: map[string]models.ModInstall{},
		stagedPaths: map[string]string{},
		changeState: state,
	}

	err := applySwitchForItem(input, map[string]models.ModInstall{}, input.meta.ModsFolderPath(input.cfg), items[0], &switchState{})
	assert.Error(t, err)
	assert.Equal(t, changeSwitchFailed, state.snapshot()[0].SwitchStatus)
}

func TestApplySwitchForItemMissingStagedPath(t *testing.T) {
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}
	state := newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil)

	lockEntry := models.ModInstall{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}
	input := changeSwitchInput{
		ctx:         context.Background(),
		deps:        changeDeps{fs: afero.NewMemMapFs()},
		meta:        config.NewMetadata("/cfg/modlist.json"),
		cfg:         models.ModsJSON{ModsFolder: "mods"},
		items:       items,
		lockEntries: map[string]models.ModInstall{changeModKey(mod): lockEntry},
		stagedPaths: map[string]string{},
		changeState: state,
	}

	err := applySwitchForItem(input, map[string]models.ModInstall{}, input.meta.ModsFolderPath(input.cfg), items[0], &switchState{})
	assert.Error(t, err)
	assert.Equal(t, changeSwitchFailed, state.snapshot()[0].SwitchStatus)
}

func TestApplySwitchForItemBackupError(t *testing.T) {
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}
	state := newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil)

	lockIndex := map[string]models.ModInstall{
		changeModKey(mod): {ID: "alpha", Type: models.MODRINTH, FileName: "mods/alpha.jar"},
	}

	input := changeSwitchInput{
		ctx:         context.Background(),
		deps:        changeDeps{fs: afero.NewMemMapFs()},
		meta:        config.NewMetadata("/cfg/modlist.json"),
		cfg:         models.ModsJSON{ModsFolder: "mods"},
		items:       items,
		lockEntries: map[string]models.ModInstall{changeModKey(mod): {ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}},
		stagedPaths: map[string]string{changeModKey(mod): "/staging/alpha.jar"},
		changeState: state,
	}

	err := applySwitchForItem(input, lockIndex, input.meta.ModsFolderPath(input.cfg), items[0], &switchState{})
	assert.Error(t, err)
	assert.Equal(t, changeSwitchFailed, state.snapshot()[0].SwitchStatus)
}

func TestApplySwitchForItemResolveError(t *testing.T) {
	meta := config.NewMetadata(filepath.Join(t.TempDir(), "missing", "modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}
	state := newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil)

	lockEntry := models.ModInstall{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}
	input := changeSwitchInput{
		ctx:         context.Background(),
		deps:        changeDeps{fs: afero.NewOsFs()},
		meta:        meta,
		cfg:         cfg,
		items:       items,
		lockEntries: map[string]models.ModInstall{changeModKey(mod): lockEntry},
		stagedPaths: map[string]string{changeModKey(mod): "/tmp/staged.jar"},
		changeState: state,
	}

	err := applySwitchForItem(input, map[string]models.ModInstall{}, meta.ModsFolderPath(cfg), items[0], &switchState{})
	assert.Error(t, err)
}

func TestApplySwitchForItemRenameError(t *testing.T) {
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}
	state := newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil)
	fs := afero.NewMemMapFs()

	lockEntry := models.ModInstall{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}
	input := changeSwitchInput{
		ctx:         context.Background(),
		deps:        changeDeps{fs: fs, renameFile: func(afero.Fs, string, string) error { return errors.New("rename failed") }},
		meta:        config.NewMetadata("/cfg/modlist.json"),
		cfg:         models.ModsJSON{ModsFolder: "mods"},
		items:       items,
		lockEntries: map[string]models.ModInstall{changeModKey(mod): lockEntry},
		stagedPaths: map[string]string{changeModKey(mod): "/staging/alpha.jar"},
		changeState: state,
	}

	err := applySwitchForItem(input, map[string]models.ModInstall{}, input.meta.ModsFolderPath(input.cfg), items[0], &switchState{})
	assert.Error(t, err)
	assert.Equal(t, changeSwitchFailed, state.snapshot()[0].SwitchStatus)
}

func TestDisableExistingJarFailsWhenDisabledExists(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	lockEntry := models.ModInstall{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "alpha.jar"), []byte("old"), 0o644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "alpha.jar.disabled"), []byte("disabled"), 0o644))

	input := changeSwitchInput{
		deps: changeDeps{fs: fs, renameFile: func(fs afero.Fs, from, to string) error { return fs.Rename(from, to) }},
		meta: meta,
		cfg:  cfg,
	}
	plan := &switchState{}

	err := disableExistingJar(input, map[string]models.ModInstall{changeModKey(mod): lockEntry}, meta.ModsFolderPath(cfg), mod, plan)
	assert.Error(t, err)
	assert.Empty(t, plan.disableds)
}

func TestDisableExistingJarNoLockEntry(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}

	input := changeSwitchInput{
		deps: changeDeps{fs: fs},
		meta: meta,
		cfg:  cfg,
	}
	plan := &switchState{}

	err := disableExistingJar(input, map[string]models.ModInstall{}, meta.ModsFolderPath(cfg), mod, plan)
	assert.NoError(t, err)
	assert.Empty(t, plan.disableds)
}

func TestDisableExistingJarInvalidFileName(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	lockEntry := models.ModInstall{ID: "alpha", Type: models.MODRINTH, FileName: "mods/alpha.jar"}

	input := changeSwitchInput{
		deps: changeDeps{fs: fs},
		meta: meta,
		cfg:  cfg,
	}
	plan := &switchState{}

	err := disableExistingJar(input, map[string]models.ModInstall{changeModKey(mod): lockEntry}, meta.ModsFolderPath(cfg), mod, plan)
	assert.Error(t, err)
}

func TestDisableExistingJarMissingOriginalFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	lockEntry := models.ModInstall{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}

	input := changeSwitchInput{
		deps: changeDeps{fs: fs},
		meta: meta,
		cfg:  cfg,
	}
	plan := &switchState{}

	err := disableExistingJar(input, map[string]models.ModInstall{changeModKey(mod): lockEntry}, meta.ModsFolderPath(cfg), mod, plan)
	assert.NoError(t, err)
}

func TestDisableExistingJarRenameError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	lockEntry := models.ModInstall{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "alpha.jar"), []byte("old"), 0o644))

	input := changeSwitchInput{
		deps: changeDeps{
			fs:         fs,
			renameFile: func(afero.Fs, string, string) error { return errors.New("rename failed") },
		},
		meta: meta,
		cfg:  cfg,
	}
	plan := &switchState{}

	err := disableExistingJar(input, map[string]models.ModInstall{changeModKey(mod): lockEntry}, meta.ModsFolderPath(cfg), mod, plan)
	assert.Error(t, err)
}

func TestDisableExistingJarSuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	lockEntry := models.ModInstall{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "alpha.jar"), []byte("old"), 0o644))

	input := changeSwitchInput{
		deps: changeDeps{
			fs:         fs,
			renameFile: func(fs afero.Fs, from, to string) error { return fs.Rename(from, to) },
		},
		meta: meta,
		cfg:  cfg,
	}
	plan := &switchState{}

	err := disableExistingJar(input, map[string]models.ModInstall{changeModKey(mod): lockEntry}, meta.ModsFolderPath(cfg), mod, plan)
	assert.NoError(t, err)
	assert.Len(t, plan.disableds, 1)

	exists, statErr := afero.Exists(fs, filepath.Join(meta.ModsFolderPath(cfg), "alpha.jar.disabled"))
	require.NoError(t, statErr)
	assert.True(t, exists)
}

func TestDisableExistingJarResolveError(t *testing.T) {
	meta := config.NewMetadata(filepath.Join(t.TempDir(), "missing", "modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	lockEntry := models.ModInstall{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}

	input := changeSwitchInput{
		deps: changeDeps{fs: afero.NewOsFs()},
		meta: meta,
		cfg:  cfg,
	}
	plan := &switchState{}

	err := disableExistingJar(input, map[string]models.ModInstall{changeModKey(mod): lockEntry}, meta.ModsFolderPath(cfg), mod, plan)
	assert.Error(t, err)
}

func TestDisableExistingJarOriginalExistsError(t *testing.T) {
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	lockEntry := models.ModInstall{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}

	originalPath := filepath.Join(meta.ModsFolderPath(cfg), "alpha.jar")
	fs := statErrorFs{Fs: afero.NewMemMapFs(), failPath: originalPath, err: errors.New("stat failed")}

	input := changeSwitchInput{
		deps: changeDeps{fs: fs},
		meta: meta,
		cfg:  cfg,
	}
	plan := &switchState{}

	err := disableExistingJar(input, map[string]models.ModInstall{changeModKey(mod): lockEntry}, meta.ModsFolderPath(cfg), mod, plan)
	assert.Error(t, err)
}

func TestDisableExistingJarDisabledExistsError(t *testing.T) {
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	lockEntry := models.ModInstall{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}

	originalPath := filepath.Join(meta.ModsFolderPath(cfg), "alpha.jar")
	disabledPath := filepath.Join(meta.ModsFolderPath(cfg), "alpha.jar.disabled")

	memfs := afero.NewMemMapFs()
	require.NoError(t, memfs.MkdirAll(meta.ModsFolderPath(cfg), 0o755))
	require.NoError(t, afero.WriteFile(memfs, originalPath, []byte("old"), 0o644))

	fs := statErrorFs{Fs: memfs, failPath: disabledPath, err: errors.New("stat failed")}

	input := changeSwitchInput{
		deps: changeDeps{fs: fs},
		meta: meta,
		cfg:  cfg,
	}
	plan := &switchState{}

	err := disableExistingJar(input, map[string]models.ModInstall{changeModKey(mod): lockEntry}, meta.ModsFolderPath(cfg), mod, plan)
	assert.Error(t, err)
}

func TestDisableExistingJarDisabledResolveError(t *testing.T) {
	root := t.TempDir()
	meta := config.NewMetadata(filepath.Join(root, "modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	lockEntry := models.ModInstall{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}

	modsDir := meta.ModsFolderPath(cfg)
	require.NoError(t, os.MkdirAll(modsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(modsDir, "alpha.jar"), []byte("old"), 0o644))

	disabledPath := filepath.Join(modsDir, "alpha.jar.disabled")
	fs := lstatErrorPathFs{OsFs: afero.NewOsFs().(*afero.OsFs), failPath: disabledPath, err: errors.New("lstat failed")}

	input := changeSwitchInput{
		deps: changeDeps{fs: fs},
		meta: meta,
		cfg:  cfg,
	}
	plan := &switchState{}

	err := disableExistingJar(input, map[string]models.ModInstall{changeModKey(mod): lockEntry}, modsDir, mod, plan)
	assert.Error(t, err)
}

func TestApplySwitchForItemSuccess(t *testing.T) {
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}
	state := newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	stagedPath := filepath.Join(meta.ModsFolderPath(cfg), ".mmm-staging", "downloads", "alpha.jar")
	require.NoError(t, fs.MkdirAll(filepath.Dir(stagedPath), 0o755))
	require.NoError(t, afero.WriteFile(fs, stagedPath, []byte("new"), 0o644))

	lockEntry := models.ModInstall{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}
	input := changeSwitchInput{
		ctx:         context.Background(),
		deps:        changeDeps{fs: fs, renameFile: func(fs afero.Fs, source, dest string) error { return fs.Rename(source, dest) }},
		meta:        meta,
		cfg:         cfg,
		items:       items,
		lockEntries: map[string]models.ModInstall{changeModKey(mod): lockEntry},
		stagedPaths: map[string]string{changeModKey(mod): stagedPath},
		changeState: state,
	}

	switchPlan := switchState{}
	err := applySwitchForItem(input, map[string]models.ModInstall{}, meta.ModsFolderPath(cfg), items[0], &switchPlan)
	assert.NoError(t, err)
	assert.Equal(t, changeSwitchSucceeded, state.snapshot()[0].SwitchStatus)
	assert.Len(t, switchPlan.installs, 1)
}

func TestFinalizeSwitchWriteLockError(t *testing.T) {
	input := changeSwitchInput{
		ctx: context.Background(),
		deps: changeDeps{fs: afero.NewMemMapFs(), writeLock: func(context.Context, afero.Fs, config.Metadata, []models.ModInstall) error {
			return errors.New("lock failed")
		}},
		meta:          config.NewMetadata("/cfg/modlist.json"),
		cfg:           models.ModsJSON{ModsFolder: "mods"},
		lock:          []models.ModInstall{},
		targetVersion: "1.20.1",
		items:         []changeItem{},
		lockEntries:   map[string]models.ModInstall{},
	}

	err := finalizeSwitch(input, switchState{})
	assert.Error(t, err)
}

func TestFinalizeSwitchWriteConfigError(t *testing.T) {
	lockCalls := 0
	input := changeSwitchInput{
		ctx: context.Background(),
		deps: changeDeps{
			fs: afero.NewMemMapFs(),
			writeLock: func(context.Context, afero.Fs, config.Metadata, []models.ModInstall) error {
				lockCalls++
				if lockCalls > 1 {
					return errors.New("restore failed")
				}
				return nil
			},
			writeConfig: func(context.Context, afero.Fs, config.Metadata, models.ModsJSON) error {
				return errors.New("config failed")
			},
		},
		meta:          config.NewMetadata("/cfg/modlist.json"),
		cfg:           models.ModsJSON{ModsFolder: "mods"},
		lock:          []models.ModInstall{},
		targetVersion: "1.20.1",
		items:         []changeItem{},
		lockEntries:   map[string]models.ModInstall{},
	}

	err := finalizeSwitch(input, switchState{})
	assert.Error(t, err)
}

func TestMoveExistingToBackupNoLockEntry(t *testing.T) {
	input := changeSwitchInput{
		deps:          changeDeps{fs: afero.NewMemMapFs()},
		meta:          config.NewMetadata("/cfg/modlist.json"),
		cfg:           models.ModsJSON{ModsFolder: "mods"},
		stagingBackup: "/staging/backup",
	}

	state := switchState{}
	err := moveExistingToBackup(input, map[string]models.ModInstall{}, input.meta.ModsFolderPath(input.cfg), models.Mod{ID: "alpha", Type: models.MODRINTH}, &state)
	assert.NoError(t, err)
	assert.Empty(t, state.backups)
}

func TestMoveExistingToBackupNormalizeError(t *testing.T) {
	input := changeSwitchInput{
		deps:          changeDeps{fs: afero.NewMemMapFs()},
		meta:          config.NewMetadata("/cfg/modlist.json"),
		cfg:           models.ModsJSON{ModsFolder: "mods"},
		stagingBackup: "/staging/backup",
	}

	state := switchState{}
	lockIndex := map[string]models.ModInstall{
		"modrinth:alpha": {ID: "alpha", Type: models.MODRINTH, FileName: "mods/alpha.jar"},
	}
	err := moveExistingToBackup(input, lockIndex, input.meta.ModsFolderPath(input.cfg), models.Mod{ID: "alpha", Type: models.MODRINTH}, &state)
	assert.Error(t, err)
}

func TestMoveExistingToBackupResolveError(t *testing.T) {
	meta := config.NewMetadata(filepath.Join(t.TempDir(), "missing", "modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	input := changeSwitchInput{
		deps:          changeDeps{fs: afero.NewOsFs()},
		meta:          meta,
		cfg:           cfg,
		stagingBackup: "/staging/backup",
	}

	state := switchState{}
	lockIndex := map[string]models.ModInstall{
		"modrinth:alpha": {ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"},
	}
	err := moveExistingToBackup(input, lockIndex, meta.ModsFolderPath(cfg), models.Mod{ID: "alpha", Type: models.MODRINTH}, &state)
	assert.Error(t, err)
}

func TestMoveExistingToBackupBackupResolveError(t *testing.T) {
	root := t.TempDir()
	meta := config.NewMetadata(filepath.Join(root, "modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	modsDir := meta.ModsFolderPath(cfg)
	require.NoError(t, os.MkdirAll(modsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(modsDir, "alpha.jar"), []byte("old"), 0o644))

	input := changeSwitchInput{
		deps:          changeDeps{fs: afero.NewOsFs(), renameFile: func(fs afero.Fs, source, dest string) error { return fs.Rename(source, dest) }},
		meta:          meta,
		cfg:           cfg,
		stagingBackup: filepath.Join(root, ".mmm-staging", "backup"),
	}

	state := switchState{}
	lockIndex := map[string]models.ModInstall{
		"modrinth:alpha": {ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"},
	}
	err := moveExistingToBackup(input, lockIndex, modsDir, models.Mod{ID: "alpha", Type: models.MODRINTH}, &state)
	assert.Error(t, err)
}

func TestMoveExistingToBackupExistsError(t *testing.T) {
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	originalPath := filepath.Join(meta.ModsFolderPath(cfg), "alpha.jar")
	fs := statErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: originalPath,
		err:      errors.New("stat failed"),
	}

	input := changeSwitchInput{
		deps:          changeDeps{fs: fs},
		meta:          meta,
		cfg:           cfg,
		stagingBackup: "/staging/backup",
	}

	state := switchState{}
	lockIndex := map[string]models.ModInstall{
		"modrinth:alpha": {ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"},
	}
	err := moveExistingToBackup(input, lockIndex, meta.ModsFolderPath(cfg), models.Mod{ID: "alpha", Type: models.MODRINTH}, &state)
	assert.Error(t, err)
}

func TestMoveExistingToBackupExistsFalse(t *testing.T) {
	fs := afero.NewMemMapFs()
	input := changeSwitchInput{
		deps:          changeDeps{fs: fs},
		meta:          config.NewMetadata("/cfg/modlist.json"),
		cfg:           models.ModsJSON{ModsFolder: "mods"},
		stagingBackup: "/staging/backup",
	}

	state := switchState{}
	lockIndex := map[string]models.ModInstall{
		"modrinth:alpha": {ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"},
	}
	err := moveExistingToBackup(input, lockIndex, input.meta.ModsFolderPath(input.cfg), models.Mod{ID: "alpha", Type: models.MODRINTH}, &state)
	assert.NoError(t, err)
	assert.Empty(t, state.backups)
}

func TestMoveExistingToBackupRenameError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "alpha.jar"), []byte("old"), 0o644))

	input := changeSwitchInput{
		deps:          changeDeps{fs: fs, renameFile: func(afero.Fs, string, string) error { return errors.New("rename failed") }},
		meta:          meta,
		cfg:           cfg,
		stagingBackup: "/staging/backup",
	}

	state := switchState{}
	lockIndex := map[string]models.ModInstall{
		"modrinth:alpha": {ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"},
	}
	err := moveExistingToBackup(input, lockIndex, meta.ModsFolderPath(cfg), models.Mod{ID: "alpha", Type: models.MODRINTH}, &state)
	assert.Error(t, err)
}

func TestMoveExistingToBackupSuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "alpha.jar"), []byte("old"), 0o644))

	input := changeSwitchInput{
		deps:          changeDeps{fs: fs, renameFile: func(fs afero.Fs, source, dest string) error { return fs.Rename(source, dest) }},
		meta:          meta,
		cfg:           cfg,
		stagingBackup: filepath.Join(meta.ModsFolderPath(cfg), ".mmm-staging", "backup"),
	}

	state := switchState{}
	lockIndex := map[string]models.ModInstall{
		"modrinth:alpha": {ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"},
	}
	err := moveExistingToBackup(input, lockIndex, meta.ModsFolderPath(cfg), models.Mod{ID: "alpha", Type: models.MODRINTH}, &state)
	assert.NoError(t, err)
	assert.Len(t, state.backups, 1)
}

func TestRollbackSwitchJoinErrors(t *testing.T) {
	fs := afero.NewMemMapFs()
	input := changeSwitchInput{
		deps: changeDeps{
			fs:         fs,
			removeFile: func(afero.Fs, string) error { return errors.New("remove failed") },
			renameFile: func(afero.Fs, string, string) error { return errors.New("rename failed") },
		},
	}
	state := switchState{
		installs: []switchInstall{{destination: "/mods/alpha.jar"}},
		backups:  []switchBackup{{original: "/mods/old.jar", backup: "/backup/old.jar"}},
	}

	err := rollbackSwitch(input, state, errors.New("original"))
	assert.Error(t, err)
}

func TestRollbackSwitchRestoresDisabled(t *testing.T) {
	fs := afero.NewMemMapFs()
	original := "/mods/alpha.jar"
	disabled := "/mods/alpha.jar.disabled"

	require.NoError(t, fs.MkdirAll("/mods", 0o755))
	require.NoError(t, afero.WriteFile(fs, disabled, []byte("old"), 0o644))

	input := changeSwitchInput{
		deps: changeDeps{
			fs:         fs,
			removeFile: func(afero.Fs, string) error { return nil },
			renameFile: func(fs afero.Fs, from, to string) error { return fs.Rename(from, to) },
		},
	}
	state := switchState{
		disableds: []switchDisabled{{original: original, disabled: disabled}},
	}

	err := rollbackSwitch(input, state, errors.New("original"))
	assert.Error(t, err)

	exists, err := afero.Exists(fs, original)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestRollbackSwitchDisabledRenameError(t *testing.T) {
	fs := afero.NewMemMapFs()
	original := "/mods/alpha.jar"
	disabled := "/mods/alpha.jar.disabled"

	require.NoError(t, fs.MkdirAll("/mods", 0o755))
	require.NoError(t, afero.WriteFile(fs, disabled, []byte("old"), 0o644))

	input := changeSwitchInput{
		deps: changeDeps{
			fs:         fs,
			removeFile: func(afero.Fs, string) error { return nil },
			renameFile: func(afero.Fs, string, string) error { return errors.New("rename failed") },
		},
	}
	state := switchState{
		disableds: []switchDisabled{{original: original, disabled: disabled}},
	}

	err := rollbackSwitch(input, state, errors.New("original"))
	assert.Error(t, err)
}

func TestRollbackSwitchNoop(t *testing.T) {
	input := changeSwitchInput{
		deps: changeDeps{
			removeFile: func(afero.Fs, string) error { return nil },
			renameFile: func(afero.Fs, string, string) error { return nil },
		},
	}
	err := rollbackSwitch(input, switchState{}, errors.New("original"))
	assert.Error(t, err)
}

func TestRollbackSwitchNoopWithoutError(t *testing.T) {
	input := changeSwitchInput{
		deps: changeDeps{
			removeFile: func(afero.Fs, string) error { return nil },
			renameFile: func(afero.Fs, string, string) error { return nil },
		},
	}

	err := rollbackSwitch(input, switchState{}, nil)
	assert.NoError(t, err)
}

func TestBuildNewLockSkipsMissingAndSkipped(t *testing.T) {
	items := []changeItem{
		{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}},
		{Mod: models.Mod{ID: "beta", Type: models.MODRINTH}, Skipped: true},
		{Mod: models.Mod{ID: "gamma", Type: models.MODRINTH}},
	}
	lockEntries := map[string]models.ModInstall{
		"modrinth:alpha": {ID: "alpha", Type: models.MODRINTH},
	}

	newLock := buildNewLock(items, lockEntries)
	require.Len(t, newLock, 1)
	assert.Equal(t, "alpha", newLock[0].ID)
}

func TestUpdateConfigForChangeRenames(t *testing.T) {
	cfg := models.ModsJSON{
		GameVersion: "1.19.4",
		Mods: []models.Mod{
			{ID: "alpha", Type: models.MODRINTH, Name: "Old"},
		},
	}
	resolved := map[string]string{"modrinth:alpha": "New"}

	updated := updateConfigForChange(cfg, resolved, "1.20.1", changeForcePolicyKeepConfig, nil)
	assert.Equal(t, "1.20.1", updated.GameVersion)
	assert.Equal(t, "New", updated.Mods[0].Name)
}

type lstatErrorPathFs struct {
	*afero.OsFs
	failPath string
	err      error
}

func (filesystem lstatErrorPathFs) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return nil, true, filesystem.err
	}
	return filesystem.OsFs.LstatIfPossible(name)
}
