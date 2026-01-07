package remove

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

type removeItemStatus int

const (
	removeItemPending removeItemStatus = iota
	removeItemSuccess
	removeItemFailed
)

type removeItem struct {
	Mod           models.Mod
	Status        removeItemStatus
	FailureReason string
	HasLockEntry  bool
	LockFileName  string
}

type removeExecutionErrorType int

const (
	removeExecutionErrorNone removeExecutionErrorType = iota
	removeExecutionErrorDelete
	removeExecutionErrorWriteLock
	removeExecutionErrorWriteConfig
	removeExecutionErrorUnknown
)

type removeExecutionOutcome struct {
	items   []removeItem
	err     error
	errType removeExecutionErrorType
}

type removeExecSender struct {
	send func(msg tea.Msg)
}

func (sender removeExecSender) Send(msg tea.Msg) {
	if sender.send != nil {
		sender.send(msg)
	}
}
