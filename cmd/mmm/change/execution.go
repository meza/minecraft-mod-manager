package change

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"golang.org/x/time/rate"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/modpath"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

var errCompatibilityFailed = errors.New("compatibility check failed")

type changeExecutionInput struct {
	meta          config.Metadata
	cfg           models.ModsJSON
	lock          []models.ModInstall
	targetVersion string
	force         bool
	deps          changeDeps
	items         []changeItem
	indexByKey    map[string]int
}

type changeExecutionState struct {
	items                 []changeItem
	indexByKey            map[string]int
	sender                httpclient.Sender
	compatFailureDetected bool
	mu                    sync.Mutex
}

type changeStagingPaths struct {
	root      string
	downloads string
	backup    string
}

type downloadPhaseResult struct {
	lockEntries    map[string]models.ModInstall
	stagedPaths    map[string]string
	resolvedNames  map[string]string
	compatFailed   bool
	downloadFailed bool
	downloadErr    error
}

type downloadPhaseState struct {
	mu             sync.Mutex
	lockEntries    map[string]models.ModInstall
	stagedPaths    map[string]string
	resolvedNames  map[string]string
	compatFailed   bool
	downloadFailed bool
	downloadErr    error
}

func newDownloadPhaseState() *downloadPhaseState {
	return &downloadPhaseState{
		lockEntries:   make(map[string]models.ModInstall),
		stagedPaths:   make(map[string]string),
		resolvedNames: make(map[string]string),
	}
}

func (state *downloadPhaseState) markCompatFailed() bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.compatFailed {
		return false
	}
	state.compatFailed = true
	return true
}

func (state *downloadPhaseState) markDownloadFailed(err error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.downloadFailed = true
	if state.downloadErr == nil && err != nil {
		state.downloadErr = err
	}
}

func (state *downloadPhaseState) compatFailureDetected() bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.compatFailed
}

func (state *downloadPhaseState) recordDownloadSuccess(key string, entry models.ModInstall, stagedPath string, resolvedName string) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.lockEntries[key] = entry
	state.stagedPaths[key] = stagedPath
	state.resolvedNames[key] = resolvedName
}

func (state *downloadPhaseState) snapshot() downloadPhaseResult {
	state.mu.Lock()
	defer state.mu.Unlock()
	return downloadPhaseResult{
		lockEntries:    state.lockEntries,
		stagedPaths:    state.stagedPaths,
		resolvedNames:  state.resolvedNames,
		compatFailed:   state.compatFailed,
		downloadFailed: state.downloadFailed,
		downloadErr:    state.downloadErr,
	}
}

type downloadWorkerInput struct {
	compatCtx        context.Context
	downloadCtx      context.Context
	cancelAll        context.CancelFunc
	cancelDownloads  context.CancelFunc
	input            changeExecutionInput
	state            *changeExecutionState
	phaseState       *downloadPhaseState
	installer        *modinstall.Installer
	downloadClient   httpclient.Doer
	stagingDownloads string
	compatSlots      chan struct{}
}

func runChangeExecution(ctx context.Context, sender httpclient.Sender, input changeExecutionInput) changeOutcome {
	state := newChangeExecutionState(input, sender)
	staging := buildChangeStagingPaths(input.meta, input.cfg)

	if err := resetStaging(input.deps, staging.root, staging.downloads); err != nil {
		return changeOutcome{
			Stage:         changeStageDownloadFailed,
			TargetVersion: input.targetVersion,
			Items:         state.snapshot(),
			Err:           err,
		}
	}

	execCtx, cancelAll := context.WithCancel(ctx)
	defer cancelAll()
	downloadCtx, cancelDownloads := context.WithCancel(execCtx)
	defer cancelDownloads()

	downloadResult := runDownloadPhase(execCtx, downloadCtx, cancelAll, cancelDownloads, input, state, staging.downloads)
	if outcome, handled := outcomeFromDownloadPhase(execCtx, input, state, downloadResult, staging.root); handled {
		return outcome
	}

	return runSwitchPhase(execCtx, input, state, downloadResult, staging)
}

func newChangeExecutionState(input changeExecutionInput, sender httpclient.Sender) *changeExecutionState {
	items := cloneChangeItems(input.items)
	return &changeExecutionState{
		items:      items,
		indexByKey: input.indexByKey,
		sender:     sender,
	}
}

