//go:build !windows

package scan

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
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modsetup"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestScanCommandInteractivePTYRunningSnapshotShortHeight(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)

	originalRunScanProgram := runScanProgram
	runScanProgram = defaultRunScanProgram
	t.Cleanup(func() { runScanProgram = originalRunScanProgram })

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 120, Rows: 6}))

	release := make(chan struct{})
	updatesSent := make(chan struct{})
	var runningModel *scanModel

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command) error {
		items := []scanItem{
			{FileName: "alpha.jar", Status: scanItemStatusScanning},
			{FileName: "beta.jar", Status: scanItemStatusScanning},
			{FileName: "gamma.jar", Status: scanItemStatusScanning},
			{FileName: "delta.jar", Status: scanItemStatusScanning},
			{FileName: "epsilon.jar", Status: scanItemStatusScanning},
			{FileName: "zeta.jar", Status: scanItemStatusScanning},
			{FileName: "eta.jar", Status: scanItemStatusScanning},
			{FileName: "theta.jar", Status: scanItemStatusScanning},
			{FileName: "iota.jar", Status: scanItemStatusScanning},
			{FileName: "kappa.jar", Status: scanItemStatusScanning},
			{FileName: "lambda.jar", Status: scanItemStatusScanning},
			{FileName: "mu.jar", Status: scanItemStatusScanning},
			{FileName: "nu.jar", Status: scanItemStatusScanning},
			{FileName: "xi.jar", Status: scanItemStatusScanning},
			{
				FileName: "appleskin.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "appleskin.jar",
					Name:      "AppleSkin",
					ProjectID: "appleskin",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "bc.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "bc.jar",
					Name:      "Better Clouds",
					ProjectID: "better-clouds",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "inventorysorter.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "inventorysorter.jar",
					Name:      "Inventory Sorting",
					ProjectID: "inventory-sorting",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "lithium-123.23.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "lithium-123.23.jar",
					Name:      "Lithium",
					ProjectID: "lithium",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "shulkerbox.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "shulkerbox.jar",
					Name:      "Shulker Box Tooltip",
					ProjectID: "shulker-tooltip",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "sodium.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "sodium.jar",
					Name:      "Sodium",
					ProjectID: "sodium",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "soundsbegone.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "soundsbegone.jar",
					Name:      "Sounds Be Gone!",
					ProjectID: "sounds-be-gone",
					Platform:  models.MODRINTH,
				},
			},
			{FileName: "unmanaged-a.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-b.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-c.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-d.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-e.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-f.jar", Status: scanItemStatusUnknown},
			{FileName: "flaky-a.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-b.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-c.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-d.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-e.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-f.jar", Status: scanItemStatusUnsure},
		}
		index := scanIndexByFile(items)

		execRunner := func(ctx context.Context, sender scanExecSender) scanExecutionOutcome {
			close(updatesSent)
			<-release

			updated := cloneScanItems(items)
			return scanExecutionOutcome{items: updated}
		}

		model := newScanModel(scanModelInput{
			ctx:        ctx,
			colorMode:  colorModeForOutput(cmd.OutOrStdout()),
			items:      items,
			indexByKey: index,
			execRunner: execRunner,
		})
		runningModel = model

		result, runErr := runScanProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
		if runErr != nil {
			return runErr
		}
		_, outcomeErr := finalizePTYRun(cmd, result)
		return outcomeErr
	})

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)

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

	waitForOutput(t, output, "cmd.scan.header.running")
	require.NotNil(t, runningModel)
	snaps.MatchSnapshot(t, normalizeViewportSnapshot(runningModel.View()))

	close(release)
	err = <-execErr
	require.NoError(t, err)
	waitForOutput(t, output, "cmd.scan.header.results")
	closePTY(t, slave)

	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PTY output")
	}
	require.NoError(t, normalizePTYReadError(<-readErr))
}

