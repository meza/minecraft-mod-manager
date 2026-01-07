package prune

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/view"
)

type pruneForceResult struct {
	results []pruneFileResult
	err     error
}

type pruneForceModel struct {
	colorMode  view.ColorMode
	unmanaged  []string
	deleteFunc func() ([]pruneFileResult, error)
	results    []pruneFileResult
	deleteErr  error
	done       bool
}

func newPruneForceModel(colorMode view.ColorMode, unmanaged []string, deleteFunc func() ([]pruneFileResult, error)) pruneForceModel {
	return pruneForceModel{
		colorMode:  colorMode,
		unmanaged:  unmanaged,
		deleteFunc: deleteFunc,
	}
}

func (model pruneForceModel) Init() tea.Cmd {
	return func() tea.Msg {
		results, err := model.deleteFunc()
		return pruneForceResult{results: results, err: err}
	}
}

func (model pruneForceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case pruneForceResult:
		model.results = typed.results
		model.deleteErr = typed.err
		model.done = true
		return model, tea.Quit
	default:
		return model, nil
	}
}

func (model pruneForceModel) View() string {
	if !model.done {
		return renderDeletingList(model.colorMode, model.unmanaged)
	}
	if model.deleteErr != nil {
		return renderDeleteFailedView(model.colorMode, model.results)
	}
	return renderDeleteSuccessView(model.colorMode, model.results)
}

func pruneForceResultFromModel(result tea.Model) (results []pruneFileResult, deleteErr error, resultErr error) {
	switch typed := result.(type) {
	case pruneForceModel:
		return typed.results, typed.deleteErr, nil
	case *pruneForceModel:
		return typed.results, typed.deleteErr, nil
	default:
		return nil, nil, errors.New("unexpected prune force model")
	}
}
