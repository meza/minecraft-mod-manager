package test

import (
	"context"
	"errors"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/meza/minecraft-mod-manager/testutil/terminal"
	terminalpty "github.com/meza/minecraft-mod-manager/testutil/terminal/pty"
)

func TestTestCommandInteractivePTYOutput(t *testing.T) {
	terminal.ApplyFixtures(t)

	originalRunTestProgram := runTestProgram
	runTestProgram = defaultRunTestProgram
	t.Cleanup(func() { runTestProgram = originalRunTestProgram })

	rows := uint16(40)
	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: 120, Rows: int(rows)}))
	require.NotNil(t, session)

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, _ testOptions, _ testDeps) (Result, error) {
		items := []testItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "beta", Name: "Beta", Type: models.CURSEFORGE}, Status: testItemStatusChecking},
		}
		index := map[string]int{
			testModKey(items[0].Mod): 0,
			testModKey(items[1].Mod): 1,
		}

		execRunner := func(ctx context.Context, sender testExecSender) testExecutionOutcome {
			time.Sleep(25 * time.Millisecond)
			sender.Send(testItemUpdateMsg{key: testModKey(items[0].Mod), status: testItemStatusSupported})
			sender.Send(testItemUpdateMsg{key: testModKey(items[1].Mod), status: testItemStatusInconclusive, reason: "reason"})
			updated := cloneTestItems(items)
			updated[0].Status = testItemStatusSupported
			updated[1].Status = testItemStatusInconclusive
			updated[1].Reason = "reason"
			return testExecutionOutcome{items: updated}
		}

		model := newTestModel(testModelInput{
			ctx:               ctx,
			targetVersion:     "1.21.1",
			colorMode:         colorModeForOutput(cmd.OutOrStdout()),
			items:             items,
			indexByKey:        index,
			showCompatibility: true,
			execRunner:        execRunner,
		})

		result, runErr := runTestProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
		if runErr != nil {
			return Result{ExitCode: 1, Interactive: true}, runErr
		}
		outcome, outcomeErr := finalizePTYRun(cmd, result)
		if outcomeErr != nil {
			return Result{ExitCode: 1, Interactive: true}, outcomeErr
		}
		if outcome.err != nil {
			return Result{ExitCode: 1, Interactive: true}, outcome.err
		}

		return Result{ExitCode: 1, Interactive: true}, clierrors.MarkHandled(errUnsupportedMods)
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
	require.ErrorIs(t, execErr, errUnsupportedMods)
	session.WaitForOutput(t, func(output []byte) bool {
		normalized := terminal.NormalizeOutput(string(output), terminal.NormalizeOptions{
			StripControlSequences: true,
		})
		return strings.Contains(normalized, "cmd.test.summary.inconclusive")
	}, terminalpty.WithWaitDuration(2*time.Second))
	require.NoError(t, session.Close())

	snaps.MatchSnapshot(t, normalizeTestPTYOutput(session.OutputString(), 0))
}

func TestTestCommandInteractivePTYRunningShowsHeadersWhenShort(t *testing.T) {
	terminal.ApplyFixtures(t)

	originalRunTestProgram := runTestProgram
	runTestProgram = defaultRunTestProgram
	t.Cleanup(func() { runTestProgram = originalRunTestProgram })

	rows := uint16(25)
	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: 120, Rows: int(rows)}))
	require.NotNil(t, session)

	release := make(chan struct{})
	updatesSent := make(chan struct{})

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, _ testOptions, _ testDeps) (Result, error) {
		items := []testItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "beta", Name: "Beta", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "gamma", Name: "Gamma", Type: models.MODRINTH}, Status: testItemStatusChecking},
		}
		index := map[string]int{
			testModKey(items[0].Mod): 0,
			testModKey(items[1].Mod): 1,
			testModKey(items[2].Mod): 2,
		}

		execRunner := func(ctx context.Context, sender testExecSender) testExecutionOutcome {
			sender.Send(testItemUpdateMsg{key: testModKey(items[0].Mod), status: testItemStatusSupported})
			sender.Send(testItemUpdateMsg{key: testModKey(items[1].Mod), status: testItemStatusUnsupported})
			sender.Send(testItemUpdateMsg{key: testModKey(items[2].Mod), status: testItemStatusChecking})
			close(updatesSent)
			<-release

			updated := cloneTestItems(items)
			updated[0].Status = testItemStatusSupported
			updated[1].Status = testItemStatusUnsupported
			updated[2].Status = testItemStatusSupported
			return testExecutionOutcome{items: updated}
		}

		model := newTestModel(testModelInput{
			ctx:               ctx,
			targetVersion:     "1.21.11",
			colorMode:         colorModeForOutput(cmd.OutOrStdout()),
			items:             items,
			indexByKey:        index,
			showCompatibility: true,
			execRunner:        execRunner,
		})

		result, runErr := runTestProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
		if runErr != nil {
			return Result{ExitCode: 1, Interactive: true}, runErr
		}
		outcome, outcomeErr := finalizePTYRun(cmd, result)
		if outcomeErr != nil {
			return Result{ExitCode: 1, Interactive: true}, outcomeErr
		}
		if outcome.err != nil {
			return Result{ExitCode: 1, Interactive: true}, outcome.err
		}

		return Result{ExitCode: 1, Interactive: true}, clierrors.MarkHandled(errUnsupportedMods)
	})

	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("unattended", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")

	cmd.SetIn(session.Input())
	cmd.SetOut(session.Output())
	cmd.SetErr(session.Output())
	cmd.SetArgs([]string{"1.21.11"})

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	select {
	case <-updatesSent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for running updates")
	}

	session.WaitForOutput(t, func(output []byte) bool {
		normalized := normalizeTestPTYOutput(string(output), rows)
		return normalizedHasLine(normalized, "cmd.test.section.compatible") &&
			normalizedHasLine(normalized, "cmd.test.section.not_compatible") &&
			strings.Contains(normalized, "❌ Beta (beta) [modrinth]")
	}, terminalpty.WithWaitDuration(2*time.Second))
	snaps.MatchSnapshot(t, normalizeTestPTYOutput(session.OutputString(), rows))

	close(release)
	execErrValue := <-execErr
	require.ErrorIs(t, execErrValue, errUnsupportedMods)
	require.NoError(t, session.Close())
}

