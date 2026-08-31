package install

import (
	"context"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/meza/minecraft-mod-manager/testutil/terminal"
	terminalpty "github.com/meza/minecraft-mod-manager/testutil/terminal/pty"
)

func TestInstallCommandInteractivePTYFinalSnapshotLongListMediumHeight(t *testing.T) {
	runInstallPTYFinalSnapshotWithMods(t, 25, sampleInstallModsLongList())
}

func TestInstallCommandInteractivePTYFinalSnapshotLongListTallHeight(t *testing.T) {
	runInstallPTYFinalSnapshotWithMods(t, installTallSnapshotRows(), sampleInstallModsLongList())
}

func TestInstallCommandInteractivePTYRunningSnapshotLongListShortHeight(t *testing.T) {
	runInstallPTYRunningSnapshotWithMods(t, 12, sampleInstallModsLongList())
}

func runInstallPTYRunningSnapshotWithMods(t *testing.T, rows uint16, cfg models.ModsJSON) {
	t.Helper()
	terminal.ApplyFixtures(t)

	originalRunInstallProgram := runInstallProgram
	runInstallProgram = defaultRunInstallProgram
	t.Cleanup(func() { runInstallProgram = originalRunInstallProgram })

	items, _ := buildInstallItems(cfg)
	lastItem := items[len(items)-1].DisplayName

	columns := 120
	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: columns, Rows: int(rows)}))
	require.NotNil(t, session)

	release := make(chan struct{})
	updatesSent := make(chan struct{})

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, _ installOptions, _ installDeps) (Result, error) {
		installItems, indexByKey := buildInstallItems(cfg)
		execRunner := func(ctx context.Context, sender httpclient.Sender) installExecutionOutcome {
			for _, item := range installItems {
				sender.Send(installItemSuccessMsg{
					key:         installModKey(item.Mod),
					displayName: item.DisplayName,
				})
			}
			close(updatesSent)
			<-release
			return installExecutionOutcome{items: installItems, errType: installExecutionErrorNone}
		}

		model := newInstallModel(ctx, colorModeForOutput(cmd.OutOrStdout()), installItems, indexByKey, nil, execRunner, nil)
		model.windowW = columns
		model.windowH = int(rows)
		options := view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())
		_, err := runInstallProgram(model, options...)
		if err != nil {
			return Result{}, err
		}
		return Result{InstalledCount: len(items)}, model.outcome.err
	})

	addPersistentFlagsForTesting(cmd)
	cmd.SetIn(session.Input())
	cmd.SetOut(session.Output())
	cmd.SetErr(session.Output())
	cmd.SetArgs([]string{})

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	select {
	case <-updatesSent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for install running updates")
	}

	expectedLine := view.SuccessIcon(view.ColorDisabled) + " " + lastItem
	session.WaitForOutput(t, func(output []byte) bool {
		normalized := terminal.NormalizeOutput(string(output), terminal.NormalizeOptions{
			StripControlSequences: true,
		})
		return strings.Contains(normalized, expectedLine)
	}, terminalpty.WithWaitDuration(2*time.Second))

	rawOutput := terminal.NormalizeOutput(session.OutputString(), terminal.NormalizeOptions{
		StripControlSequences: true,
	})
	require.Contains(t, rawOutput, "cmd.install.header.success")
	visibleOutput := normalizeInstallVisibleOutput(session.OutputString(), rows)
	require.Contains(t, visibleOutput, "cmd.install.header.success")
	for index := range items {
		items[index].Status = installItemSuccess
	}
	normalized := normalizeInstallRunningOutput(session.OutputString(), items)
	require.Contains(t, normalized, "cmd.install.header.success")
	snaps.MatchSnapshot(t, normalized)

	close(release)
	select {
	case err := <-execErr:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for install command to finish")
	}
	require.NoError(t, session.Close())
}

