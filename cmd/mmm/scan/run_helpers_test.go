package scan

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"github.com/meza/minecraft-mod-manager/internal/modsetup"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestShouldPromptForAdoption(t *testing.T) {
	assert.False(t, shouldPromptForAdoption(scanOptions{Add: true}, []scanMatch{{}}))
	assert.False(t, shouldPromptForAdoption(scanOptions{Unattended: true}, []scanMatch{{}}))
	assert.False(t, shouldPromptForAdoption(scanOptions{}, nil))
	assert.True(t, shouldPromptForAdoption(scanOptions{}, []scanMatch{{}}))
}

func TestRunScanByModeUnattendedUsesInteractiveWhenTTY(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	originalRunScanProgram := runScanProgram
	originalRunScanTranscriptProgram := runScanTranscriptProgram
	t.Cleanup(func() {
		runScanProgram = originalRunScanProgram
		runScanTranscriptProgram = originalRunScanTranscriptProgram
	})

	runScanProgram = func(model *scanModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = scanExecutionOutcome{unknown: []string{"interactive"}}
		return model, nil
	}
	runScanTranscriptProgram = func(model *scanTranscriptModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = scanExecutionOutcome{unknown: []string{"transcript"}}
		return model, nil
	}

	cmd := &cobra.Command{}
	writer := fakeTTY{Buffer: &bytes.Buffer{}}
	cmd.SetIn(writer)
	cmd.SetOut(writer)

	outcome, err := runScanByMode(
		context.Background(),
		cmd,
		scanExecutionInput{deps: scanDeps{runTea: runTeaStub(runTeaScenario{})}},
		interaction.ExecutionModeUnattended,
		scanOptions{Unattended: true},
	)
	assert.NoError(t, err)
	assert.Equal(t, []string{"interactive"}, outcome.unknown)
}

func TestRunScanByModeUnattendedWithoutTTYFallsBackToTranscript(t *testing.T) {
	originalRunScanProgram := runScanProgram
	originalRunScanTranscriptProgram := runScanTranscriptProgram
	t.Cleanup(func() {
		runScanProgram = originalRunScanProgram
		runScanTranscriptProgram = originalRunScanTranscriptProgram
	})

	runScanProgram = func(model *scanModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = scanExecutionOutcome{unknown: []string{"interactive"}}
		return model, nil
	}
	runScanTranscriptProgram = func(model *scanTranscriptModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = scanExecutionOutcome{unknown: []string{"transcript"}}
		return model, nil
	}

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	outcome, err := runScanByMode(
		context.Background(),
		cmd,
		scanExecutionInput{deps: scanDeps{runTea: runTeaStub(runTeaScenario{})}},
		interaction.ExecutionModeUnattended,
		scanOptions{Unattended: true},
	)
	assert.NoError(t, err)
	assert.Equal(t, []string{"transcript"}, outcome.unknown)
}

func TestRunInteractiveScanProgramErrorsWithoutRunner(t *testing.T) {
	original := runScanProgram
	t.Cleanup(func() { runScanProgram = original })
	runScanProgram = nil

	cmd := &cobra.Command{}
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})

	_, err := runInteractiveScanProgram(cmd, model)
	assert.Error(t, err)
}

func TestRunInteractiveScanProgramPropagatesOutcomeError(t *testing.T) {
	original := runScanProgram
	t.Cleanup(func() { runScanProgram = original })

	runScanProgram = func(model *scanModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = scanExecutionOutcome{err: errors.New("boom")}
		return model, nil
	}

	cmd := &cobra.Command{}
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})

	_, err := runInteractiveScanProgram(cmd, model)
	assert.Error(t, err)
}

func TestRunInteractiveScanProgramTreatsContextCancelAsSuccess(t *testing.T) {
	original := runScanProgram
	t.Cleanup(func() { runScanProgram = original })

	runScanProgram = func(model *scanModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = scanExecutionOutcome{err: context.Canceled}
		return model, nil
	}

	cmd := &cobra.Command{}
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})

	outcome, err := runInteractiveScanProgram(cmd, model)
	assert.NoError(t, err)
	assert.ErrorIs(t, outcome.err, context.Canceled)
}

func TestRunInteractiveScanProgramReturnsUnexpectedModelError(t *testing.T) {
	original := runScanProgram
	t.Cleanup(func() { runScanProgram = original })

	runScanProgram = func(_ *scanModel, _ ...tea.ProgramOption) (tea.Model, error) {
		return scanFakePromptModel{}, nil
	}

	cmd := &cobra.Command{}
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})

	_, err := runInteractiveScanProgram(cmd, model)
	assert.Error(t, err)
}

func TestRunInteractiveScanProgramPropagatesRunError(t *testing.T) {
	original := runScanProgram
	t.Cleanup(func() { runScanProgram = original })

	runScanProgram = func(*scanModel, ...tea.ProgramOption) (tea.Model, error) {
		return nil, errors.New("boom")
	}

	cmd := &cobra.Command{}
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})

	_, err := runInteractiveScanProgram(cmd, model)
	assert.Error(t, err)
}

func TestRunInteractiveScanProgramSuccess(t *testing.T) {
	original := runScanProgram
	t.Cleanup(func() { runScanProgram = original })

	expected := scanExecutionOutcome{unknown: []string{"alpha.jar"}}
	runScanProgram = func(model *scanModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = expected
		return model, nil
	}

	cmd := &cobra.Command{}
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})

	outcome, err := runInteractiveScanProgram(cmd, model)
	assert.NoError(t, err)
	assert.Equal(t, expected, outcome)
}