func TestTestCommandInteractivePTYRunningSnapshotMediumHeight(t *testing.T) {
	terminal.ApplyFixtures(t)

	originalRunTestProgram := runTestProgram
	runTestProgram = defaultRunTestProgram
	t.Cleanup(func() { runTestProgram = originalRunTestProgram })

	rows := uint16(12)
	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: 120, Rows: int(rows)}))
	require.NotNil(t, session)

	release := make(chan struct{})
	updatesSent := make(chan struct{})

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, _ testOptions, _ testDeps) (Result, error) {
		items := []testItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "beta", Name: "Beta", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "gamma", Name: "Gamma", Type: models.MODRINTH}, Status: testItemStatusChecking},
		}
		index := map[string]int{
			testModKey(items[0].Mod): 0,
			testModKey(items[1].Mod): 1,
			testModKey(items[2].Mod): 2,
		}

		execRunner := func(ctx context.Context, sender testExecSender) testExecutionOutcome {
			sender.Send(testItemUpdateMsg{key: testModKey(items[0].Mod), status: testItemStatusSupported})
			close(updatesSent)
			<-release

			updated := cloneTestItems(items)
			for itemIndex := range updated {
				updated[itemIndex].Status = testItemStatusSupported
			}
			return testExecutionOutcome{items: updated}
		}

		model := newTestModel(testModelInput{
			ctx:               ctx,
			targetVersion:     "1.21.11",
			colorMode:         colorModeForOutput(cmd.OutOrStdout()),
			items:             items,
			indexByKey:        index,
			showCompatibility: true,
			execRunner:        execRunner,
		})

		result, runErr := runTestProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
		if runErr != nil {
			return Result{ExitCode: 1, Interactive: true}, runErr
		}
		outcome, outcomeErr := finalizePTYRun(cmd, result)
		if outcomeErr != nil {
			return Result{ExitCode: 1, Interactive: true}, outcomeErr
		}
		if outcome.err != nil {
			return Result{ExitCode: 1, Interactive: true}, outcome.err
		}

		return Result{ExitCode: 0, Interactive: true}, nil
	})

	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("unattended", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")

	cmd.SetIn(session.Input())
	cmd.SetOut(session.Output())
	cmd.SetErr(session.Output())
	cmd.SetArgs([]string{"1.21.11"})

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	select {
	case <-updatesSent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for running updates")
	}

	session.WaitForOutput(t, func(output []byte) bool {
		return outputHasLine(output, "cmd.compatibility.section")
	}, terminalpty.WithWaitDuration(2*time.Second))
	session.WaitForOutput(t, func(output []byte) bool {
		return strings.Contains(string(output), "✅ Alpha (alpha) [modrinth]")
	}, terminalpty.WithWaitDuration(2*time.Second))
	snaps.MatchSnapshot(t, normalizeTestPTYOutputSection(session.OutputString(), "cmd.test.section.not_compatible"))

	close(release)
	execErrValue := <-execErr
	require.NoError(t, execErrValue)
	session.WaitForOutput(t, func(output []byte) bool {
		normalized := terminal.NormalizeOutput(string(output), terminal.NormalizeOptions{
			StripControlSequences: true,
		})
		return strings.Contains(normalized, "cmd.test.success")
	}, terminalpty.WithWaitDuration(2*time.Second))
	require.NoError(t, session.Close())
}

