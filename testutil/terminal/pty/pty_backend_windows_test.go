//go:build windows

package pty

import (
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"

	"github.com/meza/minecraft-mod-manager/testutil/terminal"
)

type backendSnapshot struct {
	getStdHandleFunc                 func(uint32) (windows.Handle, error)
	getConsoleModeFunc               func(windows.Handle, *uint32) error
	getConsoleScreenBufferInfoFunc   func(windows.Handle, *windows.ConsoleScreenBufferInfo) error
	openFileFunc                     func(string, int, os.FileMode) (*os.File, error)
	pipeFunc                         func() (*os.File, *os.File, error)
	allocConsoleCall                 func() (uintptr, error)
	createConsoleScreenBufferCall    func(uintptr, uintptr, uintptr) (uintptr, error)
	setConsoleActiveScreenBufferCall func(uintptr) (uintptr, error)
	setConsoleScreenBufferSizeCall   func(uintptr, uintptr) (uintptr, error)
	setConsoleWindowInfoCall         func(uintptr, uintptr, uintptr) (uintptr, error)
	closePipeFunc                    func(*os.File) error
}

func snapshotBackend() backendSnapshot {
	return backendSnapshot{
		getStdHandleFunc:                 getStdHandleFunc,
		getConsoleModeFunc:               getConsoleModeFunc,
		getConsoleScreenBufferInfoFunc:   getConsoleScreenBufferInfoFunc,
		openFileFunc:                     openFileFunc,
		pipeFunc:                         pipeFunc,
		allocConsoleCall:                 allocConsoleCall,
		createConsoleScreenBufferCall:    createConsoleScreenBufferCall,
		setConsoleActiveScreenBufferCall: setConsoleActiveScreenBufferCall,
		setConsoleScreenBufferSizeCall:   setConsoleScreenBufferSizeCall,
		setConsoleWindowInfoCall:         setConsoleWindowInfoCall,
		closePipeFunc:                    closePipeFunc,
	}
}

func restoreBackend(snapshot backendSnapshot) {
	getStdHandleFunc = snapshot.getStdHandleFunc
	getConsoleModeFunc = snapshot.getConsoleModeFunc
	getConsoleScreenBufferInfoFunc = snapshot.getConsoleScreenBufferInfoFunc
	openFileFunc = snapshot.openFileFunc
	pipeFunc = snapshot.pipeFunc
	allocConsoleCall = snapshot.allocConsoleCall
	createConsoleScreenBufferCall = snapshot.createConsoleScreenBufferCall
	setConsoleActiveScreenBufferCall = snapshot.setConsoleActiveScreenBufferCall
	setConsoleScreenBufferSizeCall = snapshot.setConsoleScreenBufferSizeCall
	setConsoleWindowInfoCall = snapshot.setConsoleWindowInfoCall
	closePipeFunc = snapshot.closePipeFunc
}

type winCloserFunc func() error

func (closer winCloserFunc) Close() error {
	return closer()
}

type fakeTerminalFile struct {
	fd uintptr
}

func (file fakeTerminalFile) Read([]byte) (int, error)  { return 0, io.EOF }
func (file fakeTerminalFile) Write([]byte) (int, error) { return 0, errors.New("write") }
func (file fakeTerminalFile) Fd() uintptr               { return file.fd }
func (file fakeTerminalFile) Close() error              { return nil }

func TestVirtualPTYReadWriteAndClose(t *testing.T) {
	reader, writer, err := os.Pipe()
	require.NoError(t, err)

	pty := &virtualPTY{reader: reader, writer: writer}
	_, writeErr := pty.Write([]byte("ok"))
	require.NoError(t, writeErr)

	buffer := make([]byte, 2)
	count, readErr := pty.Read(buffer)
	require.NoError(t, readErr)
	require.Equal(t, 2, count)
	require.Equal(t, "ok", string(buffer))

	require.NoError(t, pty.Close())
}

func TestVirtualPTYNilReadWrite(t *testing.T) {
	pty := &virtualPTY{}

	buffer := make([]byte, 1)
	_, readErr := pty.Read(buffer)
	require.ErrorIs(t, readErr, io.EOF)

	_, writeErr := pty.Write([]byte("x"))
	require.Error(t, writeErr)
}

