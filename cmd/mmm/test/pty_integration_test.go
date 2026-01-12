package test

import (
	"context"
	"errors"
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
	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: 80, Rows: int(rows)}))
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
	require.NoError(t, session.Close())

	snaps.MatchSnapshot(t, normalizeTestPTYOutput(session.OutputString(), rows))
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
		return strings.Contains(string(output), "cmd.test.section.compatible")
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
		return strings.Contains(string(output), "cmd.test.section.compatibility")
	}, terminalpty.WithWaitDuration(2*time.Second))
	snaps.MatchSnapshot(t, normalizeTestPTYOutput(session.OutputString(), rows))

	close(release)
	execErrValue := <-execErr
	require.NoError(t, execErrValue)
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
		return strings.Contains(string(output), "cmd.test.section.compatible")
	}, terminalpty.WithWaitDuration(2*time.Second))

	_, writeErr := session.SendInput([]byte("\x1b[6~"))
	require.NoError(t, writeErr)

	session.WaitForOutput(t, func(output []byte) bool {
		return strings.Contains(string(output), "cmd.test.section.not_compatible")
	}, terminalpty.WithWaitDuration(2*time.Second))
	snaps.MatchSnapshot(t, normalizeTestPTYOutput(session.OutputString(), rows))

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
		return strings.Contains(string(output), "cmd.test.section.not_compatible")
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
		return strings.Contains(string(output), "cmd.test.section.compatible")
	}, terminalpty.WithWaitDuration(2*time.Second))

	for scrollStep := 0; scrollStep < 3; scrollStep++ {
		_, writeErr := session.SendInput([]byte("\x1b[<65;1;1M"))
		require.NoError(t, writeErr)
	}

	session.WaitForOutput(t, func(output []byte) bool {
		return strings.Contains(string(output), "cmd.test.section.not_compatible")
	}, terminalpty.WithWaitDuration(2*time.Second))
	snaps.MatchSnapshot(t, normalizeTestPTYOutput(session.OutputString(), rows))

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
		return strings.Contains(string(output), "cmd.test.section.compatible")
	}, terminalpty.WithWaitDuration(2*time.Second))

	for scrollStep := 0; scrollStep < 3; scrollStep++ {
		_, writeErr := session.SendInput([]byte("\x1b[<65;1;1M"))
		require.NoError(t, writeErr)
	}

	close(release)
	execErrValue := <-execErr
	require.ErrorIs(t, execErrValue, errUnsupportedMods)
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
	require.NoError(t, session.Close())

	snaps.MatchSnapshot(t, normalizeTestPTYOutput(session.OutputString(), rows))
}

func normalizeTestPTYOutput(output string, rows uint16) string {
	trimmed := trimToLastTestFrame(output)
	return terminal.NormalizeOutput(trimmed, terminal.NormalizeOptions{
		StripControlSequences:  true,
		TrimTrailingWhitespace: true,
		TrimTrailingEmptyLines: true,
		TrimSpace:              true,
		RowLimit:               int(rows),
		PadRows:                true,
	})
}

func tallSnapshotRows() uint16 {
	if runtime.GOOS == "windows" {
		return 40
	}
	return 80
}

func trimToLastTestFrame(value string) string {
	header := "cmd.test.header"
	index := strings.LastIndex(value, header)
	if index < 0 {
		return value
	}
	return value[index:]
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