func TestRunInteractiveScanReturnsProgramError(t *testing.T) {
	original := runScanProgram
	t.Cleanup(func() { runScanProgram = original })

	runScanProgram = func(*scanModel, ...tea.ProgramOption) (tea.Model, error) {
		return nil, errors.New("boom")
	}

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	_, err := runInteractiveScan(context.Background(), cmd, scanExecutionInput{}, scanOptions{})
	assert.Error(t, err)
}

func TestRunInteractiveScanWritesResults(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	original := runScanProgram
	t.Cleanup(func() { runScanProgram = original })

	runScanProgram = func(model *scanModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = model.execRunner(context.Background(), scanExecSender{})
		return model, nil
	}

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)

	outcome, err := runInteractiveScan(context.Background(), cmd, scanExecutionInput{
		candidates:     []scanCandidate{{Path: "/mods/alpha.jar", FileName: "alpha.jar", Sha1: "alpha"}},
		preferPlatform: models.MODRINTH,
		deps: scanDeps{
			runTea: runTeaStub(runTeaScenario{}),
			clients: platform.Clients{
				Modrinth:   noopDoer{},
				Curseforge: noopDoer{},
			},
			modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
				return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup("alpha", modrinth.SHA1)}
			},
			curseforgeFingerprint: func(string) uint32 { return 0 },
			curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
				return &curseforge.FingerprintResult{}, nil
			},
		},
	}, scanOptions{})
	assert.NoError(t, err)
	assert.Contains(t, outcome.unknown, "alpha.jar")
	assert.Contains(t, out.String(), "cmd.scan.header.results")
}

func TestRunInteractiveScanIgnoresContextCancellation(t *testing.T) {
	original := runScanProgram
	t.Cleanup(func() { runScanProgram = original })

	runScanProgram = func(model *scanModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = scanExecutionOutcome{err: context.Canceled}
		return model, nil
	}

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)

	outcome, err := runInteractiveScan(context.Background(), cmd, scanExecutionInput{
		candidates: []scanCandidate{{Path: "/mods/alpha.jar", FileName: "alpha.jar", Sha1: "alpha"}},
	}, scanOptions{})
	assert.NoError(t, err)
	assert.Empty(t, outcome.matches)
	assert.Empty(t, strings.TrimSpace(out.String()))
}

func TestRunInteractiveScanIgnoresProgramCancelError(t *testing.T) {
	original := runScanProgram
	t.Cleanup(func() { runScanProgram = original })

	runScanProgram = func(*scanModel, ...tea.ProgramOption) (tea.Model, error) {
		return nil, context.Canceled
	}

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)

	outcome, err := runInteractiveScan(context.Background(), cmd, scanExecutionInput{
		candidates: []scanCandidate{{Path: "/mods/alpha.jar", FileName: "alpha.jar", Sha1: "alpha"}},
	}, scanOptions{})
	assert.NoError(t, err)
	assert.Empty(t, outcome.matches)
	assert.Empty(t, strings.TrimSpace(out.String()))
}

func TestHandleInteractiveScanAddReturnsPersistError(t *testing.T) {
	cmd := &cobra.Command{}
	input := scanExecutionInput{}
	outcome := scanExecutionOutcome{matches: []scanMatch{{FileName: "alpha.jar"}}}

	_, err := handleInteractiveScanAdd(context.Background(), cmd, input, view.ColorDisabled, outcome)
	assert.Error(t, err)
}

func TestHandleInteractiveScanAddReturnsOutputError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/cfg", 0o755))
	meta := config.NewMetadata("/cfg/modlist.json")
	input := scanExecutionInput{
		meta:             meta,
		cfg:              models.ModsJSON{ModsFolder: "mods"},
		lock:             []models.ModInstall{},
		setupCoordinator: modsetup.NewSetupCoordinator(fs, nil, nil),
		deps: scanDeps{
			fs:     fs,
			runTea: runTeaStub(runTeaScenario{err: errors.New("write failed")}),
		},
	}

	outcome := scanExecutionOutcome{matches: []scanMatch{scanMatchFixture()}}
	_, err := handleInteractiveScanAdd(context.Background(), cmd, input, view.ColorDisabled, outcome)
	assert.Error(t, err)
}

func TestHandleInteractiveScanAddSuccess(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/cfg", 0o755))
	meta := config.NewMetadata("/cfg/modlist.json")
	input := scanExecutionInput{
		meta:             meta,
		cfg:              models.ModsJSON{ModsFolder: "mods"},
		lock:             []models.ModInstall{},
		setupCoordinator: modsetup.NewSetupCoordinator(fs, nil, nil),
		deps: scanDeps{
			fs:     fs,
			runTea: runTeaStub(runTeaScenario{}),
		},
	}

	outcome := scanExecutionOutcome{matches: []scanMatch{scanMatchFixture()}}
	updated, err := handleInteractiveScanAdd(context.Background(), cmd, input, view.ColorDisabled, outcome)
	assert.NoError(t, err)
	assert.Len(t, updated.added, 1)
	assert.Contains(t, out.String(), "cmd.scan.section.added")
}

