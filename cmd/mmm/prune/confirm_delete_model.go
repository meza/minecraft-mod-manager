package prune

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/cobra"
)

type pruneDeleteResult struct {
	results []pruneFileResult
	err     error
}

type pruneConfirmDeleteModel struct {
	listView  string
	prompt    confirmPromptModel
	colorMode view.ColorMode
	unmanaged []string
	deleteFn  func() ([]pruneFileResult, error)
	results   []pruneFileResult
	deleteErr error
	done      bool
	canceled  bool
	confirmed bool
}

type confirmDeleteOutcome struct {
	confirmed bool
	canceled  bool
	results   []pruneFileResult
	deleteErr error
}

func newPruneConfirmDeleteModel(listView string, question string, colorMode view.ColorMode, unmanaged []string, deleteFn func() ([]pruneFileResult, error)) pruneConfirmDeleteModel {
	return pruneConfirmDeleteModel{
		listView:  listView,
		prompt:    newConfirmPromptModel(question),
		colorMode: colorMode,
		unmanaged: unmanaged,
		deleteFn:  deleteFn,
	}
}

func (model pruneConfirmDeleteModel) Init() tea.Cmd {
	return nil
}

func (model pruneConfirmDeleteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		if typed.String() == "ctrl+c" || typed.String() == "esc" {
			model.canceled = true
			return model, tea.Quit
		}
	case confirmSelectedMessage:
		model.confirmed = typed.confirmed
		if !typed.confirmed {
			return model, tea.Quit
		}
		return model, model.runDelete()
	case pruneDeleteResult:
		model.results = typed.results
		model.deleteErr = typed.err
		model.done = true
		return model, tea.Quit
	}

	updatedPrompt, cmd := model.prompt.Update(msg)
	model.prompt = updatedPrompt
	return model, cmd
}

func (model pruneConfirmDeleteModel) View() string {
	if model.done {
		if model.deleteErr != nil {
			return renderDeleteFailedView(model.colorMode, model.results)
		}
		return renderDeleteSuccessView(model.colorMode, model.results)
	}
	if model.confirmed {
		return renderDeletingList(model.colorMode, model.unmanaged)
	}
	if model.listView == "" {
		return model.prompt.View()
	}
	sections := []string{model.listView, model.prompt.View()}
	return view.RenderViewSections(sections, view.SectionSeparatorParagraph)
}

func (model pruneConfirmDeleteModel) runDelete() tea.Cmd {
	return func() tea.Msg {
		results, err := model.deleteFn()
		return pruneDeleteResult{results: results, err: err}
	}
}

func runConfirmDeleteFlow(
	cmd *cobra.Command,
	deps pruneDeps,
	colorMode view.ColorMode,
	unmanaged []string,
	deleteFn func() ([]pruneFileResult, error),
) (confirmDeleteOutcome, error) {
	listView := renderUnmanagedList(colorMode, unmanaged)
	question := i18n.T("cmd.prune.confirm", nil)
	model := newPruneConfirmDeleteModel(listView, question, colorMode, unmanaged, deleteFn)

	runTea := deps.runTea
	if runTea == nil {
		runTea = runTeaProgram
	}

	result, err := runTea(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
	if err != nil {
		return confirmDeleteOutcome{}, err
	}

	outcome, err := pruneConfirmDeleteResult(result)
	return outcome, err
}

func pruneConfirmDeleteResult(result tea.Model) (confirmDeleteOutcome, error) {
	switch typed := result.(type) {
	case pruneConfirmDeleteModel:
		return confirmDeleteOutcome{
			confirmed: typed.confirmed,
			canceled:  typed.canceled,
			results:   typed.results,
			deleteErr: typed.deleteErr,
		}, nil
	case *pruneConfirmDeleteModel:
		return confirmDeleteOutcome{
			confirmed: typed.confirmed,
			canceled:  typed.canceled,
			results:   typed.results,
			deleteErr: typed.deleteErr,
		}, nil
	default:
		return confirmDeleteOutcome{}, errors.New("unexpected confirm delete model")
	}
}
