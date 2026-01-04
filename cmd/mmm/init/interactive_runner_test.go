package init

import (
	"bytes"
	"context"
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

type fakeTerminalReader struct {
	bytes.Buffer
}

func (reader *fakeTerminalReader) Fd() uintptr {
	return 0
}

type fakeTerminalWriter struct {
	bytes.Buffer
}

func (writer *fakeTerminalWriter) Fd() uintptr {
	return 1
}

func TestRunInteractiveInitRunsInitFlow(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	minecraft.ClearManifestCache()

	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(models.ModsJSON{ModsFolder: "mods"}), 0755))

	runTeaCalled := false
	runTea := func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
		runTeaCalled = true
		return CommandModel{
			state: done,
			result: initOptions{
				ConfigPath:   meta.ConfigPath,
				Loader:       models.FABRIC,
				GameVersion:  "1.21.1",
				ReleaseTypes: []models.ReleaseType{models.Release},
				ModsFolder:   "mods",
				Provided:     providedFlags{Loader: true, GameVersion: true, ReleaseTypes: true, ModsFolder: true},
				Unattended:   false,
			},
		}, nil
	}

	command := &cobra.Command{}
	command.SetIn(&fakeTerminalReader{})
	command.SetOut(&fakeTerminalWriter{})
	command.SetErr(&fakeTerminalWriter{})

	out := output.New(io.Discard, io.Discard, false)
	err := RunInteractiveInit(context.Background(), command, InteractiveInitDeps{
		FS:              fs,
		Output:          out,
		MinecraftClient: manifestDoer([]string{"1.21.1"}),
		RunTea:          runTea,
	}, InteractiveInitOptions{
		ConfigPath: meta.ConfigPath,
	})

	assert.NoError(t, err)
	assert.True(t, runTeaCalled)
	loadedConfig, configErr := config.ReadConfig(context.Background(), fs, meta)
	assert.NoError(t, configErr)
	assert.Equal(t, "mods", loadedConfig.ModsFolder)
	_, lockErr := config.ReadLock(context.Background(), fs, meta)
	assert.NoError(t, lockErr)
}