func TestVirtualPTYCloseReaderOnly(t *testing.T) {
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	defer writer.Close()

	pty := &virtualPTY{reader: reader}
	require.NoError(t, pty.Close())
}

func TestVirtualPTYCloseWriterOnly(t *testing.T) {
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	defer reader.Close()

	pty := &virtualPTY{writer: writer}
	require.NoError(t, pty.Close())
}

func TestVirtualPTYCloseBothSides(t *testing.T) {
	snapshot := snapshotBackend()
	t.Cleanup(func() { restoreBackend(snapshot) })

	reader, err := os.CreateTemp("", "pty-reader")
	require.NoError(t, err)
	defer reader.Close()

	writer, err := os.CreateTemp("", "pty-writer")
	require.NoError(t, err)
	defer writer.Close()

	callCount := 0
	closePipeFunc = func(*os.File) error {
		callCount++
		return nil
	}

	pty := &virtualPTY{reader: reader, writer: writer}
	require.NoError(t, pty.Close())
	require.Equal(t, 2, callCount)
}

func TestClosePipeFile(t *testing.T) {
	require.NoError(t, closePipeFile(nil))

	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	require.NoError(t, closePipeFile(reader))
	require.NoError(t, closePipeFile(writer))

	snapshot := snapshotBackend()
	t.Cleanup(func() { restoreBackend(snapshot) })

	closePipeFunc = func(*os.File) error {
		return errors.New("boom")
	}
	temp, err := os.CreateTemp("", "pipe")
	require.NoError(t, err)
	defer temp.Close()
	require.Error(t, closePipeFile(temp))
}

func TestCloseWithJoin(t *testing.T) {
	err := closeWithJoin(nil, winCloserFunc(func() error { return errors.New("boom") }), winCloserFunc(func() error { return nil }), nil)
	require.Error(t, err)
}

func TestConsoleRestoreClose(t *testing.T) {
	require.NoError(t, consoleRestore{restore: false}.Close())

	snapshot := snapshotBackend()
	t.Cleanup(func() { restoreBackend(snapshot) })

	setConsoleActiveScreenBufferCall = func(uintptr) (uintptr, error) {
		return 1, nil
	}
	restore := consoleRestore{handle: windows.Handle(1), restore: true}
	require.NoError(t, restore.Close())
}

func TestIsConsoleHandle(t *testing.T) {
	require.False(t, isConsoleHandle(0))

	setup, err := openPTYDefault()
	require.NoError(t, err)
	require.True(t, isConsoleHandle(windows.Handle(setup.output.Fd())))
	require.NoError(t, closeSetup(setup))
}

