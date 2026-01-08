//go:build !windows

package test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/creack/pty"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestTestCommandInteractivePTYOutput(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)

	originalRunTestProgram := runTestProgram
	runTestProgram = defaultRunTestProgram
	t.Cleanup(func() { runTestProgram = originalRunTestProgram })

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 120, Rows: 40}))

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

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)
	cmd.SetArgs([]string{"1.21.1"})

	output := &lockedBuffer{}
	readDone := make(chan struct{})
	readErr := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(output, master)
		readErr <- copyErr
		close(readDone)
	}()

	execErr := cmd.Execute()
	require.ErrorIs(t, execErr, errUnsupportedMods)
	closePTY(t, slave)

	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PTY output")
	}
	require.NoError(t, normalizePTYReadError(<-readErr))

	snaps.MatchSnapshot(t, normalizePTYSnapshot(output.String()))
}

func TestTestCommandInteractivePTYRunningShowsHeadersWhenShort(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)

	originalRunTestProgram := runTestProgram
	runTestProgram = defaultRunTestProgram
	t.Cleanup(func() { runTestProgram = originalRunTestProgram })

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 120, Rows: 6}))

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

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)
	cmd.SetArgs([]string{"1.21.11"})

	output := &lockedBuffer{}
	readDone := make(chan struct{})
	readErr := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(output, master)
		readErr <- copyErr
		close(readDone)
	}()

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	select {
	case <-updatesSent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for running updates")
	}

	waitForOutput(t, output, "cmd.test.section.compatible")
	snaps.MatchSnapshot(t, normalizePTYSnapshot(output.String()))

	close(release)
	err = <-execErr
	require.ErrorIs(t, err, errUnsupportedMods)
	closePTY(t, slave)

	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PTY output")
	}
	require.NoError(t, normalizePTYReadError(<-readErr))
}

func TestTestCommandInteractivePTYRunningSnapshotMediumHeight(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)

	originalRunTestProgram := runTestProgram
	runTestProgram = defaultRunTestProgram
	t.Cleanup(func() { runTestProgram = originalRunTestProgram })

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 120, Rows: 12}))

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

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)
	cmd.SetArgs([]string{"1.21.11"})

	output := &lockedBuffer{}
	readDone := make(chan struct{})
	readErr := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(output, master)
		readErr <- copyErr
		close(readDone)
	}()

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	select {
	case <-updatesSent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for running updates")
	}

	waitForOutput(t, output, "cmd.test.section.compatibility")
	snaps.MatchSnapshot(t, normalizePTYSnapshot(output.String()))

	close(release)
	err = <-execErr
	require.NoError(t, err)
	closePTY(t, slave)

	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PTY output")
	}
	require.NoError(t, normalizePTYReadError(<-readErr))
}

func TestTestCommandInteractivePTYRunningScrollsToNotCompatible(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)

	originalRunTestProgram := runTestProgram
	runTestProgram = defaultRunTestProgram
	t.Cleanup(func() { runTestProgram = originalRunTestProgram })

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 60, Rows: 6}))

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

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)
	cmd.SetArgs([]string{"1.21.11"})

	output := &lockedBuffer{}
	readDone := make(chan struct{})
	readErr := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(output, master)
		readErr <- copyErr
		close(readDone)
	}()

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	select {
	case <-updatesSent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for running updates")
	}

	waitForOutput(t, output, "cmd.test.header")
	waitForOutput(t, output, "cmd.test.section.compatible")

	_, writeErr := master.Write([]byte("\x1b[6~"))
	require.NoError(t, writeErr)

	waitForOutput(t, output, "cmd.test.section.not_compatible")
	snaps.MatchSnapshot(t, normalizePTYSnapshot(output.String()))

	close(release)
	err = <-execErr
	require.ErrorIs(t, err, errUnsupportedMods)
	closePTY(t, slave)

	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PTY output")
	}
	require.NoError(t, normalizePTYReadError(<-readErr))
}

func TestTestCommandInteractivePTYRunningSnapshotTallHeightFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)

	originalRunTestProgram := runTestProgram
	runTestProgram = defaultRunTestProgram
	t.Cleanup(func() { runTestProgram = originalRunTestProgram })

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 120, Rows: 40}))

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

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)
	cmd.SetArgs([]string{"1.21.11"})

	output := &lockedBuffer{}
	readDone := make(chan struct{})
	readErr := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(output, master)
		readErr <- copyErr
		close(readDone)
	}()

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	select {
	case <-updatesSent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for running updates")
	}

	waitForOutput(t, output, "cmd.test.section.not_compatible")
	snaps.MatchSnapshot(t, normalizePTYSnapshot(output.String()))

	close(release)
	err = <-execErr
	require.ErrorIs(t, err, errUnsupportedMods)
	closePTY(t, slave)

	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PTY output")
	}
	require.NoError(t, normalizePTYReadError(<-readErr))
}

