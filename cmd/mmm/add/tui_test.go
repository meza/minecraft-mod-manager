package add

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/gkampitakis/go-snaps/snaps"

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
		perf.ClearPerformanceLog()
		model := newAddTUIModel(addTUIStateUnknownPlatformSelect, models.Platform("invalid"), "abc", cfg, nil)
		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(60, 20))
		waitForOutput(t, tm, "cmd.add.tui.unknown_platform")
		_ = tm.Quit()
		final := ensureAddTUIModel(t, tm.FinalModel(t))
		matchSnapshot(t, final.View())
		assertPerfMarkExistsInTUILog(t, "tui.add.state.enter")
	})

	t.Run("mod_not_found_confirm", func(t *testing.T) {
		perf.ClearPerformanceLog()
		model := newAddTUIModel(addTUIStateModNotFoundConfirm, models.MODRINTH, "abc", cfg, nil)
		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(60, 20))
		waitForOutput(t, tm, "cmd.add.tui.mod_not_found")
		_ = tm.Quit()
		final := ensureAddTUIModel(t, tm.FinalModel(t))
		matchSnapshot(t, final.View())
		assertPerfMarkExistsInTUILog(t, "tui.add.state.enter")
	})

	t.Run("mod_not_found_select_platform", func(t *testing.T) {
		perf.ClearPerformanceLog()
		model := newAddTUIModel(addTUIStateModNotFoundConfirm, models.MODRINTH, "abc", cfg, nil)
		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(60, 20))
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		waitForOutput(t, tm, "cmd.add.tui.choose_platform")
		_ = tm.Quit()
		final := ensureAddTUIModel(t, tm.FinalModel(t))
		matchSnapshot(t, final.View())
		assertPerfMarkExistsInTUILog(t, "tui.add.action.confirm_yes")
		assertPerfMarkExistsInTUILog(t, "tui.add.state.enter")
	})

	t.Run("mod_not_found_enter_project_id", func(t *testing.T) {
		perf.ClearPerformanceLog()
		model := newAddTUIModel(addTUIStateModNotFoundConfirm, models.MODRINTH, "abc", cfg, nil)
		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(60, 20))
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		waitForOutput(t, tm, "cmd.add.tui.choose_platform")
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		waitForOutput(t, tm, "cmd.add.tui.enter_project_id")
		_ = tm.Quit()
		final := ensureAddTUIModel(t, tm.FinalModel(t))
		matchSnapshot(t, final.View())
		assertPerfMarkExistsInTUILog(t, "tui.add.action.confirm_yes")
		assertPerfMarkExistsInTUILog(t, "tui.add.action.select_platform")
	})

	t.Run("no_file_confirm", func(t *testing.T) {
		perf.ClearPerformanceLog()
		model := newAddTUIModel(addTUIStateNoFileConfirm, models.MODRINTH, "abc", cfg, nil)
		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(60, 20))
		waitForOutput(t, tm, "cmd.add.tui.no_file_found")
		_ = tm.Quit()
		final := ensureAddTUIModel(t, tm.FinalModel(t))
		matchSnapshot(t, final.View())
		assertPerfMarkExistsInTUILog(t, "tui.add.state.enter")
	})

	t.Run("no_file_enter_project_id", func(t *testing.T) {
		perf.ClearPerformanceLog()
		model := newAddTUIModel(addTUIStateNoFileConfirm, models.MODRINTH, "abc", cfg, nil)
		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(60, 20))
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		waitForOutput(t, tm, "cmd.add.tui.enter_project_id_on")
		_ = tm.Quit()
		final := ensureAddTUIModel(t, tm.FinalModel(t))
		matchSnapshot(t, final.View())
		assertPerfMarkExistsInTUILog(t, "tui.add.action.confirm_yes")
	})

	t.Run("fatal_error", func(t *testing.T) {
		perf.ClearPerformanceLog()
		model := newAddTUIModel(addTUIStateUnknownPlatformSelect, models.Platform("invalid"), "abc", cfg, func(p models.Platform, id string) tea.Cmd {
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
		assertPerfMarkExistsInTUILog(t, "tui.add.action.select_platform")
		assertPerfRegionExists(t, "tui.add.fetch")
	})

	t.Run("done", func(t *testing.T) {
		perf.ClearPerformanceLog()
		model := newAddTUIModel(addTUIStateUnknownPlatformSelect, models.Platform("invalid"), "abc", cfg, func(p models.Platform, id string) tea.Cmd {
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
		assertPerfMarkExistsInTUILog(t, "tui.add.outcome.resolved")
		assertPerfRegionExists(t, "tui.add.fetch")
	})

	t.Run("aborted", func(t *testing.T) {
		perf.ClearPerformanceLog()
		model := newAddTUIModel(addTUIStateModNotFoundConfirm, models.MODRINTH, "abc", cfg, nil)
		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(60, 20))
		tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
		tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
		final := ensureAddTUIModel(t, tm.FinalModel(t))
		matchSnapshot(t, final.View())
		assertPerfMarkExistsInTUILog(t, "tui.add.action.abort")
	})
}

func TestAddTUIThinkingTime_RecordsWaitRegions(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	perf.ClearPerformanceLog()

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}

	model := newAddTUIModel(addTUIStateUnknownPlatformSelect, models.Platform("invalid"), "abc", cfg, func(p models.Platform, id string) tea.Cmd {
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

	assertPerfRegionExists(t, "tui.add.wait.unknown_platform_select")
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

func assertPerfMarkExistsInTUILog(t *testing.T, name string) {
	t.Helper()
	for _, entry := range perf.GetPerformanceLog() {
		if entry.Type == perf.MarkType && entry.Name == name {
			return
		}
	}
	t.Fatalf("expected perf mark %q not found", name)
}

func assertPerfRegionExists(t *testing.T, name string) {
	t.Helper()
	hasStart := false
	hasEnd := false
	hasDuration := false
	for _, entry := range perf.GetPerformanceLog() {
		if entry.Type == perf.MarkType && entry.Name == name {
			hasStart = true
		}
		if entry.Type == perf.MarkType && entry.Name == name+"-end" {
			hasEnd = true
		}
		if entry.Type == perf.MeasureType && entry.Name == name+"-duration" {
			hasDuration = true
		}
	}
	if hasStart && hasEnd && hasDuration {
		return
	}
	t.Fatalf("expected perf region %q not fully recorded (start=%v end=%v duration=%v)", name, hasStart, hasEnd, hasDuration)
}
