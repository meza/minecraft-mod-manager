package test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/locksync"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCommandWithRunnerMissingConfigFlagErrors(t *testing.T) {
	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (Result, error) {
		return Result{}, nil
	})
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{})

	assert.Error(t, cmd.Execute())
}

func TestCommandWithRunnerMissingUnattendedFlagErrors(t *testing.T) {
	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (Result, error) {
		return Result{}, nil
	})
	cmd.PersistentFlags().StringP("config", "c", "modlist.json", "config")
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{})

	assert.Error(t, cmd.Execute())
}

func TestCommandWithRunnerMissingQuietFlagErrors(t *testing.T) {
	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (Result, error) {
		return Result{}, nil
	})
	cmd.PersistentFlags().StringP("config", "c", "modlist.json", "config")
	cmd.PersistentFlags().Bool("unattended", false, "unattended")
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{})

	assert.Error(t, cmd.Execute())
}

func TestCommandWithRunnerMissingDebugFlagErrors(t *testing.T) {
	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (Result, error) {
		return Result{}, nil
	})
	cmd.PersistentFlags().StringP("config", "c", "modlist.json", "config")
	cmd.PersistentFlags().Bool("unattended", false, "unattended")
	cmd.PersistentFlags().BoolP("quiet", "q", false, "quiet")
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{})

	assert.Error(t, cmd.Execute())
}

func TestCommandWithRunnerSuccess(t *testing.T) {
	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (Result, error) {
		return Result{}, nil
	})
	addPersistentFlagsForTesting(cmd)
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{"1.20.1"})

	assert.NoError(t, cmd.Execute())
	assert.False(t, cmd.SilenceUsage)
	assert.False(t, cmd.SilenceErrors)
}

func TestCommandWithRunnerInvalidVersionSilencesErrors(t *testing.T) {
	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (Result, error) {
		return Result{ExitCode: 1}, clierrors.MarkHandled(errInvalidVersion)
	})
	addPersistentFlagsForTesting(cmd)
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{"1.20.1"})

	assert.Error(t, cmd.Execute())
	assert.True(t, cmd.SilenceUsage)
	assert.True(t, cmd.SilenceErrors)
}

func TestCommandWithRunnerLatestVersionRequiredSilencesErrors(t *testing.T) {
	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (Result, error) {
		return Result{ExitCode: 1}, clierrors.MarkHandled(errLatestVersionRequired)
	})
	addPersistentFlagsForTesting(cmd)
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{})

	assert.Error(t, cmd.Execute())
	assert.True(t, cmd.SilenceUsage)
	assert.True(t, cmd.SilenceErrors)
}

func TestCommandWithRunnerGenericErrorDoesNotSilence(t *testing.T) {
	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (Result, error) {
		return Result{ExitCode: 1}, errors.New("boom")
	})
	addPersistentFlagsForTesting(cmd)
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{"1.20.1"})

	assert.Error(t, cmd.Execute())
	assert.True(t, cmd.SilenceUsage)
	assert.False(t, cmd.SilenceErrors)
}

func TestCommandWithRunnerUsesPlainOutputWhenNonTTY(t *testing.T) {
	outputWriter := &bytes.Buffer{}
	cmd := commandWithRunner(func(_ context.Context, _ *cobra.Command, _ testOptions, deps testDeps) (Result, error) {
		return Result{}, deps.output.Log("hello", output.LogForce)
	})
	addPersistentFlagsForTesting(cmd)
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(outputWriter)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"1.20.1"})

	assert.NoError(t, cmd.Execute())
	assert.Equal(t, "hello\n", outputWriter.String())
}

func TestCommandWithRunnerQuietSuppressesOutput(t *testing.T) {
	outputWriter := &bytes.Buffer{}
	cmd := commandWithRunner(func(_ context.Context, _ *cobra.Command, _ testOptions, deps testDeps) (Result, error) {
		return Result{}, deps.output.Log("quiet", output.LogQuiet)
	})
	addPersistentFlagsForTesting(cmd)
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(outputWriter)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--unattended", "--quiet", "1.20.1"})

	assert.NoError(t, cmd.Execute())
	assert.Empty(t, outputWriter.String())
}

func TestTestOptionsFromFlagsReturnsLockSyncFlagError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("unattended", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")
	cmd.Flags().String(locksync.FlagAdd, "", "")

	_, err := testOptionsFromFlags(cmd, []string{"1.21.1"})
	assert.Error(t, err)
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
