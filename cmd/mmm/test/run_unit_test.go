package test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/locksync"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestRunInteractiveTestReturnsOutcome(t *testing.T) {
	originalRunner := runTestProgram
	t.Cleanup(func() { runTestProgram = originalRunner })
	runTestProgram = func(model *testModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = model.execRunner(model.ctx, testExecSender{})
		return model, nil
	}

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	cfg := models.ModsJSON{
		Loader: models.FABRIC,
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	items, indexByKey := buildTestItems(cfg)

	outcome, err := runInteractiveTest(context.Background(), cmd, testExecutionInput{
		targetVersion: "1.20.1",
		cfg:           cfg,
		items:         items,
		indexByKey:    indexByKey,
		deps: testDeps{
			fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
				return platform.RemoteMod{}, nil
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, outcome.items, 1)
	assert.Equal(t, testItemStatusSupported, outcome.items[0].Status)
}

func TestRunInteractiveTestMissingRunner(t *testing.T) {
	originalRunner := runTestProgram
	t.Cleanup(func() { runTestProgram = originalRunner })
	runTestProgram = nil

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	_, err := runInteractiveTest(context.Background(), cmd, testExecutionInput{
		targetVersion: "1.20.1",
		items:         []testItem{},
		indexByKey:    map[string]int{},
	})
	assert.Error(t, err)
}

func TestRunInteractiveTestReturnsRunnerError(t *testing.T) {
	originalRunner := runTestProgram
	t.Cleanup(func() { runTestProgram = originalRunner })
	runErr := errors.New("boom")
	runTestProgram = func(_ *testModel, _ ...tea.ProgramOption) (tea.Model, error) {
		return nil, runErr
	}

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	_, err := runInteractiveTest(context.Background(), cmd, testExecutionInput{
		targetVersion: "1.20.1",
		items:         []testItem{},
		indexByKey:    map[string]int{},
	})
	assert.ErrorIs(t, err, runErr)
}

func TestRunInteractiveTestReturnsOutcomeError(t *testing.T) {
	originalRunner := runTestProgram
	t.Cleanup(func() { runTestProgram = originalRunner })
	runTestProgram = func(_ *testModel, _ ...tea.ProgramOption) (tea.Model, error) {
		return fakePromptModel{}, nil
	}

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	_, err := runInteractiveTest(context.Background(), cmd, testExecutionInput{
		targetVersion: "1.20.1",
		items:         []testItem{},
		indexByKey:    map[string]int{},
	})
	assert.Error(t, err)
}

func TestRunInteractiveTestTranscriptWriteError(t *testing.T) {
	originalRunner := runTestProgram
	t.Cleanup(func() { runTestProgram = originalRunner })
	runTestProgram = func(model *testModel, _ ...tea.ProgramOption) (tea.Model, error) {
		return model, nil
	}

	writeErr := errors.New("write failed")

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	cfg := models.ModsJSON{
		Loader: models.FABRIC,
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	items, indexByKey := buildTestItems(cfg)

	_, err := runInteractiveTest(context.Background(), cmd, testExecutionInput{
		targetVersion: "1.20.1",
		cfg:           cfg,
		items:         items,
		indexByKey:    indexByKey,
		deps: testDeps{
			runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
				return nil, writeErr
			},
		},
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestRunInteractiveTestReturnsOutcomeErr(t *testing.T) {
	originalRunner := runTestProgram
	t.Cleanup(func() { runTestProgram = originalRunner })
	outcomeErr := errors.New("outcome failed")
	runTestProgram = func(model *testModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = testExecutionOutcome{err: outcomeErr}
		return model, nil
	}

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	_, err := runInteractiveTest(context.Background(), cmd, testExecutionInput{
		targetVersion: "1.20.1",
		items:         []testItem{},
		indexByKey:    map[string]int{},
		deps: testDeps{
			runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
				return model, nil
			},
		},
	})
	assert.ErrorIs(t, err, outcomeErr)
}

func TestRunTestByModeInteractive(t *testing.T) {
	originalRunner := runTestProgram
	t.Cleanup(func() { runTestProgram = originalRunner })
	runTestProgram = func(_ *testModel, _ ...tea.ProgramOption) (tea.Model, error) {
		return &testModel{outcome: testExecutionOutcome{items: []testItem{{Status: testItemStatusSupported}}}}, nil
	}

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	outcome, err := runTestByMode(context.Background(), cmd, testOptions{}, interaction.ExecutionModeInteractive, testExecutionInput{
		targetVersion: "1.20.1",
		items:         []testItem{},
		indexByKey:    map[string]int{},
	})
	require.NoError(t, err)
	require.Len(t, outcome.items, 1)
	assert.Equal(t, testItemStatusSupported, outcome.items[0].Status)
}

func TestWriteInteractiveTestTranscript(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	teaInvoked := false
	var capturedLines []string
	deps := testDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			teaInvoked = true
			if outputModel, ok := model.(view.OutputLinesModel); ok {
				capturedLines = outputModel.Lines
			}
			return model, nil
		},
	}

	model := &testModel{
		targetVersion: "1.21.1",
		colorMode:     view.ColorDisabled,
		items: []testItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, Status: testItemStatusSupported},
		},
	}

	err := writeInteractiveTestTranscript(cmd, testExecutionInput{deps: deps}, model)
	assert.NoError(t, err)
	assert.True(t, teaInvoked)
	assert.NotEmpty(t, capturedLines)
}

