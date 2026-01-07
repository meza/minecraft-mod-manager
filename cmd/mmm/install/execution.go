package install

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/modpath"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

type installExecutionErrorType int

const (
	installExecutionErrorNone installExecutionErrorType = iota
	installExecutionErrorDownload
	installExecutionErrorWriteLock
	installExecutionErrorWriteConfig
	installExecutionErrorCanceled
	installExecutionErrorUnknown
)

type installExecutionInput struct {
	meta       config.Metadata
	cfg        models.ModsJSON
	lock       []models.ModInstall
	deps       installDeps
	items      []installItem
	indexByKey map[string]int
}

type installExecutionOutcome struct {
	items      []installItem
	err        error
	errType    installExecutionErrorType
	lockPath   string
	configPath string
}

type installExecutionState struct {
	items      []installItem
	indexByKey map[string]int
	sender     httpclient.Sender
	mu         sync.Mutex
}

type installWriteState struct {
	meta       config.Metadata
	deps       installDeps
	cfg        models.ModsJSON
	lock       []models.ModInstall
	lockPath   string
	configPath string
	cancel     func()
	mu         sync.Mutex
	writeErr   error
	writeType  installExecutionErrorType
}

func runInstallExecution(ctx context.Context, input installExecutionInput, sender httpclient.Sender) installExecutionOutcome {
	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	state := newInstallExecutionState(input, sender)
	writeState := newInstallWriteState(input, cancel)

	configured, err := installConfiguredMods(installConfiguredInputs{
		ctx:        execCtx,
		meta:       input.meta,
		cfg:        input.cfg,
		lock:       input.lock,
		deps:       input.deps,
		colorize:   false,
		state:      state,
		writeState: writeState,
	})
	if err != nil {
		if writeErrType, writeStateErr := writeState.writeError(); writeStateErr != nil {
			state.abortRemaining()
			return installExecutionOutcome{
				items:      state.snapshot(),
				err:        writeStateErr,
				errType:    writeErrType,
				lockPath:   writeState.lockPath,
				configPath: writeState.configPath,
			}
		}
		if isContextCancellation(err) {
			state.abortRemaining()
			return installExecutionOutcome{items: state.snapshot(), err: err, errType: installExecutionErrorCanceled}
		}
		return installExecutionOutcome{items: state.snapshot(), err: err, errType: installExecutionErrorUnknown}
	}
	if configured.failedCount > 0 {
		return installExecutionOutcome{
			items:   state.snapshot(),
			err:     errInstallFailures,
			errType: installExecutionErrorDownload,
		}
	}

	return installExecutionOutcome{items: state.snapshot(), err: nil, errType: installExecutionErrorNone}
}

func newInstallExecutionState(input installExecutionInput, sender httpclient.Sender) *installExecutionState {
	return &installExecutionState{
		items:      cloneInstallItems(input.items),
		indexByKey: input.indexByKey,
		sender:     sender,
	}
}

func newInstallWriteState(input installExecutionInput, cancel func()) *installWriteState {
	cfgCopy := input.cfg
	if len(input.cfg.Mods) > 0 {
		cfgCopy.Mods = append([]models.Mod(nil), input.cfg.Mods...)
	}
	lockCopy := append([]models.ModInstall(nil), input.lock...)
	return &installWriteState{
		meta:       input.meta,
		deps:       input.deps,
		cfg:        cfgCopy,
		lock:       lockCopy,
		lockPath:   input.meta.LockPath(),
		configPath: input.meta.ConfigPath,
		cancel:     cancel,
	}
}

func (state *installWriteState) recordDownloadSuccess(ctx context.Context, modIndex int, outcome modInstallOutcome) error {
	if outcome.lockEntry == nil && strings.TrimSpace(outcome.newName) == "" {
		return nil
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	if strings.TrimSpace(outcome.newName) != "" && modIndex >= 0 && modIndex < len(state.cfg.Mods) {
		state.cfg.Mods[modIndex].Name = outcome.newName
	}
	if outcome.lockEntry != nil {
		state.lock = append(state.lock, *outcome.lockEntry)
		orderLockEntriesForConfig(state.lock, state.cfg)
	}

	if state.writeErr != nil {
		return state.writeErr
	}

	if outcome.lockEntry == nil {
		return nil
	} else if err := config.WriteLock(ctx, state.deps.fs, state.meta, state.lock); err != nil {
		state.setWriteError(err, installExecutionErrorWriteLock)
		return err
	}

	if strings.TrimSpace(outcome.newName) == "" {
		return nil
	} else if err := config.WriteConfig(ctx, state.deps.fs, state.meta, state.cfg); err != nil {
		state.setWriteError(err, installExecutionErrorWriteConfig)
		return err
	}
	return nil
}

func orderLockEntriesForConfig(lock []models.ModInstall, cfg models.ModsJSON) {
	if len(lock) == 0 {
		return
	}
	order := make(map[string]int, len(cfg.Mods))
	for index, mod := range cfg.Mods {
		order[installModKey(mod)] = index
	}
	sort.SliceStable(lock, func(leftIndex int, rightIndex int) bool {
		leftKey := lockEntryKey(lock[leftIndex])
		rightKey := lockEntryKey(lock[rightIndex])
		leftOrder, leftOk := order[leftKey]
		rightOrder, rightOk := order[rightKey]
		if leftOk && rightOk {
			return leftOrder < rightOrder
		}
		if leftOk != rightOk {
			return leftOk
		}
		leftPlatform := strings.ToLower(string(lock[leftIndex].Type))
		rightPlatform := strings.ToLower(string(lock[rightIndex].Type))
		if leftPlatform == rightPlatform {
			return strings.ToLower(lock[leftIndex].ID) < strings.ToLower(lock[rightIndex].ID)
		}
		return leftPlatform < rightPlatform
	})
}

func lockEntryKey(entry models.ModInstall) string {
	return string(entry.Type) + ":" + entry.ID
}

func (state *installWriteState) writeError() (installExecutionErrorType, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.writeErr == nil {
		return installExecutionErrorNone, nil
	}
	return state.writeType, state.writeErr
}

func (state *installWriteState) setWriteError(err error, errType installExecutionErrorType) {
	state.writeErr = err
	state.writeType = errType
	if state.cancel != nil {
		state.cancel()
	}
}

func cloneInstallItems(items []installItem) []installItem {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]installItem, len(items))
	copy(cloned, items)
	return cloned
}

