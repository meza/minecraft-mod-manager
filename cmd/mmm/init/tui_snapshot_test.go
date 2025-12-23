package init

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
)

const mockLatestVersion = "1.21.1"

func TestInitTUIStateSnapshots(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	t.Run("loader", func(t *testing.T) {
		enablePerf(t)
		model := newSnapshotModel(t)
		model = applyWindowSize(t, model, 60)

		matchSnapshot(t, model.View())
		spans := finalizePerfForModel(t, &model)
		assertPerfEventExistsInit(t, spans, "tui.init.session", "tui.init.state.enter")
	})

	t.Run("game_version", func(t *testing.T) {
		enablePerf(t)
		model := newSnapshotModel(t)
		model = applyWindowSize(t, model, 60)
		model = selectLoader(t, model, models.FABRIC)
		model.gameVersionQuestion.input.SetValue(mockLatestVersion)

		matchSnapshot(t, model.View())
		spans := finalizePerfForModel(t, &model)
		assertPerfEventExistsInit(t, spans, "tui.init.session", "tui.init.action.select_loader")
	})

	t.Run("release_types", func(t *testing.T) {
		enablePerf(t)
		model := newSnapshotModel(t)
		model = applyWindowSize(t, model, 60)
		model = selectLoader(t, model, models.FABRIC)
		model = enterGameVersion(t, model, "1.21.1")
		model = applyWindowSize(t, model, 60)

		matchSnapshot(t, model.View())
		spans := finalizePerfForModel(t, &model)
		assertPerfEventExistsInit(t, spans, "tui.init.session", "tui.init.action.select_game_version")
	})

	t.Run("mods_folder", func(t *testing.T) {
		enablePerf(t)
		model := newSnapshotModel(t)
		model = applyWindowSize(t, model, 60)
		model = selectLoader(t, model, models.FABRIC)
		model = enterGameVersion(t, model, "1.21.1")
		model = applyWindowSize(t, model, 60)
		model = confirmReleaseTypes(t, model)
		model.modsFolderQuestion.input.SetValue("mods")

		matchSnapshot(t, model.View())
		spans := finalizePerfForModel(t, &model)
		assertPerfEventExistsInit(t, spans, "tui.init.session", "tui.init.action.select_release_types")
	})

	t.Run("done", func(t *testing.T) {
		enablePerf(t)
		model := newSnapshotModel(t)
		model = applyWindowSize(t, model, 60)
		model = selectLoader(t, model, models.FABRIC)
		model = enterGameVersion(t, model, "1.21.1")
		model = applyWindowSize(t, model, 60)
		model = confirmReleaseTypes(t, model)
		model = enterModsFolder(t, model, "mods")

		matchSnapshot(t, model.View())
		spans := finalizePerfForModel(t, &model)
		assertPerfEventExistsInit(t, spans, "tui.init.session", "tui.init.outcome.completed")
		assertPerfSpanExistsInit(t, spans, "tui.init.wait.loader")
		assertPerfSpanExistsInit(t, spans, "tui.init.wait.game_version")
	})
}

func TestInitTUIErrorSnapshots(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	t.Run("game_version_invalid", func(t *testing.T) {
		enablePerf(t)
		model := newSnapshotModel(t)
		model = applyWindowSize(t, model, 60)
		model = selectLoader(t, model, models.FABRIC)

		model.gameVersionQuestion.input.SetValue("invalid")
		updated, cmd := model.gameVersionQuestion.Update(tea.KeyMsg{Type: tea.KeyEnter})
		model.gameVersionQuestion = updated
		model = runCmd(t, model, cmd)

		matchSnapshot(t, model.View())
		spans := finalizePerfForModel(t, &model)
		assertPerfEventExistsInit(t, spans, "tui.init.session", "tui.init.action.select_loader")
	})

	t.Run("release_types_empty", func(t *testing.T) {
		enablePerf(t)
		model := newSnapshotModel(t)
		model = applyWindowSize(t, model, 60)
		model = selectLoader(t, model, models.FABRIC)
		model = enterGameVersion(t, model, "1.21.1")
		model = applyWindowSize(t, model, 60)

		// Clear defaults so enter triggers error
		model.releaseTypesQuestion.selected = map[models.ReleaseType]bool{}

		updated, cmd := model.releaseTypesQuestion.Update(tea.KeyMsg{Type: tea.KeyEnter})
		model.releaseTypesQuestion = updated
		model = runCmd(t, model, cmd)

		matchSnapshot(t, model.View())
		spans := finalizePerfForModel(t, &model)
		assertPerfEventExistsInit(t, spans, "tui.init.session", "tui.init.state.enter")
	})

	t.Run("mods_folder_missing", func(t *testing.T) {
		enablePerf(t)
		model := newSnapshotModelWithOptions(t, initOptions{
			ModsFolder: "missing",
		}, false)
		model = applyWindowSize(t, model, 60)
		model = selectLoader(t, model, models.FABRIC)
		model = enterGameVersion(t, model, "1.21.1")
		model = applyWindowSize(t, model, 60)
		model = confirmReleaseTypes(t, model)

		updated, cmd := model.modsFolderQuestion.Update(tea.KeyMsg{Type: tea.KeyEnter})
		model.modsFolderQuestion = updated
		model = runCmd(t, model, cmd)

		matchSnapshot(t, model.View())
		spans := finalizePerfForModel(t, &model)
		assertPerfEventExistsInit(t, spans, "tui.init.session", "tui.init.state.enter")
	})
}

func TestInitTUIQuietNoTTYSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	// Quiet mode should bypass TUI entirely.
	model := newSnapshotModelWithOptions(t, initOptions{
		Quiet: true,
	}, true)

	matchSnapshot(t, model.View())
}