func TestHandleInteractivePromptConfirmErrorPaths(t *testing.T) {
	cmd := &cobra.Command{}
	input := scanExecutionInput{}
	outcome := scanExecutionOutcome{matches: []scanMatch{scanMatchFixture()}}
	prompt := newScanConfirmPromptModel("Question?")

	_, err := handleInteractivePromptConfirm(context.Background(), cmd, input, view.ColorDisabled, outcome, prompt)
	assert.Error(t, err)

	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/cfg", 0o755))
	meta := config.NewMetadata("/cfg/modlist.json")
	input = scanExecutionInput{
		meta:             meta,
		cfg:              models.ModsJSON{ModsFolder: "mods"},
		lock:             []models.ModInstall{},
		setupCoordinator: modsetup.NewSetupCoordinator(fs, nil, nil),
		deps: scanDeps{
			fs:     fs,
			runTea: runTeaStub(runTeaScenario{err: errors.New("write failed")}),
		},
	}

	_, err = handleInteractivePromptConfirm(context.Background(), cmd, input, view.ColorDisabled, outcome, prompt)
	assert.Error(t, err)
}

func TestHandleInteractivePromptConfirmSuccess(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/cfg", 0o755))
	meta := config.NewMetadata("/cfg/modlist.json")
	input := scanExecutionInput{
		meta:             meta,
		cfg:              models.ModsJSON{ModsFolder: "mods"},
		lock:             []models.ModInstall{},
		setupCoordinator: modsetup.NewSetupCoordinator(fs, nil, nil),
		deps: scanDeps{
			fs:     fs,
			runTea: runTeaStub(runTeaScenario{}),
		},
	}

	outcome := scanExecutionOutcome{matches: []scanMatch{scanMatchFixture()}}
	prompt := newScanConfirmPromptModel("Question?")
	prompt.Value = prompt.yesOption.short

	updated, err := handleInteractivePromptConfirm(context.Background(), cmd, input, view.ColorDisabled, outcome, prompt)
	assert.NoError(t, err)
	assert.Len(t, updated.added, 1)
	assert.Contains(t, out.String(), "cmd.scan.section.added")
}

func TestHandleInteractivePromptConfirmWritesPromptSectionsWhenUncertain(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/cfg", 0o755))
	meta := config.NewMetadata("/cfg/modlist.json")
	input := scanExecutionInput{
		meta:             meta,
		cfg:              models.ModsJSON{ModsFolder: "mods"},
		lock:             []models.ModInstall{},
		setupCoordinator: modsetup.NewSetupCoordinator(fs, nil, nil),
		deps: scanDeps{
			fs:     fs,
			runTea: runTeaStub(runTeaScenario{}),
		},
	}

	outcome := scanExecutionOutcome{
		matches: []scanMatch{scanMatchFixture()},
		unknown: []string{"unmanaged.jar"},
	}
	prompt := newScanConfirmPromptModel("Question?")
	prompt.Value = prompt.yesOption.short
	prompt.input.SetValue(prompt.Value)

	updated, err := handleInteractivePromptConfirm(context.Background(), cmd, input, view.ColorDisabled, outcome, prompt)
	assert.NoError(t, err)
	assert.Len(t, updated.added, 1)
	assert.Contains(t, out.String(), "cmd.scan.section.unknown")
	assert.Contains(t, out.String(), "cmd.scan.section.added")
	assert.Contains(t, out.String(), "Question?")
}

func TestHandleInteractivePromptConfirmUncertainOutputError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cmd := &cobra.Command{}
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/cfg", 0o755))
	meta := config.NewMetadata("/cfg/modlist.json")
	input := scanExecutionInput{
		meta:             meta,
		cfg:              models.ModsJSON{ModsFolder: "mods"},
		lock:             []models.ModInstall{},
		setupCoordinator: modsetup.NewSetupCoordinator(fs, nil, nil),
		deps: scanDeps{
			fs:     fs,
			runTea: runTeaStub(runTeaScenario{err: errors.New("write failed")}),
		},
	}

	outcome := scanExecutionOutcome{
		matches: []scanMatch{scanMatchFixture()},
		unknown: []string{"unmanaged.jar"},
	}
	prompt := newScanConfirmPromptModel("Question?")

	_, err := handleInteractivePromptConfirm(context.Background(), cmd, input, view.ColorDisabled, outcome, prompt)
	assert.Error(t, err)
}

func TestHandleInteractiveScanPromptCanceled(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)
	input := scanExecutionInput{
		deps: scanDeps{
			runTea: runTeaStub(runTeaScenario{
				adoptionResult: &scanAdoptionPromptModel{canceled: true},
			}),
		},
	}
	outcome := scanExecutionOutcome{matches: []scanMatch{scanMatchFixture()}}

	_, err := handleInteractiveScanPrompt(context.Background(), cmd, input, view.ColorDisabled, outcome)
	assert.NoError(t, err)
	assert.Empty(t, strings.TrimSpace(out.String()))
}

func TestHandleInteractiveScanPromptDeclined(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)
	input := scanExecutionInput{
		deps: scanDeps{
			runTea: runTeaStub(runTeaScenario{
				adoptionResult: &scanAdoptionPromptModel{confirmed: false},
			}),
		},
	}
	outcome := scanExecutionOutcome{matches: []scanMatch{scanMatchFixture()}}

	_, err := handleInteractiveScanPrompt(context.Background(), cmd, input, view.ColorDisabled, outcome)
	assert.NoError(t, err)
	assert.Contains(t, out.String(), "cmd.scan.adoption.cancelled")
}

