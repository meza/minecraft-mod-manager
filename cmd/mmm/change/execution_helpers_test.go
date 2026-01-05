package change

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

type recordingSender struct {
	msgs []tea.Msg
}

func (sender *recordingSender) Send(msg tea.Msg) {
	sender.msgs = append(sender.msgs, msg)
}

type errorWriter struct {
	err error
}

func (writer errorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

func newDownloadWorkerInputForTest(
	compatCtx context.Context,
	downloadCtx context.Context,
	input changeExecutionInput,
	state *changeExecutionState,
	phaseState *downloadPhaseState,
) downloadWorkerInput {
	return downloadWorkerInput{
		compatCtx:       compatCtx,
		downloadCtx:     downloadCtx,
		cancelAll:       func() {},
		cancelDownloads: func() {},
		input:           input,
		state:           state,
		phaseState:      phaseState,
	}
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

func TestChangeProgressSenderSendUpdatesProgress(t *testing.T) {
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}
	captured := &recordingSender{}

	state := newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, captured)
	sender := changeProgressSender{key: changeModKey(mod), state: state}

	sender.Send(httpclient.DownloadProgressMsg{Ratio: 0.5, Downloaded: 10, Total: 20})

	updated := state.snapshot()
	assert.Equal(t, changeDownloadInProgress, updated[0].DownloadStatus)
	assert.Equal(t, float64(0.5), updated[0].Download.ratio)
	assert.Len(t, captured.msgs, 1)
	_, ok := captured.msgs[0].(changeDownloadProgressMsg)
	assert.True(t, ok)
}

func TestChangeProgressSenderSendDownloadError(t *testing.T) {
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}
	captured := &recordingSender{}

	state := newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, captured)
	sender := changeProgressSender{key: changeModKey(mod), state: state}

	sender.Send(httpclient.DownloadProgressErrMsg{Err: errors.New("boom")})

	updated := state.snapshot()
	assert.Equal(t, "boom", updated[0].ErrorReason)
	assert.Len(t, captured.msgs, 1)
	_, ok := captured.msgs[0].(changeDownloadProgressErrMsg)
	assert.True(t, ok)
}

func TestChangeProgressSenderSendIgnoresOtherMsg(t *testing.T) {
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}
	captured := &recordingSender{}

	state := newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, captured)
	sender := changeProgressSender{key: changeModKey(mod), state: state}

	sender.Send(tea.Quit())

	updated := state.snapshot()
	assert.Empty(t, updated[0].ErrorReason)
	assert.Empty(t, captured.msgs)
}

func TestChangeExecutionStateUpdateItemMissingKey(t *testing.T) {
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod, DisplayName: "Alpha"}}
	index := map[string]int{changeModKey(mod): 0}

	state := newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil)
	state.updateItem("missing", func(item *changeItem) {
		item.DisplayName = "Updated"
	})

	updated := state.snapshot()
	assert.Equal(t, "Alpha", updated[0].DisplayName)
}

func TestChangeExecutionStateSetDownloadProgressErrorIgnoresNil(t *testing.T) {
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}

	state := newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil)
	state.setDownloadProgressError(changeModKey(mod), nil)

	updated := state.snapshot()
	assert.Empty(t, updated[0].ErrorReason)
}

func TestChangeExecutionStateSetSwitchSkipped(t *testing.T) {
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}
	captured := &recordingSender{}

	state := newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, captured)
	state.setSwitchSkipped(changeModKey(mod))

	updated := state.snapshot()
	assert.Equal(t, changeSwitchSkipped, updated[0].SwitchStatus)
	assert.Len(t, captured.msgs, 1)
	_, ok := captured.msgs[0].(changeSwitchSkippedMsg)
	assert.True(t, ok)
}

func TestResetStagingReturnsErrorOnRemove(t *testing.T) {
	deps := changeDeps{
		fs:        afero.NewMemMapFs(),
		removeAll: func(afero.Fs, string) error { return errors.New("remove failed") },
		mkdirAll:  func(afero.Fs, string, os.FileMode) error { return nil },
	}

	err := resetStaging(deps, "/staging", "/staging/downloads")
	assert.Error(t, err)
}