func buildChangeStagingPaths(meta config.Metadata, cfg models.ModsJSON) changeStagingPaths {
	root := filepath.Join(meta.ModsFolderPath(cfg), ".mmm-staging")
	return changeStagingPaths{
		root:      root,
		downloads: filepath.Join(root, "downloads"),
		backup:    filepath.Join(root, "backup"),
	}
}

func runDownloadPhase(
	compatCtx context.Context,
	downloadCtx context.Context,
	cancelAll context.CancelFunc,
	cancelDownloads context.CancelFunc,
	input changeExecutionInput,
	state *changeExecutionState,
	stagingDownloads string,
) downloadPhaseResult {
	downloadClient := resolveChangeDownloadClient(input.deps)
	installer := modinstall.NewInstaller(input.deps.fs, modinstall.Downloader(input.deps.downloader))
	phaseState := newDownloadPhaseState()
	compatSlots := make(chan struct{}, compatSlotCapacity(input.deps.limiter))
	workerInput := downloadWorkerInput{
		compatCtx:        compatCtx,
		downloadCtx:      downloadCtx,
		cancelAll:        cancelAll,
		cancelDownloads:  cancelDownloads,
		input:            input,
		state:            state,
		phaseState:       phaseState,
		installer:        installer,
		downloadClient:   downloadClient,
		stagingDownloads: stagingDownloads,
		compatSlots:      compatSlots,
	}

	var wg sync.WaitGroup
	for _, item := range state.snapshot() {
		mod := item.Mod
		key := changeModKey(mod)
		wg.Add(1)
		go func(mod models.Mod, key string) {
			defer wg.Done()
			runDownloadWorker(workerInput, mod, key)
		}(mod, key)
	}

	wg.Wait()
	return phaseState.snapshot()
}

func runDownloadWorker(worker downloadWorkerInput, mod models.Mod, key string) {
	request, handled := prepareDownloadRequest(worker, mod, key)
	if handled {
		return
	}
	performDownload(worker, mod, key, request)
}

type downloadRequest struct {
	remote     platform.RemoteMod
	stagedPath string
}

func prepareDownloadRequest(worker downloadWorkerInput, mod models.Mod, key string) (downloadRequest, bool) {
	if worker.compatCtx.Err() != nil {
		return downloadRequest{}, true
	}

	if !worker.acquireCompatSlot() {
		return downloadRequest{}, true
	}
	worker.state.setCompatChecking(key)
	defer worker.releaseCompatSlot()

	remote, handled := fetchCompatRemote(worker, mod, key)
	if handled {
		return downloadRequest{}, true
	}
	if worker.compatCtx.Err() != nil || worker.phaseState.compatFailureDetected() {
		return downloadRequest{}, true
	}

	return buildDownloadRequest(worker, key, remote)
}

func fetchCompatRemote(worker downloadWorkerInput, mod models.Mod, key string) (platform.RemoteMod, bool) {
	remote, fetchErr := fetchRemoteModForChange(worker.compatCtx, mod, worker.input.cfg, worker.input.targetVersion, worker.input.deps)
	if fetchErr == nil {
		worker.state.setCompatSupported(key, remote.Name)
		return remote, false
	}
	if worker.compatCtx.Err() != nil {
		return platform.RemoteMod{}, true
	}

	skipped := worker.input.force
	worker.state.setCompatUnsupported(key, skipped)
	if skipped {
		return platform.RemoteMod{}, true
	}

	if worker.phaseState.markCompatFailed() {
		worker.state.markCompatFailure()
	}
	worker.cancelDownloads()
	return platform.RemoteMod{}, true
}

func buildDownloadRequest(worker downloadWorkerInput, key string, remote platform.RemoteMod) (downloadRequest, bool) {
	remote, normalizeErr := normalizeRemoteForChange(remote)
	if normalizeErr != nil {
		recordDownloadFailure(worker, key, normalizeErr)
		return downloadRequest{}, true
	}

	stagedPath, resolveErr := resolveStagingPath(worker.input.deps, worker.input.meta, worker.input.cfg, worker.stagingDownloads, remote.FileName)
	if resolveErr != nil {
		recordDownloadFailure(worker, key, resolveErr)
		return downloadRequest{}, true
	}

	return downloadRequest{
		remote:     remote,
		stagedPath: stagedPath,
	}, false
}