func TestOpenPTYDefaultErrorPaths(t *testing.T) {
	t.Run("allocConsole error", func(t *testing.T) {
		snapshot := snapshotBackend()
		t.Cleanup(func() { restoreBackend(snapshot) })

		allocConsoleCall = func() (uintptr, error) { return 0, errors.New("boom") }
		_, err := openPTYDefault()
		require.Error(t, err)
	})

	t.Run("getStdHandle error", func(t *testing.T) {
		snapshot := snapshotBackend()
		t.Cleanup(func() { restoreBackend(snapshot) })

		getStdHandleFunc = func(uint32) (windows.Handle, error) {
			return 0, errors.New("boom")
		}
		_, err := openPTYDefault()
		require.Error(t, err)
	})

	t.Run("createConsoleScreenBuffer error", func(t *testing.T) {
		snapshot := snapshotBackend()
		t.Cleanup(func() { restoreBackend(snapshot) })

		createConsoleScreenBufferCall = func(uintptr, uintptr, uintptr) (uintptr, error) {
			return 0, errors.New("boom")
		}
		_, err := openPTYDefault()
		require.Error(t, err)
	})

	t.Run("setConsoleActiveScreenBuffer error", func(t *testing.T) {
		snapshot := snapshotBackend()
		t.Cleanup(func() { restoreBackend(snapshot) })

		createConsoleScreenBufferCall = func(uintptr, uintptr, uintptr) (uintptr, error) {
			return 1, nil
		}
		setConsoleActiveScreenBufferCall = func(uintptr) (uintptr, error) {
			return 0, errors.New("boom")
		}
		_, err := openPTYDefault()
		require.Error(t, err)
	})

	t.Run("openConsoleFiles error", func(t *testing.T) {
		snapshot := snapshotBackend()
		t.Cleanup(func() { restoreBackend(snapshot) })

		openFileFunc = func(string, int, os.FileMode) (*os.File, error) {
			return nil, errors.New("boom")
		}
		_, err := openPTYDefault()
		require.Error(t, err)
	})

	t.Run("openPTYPipes error", func(t *testing.T) {
		snapshot := snapshotBackend()
		t.Cleanup(func() { restoreBackend(snapshot) })

		callCount := 0
		pipeFunc = func() (*os.File, *os.File, error) {
			callCount++
			if callCount == 2 {
				return nil, nil, errors.New("boom")
			}
			return os.Pipe()
		}

		_, err := openPTYDefault()
		require.Error(t, err)
	})

	t.Run("openPTYPipes first error", func(t *testing.T) {
		snapshot := snapshotBackend()
		t.Cleanup(func() { restoreBackend(snapshot) })

		pipeFunc = func() (*os.File, *os.File, error) {
			return nil, nil, errors.New("boom")
		}

		_, err := openPTYDefault()
		require.Error(t, err)
	})
}

func TestSetPTYSizeDefaultBranches(t *testing.T) {
	setup, err := openPTYDefault()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, closeSetup(setup)) })

	handle := windows.Handle(setup.master.Fd())
	var info windows.ConsoleScreenBufferInfo
	require.NoError(t, getConsoleScreenBufferInfoFunc(handle, &info))

	currentWidth := int(info.Window.Right-info.Window.Left) + 1
	currentHeight := int(info.Window.Bottom-info.Window.Top) + 1

	stableSize := terminal.Size{Columns: currentWidth, Rows: currentHeight}
	require.NoError(t, setPTYSizeDefault(setup.master, stableSize))

	shrinkColumns := currentWidth
	shrinkRows := currentHeight
	if currentWidth > 1 {
		shrinkColumns = currentWidth - 1
	}
	if currentHeight > 1 {
		shrinkRows = currentHeight - 1
	}

	if shrinkColumns == currentWidth && shrinkRows == currentHeight {
		expanded := terminal.Size{Columns: currentWidth + 1, Rows: currentHeight + 1}
		require.NoError(t, setPTYSizeDefault(setup.master, expanded))
		shrinkColumns = expanded.Columns - 1
		shrinkRows = expanded.Rows - 1
	}

	require.NoError(t, setPTYSizeDefault(setup.master, terminal.Size{Columns: shrinkColumns, Rows: shrinkRows}))
}

func TestSetPTYSizeDefaultExpandResize(t *testing.T) {
	snapshot := snapshotBackend()
	t.Cleanup(func() { restoreBackend(snapshot) })

	currentWidth := int16(80)
	currentHeight := int16(25)
	getConsoleScreenBufferInfoFunc = func(_ windows.Handle, info *windows.ConsoleScreenBufferInfo) error {
		info.Window = windows.SmallRect{Left: 0, Top: 0, Right: currentWidth - 1, Bottom: currentHeight - 1}
		return nil
	}

	var events []string
	setConsoleWindowInfoCall = func(_ uintptr, _ uintptr, _ uintptr) (uintptr, error) {
		events = append(events, "window")
		return 1, nil
	}
	setConsoleScreenBufferSizeCall = func(uintptr, uintptr) (uintptr, error) {
		events = append(events, "buffer")
		return 1, nil
	}

	err := setPTYSizeDefault(fakeTerminalFile{fd: 1}, terminal.Size{Columns: 120, Rows: 30})
	require.NoError(t, err)
	require.Equal(t, []string{"buffer", "window"}, events)
}

