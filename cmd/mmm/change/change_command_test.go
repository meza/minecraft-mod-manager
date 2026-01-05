package change

import (
	"context"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
)

func TestChangeOptionsFromFlags(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("unattended", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")
	cmd.Flags().Bool("force", false, "")

	assert.NoError(t, cmd.Flags().Set("config", "/custom/modlist.json"))
	assert.NoError(t, cmd.Flags().Set("unattended", "true"))
	assert.NoError(t, cmd.Flags().Set("quiet", "true"))
	assert.NoError(t, cmd.Flags().Set("debug", "true"))
	assert.NoError(t, cmd.Flags().Set("force", "true"))

	opts, err := changeOptionsFromFlags(cmd, nil)
	assert.NoError(t, err)
	assert.Equal(t, "/custom/modlist.json", opts.ConfigPath)
	assert.True(t, opts.Unattended)
	assert.True(t, opts.Quiet)
	assert.True(t, opts.Debug)
	assert.True(t, opts.Force)
	assert.Equal(t, "latest", opts.GameVersion)

	withArg, err := changeOptionsFromFlags(cmd, []string{"1.21.1"})
	assert.NoError(t, err)
	assert.Equal(t, "1.21.1", withArg.GameVersion)
}

func TestChangeOptionsFromFlagsErrors(t *testing.T) {
	cases := []struct {
		name  string
		flags func(cmd *cobra.Command)
	}{
		{
			name: "missing config",
			flags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("force", false, "")
				cmd.Flags().Bool("unattended", false, "")
				cmd.Flags().Bool("quiet", false, "")
				cmd.Flags().Bool("debug", false, "")
			},
		},
		{
			name: "missing unattended",
			flags: func(cmd *cobra.Command) {
				cmd.Flags().String("config", "./modlist.json", "")
				cmd.Flags().Bool("force", false, "")
				cmd.Flags().Bool("quiet", false, "")
				cmd.Flags().Bool("debug", false, "")
			},
		},
		{
			name: "missing quiet",
			flags: func(cmd *cobra.Command) {
				cmd.Flags().String("config", "./modlist.json", "")
				cmd.Flags().Bool("force", false, "")
				cmd.Flags().Bool("unattended", false, "")
				cmd.Flags().Bool("debug", false, "")
			},
		},
		{
			name: "missing debug",
			flags: func(cmd *cobra.Command) {
				cmd.Flags().String("config", "./modlist.json", "")
				cmd.Flags().Bool("force", false, "")
				cmd.Flags().Bool("unattended", false, "")
				cmd.Flags().Bool("quiet", false, "")
			},
		},
		{
			name: "missing force",
			flags: func(cmd *cobra.Command) {
				cmd.Flags().String("config", "./modlist.json", "")
				cmd.Flags().Bool("unattended", false, "")
				cmd.Flags().Bool("quiet", false, "")
				cmd.Flags().Bool("debug", false, "")
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			testCase.flags(cmd)
			_, err := changeOptionsFromFlags(cmd, nil)
			assert.Error(t, err)
		})
	}
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

func TestRecordChangeTelemetry(t *testing.T) {
	var captured telemetry.CommandTelemetry
	recordChangeTelemetry(func(payload telemetry.CommandTelemetry) {
		captured = payload
	}, changeOptions{
		Force:       true,
		GameVersion: "1.21.1",
		Unattended:  true,
	}, changeResult{
		TargetVersion:  "1.21.1",
		TotalMods:      2,
		SkippedMods:    1,
		DownloadedMods: 1,
		ExitCode:       0,
		Interactive:    true,
	}, nil)

	assert.True(t, captured.Success)
	assert.Equal(t, 0, captured.ExitCode)
	assert.Equal(t, "change", captured.Command)
	assert.Equal(t, 2, captured.Extra["totalMods"])
	assert.Equal(t, 1, captured.Extra["skippedMods"])
	assert.Equal(t, 1, captured.Extra["downloadedMods"])
	assert.Equal(t, "1.21.1", captured.Extra["targetVersion"])
	assert.True(t, captured.Interactive)
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
	cmd.Flags().Bool("unattended", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")

	cmd.SetArgs([]string{"--force", "--config", "/tmp/modlist.json", "--unattended", "--quiet", "--debug", "1.20.4"})

	assert.NoError(t, cmd.ExecuteContext(context.Background()))
	assert.True(t, received.Force)
	assert.True(t, received.Unattended)
	assert.True(t, received.Quiet)
	assert.True(t, received.Debug)
	assert.Equal(t, "/tmp/modlist.json", received.ConfigPath)
	assert.Equal(t, "1.20.4", received.GameVersion)
}

func TestNewChangeDepsDefaults(t *testing.T) {
	memFS := afero.NewMemMapFs()
	common := cmddeps.NewCommonDeps(&cobra.Command{}, cmddeps.CommonDepsOptions{FS: memFS})
	deps := newChangeDeps(common)

	require.NotNil(t, deps.downloader)
	require.NotNil(t, deps.fetchMod)
	require.NotNil(t, deps.latestVersion)
	require.NotNil(t, deps.isValidVersion)

	path := "/file.txt"
	require.NoError(t, afero.WriteFile(memFS, path, []byte("data"), 0644))
	require.NoError(t, deps.removeFile(memFS, path))

	dir := "/dir"
	require.NoError(t, deps.mkdirAll(memFS, dir, 0o755))
	require.NoError(t, afero.WriteFile(memFS, "/source.txt", []byte("data"), 0644))
	require.NoError(t, deps.renameFile(memFS, "/source.txt", "/dest.txt"))
	require.NoError(t, deps.removeAll(memFS, dir))
}

func TestCommandConstructed(t *testing.T) {
	cmd := Command()
	assert.Equal(t, "change [game_version]", cmd.Use)
}

func TestChangeResultCounts(t *testing.T) {
	items := []changeItem{
		{DownloadStatus: changeDownloadSucceeded},
		{DownloadStatus: changeDownloadFailed, Skipped: true},
	}
	assert.Equal(t, 1, countDownloaded(items))
	assert.Equal(t, 1, countSkipped(items))
}

func TestChangeModKey(t *testing.T) {
	mod := models.Mod{Type: models.MODRINTH, ID: "abc"}
	assert.Equal(t, "modrinth:abc", changeModKey(mod))
}
