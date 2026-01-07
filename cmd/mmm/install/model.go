package install

import (
	"context"
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type installViewState int

const (
	installViewRunning installViewState = iota
	installViewSuccess
	installViewDownloadFailed
	installViewWriteLockFailed
	installViewWriteConfigFailed
	installViewCanceled
	installViewFailed
)

type installModel struct {
	ctx        context.Context
	execRunner func(context.Context, httpclient.Sender) installExecutionOutcome
	cancel     func()
	colorMode  view.ColorMode
	items      []installItem
	indexByKey map[string]int
	state      installViewState
	outcome    installExecutionOutcome
	sender     installExecSender
}

type installExecSender struct {
	send func(tea.Msg)
}

func (sender installExecSender) Send(msg tea.Msg) {
	if sender.send == nil {
		return
	}
	sender.send(msg)
}

func newInstallModel(ctx context.Context, colorMode view.ColorMode, items []installItem, indexByKey map[string]int, cancel func(), execRunner func(context.Context, httpclient.Sender) installExecutionOutcome) *installModel {
	return &installModel{
		ctx:        ctx,
		execRunner: execRunner,
		cancel:     cancel,
		colorMode:  colorMode,
		items:      items,
		indexByKey: indexByKey,
		state:      installViewRunning,
	}
}

func (model *installModel) bindSender(send func(tea.Msg)) {
	model.sender = installExecSender{send: send}
}

func (model *installModel) Init() tea.Cmd {
	if model.sender.send == nil || model.execRunner == nil {
		return tea.Quit
	}
	return model.startInstallCmd()
}

func (model *installModel) startInstallCmd() tea.Cmd {
	return func() tea.Msg {
		if model.execRunner == nil {
			return installExecutionFinishedMsg{outcome: installExecutionOutcome{
				err:     errors.New("missing execution runner"),
				errType: installExecutionErrorUnknown,
			}}
		}
		return installExecutionFinishedMsg{outcome: model.execRunner(model.ctx, model.sender)}
	}
}

func (model *installModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case installItemProgressMsg:
		model.applyProgress(typed)
		return model, nil
	case installItemProgressErrMsg:
		model.applyProgressError(typed)
		return model, nil
	case installItemSuccessMsg:
		model.applySuccess(typed)
		return model, nil
	case installItemFailureMsg:
		model.applyFailure(typed)
		return model, nil
	case installItemAbortedMsg:
		model.applyAborted(typed)
		return model, nil
	case installExecutionFinishedMsg:
		model.outcome = typed.outcome
		model.state = viewStateFromOutcome(typed.outcome.errType)
		return model, tea.Quit
	case tea.KeyMsg:
		switch typed.String() {
		case "ctrl+c", "q", "esc":
			if model.cancel != nil {
				model.cancel()
			}
		}
		return model, nil
	default:
		return model, nil
	}
}

func (model *installModel) View() string {
	switch model.state {
	case installViewSuccess:
		return renderInstallSuccessView(model.colorMode, model.items)
	case installViewDownloadFailed:
		return renderInstallDownloadFailedViewWithHint(model.colorMode, model.items)
	case installViewWriteLockFailed:
		return renderInstallWriteLockFailedView(model.colorMode, model.items, model.outcome.lockPath)
	case installViewWriteConfigFailed:
		return renderInstallWriteConfigFailedView(model.colorMode, model.items, model.outcome.configPath)
	case installViewCanceled:
		return renderInstallCanceledView(model.colorMode, model.items)
	case installViewFailed:
		return renderInstallExecutionFailedView(model.colorMode, model.items, model.outcome.err)
	default:
		return renderInstallRunningView(model.colorMode, model.items)
	}
}

func (model *installModel) updateItem(key string, update func(*installItem)) {
	index, ok := model.indexByKey[key]
	if !ok {
		return
	}
	item := model.items[index]
	update(&item)
	model.items[index] = item
}

func (model *installModel) applyProgress(msg installItemProgressMsg) {
	model.updateItem(msg.key, func(item *installItem) {
		item.Status = installItemDownloading
		if item.Progress == nil {
			item.Progress = &installProgress{}
		}
		item.Progress.ratio = msg.progress.Ratio
		item.Progress.downloaded = msg.progress.Downloaded
		item.Progress.total = msg.progress.Total
	})
}

func (model *installModel) applyProgressError(msg installItemProgressErrMsg) {
	if msg.err == nil {
		return
	}
	model.updateItem(msg.key, func(item *installItem) {
		if strings.TrimSpace(item.FailureReason) == "" {
			item.FailureReason = msg.err.Error()
		}
	})
}

func (model *installModel) applySuccess(msg installItemSuccessMsg) {
	model.updateItem(msg.key, func(item *installItem) {
		item.Status = installItemSuccess
		item.Progress = nil
		item.FailureReason = ""
		if strings.TrimSpace(msg.displayName) != "" {
			item.DisplayName = msg.displayName
		}
	})
}

func (model *installModel) applyFailure(msg installItemFailureMsg) {
	model.updateItem(msg.key, func(item *installItem) {
		item.Status = installItemFailed
		item.Progress = nil
		item.FailureReason = msg.reason
	})
}

func (model *installModel) applyAborted(msg installItemAbortedMsg) {
	model.updateItem(msg.key, func(item *installItem) {
		item.Status = installItemAborted
		item.Progress = nil
		item.FailureReason = ""
	})
}

func viewStateFromOutcome(errType installExecutionErrorType) installViewState {
	switch errType {
	case installExecutionErrorDownload:
		return installViewDownloadFailed
	case installExecutionErrorWriteLock:
		return installViewWriteLockFailed
	case installExecutionErrorWriteConfig:
		return installViewWriteConfigFailed
	case installExecutionErrorCanceled:
		return installViewCanceled
	case installExecutionErrorUnknown:
		return installViewFailed
	default:
		return installViewSuccess
	}
}