func TestSetPTYSizeDefaultMixedResize(t *testing.T) {
	snapshot := snapshotBackend()
	t.Cleanup(func() { restoreBackend(snapshot) })

	currentWidth := int16(120)
	currentHeight := int16(30)
	getConsoleScreenBufferInfoFunc = func(_ windows.Handle, info *windows.ConsoleScreenBufferInfo) error {
		info.Window = windows.SmallRect{Left: 0, Top: 0, Right: currentWidth - 1, Bottom: currentHeight - 1}
		return nil
	}

	var events []string
	setConsoleWindowInfoCall = func(_ uintptr, _ uintptr, windowPtr uintptr) (uintptr, error) {
		events = append(events, "window")
		return 1, nil
	}

	setConsoleScreenBufferSizeCall = func(uintptr, uintptr) (uintptr, error) {
		events = append(events, "buffer")
		return 1, nil
	}

	err := setPTYSizeDefault(fakeTerminalFile{fd: 1}, terminal.Size{Columns: 80, Rows: 40})
	require.NoError(t, err)
	require.Equal(t, []string{"window", "buffer", "window"}, events)
}

func TestSetPTYSizeDefaultMixedResizeColumnsExpand(t *testing.T) {
	snapshot := snapshotBackend()
	t.Cleanup(func() { restoreBackend(snapshot) })

	currentWidth := int16(80)
	currentHeight := int16(40)
	getConsoleScreenBufferInfoFunc = func(_ windows.Handle, info *windows.ConsoleScreenBufferInfo) error {
		info.Window = windows.SmallRect{Left: 0, Top: 0, Right: currentWidth - 1, Bottom: currentHeight - 1}
		return nil
	}

	var events []string
	setConsoleWindowInfoCall = func(_ uintptr, _ uintptr, _ uintptr) (uintptr, error) {
		events = append(events, "window")
		return 1, nil
	}
	setConsoleScreenBufferSizeCall = func(uintptr, uintptr) (uintptr, error) {
		events = append(events, "buffer")
		return 1, nil
	}

	err := setPTYSizeDefault(fakeTerminalFile{fd: 1}, terminal.Size{Columns: 120, Rows: 30})
	require.NoError(t, err)
	require.Equal(t, []string{"window", "buffer", "window"}, events)
}

func TestSetPTYSizeDefaultExpandResizeBufferError(t *testing.T) {
	snapshot := snapshotBackend()
	t.Cleanup(func() { restoreBackend(snapshot) })

	getConsoleScreenBufferInfoFunc = func(_ windows.Handle, info *windows.ConsoleScreenBufferInfo) error {
		info.Window = windows.SmallRect{Left: 0, Top: 0, Right: 79, Bottom: 24}
		return nil
	}
	setConsoleScreenBufferSizeCall = func(uintptr, uintptr) (uintptr, error) {
		return 0, errors.New("boom")
	}

	err := setPTYSizeDefault(fakeTerminalFile{fd: 1}, terminal.Size{Columns: 120, Rows: 30})
	require.Error(t, err)
}

func TestSetPTYSizeDefaultExpandResizeWindowError(t *testing.T) {
	snapshot := snapshotBackend()
	t.Cleanup(func() { restoreBackend(snapshot) })

	getConsoleScreenBufferInfoFunc = func(_ windows.Handle, info *windows.ConsoleScreenBufferInfo) error {
		info.Window = windows.SmallRect{Left: 0, Top: 0, Right: 79, Bottom: 24}
		return nil
	}
	setConsoleScreenBufferSizeCall = func(uintptr, uintptr) (uintptr, error) {
		return 1, nil
	}
	setConsoleWindowInfoCall = func(uintptr, uintptr, uintptr) (uintptr, error) {
		return 0, errors.New("boom")
	}

	err := setPTYSizeDefault(fakeTerminalFile{fd: 1}, terminal.Size{Columns: 120, Rows: 30})
	require.Error(t, err)
}

