//go:build !windows

package change

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestChangeCommandInteractivePTYOutput(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)

	originalRunChangeProgram := runChangeProgram
	runChangeProgram = defaultRunChangeProgram
	t.Cleanup(func() { runChangeProgram = originalRunChangeProgram })

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 120, Rows: 40}))

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, _ changeOptions, _ changeDeps) (changeResult, error) {
		items := []changeItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, DisplayName: "Alpha"},
			{Mod: models.Mod{ID: "beta", Name: "Beta", Type: models.CURSEFORGE}, DisplayName: "Beta"},
		}
		index := map[string]int{
			changeModKey(items[0].Mod): 0,
			changeModKey(items[1].Mod): 1,
		}
		key := changeModKey(items[1].Mod)

		updatedItems := cloneChangeItems(items)
		updatedIndex := index[key]
		updatedItems[updatedIndex].CompatStatus = changeCompatSupported
		updatedItems[updatedIndex].DownloadStatus = changeDownloadInProgress
		updatedItems[updatedIndex].Download = changeDownloadProgress{
			ratio:      0.5,
			downloaded: 512 * 1024,
			total:      1024 * 1024,
		}

		execRunner := func(ctx context.Context, sender httpclient.Sender) changeOutcome {
			if sender != nil {
				sender.Send(changeCompatResultMsg{key: key, supported: true, resolvedName: "Beta"})
				sender.Send(changeDownloadProgressMsg{
					key: key,
					progress: httpclient.DownloadProgressMsg{
						Ratio:      0.5,
						Downloaded: 512 * 1024,
						Total:      1024 * 1024,
					},
				})
			}
			return changeOutcome{
				Stage:         changeStageRunning,
				TargetVersion: "1.21.1",
				Items:         updatedItems,
			}
		}

		model := newChangeModel(changeModelInput{
			ctx:        ctx,
			target:     "1.21.1",
			colorMode:  colorModeForOutput(cmd.OutOrStdout()),
			items:      items,
			indexByKey: index,
			execRunner: execRunner,
		})

		result, err := runChangeProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
		if err != nil {
			return changeResult{ExitCode: 1, Interactive: true}, err
		}

		outcome, err := changeOutcomeFromModel(result)
		if err != nil {
			return changeResult{ExitCode: 1, Interactive: true}, err
		}
		if outcome.Err != nil {
			return changeResult{ExitCode: 1, Interactive: true}, outcome.Err
		}
		return changeResult{ExitCode: 0, Interactive: true}, nil
	})

	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("unattended", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)
	cmd.SetArgs([]string{"1.21.1"})

	var output bytes.Buffer
	readDone := make(chan struct{})
	readErr := make(chan error, 1)
	go func() {
		_, err := io.Copy(&output, master)
		readErr <- err
		close(readDone)
	}()

	execErr := cmd.Execute()
	require.NoError(t, execErr)
	closePTY(t, slave)

	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PTY output")
	}
	require.NoError(t, normalizePTYReadError(<-readErr))

	normalized := stripControlSequences(output.String())

	require.Contains(t, normalized, "cmd.change.header")
	require.Contains(t, normalized, "cmd.change.notice")
	require.Contains(t, normalized, "cmd.change.section.compatibility")
	require.Contains(t, normalized, "cmd.change.section.downloading")
	require.NotContains(t, normalized, "cmd.change.section.switching")
}

func stripControlSequences(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")

	value = stripOSCSequences(value)
	value = stripCSISequences(value)

	return value
}

func closePTY(t *testing.T, file *os.File) {
	if file == nil {
		return
	}
	if err := file.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		require.NoError(t, err)
	}
}

func normalizePTYReadError(err error) error {
	if err == nil || errors.Is(err, os.ErrClosed) || errors.Is(err, syscall.EIO) {
		return nil
	}
	return err
}

var oscSequence = regexp.MustCompile(`\x1b\][^\x07]*(\x07|\x1b\\)`)

func stripOSCSequences(value string) string {
	return oscSequence.ReplaceAllString(value, "")
}

var csiSequence = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

func stripCSISequences(value string) string {
	return csiSequence.ReplaceAllString(value, "")
}
