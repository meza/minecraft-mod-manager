package locksync

import (
	"bytes"
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/meza/minecraft-mod-manager/testutil/terminal"
)

func TestLockSyncPromptSnapshots(t *testing.T) {
	terminal.ApplyFixtures(t)

	extras := lockSyncSnapshotExtras()
	listView := renderLockSyncList(extras, view.ColorDisabled)

	sizes := []struct {
		name   string
		height int
	}{
		{name: "short", height: 25},
		{name: "tall", height: 80},
	}

	cases := []struct {
		name   string
		policy Policy
		index  int
	}{
		{name: "add", policy: PolicyAdd, index: 0},
		{name: "delete", policy: PolicyDelete, index: 1},
		{name: "ignore", policy: PolicyIgnore, index: 2},
		{name: "skip", policy: PolicySkip, index: 3},
	}

	for _, size := range sizes {
		t.Run(size.name, func(t *testing.T) {
			model := newLockSyncPromptModel(listView)
			updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: size.height})
			snaps.MatchSnapshot(t, normalizeLockSyncSnapshot(updated.(lockSyncPromptModel).View()))

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					answerModel := newLockSyncPromptModel(listView)
					answerModel.list.Select(tc.index)
					answered, _ := answerModel.Update(tea.WindowSizeMsg{Width: 80, Height: size.height})
					answered, _ = answered.Update(tea.KeyMsg{Type: tea.KeyEnter})
					snaps.MatchSnapshot(t, normalizeLockSyncSnapshot(answered.(lockSyncPromptModel).View()))
				})
			}
		})
	}
}

func TestLockSyncSummarySnapshots(t *testing.T) {
	terminal.ApplyFixtures(t)

	meta := config.NewMetadata("modlist.json")
	extras := lockSyncSnapshotExtras()

	cases := []struct {
		name   string
		policy Policy
	}{
		{name: "add", policy: PolicyAdd},
		{name: "delete", policy: PolicyDelete},
		{name: "ignore", policy: PolicyIgnore},
		{name: "skip", policy: PolicySkip},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buffer := &bytes.Buffer{}
			runTea := func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
				if _, err := fmt.Fprint(buffer, model.View()); err != nil {
					return model, err
				}
				return model, nil
			}

			err := writeLockSyncSummary(lockSyncSummaryInput{
				runTea:    runTea,
				in:        bytes.NewBuffer(nil),
				out:       buffer,
				colorMode: view.ColorDisabled,
				meta:      meta,
				extras:    extras,
				policy:    tc.policy,
				command:   "install",
			})
			require.NoError(t, err)

			snaps.MatchSnapshot(t, normalizeLockSyncSnapshot(buffer.String()))
		})
	}
}

func lockSyncSnapshotExtras() []extraLockEntry {
	return []extraLockEntry{
		{
			Install: models.ModInstall{
				Type:     models.MODRINTH,
				ID:       "alpha",
				Name:     "Alpha",
				FileName: "alpha.jar",
			},
			DisplayName: "Alpha",
			FileStatus:  fileStatusMissing,
		},
		{
			Install: models.ModInstall{
				Type:     models.CURSEFORGE,
				ID:       "beta",
				Name:     "Beta",
				FileName: "mods/beta.jar",
			},
			DisplayName: "Beta",
			FileStatus:  fileStatusInvalid,
		},
		{
			Install: models.ModInstall{
				Type:     models.MODRINTH,
				ID:       "gamma",
				Name:     "Gamma",
				FileName: "gamma.jar",
			},
			DisplayName: "Gamma",
			FileStatus:  fileStatusPresent,
		},
	}
}

func normalizeLockSyncSnapshot(value string) string {
	return terminal.NormalizeOutput(value, terminal.NormalizeOptions{
		StripControlSequences:  true,
		TrimTrailingWhitespace: true,
		TrimTrailingEmptyLines: true,
		TrimSpace:              true,
	})
}