func TestTestCommandInteractivePTYRunningScrollsToNotCompatible(t *testing.T) {
	terminal.ApplyFixtures(t)

	originalRunTestProgram := runTestProgram
	runTestProgram = defaultRunTestProgram
	t.Cleanup(func() { runTestProgram = originalRunTestProgram })

	rows := uint16(6)
	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: 60, Rows: int(rows)}))
	require.NotNil(t, session)

	release := make(chan struct{})
	updatesSent := make(chan struct{})

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, _ testOptions, _ testDeps) (Result, error) {
		items := []testItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "beta", Name: "Beta", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "gamma", Name: "Gamma", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "delta", Name: "Delta", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "epsilon", Name: "Epsilon", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "zeta", Name: "Zeta", Type: models.MODRINTH}, Status: testItemStatusChecking},
		}
		index := make(map[string]int, len(items))
		for itemIndex, item := range items {
			index[testModKey(item.Mod)] = itemIndex
		}

		execRunner := func(ctx context.Context, sender testExecSender) testExecutionOutcome {
			for itemIndex := range items {
				status := testItemStatusSupported
				if itemIndex == len(items)-1 {
					status = testItemStatusUnsupported
				}
				sender.Send(testItemUpdateMsg{key: testModKey(items[itemIndex].Mod), status: status})
			}
			close(updatesSent)
			<-release

			updated := cloneTestItems(items)
			for itemIndex := range updated {
				if itemIndex == len(updated)-1 {
					updated[itemIndex].Status = testItemStatusUnsupported
					continue
				}
				updated[itemIndex].Status = testItemStatusSupported
			}
			return testExecutionOutcome{items: updated}
		}

		model := newTestModel(testModelInput{
			ctx:               ctx,
			targetVersion:     "1.21.11",
			colorMode:         colorModeForOutput(cmd.OutOrStdout()),
			items:             items,
			indexByKey:        index,
			showCompatibility: true,
			execRunner:        execRunner,
		})

		result, runErr := runTestProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
		if runErr != nil {
			return Result{ExitCode: 1, Interactive: true}, runErr
		}
		outcome, outcomeErr := finalizePTYRun(cmd, result)
		if outcomeErr != nil {
			return Result{ExitCode: 1, Interactive: true}, outcomeErr
		}
		if outcome.err != nil {
			return Result{ExitCode: 1, Interactive: true}, outcome.err
		}

		return Result{ExitCode: 1, Interactive: true}, clierrors.MarkHandled(errUnsupportedMods)
	})

	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("unattended", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")

	cmd.SetIn(session.Input())
	cmd.SetOut(session.Output())
	cmd.SetErr(session.Output())
	cmd.SetArgs([]string{"1.21.11"})

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	select {
	case <-updatesSent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for running updates")
	}

	session.WaitForOutput(t, func(output []byte) bool {
		return strings.Contains(string(output), "cmd.test.header")
	}, terminalpty.WithWaitDuration(2*time.Second))
	session.WaitForOutput(t, func(output []byte) bool {
		return outputHasLine(output, "cmd.test.section.compatible")
	}, terminalpty.WithWaitDuration(2*time.Second))

	_, writeErr := session.SendInput([]byte("\x1b[6~"))
	require.NoError(t, writeErr)

	session.WaitForOutput(t, func(output []byte) bool {
		return outputHasLine(output, "cmd.test.section.not_compatible")
	}, terminalpty.WithWaitDuration(2*time.Second))
	session.WaitForOutput(t, func(output []byte) bool {
		return strings.Contains(string(output), "❌ Zeta (zeta) [modrinth]")
	}, terminalpty.WithWaitDuration(2*time.Second))
	normalized := terminal.NormalizeOutput(session.OutputString(), terminal.NormalizeOptions{
		StripControlSequences:  true,
		TrimTrailingWhitespace: true,
		TrimTrailingEmptyLines: true,
	})
	snaps.MatchSnapshot(t, trimToSectionValue(normalized, "cmd.test.section.not_compatible"))

	close(release)
	execErrValue := <-execErr
	require.ErrorIs(t, execErrValue, errUnsupportedMods)
	require.NoError(t, session.Close())
}

func TestTestCommandInteractivePTYRunningSnapshotTallHeightFailure(t *testing.T) {
	terminal.ApplyFixtures(t)

	originalRunTestProgram := runTestProgram
	runTestProgram = defaultRunTestProgram
	t.Cleanup(func() { runTestProgram = originalRunTestProgram })

	rows := tallSnapshotRows()
	columns := 120
	if rows == tallSnapshotRows() {
		columns = 80
	}
	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: columns, Rows: int(rows)}))
	require.NotNil(t, session)

	release := make(chan struct{})
	updatesSent := make(chan struct{})

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, _ testOptions, _ testDeps) (Result, error) {
		items := []testItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "beta", Name: "Beta", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "gamma", Name: "Gamma", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "delta", Name: "Delta", Type: models.MODRINTH}, Status: testItemStatusChecking},
		}
		index := make(map[string]int, len(items))
		for itemIndex, item := range items {
			index[testModKey(item.Mod)] = itemIndex
		}

		execRunner := func(ctx context.Context, sender testExecSender) testExecutionOutcome {
			sender.Send(testItemUpdateMsg{key: testModKey(items[0].Mod), status: testItemStatusSupported})
			sender.Send(testItemUpdateMsg{key: testModKey(items[1].Mod), status: testItemStatusUnsupported})
			sender.Send(testItemUpdateMsg{key: testModKey(items[2].Mod), status: testItemStatusChecking})
			sender.Send(testItemUpdateMsg{key: testModKey(items[3].Mod), status: testItemStatusChecking})
			close(updatesSent)
			<-release

			updated := cloneTestItems(items)
			updated[0].Status = testItemStatusSupported
			updated[1].Status = testItemStatusUnsupported
			updated[2].Status = testItemStatusSupported
			updated[3].Status = testItemStatusSupported
			return testExecutionOutcome{items: updated}
		}

		model := newTestModel(testModelInput{
			ctx:               ctx,
			targetVersion:     "1.21.11",
			colorMode:         colorModeForOutput(cmd.OutOrStdout()),
			items:             items,
			indexByKey:        index,
			showCompatibility: true,
			execRunner:        execRunner,
		})

		result, runErr := runTestProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
		if runErr != nil {
			return Result{ExitCode: 1, Interactive: true}, runErr
		}
		outcome, outcomeErr := finalizePTYRun(cmd, result)
		if outcomeErr != nil {
			return Result{ExitCode: 1, Interactive: true}, outcomeErr
		}
		if outcome.err != nil {
			return Result{ExitCode: 1, Interactive: true}, outcome.err
		}

		return Result{ExitCode: 1, Interactive: true}, clierrors.MarkHandled(errUnsupportedMods)
	})

	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("unattended", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")

	cmd.SetIn(session.Input())
	cmd.SetOut(session.Output())
	cmd.SetErr(session.Output())
	cmd.SetArgs([]string{"1.21.11"})

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	select {
	case <-updatesSent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for running updates")
	}

	session.WaitForOutput(t, func(output []byte) bool {
		return outputHasLine(output, "cmd.test.section.not_compatible")
	}, terminalpty.WithWaitDuration(2*time.Second))
	session.WaitForOutput(t, func(output []byte) bool {
		return strings.Contains(string(output), "❌ Beta (beta) [modrinth]")
	}, terminalpty.WithWaitDuration(2*time.Second))
	snaps.MatchSnapshot(t, normalizeTestPTYOutput(session.OutputString(), rows))

	close(release)
	execErrValue := <-execErr
	require.ErrorIs(t, execErrValue, errUnsupportedMods)
	require.NoError(t, session.Close())
}

