package scan

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

func TestScanExecStateUpdateStatus(t *testing.T) {
	items := []scanItem{
		{FileName: "alpha.jar", Status: scanItemStatusPending},
		{FileName: "beta.jar", Status: scanItemStatusRecognized},
	}
	index := scanIndexByFile(items)
	sent := false
	state := newScanExecState(items, index, scanExecSender{send: func(msg tea.Msg) { sent = true }})

	state.updateStatus("missing.jar", scanItemStatusUnknown, scanMatch{})
	assert.False(t, sent)

	state.updateStatus("beta.jar", scanItemStatusUnknown, scanMatch{})
	assert.Equal(t, scanItemStatusRecognized, state.items[index["beta.jar"]].Status)

	match := scanMatch{FileName: "alpha.jar", Name: "Alpha", ProjectID: "alpha", Platform: models.MODRINTH}
	state.updateStatus("alpha.jar", scanItemStatusRecognized, match)
	assert.Equal(t, scanItemStatusRecognized, state.items[index["alpha.jar"]].Status)
	assert.Equal(t, match, state.items[index["alpha.jar"]].Match)
	assert.True(t, sent)
}

func TestLookupPlatformWithUpdatesUnknownPlatformReturnsMisses(t *testing.T) {
	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar"}}
	items, index := buildScanItems(candidates)
	updates := make(chan tea.Msg, 20)
	state := newScanExecState(items, index, scanExecSender{send: func(msg tea.Msg) { updates <- msg }})

	outcome, err := lookupPlatformWithUpdates(context.Background(), models.Platform("custom"), candidates, scanDeps{}, state)
	assert.NoError(t, err)
	assert.Equal(t, candidates, outcome.misses)
	assert.Empty(t, outcome.matches)
	assert.Empty(t, outcome.unsure)
}

func TestLookupModrinthWithUpdatesHandlesMatchMissAndUnsure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{
		{Path: "/mods/match.jar", FileName: "match.jar", Sha1: "match"},
		{Path: "/mods/miss.jar", FileName: "miss.jar", Sha1: "miss"},
		{Path: "/mods/fallback.jar", FileName: "fallback.jar", Sha1: "fallback"},
		{Path: "/mods/timeout.jar", FileName: "timeout.jar", Sha1: "timeout"},
	}
	items, index := buildScanItems(candidates)
	state := newScanExecState(items, index, scanExecSender{})

	version := &modrinth.Version{
		ProjectID:     "proj",
		DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Files: []modrinth.VersionFile{
			{URL: "https://example.invalid/match.jar", Primary: true},
		},
	}

	deps := scanDeps{
		clients: platform.Clients{Modrinth: noopDoer{}},
		modrinthVersionForSha: func(_ context.Context, hash string, _ httpclient.Doer) (*modrinth.Version, error) {
			switch hash {
			case "match":
				return version, nil
			case "miss":
				return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup(hash, modrinth.SHA1)}
			case "timeout":
				return nil, httpclient.WrapTimeoutError(context.DeadlineExceeded)
			default:
				return nil, errors.New("boom")
			}
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Match Title", nil
		},
	}

	outcome, err := lookupModrinthWithUpdates(context.Background(), candidates, deps, state)
	assert.NoError(t, err)
	assert.Len(t, outcome.matches, 1)
	assert.Len(t, outcome.misses, 2)
	assert.Contains(t, outcome.unsure, "/mods/timeout.jar")

	assert.Equal(t, scanItemStatusRecognized, state.items[index["match.jar"]].Status)
	assert.Equal(t, scanItemStatusUnsure, state.items[index["timeout.jar"]].Status)
	assert.Equal(t, scanItemStatusScanning, state.items[index["fallback.jar"]].Status)
}

func TestLookupModrinthWithUpdatesMarksAllItemsScanning(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{
		{Path: "/mods/Alpha.jar", FileName: "Alpha.jar", Sha1: "a"},
		{Path: "/mods/bravo.jar", FileName: "bravo.jar", Sha1: "b"},
		{Path: "/mods/charlie.jar", FileName: "charlie.jar", Sha1: "c"},
	}
	items, index := buildScanItems(candidates)
	updates := make(chan tea.Msg, 20)
	state := newScanExecState(items, index, scanExecSender{send: func(msg tea.Msg) { updates <- msg }})

	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	lookupErr := make(chan error, 1)

	deps := scanDeps{
		clients: platform.Clients{Modrinth: noopDoer{}},
		modrinthVersionForSha: func(_ context.Context, hash string, _ httpclient.Doer) (*modrinth.Version, error) {
			select {
			case <-started:
			default:
				close(started)
			}
			<-release
			return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup(hash, modrinth.SHA1)}
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Title", nil
		},
	}

	go func() {
		_, err := lookupModrinthWithUpdates(context.Background(), candidates, deps, state)
		lookupErr <- err
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for modrinth lookup")
	}

	scanningKeys := make(map[string]struct{}, len(candidates))
	for len(scanningKeys) < len(candidates) {
		select {
		case msg := <-updates:
			update, ok := msg.(scanItemUpdateMsg)
			if !ok {
				continue
			}
			if update.status == scanItemStatusScanning {
				scanningKeys[update.key] = struct{}{}
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for scanning updates")
		}
	}

	assert.Len(t, scanningKeys, len(candidates))

	close(release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for modrinth completion")
	}
	assert.NoError(t, <-lookupErr)
}