func TestTestCommandInteractivePTYRunningMouseScrollsToNotCompatible(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)

	originalRunTestProgram := runTestProgram
	runTestProgram = defaultRunTestProgram
	t.Cleanup(func() { runTestProgram = originalRunTestProgram })

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 60, Rows: 6}))

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

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)
	cmd.SetArgs([]string{"1.21.11"})

	output := &lockedBuffer{}
	readDone := make(chan struct{})
	readErr := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(output, master)
		readErr <- copyErr
		close(readDone)
	}()

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	select {
	case <-updatesSent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for running updates")
	}

	waitForOutput(t, output, "cmd.test.header")
	waitForOutput(t, output, "cmd.test.section.compatible")

	for i := 0; i < 3; i++ {
		_, writeErr := master.Write([]byte("\x1b[<65;1;1M"))
		require.NoError(t, writeErr)
	}

	waitForOutput(t, output, "cmd.test.section.not_compatible")
	snaps.MatchSnapshot(t, normalizePTYSnapshot(output.String()))

	close(release)
	err = <-execErr
	require.ErrorIs(t, err, errUnsupportedMods)
	closePTY(t, slave)

	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PTY output")
	}
	require.NoError(t, normalizePTYReadError(<-readErr))
}

func TestTestCommandInteractivePTYFinalTranscriptSuccessShortHeight(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)

	originalRunTestProgram := runTestProgram
	runTestProgram = defaultRunTestProgram
	t.Cleanup(func() { runTestProgram = originalRunTestProgram })

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 120, Rows: 6}))

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

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)
	cmd.SetArgs([]string{"1.21.11"})

	output := &lockedBuffer{}
	readDone := make(chan struct{})
	readErr := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(output, master)
		readErr <- copyErr
		close(readDone)
	}()

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
	err = <-execErr
	require.NoError(t, err)
	closePTY(t, slave)

	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PTY output")
	}
	require.NoError(t, normalizePTYReadError(<-readErr))

	snaps.MatchSnapshot(t, normalizePTYSnapshot(output.String()))
}

func TestTestCommandInteractivePTYFinalTranscriptIncludesSummaryAfterScroll(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)

	originalRunTestProgram := runTestProgram
	runTestProgram = defaultRunTestProgram
	t.Cleanup(func() { runTestProgram = originalRunTestProgram })

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 60, Rows: 6}))

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

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)
	cmd.SetArgs([]string{"1.21.11"})

	output := &lockedBuffer{}
	readDone := make(chan struct{})
	readErr := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(output, master)
		readErr <- copyErr
		close(readDone)
	}()

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	select {
	case <-updatesSent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for running updates")
	}

	waitForOutput(t, output, "cmd.test.section.compatible")

	for i := 0; i < 3; i++ {
		_, writeErr := master.Write([]byte("\x1b[<65;1;1M"))
		require.NoError(t, writeErr)
	}

	close(release)
	err = <-execErr
	require.ErrorIs(t, err, errUnsupportedMods)
	closePTY(t, slave)

	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PTY output")
	}
	require.NoError(t, normalizePTYReadError(<-readErr))

	snaps.MatchSnapshot(t, normalizePTYSnapshot(output.String()))
}

func TestTestCommandInteractivePTYFinalTranscriptInconclusiveMediumHeight(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)

	originalRunTestProgram := runTestProgram
	runTestProgram = defaultRunTestProgram
	t.Cleanup(func() { runTestProgram = originalRunTestProgram })

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 120, Rows: 12}))

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

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)
	cmd.SetArgs([]string{"1.21.11"})

	output := &lockedBuffer{}
	readDone := make(chan struct{})
	readErr := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(output, master)
		readErr <- copyErr
		close(readDone)
	}()

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
	err = <-execErr
	require.ErrorIs(t, err, errUnsupportedMods)
	closePTY(t, slave)

	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PTY output")
	}
	require.NoError(t, normalizePTYReadError(<-readErr))

	snaps.MatchSnapshot(t, normalizePTYSnapshot(output.String()))
}

func TestTestCommandInteractivePTYFinalTranscriptIncludesSummaryWithoutScroll(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)

	originalRunTestProgram := runTestProgram
	runTestProgram = defaultRunTestProgram
	t.Cleanup(func() { runTestProgram = originalRunTestProgram })

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 60, Rows: 6}))

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

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)
	cmd.SetArgs([]string{"1.21.11"})

	output := &lockedBuffer{}
	readDone := make(chan struct{})
	readErr := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(output, master)
		readErr <- copyErr
		close(readDone)
	}()

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
	err = <-execErr
	require.ErrorIs(t, err, errUnsupportedMods)
	closePTY(t, slave)

	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PTY output")
	}
	require.NoError(t, normalizePTYReadError(<-readErr))

	snaps.MatchSnapshot(t, normalizePTYSnapshot(output.String()))
}

func waitForOutput(t *testing.T, output *lockedBuffer, needle string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		normalized := stripControlSequences(output.String())
		if strings.Contains(normalized, needle) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for output to contain %q", needle)
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (buffer *lockedBuffer) Write(p []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buf.Write(p)
}

func (buffer *lockedBuffer) String() string {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buf.String()
}

func (buffer *lockedBuffer) Reset() {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	buffer.buf.Reset()
}

func stripControlSequences(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")

	value = stripOSCSequences(value)
	value = stripCSISequences(value)

	return value
}

func normalizePTYSnapshot(value string) string {
	normalized := stripControlSequences(value)
	lines := strings.Split(normalized, "\n")
	for index, line := range lines {
		lines[index] = strings.TrimRight(line, " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
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