func TestResetStagingReturnsErrorOnMkdir(t *testing.T) {
	deps := changeDeps{
		fs:        afero.NewMemMapFs(),
		removeAll: func(afero.Fs, string) error { return nil },
		mkdirAll:  func(afero.Fs, string, os.FileMode) error { return errors.New("mkdir failed") },
	}

	err := resetStaging(deps, "/staging", "/staging/downloads")
	assert.Error(t, err)
}

func TestCleanupStagingReturnsErrorWhenLoggerNil(t *testing.T) {
	deps := changeDeps{
		fs:        afero.NewMemMapFs(),
		removeAll: func(afero.Fs, string) error { return errors.New("remove failed") },
	}

	err := cleanupStaging(deps, "/staging")
	assert.Error(t, err)
}

func TestCleanupStagingReturnsLoggerError(t *testing.T) {
	log := logger.New(errorWriter{err: errors.New("log failed")}, errorWriter{err: errors.New("log failed")}, false, true)
	deps := changeDeps{
		fs:        afero.NewMemMapFs(),
		logger:    log,
		removeAll: func(afero.Fs, string) error { return errors.New("remove failed") },
	}

	err := cleanupStaging(deps, "/staging")
	assert.Error(t, err)
}

func TestCleanupStagingSwallowsErrorWhenLoggerQuiet(t *testing.T) {
	log := logger.New(errorWriter{err: errors.New("log failed")}, errorWriter{err: errors.New("log failed")}, false, false)
	deps := changeDeps{
		fs:        afero.NewMemMapFs(),
		logger:    log,
		removeAll: func(afero.Fs, string) error { return errors.New("remove failed") },
	}

	err := cleanupStaging(deps, "/staging")
	assert.NoError(t, err)
}

type stubDoer struct{}

func (stubDoer) Do(*http.Request) (*http.Response, error) {
	return nil, errors.New("stub")
}

func TestResolveChangeDownloadClientPrefersExplicitClient(t *testing.T) {
	explicit := stubDoer{}
	deps := changeDeps{
		downloadClient: explicit,
		clients:        platform.Clients{},
	}

	client := resolveChangeDownloadClient(deps)
	assert.Equal(t, explicit, client)
}

func TestResolveChangeDownloadClientFallsBackToPreferredClient(t *testing.T) {
	fallback := stubDoer{}
	deps := changeDeps{
		clients: platform.Clients{Modrinth: fallback},
	}

	client := resolveChangeDownloadClient(deps)
	assert.Equal(t, fallback, client)
}

func TestCleanupStagingNoError(t *testing.T) {
	deps := changeDeps{
		fs:        afero.NewMemMapFs(),
		removeAll: func(afero.Fs, string) error { return nil },
	}

	err := cleanupStaging(deps, "/staging")
	assert.NoError(t, err)
}

func TestFetchRemoteModForChangeUsesFixedVersion(t *testing.T) {
	version := "1.20.1"
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH, Version: &version}
	cfg := models.ModsJSON{Loader: models.FABRIC}

	var captured platform.FetchOptions
	deps := changeDeps{
		fetchMod: func(_ context.Context, _ models.Platform, _ string, opts platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			captured = opts
			return platform.RemoteMod{}, nil
		},
	}

	_, err := fetchRemoteModForChange(context.Background(), mod, cfg, "1.20.4", deps)
	assert.NoError(t, err)
	assert.Equal(t, version, captured.FixedVersion)
}

func TestNormalizeRemoteForChangeMissingHash(t *testing.T) {
	_, err := normalizeRemoteForChange(platform.RemoteMod{FileName: "alpha.jar"})
	var missingHash modinstall.MissingHashError
	assert.ErrorAs(t, err, &missingHash)
}

func TestNormalizeRemoteForChangeInvalidFilename(t *testing.T) {
	_, err := normalizeRemoteForChange(platform.RemoteMod{FileName: "mods/alpha.jar", Hash: "abc"})
	assert.Error(t, err)
}