func TestScanCommandInteractivePTYRunningSnapshotShortHeightManyMods(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)

	originalRunScanProgram := runScanProgram
	runScanProgram = defaultRunScanProgram
	t.Cleanup(func() { runScanProgram = originalRunScanProgram })

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 120, Rows: 6}))

	release := make(chan struct{})
	updatesSent := make(chan struct{})
	var runningModel *scanModel

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command) error {
		fileNames := sampleModFileNames()
		require.Len(t, fileNames, 30)
		items := scanItemsFromFileNames(fileNames)
		index := scanIndexByFile(items)

		execRunner := func(ctx context.Context, sender scanExecSender) scanExecutionOutcome {
			close(updatesSent)
			<-release

			updated := cloneScanItems(items)
			return scanExecutionOutcome{items: updated}
		}

		model := newScanModel(scanModelInput{
			ctx:        ctx,
			colorMode:  colorModeForOutput(cmd.OutOrStdout()),
			items:      items,
			indexByKey: index,
			execRunner: execRunner,
		})
		runningModel = model

		result, runErr := runScanProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
		if runErr != nil {
			return runErr
		}
		_, outcomeErr := finalizePTYRun(cmd, result)
		return outcomeErr
	})

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)

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

	waitForOutput(t, output, "cmd.scan.header.running")
	require.NotNil(t, runningModel)
	snaps.MatchSnapshot(t, normalizeViewportSnapshot(runningModel.View()))

	close(release)
	err = <-execErr
	require.NoError(t, err)
	waitForOutput(t, output, "cmd.scan.header.results")
	closePTY(t, slave)

	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PTY output")
	}
	require.NoError(t, normalizePTYReadError(<-readErr))
}

func TestScanCommandInteractivePTYRunningSnapshotMediumHeight(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)

	originalRunScanProgram := runScanProgram
	runScanProgram = defaultRunScanProgram
	t.Cleanup(func() { runScanProgram = originalRunScanProgram })

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 120, Rows: 12}))

	release := make(chan struct{})
	updatesSent := make(chan struct{})
	var runningModel *scanModel

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command) error {
		items := []scanItem{
			{FileName: "alpha.jar", Status: scanItemStatusScanning},
			{FileName: "beta.jar", Status: scanItemStatusScanning},
			{FileName: "gamma.jar", Status: scanItemStatusScanning},
			{FileName: "delta.jar", Status: scanItemStatusScanning},
			{FileName: "epsilon.jar", Status: scanItemStatusScanning},
			{FileName: "zeta.jar", Status: scanItemStatusScanning},
			{
				FileName: "appleskin.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "appleskin.jar",
					Name:      "AppleSkin",
					ProjectID: "appleskin",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "bc.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "bc.jar",
					Name:      "Better Clouds",
					ProjectID: "better-clouds",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "inventorysorter.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "inventorysorter.jar",
					Name:      "Inventory Sorting",
					ProjectID: "inventory-sorting",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "lithium-123.23.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "lithium-123.23.jar",
					Name:      "Lithium",
					ProjectID: "lithium",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "shulkerbox.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "shulkerbox.jar",
					Name:      "Shulker Box Tooltip",
					ProjectID: "shulker-tooltip",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "soundsbegone.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "soundsbegone.jar",
					Name:      "Sounds Be Gone!",
					ProjectID: "sounds-be-gone",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "sodium.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "sodium.jar",
					Name:      "Sodium",
					ProjectID: "sodium",
					Platform:  models.MODRINTH,
				},
			},
			{FileName: "unmanaged-a.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-b.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-c.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-d.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-e.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-f.jar", Status: scanItemStatusUnknown},
			{FileName: "flaky-a.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-b.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-c.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-d.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-e.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-f.jar", Status: scanItemStatusUnsure},
		}
		index := scanIndexByFile(items)

		execRunner := func(ctx context.Context, sender scanExecSender) scanExecutionOutcome {
			close(updatesSent)
			<-release
			updated := cloneScanItems(items)
			return scanExecutionOutcome{items: updated}
		}

		model := newScanModel(scanModelInput{
			ctx:        ctx,
			colorMode:  colorModeForOutput(cmd.OutOrStdout()),
			items:      items,
			indexByKey: index,
			execRunner: execRunner,
		})
		runningModel = model

		result, runErr := runScanProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
		if runErr != nil {
			return runErr
		}
		_, outcomeErr := finalizePTYRun(cmd, result)
		return outcomeErr
	})

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)

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

	waitForOutput(t, output, "cmd.scan.section.recognized")
	require.NotNil(t, runningModel)
	snaps.MatchSnapshot(t, normalizeViewportSnapshot(runningModel.View()))

	close(release)
	err = <-execErr
	require.NoError(t, err)
	waitForOutput(t, output, "cmd.scan.header.results")
	closePTY(t, slave)

	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PTY output")
	}
	require.NoError(t, normalizePTYReadError(<-readErr))
}