func (worker downloadWorkerInput) acquireCompatSlot() bool {
	if worker.compatSlots == nil {
		return true
	}
	select {
	case worker.compatSlots <- struct{}{}:
		return true
	case <-worker.compatCtx.Done():
		return false
	}
}

func (worker downloadWorkerInput) releaseCompatSlot() {
	if worker.compatSlots == nil {
		return
	}
	select {
	case <-worker.compatSlots:
	default:
	}
}

func compatSlotCapacity(limiter *rate.Limiter) int {
	if limiter == nil {
		return httpclient.DefaultRateLimitBurst
	}
	if limiter.Burst() < 1 {
		return 1
	}
	return limiter.Burst()
}

func performDownload(worker downloadWorkerInput, mod models.Mod, key string, request downloadRequest) {
	if worker.downloadCtx.Err() != nil {
		return
	}
	progressSender := changeProgressSender{
		key:   key,
		state: worker.state,
	}

	downloadErr := worker.installer.DownloadAndVerify(worker.downloadCtx, request.remote.DownloadURL, request.stagedPath, request.remote.Hash, worker.downloadClient, progressSender)
	if downloadErr != nil {
		if worker.downloadCtx.Err() != nil && worker.phaseState.compatFailureDetected() {
			return
		}
		recordDownloadFailure(worker, key, downloadErr)
		return
	}

	lockEntry := buildLockEntry(mod, request.remote)
	worker.phaseState.recordDownloadSuccess(key, lockEntry, request.stagedPath, request.remote.Name)
	worker.state.setDownloadSuccess(key, request.remote.Name)
}

func recordDownloadFailure(worker downloadWorkerInput, key string, err error) {
	if worker.downloadCtx.Err() != nil && worker.phaseState.compatFailureDetected() {
		return
	}
	worker.state.setDownloadFailure(key, err)
	worker.phaseState.markDownloadFailed(err)
	worker.cancelAll()
}

func outcomeFromDownloadPhase(
	ctx context.Context,
	input changeExecutionInput,
	state *changeExecutionState,
	result downloadPhaseResult,
	stagingRoot string,
) (changeOutcome, bool) {
	items := state.snapshot()
	if result.compatFailed && !input.force {
		cleanupErr := cleanupStaging(input.deps, stagingRoot)
		return changeOutcome{
			Stage:         changeStageCompatibilityFailed,
			TargetVersion: input.targetVersion,
			Items:         items,
			Err:           errors.Join(errCompatibilityFailed, cleanupErr),
		}, true
	}

	if result.downloadFailed {
		cleanupErr := cleanupStaging(input.deps, stagingRoot)
		return changeOutcome{
			Stage:         changeStageDownloadFailed,
			TargetVersion: input.targetVersion,
			Items:         items,
			Err:           errors.Join(result.downloadErr, cleanupErr),
		}, true
	}

	if len(items) == 0 {
		switchErr := updateConfigWithoutMods(ctx, input, stagingRoot)
		if switchErr != nil {
			return changeOutcome{
				Stage:         changeStageSwitchFailed,
				TargetVersion: input.targetVersion,
				Items:         items,
				Err:           switchErr,
			}, true
		}
		return changeOutcome{
			Stage:         changeStageSuccess,
			TargetVersion: input.targetVersion,
			Items:         items,
		}, true
	}

	return changeOutcome{}, false
}

