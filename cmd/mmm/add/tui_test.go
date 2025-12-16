package add

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

func TestAddTUIStateSnapshots(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}

	t.Run("unknown_platform_select", func(t *testing.T) {
		ctx, span := startAddTUIPerf(t)
		model := newAddTUIModel(ctx, span, addTUIStateUnknownPlatformSelect, models.Platform("invalid"), "abc", cfg, nil)
		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(60, 20))
		waitForOutput(t, tm, "cmd.add.tui.unknown_platform")
		_ = tm.Quit()
		final := ensureAddTUIModel(t, tm.FinalModel(t))
		matchSnapshot(t, final.View())
		spans := finalizeAddTUIPerf(t, &final)
		assertPerfEventExistsInTUI(t, spans, "tui.add.session", "tui.add.state.enter")
	})

	t.Run("mod_not_found_confirm", func(t *testing.T) {
		ctx, span := startAddTUIPerf(t)
		model := newAddTUIModel(ctx, span, addTUIStateModNotFoundConfirm, models.MODRINTH, "abc", cfg, nil)
		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(60, 20))
		waitForOutput(t, tm, "cmd.add.tui.mod_not_found")
		_ = tm.Quit()
		final := ensureAddTUIModel(t, tm.FinalModel(t))
		matchSnapshot(t, final.View())
		spans := finalizeAddTUIPerf(t, &final)
		assertPerfEventExistsInTUI(t, spans, "tui.add.session", "tui.add.state.enter")
	})

	t.Run("mod_not_found_select_platform", func(t *testing.T) {
		ctx, span := startAddTUIPerf(t)
		model := newAddTUIModel(ctx, span, addTUIStateModNotFoundConfirm, models.MODRINTH, "abc", cfg, nil)
		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(60, 20))
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		waitForOutput(t, tm, "cmd.add.tui.choose_platform")
		_ = tm.Quit()
		final := ensureAddTUIModel(t, tm.FinalModel(t))
		matchSnapshot(t, final.View())
		spans := finalizeAddTUIPerf(t, &final)
		assertPerfEventExistsInTUI(t, spans, "tui.add.session", "tui.add.action.confirm_yes")
		assertPerfEventExistsInTUI(t, spans, "tui.add.session", "tui.add.state.enter")
	})

	t.Run("mod_not_found_enter_project_id", func(t *testing.T) {
		ctx, span := startAddTUIPerf(t)
		model := newAddTUIModel(ctx, span, addTUIStateModNotFoundConfirm, models.MODRINTH, "abc", cfg, nil)
		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(60, 20))
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		waitForOutput(t, tm, "cmd.add.tui.choose_platform")
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		waitForOutput(t, tm, "cmd.add.tui.enter_project_id")
		_ = tm.Quit()
		final := ensureAddTUIModel(t, tm.FinalModel(t))
		matchSnapshot(t, final.View())
		spans := finalizeAddTUIPerf(t, &final)
		assertPerfEventExistsInTUI(t, spans, "tui.add.session", "tui.add.action.confirm_yes")
		assertPerfEventExistsInTUI(t, spans, "tui.add.session", "tui.add.action.select_platform")
	})

	t.Run("no_file_confirm", func(t *testing.T) {
		ctx, span := startAddTUIPerf(t)
		model := newAddTUIModel(ctx, span, addTUIStateNoFileConfirm, models.MODRINTH, "abc", cfg, nil)
		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(60, 20))
		waitForOutput(t, tm, "cmd.add.tui.no_file_found")
		_ = tm.Quit()
		final := ensureAddTUIModel(t, tm.FinalModel(t))
		matchSnapshot(t, final.View())
		spans := finalizeAddTUIPerf(t, &final)
		assertPerfEventExistsInTUI(t, spans, "tui.add.session", "tui.add.state.enter")
	})

	t.Run("no_file_enter_project_id", func(t *testing.T) {
		ctx, span := startAddTUIPerf(t)
		model := newAddTUIModel(ctx, span, addTUIStateNoFileConfirm, models.MODRINTH, "abc", cfg, nil)
		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(60, 20))
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		waitForOutput(t, tm, "cmd.add.tui.enter_project_id_on")
		_ = tm.Quit()
		final := ensureAddTUIModel(t, tm.FinalModel(t))
		matchSnapshot(t, final.View())
		spans := finalizeAddTUIPerf(t, &final)
		assertPerfEventExistsInTUI(t, spans, "tui.add.session", "tui.add.action.confirm_yes")
	})

	t.Run("fatal_error", func(t *testing.T) {
		ctx, span := startAddTUIPerf(t)
		model := newAddTUIModel(ctx, span, addTUIStateUnknownPlatformSelect, models.Platform("invalid"), "abc", cfg, func(p models.Platform, id string) tea.Cmd {
			return func() tea.Msg {
				return addTUIFetchResultMsg{
					platform:  p,
					projectID: id,
					err:       errors.New("boom"),
				}
			}
		})

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(60, 20))
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // select default and retry
		waitForOutput(t, tm, "boom")
		tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
		final := ensureAddTUIModel(t, tm.FinalModel(t))
		matchSnapshot(t, final.View())
		spans := finalizeAddTUIPerf(t, &final)
		assertPerfEventExistsInTUI(t, spans, "tui.add.session", "tui.add.action.select_platform")
		assertPerfSpanExistsInTUI(t, spans, "tui.add.fetch")
	})

	t.Run("done", func(t *testing.T) {
		ctx, span := startAddTUIPerf(t)
		model := newAddTUIModel(ctx, span, addTUIStateUnknownPlatformSelect, models.Platform("invalid"), "abc", cfg, func(p models.Platform, id string) tea.Cmd {
			return func() tea.Msg {
				return addTUIFetchResultMsg{
					platform:  p,
					projectID: id,
					remote: platform.RemoteMod{
						Name:        "Example",
						FileName:    "example.jar",
						Hash:        "hash",
						ReleaseDate: "2024-01-01T00:00:00Z",
						DownloadURL: "https://example.com/example.jar",
					},
				}
			}
		})

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(60, 20))
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
		final := ensureAddTUIModel(t, tm.FinalModel(t))
		matchSnapshot(t, final.View())
		spans := finalizeAddTUIPerf(t, &final)
		assertPerfEventExistsInTUI(t, spans, "tui.add.session", "tui.add.outcome.resolved")
		assertPerfSpanExistsInTUI(t, spans, "tui.add.fetch")
	})

	t.Run("aborted", func(t *testing.T) {
		ctx, span := startAddTUIPerf(t)
		model := newAddTUIModel(ctx, span, addTUIStateModNotFoundConfirm, models.MODRINTH, "abc", cfg, nil)
		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(60, 20))
		tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
		tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
		final := ensureAddTUIModel(t, tm.FinalModel(t))
		matchSnapshot(t, final.View())
		spans := finalizeAddTUIPerf(t, &final)
		assertPerfEventExistsInTUI(t, spans, "tui.add.session", "tui.add.action.abort")
	})
}