func TestScanCommandInteractivePTYFinalTranscriptShortHeight(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)

	originalRunScanProgram := runScanProgram
	runScanProgram = defaultRunScanProgram
	t.Cleanup(func() { runScanProgram = originalRunScanProgram })

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 120, Rows: 6}))

	release := make(chan struct{})
	updatesSent := make(chan struct{})

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command) error {
		items := []scanItem{
			{
				FileName: "appleskin.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "appleskin.jar",
					Name:      "AppleSkin",
					ProjectID: "appleskin",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "inventorysorter.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "inventorysorter.jar",
					Name:      "Inventory Sorting",
					ProjectID: "inventory-sorting",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "bc.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "bc.jar",
					Name:      "Better Clouds",
					ProjectID: "better-clouds",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "lithium-123.23.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "lithium-123.23.jar",
					Name:      "Lithium",
					ProjectID: "lithium",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "soundsbegone.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "soundsbegone.jar",
					Name:      "Sounds Be Gone!",
					ProjectID: "sounds-be-gone",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "shulkerbox.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "shulkerbox.jar",
					Name:      "Shulker Box Tooltip",
					ProjectID: "shulker-tooltip",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "sodium.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "sodium.jar",
					Name:      "Sodium",
					ProjectID: "sodium",
					Platform:  models.MODRINTH,
				},
			},
			{FileName: "unmanaged-a.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-b.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-c.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-d.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-e.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-f.jar", Status: scanItemStatusUnknown},
			{FileName: "flaky-a.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-b.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-c.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-d.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-e.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-f.jar", Status: scanItemStatusUnsure},
		}
		index := scanIndexByFile(items)

		execRunner := func(ctx context.Context, sender scanExecSender) scanExecutionOutcome {
			close(updatesSent)
			<-release
			updated := cloneScanItems(items)
			return scanExecutionOutcome{
				items: updated,
				matches: []scanMatch{
					{
						FileName:  "appleskin.jar",
						Name:      "AppleSkin",
						ProjectID: "appleskin",
						Platform:  models.MODRINTH,
					},
					{
						FileName:  "inventorysorter.jar",
						Name:      "Inventory Sorting",
						ProjectID: "inventory-sorting",
						Platform:  models.MODRINTH,
					},
					{
						FileName:  "bc.jar",
						Name:      "Better Clouds",
						ProjectID: "better-clouds",
						Platform:  models.MODRINTH,
					},
					{
						FileName:  "lithium-123.23.jar",
						Name:      "Lithium",
						ProjectID: "lithium",
						Platform:  models.MODRINTH,
					},
					{
						FileName:  "soundsbegone.jar",
						Name:      "Sounds Be Gone!",
						ProjectID: "sounds-be-gone",
						Platform:  models.MODRINTH,
					},
					{
						FileName:  "shulkerbox.jar",
						Name:      "Shulker Box Tooltip",
						ProjectID: "shulker-tooltip",
						Platform:  models.MODRINTH,
					},
					{
						FileName:  "sodium.jar",
						Name:      "Sodium",
						ProjectID: "sodium",
						Platform:  models.MODRINTH,
					},
				},
				unknown: []string{
					"unmanaged-a.jar",
					"unmanaged-b.jar",
					"unmanaged-c.jar",
					"unmanaged-d.jar",
					"unmanaged-e.jar",
					"unmanaged-f.jar",
				},
				unsure: []scanUnsure{
					{Path: "flaky-a.jar", Error: errors.New("timeout")},
					{Path: "flaky-b.jar", Error: errors.New("timeout")},
					{Path: "flaky-c.jar", Error: errors.New("timeout")},
					{Path: "flaky-d.jar", Error: errors.New("timeout")},
					{Path: "flaky-e.jar", Error: errors.New("timeout")},
					{Path: "flaky-f.jar", Error: errors.New("timeout")},
				},
			}
		}

		model := newScanModel(scanModelInput{
			ctx:        ctx,
			colorMode:  colorModeForOutput(cmd.OutOrStdout()),
			items:      items,
			indexByKey: index,
			execRunner: execRunner,
		})

		result, runErr := runScanProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
		if runErr != nil {
			return runErr
		}
		_, outcomeErr := finalizePTYRun(cmd, result)
		return outcomeErr
	})

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)

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
	waitForOutput(t, output, "cmd.scan.header.results")
	closePTY(t, slave)

	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PTY output")
	}
	require.NoError(t, normalizePTYReadError(<-readErr))

	snaps.MatchSnapshot(t, normalizePTYSnapshot(output.String()))
}

