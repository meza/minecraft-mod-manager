//go:build !windows

package pty

import (
	"errors"
	"os"

	creackpty "github.com/creack/pty"

	"github.com/meza/minecraft-mod-manager/testutil/terminal"
)

var (
	openPTYFunc = creackpty.Open
	setPTYFunc  = creackpty.Setsize
)

func openPTYDefault() (ptySetup, error) {
	master, slave, err := openPTYFunc()
	if err != nil {
		return ptySetup{}, err
	}
	return ptySetup{master: master, input: slave, output: slave}, nil
}

func setPTYSizeDefault(file terminalFile, size terminal.Size) error {
	osFile, ok := file.(*os.File)
	if !ok {
		return errors.New("pty size unsupported")
	}
	winsize, err := winsizeFromSize(size)
	if err != nil {
		return err
	}
	return setPTYFunc(osFile, winsize)
}

func winsizeFromSize(size terminal.Size) (*creackpty.Winsize, error) {
	if err := validateSize(size); err != nil {
		return nil, err
	}
	//nolint:gosec // Bounds checked in validateSize.
	return &creackpty.Winsize{Cols: uint16(size.Columns), Rows: uint16(size.Rows)}, nil
}