func TestSetPTYSizeDefaultMixedResizeWindowError(t *testing.T) {
	snapshot := snapshotBackend()
	t.Cleanup(func() { restoreBackend(snapshot) })

	getConsoleScreenBufferInfoFunc = func(_ windows.Handle, info *windows.ConsoleScreenBufferInfo) error {
		info.Window = windows.SmallRect{Left: 0, Top: 0, Right: 119, Bottom: 29}
		return nil
	}
	setConsoleWindowInfoCall = func(uintptr, uintptr, uintptr) (uintptr, error) {
		return 0, errors.New("boom")
	}

	err := setPTYSizeDefault(fakeTerminalFile{fd: 1}, terminal.Size{Columns: 80, Rows: 40})
	require.Error(t, err)
}

func TestSetPTYSizeDefaultMixedResizeBufferError(t *testing.T) {
	snapshot := snapshotBackend()
	t.Cleanup(func() { restoreBackend(snapshot) })

	getConsoleScreenBufferInfoFunc = func(_ windows.Handle, info *windows.ConsoleScreenBufferInfo) error {
		info.Window = windows.SmallRect{Left: 0, Top: 0, Right: 119, Bottom: 29}
		return nil
	}
	setConsoleWindowInfoCall = func(uintptr, uintptr, uintptr) (uintptr, error) {
		return 1, nil
	}
	setConsoleScreenBufferSizeCall = func(uintptr, uintptr) (uintptr, error) {
		return 0, errors.New("boom")
	}

	err := setPTYSizeDefault(fakeTerminalFile{fd: 1}, terminal.Size{Columns: 80, Rows: 40})
	require.Error(t, err)
}

func TestSetPTYSizeDefaultMixedResizeFinalWindowError(t *testing.T) {
	snapshot := snapshotBackend()
	t.Cleanup(func() { restoreBackend(snapshot) })

	getConsoleScreenBufferInfoFunc = func(_ windows.Handle, info *windows.ConsoleScreenBufferInfo) error {
		info.Window = windows.SmallRect{Left: 0, Top: 0, Right: 119, Bottom: 29}
		return nil
	}

	callCount := 0
	setConsoleWindowInfoCall = func(uintptr, uintptr, uintptr) (uintptr, error) {
		callCount++
		if callCount == 2 {
			return 0, errors.New("boom")
		}
		return 1, nil
	}
	setConsoleScreenBufferSizeCall = func(uintptr, uintptr) (uintptr, error) {
		return 1, nil
	}

	err := setPTYSizeDefault(fakeTerminalFile{fd: 1}, terminal.Size{Columns: 80, Rows: 40})
	require.Error(t, err)
}