func TestWriteInteractiveTestTranscriptSkipsNilModel(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	teaInvoked := false
	deps := testDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			teaInvoked = true
			return model, nil
		},
	}

	err := writeInteractiveTestTranscript(cmd, testExecutionInput{deps: deps}, nil)
	assert.NoError(t, err)
	assert.False(t, teaInvoked)
}

func TestRunTestByModeQuietOutputsLines(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	deps := testDeps{
		output: output.New(out, errOut, false),
		runTea: runTeaProgram,
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.NoCompatibleFileError{Platform: models.MODRINTH, ProjectID: "alpha"}
		},
	}

	cfg := models.ModsJSON{
		Loader: models.FABRIC,
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	items, indexByKey := buildTestItems(cfg)

	outcome, err := runTestByMode(context.Background(), cmd, testOptions{Quiet: true}, interaction.ExecutionModeNonTTY, testExecutionInput{
		cfg:           cfg,
		targetVersion: "1.20.1",
		items:         items,
		indexByKey:    indexByKey,
		deps:          deps,
	})
	require.NoError(t, err)
	require.Len(t, outcome.items, 1)
	assert.Contains(t, out.String(), "Alpha (alpha) [modrinth]")
}

func TestRunQuietTestSkipsOutputWhenNoIssues(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	deps := testDeps{
		output: output.New(out, errOut, false),
		runTea: runTeaProgram,
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, nil
		},
	}

	cfg := models.ModsJSON{
		Loader: models.FABRIC,
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	items, indexByKey := buildTestItems(cfg)

	outcome, err := runQuietTest(context.Background(), cmd, testExecutionInput{
		cfg:           cfg,
		targetVersion: "1.20.1",
		items:         items,
		indexByKey:    indexByKey,
		deps:          deps,
	})
	require.NoError(t, err)
	require.Len(t, outcome.items, 1)
	assert.Empty(t, out.String())
}

func TestRunQuietTestReturnsOutcomeError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	outcome, err := runQuietTest(ctx, cmd, testExecutionInput{
		targetVersion: "1.20.1",
		items:         []testItem{},
		indexByKey:    map[string]int{},
	})
	assert.ErrorIs(t, err, context.Canceled)
	assert.ErrorIs(t, outcome.err, context.Canceled)
}

func TestRunQuietTestReturnsOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(errorWriter{err: errors.New("write failed")})

	deps := testDeps{
		runTea: runTeaProgram,
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.NoCompatibleFileError{Platform: models.MODRINTH, ProjectID: "alpha"}
		},
	}

	cfg := models.ModsJSON{
		Loader: models.FABRIC,
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	items, indexByKey := buildTestItems(cfg)

	_, err := runQuietTest(context.Background(), cmd, testExecutionInput{
		cfg:           cfg,
		targetVersion: "1.20.1",
		items:         items,
		indexByKey:    indexByKey,
		deps:          deps,
	})
	assert.Error(t, err)
}

func TestRunTestByModeNonInteractiveOutputsSections(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	deps := testDeps{
		output: output.New(out, errOut, false),
		runTea: runTeaProgram,
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, nil
		},
	}

	cfg := models.ModsJSON{
		Loader: models.FABRIC,
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	items, indexByKey := buildTestItems(cfg)

	outcome, err := runTestByMode(context.Background(), cmd, testOptions{}, interaction.ExecutionModeNonTTY, testExecutionInput{
		cfg:           cfg,
		targetVersion: "1.20.1",
		items:         items,
		indexByKey:    indexByKey,
		deps:          deps,
	})
	require.NoError(t, err)
	require.Len(t, outcome.items, 1)
	assert.Contains(t, out.String(), "Alpha (alpha) [modrinth]")
}

func TestRunNonInteractiveTestReturnsOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(errorWriter{err: errors.New("write failed")})

	deps := testDeps{
		runTea: runTeaProgram,
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, nil
		},
	}

	cfg := models.ModsJSON{
		Loader: models.FABRIC,
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	items, indexByKey := buildTestItems(cfg)

	_, err := runNonInteractiveTest(context.Background(), cmd, testExecutionInput{
		cfg:           cfg,
		targetVersion: "1.20.1",
		items:         items,
		indexByKey:    indexByKey,
		deps:          deps,
	})
	assert.Error(t, err)
}

func TestRunTestHandlesEmptyModList(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")

	cfg := models.ModsJSON{
		Loader:      models.FABRIC,
		GameVersion: "1.20.1",
		Mods:        []models.Mod{},
	}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})

	result, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.20.2",
	}, testDeps{
		fs:     fs,
		runTea: runTeaProgram,
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) {
			return true, nil
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, out.String(), "1.20.2")
}

func TestRunTestStopsWhenConfigPromptCanceled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")

	cmd := &cobra.Command{}
	in := fakeTTY{Buffer: &bytes.Buffer{}}
	out := fakeTTY{Buffer: &bytes.Buffer{}}
	cmd.SetIn(in)
	cmd.SetOut(out)

	result, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.20.2",
	}, testDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{canceled: true}, nil
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
}

func TestRunTestReturnsResolveError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		Loader:      models.FABRIC,
		GameVersion: "1.20.1",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	result, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.20.2",
	}, testDeps{
		fs: fs,
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) {
			return false, errors.New("boom")
		},
	})
	assert.Error(t, err)
	assert.Equal(t, 1, result.ExitCode)
}

func TestResolveLatestVersionUsesPrompt(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	deps := testDeps{
		minecraftClient: noopDoer{},
		latestVersion: func(context.Context, httpclient.Doer) (string, error) {
			return "", errors.New("boom")
		},
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return testVersionPromptModel{selected: "1.20.1"}, nil
		},
	}

	version, resolution, err := resolveLatestVersion(context.Background(), cmd, deps, interaction.ExecutionModeInteractive, view.ColorDisabled, "latest")
	require.NoError(t, err)
	assert.Nil(t, resolution)
	assert.Equal(t, "1.20.1", version)
}

func TestResolveLatestVersionSkipsWhenNotLatest(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	version, resolution, err := resolveLatestVersion(context.Background(), cmd, testDeps{}, interaction.ExecutionModeNonTTY, view.ColorDisabled, "1.20.1")
	require.NoError(t, err)
	assert.Nil(t, resolution)
	assert.Equal(t, "1.20.1", version)
}

func TestResolveLatestVersionUsesResolvedLatest(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	version, resolution, err := resolveLatestVersion(context.Background(), cmd, testDeps{
		latestVersion: func(context.Context, httpclient.Doer) (string, error) {
			return "1.21.1", nil
		},
	}, interaction.ExecutionModeNonTTY, view.ColorDisabled, "latest")
	require.NoError(t, err)
	assert.Nil(t, resolution)
	assert.Equal(t, "1.21.1", version)
}

func TestResolveLatestVersionNonInteractiveReturnsError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	version, resolution, err := resolveLatestVersion(context.Background(), cmd, testDeps{
		latestVersion: func(context.Context, httpclient.Doer) (string, error) {
			return "", errors.New("boom")
		},
		runTea: runTeaProgram,
	}, interaction.ExecutionModeNonTTY, view.ColorDisabled, "latest")
	assert.ErrorIs(t, err, errLatestVersionRequired)
	assert.Empty(t, version)
	require.NotNil(t, resolution)
	assert.Equal(t, 1, resolution.exitCode)
}

