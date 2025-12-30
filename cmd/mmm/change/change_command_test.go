package change

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/spf13/cobra"

	"github.com/meza/minecraft-mod-manager/cmd/mmm/install"
	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/tui"
	"github.com/spf13/afero"
)

func TestChangeOptionsFromFlags(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("non-interactive", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")
	cmd.Flags().Bool("force", false, "")

	assert.NoError(t, cmd.Flags().Set("config", "/custom/modlist.json"))
	assert.NoError(t, cmd.Flags().Set("non-interactive", "true"))
	assert.NoError(t, cmd.Flags().Set("quiet", "true"))
	assert.NoError(t, cmd.Flags().Set("debug", "true"))
	assert.NoError(t, cmd.Flags().Set("force", "true"))

	opts, err := changeOptionsFromFlags(cmd, nil)
	assert.NoError(t, err)
	assert.Equal(t, "/custom/modlist.json", opts.ConfigPath)
	assert.True(t, opts.NonInteractive)
	assert.True(t, opts.Quiet)
	assert.True(t, opts.Debug)
	assert.True(t, opts.Force)
	assert.Equal(t, "latest", opts.GameVersion)

	withArg, err := changeOptionsFromFlags(cmd, []string{"1.21.1"})
	assert.NoError(t, err)
	assert.Equal(t, "1.21.1", withArg.GameVersion)
}

func TestApplyChangeCommandErrorPolicy(t *testing.T) {
	cmd := &cobra.Command{}

	applyChangeCommandErrorPolicy(cmd, clierrors.MarkHandled(assert.AnError))
	assert.True(t, cmd.SilenceErrors)
	assert.True(t, cmd.SilenceUsage)

	cmd2 := &cobra.Command{}
	applyChangeCommandErrorPolicy(cmd2, assert.AnError)
	assert.False(t, cmd2.SilenceErrors)
	assert.True(t, cmd2.SilenceUsage)
}

func TestWrapWriterWithFDPreservesFD(t *testing.T) {
	buf := &bytes.Buffer{}
	wrapped := wrapWriterWithFD(buf, fdWriter{fd: 42})

	fdAware, ok := wrapped.(interface{ Fd() uintptr })
	assert.True(t, ok)
	assert.Equal(t, uintptr(42), fdAware.Fd())

	wrappedFallback := wrapWriterWithFD(buf, buf)
	assert.Equal(t, buf, wrappedFallback)
}

func TestRecordChangeTelemetry(t *testing.T) {
	var captured telemetry.CommandTelemetry
	recordChangeTelemetry(func(payload telemetry.CommandTelemetry) {
		captured = payload
	}, changeOptions{
		Force:          true,
		GameVersion:    "1.21.1",
		NonInteractive: true,
	}, changeResult{
		TargetVersion:   "1.21.1",
		UnsupportedMods: []models.Mod{{ID: "a"}, {ID: "b"}},
		InstallResult: install.Result{
			InstalledCount: 2,
			UnmanagedFound: true,
		},
		ExitCode: 0,
	}, nil)

	assert.True(t, captured.Success)
	assert.Equal(t, 0, captured.ExitCode)
	assert.Equal(t, "change", captured.Command)
	assert.Equal(t, 2, captured.Extra["unsupportedMods"])
	assert.Equal(t, "1.21.1", captured.Extra["targetVersion"])
}

func TestRecordChangeTelemetryFailure(t *testing.T) {
	var captured telemetry.CommandTelemetry
	recordChangeTelemetry(func(payload telemetry.CommandTelemetry) {
		captured = payload
	}, changeOptions{}, changeResult{ExitCode: 2}, assert.AnError)

	assert.False(t, captured.Success)
	assert.Equal(t, 2, captured.ExitCode)
}

func TestRecordChangeTelemetryExitCodeOnly(t *testing.T) {
	var captured telemetry.CommandTelemetry
	recordChangeTelemetry(func(payload telemetry.CommandTelemetry) {
		captured = payload
	}, changeOptions{}, changeResult{ExitCode: 3}, nil)

	assert.False(t, captured.Success)
	assert.Equal(t, 3, captured.ExitCode)
}

func TestCommandWithRunnerPassesOptions(t *testing.T) {
	var received changeOptions
	cmd := commandWithRunner(func(_ context.Context, _ *cobra.Command, opts changeOptions, _ changeDeps) (changeResult, error) {
		received = opts
		return changeResult{}, nil
	})

	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("non-interactive", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")

	cmd.SetArgs([]string{"--force", "--config", "/tmp/modlist.json", "--non-interactive", "--quiet", "--debug", "1.20.4"})

	assert.NoError(t, cmd.ExecuteContext(context.Background()))
	assert.True(t, received.Force)
	assert.True(t, received.NonInteractive)
	assert.True(t, received.Quiet)
	assert.True(t, received.Debug)
	assert.Equal(t, "/tmp/modlist.json", received.ConfigPath)
	assert.Equal(t, "1.20.4", received.GameVersion)
}

func TestSetupChangeIOTUI(t *testing.T) {
	restore := tui.SetIsTerminalFuncForTesting(func(int) bool { return true })
	defer restore()

	cmd := &cobra.Command{}
	reader := fdReader{fd: 1}
	writer := fdWriter{fd: 1}
	cmd.SetIn(reader)
	cmd.SetOut(writer)
	cmd.SetErr(writer)

	ioConfig := setupChangeIO(cmd, changeOptions{})
	assert.NotNil(t, ioConfig.logProgram)
	assert.NotNil(t, ioConfig.testCmd)
	assert.NotNil(t, ioConfig.installCmd)
	assert.NoError(t, ioConfig.logProgram.Stop())
}

func TestChangeOptionsFromFlagsErrorsWithoutConfigFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("force", false, "")
	cmd.Flags().Bool("non-interactive", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")

	_, err := changeOptionsFromFlags(cmd, nil)
	assert.Error(t, err)
}

func TestChangeOptionsFromFlagsErrorsWithoutNonInteractiveFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("force", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")

	_, err := changeOptionsFromFlags(cmd, nil)
	assert.Error(t, err)
}

func TestChangeOptionsFromFlagsErrorsWithoutQuietFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("force", false, "")
	cmd.Flags().Bool("non-interactive", false, "")
	cmd.Flags().Bool("debug", false, "")

	_, err := changeOptionsFromFlags(cmd, nil)
	assert.Error(t, err)
}

func TestChangeOptionsFromFlagsErrorsWithoutDebugFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("force", false, "")
	cmd.Flags().Bool("non-interactive", false, "")
	cmd.Flags().Bool("quiet", false, "")

	_, err := changeOptionsFromFlags(cmd, nil)
	assert.Error(t, err)
}

