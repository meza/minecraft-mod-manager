package scan

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
		Name:          "Match Title",
		DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Files: []modrinth.VersionFile{
			{URL: "https://example.invalid/match.jar", Primary: true},
		},
	}

	deps := scanDeps{
		clients: platform.Clients{Modrinth: noopDoer{}},
		modrinthVersionForSha: func(ctx context.Context, hash string, _ httpclient.Doer) (*modrinth.Version, error) {
			triggerRequestStartHook(ctx, t)
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
	}

	outcome, err := lookupModrinthWithUpdates(context.Background(), candidates, deps, state)
	assert.NoError(t, err)
	assert.Len(t, outcome.matches, 1)
	assert.Len(t, outcome.misses, 2)
	assert.Contains(t, outcome.unsure, "/mods/timeout.jar")

	assert.Equal(t, scanItemStatusRecognized, state.items[index["match.jar"]].Status)
	assert.Equal(t, scanItemStatusUnsure, state.items[index["timeout.jar"]].Status)
	assert.Equal(t, scanItemStatusPending, state.items[index["fallback.jar"]].Status)
}

func TestLookupModrinthWithUpdatesWithNoCandidates(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	items, index := buildScanItems(nil)
	state := newScanExecState(items, index, scanExecSender{})

	outcome, err := lookupModrinthWithUpdates(context.Background(), nil, scanDeps{}, state)
	assert.NoError(t, err)
	assert.Empty(t, outcome.matches)
	assert.Empty(t, outcome.misses)
	assert.Empty(t, outcome.unsure)
}

func TestLookupModrinthWithUpdatesLeavesQueuedItemsPending(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidateCount := 20

	candidates := make([]scanCandidate, 0, candidateCount)
	for index := 0; index < candidateCount; index++ {
		fileName := fmt.Sprintf("mod-%d.jar", index)
		candidates = append(candidates, scanCandidate{
			Path:     "/mods/" + fileName,
			FileName: fileName,
			Sha1:     fmt.Sprintf("hash-%d", index),
		})
	}
	items, index := buildScanItems(candidates)
	updates := make(chan tea.Msg, candidateCount*3)
	state := newScanExecState(items, index, scanExecSender{send: func(msg tea.Msg) { updates <- msg }})

	release := make(chan struct{})
	startTokens := make(chan struct{}, 1)
	startTokens <- struct{}{}
	done := make(chan struct{})
	lookupErr := make(chan error, 1)

	deps := scanDeps{
		clients: platform.Clients{Modrinth: noopDoer{}},
		modrinthVersionForSha: func(ctx context.Context, hash string, _ httpclient.Doer) (*modrinth.Version, error) {
			<-startTokens
			triggerRequestStartHook(ctx, t)
			<-release
			return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup(hash, modrinth.SHA1)}
		},
	}

	go func() {
		_, err := lookupModrinthWithUpdates(context.Background(), candidates, deps, state)
		lookupErr <- err
		close(done)
	}()

	firstKey := waitForScanUpdateStatus(t, updates, scanItemStatusScanning)
	select {
	case msg := <-updates:
		update, ok := msg.(scanItemUpdateMsg)
		if ok && update.status == scanItemStatusScanning {
			t.Fatalf("expected queued items to remain pending; got extra scanning update for %q after %q", update.key, firstKey)
		}
	case <-time.After(100 * time.Millisecond):
	}

	close(startTokens)
	close(release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for modrinth completion")
	}
	assert.NoError(t, <-lookupErr)
}