func TestResolveLatestVersionCanceled(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	deps := testDeps{
		minecraftClient: noopDoer{},
		latestVersion: func(context.Context, httpclient.Doer) (string, error) {
			return "", errors.New("boom")
		},
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return testVersionPromptModel{canceled: true}, nil
		},
	}

	version, resolution, err := resolveLatestVersion(context.Background(), cmd, deps, interaction.ExecutionModeInteractive, view.ColorDisabled, "latest")
	require.NoError(t, err)
	assert.Empty(t, version)
	require.NotNil(t, resolution)
	assert.False(t, resolution.shouldContinue)
}

func TestResolveLatestVersionReturnsPromptError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	runErr := errors.New("boom")
	version, resolution, err := resolveLatestVersion(context.Background(), cmd, testDeps{
		minecraftClient: noopDoer{},
		latestVersion: func(context.Context, httpclient.Doer) (string, error) {
			return "", errors.New("latest failed")
		},
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, runErr
		},
	}, interaction.ExecutionModeInteractive, view.ColorDisabled, "latest")
	assert.ErrorIs(t, err, runErr)
	assert.Empty(t, version)
	require.NotNil(t, resolution)
	assert.Equal(t, 1, resolution.exitCode)
}

func TestResolveInvalidVersionUsesPrompt(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	deps := testDeps{
		minecraftClient: noopDoer{},
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return testVersionPromptModel{selected: "1.20.2"}, nil
		},
	}

	version, resolution, err := resolveInvalidVersion(context.Background(), cmd, deps, interaction.ExecutionModeInteractive, view.ColorDisabled, "nope")
	require.NoError(t, err)
	assert.Nil(t, resolution)
	assert.Equal(t, "1.20.2", version)
}

func TestResolveInvalidVersionCanceled(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	deps := testDeps{
		minecraftClient: noopDoer{},
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return testVersionPromptModel{canceled: true}, nil
		},
	}

	version, resolution, err := resolveInvalidVersion(context.Background(), cmd, deps, interaction.ExecutionModeInteractive, view.ColorDisabled, "nope")
	require.NoError(t, err)
	assert.Empty(t, version)
	require.NotNil(t, resolution)
	assert.False(t, resolution.shouldContinue)
}

func TestResolveInvalidVersionNonInteractive(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	version, resolution, err := resolveInvalidVersion(context.Background(), cmd, testDeps{runTea: runTeaProgram}, interaction.ExecutionModeNonTTY, view.ColorDisabled, "nope")
	assert.ErrorIs(t, err, errInvalidVersion)
	assert.Empty(t, version)
	require.NotNil(t, resolution)
	assert.Equal(t, 1, resolution.exitCode)
}

func TestResolveInvalidVersionReturnsOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(errorWriter{err: errors.New("write failed")})

	version, resolution, err := resolveInvalidVersion(context.Background(), cmd, testDeps{runTea: runTeaProgram}, interaction.ExecutionModeNonTTY, view.ColorDisabled, "nope")
	assert.Error(t, err)
	assert.Empty(t, version)
	require.NotNil(t, resolution)
	assert.Equal(t, 1, resolution.exitCode)
}

func TestResolveInvalidVersionNonInteractiveColorEnabled(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	version, resolution, err := resolveInvalidVersion(context.Background(), cmd, testDeps{runTea: runTeaProgram}, interaction.ExecutionModeNonTTY, view.ColorEnabled, "nope")
	assert.ErrorIs(t, err, errInvalidVersion)
	assert.Empty(t, version)
	require.NotNil(t, resolution)
	assert.Equal(t, 1, resolution.exitCode)
}

func TestResolveInvalidVersionReturnsPromptError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	runErr := errors.New("boom")
	version, resolution, err := resolveInvalidVersion(context.Background(), cmd, testDeps{
		minecraftClient: noopDoer{},
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, runErr
		},
	}, interaction.ExecutionModeInteractive, view.ColorDisabled, "nope")
	assert.ErrorIs(t, err, runErr)
	assert.Empty(t, version)
	require.NotNil(t, resolution)
	assert.Equal(t, 1, resolution.exitCode)
}

func TestValidateTargetVersionHandlesInvalidPrompt(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	callCount := 0
	deps := testDeps{
		minecraftClient: noopDoer{},
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) {
			callCount++
			return callCount > 1, nil
		},
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return testVersionPromptModel{selected: "1.20.1"}, nil
		},
	}

	version, resolution, err := validateTargetVersion(context.Background(), cmd, deps, interaction.ExecutionModeInteractive, view.ColorDisabled, "nope")
	require.NoError(t, err)
	assert.Nil(t, resolution)
	assert.Equal(t, "1.20.1", version)
}