func TestScanCommandInteractivePTYRunningSnapshotTallHeight(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)

	originalRunScanProgram := runScanProgram
	runScanProgram = defaultRunScanProgram
	t.Cleanup(func() { runScanProgram = originalRunScanProgram })

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 120, Rows: 40}))

	release := make(chan struct{})
	updatesSent := make(chan struct{})
	var runningModel *scanModel

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command) error {
		items := []scanItem{
			{FileName: "alpha.jar", Status: scanItemStatusScanning},
			{FileName: "beta.jar", Status: scanItemStatusScanning},
			{FileName: "gamma.jar", Status: scanItemStatusScanning},
			{FileName: "delta.jar", Status: scanItemStatusScanning},
			{FileName: "epsilon.jar", Status: scanItemStatusScanning},
			{FileName: "zeta.jar", Status: scanItemStatusScanning},
			{
				FileName: "appleskin.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "appleskin.jar",
					Name:      "AppleSkin",
					ProjectID: "appleskin",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "inventorysorter.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "inventorysorter.jar",
					Name:      "Inventory Sorting",
					ProjectID: "inventory-sorting",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "bc.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "bc.jar",
					Name:      "Better Clouds",
					ProjectID: "better-clouds",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "lithium-123.23.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "lithium-123.23.jar",
					Name:      "Lithium",
					ProjectID: "lithium",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "soundsbegone.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "soundsbegone.jar",
					Name:      "Sounds Be Gone!",
					ProjectID: "sounds-be-gone",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "shulkerbox.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "shulkerbox.jar",
					Name:      "Shulker Box Tooltip",
					ProjectID: "shulker-tooltip",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "sodium.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "sodium.jar",
					Name:      "Sodium",
					ProjectID: "sodium",
					Platform:  models.MODRINTH,
				},
			},
			{FileName: "unmanaged-a.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-b.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-c.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-d.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-e.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-f.jar", Status: scanItemStatusUnknown},
			{FileName: "flaky-a.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-b.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-c.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-d.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-e.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-f.jar", Status: scanItemStatusUnsure},
		}
		index := scanIndexByFile(items)

		execRunner := func(ctx context.Context, sender scanExecSender) scanExecutionOutcome {
			close(updatesSent)
			<-release
			updated := cloneScanItems(items)
			return scanExecutionOutcome{items: updated}
		}

		model := newScanModel(scanModelInput{
			ctx:        ctx,
			colorMode:  colorModeForOutput(cmd.OutOrStdout()),
			items:      items,
			indexByKey: index,
			execRunner: execRunner,
		})
		runningModel = model

		result, runErr := runScanProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
		if runErr != nil {
			return runErr
		}
		_, outcomeErr := finalizePTYRun(cmd, result)
		return outcomeErr
	})

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)

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

	waitForOutput(t, output, "cmd.scan.section.unsure")
	require.NotNil(t, runningModel)
	snaps.MatchSnapshot(t, normalizeViewportSnapshot(runningModel.View()))

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

