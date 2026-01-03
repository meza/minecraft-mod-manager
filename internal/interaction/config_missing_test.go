package interaction

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/tui"
)

type fakeTerminal struct {
	bytes.Buffer
}

func (terminal *fakeTerminal) Fd() uintptr {
	return uintptr(0)
}

func TestCheckConfigInitGateReturnsNilWhenPromptAllowed(t *testing.T) {
	restore := tui.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	meta := config.NewMetadata("modlist.json")
	terminal := &fakeTerminal{}
	err := CheckConfigInitGate(meta, ConfigInitGate{
		In:         terminal,
		Out:        terminal,
		Unattended: false,
		UnattendedError: func(config.Metadata) error {
			t.Fatal("UnattendedError should not be called")
			return nil
		},
		NoTTYError: func(config.Metadata) error {
			t.Fatal("NoTTYError should not be called")
			return nil
		},
	})

	assert.NoError(t, err)
}

func TestCheckConfigInitGateReturnsUnattendedError(t *testing.T) {
	restore := tui.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	meta := config.NewMetadata("modlist.json")
	expected := errors.New("unattended")
	terminal := &fakeTerminal{}
	err := CheckConfigInitGate(meta, ConfigInitGate{
		In:         terminal,
		Out:        terminal,
		Unattended: true,
		UnattendedError: func(config.Metadata) error {
			return expected
		},
		NoTTYError: func(config.Metadata) error {
			t.Fatal("NoTTYError should not be called")
			return nil
		},
	})

	assert.ErrorIs(t, err, expected)
}

func TestCheckConfigInitGateReturnsNoTTYError(t *testing.T) {
	restore := tui.SetIsTerminalFuncForTesting(func(int) bool { return false })
	t.Cleanup(restore)

	meta := config.NewMetadata("modlist.json")
	expected := errors.New("no tty")
	terminal := &fakeTerminal{}
	err := CheckConfigInitGate(meta, ConfigInitGate{
		In:         terminal,
		Out:        terminal,
		Unattended: false,
		UnattendedError: func(config.Metadata) error {
			t.Fatal("UnattendedError should not be called")
			return nil
		},
		NoTTYError: func(config.Metadata) error {
			return expected
		},
	})

	assert.ErrorIs(t, err, expected)
}

func TestCheckConfigInitGateHandlesNonTerminalReaders(t *testing.T) {
	restore := tui.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	meta := config.NewMetadata("modlist.json")
	expected := errors.New("no tty")
	err := CheckConfigInitGate(meta, ConfigInitGate{
		In:         bytes.NewBufferString(""),
		Out:        bytes.NewBufferString(""),
		Unattended: false,
		UnattendedError: func(config.Metadata) error {
			t.Fatal("UnattendedError should not be called")
			return nil
		},
		NoTTYError: func(config.Metadata) error {
			return expected
		},
	})

	assert.ErrorIs(t, err, expected)
}

var _ io.Reader = (*fakeTerminal)(nil)
var _ io.Writer = (*fakeTerminal)(nil)