func TestAddTUIThinkingTime_RecordsWaitRegions(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	ctx, span := startAddTUIPerf(t)

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}

	model := newAddTUIModel(ctx, span, addTUIStateUnknownPlatformSelect, models.Platform("invalid"), "abc", cfg, func(p models.Platform, id string) tea.Cmd {
		return func() tea.Msg {
			return addTUIFetchResultMsg{
				platform:  p,
				projectID: id,
				remote: platform.RemoteMod{
					Name:        "Example",
					FileName:    "example.jar",
					Hash:        "hash",
					ReleaseDate: "2024-01-01T00:00:00Z",
					DownloadURL: "https://example.com/example.jar",
				},
			}
		}
	})

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(60, 20))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	final := ensureAddTUIModel(t, tm.FinalModel(t))
	spans := finalizeAddTUIPerf(t, &final)
	assertPerfSpanExistsInTUI(t, spans, "tui.add.wait.unknown_platform_select")
}

func waitForOutput(t *testing.T, tm *teatest.TestModel, contains string) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), contains)
	}, teatest.WithDuration(2*time.Second))
}

func ensureAddTUIModel(t *testing.T, model tea.Model) addTUIModel {
	t.Helper()
	typed, ok := model.(addTUIModel)
	if !ok {
		t.Fatalf("unexpected model type %T", model)
	}
	return typed
}

func matchSnapshot(t *testing.T, content string) {
	t.Helper()
	snaps.MatchSnapshot(t, normalizeSnapshot(content))
}

func normalizeSnapshot(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return content
}

func startAddTUIPerf(t *testing.T) (context.Context, *perf.Span) {
	t.Helper()
	perf.Reset()
	t.Cleanup(perf.Reset)
	assert.NoError(t, perf.Init(perf.Config{Enabled: true}))
	ctx, span := perf.StartSpan(context.Background(), "tui.add.session")
	return ctx, span
}

func finalizeAddTUIPerf(t *testing.T, model *addTUIModel) []perf.SpanSnapshot {
	t.Helper()
	if model != nil {
		model.endWait("snapshot")
		if model.sessionSpan != nil {
			model.sessionSpan.End()
		}
	}
	spans, err := perf.GetSpans()
	assert.NoError(t, err)
	return spans
}

func assertPerfSpanExistsInTUI(t *testing.T, spans []perf.SpanSnapshot, name string) {
	t.Helper()
	_, ok := perf.FindSpanByName(spans, name)
	assert.True(t, ok, "expected span %q", name)
}

func assertPerfEventExistsInTUI(t *testing.T, spans []perf.SpanSnapshot, spanName string, eventName string) {
	t.Helper()
	span, ok := perf.FindSpanByName(spans, spanName)
	assert.True(t, ok, "expected span %q", spanName)
	for _, e := range span.Events {
		if e.Name == eventName {
			return
		}
	}
	t.Fatalf("expected event %q on span %q", eventName, spanName)
}