func TestScanCommandInteractivePTYFinalTranscriptMediumHeight(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)

	originalRunScanProgram := runScanProgram
	runScanProgram = defaultRunScanProgram
	t.Cleanup(func() { runScanProgram = originalRunScanProgram })

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 120, Rows: 12}))

	release := make(chan struct{})
	updatesSent := make(chan struct{})

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command) error {
		items := []scanItem{
			{
				FileName: "appleskin.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "appleskin.jar",
					Name:      "AppleSkin",
					ProjectID: "appleskin",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "inventorysorter.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "inventorysorter.jar",
					Name:      "Inventory Sorting",
					ProjectID: "inventory-sorting",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "bc.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "bc.jar",
					Name:      "Better Clouds",
					ProjectID: "better-clouds",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "lithium-123.23.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "lithium-123.23.jar",
					Name:      "Lithium",
					ProjectID: "lithium",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "soundsbegone.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "soundsbegone.jar",
					Name:      "Sounds Be Gone!",
					ProjectID: "sounds-be-gone",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "shulkerbox.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "shulkerbox.jar",
					Name:      "Shulker Box Tooltip",
					ProjectID: "shulker-tooltip",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "sodium.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "sodium.jar",
					Name:      "Sodium",
					ProjectID: "sodium",
					Platform:  models.MODRINTH,
				},
			},
			{FileName: "unmanaged-a.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-b.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-c.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-d.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-e.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-f.jar", Status: scanItemStatusUnknown},
			{FileName: "flaky-a.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-b.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-c.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-d.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-e.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-f.jar", Status: scanItemStatusUnsure},
		}
		index := scanIndexByFile(items)

		execRunner := func(ctx context.Context, sender scanExecSender) scanExecutionOutcome {
			close(updatesSent)
			<-release
			updated := cloneScanItems(items)
			return scanExecutionOutcome{
				items: updated,
				matches: []scanMatch{
					{
						FileName:  "appleskin.jar",
						Name:      "AppleSkin",
						ProjectID: "appleskin",
						Platform:  models.MODRINTH,
					},
					{
						FileName:  "inventorysorter.jar",
						Name:      "Inventory Sorting",
						ProjectID: "inventory-sorting",
						Platform:  models.MODRINTH,
					},
					{
						FileName:  "bc.jar",
						Name:      "Better Clouds",
						ProjectID: "better-clouds",
						Platform:  models.MODRINTH,
					},
					{
						FileName:  "lithium-123.23.jar",
						Name:      "Lithium",
						ProjectID: "lithium",
						Platform:  models.MODRINTH,
					},
					{
						FileName:  "soundsbegone.jar",
						Name:      "Sounds Be Gone!",
						ProjectID: "sounds-be-gone",
						Platform:  models.MODRINTH,
					},
					{
						FileName:  "shulkerbox.jar",
						Name:      "Shulker Box Tooltip",
						ProjectID: "shulker-tooltip",
						Platform:  models.MODRINTH,
					},
					{
						FileName:  "sodium.jar",
						Name:      "Sodium",
						ProjectID: "sodium",
						Platform:  models.MODRINTH,
					},
				},
				unknown: []string{
					"unmanaged-a.jar",
					"unmanaged-b.jar",
					"unmanaged-c.jar",
					"unmanaged-d.jar",
					"unmanaged-e.jar",
					"unmanaged-f.jar",
				},
				unsure: []scanUnsure{
					{Path: "flaky-a.jar", Error: errors.New("timeout")},
					{Path: "flaky-b.jar", Error: errors.New("timeout")},
					{Path: "flaky-c.jar", Error: errors.New("timeout")},
					{Path: "flaky-d.jar", Error: errors.New("timeout")},
					{Path: "flaky-e.jar", Error: errors.New("timeout")},
					{Path: "flaky-f.jar", Error: errors.New("timeout")},
				},
			}
		}

		model := newScanModel(scanModelInput{
			ctx:        ctx,
			colorMode:  colorModeForOutput(cmd.OutOrStdout()),
			items:      items,
			indexByKey: index,
			execRunner: execRunner,
		})

		result, runErr := runScanProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
		if runErr != nil {
			return runErr
		}
		_, outcomeErr := finalizePTYRun(cmd, result)
		return outcomeErr
	})

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)

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