func runInstallPTYFinalSnapshotWithMods(t *testing.T, rows uint16, cfg models.ModsJSON) {
	t.Helper()
	terminal.ApplyFixtures(t)

	originalRunInstallProgram := runInstallProgram
	runInstallProgram = defaultRunInstallProgram
	t.Cleanup(func() { runInstallProgram = originalRunInstallProgram })

	columns := 120
	if rows == installTallSnapshotRows() {
		columns = 80
	}
	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: columns, Rows: int(rows)}))
	require.NotNil(t, session)
	finalViewOutput := ""

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, _ installOptions, _ installDeps) (Result, error) {
		items, indexByKey := buildInstallItems(cfg)
		execRunner := func(ctx context.Context, sender httpclient.Sender) installExecutionOutcome {
			for _, item := range items {
				sender.Send(installItemSuccessMsg{
					key:         installModKey(item.Mod),
					displayName: item.DisplayName,
				})
			}
			return installExecutionOutcome{items: items, errType: installExecutionErrorNone}
		}

		model := newInstallModel(ctx, colorModeForOutput(cmd.OutOrStdout()), items, indexByKey, nil, execRunner, nil)
		model.windowW = columns
		model.windowH = int(rows)
		options := view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())
		_, err := runInstallProgram(model, options...)
		if err != nil {
			return Result{}, err
		}
		finalViewOutput = model.View()
		return Result{InstalledCount: len(items)}, model.outcome.err
	})

	addPersistentFlagsForTesting(cmd)
	cmd.SetIn(session.Input())
	cmd.SetOut(session.Output())
	cmd.SetErr(session.Output())
	cmd.SetArgs([]string{})

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	session.WaitForOutput(t, func(output []byte) bool {
		normalized := terminal.NormalizeOutput(string(output), terminal.NormalizeOptions{
			StripControlSequences: true,
		})
		return strings.Contains(normalized, "cmd.install.summary.success")
	}, terminalpty.WithWaitDuration(2*time.Second))

	select {
	case err := <-execErr:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for install command to finish")
	}

	output := normalizeInstallViewOutput(finalViewOutput)
	require.Contains(t, output, "cmd.install.header.success")
	require.Contains(t, output, "cmd.install.summary.success")
	snaps.MatchSnapshot(t, output)

	finalScreenOutput := normalizeInstallFinalScreenOutput(session.OutputString())
	require.Contains(t, finalScreenOutput, "cmd.install.header.success")
	require.Contains(t, finalScreenOutput, "cmd.install.summary.success")
	visibleOutput := normalizeInstallVisibleOutput(session.OutputString(), rows)
	require.Contains(t, visibleOutput, "cmd.install.header.success")
	require.Contains(t, visibleOutput, "cmd.install.summary.success")
}

func normalizeInstallViewOutput(output string) string {
	normalized := terminal.NormalizeOutput(output, terminal.NormalizeOptions{
		TrimTrailingWhitespace: true,
		TrimTrailingEmptyLines: true,
		TrimSpace:              true,
	})
	return collapseInstallBlankLines(normalized)
}

func normalizeInstallFinalScreenOutput(output string) string {
	trimmed := trimToLastInstallFrame(output)
	trimmed = cursorHomeSequence.ReplaceAllString(trimmed, "\n")
	normalized := terminal.NormalizeOutput(trimmed, terminal.NormalizeOptions{
		StripControlSequences:  true,
		TrimTrailingWhitespace: true,
		TrimTrailingEmptyLines: true,
		TrimSpace:              true,
	})
	return collapseInstallBlankLines(normalized)
}

func normalizeInstallVisibleOutput(output string, rows uint16) string {
	trimmed := trimToLastInstallFrame(output)
	trimmed = cursorHomeSequence.ReplaceAllString(trimmed, "\n")
	normalized := terminal.NormalizeOutput(trimmed, terminal.NormalizeOptions{
		StripControlSequences:  true,
		TrimTrailingWhitespace: true,
		TrimTrailingEmptyLines: true,
		TrimSpace:              true,
		RowLimit:               int(rows),
	})
	return collapseInstallBlankLines(normalized)
}