func TestPrepareDownloadRequestCanceled(t *testing.T) {
	compatCtx, cancel := context.WithCancel(context.Background())
	cancel()
	downloadCtx := context.Background()

	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}

	worker := newDownloadWorkerInputForTest(
		compatCtx,
		downloadCtx,
		changeExecutionInput{cfg: models.ModsJSON{}, deps: changeDeps{}},
		newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil),
		newDownloadPhaseState(),
	)

	_, handled := prepareDownloadRequest(worker, mod, changeModKey(mod))
	assert.True(t, handled)
}

func TestPrepareDownloadRequestFetchErrorForce(t *testing.T) {
	compatCtx, cancelCompat := context.WithCancel(context.Background())
	defer cancelCompat()
	downloadCtx := context.Background()

	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}
	deps := changeDeps{
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, errors.New("unsupported")
		},
	}

	worker := newDownloadWorkerInputForTest(
		compatCtx,
		downloadCtx,
		changeExecutionInput{cfg: models.ModsJSON{}, deps: deps, force: true},
		newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil),
		newDownloadPhaseState(),
	)

	_, handled := prepareDownloadRequest(worker, mod, changeModKey(mod))
	assert.True(t, handled)
	assert.Equal(t, changeCompatUnsupported, worker.state.snapshot()[0].CompatStatus)
	assert.True(t, worker.state.snapshot()[0].Skipped)
}

func TestPrepareDownloadRequestFetchErrorNoForce(t *testing.T) {
	compatCtx, cancelCompat := context.WithCancel(context.Background())
	defer cancelCompat()
	downloadCtx := context.Background()

	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}
	deps := changeDeps{
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, errors.New("unsupported")
		},
	}

	worker := newDownloadWorkerInputForTest(
		compatCtx,
		downloadCtx,
		changeExecutionInput{cfg: models.ModsJSON{}, deps: deps, force: false},
		newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil),
		newDownloadPhaseState(),
	)

	_, handled := prepareDownloadRequest(worker, mod, changeModKey(mod))
	assert.True(t, handled)
	assert.Equal(t, changeCompatUnsupported, worker.state.snapshot()[0].CompatStatus)
	assert.False(t, worker.state.snapshot()[0].Skipped)
	assert.NoError(t, compatCtx.Err())
}

func TestPrepareDownloadRequestFetchErrorCanceledContext(t *testing.T) {
	compatCtx, cancelCompat := context.WithCancel(context.Background())
	downloadCtx := context.Background()

	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}
	deps := changeDeps{
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			cancelCompat()
			return platform.RemoteMod{}, errors.New("unsupported")
		},
	}

	worker := newDownloadWorkerInputForTest(
		compatCtx,
		downloadCtx,
		changeExecutionInput{cfg: models.ModsJSON{}, deps: deps, force: false},
		newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil),
		newDownloadPhaseState(),
	)

	_, handled := prepareDownloadRequest(worker, mod, changeModKey(mod))
	assert.True(t, handled)
	assert.Equal(t, changeCompatChecking, worker.state.snapshot()[0].CompatStatus)
}

func TestPrepareDownloadRequestReleasesCompatSlotOnFetchError(t *testing.T) {
	compatCtx := context.Background()
	downloadCtx := context.Background()

	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}
	deps := changeDeps{
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, errors.New("unsupported")
		},
	}

	compatSlots := make(chan struct{}, 1)
	worker := newDownloadWorkerInputForTest(
		compatCtx,
		downloadCtx,
		changeExecutionInput{cfg: models.ModsJSON{}, deps: deps, force: true},
		newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil),
		newDownloadPhaseState(),
	)
	worker.compatSlots = compatSlots

	_, handled := prepareDownloadRequest(worker, mod, changeModKey(mod))
	assert.True(t, handled)
	assert.Equal(t, 0, len(compatSlots))
}