func TestScanCommandInteractivePTYFinalTranscriptTallHeight(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)

	originalRunScanProgram := runScanProgram
	runScanProgram = defaultRunScanProgram
	t.Cleanup(func() { runScanProgram = originalRunScanProgram })

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 120, Rows: 40}))

	release := make(chan struct{})
	updatesSent := make(chan struct{})

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command) error {
		items := []scanItem{
			{
				FileName: "appleskin.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "appleskin.jar",
					Name:      "AppleSkin",
					ProjectID: "appleskin",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "inventorysorter.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "inventorysorter.jar",
					Name:      "Inventory Sorting",
					ProjectID: "inventory-sorting",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "bc.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "bc.jar",
					Name:      "Better Clouds",
					ProjectID: "better-clouds",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "lithium-123.23.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "lithium-123.23.jar",
					Name:      "Lithium",
					ProjectID: "lithium",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "soundsbegone.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "soundsbegone.jar",
					Name:      "Sounds Be Gone!",
					ProjectID: "sounds-be-gone",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "shulkerbox.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "shulkerbox.jar",
					Name:      "Shulker Box Tooltip",
					ProjectID: "shulker-tooltip",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "sodium.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "sodium.jar",
					Name:      "Sodium",
					ProjectID: "sodium",
					Platform:  models.MODRINTH,
				},
			},
			{FileName: "unmanaged-a.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-b.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-c.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-d.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-e.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-f.jar", Status: scanItemStatusUnknown},
			{FileName: "flaky-a.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-b.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-c.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-d.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-e.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-f.jar", Status: scanItemStatusUnsure},
		}
		index := scanIndexByFile(items)

		execRunner := func(ctx context.Context, sender scanExecSender) scanExecutionOutcome {
			close(updatesSent)
			<-release
			updated := cloneScanItems(items)
			return scanExecutionOutcome{
				items: updated,
				matches: []scanMatch{
					{
						FileName:  "appleskin.jar",
						Name:      "AppleSkin",
						ProjectID: "appleskin",
						Platform:  models.MODRINTH,
					},
					{
						FileName:  "inventorysorter.jar",
						Name:      "Inventory Sorting",
						ProjectID: "inventory-sorting",
						Platform:  models.MODRINTH,
					},
					{
						FileName:  "bc.jar",
						Name:      "Better Clouds",
						ProjectID: "better-clouds",
						Platform:  models.MODRINTH,
					},
					{
						FileName:  "lithium-123.23.jar",
						Name:      "Lithium",
						ProjectID: "lithium",
						Platform:  models.MODRINTH,
					},
					{
						FileName:  "soundsbegone.jar",
						Name:      "Sounds Be Gone!",
						ProjectID: "sounds-be-gone",
						Platform:  models.MODRINTH,
					},
					{
						FileName:  "shulkerbox.jar",
						Name:      "Shulker Box Tooltip",
						ProjectID: "shulker-tooltip",
						Platform:  models.MODRINTH,
					},
					{
						FileName:  "sodium.jar",
						Name:      "Sodium",
						ProjectID: "sodium",
						Platform:  models.MODRINTH,
					},
				},
				unknown: []string{
					"unmanaged-a.jar",
					"unmanaged-b.jar",
					"unmanaged-c.jar",
					"unmanaged-d.jar",
					"unmanaged-e.jar",
					"unmanaged-f.jar",
				},
				unsure: []scanUnsure{
					{Path: "flaky-a.jar", Error: errors.New("timeout")},
					{Path: "flaky-b.jar", Error: errors.New("timeout")},
					{Path: "flaky-c.jar", Error: errors.New("timeout")},
					{Path: "flaky-d.jar", Error: errors.New("timeout")},
					{Path: "flaky-e.jar", Error: errors.New("timeout")},
					{Path: "flaky-f.jar", Error: errors.New("timeout")},
				},
			}
		}

		model := newScanModel(scanModelInput{
			ctx:        ctx,
			colorMode:  colorModeForOutput(cmd.OutOrStdout()),
			items:      items,
			indexByKey: index,
			execRunner: execRunner,
		})

		result, runErr := runScanProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
		if runErr != nil {
			return runErr
		}
		_, outcomeErr := finalizePTYRun(cmd, result)
		return outcomeErr
	})

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)

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

