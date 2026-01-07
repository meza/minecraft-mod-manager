package install

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

func TestInstallProgressSenderUpdatesState(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}},
	}
	items, indexByKey := buildInstallItems(cfg)
	state := &installExecutionState{items: items, indexByKey: indexByKey}
	key := installModKey(cfg.Mods[0])

	sender := installProgressSender{key: key, state: state}
	sender.Send(httpclient.DownloadProgressMsg{Downloaded: 5, Total: 10, Ratio: 0.5})

	item := state.items[indexByKey[key]]
	assert.Equal(t, installItemDownloading, item.Status)
	assert.NotNil(t, item.Progress)
	assert.Equal(t, int64(5), item.Progress.downloaded)

	sender.Send(httpclient.DownloadProgressErrMsg{Err: errors.New("boom")})
	item = state.items[indexByKey[key]]
	assert.Contains(t, item.FailureReason, "boom")

	sender.Send(tea.KeyMsg{})
}

func TestInstallProgressErrorSkipsWhenNil(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}},
	}
	items, indexByKey := buildInstallItems(cfg)
	state := &installExecutionState{items: items, indexByKey: indexByKey}
	key := installModKey(cfg.Mods[0])

	state.setDownloadProgressError(key, nil)
	item := state.items[indexByKey[key]]
	assert.Empty(t, item.FailureReason)
}

func TestInstallExecutionStateUpdateItemSkipsUnknownKey(t *testing.T) {
	state := &installExecutionState{items: nil, indexByKey: map[string]int{}}
	state.updateItem("missing", func(item *installItem) {
		item.Status = installItemSuccess
	})
}

func TestInstallExecutionStateAbortRemainingMarksPendingAndDownloading(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
			{ID: "beta", Name: "Beta", Type: models.CURSEFORGE},
			{ID: "gamma", Name: "Gamma", Type: models.MODRINTH},
		},
	}
	items, indexByKey := buildInstallItems(cfg)
	recorder := &messageRecorder{}
	state := &installExecutionState{items: items, indexByKey: indexByKey, sender: recorder}

	state.updateItem(installModKey(cfg.Mods[0]), func(item *installItem) {
		item.Status = installItemSuccess
	})
	state.updateItem(installModKey(cfg.Mods[1]), func(item *installItem) {
		item.Status = installItemPending
	})
	state.updateItem(installModKey(cfg.Mods[2]), func(item *installItem) {
		item.Status = installItemDownloading
		item.Progress = &installProgress{ratio: 0.5, downloaded: 10, total: 20}
	})

	state.abortRemaining()

	assert.Equal(t, installItemSuccess, state.items[indexByKey[installModKey(cfg.Mods[0])]].Status)
	assert.Equal(t, installItemAborted, state.items[indexByKey[installModKey(cfg.Mods[1])]].Status)
	assert.Nil(t, state.items[indexByKey[installModKey(cfg.Mods[1])]].Progress)
	assert.Equal(t, installItemAborted, state.items[indexByKey[installModKey(cfg.Mods[2])]].Status)
	assert.Nil(t, state.items[indexByKey[installModKey(cfg.Mods[2])]].Progress)

	abortedKeys := recorder.abortedKeys()
	assert.ElementsMatch(t, []string{
		installModKey(cfg.Mods[1]),
		installModKey(cfg.Mods[2]),
	}, abortedKeys)
}

func TestInstallWriteStateRecordDownloadSuccessSkipsWhenNoChanges(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, baseFs.MkdirAll(meta.Dir(), 0755))

	input := installExecutionInput{
		meta: meta,
		cfg:  models.ModsJSON{},
		lock: nil,
		deps: installDeps{fs: baseFs},
	}
	state := newInstallWriteState(input, nil)
	assert.NoError(t, state.recordDownloadSuccess(context.Background(), 0, modInstallOutcome{}))

	errType, err := state.writeError()
	assert.NoError(t, err)
	assert.Equal(t, installExecutionErrorNone, errType)
}