func newSnapshotModel(t *testing.T) CommandModel {
	return newSnapshotModelWithOptions(t, initOptions{}, true)
}

func newSnapshotModelWithOptions(t *testing.T, options initOptions, createModsFolder bool) CommandModel {
	t.Helper()

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	if createModsFolder {
		mods := options.ModsFolder
		if mods == "" {
			mods = "mods"
		}
		assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(models.ModsJSON{ModsFolder: mods}), 0755))
	}

	deps := initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{mockLatestVersion, "1.20.4", "1.19.4"}),
	}

	if options.ConfigPath == "" {
		options.ConfigPath = meta.ConfigPath
	}
	if options.ModsFolder == "" {
		options.ModsFolder = "mods"
	}
	if options.ReleaseTypes == nil {
		options.ReleaseTypes = []models.ReleaseType{models.Release}
	}

	ctx, span := perf.StartSpan(context.Background(), "tui.init.session")
	model := NewModel(ctx, span, options, deps, meta)

	return *model
}

func applyWindowSize(t *testing.T, model CommandModel, width int) CommandModel {
	t.Helper()
	next, _ := model.Update(tea.WindowSizeMsg{Width: width})
	return ensureModel(t, next)
}

func selectLoader(t *testing.T, model CommandModel, loader models.Loader) CommandModel {
	t.Helper()

	for idx, item := range model.loaderQuestion.list.Items() {
		if candidate, ok := item.(loaderType); ok && string(candidate) == loader.String() {
			model.loaderQuestion.list.Select(idx)
			break
		}
	}

	var cmd tea.Cmd
	model.loaderQuestion, cmd = model.loaderQuestion.Update(tea.KeyMsg{Type: tea.KeyEnter})

	return runCmd(t, model, cmd)
}

func TestInitTUIHidesProvidedGameVersion(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(models.ModsJSON{ModsFolder: "mods"}), 0755))

	deps := initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{mockLatestVersion, "1.20.4"}),
	}

	model := NewModel(context.Background(), nil, initOptions{
		ConfigPath:  meta.ConfigPath,
		ModsFolder:  "mods",
		GameVersion: mockLatestVersion,
		Provided: providedFlags{
			GameVersion: true,
		},
	}, deps, meta)

	current := applyWindowSize(t, *model, 60)
	current = selectLoader(t, current, models.FABRIC)

	assert.NotContains(t, current.View(), "What exact Minecraft version")

	current = confirmReleaseTypes(t, current)
	current = enterModsFolder(t, current, "mods")

	assert.NotContains(t, current.View(), "What exact Minecraft version")
}

func TestModsFolderPlaceholderUsesResolvedPath(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(models.ModsJSON{ModsFolder: "mods"}), 0755))

	model := NewModsFolderModel("mods", meta, fs, false)
	assert.Equal(t, "mods", model.input.Placeholder)
	assert.GreaterOrEqual(t, model.input.Width, len(meta.ModsFolderPath(models.ModsJSON{ModsFolder: "mods"})))
}

func enterGameVersion(t *testing.T, model CommandModel, version string) CommandModel {
	t.Helper()

	model.gameVersionQuestion.input.SetValue(version)
	updated, cmd := model.gameVersionQuestion.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model.gameVersionQuestion = updated

	return runCmd(t, model, cmd)
}

func confirmReleaseTypes(t *testing.T, model CommandModel) CommandModel {
	t.Helper()

	updated, cmd := model.releaseTypesQuestion.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model.releaseTypesQuestion = updated

	return runCmd(t, model, cmd)
}

func enterModsFolder(t *testing.T, model CommandModel, modsFolder string) CommandModel {
	t.Helper()

	model.modsFolderQuestion.input.SetValue(modsFolder)
	updated, cmd := model.modsFolderQuestion.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model.modsFolderQuestion = updated

	return runCmd(t, model, cmd)
}

func runCmd(t *testing.T, model CommandModel, cmd tea.Cmd) CommandModel {
	t.Helper()

	if cmd == nil {
		return model
	}

	msg := cmd()
	if msg == nil {
		return model
	}

	next, _ := model.Update(msg)
	return ensureModel(t, next)
}

func ensureModel(t *testing.T, model tea.Model) CommandModel {
	t.Helper()

	if typed, ok := model.(CommandModel); ok {
		return typed
	}

	if typedPtr, ok := model.(*CommandModel); ok {
		return *typedPtr
	}

	t.Fatalf("unexpected model type %T", model)
	return CommandModel{}
}

func matchSnapshot(t *testing.T, content string) {
	t.Helper()
	snaps.MatchSnapshot(t, normalizeSnapshot(content))
}

func normalizeSnapshot(content string) string {
	// Normalize path separators for cross-platform stability.
	return strings.ReplaceAll(content, "\\", "/")
}

func enablePerf(t *testing.T) {
	t.Helper()
	perf.Reset()
	t.Cleanup(perf.Reset)
	assert.NoError(t, perf.Init(perf.Config{Enabled: true}))
}

func finalizePerfForModel(t *testing.T, model *CommandModel) []perf.SpanSnapshot {
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

func assertPerfSpanExistsInit(t *testing.T, spans []perf.SpanSnapshot, name string) {
	t.Helper()
	_, ok := perf.FindSpanByName(spans, name)
	assert.True(t, ok, "expected span %q", name)
}

func assertPerfEventExistsInit(t *testing.T, spans []perf.SpanSnapshot, sessionSpanName string, eventName string) {
	t.Helper()

	session, ok := perf.FindSpanByName(spans, sessionSpanName)
	assert.True(t, ok, "expected session span %q", sessionSpanName)

	for _, e := range session.Events {
		if e.Name == eventName {
			return
		}
	}
	t.Fatalf("expected event %q on span %q", eventName, sessionSpanName)
}