func TestLookupModrinthWithUpdatesTogglesStatusPerRequest(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{{Path: "/mods/alpha.jar", FileName: "alpha.jar", Sha1: "alpha"}}
	items, index := buildScanItems(candidates)
	updates := make(chan tea.Msg, 10)
	state := newScanExecState(items, index, scanExecSender{send: func(msg tea.Msg) { updates <- msg }})

	versionStarted := make(chan struct{})
	releaseVersion := make(chan struct{})

	deps := scanDeps{
		clients: platform.Clients{Modrinth: noopDoer{}},
		modrinthVersionForSha: func(ctx context.Context, _ string, _ httpclient.Doer) (*modrinth.Version, error) {
			triggerRequestStartHook(ctx, t)
			close(versionStarted)
			<-releaseVersion
			return &modrinth.Version{
				ProjectID:     "alpha-project",
				Name:          "Alpha",
				DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
				Files:         []modrinth.VersionFile{{URL: "https://example.invalid/alpha.jar", Primary: true}},
			}, nil
		},
	}

	done := make(chan error, 1)
	go func() {
		_, err := lookupModrinthWithUpdates(context.Background(), candidates, deps, state)
		done <- err
	}()

	select {
	case <-versionStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for version lookup to start")
	}
	waitForScanUpdateStatus(t, updates, scanItemStatusScanning)

	close(releaseVersion)
	waitForScanUpdateStatus(t, updates, scanItemStatusRecognized)

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for modrinth completion")
	}

	assert.Equal(t, scanItemStatusRecognized, state.items[index["alpha.jar"]].Status)
}

func TestModrinthLookupWorkerCountUsesCandidateCount(t *testing.T) {
	tests := []struct {
		name           string
		candidateCount int
		expected       int
	}{
		{
			name:           "no candidates",
			candidateCount: 0,
			expected:       0,
		},
		{
			name:           "single candidate",
			candidateCount: 1,
			expected:       1,
		},
		{
			name:           "multiple candidates",
			candidateCount: 5,
			expected:       5,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, modrinthLookupWorkerCount(test.candidateCount))
		})
	}
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
		modrinthVersionForSha: func(ctx context.Context, _ string, _ httpclient.Doer) (*modrinth.Version, error) {
			triggerRequestStartHook(ctx, t)
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
		modrinthVersionForSha: func(ctx context.Context, _ string, _ httpclient.Doer) (*modrinth.Version, error) {
			triggerRequestStartHook(ctx, t)
			return nil, errors.New("boom")
		},
	}

	_, err := lookupModrinthWithUpdates(ctx, candidates, deps, state)
	assert.Error(t, err)
}