func TestResolutionOrErrorUsesExitCodeOnNilResolution(t *testing.T) {
	resolution, err := resolutionOrError(nil, errors.New("boom"))
	assert.Error(t, err)
	assert.Equal(t, 1, resolution.exitCode)
}

func TestTestOutcomeFromModelUnexpectedModel(t *testing.T) {
	err := testOutcomeFromModel(fakePromptModel{})
	assert.Error(t, err)
}

func TestEvaluateTestOutcomeReturnsErrorWhenOutcomeErrSet(t *testing.T) {
	sentinel := errors.New("boom")
	exitCode, err := evaluateTestOutcome(testExecutionOutcome{err: sentinel})
	assert.ErrorIs(t, err, sentinel)
	assert.Equal(t, 1, exitCode)
}

func TestToUnsupportedModsReturnsNilWhenEmpty(t *testing.T) {
	assert.Nil(t, toUnsupportedMods(nil))
}

func TestConfigMissingPromptErrorAllowsPrompting(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	in := fakeTTY{Buffer: &bytes.Buffer{}}
	out := fakeTTY{Buffer: &bytes.Buffer{}}
	cmd.SetIn(in)
	cmd.SetOut(out)

	err := configMissingPromptError(testOptions{}, cmd, config.NewMetadata("/cfg/modlist.json"))
	assert.NoError(t, err)
}

func TestConfigMissingPromptErrorReturnsUnattendedError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	err := configMissingPromptError(testOptions{Unattended: true}, cmd, config.NewMetadata("/cfg/modlist.json"))
	assert.Error(t, err)
}

func TestConfigMissingPromptErrorReturnsNoTTYError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetIn(&bytes.Buffer{})

	err := configMissingPromptError(testOptions{}, cmd, config.NewMetadata("/cfg/modlist.json"))
	assert.Error(t, err)
}

func TestHandleLatestUnavailableNonInteractiveReturnsOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(errorWriter{err: errors.New("write failed")})

	_, err := handleLatestUnavailableNonInteractive(cmd, testDeps{runTea: runTeaProgram}, view.ColorDisabled)
	assert.Error(t, err)
}

func TestHandleLatestUnavailableNonInteractiveColorEnabled(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	_, err := handleLatestUnavailableNonInteractive(cmd, testDeps{runTea: runTeaProgram}, view.ColorEnabled)
	assert.ErrorIs(t, err, errLatestVersionRequired)
}

func TestEnsureTestConfigRunsInitWhenConfirmed(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")

	cmd := &cobra.Command{}
	in := fakeTTY{Buffer: &bytes.Buffer{}}
	out := fakeTTY{Buffer: &bytes.Buffer{}}
	cmd.SetIn(in)
	cmd.SetOut(out)

	deps := testDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
		runInit: func(ctx context.Context, _ *cobra.Command, req initRequest) error {
			return config.WriteConfig(ctx, fs, config.NewMetadata(req.configPath), models.ModsJSON{
				Loader:      models.FABRIC,
				GameVersion: "1.20.1",
			})
		},
	}

	state, err := ensureTestConfig(context.Background(), cmd, testOptions{}, deps, meta, interaction.ExecutionModeInteractive)
	require.NoError(t, err)
	assert.True(t, state.shouldContinue)
	assert.Equal(t, "1.20.1", state.cfg.GameVersion)
}

func TestEnsureTestConfigStopsWhenPromptCanceled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")

	cmd := &cobra.Command{}
	in := fakeTTY{Buffer: &bytes.Buffer{}}
	out := fakeTTY{Buffer: &bytes.Buffer{}}
	cmd.SetIn(in)
	cmd.SetOut(out)

	deps := testDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{canceled: true}, nil
		},
	}

	state, err := ensureTestConfig(context.Background(), cmd, testOptions{}, deps, meta, interaction.ExecutionModeInteractive)
	require.NoError(t, err)
	assert.False(t, state.shouldContinue)
}

func TestEnsureTestConfigStopsWhenInitCanceled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")

	cmd := &cobra.Command{}
	in := fakeTTY{Buffer: &bytes.Buffer{}}
	out := fakeTTY{Buffer: &bytes.Buffer{}}
	cmd.SetIn(in)
	cmd.SetOut(out)

	deps := testDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
		runInit: func(context.Context, *cobra.Command, initRequest) error {
			return initCmd.ErrInitCanceled
		},
	}

	state, err := ensureTestConfig(context.Background(), cmd, testOptions{}, deps, meta, interaction.ExecutionModeInteractive)
	require.NoError(t, err)
	assert.False(t, state.shouldContinue)
}

