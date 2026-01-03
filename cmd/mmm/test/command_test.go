package test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/tui"
)

func TestCommandWithRunnerMissingConfigFlagErrors(t *testing.T) {
	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (int, error) {
		return 0, nil
	})
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{})

	assert.Error(t, cmd.Execute())
}

func TestCommandWithRunnerMissingUnattendedFlagErrors(t *testing.T) {
	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (int, error) {
		return 0, nil
	})
	cmd.PersistentFlags().StringP("config", "c", "modlist.json", "config")
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{})

	assert.Error(t, cmd.Execute())
}

func TestCommandWithRunnerMissingQuietFlagErrors(t *testing.T) {
	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (int, error) {
		return 0, nil
	})
	cmd.PersistentFlags().StringP("config", "c", "modlist.json", "config")
	cmd.PersistentFlags().Bool("unattended", false, "unattended")
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{})

	assert.Error(t, cmd.Execute())
}

func TestCommandWithRunnerMissingDebugFlagErrors(t *testing.T) {
	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (int, error) {
		return 0, nil
	})
	cmd.PersistentFlags().StringP("config", "c", "modlist.json", "config")
	cmd.PersistentFlags().Bool("unattended", false, "unattended")
	cmd.PersistentFlags().BoolP("quiet", "q", false, "quiet")
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{})

	assert.Error(t, cmd.Execute())
}

func TestCommandWithRunnerSuccess(t *testing.T) {
	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (int, error) {
		return 0, nil
	})
	addPersistentFlagsForTesting(cmd)
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{"1.20.1"})

	assert.NoError(t, cmd.Execute())
	assert.False(t, cmd.SilenceUsage)
	assert.False(t, cmd.SilenceErrors)
}

func TestCommandWithRunnerExitCodeErrorSilencesErrors(t *testing.T) {
	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (int, error) {
		return 2, clierrors.MarkHandled(errSameVersion)
	})
	addPersistentFlagsForTesting(cmd)
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{})

	assert.Error(t, cmd.Execute())
	assert.True(t, cmd.SilenceUsage)
	assert.True(t, cmd.SilenceErrors)
}

func TestCommandWithRunnerInvalidVersionSilencesErrors(t *testing.T) {
	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (int, error) {
		return 1, clierrors.MarkHandled(errInvalidVersion)
	})
	addPersistentFlagsForTesting(cmd)
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{"1.20.1"})

	assert.Error(t, cmd.Execute())
	assert.True(t, cmd.SilenceUsage)
	assert.True(t, cmd.SilenceErrors)
}

func TestCommandWithRunnerLatestVersionRequiredSilencesErrors(t *testing.T) {
	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (int, error) {
		return 1, clierrors.MarkHandled(errLatestVersionRequired)
	})
	addPersistentFlagsForTesting(cmd)
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{})

	assert.Error(t, cmd.Execute())
	assert.True(t, cmd.SilenceUsage)
	assert.True(t, cmd.SilenceErrors)
}

func TestCommandWithRunnerGenericErrorDoesNotSilence(t *testing.T) {
	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (int, error) {
		return 1, errors.New("boom")
	})
	addPersistentFlagsForTesting(cmd)
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{"1.20.1"})

	assert.Error(t, cmd.Execute())
	assert.True(t, cmd.SilenceUsage)
	assert.False(t, cmd.SilenceErrors)
}

func TestCommandWithRunnerUsesPlainOutputWhenTerminal(t *testing.T) {
	restore := tui.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	defer restore()

	outputWriter := &terminalWriter{}
	cmd := commandWithRunner(func(_ context.Context, _ *cobra.Command, _ testOptions, deps testDeps) (int, error) {
		return 0, deps.output.Log("hello", output.LogForce)
	})
	addPersistentFlagsForTesting(cmd)
	cmd.SetIn(terminalReader{Reader: bytes.NewBuffer(nil)})
	cmd.SetOut(outputWriter)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"1.20.1"})

	assert.NoError(t, cmd.Execute())
	assert.Equal(t, "hello\n", outputWriter.String())
}

func TestCommandWithRunnerQuietSuppressesOutput(t *testing.T) {
	restore := tui.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	defer restore()

	outputWriter := &terminalWriter{}
	cmd := commandWithRunner(func(_ context.Context, _ *cobra.Command, _ testOptions, deps testDeps) (int, error) {
		return 0, deps.output.Log("quiet", output.LogQuiet)
	})
	addPersistentFlagsForTesting(cmd)
	cmd.SetIn(terminalReader{Reader: bytes.NewBuffer(nil)})
	cmd.SetOut(outputWriter)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--unattended", "--quiet", "1.20.1"})

	assert.NoError(t, cmd.Execute())
	assert.Empty(t, outputWriter.String())
}

func TestExitCodeErrorMessage(t *testing.T) {
	err := &exitCodeError{code: 3}
	assert.Equal(t, "exit code 3", err.Error())
}

func addPersistentFlagsForTesting(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("config", "c", "./modlist.json", "An alternative JSON file containing the configuration")
	cmd.PersistentFlags().Bool("unattended", false, "Disable prompts and fail fast when required inputs are missing")
	cmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress non-essential output (errors and required results still print)")
	cmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug messages")
}

func setCommandOutputForTesting(cmd *cobra.Command) {
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
}

type terminalReader struct {
	io.Reader
}

func (terminalReader) Fd() uintptr {
	return 0
}

type terminalWriter struct {
	bytes.Buffer
}

func (writer *terminalWriter) Fd() uintptr {
	return 1
}