func TestHandleInteractiveScanPromptPropagatesPromptError(t *testing.T) {
	cmd := &cobra.Command{}
	input := scanExecutionInput{
		deps: scanDeps{runTea: runTeaStub(runTeaScenario{err: errors.New("prompt failed")})},
	}
	outcome := scanExecutionOutcome{matches: []scanMatch{scanMatchFixture()}}

	_, err := handleInteractiveScanPrompt(context.Background(), cmd, input, view.ColorDisabled, outcome)
	assert.Error(t, err)
}

func TestHandleInteractiveScanPromptPropagatesOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	runTea := func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		switch typed := model.(type) {
		case *scanAdoptionPromptModel:
			return &scanAdoptionPromptModel{canceled: true, prompt: typed.prompt}, nil
		default:
			return nil, errors.New("write failed")
		}
	}

	input := scanExecutionInput{
		deps: scanDeps{runTea: runTea},
	}
	outcome := scanExecutionOutcome{matches: []scanMatch{scanMatchFixture()}}

	_, err := handleInteractiveScanPrompt(context.Background(), cmd, input, view.ColorDisabled, outcome)
	assert.NoError(t, err)
}

func TestHandleInteractiveScanPromptDeclineOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	runTea := func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		switch typed := model.(type) {
		case *scanAdoptionPromptModel:
			return &scanAdoptionPromptModel{confirmed: false, prompt: typed.prompt}, nil
		default:
			return nil, errors.New("write failed")
		}
	}

	input := scanExecutionInput{
		deps: scanDeps{runTea: runTea},
	}
	outcome := scanExecutionOutcome{matches: []scanMatch{scanMatchFixture()}}

	_, err := handleInteractiveScanPrompt(context.Background(), cmd, input, view.ColorDisabled, outcome)
	assert.Error(t, err)
}

func TestHandleInteractiveScanPromptConfirmed(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/cfg", 0o755))
	meta := config.NewMetadata("/cfg/modlist.json")
	prompt := newScanConfirmPromptModel("Question?")
	prompt.Value = prompt.yesOption.short

	input := scanExecutionInput{
		meta:             meta,
		cfg:              models.ModsJSON{ModsFolder: "mods"},
		lock:             []models.ModInstall{},
		setupCoordinator: modsetup.NewSetupCoordinator(fs, nil, nil),
		deps: scanDeps{
			fs: fs,
			runTea: runTeaStub(runTeaScenario{
				adoptionResult: &scanAdoptionPromptModel{confirmed: true, prompt: prompt},
			}),
		},
	}
	outcome := scanExecutionOutcome{matches: []scanMatch{scanMatchFixture()}}

	updated, err := handleInteractiveScanPrompt(context.Background(), cmd, input, view.ColorDisabled, outcome)
	assert.NoError(t, err)
	assert.Len(t, updated.added, 1)
	assert.Contains(t, out.String(), "cmd.scan.section.added")
}

func TestHandleInteractiveScanAddWritesResultsWhenUncertain(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/cfg", 0o755))
	meta := config.NewMetadata("/cfg/modlist.json")
	input := scanExecutionInput{
		meta:             meta,
		cfg:              models.ModsJSON{ModsFolder: "mods"},
		lock:             []models.ModInstall{},
		setupCoordinator: modsetup.NewSetupCoordinator(fs, nil, nil),
		deps: scanDeps{
			fs:     fs,
			runTea: runTeaStub(runTeaScenario{}),
		},
	}
	outcome := scanExecutionOutcome{
		matches: []scanMatch{scanMatchFixture()},
		unknown: []string{"unmanaged.jar"},
	}

	updated, err := handleInteractiveScanAdd(context.Background(), cmd, input, view.ColorDisabled, outcome)
	assert.NoError(t, err)
	assert.Len(t, updated.added, 1)
	assert.Contains(t, out.String(), "cmd.scan.section.unknown")
	assert.Contains(t, out.String(), "cmd.scan.section.added")
}

func TestHandleInteractiveScanAddUncertainOutputError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cmd := &cobra.Command{}
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/cfg", 0o755))
	meta := config.NewMetadata("/cfg/modlist.json")
	input := scanExecutionInput{
		meta:             meta,
		cfg:              models.ModsJSON{ModsFolder: "mods"},
		lock:             []models.ModInstall{},
		setupCoordinator: modsetup.NewSetupCoordinator(fs, nil, nil),
		deps: scanDeps{
			fs:     fs,
			runTea: runTeaStub(runTeaScenario{err: errors.New("write failed")}),
		},
	}
	outcome := scanExecutionOutcome{
		matches: []scanMatch{scanMatchFixture()},
		unknown: []string{"unmanaged.jar"},
	}

	_, err := handleInteractiveScanAdd(context.Background(), cmd, input, view.ColorDisabled, outcome)
	assert.Error(t, err)
}

func TestHandleInteractiveScanOutcomeUsesPromptPath(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)
	input := scanExecutionInput{
		deps: scanDeps{
			runTea: runTeaStub(runTeaScenario{
				adoptionResult: &scanAdoptionPromptModel{canceled: true},
			}),
		},
	}
	outcome := scanExecutionOutcome{matches: []scanMatch{scanMatchFixture()}}

	_, err := handleInteractiveScanOutcome(context.Background(), cmd, input, scanOptions{}, view.ColorDisabled, outcome)
	assert.NoError(t, err)
	assert.Empty(t, strings.TrimSpace(out.String()))
}