func runSwitchPhase(
	ctx context.Context,
	input changeExecutionInput,
	state *changeExecutionState,
	result downloadPhaseResult,
	staging changeStagingPaths,
) changeOutcome {
	switchItems := state.snapshot()
	state.send(changeSwitchingStartedMsg{})

	switchErr := switchToDownloadedMods(changeSwitchInput{
		ctx:           ctx,
		deps:          input.deps,
		meta:          input.meta,
		cfg:           input.cfg,
		lock:          input.lock,
		targetVersion: input.targetVersion,
		items:         switchItems,
		lockEntries:   result.lockEntries,
		stagedPaths:   result.stagedPaths,
		resolvedNames: result.resolvedNames,
		stagingBackup: staging.backup,
		changeState:   state,
	})
	outcomeItems := state.snapshot()
	if switchErr != nil {
		cleanupErr := cleanupStaging(input.deps, staging.root)
		return changeOutcome{
			Stage:         changeStageSwitchFailed,
			TargetVersion: input.targetVersion,
			Items:         outcomeItems,
			Err:           errors.Join(switchErr, cleanupErr),
		}
	}

	if cleanupErr := cleanupStaging(input.deps, staging.root); cleanupErr != nil {
		return changeOutcome{
			Stage:         changeStageSwitchFailed,
			TargetVersion: input.targetVersion,
			Items:         outcomeItems,
			Err:           cleanupErr,
		}
	}

	return changeOutcome{
		Stage:         changeStageSuccess,
		TargetVersion: input.targetVersion,
		Items:         outcomeItems,
	}
}

func updateConfigWithoutMods(ctx context.Context, input changeExecutionInput, stagingRoot string) error {
	modsDir := input.meta.ModsFolderPath(input.cfg)

	for _, lockEntry := range input.lock {
		normalizedFileName, err := modfilename.Normalize(lockEntry.FileName)
		if err != nil {
			return err
		}
		path := filepath.Join(modsDir, normalizedFileName)
		resolvedPath, err := modpath.ResolveWritablePath(input.deps.fs, modsDir, path)
		if err != nil {
			return err
		}
		exists, err := afero.Exists(input.deps.fs, resolvedPath)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		if err := input.deps.removeFile(input.deps.fs, resolvedPath); err != nil {
			return err
		}
	}

	newConfig := input.cfg
	newConfig.GameVersion = input.targetVersion

	if err := input.deps.writeLock(ctx, input.deps.fs, input.meta, []models.ModInstall{}); err != nil {
		return err
	}
	if err := input.deps.writeConfig(ctx, input.deps.fs, input.meta, newConfig); err != nil {
		restoreErr := input.deps.writeLock(ctx, input.deps.fs, input.meta, input.lock)
		return errors.Join(err, restoreErr)
	}

	if err := input.deps.removeAll(input.deps.fs, stagingRoot); err != nil {
		return err
	}

	return nil
}

func resolveChangeDownloadClient(deps changeDeps) httpclient.Doer {
	if deps.downloadClient != nil {
		return deps.downloadClient
	}
	return platform.PreferredDownloadClient(deps.clients)
}

type changeProgressSender struct {
	key   string
	state *changeExecutionState
}

func (sender changeProgressSender) Send(msg tea.Msg) {
	switch typed := msg.(type) {
	case httpclient.DownloadProgressMsg:
		sender.state.setDownloadProgress(sender.key, typed)
	case httpclient.DownloadProgressErrMsg:
		sender.state.setDownloadProgressError(sender.key, typed.Err)
	default:
		return
	}
}

func (state *changeExecutionState) send(msg tea.Msg) {
	if state.sender == nil {
		return
	}
	state.sender.Send(msg)
}

func (state *changeExecutionState) markCompatFailure() {
	shouldSend := false
	state.mu.Lock()
	if !state.compatFailureDetected {
		state.compatFailureDetected = true
		shouldSend = true
	}
	state.mu.Unlock()
	if shouldSend {
		state.send(changeCompatFailureMsg{})
	}
}

func (state *changeExecutionState) snapshot() []changeItem {
	state.mu.Lock()
	defer state.mu.Unlock()
	return cloneChangeItems(state.items)
}