func TestScanCommandInteractivePTYPromptConfirmAddedOnly(t *testing.T) {
	t.Setenv("LANG", "en_GB.UTF-8")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 120, Rows: 12}))

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command) error {
		fs := afero.NewMemMapFs()
		require.NoError(t, fs.MkdirAll("/cfg", 0o755))
		meta := config.NewMetadata("/cfg/modlist.json")
		cfg := models.ModsJSON{ModsFolder: "mods"}
		require.NoError(t, config.WriteConfig(ctx, fs, meta, cfg))
		require.NoError(t, config.WriteLock(ctx, fs, meta, []models.ModInstall{}))

		input := scanExecutionInput{
			meta:             meta,
			cfg:              cfg,
			lock:             []models.ModInstall{},
			setupCoordinator: modsetup.NewSetupCoordinator(fs, nil, nil),
			deps: scanDeps{
				fs: fs,
			},
		}
		outcome := scanExecutionOutcome{matches: []scanMatch{scanMatchFixture()}}

		_, runErr := handleInteractiveScanPrompt(ctx, cmd, input, colorModeForOutput(cmd.OutOrStdout()), outcome)
		return runErr
	})

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)

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

	waitForOutput(t, output, "Add recognized mods to your configuration?")
	_, err = master.Write([]byte("y\r"))
	require.NoError(t, err)

	select {
	case err = <-execErr:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for prompt acceptance to finish")
	}
	closePTY(t, slave)

	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PTY output")
	}
	require.NoError(t, normalizePTYReadError(<-readErr))

	snaps.MatchSnapshot(t, normalizePromptPTYSnapshot(output.String()))
}

func TestScanCommandInteractivePTYPromptDeclineCancelledOnly(t *testing.T) {
	t.Setenv("LANG", "en_GB.UTF-8")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 120, Rows: 12}))

	cmd := commandWithRunner(func(ctx context.Context, cmd *cobra.Command) error {
		input := scanExecutionInput{
			deps: scanDeps{},
		}
		outcome := scanExecutionOutcome{matches: []scanMatch{scanMatchFixture()}}

		_, runErr := handleInteractiveScanPrompt(ctx, cmd, input, colorModeForOutput(cmd.OutOrStdout()), outcome)
		return runErr
	})

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)

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

	waitForOutput(t, output, "Add recognized mods to your configuration?")
	_, err = master.Write([]byte("n\r"))
	require.NoError(t, err)

	select {
	case err = <-execErr:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for prompt decline to finish")
	}
	closePTY(t, slave)

	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PTY output")
	}
	require.NoError(t, normalizePTYReadError(<-readErr))

	snaps.MatchSnapshot(t, normalizePromptPTYSnapshot(output.String()))
}

func commandWithRunner(run func(context.Context, *cobra.Command) error) *cobra.Command {
	cmd := &cobra.Command{
		Use: "scan",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.Context(), cmd)
		},
	}
	return cmd
}

