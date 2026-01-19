package locksync

import (
	"errors"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type lockSyncSummaryInput struct {
	runTea    func(tea.Model, ...tea.ProgramOption) (tea.Model, error)
	in        io.Reader
	out       io.Writer
	colorMode view.ColorMode
	meta      config.Metadata
	extras    []extraLockEntry
	policy    Policy
	command   string
}

func writeLockSyncSummary(input lockSyncSummaryInput) error {
	if input.runTea == nil {
		return errors.New("missing lock sync summary runner")
	}
	listView := renderLockSyncList(input.extras, input.colorMode)
	actionLines := strings.Join(lockSyncResolutionLines(input.command, input.colorMode, input.policy), "\n")
	model := view.OutputLinesModel{
		Lines:     []string{listView, actionLines},
		Output:    input.out,
		Separator: view.SectionSeparatorParagraph,
	}
	return view.RunOutputLines(input.runTea, model, lockSyncOutputProgramOptions(input.out)...)
}