func TestLookupModrinthWithUpdatesCanceledBeforeSendingWork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	candidates := []scanCandidate{
		{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"},
		{Path: "/mods/b.jar", FileName: "b.jar", Sha1: "b"},
	}
	items, index := buildScanItems(candidates)
	state := newScanExecState(items, index, scanExecSender{})

	deps := scanDeps{
		clients: platform.Clients{Modrinth: noopDoer{}},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			t.Fatal("modrinthVersionForSha should not be called when context is canceled")
			return nil, errors.New("unexpected call")
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
		modrinthVersionForSha: func(ctx context.Context, _ string, _ httpclient.Doer) (*modrinth.Version, error) {
			triggerRequestStartHook(ctx, t)
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

func TestLookupModrinthWithUpdatesRecognizesBeforeOtherCandidateCompletes(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{
		{Path: "/mods/alpha.jar", FileName: "alpha.jar", Sha1: "alpha"},
		{Path: "/mods/bravo.jar", FileName: "bravo.jar", Sha1: "bravo"},
	}
	items, index := buildScanItems(candidates)
	updates := make(chan tea.Msg, 20)
	state := newScanExecState(items, index, scanExecSender{send: func(msg tea.Msg) { updates <- msg }})

	bravoStarted := make(chan struct{})
	releaseBravo := make(chan struct{})
	done := make(chan error, 1)

	deps := scanDeps{
		clients: platform.Clients{Modrinth: noopDoer{}},
		modrinthVersionForSha: func(ctx context.Context, hash string, _ httpclient.Doer) (*modrinth.Version, error) {
			switch hash {
			case "alpha":
				triggerRequestStartHook(ctx, t)
				return &modrinth.Version{
					ProjectID:     "alpha-project",
					Name:          "Alpha",
					DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
					Files:         []modrinth.VersionFile{{URL: "https://example.invalid/alpha.jar", Primary: true}},
				}, nil
			default:
				triggerRequestStartHook(ctx, t)
				close(bravoStarted)
				<-releaseBravo
				return &modrinth.Version{
					ProjectID:     "bravo-project",
					Name:          "Bravo",
					DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
					Files:         []modrinth.VersionFile{{URL: "https://example.invalid/bravo.jar", Primary: true}},
				}, nil
			}
		},
	}

	go func() {
		_, err := lookupModrinthWithUpdates(context.Background(), candidates, deps, state)
		done <- err
	}()

	select {
	case <-bravoStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for bravo lookup to start")
	}

	waitForScanUpdateStatus(t, updates, scanItemStatusRecognized)

	close(releaseBravo)
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for modrinth completion")
	}

	assert.Equal(t, scanItemStatusRecognized, state.items[index["alpha.jar"]].Status)
	assert.Equal(t, scanItemStatusRecognized, state.items[index["bravo.jar"]].Status)
}

func TestLookupModrinthWithUpdatesStopsOnContextCancel(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	candidates := []scanCandidate{
		{Path: "/mods/alpha.jar", FileName: "alpha.jar", Sha1: "alpha"},
		{Path: "/mods/bravo.jar", FileName: "bravo.jar", Sha1: "bravo"},
	}
	items, index := buildScanItems(candidates)
	state := newScanExecState(items, index, scanExecSender{})

	started := make(chan struct{}, len(candidates))
	deps := scanDeps{
		clients: platform.Clients{Modrinth: noopDoer{}},
		modrinthVersionForSha: func(ctx context.Context, _ string, _ httpclient.Doer) (*modrinth.Version, error) {
			started <- struct{}{}
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	done := make(chan error, 1)
	go func() {
		_, err := lookupModrinthWithUpdates(ctx, candidates, deps, state)
		done <- err
	}()

	for startedCount := 0; startedCount < len(candidates); startedCount++ {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for lookup start")
		}
	}
	cancel()

	select {
	case err := <-done:
		assert.Error(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for cancellation")
	}
}

func TestLookupCurseforgeWithUpdatesSkipsMissesAndUpdatesUnsure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{
		{Path: "/mods/skip.jar", FileName: "skip.jar", Sha1: "a"},
		{Path: "/mods/recognized.jar", FileName: "recognized.jar", Sha1: "b"},
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
		curseforgeFingerprintMatch: func(ctx context.Context, _ []uint32, _ httpclient.Doer) (*curseforge.FingerprintResult, error) {
			triggerRequestStartHook(ctx, t)
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{
					{
						ProjectID:   1,
						Fingerprint: 101,
						DownloadURL: "",
						FileDate:    time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
					},
					{
						ProjectID:   2,
						Fingerprint: 202,
						DisplayName: "Recognized File",
						DownloadURL: "https://example.invalid/file.jar",
						FileDate:    time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC),
					},
				},
			}, nil
		},
	}

	outcome, err := lookupCurseforgeWithUpdates(context.Background(), candidates, deps, state)
	assert.NoError(t, err)
	assert.Len(t, outcome.misses, 1)
	assert.Contains(t, outcome.unsure, "/mods/skip.jar")
	assert.NotContains(t, outcome.unsure, "/mods/recognized.jar")

	assert.Equal(t, scanItemStatusPending, state.items[index["skip.jar"]].Status)
	assert.Equal(t, scanItemStatusRecognized, state.items[index["recognized.jar"]].Status)
}

func TestLookupCurseforgeWithUpdatesBatchFailureMarksUnsure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{
		{Path: "/mods/timeout.jar", FileName: "timeout.jar", Sha1: "a"},
	}
	items, index := buildScanItems(candidates)
	state := newScanExecState(items, index, scanExecSender{})

	deps := scanDeps{
		clients:               platform.Clients{Curseforge: noopDoer{}},
		curseforgeFingerprint: func(string) uint32 { return 101 },
		curseforgeFingerprintMatch: func(ctx context.Context, _ []uint32, _ httpclient.Doer) (*curseforge.FingerprintResult, error) {
			triggerRequestStartHook(ctx, t)
			return nil, httpclient.WrapTimeoutError(context.DeadlineExceeded)
		},
	}

	outcome, err := lookupCurseforgeWithUpdates(context.Background(), candidates, deps, state)
	assert.NoError(t, err)
	assert.Empty(t, outcome.matches)
	assert.Empty(t, outcome.misses)
	assert.Contains(t, outcome.unsure, "/mods/timeout.jar")
	assert.Equal(t, scanItemStatusUnsure, state.items[index["timeout.jar"]].Status)
}