func finalizePTYRun(cmd *cobra.Command, result tea.Model) (scanExecutionOutcome, error) {
	model, ok := result.(*scanModel)
	if !ok {
		return scanExecutionOutcome{}, errors.New("unexpected scan model")
	}
	if outputErr := writeInteractiveScanTranscript(cmd, scanExecutionInput{}, model); outputErr != nil {
		return model.outcome, outputErr
	}
	if model.outcome.err != nil {
		return model.outcome, model.outcome.err
	}
	return model.outcome, nil
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

func stripControlSequences(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")

	value = stripOSCSequences(value)
	value = stripCSISequences(value)

	return value
}

func normalizePTYSnapshot(value string) string {
	normalized := stripControlSequences(trimToLastFrame(value))
	lines := strings.Split(normalized, "\n")
	for index, line := range lines {
		lines[index] = strings.TrimRight(line, " \t")
	}
	normalized = strings.TrimSpace(strings.Join(lines, "\n"))
	if headerIndex := strings.LastIndex(normalized, "cmd.scan.header.results"); headerIndex >= 0 {
		normalized = strings.TrimSpace(normalized[headerIndex:])
		return normalized
	}
	if headerIndex := strings.LastIndex(normalized, "cmd.scan.header.running"); headerIndex >= 0 {
		normalized = strings.TrimSpace(normalized[headerIndex:])
	}
	return normalized
}

func normalizePromptPTYSnapshot(value string) string {
	normalized := stripControlSequences(trimAfterAltScreenExit(value))
	lines := strings.Split(normalized, "\n")
	for index, line := range lines {
		lines[index] = strings.TrimRight(line, " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func trimAfterAltScreenExit(value string) string {
	const altScreenExitSequence = "\x1b[?1049l"
	index := strings.LastIndex(value, altScreenExitSequence)
	if index < 0 {
		return value
	}
	return value[index+len(altScreenExitSequence):]
}

func normalizeViewportSnapshot(value string) string {
	lines := strings.Split(value, "\n")
	for index, line := range lines {
		lines[index] = strings.TrimRight(line, " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func trimToLastFrame(value string) string {
	indices := cursorHomeSequence.FindAllStringIndex(value, -1)
	if len(indices) == 0 {
		return value
	}
	return value[indices[len(indices)-1][0]:]
}

func scanItemsFromFileNames(fileNames []string) []scanItem {
	ordered := cloneStrings(fileNames)
	sortStringsCaseInsensitive(ordered)

	items := make([]scanItem, 0, len(ordered))
	for _, name := range ordered {
		items = append(items, scanItem{FileName: name, Status: scanItemStatusScanning})
	}
	return items
}

func sampleModFileNames() []string {
	return []string{
		"ArmorPoser-fabric-1.21.10-12.3.0.jar",
		"Axiom-5.2.1-for-MC1.21.10.jar",
		"BetterServerPacksFabric-1.2.0.jar",
		"CraterLib-Fabric-1.21.9-3.0.1.jar",
		"MaintenanceMode-Universal-1.3.1.jar",
		"SimpleDiscordLink-Universal-3.3.4.jar",
		"audioplayer-fabric-2.1.0+1.21.10.jar",
		"carpet-tis-addition-v1.74.0-mc1.21.10.jar",
		"cicada-lib-0.14.3+1.21.9-1.21.10.jar",
		"cloth-config-20.0.149-fabric.jar",
		"coordfinder-fabric-1.21.10-1.1.0.jar",
		"do_a_barrel_roll-fabric-3.8.3+1.21.9.jar",
		"entityculling-fabric-1.9.5-mc1.21.10.jar",
		"fabric-api-0.138.4+1.21.10.jar",
		"fabric-carpet-1.21.10-1.4.188+v251016.jar",
		"fabric-language-kotlin-1.13.8+kotlin.2.3.0.jar",
		"ferritecore-8.0.2-fabric.jar",
		"inventorysorter-fabric-2.1.4+mc1.21.9.jar",
		"krypton-0.2.10.jar",
		"lithium-fabric-0.20.1+mc1.21.10.jar",
		"malilib-fabric-1.21.10-0.26.8.jar",
		"restart-detector-1.2.7+1.21.10.jar",
		"servux-fabric-1.21.10-0.8.5.jar",
		"spark-1.10.152-fabric.jar",
		"status-fabric-1.21.10-1.1.0.jar",
		"syncmatica-fabric-1.21.10-0.3.16.jar",
		"vanish-1.6.5+1.21.10.jar",
		"vcinteraction-fabric-1.21.10-1.0.8.jar",
		"viewdistancefix-fabric-1.21.10-1.0.2.jar",
		"voicechat-fabric-1.21.10-2.6.11.jar",
	}
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

var cursorHomeSequence = regexp.MustCompile(`\x1b\[[0-9;]*H`)

var oscSequence = regexp.MustCompile(`\x1b\][^\x07]*(\x07|\x1b\\)`)

func stripOSCSequences(value string) string {
	return oscSequence.ReplaceAllString(value, "")
}

var csiSequence = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

func stripCSISequences(value string) string {
	return csiSequence.ReplaceAllString(value, "")
}