func TestTestCommandInteractivePTYRunningMouseScrollsToNotCompatible(t *testing.T) {
	terminal.ApplyFixtures(t)

	originalRunTestProgram := runTestProgram
	runTestProgram = defaultRunTestProgram
	t.Cleanup(func() { runTestProgram = originalRunTestProgram })

	rows := uint16(25)
	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: 60, Rows: int(rows)}))
	require.NotNil(t, session)

	release := make(chan struct{})
	updatesSent := make(chan struct{})

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, _ testOptions, _ testDeps) (Result, error) {
		items := []testItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "beta", Name: "Beta", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "gamma", Name: "Gamma", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "delta", Name: "Delta", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "epsilon", Name: "Epsilon", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "zeta", Name: "Zeta", Type: models.MODRINTH}, Status: testItemStatusChecking},
		}
		index := make(map[string]int, len(items))
		for itemIndex, item := range items {
			index[testModKey(item.Mod)] = itemIndex
		}

		execRunner := func(ctx context.Context, sender testExecSender) testExecutionOutcome {
			for itemIndex := range items {
				status := testItemStatusSupported
				if itemIndex == len(items)-1 {
					status = testItemStatusUnsupported
				}
				sender.Send(testItemUpdateMsg{key: testModKey(items[itemIndex].Mod), status: status})
			}
			close(updatesSent)
			<-release

			updated := cloneTestItems(items)
			for itemIndex := range updated {
				if itemIndex == len(updated)-1 {
					updated[itemIndex].Status = testItemStatusUnsupported
					continue
				}
				updated[itemIndex].Status = testItemStatusSupported
			}
			return testExecutionOutcome{items: updated}
		}

		model := newTestModel(testModelInput{
			ctx:               ctx,
			targetVersion:     "1.21.11",
			colorMode:         colorModeForOutput(cmd.OutOrStdout()),
			items:             items,
			indexByKey:        index,
			showCompatibility: true,
			execRunner:        execRunner,
		})

		result, runErr := runTestProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
		if runErr != nil {
			return Result{ExitCode: 1, Interactive: true}, runErr
		}
		outcome, outcomeErr := finalizePTYRun(cmd, result)
		if outcomeErr != nil {
			return Result{ExitCode: 1, Interactive: true}, outcomeErr
		}
		if outcome.err != nil {
			return Result{ExitCode: 1, Interactive: true}, outcome.err
		}

		return Result{ExitCode: 1, Interactive: true}, clierrors.MarkHandled(errUnsupportedMods)
	})

	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("unattended", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")

	cmd.SetIn(session.Input())
	cmd.SetOut(session.Output())
	cmd.SetErr(session.Output())
	cmd.SetArgs([]string{"1.21.11"})

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	select {
	case <-updatesSent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for running updates")
	}

	session.WaitForOutput(t, func(output []byte) bool {
		return strings.Contains(string(output), "cmd.test.header")
	}, terminalpty.WithWaitDuration(2*time.Second))
	session.WaitForOutput(t, func(output []byte) bool {
		return outputHasLine(output, "cmd.test.section.compatible")
	}, terminalpty.WithWaitDuration(2*time.Second))
	session.WaitForOutput(t, func(output []byte) bool {
		return strings.Contains(string(output), "✅ Delta (delta) [modrinth]")
	}, terminalpty.WithWaitDuration(2*time.Second))

	for scrollStep := 0; scrollStep < 3; scrollStep++ {
		_, writeErr := session.SendInput([]byte("\x1b[<65;1;1M"))
		require.NoError(t, writeErr)
	}

	session.WaitForOutput(t, func(output []byte) bool {
		return outputHasLine(output, "cmd.test.section.not_compatible")
	}, terminalpty.WithWaitDuration(2*time.Second))
	session.WaitForOutput(t, func(output []byte) bool {
		return strings.Contains(string(output), "❌ Zeta (zeta) [modrinth]")
	}, terminalpty.WithWaitDuration(2*time.Second))
	snaps.MatchSnapshot(t, normalizeTestPTYOutputSection(session.OutputString(), "cmd.test.section.not_compatible"))

	close(release)
	execErrValue := <-execErr
	require.ErrorIs(t, execErrValue, errUnsupportedMods)
	require.NoError(t, session.Close())
}