func TestInstallWriteStateRecordDownloadSuccessWriteLockErrorCancels(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, baseFs.MkdirAll(meta.Dir(), 0755))

	filesystem := renameErrorFs{Fs: baseFs, failNew: meta.LockPath(), err: errors.New("rename failed")}
	canceled := false
	input := installExecutionInput{
		meta: meta,
		cfg:  models.ModsJSON{},
		lock: nil,
		deps: installDeps{fs: filesystem},
	}
	state := newInstallWriteState(input, func() { canceled = true })
	outcome := modInstallOutcome{
		lockEntry: &models.ModInstall{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"},
	}

	err := state.recordDownloadSuccess(context.Background(), 0, outcome)
	assert.Error(t, err)

	errType, err := state.writeError()
	assert.Error(t, err)
	assert.Equal(t, installExecutionErrorWriteLock, errType)
	assert.True(t, canceled)
}

func TestInstallWriteStateRecordDownloadSuccessWriteConfigErrorCancels(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, baseFs.MkdirAll(meta.Dir(), 0755))

	filesystem := renameErrorFs{Fs: baseFs, failNew: meta.ConfigPath, err: errors.New("rename failed")}
	canceled := false
	input := installExecutionInput{
		meta: meta,
		cfg: models.ModsJSON{
			Mods: []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}},
		},
		lock: nil,
		deps: installDeps{fs: filesystem},
	}
	state := newInstallWriteState(input, func() { canceled = true })
	outcome := modInstallOutcome{
		newName:   "Alpha Remote",
		lockEntry: &models.ModInstall{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"},
	}

	err := state.recordDownloadSuccess(context.Background(), 0, outcome)
	assert.Error(t, err)

	errType, err := state.writeError()
	assert.Error(t, err)
	assert.Equal(t, installExecutionErrorWriteConfig, errType)
	assert.True(t, canceled)
}

func TestInstallWriteStateRecordDownloadSuccessWritesLockAndConfig(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Mods: []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}},
	}
	assert.NoError(t, baseFs.MkdirAll(meta.Dir(), 0755))

	input := installExecutionInput{
		meta: meta,
		cfg:  cfg,
		lock: nil,
		deps: installDeps{fs: baseFs},
	}
	state := newInstallWriteState(input, nil)
	outcome := modInstallOutcome{
		newName:   "Alpha Remote",
		lockEntry: &models.ModInstall{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"},
	}

	assert.NoError(t, state.recordDownloadSuccess(context.Background(), 0, outcome))

	errType, err := state.writeError()
	assert.NoError(t, err)
	assert.Equal(t, installExecutionErrorNone, errType)

	updatedConfig, configErr := config.ReadConfig(context.Background(), baseFs, meta)
	assert.NoError(t, configErr)
	assert.Equal(t, "Alpha Remote", updatedConfig.Mods[0].Name)

	updatedLock, lockErr := config.ReadLock(context.Background(), baseFs, meta)
	assert.NoError(t, lockErr)
	if assert.Len(t, updatedLock, 1) {
		assert.Equal(t, "alpha.jar", updatedLock[0].FileName)
	}
}

func TestInstallWriteStateRecordDownloadSuccessWritesLockOnly(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, baseFs.MkdirAll(meta.Dir(), 0755))

	input := installExecutionInput{
		meta: meta,
		cfg:  models.ModsJSON{},
		lock: nil,
		deps: installDeps{fs: baseFs},
	}
	state := newInstallWriteState(input, nil)
	outcome := modInstallOutcome{
		lockEntry: &models.ModInstall{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"},
	}

	assert.NoError(t, state.recordDownloadSuccess(context.Background(), 0, outcome))

	errType, err := state.writeError()
	assert.NoError(t, err)
	assert.Equal(t, installExecutionErrorNone, errType)

	updatedLock, lockErr := config.ReadLock(context.Background(), baseFs, meta)
	assert.NoError(t, lockErr)
	if assert.Len(t, updatedLock, 1) {
		assert.Equal(t, "alpha.jar", updatedLock[0].FileName)
	}
}

