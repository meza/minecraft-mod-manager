package test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCommandWithRunnerMissingConfigFlagErrors(t *testing.T) {
	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (int, error) {
		return 0, nil
	})
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{})

	assert.Error(t, cmd.Execute())
}

func TestCommandWithRunnerMissingQuietFlagErrors(t *testing.T) {
	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (int, error) {
		return 0, nil
	})
	cmd.PersistentFlags().StringP("config", "c", "modlist.json", "config")
	setCommandOutputForTesting(cmd)
	cmd.SetArgs([]string{})

	assert.Error(t, cmd.Execute())
}

func TestCommandWithRunnerMissingDebugFlagErrors(t *testing.T) {
	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (int, error) {
		return 0, nil
	})
	cmd.PersistentFlags().StringP("config", "c", "modlist.json", "config")
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
		return 2, errSameVersion
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
		return 1, errInvalidVersion
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
		return 1, errLatestVersionRequired
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

func TestExitCodeErrorMessage(t *testing.T) {
	err := &exitCodeError{code: 3}
	assert.Equal(t, "exit code 3", err.Error())
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