func TestTestCommandInteractivePTYFinalTranscriptSuccessShortHeight(t *testing.T) {
	terminal.ApplyFixtures(t)

	originalRunTestProgram := runTestProgram
	runTestProgram = defaultRunTestProgram
	t.Cleanup(func() { runTestProgram = originalRunTestProgram })

	rows := uint16(6)
	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: 120, Rows: int(rows)}))
	require.NotNil(t, session)

	release := make(chan struct{})
	updatesSent := make(chan struct{})

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, _ testOptions, _ testDeps) (Result, error) {
		items := []testItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "beta", Name: "Beta", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "gamma", Name: "Gamma", Type: models.MODRINTH}, Status: testItemStatusChecking},
		}
		index := make(map[string]int, len(items))
		for itemIndex, item := range items {
			index[testModKey(item.Mod)] = itemIndex
		}

		execRunner := func(ctx context.Context, sender testExecSender) testExecutionOutcome {
			for itemIndex := range items {
				sender.Send(testItemUpdateMsg{key: testModKey(items[itemIndex].Mod), status: testItemStatusSupported})
			}
			close(updatesSent)
			<-release

			updated := cloneTestItems(items)
			for itemIndex := range updated {
				updated[itemIndex].Status = testItemStatusSupported
			}
			return testExecutionOutcome{items: updated}
		}

		model := newTestModel(testModelInput{
			ctx:               ctx,
			targetVersion:     "1.21.11",
			colorMode:         colorModeForOutput(cmd.OutOrStdout()),
			items:             items,
			indexByKey:        index,
			showCompatibility: true,
			execRunner:        execRunner,
		})

		result, runErr := runTestProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
		if runErr != nil {
			return Result{ExitCode: 1, Interactive: true}, runErr
		}
		outcome, outcomeErr := finalizePTYRun(cmd, result)
		if outcomeErr != nil {
			return Result{ExitCode: 1, Interactive: true}, outcomeErr
		}
		if outcome.err != nil {
			return Result{ExitCode: 1, Interactive: true}, outcome.err
		}

		return Result{ExitCode: 0, Interactive: true}, nil
	})

	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("unattended", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")

	cmd.SetIn(session.Input())
	cmd.SetOut(session.Output())
	cmd.SetErr(session.Output())
	cmd.SetArgs([]string{"1.21.11"})

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	select {
	case <-updatesSent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for running updates")
	}

	close(release)
	execErrValue := <-execErr
	require.NoError(t, execErrValue)
	session.WaitForOutput(t, func(output []byte) bool {
		normalized := terminal.NormalizeOutput(string(output), terminal.NormalizeOptions{
			StripControlSequences: true,
		})
		return strings.Contains(normalized, "cmd.test.success")
	}, terminalpty.WithWaitDuration(2*time.Second))
	require.NoError(t, session.Close())

	snaps.MatchSnapshot(t, normalizeTestPTYOutput(session.OutputString(), rows))
}

func TestTestCommandInteractivePTYFinalTranscriptIncludesSummaryAfterScroll(t *testing.T) {
	terminal.ApplyFixtures(t)

	originalRunTestProgram := runTestProgram
	runTestProgram = defaultRunTestProgram
	t.Cleanup(func() { runTestProgram = originalRunTestProgram })

	rows := uint16(6)
	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: 60, Rows: int(rows)}))
	require.NotNil(t, session)

	release := make(chan struct{})
	updatesSent := make(chan struct{})

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, _ testOptions, _ testDeps) (Result, error) {
		items := []testItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "beta", Name: "Beta", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "gamma", Name: "Gamma", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "delta", Name: "Delta", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "epsilon", Name: "Epsilon", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "zeta", Name: "Zeta", Type: models.MODRINTH}, Status: testItemStatusChecking},
		}
		index := make(map[string]int, len(items))
		for itemIndex, item := range items {
			index[testModKey(item.Mod)] = itemIndex
		}

		execRunner := func(ctx context.Context, sender testExecSender) testExecutionOutcome {
			for itemIndex := range items {
				status := testItemStatusSupported
				if itemIndex == len(items)-1 {
					status = testItemStatusUnsupported
				}
				sender.Send(testItemUpdateMsg{key: testModKey(items[itemIndex].Mod), status: status})
			}
			close(updatesSent)
			<-release

			updated := cloneTestItems(items)
			for itemIndex := range updated {
				if itemIndex == len(updated)-1 {
					updated[itemIndex].Status = testItemStatusUnsupported
					continue
				}
				updated[itemIndex].Status = testItemStatusSupported
			}
			return testExecutionOutcome{items: updated}
		}

		model := newTestModel(testModelInput{
			ctx:               ctx,
			targetVersion:     "1.21.11",
			colorMode:         colorModeForOutput(cmd.OutOrStdout()),
			items:             items,
			indexByKey:        index,
			showCompatibility: true,
			execRunner:        execRunner,
		})

		result, runErr := runTestProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
		if runErr != nil {
			return Result{ExitCode: 1, Interactive: true}, runErr
		}
		outcome, outcomeErr := finalizePTYRun(cmd, result)
		if outcomeErr != nil {
			return Result{ExitCode: 1, Interactive: true}, outcomeErr
		}
		if outcome.err != nil {
			return Result{ExitCode: 1, Interactive: true}, outcome.err
		}

		return Result{ExitCode: 1, Interactive: true}, clierrors.MarkHandled(errUnsupportedMods)
	})

	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("unattended", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")

	cmd.SetIn(session.Input())
	cmd.SetOut(session.Output())
	cmd.SetErr(session.Output())
	cmd.SetArgs([]string{"1.21.11"})

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	select {
	case <-updatesSent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for running updates")
	}

	session.WaitForOutput(t, func(output []byte) bool {
		return outputHasLine(output, "cmd.test.section.compatible")
	}, terminalpty.WithWaitDuration(2*time.Second))

	for scrollStep := 0; scrollStep < 3; scrollStep++ {
		_, writeErr := session.SendInput([]byte("\x1b[<65;1;1M"))
		require.NoError(t, writeErr)
	}

	close(release)
	execErrValue := <-execErr
	require.ErrorIs(t, execErrValue, errUnsupportedMods)

	session.WaitForOutput(t, func(output []byte) bool {
		normalized := terminal.NormalizeOutput(string(output), terminal.NormalizeOptions{
			StripControlSequences: true,
		})
		return strings.Contains(normalized, "cmd.test.summary.unsupported")
	}, terminalpty.WithWaitDuration(2*time.Second))
	require.NoError(t, session.Close())

	snaps.MatchSnapshot(t, normalizeTestPTYOutput(session.OutputString(), rows))
}