func TestHandleInteractiveScanOutcomeWritesResults(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)
	input := scanExecutionInput{
		deps: scanDeps{runTea: runTeaStub(runTeaScenario{})},
	}
	outcome := scanExecutionOutcome{}

	_, err := handleInteractiveScanOutcome(context.Background(), cmd, input, scanOptions{}, view.ColorDisabled, outcome)
	assert.NoError(t, err)
	assert.Contains(t, out.String(), "cmd.scan.header.results")
}

func TestHandleInteractiveScanOutcomeReturnsOutputError(t *testing.T) {
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	input := scanExecutionInput{
		deps: scanDeps{runTea: runTeaStub(runTeaScenario{err: errors.New("write failed")})},
	}
	outcome := scanExecutionOutcome{}

	_, err := handleInteractiveScanOutcome(context.Background(), cmd, input, scanOptions{}, view.ColorDisabled, outcome)
	assert.Error(t, err)
}

func TestHandleInteractiveScanOutcomeUsesAddPath(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/cfg", 0o755))
	meta := config.NewMetadata("/cfg/modlist.json")
	input := scanExecutionInput{
		meta:             meta,
		cfg:              models.ModsJSON{ModsFolder: "mods"},
		lock:             []models.ModInstall{},
		setupCoordinator: modsetup.NewSetupCoordinator(fs, nil, nil),
		deps: scanDeps{
			fs:     fs,
			runTea: runTeaStub(runTeaScenario{}),
		},
	}
	outcome := scanExecutionOutcome{matches: []scanMatch{scanMatchFixture()}}

	updated, err := handleInteractiveScanOutcome(context.Background(), cmd, input, scanOptions{Add: true}, view.ColorDisabled, outcome)
	assert.NoError(t, err)
	assert.Len(t, updated.added, 1)
	assert.Contains(t, out.String(), "cmd.scan.section.added")
}

func TestRunScanTranscriptErrorsWithoutRunner(t *testing.T) {
	original := runScanTranscriptProgram
	t.Cleanup(func() { runScanTranscriptProgram = original })
	runScanTranscriptProgram = nil

	cmd := &cobra.Command{}
	_, err := runScanTranscript(context.Background(), cmd, scanExecutionInput{}, scanOptions{})
	assert.Error(t, err)
}

func TestRunScanTranscriptPropagatesOutcomeError(t *testing.T) {
	original := runScanTranscriptProgram
	t.Cleanup(func() { runScanTranscriptProgram = original })

	runScanTranscriptProgram = func(_ *scanTranscriptModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model := &scanTranscriptModel{}
		model.outcome = scanExecutionOutcome{err: errors.New("boom")}
		return model, nil
	}

	cmd := &cobra.Command{}
	_, err := runScanTranscript(context.Background(), cmd, scanExecutionInput{}, scanOptions{})
	assert.Error(t, err)
}

func TestRunScanTranscriptPropagatesRunError(t *testing.T) {
	original := runScanTranscriptProgram
	t.Cleanup(func() { runScanTranscriptProgram = original })

	runScanTranscriptProgram = func(*scanTranscriptModel, ...tea.ProgramOption) (tea.Model, error) {
		return nil, errors.New("boom")
	}

	cmd := &cobra.Command{}
	_, err := runScanTranscript(context.Background(), cmd, scanExecutionInput{}, scanOptions{})
	assert.Error(t, err)
}

func TestRunScanTranscriptReturnsUnexpectedModelError(t *testing.T) {
	original := runScanTranscriptProgram
	t.Cleanup(func() { runScanTranscriptProgram = original })

	runScanTranscriptProgram = func(*scanTranscriptModel, ...tea.ProgramOption) (tea.Model, error) {
		return scanFakePromptModel{}, nil
	}

	cmd := &cobra.Command{}
	_, err := runScanTranscript(context.Background(), cmd, scanExecutionInput{}, scanOptions{})
	assert.Error(t, err)
}

func TestRunScanTranscriptAddsResults(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	original := runScanTranscriptProgram
	t.Cleanup(func() { runScanTranscriptProgram = original })

	runScanTranscriptProgram = func(_ *scanTranscriptModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model := &scanTranscriptModel{}
		model.outcome = scanExecutionOutcome{
			matches: []scanMatch{scanMatchFixture()},
		}
		return model, nil
	}

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/cfg", 0o755))
	meta := config.NewMetadata("/cfg/modlist.json")
	input := scanExecutionInput{
		meta:             meta,
		cfg:              models.ModsJSON{ModsFolder: "mods"},
		lock:             []models.ModInstall{},
		setupCoordinator: modsetup.NewSetupCoordinator(fs, nil, nil),
		deps: scanDeps{
			fs:     fs,
			runTea: runTeaStub(runTeaScenario{}),
		},
	}

	outcome, err := runScanTranscript(context.Background(), cmd, input, scanOptions{Add: true})
	assert.NoError(t, err)
	assert.Len(t, outcome.added, 1)
	assert.Contains(t, out.String(), "cmd.scan.section.added")
}