func TestSetPTYSizeDefaultErrors(t *testing.T) {
	setup, err := openPTYDefault()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, closeSetup(setup)) })

	handle := windows.Handle(setup.master.Fd())
	var info windows.ConsoleScreenBufferInfo
	require.NoError(t, getConsoleScreenBufferInfoFunc(handle, &info))

	currentWidth := int(info.Window.Right-info.Window.Left) + 1
	currentHeight := int(info.Window.Bottom-info.Window.Top) + 1
	currentSize := terminal.Size{Columns: currentWidth, Rows: currentHeight}

	shrinkColumns := currentWidth
	shrinkRows := currentHeight
	if currentWidth > 1 {
		shrinkColumns = currentWidth - 1
	}
	if currentHeight > 1 {
		shrinkRows = currentHeight - 1
	}

	shrinkSize := terminal.Size{Columns: shrinkColumns, Rows: shrinkRows}
	if shrinkSize == currentSize {
		expanded := terminal.Size{Columns: currentWidth + 1, Rows: currentHeight + 1}
		require.NoError(t, setPTYSizeDefault(setup.master, expanded))
		currentSize = expanded
		shrinkSize = terminal.Size{Columns: expanded.Columns - 1, Rows: expanded.Rows - 1}
	}

	t.Run("invalid size", func(t *testing.T) {
		err := setPTYSizeDefault(setup.master, terminal.Size{Columns: 0, Rows: 1})
		require.Error(t, err)
	})

	t.Run("out of range", func(t *testing.T) {
		maxInt16 := int(^uint16(0) >> 1)
		err := setPTYSizeDefault(setup.master, terminal.Size{Columns: maxInt16 + 1, Rows: 1})
		require.Error(t, err)
	})

	t.Run("buffer info error", func(t *testing.T) {
		err := setPTYSizeDefault(fakeTerminalFile{fd: ^uintptr(0)}, terminal.Size{Columns: 10, Rows: 5})
		require.Error(t, err)
	})

	t.Run("window info error shrink", func(t *testing.T) {
		snapshot := snapshotBackend()
		t.Cleanup(func() { restoreBackend(snapshot) })

		setConsoleWindowInfoCall = func(uintptr, uintptr, uintptr) (uintptr, error) {
			return 0, errors.New("boom")
		}
		err := setPTYSizeDefault(setup.master, shrinkSize)
		require.Error(t, err)
	})

	t.Run("window info error expand", func(t *testing.T) {
		snapshot := snapshotBackend()
		t.Cleanup(func() { restoreBackend(snapshot) })

		setConsoleWindowInfoCall = func(uintptr, uintptr, uintptr) (uintptr, error) {
			return 0, errors.New("boom")
		}
		err := setPTYSizeDefault(setup.master, currentSize)
		require.Error(t, err)
	})

	t.Run("buffer size error shrink", func(t *testing.T) {
		snapshot := snapshotBackend()
		t.Cleanup(func() { restoreBackend(snapshot) })

		setConsoleScreenBufferSizeCall = func(uintptr, uintptr) (uintptr, error) {
			return 0, errors.New("boom")
		}
		err := setPTYSizeDefault(setup.master, shrinkSize)
		require.Error(t, err)
	})

	t.Run("buffer size error expand", func(t *testing.T) {
		snapshot := snapshotBackend()
		t.Cleanup(func() { restoreBackend(snapshot) })

		setConsoleScreenBufferSizeCall = func(uintptr, uintptr) (uintptr, error) {
			return 0, errors.New("boom")
		}
		err := setPTYSizeDefault(setup.master, currentSize)
		require.Error(t, err)
	})
}

func TestWinConsoleHelpersErrorBranches(t *testing.T) {
	snapshot := snapshotBackend()
	t.Cleanup(func() { restoreBackend(snapshot) })

	allocConsoleCall = func() (uintptr, error) { return 0, errors.New("boom") }
	require.Error(t, allocConsole())

	allocConsoleCall = func() (uintptr, error) { return 1, nil }
	require.NoError(t, allocConsole())

	createConsoleScreenBufferCall = func(uintptr, uintptr, uintptr) (uintptr, error) {
		return 0, errors.New("boom")
	}
	_, err := createConsoleScreenBuffer()
	require.Error(t, err)

	setConsoleActiveScreenBufferCall = func(uintptr) (uintptr, error) {
		return 0, errors.New("boom")
	}
	require.Error(t, setConsoleActiveScreenBuffer(windows.Handle(0)))

	setConsoleScreenBufferSizeCall = func(uintptr, uintptr) (uintptr, error) {
		return 0, errors.New("boom")
	}
	require.Error(t, setConsoleScreenBufferSize(windows.Handle(0), windows.Coord{X: 1, Y: 1}))

	setConsoleWindowInfoCall = func(uintptr, uintptr, uintptr) (uintptr, error) {
		return 0, errors.New("boom")
	}
	require.Error(t, setConsoleWindowInfo(windows.Handle(0), &windows.SmallRect{}))
}

func TestOpenConsoleFilesError(t *testing.T) {
	snapshot := snapshotBackend()
	t.Cleanup(func() { restoreBackend(snapshot) })

	openFileFunc = func(string, int, os.FileMode) (*os.File, error) {
		return nil, errors.New("boom")
	}
	_, _, err := openConsoleFiles(windows.Handle(0))
	require.Error(t, err)
}

func TestAppendConsoleClosers(t *testing.T) {
	restore := consoleRestore{handle: windows.Handle(0), restore: false}
	closers := appendConsoleClosers(nil, nil, restore)
	require.Len(t, closers, 2)

	restore.restore = true
	closers = appendConsoleClosers(nil, nil, restore)
	require.Len(t, closers, 3)
}