func TestAcquireCompatSlotReturnsFalseWhenCanceled(t *testing.T) {
	compatCtx, cancelCompat := context.WithCancel(context.Background())
	cancelCompat()

	compatSlots := make(chan struct{}, 1)
	compatSlots <- struct{}{}

	worker := downloadWorkerInput{
		compatCtx:   compatCtx,
		downloadCtx: context.Background(),
		compatSlots: compatSlots,
	}

	assert.False(t, worker.acquireCompatSlot())
}

type closedContext struct{}

func (closedContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (closedContext) Done() <-chan struct{} {
	channel := make(chan struct{})
	close(channel)
	return channel
}
func (closedContext) Err() error        { return nil }
func (closedContext) Value(key any) any { return nil }

func TestPrepareDownloadRequestCompatSlotUnavailable(t *testing.T) {
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}
	deps := changeDeps{
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{Name: "Alpha", FileName: "alpha.jar", Hash: "abc"}, nil
		},
	}

	compatSlots := make(chan struct{}, 1)
	compatSlots <- struct{}{}

	worker := downloadWorkerInput{
		compatCtx:   closedContext{},
		downloadCtx: context.Background(),
		input:       changeExecutionInput{cfg: models.ModsJSON{}, deps: deps},
		state:       newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil),
		phaseState:  newDownloadPhaseState(),
		compatSlots: compatSlots,
	}

	_, handled := prepareDownloadRequest(worker, mod, changeModKey(mod))
	assert.True(t, handled)
	assert.Equal(t, changeCompatPending, worker.state.snapshot()[0].CompatStatus)
}

func TestCompatSlotCapacityUsesLimiterBurst(t *testing.T) {
	limiter := rate.NewLimiter(rate.Every(time.Second), 3)
	assert.Equal(t, 3, compatSlotCapacity(limiter))
}

func TestCompatSlotCapacityUsesDefaultBurstWhenNil(t *testing.T) {
	assert.Equal(t, httpclient.DefaultRateLimitBurst, compatSlotCapacity(nil))
}

func TestCompatSlotCapacityUsesMinimumOne(t *testing.T) {
	limiter := rate.NewLimiter(rate.Every(time.Second), 0)
	assert.Equal(t, 1, compatSlotCapacity(limiter))
}

func TestDownloadPhaseStateMarkCompatFailedIdempotent(t *testing.T) {
	state := newDownloadPhaseState()
	assert.True(t, state.markCompatFailed())
	assert.False(t, state.markCompatFailed())
	assert.True(t, state.compatFailureDetected())
}

func TestPrepareDownloadRequestCanceledAfterFetch(t *testing.T) {
	compatCtx, cancelCompat := context.WithCancel(context.Background())
	downloadCtx := context.Background()

	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}
	deps := changeDeps{
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			cancelCompat()
			return platform.RemoteMod{Name: "Alpha", FileName: "alpha.jar", Hash: "abc"}, nil
		},
	}

	worker := newDownloadWorkerInputForTest(
		compatCtx,
		downloadCtx,
		changeExecutionInput{cfg: models.ModsJSON{}, deps: deps},
		newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil),
		newDownloadPhaseState(),
	)

	_, handled := prepareDownloadRequest(worker, mod, changeModKey(mod))
	assert.True(t, handled)
	assert.Equal(t, changeCompatSupported, worker.state.snapshot()[0].CompatStatus)
}

func TestPrepareDownloadRequestCompatFailed(t *testing.T) {
	compatCtx, cancelCompat := context.WithCancel(context.Background())
	defer cancelCompat()
	downloadCtx := context.Background()

	meta := config.NewMetadata(filepath.Join(t.TempDir(), "modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}
	deps := changeDeps{
		fs: afero.NewMemMapFs(),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{Name: "Alpha", FileName: "alpha.jar", Hash: "abc"}, nil
		},
	}

	phaseState := newDownloadPhaseState()
	phaseState.markCompatFailed()
	worker := newDownloadWorkerInputForTest(
		compatCtx,
		downloadCtx,
		changeExecutionInput{meta: meta, cfg: cfg, deps: deps, force: false},
		newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil),
		phaseState,
	)
	worker.stagingDownloads = filepath.Join(meta.ModsFolderPath(cfg), ".mmm-staging", "downloads")

	require.NoError(t, deps.fs.MkdirAll(meta.ModsFolderPath(cfg), 0o755))

	_, handled := prepareDownloadRequest(worker, mod, changeModKey(mod))
	assert.True(t, handled)
	assert.Equal(t, changeCompatSupported, worker.state.snapshot()[0].CompatStatus)
}