func TestRunScanTranscriptAddFullAdoptionWritesPerFileLines(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	original := runScanTranscriptProgram
	t.Cleanup(func() { runScanTranscriptProgram = original })

	runScanTranscriptProgram = func(model *scanTranscriptModel, _ ...tea.ProgramOption) (tea.Model, error) {
		match := scanMatchFixture()
		model.Update(scanItemUpdateMsg{key: match.FileName, status: scanItemStatusRecognized, match: match})
		model.Update(scanExecutionFinishedMsg{outcome: scanExecutionOutcome{
			items: []scanItem{{FileName: match.FileName, Status: scanItemStatusRecognized, Match: match}},
		}})
		model.outcome = scanExecutionOutcome{matches: []scanMatch{match}}
		return model, nil
	}

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/cfg", 0o755))
	meta := config.NewMetadata("/cfg/modlist.json")
	input := scanExecutionInput{
		meta:             meta,
		cfg:              models.ModsJSON{ModsFolder: "mods"},
		lock:             []models.ModInstall{},
		setupCoordinator: modsetup.NewSetupCoordinator(fs, nil, nil),
		candidates:       []scanCandidate{{Path: "/mods/alpha.jar", FileName: "alpha.jar", Sha1: "hash"}},
		deps: scanDeps{
			fs:     fs,
			runTea: runTeaStub(runTeaScenario{}),
		},
	}

	outcome, err := runScanTranscript(context.Background(), cmd, input, scanOptions{Add: true})
	assert.NoError(t, err)
	assert.Len(t, outcome.added, 1)
	assert.Contains(t, out.String(), "cmd.scan.section.added")
	assert.Contains(t, out.String(), "alpha.jar ->")
}

func TestRunScanTranscriptAddReturnsOutputError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	original := runScanTranscriptProgram
	t.Cleanup(func() { runScanTranscriptProgram = original })

	runScanTranscriptProgram = func(_ *scanTranscriptModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model := &scanTranscriptModel{}
		model.outcome = scanExecutionOutcome{matches: []scanMatch{scanMatchFixture()}}
		return model, nil
	}

	cmd := &cobra.Command{}
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/cfg", 0o755))
	meta := config.NewMetadata("/cfg/modlist.json")
	input := scanExecutionInput{
		meta:             meta,
		cfg:              models.ModsJSON{ModsFolder: "mods"},
		lock:             []models.ModInstall{},
		setupCoordinator: modsetup.NewSetupCoordinator(fs, nil, nil),
		deps: scanDeps{
			fs:     fs,
			runTea: runTeaStub(runTeaScenario{err: errors.New("write failed")}),
		},
	}

	_, err := runScanTranscript(context.Background(), cmd, input, scanOptions{Add: true})
	assert.Error(t, err)
}

func TestRunScanTranscriptAddReturnsOutputErrorOnUncertainResults(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	original := runScanTranscriptProgram
	t.Cleanup(func() { runScanTranscriptProgram = original })

	runScanTranscriptProgram = func(_ *scanTranscriptModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model := &scanTranscriptModel{}
		model.outcome = scanExecutionOutcome{
			matches: []scanMatch{scanMatchFixture()},
			unknown: []string{"unmanaged.jar"},
		}
		return model, nil
	}

	cmd := &cobra.Command{}
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/cfg", 0o755))
	meta := config.NewMetadata("/cfg/modlist.json")
	input := scanExecutionInput{
		meta:             meta,
		cfg:              models.ModsJSON{ModsFolder: "mods"},
		lock:             []models.ModInstall{},
		setupCoordinator: modsetup.NewSetupCoordinator(fs, nil, nil),
		deps: scanDeps{
			fs:     fs,
			runTea: runTeaStub(runTeaScenario{err: errors.New("write failed")}),
		},
	}

	_, err := runScanTranscript(context.Background(), cmd, input, scanOptions{Add: true})
	assert.Error(t, err)
}

func TestRunScanTranscriptAddReturnsPersistError(t *testing.T) {
	original := runScanTranscriptProgram
	t.Cleanup(func() { runScanTranscriptProgram = original })

	runScanTranscriptProgram = func(_ *scanTranscriptModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model := &scanTranscriptModel{}
		model.outcome = scanExecutionOutcome{matches: []scanMatch{scanMatchFixture()}}
		return model, nil
	}

	cmd := &cobra.Command{}
	input := scanExecutionInput{
		deps: scanDeps{runTea: runTeaStub(runTeaScenario{})},
	}

	_, err := runScanTranscript(context.Background(), cmd, input, scanOptions{Add: true})
	assert.Error(t, err)
}

func TestRunScanTranscriptReturnsOutcomeWithoutAdd(t *testing.T) {
	original := runScanTranscriptProgram
	t.Cleanup(func() { runScanTranscriptProgram = original })

	runScanTranscriptProgram = func(_ *scanTranscriptModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model := &scanTranscriptModel{}
		model.outcome = scanExecutionOutcome{unknown: []string{"alpha.jar"}}
		return model, nil
	}

	cmd := &cobra.Command{}
	outcome, err := runScanTranscript(context.Background(), cmd, scanExecutionInput{}, scanOptions{})
	assert.NoError(t, err)
	assert.Equal(t, []string{"alpha.jar"}, outcome.unknown)
}