func TestTestCommandInteractivePTYFinalTranscriptInconclusiveMediumHeight(t *testing.T) {
	terminal.ApplyFixtures(t)

	originalRunTestProgram := runTestProgram
	runTestProgram = defaultRunTestProgram
	t.Cleanup(func() { runTestProgram = originalRunTestProgram })

	rows := uint16(12)
	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: 120, Rows: int(rows)}))
	require.NotNil(t, session)

	release := make(chan struct{})
	updatesSent := make(chan struct{})

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, _ testOptions, _ testDeps) (Result, error) {
		items := []testItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "beta", Name: "Beta", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "gamma", Name: "Gamma", Type: models.MODRINTH}, Status: testItemStatusChecking},
		}
		index := make(map[string]int, len(items))
		for itemIndex, item := range items {
			index[testModKey(item.Mod)] = itemIndex
		}

		execRunner := func(ctx context.Context, sender testExecSender) testExecutionOutcome {
			sender.Send(testItemUpdateMsg{key: testModKey(items[0].Mod), status: testItemStatusSupported})
			sender.Send(testItemUpdateMsg{key: testModKey(items[1].Mod), status: testItemStatusInconclusive, reason: "timed out"})
			sender.Send(testItemUpdateMsg{key: testModKey(items[2].Mod), status: testItemStatusSupported})
			close(updatesSent)
			<-release

			updated := cloneTestItems(items)
			updated[0].Status = testItemStatusSupported
			updated[1].Status = testItemStatusInconclusive
			updated[1].Reason = "timed out"
			updated[2].Status = testItemStatusSupported
			return testExecutionOutcome{items: updated}
		}

		model := newTestModel(testModelInput{
			ctx:               ctx,
			targetVersion:     "1.21.11",
			colorMode:         colorModeForOutput(cmd.OutOrStdout()),
			items:             items,
			indexByKey:        index,
			showCompatibility: true,
			execRunner:        execRunner,
		})

		result, runErr := runTestProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
		if runErr != nil {
			return Result{ExitCode: 1, Interactive: true}, runErr
		}
		outcome, outcomeErr := finalizePTYRun(cmd, result)
		if outcomeErr != nil {
			return Result{ExitCode: 1, Interactive: true}, outcomeErr
		}
		if outcome.err != nil {
			return Result{ExitCode: 1, Interactive: true}, outcome.err
		}

		return Result{ExitCode: 1, Interactive: true}, clierrors.MarkHandled(errUnsupportedMods)
	})

	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("unattended", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")

	cmd.SetIn(session.Input())
	cmd.SetOut(session.Output())
	cmd.SetErr(session.Output())
	cmd.SetArgs([]string{"1.21.11"})

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	select {
	case <-updatesSent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for running updates")
	}

	close(release)
	execErrValue := <-execErr
	require.ErrorIs(t, execErrValue, errUnsupportedMods)

	session.WaitForOutput(t, func(output []byte) bool {
		normalized := terminal.NormalizeOutput(string(output), terminal.NormalizeOptions{
			StripControlSequences: true,
		})
		return strings.Contains(normalized, "cmd.test.summary.inconclusive")
	}, terminalpty.WithWaitDuration(2*time.Second))
	require.NoError(t, session.Close())

	snaps.MatchSnapshot(t, normalizeTestPTYOutput(session.OutputString(), rows))
}

func TestTestCommandInteractivePTYFinalTranscriptIncludesSummaryWithoutScroll(t *testing.T) {
	terminal.ApplyFixtures(t)

	originalRunTestProgram := runTestProgram
	runTestProgram = defaultRunTestProgram
	t.Cleanup(func() { runTestProgram = originalRunTestProgram })

	rows := uint16(6)
	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: 60, Rows: int(rows)}))
	require.NotNil(t, session)

	release := make(chan struct{})
	updatesSent := make(chan struct{})

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command, _ testOptions, _ testDeps) (Result, error) {
		items := []testItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "beta", Name: "Beta", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "gamma", Name: "Gamma", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "delta", Name: "Delta", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "epsilon", Name: "Epsilon", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "zeta", Name: "Zeta", Type: models.MODRINTH}, Status: testItemStatusChecking},
		}
		index := make(map[string]int, len(items))
		for itemIndex, item := range items {
			index[testModKey(item.Mod)] = itemIndex
		}

		execRunner := func(ctx context.Context, sender testExecSender) testExecutionOutcome {
			for itemIndex := range items {
				status := testItemStatusSupported
				if itemIndex == len(items)-1 {
					status = testItemStatusUnsupported
				}
				sender.Send(testItemUpdateMsg{key: testModKey(items[itemIndex].Mod), status: status})
			}
			close(updatesSent)
			<-release

			updated := cloneTestItems(items)
			for itemIndex := range updated {
				if itemIndex == len(updated)-1 {
					updated[itemIndex].Status = testItemStatusUnsupported
					continue
				}
				updated[itemIndex].Status = testItemStatusSupported
			}
			return testExecutionOutcome{items: updated}
		}

		model := newTestModel(testModelInput{
			ctx:               ctx,
			targetVersion:     "1.21.11",
			colorMode:         colorModeForOutput(cmd.OutOrStdout()),
			items:             items,
			indexByKey:        index,
			showCompatibility: true,
			execRunner:        execRunner,
		})

		result, runErr := runTestProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
		if runErr != nil {
			return Result{ExitCode: 1, Interactive: true}, runErr
		}
		outcome, outcomeErr := finalizePTYRun(cmd, result)
		if outcomeErr != nil {
			return Result{ExitCode: 1, Interactive: true}, outcomeErr
		}
		if outcome.err != nil {
			return Result{ExitCode: 1, Interactive: true}, outcome.err
		}

		return Result{ExitCode: 1, Interactive: true}, clierrors.MarkHandled(errUnsupportedMods)
	})

	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("unattended", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")

	cmd.SetIn(session.Input())
	cmd.SetOut(session.Output())
	cmd.SetErr(session.Output())
	cmd.SetArgs([]string{"1.21.11"})

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	select {
	case <-updatesSent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for running updates")
	}

	close(release)
	execErrValue := <-execErr
	require.ErrorIs(t, execErrValue, errUnsupportedMods)
	session.WaitForOutput(t, func(output []byte) bool {
		normalized := terminal.NormalizeOutput(string(output), terminal.NormalizeOptions{
			StripControlSequences: true,
		})
		return strings.Contains(normalized, "cmd.test.summary.unsupported")
	}, terminalpty.WithWaitDuration(2*time.Second))
	require.NoError(t, session.Close())

	snaps.MatchSnapshot(t, normalizeTestPTYOutput(session.OutputString(), rows))
}