func TestLookupModrinthWithUpdatesReturnsErrorOnLogFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{{Path: "/mods/bad.jar", FileName: "bad.jar", Sha1: "bad"}}
	items, index := buildScanItems(candidates)
	state := newScanExecState(items, index, scanExecSender{})

	writeErr := errors.New("write failed")
	deps := scanDeps{
		logger: logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, errors.New("boom")
		},
	}

	_, err := lookupModrinthWithUpdates(context.Background(), candidates, deps, state)
	assert.ErrorIs(t, err, writeErr)
}

func TestLookupModrinthWithUpdatesReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}
	items, index := buildScanItems(candidates)
	state := newScanExecState(items, index, scanExecSender{})

	deps := scanDeps{
		clients: platform.Clients{Modrinth: noopDoer{}},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, errors.New("boom")
		},
	}

	_, err := lookupModrinthWithUpdates(ctx, candidates, deps, state)
	assert.Error(t, err)
}

func TestLookupModrinthWithUpdatesSortsMissesByFile(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{
		{Path: "/mods/beta.jar", FileName: "beta.jar", Sha1: "beta"},
		{Path: "/mods/Alpha.jar", FileName: "Alpha.jar", Sha1: "alpha-upper"},
		{Path: "/mods/alpha.jar", FileName: "alpha.jar", Sha1: "alpha-lower"},
	}
	items, index := buildScanItems(candidates)
	state := newScanExecState(items, index, scanExecSender{})

	deps := scanDeps{
		clients: platform.Clients{Modrinth: noopDoer{}},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup("missing", modrinth.SHA1)}
		},
	}

	outcome, err := lookupModrinthWithUpdates(context.Background(), candidates, deps, state)
	assert.NoError(t, err)
	require.Equal(t, 3, len(outcome.misses))
	assert.Equal(t, "Alpha.jar", outcome.misses[0].FileName)
	assert.Equal(t, "alpha.jar", outcome.misses[1].FileName)
	assert.Equal(t, "beta.jar", outcome.misses[2].FileName)
}

func TestLookupCurseforgeWithUpdatesSkipsMissesAndUpdatesUnsure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{
		{Path: "/mods/skip.jar", FileName: "skip.jar", Sha1: "a"},
		{Path: "/mods/unsure.jar", FileName: "unsure.jar", Sha1: "b"},
	}
	items, index := buildScanItems(candidates)
	state := newScanExecState(items, index, scanExecSender{})

	deps := scanDeps{
		clients: platform.Clients{Curseforge: noopDoer{}},
		curseforgeFingerprint: func(path string) uint32 {
			if strings.Contains(path, "skip.jar") {
				return 101
			}
			return 202
		},
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{
					{ProjectID: 1, Fingerprint: 101, DownloadURL: "", FileDate: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)},
					{ProjectID: 2, Fingerprint: 202, DownloadURL: "https://example.invalid/file.jar", FileDate: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC)},
				},
			}, nil
		},
		curseforgeProjectName: func(_ context.Context, projectID string, _ httpclient.Doer) (string, error) {
			if projectID == "2" {
				return "", httpclient.WrapTimeoutError(context.DeadlineExceeded)
			}
			return "Name", nil
		},
	}

	outcome, err := lookupCurseforgeWithUpdates(context.Background(), candidates, deps, state)
	assert.NoError(t, err)
	assert.Len(t, outcome.misses, 1)
	assert.Contains(t, outcome.unsure, "/mods/skip.jar")
	assert.Contains(t, outcome.unsure, "/mods/unsure.jar")

	assert.Equal(t, scanItemStatusScanning, state.items[index["skip.jar"]].Status)
	assert.Equal(t, scanItemStatusUnsure, state.items[index["unsure.jar"]].Status)
}