func TestPrepareDownloadRequestNormalizeError(t *testing.T) {
	compatCtx, cancelCompat := context.WithCancel(context.Background())
	defer cancelCompat()
	downloadCtx := context.Background()

	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}
	deps := changeDeps{
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{FileName: "alpha.jar"}, nil
		},
	}

	worker := newDownloadWorkerInputForTest(
		compatCtx,
		downloadCtx,
		changeExecutionInput{cfg: models.ModsJSON{}, deps: deps},
		newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil),
		newDownloadPhaseState(),
	)

	_, handled := prepareDownloadRequest(worker, mod, changeModKey(mod))
	assert.True(t, handled)
	assert.Equal(t, changeDownloadFailed, worker.state.snapshot()[0].DownloadStatus)
}

func TestPrepareDownloadRequestResolveStagingError(t *testing.T) {
	compatCtx, cancelCompat := context.WithCancel(context.Background())
	defer cancelCompat()
	downloadCtx := context.Background()

	meta := config.NewMetadata(filepath.Join(t.TempDir(), "missing", "modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}
	deps := changeDeps{
		fs: afero.NewOsFs(),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{FileName: "alpha.jar", Hash: "abc"}, nil
		},
	}

	worker := newDownloadWorkerInputForTest(
		compatCtx,
		downloadCtx,
		changeExecutionInput{meta: meta, cfg: cfg, deps: deps},
		newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil),
		newDownloadPhaseState(),
	)
	worker.stagingDownloads = filepath.Join(meta.ModsFolderPath(cfg), ".mmm-staging", "downloads")

	_, handled := prepareDownloadRequest(worker, mod, changeModKey(mod))
	assert.True(t, handled)
	assert.Equal(t, changeDownloadFailed, worker.state.snapshot()[0].DownloadStatus)
}

func TestPerformDownloadRecordsFailure(t *testing.T) {
	compatCtx, cancelCompat := context.WithCancel(context.Background())
	defer cancelCompat()
	downloadCtx, cancelDownloads := context.WithCancel(context.Background())
	defer cancelDownloads()

	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}

	installer := modinstall.NewInstaller(afero.NewMemMapFs(), modinstall.Downloader(func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
		return errors.New("download failed")
	}))

	worker := newDownloadWorkerInputForTest(
		compatCtx,
		downloadCtx,
		changeExecutionInput{deps: changeDeps{}},
		newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil),
		newDownloadPhaseState(),
	)
	worker.installer = installer
	worker.cancelAll = func() {}

	performDownload(worker, mod, changeModKey(mod), downloadRequest{remote: platform.RemoteMod{Hash: "abc"}})
	assert.Equal(t, changeDownloadFailed, worker.state.snapshot()[0].DownloadStatus)
	assert.True(t, worker.phaseState.snapshot().downloadFailed)
}

func TestPerformDownloadIgnoresCanceledContext(t *testing.T) {
	compatCtx := context.Background()
	downloadCtx, cancelDownloads := context.WithCancel(context.Background())
	cancelDownloads()

	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}

	installer := modinstall.NewInstaller(afero.NewMemMapFs(), modinstall.Downloader(func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
		return errors.New("download failed")
	}))

	worker := newDownloadWorkerInputForTest(
		compatCtx,
		downloadCtx,
		changeExecutionInput{deps: changeDeps{}},
		newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil),
		newDownloadPhaseState(),
	)
	worker.installer = installer

	performDownload(worker, mod, changeModKey(mod), downloadRequest{remote: platform.RemoteMod{Hash: "abc"}})
	assert.Equal(t, changeDownloadPending, worker.state.snapshot()[0].DownloadStatus)
	assert.False(t, worker.phaseState.snapshot().downloadFailed)
}

