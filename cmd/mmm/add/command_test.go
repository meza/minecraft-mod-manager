package add

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
)

func TestCommandWithRunner_UsesRunnerAndParsesFlags(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	var gotOpts addOptions
	cmd := commandWithRunner(func(_ context.Context, _ *perf.Span, _ *cobra.Command, opts addOptions, _ addDeps) (telemetry.CommandTelemetry, error) {
		gotOpts = opts
		return telemetry.CommandTelemetry{Command: "add"}, nil
	})
	addPersistentFlagsForTesting(cmd)
	setCommandOutputForTesting(cmd)

	cmd.SetArgs([]string{"modrinth", "abc", "--version", "1.2.3", "--allow-version-fallback"})
	assert.NoError(t, cmd.Execute())
	assert.Equal(t, "modrinth", gotOpts.Platform)
	assert.Equal(t, "abc", gotOpts.ProjectID)
	assert.Equal(t, "1.2.3", gotOpts.Version)
	assert.True(t, gotOpts.AllowVersionFallback)
}

func TestCommandWithRunner_RunTeaUsesBubbleTeaProgram(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	var ran bool
	cmd := commandWithRunner(func(_ context.Context, _ *perf.Span, _ *cobra.Command, _ addOptions, deps addDeps) (telemetry.CommandTelemetry, error) {
		result, err := deps.runTea(runTeaModel{ran: &ran}, tea.WithInput(bytes.NewBuffer(nil)), tea.WithOutput(io.Discard), tea.WithoutRenderer())
		assert.NoError(t, err)
		_, ok := result.(runTeaModel)
		assert.True(t, ok)
		return telemetry.CommandTelemetry{Command: "add"}, nil
	})
	addPersistentFlagsForTesting(cmd)
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{"modrinth", "abc"})

	assert.NoError(t, cmd.Execute())
	assert.True(t, ran)
}

func TestCommandWithRunner_AbortedReturnsNil(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cmd := commandWithRunner(func(_ context.Context, _ *perf.Span, _ *cobra.Command, _ addOptions, _ addDeps) (telemetry.CommandTelemetry, error) {
		return telemetry.CommandTelemetry{Command: "add"}, errAborted
	})
	addPersistentFlagsForTesting(cmd)
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{"modrinth", "abc"})

	assert.NoError(t, cmd.Execute())
}

func TestCommandWithRunner_ErrorReturnsError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cmd := commandWithRunner(func(_ context.Context, _ *perf.Span, _ *cobra.Command, _ addOptions, _ addDeps) (telemetry.CommandTelemetry, error) {
		return telemetry.CommandTelemetry{Command: "add"}, errors.New("boom")
	})
	addPersistentFlagsForTesting(cmd)
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{"modrinth", "abc"})

	assert.Error(t, cmd.Execute())
}

func TestCommandWithRunnerMissingConfigFlagErrors(t *testing.T) {
	runE := commandWithRunner(func(_ context.Context, _ *perf.Span, _ *cobra.Command, _ addOptions, _ addDeps) (telemetry.CommandTelemetry, error) {
		return telemetry.CommandTelemetry{Command: "add"}, nil
	}).RunE

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, []string{"modrinth", "abc"}))
}

func TestCommandWithRunnerMissingQuietFlagErrors(t *testing.T) {
	runE := commandWithRunner(func(_ context.Context, _ *perf.Span, _ *cobra.Command, _ addOptions, _ addDeps) (telemetry.CommandTelemetry, error) {
		return telemetry.CommandTelemetry{Command: "add"}, nil
	}).RunE

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().StringP("config", "c", "modlist.json", "config")
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, []string{"modrinth", "abc"}))
}

func TestCommandWithRunnerMissingDebugFlagErrors(t *testing.T) {
	runE := commandWithRunner(func(_ context.Context, _ *perf.Span, _ *cobra.Command, _ addOptions, _ addDeps) (telemetry.CommandTelemetry, error) {
		return telemetry.CommandTelemetry{Command: "add"}, nil
	}).RunE

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().StringP("config", "c", "modlist.json", "config")
	cmd.Flags().BoolP("quiet", "q", false, "quiet")
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, []string{"modrinth", "abc"}))
}

func TestCommandWithRunnerMissingVersionFlagErrors(t *testing.T) {
	runE := commandWithRunner(func(_ context.Context, _ *perf.Span, _ *cobra.Command, _ addOptions, _ addDeps) (telemetry.CommandTelemetry, error) {
		return telemetry.CommandTelemetry{Command: "add"}, nil
	}).RunE

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().StringP("config", "c", "modlist.json", "config")
	cmd.Flags().BoolP("quiet", "q", false, "quiet")
	cmd.Flags().BoolP("debug", "d", false, "debug")
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, []string{"modrinth", "abc"}))
}

func TestCommandWithRunnerMissingAllowFallbackFlagErrors(t *testing.T) {
	runE := commandWithRunner(func(_ context.Context, _ *perf.Span, _ *cobra.Command, _ addOptions, _ addDeps) (telemetry.CommandTelemetry, error) {
		return telemetry.CommandTelemetry{Command: "add"}, nil
	}).RunE

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().StringP("config", "c", "modlist.json", "config")
	cmd.Flags().BoolP("quiet", "q", false, "quiet")
	cmd.Flags().BoolP("debug", "d", false, "debug")
	cmd.Flags().String("version", "", "version")
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, []string{"modrinth", "abc"}))
}

func TestCommand_ValidArgsFunctionSuggestsPlatforms(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cmd := commandWithRunner(func(context.Context, *perf.Span, *cobra.Command, addOptions, addDeps) (telemetry.CommandTelemetry, error) {
		return telemetry.CommandTelemetry{Command: "add"}, nil
	})
	addPersistentFlagsForTesting(cmd)
	setCommandOutputForTesting(cmd)

	args, _ := cmd.ValidArgsFunction(cmd, []string{}, "")
	assert.Contains(t, args, string(models.CURSEFORGE))
	assert.Contains(t, args, string(models.MODRINTH))

	args, _ = cmd.ValidArgsFunction(cmd, []string{"modrinth"}, "")
	assert.Nil(t, args)
}

func addPersistentFlagsForTesting(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("config", "c", "./modlist.json", "An alternative JSON file containing the configuration")
	cmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress all output")
	cmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug messages")
}

func setCommandOutputForTesting(cmd *cobra.Command) {
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
}

type runTeaModel struct {
	ran *bool
}

func (m runTeaModel) Init() tea.Cmd {
	return func() tea.Msg { return "start" }
}

func (m runTeaModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	if m.ran != nil {
		*m.ran = true
	}
	return m, tea.Quit
}

func (m runTeaModel) View() string { return "" }