func TestEnsureTestConfigReturnsPromptErrorWhenUnattended(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	_, err := ensureTestConfig(context.Background(), cmd, testOptions{Unattended: true}, testDeps{
		fs:     fs,
		runTea: runTeaProgram,
	}, meta, interaction.ExecutionModeUnattended)
	assert.Error(t, err)
}

func TestEnsureTestConfigReturnsOutputErrorOnPromptError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")

	cmd := &cobra.Command{}
	cmd.SetOut(errorWriter{err: errors.New("write failed")})

	_, err := ensureTestConfig(context.Background(), cmd, testOptions{Unattended: true}, testDeps{
		fs:     fs,
		runTea: runTeaProgram,
	}, meta, interaction.ExecutionModeUnattended)
	assert.Error(t, err)
}

func TestEnsureTestConfigReturnsConfigWhenPresent(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		Loader:      models.FABRIC,
		GameVersion: "1.20.1",
	}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	state, err := ensureTestConfig(context.Background(), cmd, testOptions{}, testDeps{fs: fs}, meta, interaction.ExecutionModeUnattended)
	require.NoError(t, err)
	assert.True(t, state.shouldContinue)
	assert.Equal(t, "1.20.1", state.cfg.GameVersion)
}

func TestEnsureTestConfigReturnsRunInitError(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")

	cmd := &cobra.Command{}
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})

	_, err := ensureTestConfig(context.Background(), cmd, testOptions{}, testDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
		runInit: func(context.Context, *cobra.Command, initRequest) error {
			return errors.New("boom")
		},
	}, meta, interaction.ExecutionModeInteractive)
	assert.Error(t, err)
}

func TestEnsureTestConfigReturnsReadErrorAfterInit(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")

	cmd := &cobra.Command{}
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})

	_, err := ensureTestConfig(context.Background(), cmd, testOptions{}, testDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
		runInit: func(context.Context, *cobra.Command, initRequest) error {
			return nil
		},
	}, meta, interaction.ExecutionModeInteractive)
	assert.Error(t, err)
}

func TestEnsureTestConfigReturnsReadError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, afero.WriteFile(fs, meta.ConfigPath, []byte("not-json"), 0644))

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	_, err := ensureTestConfig(context.Background(), cmd, testOptions{}, testDeps{fs: fs}, meta, interaction.ExecutionModeUnattended)
	assert.Error(t, err)
}

func TestDefaultTestDepsReturnsInitErrorWhenRunnerMissing(t *testing.T) {
	originalRunner := runInteractiveInit
	runInteractiveInit = nil
	t.Cleanup(func() { runInteractiveInit = originalRunner })

	cmd := &cobra.Command{}
	deps := defaultTestDeps(cmd, testOptions{})

	err := deps.runInit(context.Background(), cmd, initRequest{configPath: "/cfg/modlist.json"})
	assert.Error(t, err)
}

func TestEnsureTestConfigReturnsInitRunnerMissingError(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")

	cmd := &cobra.Command{}
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})

	_, err := ensureTestConfig(context.Background(), cmd, testOptions{}, testDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
	}, meta, interaction.ExecutionModeInteractive)
	assert.Error(t, err)
}

func TestEnsureTestConfigStopsWhenPromptDeclined(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")

	cmd := &cobra.Command{}
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})

	state, err := ensureTestConfig(context.Background(), cmd, testOptions{}, testDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: false}, nil
		},
	}, meta, interaction.ExecutionModeInteractive)
	require.NoError(t, err)
	assert.False(t, state.shouldContinue)
}

func TestEnsureTestConfigReturnsPromptError(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")

	cmd := &cobra.Command{}
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})

	runErr := errors.New("boom")
	_, err := ensureTestConfig(context.Background(), cmd, testOptions{}, testDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, runErr
		},
	}, meta, interaction.ExecutionModeInteractive)
	assert.ErrorIs(t, err, runErr)
}