func TestChangeOptionsFromFlagsErrorsWithoutForceFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("non-interactive", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")

	_, err := changeOptionsFromFlags(cmd, nil)
	assert.Error(t, err)
}

func TestNewChangeDepsEnablesColorWhenTerminal(t *testing.T) {
	restore := tui.SetIsTerminalFuncForTesting(func(int) bool { return true })
	defer restore()

	memFS := afero.NewMemMapFs()
	common := cmddeps.NewCommonDeps(&cobra.Command{}, cmddeps.CommonDepsOptions{FS: memFS})
	testCmd := &cobra.Command{}
	testCmd.SetOut(fdWriter{fd: 1})

	deps := newChangeDeps(common, testCmd, &cobra.Command{})
	assert.Equal(t, tui.ColorEnabled, deps.colorMode)

	path := "/file.txt"
	assert.NoError(t, afero.WriteFile(memFS, path, []byte("data"), 0644))
	assert.NoError(t, deps.removeFile(memFS, path))
}

func TestChangeCommandConstructed(t *testing.T) {
	cmd := Command()
	assert.Equal(t, "change [game_version]", cmd.Use)
}

func TestRunChangeCommandWithTUI(t *testing.T) {
	restore := tui.SetIsTerminalFuncForTesting(func(int) bool { return true })
	defer restore()

	cmd := commandWithRunner(func(context.Context, *cobra.Command, changeOptions, changeDeps) (changeResult, error) {
		return changeResult{}, nil
	})

	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("non-interactive", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")

	reader := fdReader{fd: 1}
	writer := fdWriter{fd: 1}
	cmd.SetIn(reader)
	cmd.SetOut(writer)
	cmd.SetErr(writer)
	cmd.SetArgs([]string{"--config", "./modlist.json"})

	assert.NoError(t, cmd.Execute())
}

func TestRunChangeCommandWithTUIError(t *testing.T) {
	restore := tui.SetIsTerminalFuncForTesting(func(int) bool { return true })
	defer restore()

	expectedErr := clierrors.MarkHandled(assert.AnError)
	cmd := commandWithRunner(func(context.Context, *cobra.Command, changeOptions, changeDeps) (changeResult, error) {
		return changeResult{ExitCode: 1}, expectedErr
	})

	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("non-interactive", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")

	reader := fdReader{fd: 1}
	writer := fdWriter{fd: 1}
	cmd.SetIn(reader)
	cmd.SetOut(writer)
	cmd.SetErr(writer)
	cmd.SetArgs([]string{"--config", "./modlist.json"})

	err := cmd.Execute()
	assert.Equal(t, expectedErr, err)
}

func TestRunChangeCommandOptionsError(t *testing.T) {
	cmd := commandWithRunner(func(context.Context, *cobra.Command, changeOptions, changeDeps) (changeResult, error) {
		t.Fatal("runner should not be called")
		return changeResult{}, nil
	})

	err := cmd.Execute()
	assert.Error(t, err)
}

func TestRunChangeCommandHandlesRunnerError(t *testing.T) {
	cmd := commandWithRunner(func(context.Context, *cobra.Command, changeOptions, changeDeps) (changeResult, error) {
		return changeResult{ExitCode: 2}, clierrors.MarkHandled(assert.AnError)
	})

	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("non-interactive", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")
	cmd.SetArgs([]string{"--config", "./modlist.json"})

	err := cmd.Execute()
	assert.Equal(t, clierrors.MarkHandled(assert.AnError), err)
	assert.True(t, cmd.SilenceErrors)
	assert.True(t, cmd.SilenceUsage)
}

type fdWriter struct {
	fd uintptr
}

func (writer fdWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (writer fdWriter) Fd() uintptr {
	return writer.fd
}

type fdReader struct {
	fd uintptr
}

func (reader fdReader) Read(p []byte) (int, error) {
	return 0, io.EOF
}

func (reader fdReader) Fd() uintptr {
	return reader.fd
}
