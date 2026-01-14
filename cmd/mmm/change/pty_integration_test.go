package change

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/meza/minecraft-mod-manager/testutil/terminal"
	terminalpty "github.com/meza/minecraft-mod-manager/testutil/terminal/pty"
)

func TestChangeCommandInteractivePTYOutput(t *testing.T) {
	terminal.ApplyFixtures(t)

	originalRunChangeProgram := runChangeProgram
	runChangeProgram = defaultRunChangeProgram
	t.Cleanup(func() { runChangeProgram = originalRunChangeProgram })

	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: 120, Rows: 40}))
	require.NotNil(t, session)

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

	cmd.SetIn(session.Input())
	cmd.SetOut(session.Output())
	cmd.SetErr(session.Output())
	cmd.SetArgs([]string{"1.21.1"})

	execErr := cmd.Execute()
	require.NoError(t, execErr)
	session.WaitForOutputAndClose(t, func(output []byte) bool {
		return strings.Contains(string(output), "cmd.change.header")
	}, terminalpty.WithWaitDuration(2*time.Second))

	normalized := terminal.NormalizeOutput(session.OutputString(), terminal.NormalizeOptions{StripControlSequences: true})

	require.Contains(t, normalized, "cmd.change.header")
	require.Contains(t, normalized, "cmd.change.notice")
	require.Contains(t, normalized, "cmd.compatibility.section")
	require.Contains(t, normalized, "cmd.change.section.downloading")
	require.NotContains(t, normalized, "cmd.change.section.switching")
}