func (state *changeExecutionState) updateItem(key string, update func(*changeItem)) {
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

func (state *changeExecutionState) setCompatSupported(key string, resolvedName string) {
	state.updateItem(key, func(item *changeItem) {
		item.CompatStatus = changeCompatSupported
		if item.DownloadStatus == changeDownloadPending {
			item.DownloadStatus = changeDownloadQueued
		}
		if strings.TrimSpace(resolvedName) != "" {
			item.DisplayName = resolvedName
		}
	})
	state.send(changeCompatResultMsg{
		key:          key,
		supported:    true,
		skipped:      false,
		resolvedName: resolvedName,
	})
}

func (state *changeExecutionState) setCompatUnsupported(key string, skipped bool) {
	state.updateItem(key, func(item *changeItem) {
		item.CompatStatus = changeCompatUnsupported
		item.Skipped = skipped
	})
	state.send(changeCompatResultMsg{
		key:          key,
		supported:    false,
		skipped:      skipped,
		resolvedName: "",
	})
}

func (state *changeExecutionState) setCompatChecking(key string) {
	state.updateItem(key, func(item *changeItem) {
		item.CompatStatus = changeCompatChecking
	})
	state.send(changeCompatCheckingMsg{key: key})
}

func (state *changeExecutionState) setDownloadProgress(key string, progress httpclient.DownloadProgressMsg) {
	state.updateItem(key, func(item *changeItem) {
		item.DownloadStatus = changeDownloadInProgress
		item.Download = changeDownloadProgress{
			ratio:      progress.Ratio,
			downloaded: progress.Downloaded,
			total:      progress.Total,
		}
	})
	state.send(changeDownloadProgressMsg{
		key:      key,
		progress: progress,
	})
}

func (state *changeExecutionState) setDownloadProgressError(key string, err error) {
	if err == nil {
		return
	}
	state.updateItem(key, func(item *changeItem) {
		if item.ErrorReason == "" {
			item.ErrorReason = err.Error()
		}
	})
	state.send(changeDownloadProgressErrMsg{key: key, err: err})
}

func (state *changeExecutionState) setDownloadSuccess(key string, resolvedName string) {
	state.updateItem(key, func(item *changeItem) {
		item.DownloadStatus = changeDownloadSucceeded
		if strings.TrimSpace(resolvedName) != "" {
			item.DisplayName = resolvedName
		}
	})
	state.send(changeDownloadFinishedMsg{key: key, resolvedName: resolvedName})
}

func (state *changeExecutionState) setDownloadFailure(key string, err error) {
	state.updateItem(key, func(item *changeItem) {
		item.DownloadStatus = changeDownloadFailed
		if err != nil {
			item.ErrorReason = err.Error()
		}
	})
	state.send(changeDownloadFailedMsg{key: key, err: err})
}

func (state *changeExecutionState) setSwitchStarted(key string) {
	state.updateItem(key, func(item *changeItem) {
		item.SwitchStatus = changeSwitchInProgress
	})
	state.send(changeSwitchStartedMsg{key: key})
}

func (state *changeExecutionState) setSwitchSuccess(key string) {
	state.updateItem(key, func(item *changeItem) {
		item.SwitchStatus = changeSwitchSucceeded
	})
	state.send(changeSwitchFinishedMsg{key: key})
}

func (state *changeExecutionState) setSwitchFailure(key string, err error) {
	state.updateItem(key, func(item *changeItem) {
		item.SwitchStatus = changeSwitchFailed
		if err != nil {
			item.ErrorReason = err.Error()
		}
	})
	state.send(changeSwitchFailedMsg{key: key, err: err})
}

func (state *changeExecutionState) setSwitchSkipped(key string) {
	state.updateItem(key, func(item *changeItem) {
		item.SwitchStatus = changeSwitchSkipped
	})
	state.send(changeSwitchSkippedMsg{key: key})
}

func resetStaging(deps changeDeps, stagingRoot string, downloadsPath string) error {
	if err := deps.removeAll(deps.fs, stagingRoot); err != nil {
		return err
	}
	if err := deps.mkdirAll(deps.fs, downloadsPath, 0o755); err != nil {
		return err
	}
	return nil
}

func cleanupStaging(deps changeDeps, stagingRoot string) error {
	if err := deps.removeAll(deps.fs, stagingRoot); err != nil {
		if deps.logger == nil {
			return err
		}
		if logErr := deps.logger.Debug(fmt.Sprintf("cleanup failed for %s: %v", stagingRoot, err)); logErr != nil {
			return logErr
		}
	}
	return nil
}

func fetchRemoteModForChange(ctx context.Context, mod models.Mod, cfg models.ModsJSON, targetVersion string, deps changeDeps) (platform.RemoteMod, error) {
	fetchOpts := platform.FetchOptions{
		AllowedReleaseTypes: models.EffectiveAllowedReleaseTypes(mod, cfg),
		GameVersion:         targetVersion,
		Loader:              cfg.Loader,
		AllowFallback:       mod.AllowVersionFallback != nil && *mod.AllowVersionFallback,
	}
	if mod.Version != nil && strings.TrimSpace(*mod.Version) != "" {
		fetchOpts.FixedVersion = *mod.Version
	}
	return deps.fetchMod(ctx, mod.Type, mod.ID, fetchOpts, deps.clients)
}

func normalizeRemoteForChange(remote platform.RemoteMod) (platform.RemoteMod, error) {
	normalizedFileName, err := modfilename.Normalize(remote.FileName)
	if err != nil {
		return platform.RemoteMod{}, err
	}
	remote.FileName = normalizedFileName
	if strings.TrimSpace(remote.Hash) == "" {
		return platform.RemoteMod{}, modinstall.MissingHashError{FileName: remote.FileName}
	}
	return remote, nil
}

func resolveStagingPath(deps changeDeps, meta config.Metadata, cfg models.ModsJSON, stagingRoot string, fileName string) (string, error) {
	destination := filepath.Join(stagingRoot, fileName)
	return modpath.ResolveWritablePath(deps.fs, meta.ModsFolderPath(cfg), destination)
}

func buildLockEntry(mod models.Mod, remote platform.RemoteMod) models.ModInstall {
	return models.ModInstall{
		Type:        mod.Type,
		ID:          mod.ID,
		Name:        remote.Name,
		FileName:    remote.FileName,
		ReleasedOn:  remote.ReleaseDate,
		Hash:        remote.Hash,
		DownloadURL: remote.DownloadURL,
	}
}

type changeSwitchInput struct {
	ctx           context.Context
	deps          changeDeps
	meta          config.Metadata
	cfg           models.ModsJSON
	lock          []models.ModInstall
	targetVersion string
	items         []changeItem
	lockEntries   map[string]models.ModInstall
	stagedPaths   map[string]string
	resolvedNames map[string]string
	stagingBackup string
	changeState   *changeExecutionState
}

type switchBackup struct {
	original string
	backup   string
}

type switchInstall struct {
	destination string
}

type switchState struct {
	backups  []switchBackup
	installs []switchInstall
}

func switchToDownloadedMods(input changeSwitchInput) error {
	if err := input.deps.mkdirAll(input.deps.fs, input.stagingBackup, 0o755); err != nil {
		return err
	}

	lockIndex := indexLockByMod(input.lock)
	modsDir := input.meta.ModsFolderPath(input.cfg)
	switchPlan := switchState{}

	for _, item := range input.items {
		if err := applySwitchForItem(input, lockIndex, modsDir, item, &switchPlan); err != nil {
			return err
		}
	}

	return finalizeSwitch(input, switchPlan)
}

func applySwitchForItem(
	input changeSwitchInput,
	lockIndex map[string]models.ModInstall,
	modsDir string,
	item changeItem,
	switchPlan *switchState,
) error {
	key := changeModKey(item.Mod)
	input.changeState.setSwitchStarted(key)

	if err := input.ctx.Err(); err != nil {
		return err
	}

	if err := moveExistingToBackup(input, lockIndex, modsDir, item.Mod, switchPlan); err != nil {
		input.changeState.setSwitchFailure(key, err)
		return rollbackSwitch(input, *switchPlan, err)
	}

	if item.Skipped {
		input.changeState.setSwitchSkipped(key)
		return nil
	}

	lockEntry, ok := input.lockEntries[key]
	if !ok {
		err := errors.New("missing lock entry for download")
		input.changeState.setSwitchFailure(key, err)
		return rollbackSwitch(input, *switchPlan, err)
	}

	stagedPath, ok := input.stagedPaths[key]
	if !ok {
		err := errors.New("missing staged download for mod")
		input.changeState.setSwitchFailure(key, err)
		return rollbackSwitch(input, *switchPlan, err)
	}

	destination := filepath.Join(modsDir, lockEntry.FileName)
	resolvedDestination, err := modpath.ResolveWritablePath(input.deps.fs, modsDir, destination)
	if err != nil {
		input.changeState.setSwitchFailure(key, err)
		return rollbackSwitch(input, *switchPlan, err)
	}

	if err := input.deps.renameFile(input.deps.fs, stagedPath, resolvedDestination); err != nil {
		input.changeState.setSwitchFailure(key, err)
		return rollbackSwitch(input, *switchPlan, err)
	}

	switchPlan.installs = append(switchPlan.installs, switchInstall{destination: resolvedDestination})
	input.changeState.setSwitchSuccess(key)
	return nil
}

func finalizeSwitch(input changeSwitchInput, switchPlan switchState) error {
	newLock := buildNewLock(input.items, input.lockEntries)
	newConfig := updateConfigForChange(input.cfg, input.resolvedNames, input.targetVersion)

	if err := input.deps.writeLock(input.ctx, input.deps.fs, input.meta, newLock); err != nil {
		rollbackErr := rollbackSwitch(input, switchPlan, err)
		restoreErr := input.deps.writeLock(input.ctx, input.deps.fs, input.meta, input.lock)
		return errors.Join(rollbackErr, restoreErr)
	}

	if err := input.deps.writeConfig(input.ctx, input.deps.fs, input.meta, newConfig); err != nil {
		rollbackErr := rollbackSwitch(input, switchPlan, err)
		lockErr := input.deps.writeLock(input.ctx, input.deps.fs, input.meta, input.lock)
		configErr := input.deps.writeConfig(input.ctx, input.deps.fs, input.meta, input.cfg)
		return errors.Join(rollbackErr, lockErr, configErr)
	}

	return nil
}

func moveExistingToBackup(
	input changeSwitchInput,
	lockIndex map[string]models.ModInstall,
	modsDir string,
	mod models.Mod,
	switchState *switchState,
) error {
	lockEntry, ok := lockIndex[changeModKey(mod)]
	if !ok {
		return nil
	}

	normalizedFileName, err := modfilename.Normalize(lockEntry.FileName)
	if err != nil {
		return err
	}

	originalPath := filepath.Join(modsDir, normalizedFileName)
	resolvedOriginal, err := modpath.ResolveWritablePath(input.deps.fs, modsDir, originalPath)
	if err != nil {
		return err
	}

	exists, err := afero.Exists(input.deps.fs, resolvedOriginal)
	if err != nil || !exists {
		return err
	}

	backupPath := filepath.Join(input.stagingBackup, normalizedFileName)
	resolvedBackup, err := modpath.ResolveWritablePath(input.deps.fs, modsDir, backupPath)
	if err != nil {
		return err
	}

	if err := input.deps.renameFile(input.deps.fs, resolvedOriginal, resolvedBackup); err != nil {
		return err
	}

	switchState.backups = append(switchState.backups, switchBackup{
		original: resolvedOriginal,
		backup:   resolvedBackup,
	})
	return nil
}

func rollbackSwitch(input changeSwitchInput, state switchState, err error) error {
	var rollbackErr error

	for _, install := range state.installs {
		if removeErr := input.deps.removeFile(input.deps.fs, install.destination); removeErr != nil {
			rollbackErr = errors.Join(rollbackErr, removeErr)
		}
	}

	for _, backup := range state.backups {
		if renameErr := input.deps.renameFile(input.deps.fs, backup.backup, backup.original); renameErr != nil {
			rollbackErr = errors.Join(rollbackErr, renameErr)
		}
	}

	if rollbackErr != nil {
		return errors.Join(err, rollbackErr)
	}
	return err
}

func buildNewLock(items []changeItem, lockEntries map[string]models.ModInstall) []models.ModInstall {
	entries := make([]models.ModInstall, 0)
	for _, item := range items {
		if item.Skipped {
			continue
		}
		lockEntry, ok := lockEntries[changeModKey(item.Mod)]
		if !ok {
			continue
		}
		entries = append(entries, lockEntry)
	}
	return entries
}

func updateConfigForChange(cfg models.ModsJSON, resolvedNames map[string]string, targetVersion string) models.ModsJSON {
	updated := cfg
	updated.GameVersion = targetVersion

	for i := range updated.Mods {
		mod := updated.Mods[i]
		name, ok := resolvedNames[changeModKey(mod)]
		if ok && strings.TrimSpace(name) != "" {
			updated.Mods[i].Name = name
		}
	}

	return updated
}