func normalizeTestPTYOutput(output string, rows uint16) string {
	normalizedAll := terminal.NormalizeOutput(output, terminal.NormalizeOptions{
		StripControlSequences:  true,
		TrimTrailingWhitespace: true,
		TrimTrailingEmptyLines: true,
	})
	if transcript, ok := extractTestTranscript(normalizedAll); ok {
		return normalizeTestLines(transcript, 0)
	}

	preferredMarkers := preferredTestFrameMarkers(normalizedAll)
	if len(preferredMarkers) == 0 {
		preferredMarkers = []string{
			"cmd.test.section.not_compatible",
			"cmd.test.section.compatible",
		}
	}
	if !cursorFrameSequence.MatchString(output) {
		trimmed := trimToLastHeaderBlock(normalizedAll)
		trimmed = adjustNormalizedForSectionMarker(trimmed, normalizedAll, preferredMarkers)
		trimmed = dropCompatibilitySection(trimmed)
		return normalizeTestLines(trimmed, rows)
	}

	trimmed := trimTestOutputToLastFrame(output, preferredMarkers)
	trimmed = cursorFrameSequence.ReplaceAllString(trimmed, "\n")
	normalized := terminal.NormalizeOutput(trimmed, terminal.NormalizeOptions{
		StripControlSequences:  true,
		TrimTrailingWhitespace: true,
		TrimTrailingEmptyLines: true,
	})
	normalized = adjustNormalizedForSectionMarker(normalized, normalizedAll, preferredMarkers)
	normalized = dropCompatibilitySection(normalized)
	normalized = trimToFirstHeaderBlock(normalized)
	return normalizeTestLines(normalized, rows)
}

func normalizeTestLines(normalized string, rows uint16) string {
	lines := strings.Split(normalized, "\n")
	lines = trimLeadingEmptyLines(lines)
	if rows > 0 {
		limit := int(rows)
		if len(lines) > limit {
			lines = lines[:limit]
		} else if len(lines) < limit {
			padding := make([]string, limit-len(lines))
			lines = append(lines, padding...)
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func normalizeTestPTYOutputSection(output string, section string) string {
	trimmed := trimTestOutputToLastFrame(output, []string{section})
	framed := cursorFrameSequence.ReplaceAllString(trimmed, "\n")
	normalized := terminal.NormalizeOutput(framed, terminal.NormalizeOptions{
		StripControlSequences:  true,
		TrimTrailingWhitespace: true,
		TrimTrailingEmptyLines: true,
	})
	normalized = dropCompatibilitySection(normalized)
	if block := trimToLastSectionBlock(normalized, section); block != "" {
		return normalizeTestLines(trimToSectionValue(block, section), 0)
	}
	return normalizeTestLines(trimToSectionValue(normalized, section), 0)
}

func preferredTestFrameMarkers(value string) []string {
	switch {
	case strings.Contains(value, "cmd.test.section.not_compatible"):
		return []string{"cmd.test.section.not_compatible"}
	case strings.Contains(value, "❌ "):
		return []string{"❌ "}
	case strings.Contains(value, "✅ "):
		return []string{"✅ "}
	case strings.Contains(value, "❔ "):
		return []string{"❔ "}
	case strings.Contains(value, "cmd.test.section.compatible"):
		return []string{"cmd.test.section.compatible"}
	default:
		return nil
	}
}

func outputHasLine(output []byte, line string) bool {
	return normalizedHasLine(terminal.NormalizeOutput(string(output), terminal.NormalizeOptions{
		StripControlSequences:  true,
		TrimTrailingWhitespace: true,
	}), line)
}

func dropCompatibilitySection(value string) string {
	compatibilityKey := "cmd.compatibility.section"
	compatibleKey := "cmd.test.section.compatible"
	lines := strings.Split(value, "\n")
	compatibilityIndex := -1
	compatibleIndex := -1
	for index, line := range lines {
		entry := strings.TrimSpace(line)
		if entry == compatibilityKey && compatibilityIndex == -1 {
			compatibilityIndex = index
		}
		if entry == compatibleKey && compatibleIndex == -1 {
			compatibleIndex = index
		}
	}
	if compatibilityIndex < 0 || compatibleIndex < 0 || compatibleIndex <= compatibilityIndex {
		return value
	}
	updated := append([]string{}, lines[:compatibilityIndex]...)
	updated = append(updated, lines[compatibleIndex:]...)
	return strings.TrimSpace(strings.Join(updated, "\n"))
}

func tallSnapshotRows() uint16 {
	if runtime.GOOS == "windows" {
		return 40
	}
	return 80
}

func trimTestOutputToLastFrame(value string, preferredMarkers []string) string {
	indices := cursorFrameSequence.FindAllStringIndex(value, -1)
	if len(indices) == 0 {
		return value
	}
	var fallback string
	for index := len(indices) - 1; index >= 0; index-- {
		start := indices[index][0]
		end := len(value)
		if index+1 < len(indices) {
			end = indices[index+1][0]
		}
		candidate := cursorFrameSequence.ReplaceAllString(value[start:end], "\n")
		normalized := terminal.NormalizeOutput(candidate, terminal.NormalizeOptions{
			StripControlSequences:  true,
			TrimTrailingWhitespace: true,
			TrimTrailingEmptyLines: true,
		})
		if len(preferredMarkers) > 0 {
			for _, marker := range preferredMarkers {
				if strings.Contains(normalized, marker) {
					return value[start:end]
				}
			}
		}
		if containsTestSection(normalized) {
			return value[start:end]
		}
		if strings.Contains(normalized, "cmd.test.header") {
			fallback = value[start:end]
		}
	}
	if fallback != "" {
		return fallback
	}
	return value
}

func normalizedHasLine(value string, line string) bool {
	for _, entry := range strings.Split(value, "\n") {
		if strings.TrimSpace(entry) == line {
			return true
		}
	}
	return false
}

func containsTestSection(value string) bool {
	if normalizedHasLine(value, "cmd.test.section.not_compatible") {
		return true
	}
	if normalizedHasLine(value, "cmd.test.section.compatible") {
		return true
	}
	return normalizedHasLine(value, "cmd.compatibility.section")
}

var cursorFrameSequence = regexp.MustCompile(`\x1b\[[0-9;]*H|\x1b\[[0-9;]*A`)

func extractTestTranscript(normalized string) (string, bool) {
	summaryMarkers := []string{
		"cmd.test.summary.unsupported",
		"cmd.test.summary.inconclusive",
		"cmd.test.success",
	}
	lines := strings.Split(normalized, "\n")
	summaryIndex := lastLineIndexContaining(lines, summaryMarkers)
	if summaryIndex == -1 {
		return "", false
	}
	headerIndex := lastLineIndexWithPrefix(lines[:summaryIndex], "cmd.test.header")
	if headerIndex == -1 {
		headerIndex = lastLineIndexWithPrefix(lines, "cmd.test.header")
		if headerIndex == -1 {
			return "", false
		}
	}
	return strings.Join(lines[headerIndex:], "\n"), true
}

func lastLineIndexContaining(lines []string, markers []string) int {
	for index := len(lines) - 1; index >= 0; index-- {
		for _, marker := range markers {
			if strings.Contains(lines[index], marker) {
				return index
			}
		}
	}
	return -1
}

func lastLineIndexWithPrefix(lines []string, prefix string) int {
	for index := len(lines) - 1; index >= 0; index-- {
		if strings.Contains(strings.TrimSpace(lines[index]), prefix) {
			return index
		}
	}
	return -1
}

func trimLeadingEmptyLines(lines []string) []string {
	for len(lines) > 0 {
		if strings.TrimSpace(lines[0]) != "" {
			return lines
		}
		lines = lines[1:]
	}
	return lines
}

func trimToFirstHeaderBlock(value string) string {
	lines := strings.Split(value, "\n")
	headerPrefix := "cmd.test.header"
	firstHeader := -1
	for index, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), headerPrefix) {
			firstHeader = index
			break
		}
	}
	if firstHeader == -1 {
		return value
	}
	for index := firstHeader + 1; index < len(lines); index++ {
		if strings.HasPrefix(strings.TrimSpace(lines[index]), headerPrefix) {
			return strings.Join(lines[:index], "\n")
		}
	}
	return value
}