func TestRecordDownloadFailureNoopWhenCanceled(t *testing.T) {
	compatCtx := context.Background()
	downloadCtx, cancelDownloads := context.WithCancel(context.Background())
	cancelDownloads()

	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod}}
	index := map[string]int{changeModKey(mod): 0}

	phaseState := newDownloadPhaseState()
	phaseState.markCompatFailed()
	worker := newDownloadWorkerInputForTest(
		compatCtx,
		downloadCtx,
		changeExecutionInput{deps: changeDeps{}},
		newChangeExecutionState(changeExecutionInput{items: items, indexByKey: index}, nil),
		phaseState,
	)

	recordDownloadFailure(worker, changeModKey(mod), errors.New("boom"))
	assert.Empty(t, worker.state.snapshot()[0].ErrorReason)
}

func TestOutcomeFromDownloadPhaseCompatFailed(t *testing.T) {
	input := changeExecutionInput{targetVersion: "1.20.1", force: false, deps: changeDeps{removeAll: func(afero.Fs, string) error { return nil }, fs: afero.NewMemMapFs()}}
	state := newChangeExecutionState(changeExecutionInput{items: []changeItem{}, indexByKey: map[string]int{}}, nil)
	result := downloadPhaseResult{compatFailed: true}

	outcome, handled := outcomeFromDownloadPhase(context.Background(), input, state, result, "/staging")
	assert.True(t, handled)
	assert.Equal(t, changeStageCompatibilityFailed, outcome.Stage)
	assert.ErrorIs(t, outcome.Err, errCompatibilityFailed)
}

func TestOutcomeFromDownloadPhaseDownloadFailed(t *testing.T) {
	input := changeExecutionInput{targetVersion: "1.20.1", deps: changeDeps{removeAll: func(afero.Fs, string) error { return nil }, fs: afero.NewMemMapFs()}}
	state := newChangeExecutionState(changeExecutionInput{items: []changeItem{}, indexByKey: map[string]int{}}, nil)
	result := downloadPhaseResult{downloadFailed: true, downloadErr: errors.New("boom")}

	outcome, handled := outcomeFromDownloadPhase(context.Background(), input, state, result, "/staging")
	assert.True(t, handled)
	assert.Equal(t, changeStageDownloadFailed, outcome.Stage)
	assert.Error(t, outcome.Err)
}

func TestOutcomeFromDownloadPhaseNoItemsSwitchError(t *testing.T) {
	input := changeExecutionInput{
		targetVersion: "1.20.1",
		deps: changeDeps{
			fs: afero.NewMemMapFs(),
			writeLock: func(context.Context, afero.Fs, config.Metadata, []models.ModInstall) error {
				return errors.New("lock failed")
			},
		},
		meta: config.NewMetadata("/cfg/modlist.json"),
	}
	state := newChangeExecutionState(changeExecutionInput{items: []changeItem{}, indexByKey: map[string]int{}}, nil)

	outcome, handled := outcomeFromDownloadPhase(context.Background(), input, state, downloadPhaseResult{}, "/staging")
	assert.True(t, handled)
	assert.Equal(t, changeStageSwitchFailed, outcome.Stage)
	assert.Error(t, outcome.Err)
}

func TestOutcomeFromDownloadPhaseNoItemsSuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0o755))
	input := changeExecutionInput{
		targetVersion: "1.20.1",
		deps: changeDeps{
			fs:        fs,
			writeLock: config.WriteLock,
			writeConfig: func(context.Context, afero.Fs, config.Metadata, models.ModsJSON) error {
				return nil
			},
			removeAll: func(afero.Fs, string) error { return nil },
		},
		meta: meta,
		cfg:  cfg,
	}

	state := newChangeExecutionState(changeExecutionInput{items: []changeItem{}, indexByKey: map[string]int{}}, nil)
	outcome, handled := outcomeFromDownloadPhase(context.Background(), input, state, downloadPhaseResult{}, "/staging")
	assert.True(t, handled)
	assert.Equal(t, changeStageSuccess, outcome.Stage)
}