func TestRunQuietScanOutputsUnknown(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}
	input := scanExecutionInput{
		candidates:     candidates,
		preferPlatform: models.MODRINTH,
		deps: scanDeps{
			clients: platform.Clients{Modrinth: noopDoer{}, Curseforge: noopDoer{}},
			modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
				return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup("a", modrinth.SHA1)}
			},
			curseforgeFingerprint: func(string) uint32 { return 101 },
			curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
				return &curseforge.FingerprintResult{}, nil
			},
		},
	}

	outcome, err := runQuietScan(context.Background(), cmd, input, scanOptions{})
	assert.NoError(t, err)
	assert.Empty(t, outcome.err)
	assert.Contains(t, out.String(), "cmd.scan.section.unknown")
}

func TestRunQuietScanReturnsExecutionError(t *testing.T) {
	writeErr := errors.New("write failed")
	input := scanExecutionInput{
		candidates:     []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}},
		preferPlatform: models.MODRINTH,
		deps: scanDeps{
			logger: logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
			modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
				return nil, errors.New("boom")
			},
		},
	}

	_, err := runQuietScan(context.Background(), &cobra.Command{}, input, scanOptions{})
	assert.ErrorIs(t, err, writeErr)
}

func TestRunQuietScanReturnsOutputError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}
	input := scanExecutionInput{
		candidates:     candidates,
		preferPlatform: models.MODRINTH,
		deps: scanDeps{
			runTea: runTeaStub(runTeaScenario{err: errors.New("write failed")}),
			clients: platform.Clients{
				Modrinth:   noopDoer{},
				Curseforge: noopDoer{},
			},
			modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
				return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup("a", modrinth.SHA1)}
			},
			curseforgeFingerprint: func(string) uint32 { return 101 },
			curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
				return &curseforge.FingerprintResult{}, nil
			},
		},
	}

	_, err := runQuietScan(context.Background(), cmd, input, scanOptions{})
	assert.Error(t, err)
}

func TestRunQuietScanAddPersistsResults(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/cfg", 0o755))
	meta := config.NewMetadata("/cfg/modlist.json")
	setup := modsetup.NewSetupCoordinator(fs, nil, nil)

	version := &modrinth.Version{
		ProjectID:     "alpha",
		DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Files: []modrinth.VersionFile{
			{URL: "https://example.invalid/alpha.jar", Primary: true},
		},
	}

	input := scanExecutionInput{
		meta:             meta,
		cfg:              models.ModsJSON{ModsFolder: "mods"},
		lock:             []models.ModInstall{},
		setupCoordinator: setup,
		candidates:       []scanCandidate{{Path: "/mods/alpha.jar", FileName: "alpha.jar", Sha1: "hash"}},
		preferPlatform:   models.MODRINTH,
		deps: scanDeps{
			fs:     fs,
			runTea: runTeaStub(runTeaScenario{}),
			clients: platform.Clients{
				Modrinth: noopDoer{},
			},
			modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
				return version, nil
			},
			modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
				return "Alpha", nil
			},
		},
	}

	outcome, err := runQuietScan(context.Background(), cmd, input, scanOptions{Add: true})
	assert.NoError(t, err)
	assert.Len(t, outcome.added, 1)
}

func TestRunQuietScanAddReturnsPersistError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	version := &modrinth.Version{
		ProjectID:     "alpha",
		DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Files: []modrinth.VersionFile{
			{URL: "https://example.invalid/alpha.jar", Primary: true},
		},
	}

	input := scanExecutionInput{
		cfg:            models.ModsJSON{ModsFolder: "mods"},
		lock:           []models.ModInstall{},
		candidates:     []scanCandidate{{Path: "/mods/alpha.jar", FileName: "alpha.jar", Sha1: "hash"}},
		preferPlatform: models.MODRINTH,
		deps: scanDeps{
			runTea: runTeaStub(runTeaScenario{}),
			clients: platform.Clients{
				Modrinth: noopDoer{},
			},
			modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
				return version, nil
			},
			modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
				return "Alpha", nil
			},
		},
	}

	_, err := runQuietScan(context.Background(), cmd, input, scanOptions{Add: true})
	assert.Error(t, err)
}

func TestWriteScanResultsOutputWritesSections(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)
	input := scanExecutionInput{deps: scanDeps{runTea: runTeaStub(runTeaScenario{})}}
	outcome := scanExecutionOutcome{matches: []scanMatch{scanMatchFixture()}}

	err := writeScanResultsOutput(cmd, input, outcome)
	assert.NoError(t, err)
	assert.Contains(t, out.String(), "cmd.scan.header.results")
}

func TestBuildScanResultSectionsWithoutMatchesSkipsRecognized(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	outcome := scanExecutionOutcome{
		matches: []scanMatch{scanMatchFixture()},
		unknown: []string{"unmanaged.jar"},
	}
	sections := buildScanResultSectionsWithoutMatches(outcome, view.ColorDisabled)
	joined := strings.Join(sections, "\n")
	assert.Contains(t, joined, "cmd.scan.section.unknown")
	assert.NotContains(t, joined, "cmd.scan.section.recognized")
}

func TestBuildScanPromptTranscriptSectionsIncludesPrompt(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	outcome := scanExecutionOutcome{matches: []scanMatch{scanMatchFixture()}}
	prompt := newScanConfirmPromptModel("Question?")

	sections := buildScanPromptTranscriptSections(outcome, view.ColorDisabled, prompt)
	if assert.NotEmpty(t, sections) {
		assert.Equal(t, prompt.View(), sections[len(sections)-1])
	}
}