func trimToLastHeaderBlock(value string) string {
	lines := strings.Split(value, "\n")
	headerPrefix := "cmd.test.header"
	lastHeader := -1
	for index := len(lines) - 1; index >= 0; index-- {
		if strings.HasPrefix(strings.TrimSpace(lines[index]), headerPrefix) {
			lastHeader = index
			break
		}
	}
	if lastHeader == -1 {
		return value
	}
	return strings.Join(lines[lastHeader:], "\n")
}

func trimToSectionValue(value string, section string) string {
	index := strings.Index(value, section)
	if index == -1 {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(value[index:])
}

func trimToLastSectionBlock(value string, section string) string {
	lines := strings.Split(value, "\n")
	sectionIndex := lastLineIndexContaining(lines, []string{section})
	if sectionIndex == -1 {
		return ""
	}
	headerIndex := -1
	for index := sectionIndex; index >= 0; index-- {
		if strings.Contains(strings.TrimSpace(lines[index]), "cmd.test.header") {
			headerIndex = index
			break
		}
	}
	if headerIndex == -1 {
		return ""
	}
	return strings.Join(lines[headerIndex:], "\n")
}

func adjustNormalizedForSectionMarker(normalized string, normalizedAll string, preferredMarkers []string) string {
	sectionMarker := preferredSectionMarker(preferredMarkers)
	if sectionMarker == "" {
		return normalized
	}
	if strings.Contains(normalized, sectionMarker) {
		if strings.Contains(normalized, "cmd.test.header") {
			return normalized
		}
		if sectionBlock := trimToLastSectionBlock(normalizedAll, sectionMarker); sectionBlock != "" {
			return sectionBlock
		}
		return normalized
	}
	if !strings.Contains(normalizedAll, sectionMarker) {
		return normalized
	}
	if sectionBlock := trimToLastSectionBlock(normalizedAll, sectionMarker); sectionBlock != "" {
		return sectionBlock
	}
	return normalized
}

func preferredSectionMarker(preferredMarkers []string) string {
	if len(preferredMarkers) == 0 {
		return ""
	}
	marker := preferredMarkers[0]
	if strings.HasPrefix(marker, "cmd.test.section.") {
		return marker
	}
	return ""
}

func finalizePTYRun(cmd *cobra.Command, result tea.Model) (testExecutionOutcome, error) {
	model, ok := result.(*testModel)
	if !ok {
		return testExecutionOutcome{}, errors.New("unexpected test model")
	}
	if outputErr := writeInteractiveTestTranscript(cmd, testExecutionInput{}, model); outputErr != nil {
		return model.outcome, outputErr
	}
	if model.outcome.err != nil {
		return model.outcome, model.outcome.err
	}
	return model.outcome, nil
}