func normalizeInstallRunningOutput(output string, items []installItem) string {
	if len(items) == 0 {
		return terminal.NormalizeOutput(output, terminal.NormalizeOptions{
			StripControlSequences:  true,
			TrimTrailingWhitespace: true,
			TrimTrailingEmptyLines: true,
			TrimSpace:              true,
		})
	}
	return normalizeInstallViewOutput(renderInstallRunningView(view.ColorDisabled, items, ""))
}

func trimToLastInstallFrame(value string) string {
	indices := cursorHomeSequence.FindAllStringIndex(value, -1)
	if len(indices) == 0 {
		return value
	}
	for index := len(indices) - 1; index >= 0; index-- {
		candidate := value[indices[index][0]:]
		normalized := terminal.NormalizeOutput(candidate, terminal.NormalizeOptions{
			StripControlSequences:  true,
			TrimTrailingWhitespace: true,
			TrimTrailingEmptyLines: true,
		})
		if strings.TrimSpace(normalized) != "" {
			return candidate
		}
	}
	return value
}

var cursorHomeSequence = regexp.MustCompile(`\x1b\[[0-9;]*H`)

func collapseInstallBlankLines(output string) string {
	lines := strings.Split(output, "\n")
	collapsed := make([]string, 0, len(lines))
	emptyCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			emptyCount++
			if emptyCount > 1 {
				continue
			}
		} else {
			emptyCount = 0
		}
		collapsed = append(collapsed, line)
	}
	return strings.Join(collapsed, "\n")
}

func installTallSnapshotRows() uint16 {
	if runtime.GOOS == "windows" {
		return 40
	}
	return 80
}

func sampleInstallModsLongList() models.ModsJSON {
	return models.ModsJSON{
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
			{ID: "beta", Name: "Beta", Type: models.MODRINTH},
			{ID: "gamma", Name: "Gamma", Type: models.MODRINTH},
			{ID: "delta", Name: "Delta", Type: models.MODRINTH},
			{ID: "epsilon", Name: "Epsilon", Type: models.MODRINTH},
			{ID: "zeta", Name: "Zeta", Type: models.MODRINTH},
			{ID: "eta", Name: "Eta", Type: models.MODRINTH},
			{ID: "theta", Name: "Theta", Type: models.MODRINTH},
			{ID: "iota", Name: "Iota", Type: models.MODRINTH},
			{ID: "kappa", Name: "Kappa", Type: models.MODRINTH},
			{ID: "lambda", Name: "Lambda", Type: models.MODRINTH},
			{ID: "mu", Name: "Mu", Type: models.MODRINTH},
			{ID: "nu", Name: "Nu", Type: models.MODRINTH},
			{ID: "xi", Name: "Xi", Type: models.MODRINTH},
			{ID: "omicron", Name: "Omicron", Type: models.MODRINTH},
			{ID: "pi", Name: "Pi", Type: models.MODRINTH},
			{ID: "rho", Name: "Rho", Type: models.MODRINTH},
			{ID: "sigma", Name: "Sigma", Type: models.MODRINTH},
			{ID: "tau", Name: "Tau", Type: models.MODRINTH},
			{ID: "upsilon", Name: "Upsilon", Type: models.MODRINTH},
			{ID: "phi", Name: "Phi", Type: models.MODRINTH},
			{ID: "chi", Name: "Chi", Type: models.MODRINTH},
			{ID: "psi", Name: "Psi", Type: models.MODRINTH},
			{ID: "omega", Name: "Omega", Type: models.MODRINTH},
			{ID: "beta-2", Name: "Beta Two", Type: models.MODRINTH},
			{ID: "gamma-2", Name: "Gamma Two", Type: models.MODRINTH},
			{ID: "delta-2", Name: "Delta Two", Type: models.MODRINTH},
			{ID: "epsilon-2", Name: "Epsilon Two", Type: models.MODRINTH},
			{ID: "zeta-2", Name: "Zeta Two", Type: models.MODRINTH},
			{ID: "eta-2", Name: "Eta Two", Type: models.MODRINTH},
		},
	}
}