func TestRunSwitchPhaseHandlesSwitchError(t *testing.T) {
	input := changeExecutionInput{
		meta:          config.NewMetadata("/cfg/modlist.json"),
		cfg:           models.ModsJSON{ModsFolder: "mods"},
		targetVersion: "1.20.1",
		deps: changeDeps{
			fs:        afero.NewMemMapFs(),
			mkdirAll:  func(afero.Fs, string, os.FileMode) error { return errors.New("mkdir failed") },
			removeAll: func(afero.Fs, string) error { return nil },
		},
	}
	state := newChangeExecutionState(changeExecutionInput{items: []changeItem{}, indexByKey: map[string]int{}}, nil)
	result := downloadPhaseResult{}
	staging := changeStagingPaths{root: "/staging", backup: "/staging/backup"}

	outcome := runSwitchPhase(context.Background(), input, state, result, staging)
	assert.Equal(t, changeStageSwitchFailed, outcome.Stage)
	assert.Error(t, outcome.Err)
}

func TestRunSwitchPhaseReturnsUpdatedItemsOnSwitchFailure(t *testing.T) {
	item := changeItem{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}}
	input := changeExecutionInput{
		meta:          config.NewMetadata("/cfg/modlist.json"),
		cfg:           models.ModsJSON{ModsFolder: "mods"},
		targetVersion: "1.20.1",
		deps: changeDeps{
			fs:         afero.NewMemMapFs(),
			mkdirAll:   func(afero.Fs, string, os.FileMode) error { return nil },
			removeAll:  func(afero.Fs, string) error { return nil },
			renameFile: func(afero.Fs, string, string) error { return nil },
			removeFile: func(afero.Fs, string) error { return nil },
			writeLock: func(context.Context, afero.Fs, config.Metadata, []models.ModInstall) error {
				return nil
			},
			writeConfig: func(context.Context, afero.Fs, config.Metadata, models.ModsJSON) error {
				return nil
			},
		},
	}
	state := newChangeExecutionState(changeExecutionInput{
		items:      []changeItem{item},
		indexByKey: map[string]int{changeModKey(item.Mod): 0},
	}, nil)
	result := downloadPhaseResult{}
	staging := changeStagingPaths{root: "/staging", backup: "/staging/backup"}

	outcome := runSwitchPhase(context.Background(), input, state, result, staging)
	require.Equal(t, changeStageSwitchFailed, outcome.Stage)
	require.Len(t, outcome.Items, 1)
	assert.Equal(t, changeSwitchFailed, outcome.Items[0].SwitchStatus)
	assert.Contains(t, outcome.Items[0].ErrorReason, "missing lock entry")
}

func TestRunSwitchPhaseHandlesCleanupError(t *testing.T) {
	input := changeExecutionInput{
		meta:          config.NewMetadata("/cfg/modlist.json"),
		cfg:           models.ModsJSON{ModsFolder: "mods"},
		targetVersion: "1.20.1",
		deps: changeDeps{
			fs:        afero.NewMemMapFs(),
			mkdirAll:  func(afero.Fs, string, os.FileMode) error { return nil },
			writeLock: func(context.Context, afero.Fs, config.Metadata, []models.ModInstall) error { return nil },
			writeConfig: func(context.Context, afero.Fs, config.Metadata, models.ModsJSON) error {
				return nil
			},
			removeAll: func(afero.Fs, string) error { return errors.New("cleanup failed") },
		},
	}
	state := newChangeExecutionState(changeExecutionInput{items: []changeItem{}, indexByKey: map[string]int{}}, nil)
	result := downloadPhaseResult{}
	staging := changeStagingPaths{root: "/staging", backup: "/staging/backup"}

	outcome := runSwitchPhase(context.Background(), input, state, result, staging)
	assert.Equal(t, changeStageSwitchFailed, outcome.Stage)
	assert.Error(t, outcome.Err)
}