func TestLookupCurseforgeWithUpdatesSetsRecognizedOnMatch(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{
		{Path: "/mods/alpha.jar", FileName: "alpha.jar", Sha1: "a"},
	}
	items, index := buildScanItems(candidates)
	state := newScanExecState(items, index, scanExecSender{})

	deps := scanDeps{
		clients:               platform.Clients{Curseforge: noopDoer{}},
		curseforgeFingerprint: func(string) uint32 { return 101 },
		curseforgeFingerprintMatch: func(ctx context.Context, _ []uint32, _ httpclient.Doer) (*curseforge.FingerprintResult, error) {
			triggerRequestStartHook(ctx, t)
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{
					{
						ProjectID:   22,
						Fingerprint: 101,
						DisplayName: "Curse Display",
						DownloadURL: "https://example.invalid/alpha.jar",
						FileDate:    time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
					},
				},
			}, nil
		},
	}

	outcome, err := lookupCurseforgeWithUpdates(context.Background(), candidates, deps, state)
	assert.NoError(t, err)
	assert.Len(t, outcome.matches, 1)
	assert.Equal(t, "Curse Display", outcome.matches[0].Name)
	assert.Equal(t, scanItemStatusRecognized, state.items[index["alpha.jar"]].Status)
}

func TestLookupCurseforgeWithUpdatesEmitsRecognizedBeforePendingMisses(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{
		{Path: "/mods/match.jar", FileName: "match.jar", Sha1: "a"},
		{Path: "/mods/miss.jar", FileName: "miss.jar", Sha1: "b"},
	}
	items, index := buildScanItems(candidates)
	updates := make(chan tea.Msg, 10)
	state := newScanExecState(items, index, scanExecSender{send: func(msg tea.Msg) { updates <- msg }})

	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)

	deps := scanDeps{
		clients: platform.Clients{Curseforge: noopDoer{}},
		curseforgeFingerprint: func(path string) uint32 {
			if strings.Contains(path, "match.jar") {
				return 101
			}
			return 202
		},
		curseforgeFingerprintMatch: func(ctx context.Context, _ []uint32, _ httpclient.Doer) (*curseforge.FingerprintResult, error) {
			triggerRequestStartHook(ctx, t)
			close(started)
			<-release
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{
					{
						ProjectID:   42,
						Fingerprint: 101,
						DisplayName: "Matched File",
						DownloadURL: "https://example.invalid/match.jar",
						FileDate:    time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
					},
				},
			}, nil
		},
	}

	go func() {
		_, err := lookupCurseforgeWithUpdates(context.Background(), candidates, deps, state)
		done <- err
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for curseforge request start")
	}

	seenScanning := map[string]struct{}{}
	for len(seenScanning) < len(candidates) {
		key := waitForScanUpdateStatus(t, updates, scanItemStatusScanning)
		seenScanning[key] = struct{}{}
	}

	close(release)

	recognizedKey := waitForScanUpdateStatus(t, updates, scanItemStatusRecognized)
	assert.Equal(t, "match.jar", recognizedKey)
	pendingKey := waitForScanUpdateStatus(t, updates, scanItemStatusPending)
	assert.Equal(t, "miss.jar", pendingKey)

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for curseforge completion")
	}
}