func TestInstallWriteStateOrdersLockEntriesByConfig(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
			{ID: "beta", Name: "Beta", Type: models.CURSEFORGE},
		},
	}
	assert.NoError(t, baseFs.MkdirAll(meta.Dir(), 0755))

	input := installExecutionInput{
		meta: meta,
		cfg:  cfg,
		lock: nil,
		deps: installDeps{fs: baseFs},
	}
	state := newInstallWriteState(input, nil)

	assert.NoError(t, state.recordDownloadSuccess(context.Background(), 1, modInstallOutcome{
		lockEntry: &models.ModInstall{ID: "beta", Type: models.CURSEFORGE, FileName: "beta.jar"},
	}))
	assert.NoError(t, state.recordDownloadSuccess(context.Background(), 0, modInstallOutcome{
		lockEntry: &models.ModInstall{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"},
	}))

	updatedLock, lockErr := config.ReadLock(context.Background(), baseFs, meta)
	assert.NoError(t, lockErr)
	if assert.Len(t, updatedLock, 2) {
		assert.Equal(t, "alpha", updatedLock[0].ID)
		assert.Equal(t, "beta", updatedLock[1].ID)
	}
}

func TestOrderLockEntriesForConfigSkipsEmptySlice(t *testing.T) {
	orderLockEntriesForConfig(nil, models.ModsJSON{})
}

func TestOrderLockEntriesForConfigPrefersKnownMods(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}},
	}
	lock := []models.ModInstall{
		{ID: "beta", Type: models.CURSEFORGE},
		{ID: "alpha", Type: models.MODRINTH},
	}

	orderLockEntriesForConfig(lock, cfg)

	if assert.Len(t, lock, 2) {
		assert.Equal(t, "alpha", lock[0].ID)
		assert.Equal(t, "beta", lock[1].ID)
	}
}

func TestOrderLockEntriesForConfigUsesPlatformAndID(t *testing.T) {
	lock := []models.ModInstall{
		{ID: "beta", Type: models.MODRINTH},
		{ID: "alpha", Type: models.MODRINTH},
		{ID: "gamma", Type: models.CURSEFORGE},
	}

	orderLockEntriesForConfig(lock, models.ModsJSON{})

	if assert.Len(t, lock, 3) {
		assert.Equal(t, models.CURSEFORGE, lock[0].Type)
		assert.Equal(t, "gamma", lock[0].ID)
		assert.Equal(t, "alpha", lock[1].ID)
		assert.Equal(t, "beta", lock[2].ID)
	}
}

type messageRecorder struct {
	msgs []tea.Msg
}

func (recorder *messageRecorder) Send(msg tea.Msg) {
	recorder.msgs = append(recorder.msgs, msg)
}

func (recorder *messageRecorder) abortedKeys() []string {
	keys := make([]string, 0, len(recorder.msgs))
	for _, msg := range recorder.msgs {
		typed, ok := msg.(installItemAbortedMsg)
		if !ok {
			continue
		}
		keys = append(keys, typed.key)
	}
	return keys
}

func TestInstallWriteStateRecordDownloadSuccessSkipsWhenAlreadyFailed(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, baseFs.MkdirAll(meta.Dir(), 0755))

	input := installExecutionInput{
		meta: meta,
		cfg:  models.ModsJSON{},
		lock: nil,
		deps: installDeps{fs: baseFs},
	}
	state := newInstallWriteState(input, nil)
	state.writeErr = errors.New("write failed")
	state.writeType = installExecutionErrorWriteLock

	outcome := modInstallOutcome{
		lockEntry: &models.ModInstall{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"},
	}
	assert.Error(t, state.recordDownloadSuccess(context.Background(), 0, outcome))

	errType, err := state.writeError()
	assert.Error(t, err)
	assert.Equal(t, installExecutionErrorWriteLock, errType)
}

func TestInstallWriteStateRecordDownloadSuccessUpdatesNameWithoutLockEntry(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Mods: []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}},
	}
	assert.NoError(t, baseFs.MkdirAll(meta.Dir(), 0755))

	input := installExecutionInput{
		meta: meta,
		cfg:  cfg,
		lock: nil,
		deps: installDeps{fs: baseFs},
	}
	state := newInstallWriteState(input, nil)
	outcome := modInstallOutcome{newName: "Alpha Remote"}

	assert.NoError(t, state.recordDownloadSuccess(context.Background(), 0, outcome))

	errType, err := state.writeError()
	assert.NoError(t, err)
	assert.Equal(t, installExecutionErrorNone, errType)

	assert.Equal(t, "Alpha Remote", state.cfg.Mods[0].Name)
	_, lockErr := baseFs.Stat(meta.LockPath())
	assert.Error(t, lockErr)
}