func TestWriteScanTranscriptSectionsEmpty(t *testing.T) {
	cmd := &cobra.Command{}
	err := writeScanTranscriptSections(cmd, scanExecutionInput{}, nil)
	assert.NoError(t, err)
}

func TestWriteScanAllManagedWithNilCommand(t *testing.T) {
	assert.NoError(t, writeScanAllManaged(nil, scanDeps{}))
}

func TestWriteScanAddedSectionOutputEmpty(t *testing.T) {
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	err := writeScanAddedSectionOutput(cmd, scanExecutionInput{deps: scanDeps{runTea: runTeaStub(runTeaScenario{})}}, nil)
	assert.NoError(t, err)
	assert.Empty(t, strings.TrimSpace(out.String()))
}

func TestWriteScanQuietOutputAddReturnsNil(t *testing.T) {
	cmd := &cobra.Command{}
	err := writeScanQuietOutput(cmd, scanExecutionInput{}, scanExecutionOutcome{}, scanOptions{Add: true})
	assert.NoError(t, err)
}

func TestWriteInteractiveScanTranscriptNilModel(t *testing.T) {
	cmd := &cobra.Command{}
	assert.NoError(t, writeInteractiveScanTranscript(cmd, scanExecutionInput{}, nil))
}

func TestWriteInteractiveScanTranscriptWritesSections(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})
	model.outcome = scanExecutionOutcome{unknown: []string{"alpha.jar"}}

	err := writeInteractiveScanTranscript(cmd, scanExecutionInput{deps: scanDeps{runTea: runTeaStub(runTeaScenario{})}}, model)
	assert.NoError(t, err)
	assert.Contains(t, out.String(), "cmd.scan.header.results")
}

func TestWriteConfigMissingOutputSuccess(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	deps := scanDeps{runTea: runTeaStub(runTeaScenario{})}
	meta := config.NewMetadata("/cfg/modlist.json")

	err := writeConfigMissingOutput(cmd, deps, meta)
	assert.NoError(t, err)
	assert.Contains(t, out.String(), "cmd.scan.error.config_missing")
}

func TestWriteConfigMissingOutputColorEnabled(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreProfile := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreProfile)

	out := fakeTTY{Buffer: &bytes.Buffer{}}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	deps := scanDeps{runTea: runTeaStub(runTeaScenario{})}
	meta := config.NewMetadata("/cfg/modlist.json")

	err := writeConfigMissingOutput(cmd, deps, meta)
	assert.NoError(t, err)
}

func TestWriteConfigMissingOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	deps := scanDeps{runTea: runTeaStub(runTeaScenario{err: errors.New("write failed")})}
	meta := config.NewMetadata("/cfg/modlist.json")

	err := writeConfigMissingOutput(cmd, deps, meta)
	assert.Error(t, err)
}

func TestWriteFullAdoptionOutputSkipsEmptyAdded(t *testing.T) {
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	outcome := scanExecutionOutcome{}
	updated, err := writeFullAdoptionOutput(cmd, scanExecutionInput{}, view.ColorDisabled, outcome)
	assert.NoError(t, err)
	assert.Equal(t, outcome, updated)
	assert.Empty(t, out.String())
}

func TestHandleScanFailurePropagatesOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	deps := scanDeps{runTea: runTeaStub(runTeaScenario{err: errors.New("write failed")})}

	err := handleScanFailure(cmd, deps, errors.New("boom"))
	assert.Error(t, err)
}

func TestAppendScanAddedSection(t *testing.T) {
	sections := []string{"one"}
	assert.Equal(t, sections, appendScanAddedSection(sections, nil, view.ColorDisabled))

	updated := appendScanAddedSection(sections, []scanMatch{scanMatchFixture()}, view.ColorDisabled)
	assert.Len(t, updated, 2)
}

func TestColorModeForOutputNilWriter(t *testing.T) {
	assert.Equal(t, view.ColorDisabled, colorModeForOutput(nil))
}

func TestHandleAllManagedScanQuiet(t *testing.T) {
	result, err := handleAllManagedScan(&cobra.Command{}, scanDeps{}, scanOptions{Quiet: true}, interaction.ExecutionModeUnattended)
	assert.NoError(t, err)
	assert.Equal(t, scanSuccessTelemetryWithoutArgs(interaction.ExecutionModeUnattended.String(), false), result)
}

func TestHandleAllManagedScanOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	deps := scanDeps{runTea: runTeaStub(runTeaScenario{err: errors.New("write failed")})}

	_, err := handleAllManagedScan(cmd, deps, scanOptions{}, interaction.ExecutionModeInteractive)
	assert.Error(t, err)
}

func TestRunScanAdoptionPromptErrorsWithoutRunner(t *testing.T) {
	original := runTeaProgram
	t.Cleanup(func() { runTeaProgram = original })
	runTeaProgram = nil

	cmd := &cobra.Command{}
	_, err := runScanAdoptionPrompt(cmd, scanDeps{}, scanExecutionOutcome{})
	assert.Error(t, err)
}

func scanMatchFixture() scanMatch {
	return scanMatch{
		FileName:    "alpha.jar",
		Name:        "Alpha",
		ProjectID:   "alpha",
		Platform:    models.MODRINTH,
		Hash:        "hash",
		ReleaseDate: "2024-01-01T00:00:00Z",
		DownloadURL: "https://example.invalid/alpha.jar",
	}
}