func TestLookupCurseforgeWithUpdatesTogglesBatchRequestStatus(t *testing.T) {
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
		curseforgeFingerprintMatch: func(ctx context.Context, _ []uint32, _ httpclient.Doer) (*curseforge.FingerprintResult, error) {
			triggerRequestStartHook(ctx, t)
			close(started)
			<-release
			return &curseforge.FingerprintResult{}, nil
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

	for _, item := range state.items {
		assert.Equal(t, scanItemStatusScanning, item.Status)
	}

	close(release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for curseforge completion")
	}
	assert.NoError(t, <-lookupErr)
	for _, item := range state.items {
		assert.Equal(t, scanItemStatusPending, item.Status)
	}
}

func TestLookupCurseforgeWithUpdatesUsesDisplayName(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{
		{Path: "/mods/alpha.jar", FileName: "alpha.jar", Sha1: "a"},
		{Path: "/mods/bravo.jar", FileName: "bravo.jar", Sha1: "b"},
	}
	items, index := buildScanItems(candidates)
	state := newScanExecState(items, index, scanExecSender{})

	deps := scanDeps{
		clients: platform.Clients{Curseforge: noopDoer{}},
		curseforgeFingerprint: func(path string) uint32 {
			if strings.Contains(path, "alpha") {
				return 101
			}
			return 202
		},
		curseforgeFingerprintMatch: func(ctx context.Context, _ []uint32, _ httpclient.Doer) (*curseforge.FingerprintResult, error) {
			triggerRequestStartHook(ctx, t)
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{
					{
						ProjectID:   1,
						Fingerprint: 101,
						DisplayName: "Alpha Name",
						DownloadURL: "https://example.invalid/alpha.jar",
						FileDate:    time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
					},
					{
						ProjectID:   2,
						Fingerprint: 202,
						DisplayName: "Bravo Name",
						DownloadURL: "https://example.invalid/bravo.jar",
						FileDate:    time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC),
					},
				},
			}, nil
		},
	}

	outcome, err := lookupCurseforgeWithUpdates(context.Background(), candidates, deps, state)
	assert.NoError(t, err)

	assert.Equal(t, scanItemStatusRecognized, state.items[index["alpha.jar"]].Status)
	assert.Equal(t, scanItemStatusRecognized, state.items[index["bravo.jar"]].Status)
	require.Len(t, outcome.matches, 2)
	names := map[string]string{}
	for _, match := range outcome.matches {
		names[match.FileName] = match.Name
	}
	assert.Equal(t, "Alpha Name", names["alpha.jar"])
	assert.Equal(t, "Bravo Name", names["bravo.jar"])
}

func TestLookupCurseforgeWithUpdatesUsesFileNameWhenDisplayNameEmpty(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{
		{Path: "/mods/alpha.jar", FileName: "alpha.jar", Sha1: "a"},
	}
	items, index := buildScanItems(candidates)
	state := newScanExecState(items, index, scanExecSender{})

	deps := scanDeps{
		clients: platform.Clients{Curseforge: noopDoer{}},
		curseforgeFingerprint: func(string) uint32 {
			return 101
		},
		curseforgeFingerprintMatch: func(ctx context.Context, _ []uint32, _ httpclient.Doer) (*curseforge.FingerprintResult, error) {
			triggerRequestStartHook(ctx, t)
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{
					{
						ProjectID:   1,
						Fingerprint: 101,
						DisplayName: "",
						FileName:    "Fallback Name",
						DownloadURL: "https://example.invalid/alpha.jar",
						FileDate:    time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
					},
				},
			}, nil
		},
	}

	outcome, err := lookupCurseforgeWithUpdates(context.Background(), candidates, deps, state)
	assert.NoError(t, err)

	assert.Equal(t, scanItemStatusRecognized, state.items[index["alpha.jar"]].Status)
	require.Len(t, outcome.matches, 1)
	assert.Equal(t, "Fallback Name", outcome.matches[0].Name)
}