func TestDefaultTestDepsUsesRunner(t *testing.T) {
	originalRunner := runInteractiveInit
	called := false
	runInteractiveInit = func(context.Context, *cobra.Command, initCmd.InteractiveInitDeps, initCmd.InteractiveInitOptions) error {
		called = true
		return nil
	}
	t.Cleanup(func() { runInteractiveInit = originalRunner })

	cmd := &cobra.Command{}
	deps := defaultTestDeps(cmd, testOptions{})

	require.NoError(t, deps.runInit(context.Background(), cmd, initRequest{configPath: "/cfg/modlist.json"}))
	assert.True(t, called)
}

func TestWriteConfigMissingOutputWithColorEnabled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.ANSI })
	t.Cleanup(restoreColors)

	cmd := &cobra.Command{}
	out := fakeTTY{Buffer: &bytes.Buffer{}}
	cmd.SetOut(out)

	err := writeConfigMissingOutput(cmd, testDeps{runTea: runTeaProgram}, config.NewMetadata("/cfg/modlist.json"))
	assert.NoError(t, err)
}

func TestWriteConfigMissingOutputReturnsError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(errorWriter{err: errors.New("write failed")})

	err := writeConfigMissingOutput(cmd, testDeps{runTea: runTeaProgram}, config.NewMetadata("/cfg/modlist.json"))
	assert.Error(t, err)
}

func TestRunTestLockSyncReturnsPolicyError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0o755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	_, err := runTestLockSync(context.Background(), cmd, testOptions{
		LockSync: locksync.PolicyFlags{Add: true, Delete: true},
	}, testDeps{
		fs:     fs,
		runTea: runTeaProgram,
	}, meta, interaction.ExecutionModeNonTTY, cfg)
	assert.Error(t, err)
}

func TestRunTestLockSyncReturnsErrorOnLockEnsureFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0o755))

	errFs := lockStatErrorFs{Fs: fs, failPath: meta.LockPath(), err: errors.New("stat failed")}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	_, err := runTestLockSync(context.Background(), cmd, testOptions{}, testDeps{
		fs: errFs,
	}, meta, interaction.ExecutionModeNonTTY, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stat failed")
}

func TestRunTestLockSyncStopsOnPromptCancel(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0o755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	state, err := runTestLockSync(context.Background(), cmd, testOptions{}, testDeps{
		fs: fs,
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
			return updated, nil
		},
	}, meta, interaction.ExecutionModeInteractive, cfg)
	require.NoError(t, err)
	assert.False(t, state.shouldContinue)
}

func TestRunTestLockSyncReturnsConfigWhenNoExtras(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods:       []models.Mod{{ID: "alpha", Type: models.MODRINTH}},
	}
	lock := []models.ModInstall{{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0o755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	state, err := runTestLockSync(context.Background(), cmd, testOptions{}, testDeps{
		fs: fs,
	}, meta, interaction.ExecutionModeNonTTY, cfg)
	require.NoError(t, err)
	assert.True(t, state.shouldContinue)
}

type statErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (fs statErrorFs) Stat(name string) (os.FileInfo, error) {
	if filepath.Clean(name) == filepath.Clean(fs.failPath) {
		return nil, fs.err
	}
	return fs.Fs.Stat(name)
}

func TestRunTestReturnsErrorWhenUnmanagedNoticeFails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	failPath := filepath.Join(meta.Dir(), ".mmmignore")
	wrapped := statErrorFs{Fs: fs, failPath: failPath, err: errors.New("stat failed")}

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	result, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath: meta.ConfigPath,
		LockSync:   locksync.PolicyFlags{Skip: true},
	}, testDeps{
		fs:     wrapped,
		output: output.New(&bytes.Buffer{}, &bytes.Buffer{}, false),
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	})

	assert.Error(t, err)
	assert.Equal(t, 1, result.ExitCode)
}

func TestRunTestReturnsOutputErrorWhenUnmanagedNoticeWriteFails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar"), []byte("data"), 0644))

	writeErr := errors.New("write failed")
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	result, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath: meta.ConfigPath,
		LockSync:   locksync.PolicyFlags{Skip: true},
	}, testDeps{
		fs: fs,
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			if _, ok := model.(view.OutputLinesModel); ok {
				return view.OutputLinesModel{Err: writeErr}, nil
			}
			return model, nil
		},
	})

	assert.ErrorIs(t, err, writeErr)
	assert.Equal(t, 1, result.ExitCode)
}

type lockStatErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (fs lockStatErrorFs) Stat(name string) (os.FileInfo, error) {
	if name == fs.failPath {
		return nil, fs.err
	}
	return fs.Fs.Stat(name)
}
