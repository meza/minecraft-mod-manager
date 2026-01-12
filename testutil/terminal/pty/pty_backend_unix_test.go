//go:build !windows

package pty

import (
	"bytes"
	"errors"
	"os"
	"testing"

	creackpty "github.com/creack/pty"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/testutil/terminal"
)

type fakeTerminalFile struct {
	buffer bytes.Buffer
}

func (file *fakeTerminalFile) Read(data []byte) (int, error) {
	return file.buffer.Read(data)
}

func (file *fakeTerminalFile) Write(data []byte) (int, error) {
	return file.buffer.Write(data)
}

func (file *fakeTerminalFile) Fd() uintptr {
	return 0
}

func (file *fakeTerminalFile) Close() error {
	return nil
}

func TestOpenPTYDefaultError(t *testing.T) {
	originalOpen := openPTYFunc
	openPTYFunc = func() (*os.File, *os.File, error) {
		return nil, nil, errors.New("boom")
	}
	t.Cleanup(func() { openPTYFunc = originalOpen })

	_, err := openPTYDefault()
	require.Error(t, err)
}

func TestSetPTYSizeDefaultInvalidFile(t *testing.T) {
	err := setPTYSizeDefault(&fakeTerminalFile{}, terminal.Size{Columns: 80, Rows: 25})
	require.Error(t, err)
}

func TestSetPTYSizeDefaultSetError(t *testing.T) {
	file, err := os.CreateTemp("", "pty-size")
	require.NoError(t, err)
	defer os.Remove(file.Name())
	defer file.Close()

	originalSet := setPTYFunc
	setPTYFunc = func(_ *os.File, _ *creackpty.Winsize) error {
		return errors.New("boom")
	}
	t.Cleanup(func() { setPTYFunc = originalSet })

	err = setPTYSizeDefault(file, terminal.Size{Columns: 80, Rows: 25})
	require.Error(t, err)
}

func TestWinsizeFromSizeInvalid(t *testing.T) {
	_, err := winsizeFromSize(terminal.Size{Columns: 0, Rows: 10})
	require.Error(t, err)
}

func TestSetPTYSizeDefaultValid(t *testing.T) {
	setup, err := openPTYDefault()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, closeSetup(setup))
	})

	require.NoError(t, setPTYSizeDefault(setup.master, terminal.Size{Columns: 80, Rows: 25}))
}

func TestSetPTYSizeDefaultInvalidSize(t *testing.T) {
	setup, err := openPTYDefault()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, closeSetup(setup))
	})

	require.Error(t, setPTYSizeDefault(setup.master, terminal.Size{Columns: 0, Rows: 25}))
}