func TestLookupCurseforgeWithUpdatesUsesProjectIDWhenNamesEmpty(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{
		{Path: "/mods/alpha.jar", FileName: "alpha.jar", Sha1: "a"},
	}
	items, index := buildScanItems(candidates)
	state := newScanExecState(items, index, scanExecSender{})

	deps := scanDeps{
		clients: platform.Clients{Curseforge: noopDoer{}},
		curseforgeFingerprint: func(string) uint32 {
			return 101
		},
		curseforgeFingerprintMatch: func(ctx context.Context, _ []uint32, _ httpclient.Doer) (*curseforge.FingerprintResult, error) {
			triggerRequestStartHook(ctx, t)
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{
					{
						ProjectID:   42,
						Fingerprint: 101,
						DisplayName: " ",
						FileName:    "",
						DownloadURL: "https://example.invalid/alpha.jar",
						FileDate:    time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
					},
				},
			}, nil
		},
	}

	outcome, err := lookupCurseforgeWithUpdates(context.Background(), candidates, deps, state)
	assert.NoError(t, err)

	assert.Equal(t, scanItemStatusRecognized, state.items[index["alpha.jar"]].Status)
	require.Len(t, outcome.matches, 1)
	assert.Equal(t, "42", outcome.matches[0].Name)
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

func TestMarkScanCandidatesActiveSkipsNilState(t *testing.T) {
	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar"}}

	assert.NotPanics(t, func() {
		markScanCandidatesActive(nil, candidates)
	})
}

func TestMarkScanCandidatesActiveSkipsEmptyCandidates(t *testing.T) {
	items := []scanItem{{FileName: "a.jar", Status: scanItemStatusPending}}
	index := scanIndexByFile(items)
	state := newScanExecState(items, index, scanExecSender{})

	markScanCandidatesActive(state, nil)

	assert.Equal(t, scanItemStatusPending, state.items[index["a.jar"]].Status)
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
			modrinthVersionForSha: func(ctx context.Context, _ string, _ httpclient.Doer) (*modrinth.Version, error) {
				triggerRequestStartHook(ctx, t)
				return nil, errors.New("boom")
			},
			curseforgeFingerprint: func(string) uint32 { return 101 },
			curseforgeFingerprintMatch: func(ctx context.Context, _ []uint32, _ httpclient.Doer) (*curseforge.FingerprintResult, error) {
				triggerRequestStartHook(ctx, t)
				return &curseforge.FingerprintResult{
					Matches: []curseforge.File{
						{
							ProjectID:   42,
							Fingerprint: 101,
							DisplayName: "CurseForge Mod",
							DownloadURL: "https://example.invalid/cf.jar",
							FileDate:    time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
						},
					},
				}, nil
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
			modrinthVersionForSha: func(ctx context.Context, _ string, _ httpclient.Doer) (*modrinth.Version, error) {
				triggerRequestStartHook(ctx, t)
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
			modrinthVersionForSha: func(ctx context.Context, _ string, _ httpclient.Doer) (*modrinth.Version, error) {
				triggerRequestStartHook(ctx, t)
				return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup("a", modrinth.SHA1)}
			},
			curseforgeFingerprint: func(string) uint32 { return 101 },
			curseforgeFingerprintMatch: func(ctx context.Context, _ []uint32, _ httpclient.Doer) (*curseforge.FingerprintResult, error) {
				triggerRequestStartHook(ctx, t)
				return nil, errors.New("boom")
			},
		},
	}

	outcome := runScanExecution(context.Background(), input, scanExecSender{})
	assert.ErrorIs(t, outcome.err, writeErr)
}

func triggerRequestStartHook(ctx context.Context, testingContext *testing.T) {
	testingContext.Helper()
	hook := httpclient.RequestStartHookFromContext(ctx)
	if hook == nil {
		return
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.invalid/hook", nil)
	require.NoError(testingContext, err)
	hook(request)
}

func waitForScanUpdateStatus(testingContext *testing.T, updates <-chan tea.Msg, status scanItemStatus) string {
	testingContext.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case msg := <-updates:
			update, ok := msg.(scanItemUpdateMsg)
			if !ok {
				continue
			}
			if update.status == status {
				return update.key
			}
		case <-deadline:
			testingContext.Fatalf("timed out waiting for %v scan update", status)
		}
	}
}