func (state *installExecutionState) snapshot() []installItem {
	state.mu.Lock()
	defer state.mu.Unlock()
	return cloneInstallItems(state.items)
}

func (state *installExecutionState) updateItem(key string, update func(*installItem)) {
	state.mu.Lock()
	defer state.mu.Unlock()
	index, ok := state.indexByKey[key]
	if !ok {
		return
	}
	item := state.items[index]
	update(&item)
	state.items[index] = item
}

func (state *installExecutionState) send(msg tea.Msg) {
	if state.sender == nil {
		return
	}
	state.sender.Send(msg)
}

func (state *installExecutionState) setDownloading(key string, progress httpclient.DownloadProgressMsg) {
	state.updateItem(key, func(item *installItem) {
		item.Status = installItemDownloading
		if item.Progress == nil {
			item.Progress = &installProgress{}
		}
		item.Progress.ratio = progress.Ratio
		item.Progress.downloaded = progress.Downloaded
		item.Progress.total = progress.Total
	})
	state.send(installItemProgressMsg{key: key, progress: progress})
}

func (state *installExecutionState) setDownloadProgressError(key string, err error) {
	if err == nil {
		return
	}
	state.updateItem(key, func(item *installItem) {
		if strings.TrimSpace(item.FailureReason) == "" {
			item.FailureReason = err.Error()
		}
	})
	state.send(installItemProgressErrMsg{key: key, err: err})
}

func (state *installExecutionState) setSuccess(key string, displayName string) {
	state.updateItem(key, func(item *installItem) {
		item.Status = installItemSuccess
		item.Progress = nil
		item.FailureReason = ""
		if strings.TrimSpace(displayName) != "" {
			item.DisplayName = displayName
		}
	})
	state.send(installItemSuccessMsg{key: key, displayName: displayName})
}

func (state *installExecutionState) setFailure(key string, reason string) {
	state.updateItem(key, func(item *installItem) {
		item.Status = installItemFailed
		item.Progress = nil
		item.FailureReason = reason
	})
	state.send(installItemFailureMsg{key: key, reason: reason})
}

func (state *installExecutionState) abortRemaining() {
	abortedKeys := make([]string, 0)

	state.mu.Lock()
	for index, item := range state.items {
		if isTerminalInstallStatus(item.Status) {
			continue
		}
		item.Status = installItemAborted
		item.Progress = nil
		item.FailureReason = ""
		state.items[index] = item
		abortedKeys = append(abortedKeys, installModKey(item.Mod))
	}
	state.mu.Unlock()

	for _, key := range abortedKeys {
		state.send(installItemAbortedMsg{key: key})
	}
}

type installProgressSender struct {
	key   string
	state *installExecutionState
}

func (sender installProgressSender) Send(msg tea.Msg) {
	switch typed := msg.(type) {
	case httpclient.DownloadProgressMsg:
		sender.state.setDownloading(sender.key, typed)
	case httpclient.DownloadProgressErrMsg:
		sender.state.setDownloadProgressError(sender.key, typed.Err)
	default:
		return
	}
}

func resolveInstallDownloadClient(deps installDeps) httpclient.Doer {
	return platform.PreferredDownloadClient(deps.clients)
}

func isContextCancellation(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func installFailureReason(err error, modName string) (string, bool) {
	var missingHash modinstall.MissingHashError
	if errors.As(err, &missingHash) {
		return i18n.T("cmd.install.error.missing_hash_lock", &i18n.Tvars{
			Data: &i18n.TData{"name": modName},
		}), true
	}

	var hashMismatch modinstall.HashMismatchError
	if errors.As(err, &hashMismatch) {
		return i18n.T("cmd.install.error.hash_mismatch", &i18n.Tvars{
			Data: &i18n.TData{"name": modName},
		}), true
	}

	var outsideRoot modpath.OutsideRootError
	if errors.As(err, &outsideRoot) {
		return i18n.T("cmd.install.error.symlink_outside_mods", &i18n.Tvars{
			Data: &i18n.TData{
				"name": modName,
				"path": outsideRoot.ResolvedPath,
				"root": outsideRoot.Root,
			},
		}), true
	}
	return "", false
}