func TestUpdateConfigWithoutModsInvalidFilename(t *testing.T) {
	fs := afero.NewMemMapFs()
	input := changeExecutionInput{
		meta: config.NewMetadata("/cfg/modlist.json"),
		cfg:  models.ModsJSON{ModsFolder: "mods"},
		lock: []models.ModInstall{{FileName: "mods/alpha.jar"}},
		deps: changeDeps{fs: fs},
	}

	err := updateConfigWithoutMods(context.Background(), input, "/staging")
	assert.Error(t, err)
}

func TestUpdateConfigWithoutModsResolveError(t *testing.T) {
	meta := config.NewMetadata(filepath.Join(t.TempDir(), "missing", "modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	input := changeExecutionInput{
		meta: meta,
		cfg:  cfg,
		lock: []models.ModInstall{{FileName: "alpha.jar"}},
		deps: changeDeps{fs: afero.NewOsFs()},
	}

	err := updateConfigWithoutMods(context.Background(), input, "/staging")
	assert.Error(t, err)
}

func TestUpdateConfigWithoutModsExistsError(t *testing.T) {
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	path := filepath.Join(meta.ModsFolderPath(cfg), "alpha.jar")
	fs := statErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: path,
		err:      errors.New("stat failed"),
	}

	input := changeExecutionInput{
		meta: meta,
		cfg:  cfg,
		lock: []models.ModInstall{{FileName: "alpha.jar"}},
		deps: changeDeps{fs: fs},
	}

	err := updateConfigWithoutMods(context.Background(), input, "/staging")
	assert.Error(t, err)
}

func TestUpdateConfigWithoutModsRemoveFileError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "alpha.jar"), []byte("old"), 0o644))

	input := changeExecutionInput{
		meta: meta,
		cfg:  cfg,
		lock: []models.ModInstall{{FileName: "alpha.jar"}},
		deps: changeDeps{
			fs:         fs,
			removeFile: func(afero.Fs, string) error { return errors.New("remove failed") },
		},
	}

	err := updateConfigWithoutMods(context.Background(), input, "/staging")
	assert.Error(t, err)
}

func TestUpdateConfigWithoutModsWriteLockError(t *testing.T) {
	fs := afero.NewMemMapFs()
	input := changeExecutionInput{
		meta: config.NewMetadata("/cfg/modlist.json"),
		cfg:  models.ModsJSON{ModsFolder: "mods"},
		deps: changeDeps{
			fs: fs,
			writeLock: func(context.Context, afero.Fs, config.Metadata, []models.ModInstall) error {
				return errors.New("lock failed")
			},
		},
	}

	err := updateConfigWithoutMods(context.Background(), input, "/staging")
	assert.Error(t, err)
}

func TestUpdateConfigWithoutModsWriteConfigErrorRestoresLock(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	lockCalls := 0

	input := changeExecutionInput{
		meta: meta,
		cfg:  models.ModsJSON{ModsFolder: "mods"},
		lock: []models.ModInstall{{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}},
		deps: changeDeps{
			fs: fs,
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
	}

	err := updateConfigWithoutMods(context.Background(), input, "/staging")
	assert.Error(t, err)
}

func TestUpdateConfigWithoutModsRemoveAllError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")

	input := changeExecutionInput{
		meta: meta,
		cfg:  models.ModsJSON{ModsFolder: "mods"},
		deps: changeDeps{
			fs:        fs,
			writeLock: func(context.Context, afero.Fs, config.Metadata, []models.ModInstall) error { return nil },
			writeConfig: func(context.Context, afero.Fs, config.Metadata, models.ModsJSON) error {
				return nil
			},
			removeAll: func(afero.Fs, string) error { return errors.New("remove failed") },
		},
	}

	err := updateConfigWithoutMods(context.Background(), input, "/staging")
	assert.Error(t, err)
}
