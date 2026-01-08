package scan

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modsetup"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestScanInitCanceledOutputSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	originalRunner := runInteractiveInit
	runInteractiveInit = func(context.Context, *cobra.Command, initCmd.InteractiveInitDeps, initCmd.InteractiveInitOptions) error {
		return initCmd.ErrInitCanceled
	}
	t.Cleanup(func() {
		runInteractiveInit = originalRunner
	})

	configPath := filepath.FromSlash("./modlist.json")

	out := fakeTTY{Buffer: &bytes.Buffer{}}
	errOut := fakeTTY{Buffer: &bytes.Buffer{}}
	in := fakeTTY{Buffer: &bytes.Buffer{}}
	_, err := in.WriteString("cmd.init.prompt.option.yes.short\n")
	assert.NoError(t, err)

	originalRunTea := runTeaProgram
	runTeaProgram = func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		switch typed := model.(type) {
		case scanConfigInitModel:
			typed.confirmed = true
			typed.prompt.Value = typed.prompt.yesOption.short
			typed.prompt.input.SetValue(typed.prompt.Value)
			_, writeErr := out.WriteString(typed.View())
			return typed, writeErr
		case *scanConfigInitModel:
			typed.confirmed = true
			typed.prompt.Value = typed.prompt.yesOption.short
			typed.prompt.input.SetValue(typed.prompt.Value)
			_, writeErr := out.WriteString(typed.View())
			return typed, writeErr
		default:
			viewText := model.View()
			if strings.TrimSpace(viewText) != "" {
				if _, writeErr := out.WriteString(viewText); writeErr != nil {
					return model, writeErr
				}
			}
			return model, nil
		}
	}
	t.Cleanup(func() {
		runTeaProgram = originalRunTea
	})

	cmd := Command()
	addPersistentFlagsForTesting(cmd)
	cmd.SetIn(in)
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	cmd.SetArgs([]string{"--config", configPath})

	err = cmd.Execute()
	assert.NoError(t, err)

	snapshot := fmt.Sprintf(
		"stdout:\n%s\nstderr:\n%s",
		normalizeSnapshotOutput(strings.TrimSpace(out.String())),
		strings.TrimSpace(errOut.String()),
	)
	snaps.MatchSnapshot(t, snapshot)
}

func TestScanMissingConfigUnattendedOutputSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return false })
	t.Cleanup(restoreTerminal)

	configPath := filepath.Join(t.TempDir(), "modlist.json")

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := Command()
	addPersistentFlagsForTesting(cmd)
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	cmd.SetArgs([]string{"--config", configPath, "--unattended"})

	err := cmd.Execute()
	assert.Error(t, err)

	normalizedOut := normalizeSnapshotOutput(strings.TrimSpace(out.String()))
	normalizedOut = strings.ReplaceAll(normalizedOut, normalizeSnapshotOutput(configPath), "<configPath>")

	snapshot := fmt.Sprintf(
		"stdout:\n%s\nstderr:\n%s",
		normalizedOut,
		strings.TrimSpace(errOut.String()),
	)
	snaps.MatchSnapshot(t, snapshot)
}

func TestScanInvalidPreferOutputSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return false })
	t.Cleanup(restoreTerminal)

	fs := afero.NewOsFs()
	configPath := filepath.Join(t.TempDir(), "modlist.json")
	meta := config.NewMetadata(configPath)
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := Command()
	addPersistentFlagsForTesting(cmd)
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	cmd.SetArgs([]string{"--config", configPath, "--prefer", "unknown"})

	err := cmd.Execute()
	assert.Error(t, err)

	snapshot := fmt.Sprintf(
		"stdout:\n%s\nstderr:\n%s",
		normalizeSnapshotOutput(strings.TrimSpace(out.String())),
		strings.TrimSpace(errOut.String()),
	)
	snaps.MatchSnapshot(t, snapshot)
}

func TestScanCanceledInteractiveOutputSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	originalRunScanProgram := runScanProgram
	runScanProgram = func(model *scanModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = scanExecutionOutcome{err: context.Canceled}
		return model, nil
	}
	t.Cleanup(func() {
		runScanProgram = originalRunScanProgram
	})

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	outcome, err := runInteractiveScan(context.Background(), cmd, scanExecutionInput{
		candidates: []scanCandidate{{Path: "/mods/alpha.jar", FileName: "alpha.jar", Sha1: "alpha"}},
	}, scanOptions{})
	assert.NoError(t, err)
	assert.Empty(t, outcome.matches)

	snapshot := fmt.Sprintf(
		"stdout:\n%s\nstderr:\n%s",
		strings.TrimSpace(out.String()),
		strings.TrimSpace(errOut.String()),
	)
	snaps.MatchSnapshot(t, snapshot)
}

func TestScanFullAdoptionOutputSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return false })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/cfg", 0o755))
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	prompt := newScanConfirmPromptModel("Question?")
	prompt.Value = prompt.yesOption.short
	prompt.input.SetValue(prompt.Value)

	input := scanExecutionInput{
		meta:             meta,
		cfg:              cfg,
		lock:             []models.ModInstall{},
		setupCoordinator: modsetup.NewSetupCoordinator(fs, nil, nil),
		deps: scanDeps{
			fs: fs,
			runTea: runTeaStub(runTeaScenario{
				adoptionResult: &scanAdoptionPromptModel{confirmed: true, prompt: prompt},
			}),
		},
	}
	first := scanMatchFixture()
	second := scanMatchFixture()
	second.FileName = "beta.jar"
	second.Name = "Beta"
	second.ProjectID = "beta"
	second.Hash = "hash-2"
	second.ReleaseDate = "2024-01-02T00:00:00Z"
	second.DownloadURL = "https://example.invalid/beta.jar"
	outcome := scanExecutionOutcome{matches: []scanMatch{first, second}}

	_, err := handleInteractiveScanPrompt(context.Background(), cmd, input, view.ColorDisabled, outcome)
	assert.NoError(t, err)

	snapshot := fmt.Sprintf("stdout:\n%s\nstderr:\n", normalizeSnapshotOutput(strings.TrimSpace(out.String())))
	snaps.MatchSnapshot(t, snapshot)
}

func TestScanFullAdoptionAddOutputSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return false })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/cfg", 0o755))
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	input := scanExecutionInput{
		meta:             meta,
		cfg:              cfg,
		lock:             []models.ModInstall{},
		setupCoordinator: modsetup.NewSetupCoordinator(fs, nil, nil),
		deps: scanDeps{
			fs:     fs,
			runTea: runTeaStub(runTeaScenario{}),
		},
	}

	outcome := scanExecutionOutcome{matches: []scanMatch{scanMatchFixture()}}
	_, err := handleInteractiveScanAdd(context.Background(), cmd, input, view.ColorDisabled, outcome)
	assert.NoError(t, err)

	snapshot := fmt.Sprintf("stdout:\n%s\nstderr:\n", normalizeSnapshotOutput(strings.TrimSpace(out.String())))
	snaps.MatchSnapshot(t, snapshot)
}

func TestScanAdoptionCancelledOutputSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return false })
	t.Cleanup(restoreTerminal)

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
	outcome := scanExecutionOutcome{
		matches: []scanMatch{{FileName: "alpha.jar", Name: "Alpha", ProjectID: "alpha", Platform: models.MODRINTH}},
	}

	_, err := handleInteractiveScanPrompt(context.Background(), cmd, input, view.ColorDisabled, outcome)
	assert.NoError(t, err)

	snapshot := fmt.Sprintf("stdout:\n%s\nstderr:\n", normalizeSnapshotOutput(strings.TrimSpace(out.String())))
	snaps.MatchSnapshot(t, snapshot)
}

func normalizeSnapshotOutput(value string) string {
	return strings.ReplaceAll(value, "\\", "/")
}
