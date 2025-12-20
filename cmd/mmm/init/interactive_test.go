package init

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
)

func TestRunInteractiveInitWithLaunchFlagReturnsWithoutTUIWhenDone(t *testing.T) {
	initPerf(t)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	assert.NoError(t, fs.MkdirAll("/cfg/mods", 0755))

	options := initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}
	deps := initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}

	updated, launched, err := runInteractiveInitWithLaunchFlag(context.Background(), &cobra.Command{}, options, deps, meta)
	assert.NoError(t, err)
	assert.False(t, launched)
	assert.Equal(t, options, updated)
}

func TestRunInteractiveInitWithLaunchFlagRunTeaError(t *testing.T) {
	initPerf(t)

	deps := initDeps{
		fs:              afero.NewMemMapFs(),
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, errors.New("boom")
		},
	}
	_, launched, err := runInteractiveInitWithLaunchFlag(context.Background(), &cobra.Command{}, initOptions{ModsFolder: "mods"}, deps, config.NewMetadata("/cfg/modlist.json"))
	assert.True(t, launched)
	assert.Error(t, err)
}

func TestFinalizeInteractiveResultUnexpectedModel(t *testing.T) {
	initPerf(t)

	_, err := finalizeInteractiveResult(fakeTeaModel{})
	assert.ErrorContains(t, err, "interactive init failed")
}

func TestFinalizeInteractiveResultModelError(t *testing.T) {
	initPerf(t)

	_, err := finalizeInteractiveResult(CommandModel{err: errors.New("invalid")})
	assert.ErrorContains(t, err, "invalid")
}

func TestFinalizeInteractiveResultCancelled(t *testing.T) {
	initPerf(t)

	_, err := finalizeInteractiveResult(CommandModel{state: stateLoader})
	assert.ErrorContains(t, err, "init cancelled")
}

func TestFinalizeInteractiveResultSuccess(t *testing.T) {
	initPerf(t)

	expected := initOptions{
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
	}

	result, err := finalizeInteractiveResult(&CommandModel{state: done, result: expected})
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func initPerf(t *testing.T) {
	t.Helper()
	perf.Reset()
	t.Cleanup(perf.Reset)
	assert.NoError(t, perf.Init(perf.Config{Enabled: true}))
}

type fakeTeaModel struct{}

func (fakeTeaModel) Init() tea.Cmd { return nil }

func (fakeTeaModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return fakeTeaModel{}, nil }

func (fakeTeaModel) View() string { return "fake" }