func TestLookupCurseforgeWithUpdatesMarksAllItemsScanning(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{
		{Path: "/mods/Alpha.jar", FileName: "Alpha.jar", Sha1: "a"},
		{Path: "/mods/bravo.jar", FileName: "bravo.jar", Sha1: "b"},
		{Path: "/mods/charlie.jar", FileName: "charlie.jar", Sha1: "c"},
		{Path: "/mods/delta.jar", FileName: "delta.jar", Sha1: "d"},
		{Path: "/mods/echo.jar", FileName: "echo.jar", Sha1: "e"},
		{Path: "/mods/foxtrot.jar", FileName: "foxtrot.jar", Sha1: "f"},
	}
	items, index := buildScanItems(candidates)
	state := newScanExecState(items, index, scanExecSender{})

	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	lookupErr := make(chan error, 1)

	fingerprintByPath := map[string]uint32{
		"/mods/Alpha.jar":   1,
		"/mods/bravo.jar":   2,
		"/mods/charlie.jar": 3,
		"/mods/delta.jar":   4,
		"/mods/echo.jar":    5,
		"/mods/foxtrot.jar": 6,
	}

	deps := scanDeps{
		clients: platform.Clients{Curseforge: noopDoer{}},
		curseforgeFingerprint: func(path string) uint32 {
			return fingerprintByPath[path]
		},
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			close(started)
			<-release
			return &curseforge.FingerprintResult{}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpclient.Doer) (string, error) {
			return "", errors.New("unexpected project lookup")
		},
	}

	go func() {
		_, err := lookupCurseforgeWithUpdates(context.Background(), candidates, deps, state)
		lookupErr <- err
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for curseforge lookup")
	}

	activeCount := 0
	pendingCount := 0
	for _, item := range state.items {
		if item.Status == scanItemStatusScanning {
			activeCount++
		}
		if item.Status == scanItemStatusPending {
			pendingCount++
		}
	}

	assert.Equal(t, len(candidates), activeCount)
	assert.Equal(t, 0, pendingCount)

	close(release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for curseforge completion")
	}
	assert.NoError(t, <-lookupErr)
}

func TestFinalizeScanResultsUpdatesStatuses(t *testing.T) {
	items := []scanItem{
		{FileName: "a.jar", Status: scanItemStatusPending},
		{FileName: "b.jar", Status: scanItemStatusPending},
	}
	index := scanIndexByFile(items)
	state := newScanExecState(items, index, scanExecSender{})

	preferredUnsure := map[string]error{"/mods/a.jar": errors.New("timeout")}
	fallbackMisses := []scanCandidate{
		{Path: "/mods/a.jar", FileName: "a.jar"},
		{Path: "/mods/b.jar", FileName: "b.jar"},
		{Path: "/mods/b.jar", FileName: "b.jar"},
	}

	unknown, unsure := finalizeScanResults(state, preferredUnsure, map[string]error{}, fallbackMisses, nil)
	assert.Equal(t, []string{"b.jar"}, unknown)
	assert.Len(t, unsure, 1)
	assert.Equal(t, scanItemStatusUnsure, state.items[index["a.jar"]].Status)
	assert.Equal(t, scanItemStatusUnknown, state.items[index["b.jar"]].Status)
}

func TestRunScanExecutionFallbackMatchRemovesUnsure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}
	input := scanExecutionInput{
		candidates:     candidates,
		preferPlatform: models.MODRINTH,
		deps: scanDeps{
			clients: platform.Clients{
				Modrinth:   noopDoer{},
				Curseforge: noopDoer{},
			},
			modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
				return nil, errors.New("boom")
			},
			curseforgeFingerprint: func(string) uint32 { return 101 },
			curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
				return &curseforge.FingerprintResult{
					Matches: []curseforge.File{
						{
							ProjectID:   42,
							Fingerprint: 101,
							DownloadURL: "https://example.invalid/cf.jar",
							FileDate:    time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
						},
					},
				}, nil
			},
			curseforgeProjectName: func(context.Context, string, httpclient.Doer) (string, error) {
				return "CurseForge Mod", nil
			},
		},
	}

	outcome := runScanExecution(context.Background(), input, scanExecSender{})
	assert.Len(t, outcome.matches, 1)
	assert.Empty(t, outcome.unsure)
}

func TestRunScanExecutionPreferredLookupError(t *testing.T) {
	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}
	writeErr := errors.New("write failed")

	input := scanExecutionInput{
		candidates:     candidates,
		preferPlatform: models.MODRINTH,
		deps: scanDeps{
			logger: logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
			modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
				return nil, errors.New("boom")
			},
		},
	}

	outcome := runScanExecution(context.Background(), input, scanExecSender{})
	assert.ErrorIs(t, outcome.err, writeErr)
}

func TestRunScanExecutionFallbackLookupError(t *testing.T) {
	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}
	writeErr := errors.New("write failed")

	input := scanExecutionInput{
		candidates:     candidates,
		preferPlatform: models.MODRINTH,
		deps: scanDeps{
			logger: logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
			modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
				return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup("a", modrinth.SHA1)}
			},
			curseforgeFingerprint: func(string) uint32 { return 101 },
			curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
				return nil, errors.New("boom")
			},
		},
	}

	outcome := runScanExecution(context.Background(), input, scanExecSender{})
	assert.ErrorIs(t, outcome.err, writeErr)
}
