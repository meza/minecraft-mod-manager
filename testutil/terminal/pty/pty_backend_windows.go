//go:build windows

package pty

import (
	"errors"
	"fmt"
	"io"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/meza/minecraft-mod-manager/testutil/terminal"
)

type virtualPTY struct {
	reader    *os.File
	writer    *os.File
	consoleFD uintptr
}

func (pty *virtualPTY) Read(data []byte) (int, error) {
	if pty.reader == nil {
		return 0, io.EOF
	}
	return pty.reader.Read(data)
}

func (pty *virtualPTY) Write(data []byte) (int, error) {
	if pty.writer == nil {
		return 0, errors.New("pty writer unavailable")
	}
	return pty.writer.Write(data)
}

func (pty *virtualPTY) Fd() uintptr {
	return pty.consoleFD
}

func (pty *virtualPTY) Close() error {
	return errors.Join(closePipeFile(pty.reader), closePipeFile(pty.writer))
}

type consoleRestore struct {
	handle  windows.Handle
	restore bool
}

func (restore consoleRestore) Close() error {
	if !restore.restore || restore.handle == 0 {
		return nil
	}
	return setConsoleActiveScreenBuffer(restore.handle)
}

var (
	modKernel32                = windows.NewLazySystemDLL("kernel32.dll")
	procAllocConsole           = modKernel32.NewProc("AllocConsole")
	procCreateConsoleBuffer    = modKernel32.NewProc("CreateConsoleScreenBuffer")
	procSetConsoleScreenBuffer = modKernel32.NewProc("SetConsoleScreenBufferSize")
	procSetActiveConsoleBuffer = modKernel32.NewProc("SetConsoleActiveScreenBuffer")
	procSetConsoleWindowInfo   = modKernel32.NewProc("SetConsoleWindowInfo")

	getStdHandleFunc                 = windows.GetStdHandle
	getConsoleModeFunc               = windows.GetConsoleMode
	getConsoleScreenBufferInfoFunc   = windows.GetConsoleScreenBufferInfo
	openFileFunc                     = os.OpenFile
	pipeFunc                         = os.Pipe
	allocConsoleCall                 = allocConsoleProc
	createConsoleScreenBufferCall    = createConsoleScreenBufferProc
	setConsoleActiveScreenBufferCall = setConsoleActiveScreenBufferProc
	setConsoleScreenBufferSizeCall   = setConsoleScreenBufferSizeProc
	setConsoleWindowInfoCall         = setConsoleWindowInfoProc
	closePipeFunc                    = func(file *os.File) error { return file.Close() }
)

func openPTYDefault() (ptySetup, error) {
	if err := allocConsole(); err != nil && !errors.Is(err, windows.ERROR_ACCESS_DENIED) {
		return ptySetup{}, err
	}

	previousOutputHandle, err := getStdHandleFunc(windows.STD_OUTPUT_HANDLE)
	if err != nil {
		return ptySetup{}, err
	}
	restore := consoleRestore{handle: previousOutputHandle, restore: isConsoleHandle(previousOutputHandle)}

	bufferHandle, err := createConsoleScreenBuffer()
	if err != nil {
		return ptySetup{}, err
	}

	if activeErr := setConsoleActiveScreenBuffer(bufferHandle); activeErr != nil {
		closeErr := windows.CloseHandle(bufferHandle)
		return ptySetup{}, errors.Join(activeErr, closeErr)
	}

	inputConsole, outputConsole, err := openConsoleFiles(bufferHandle)
	if err != nil {
		return ptySetup{}, err
	}

	pipes, err := openPTYPipes()
	if err != nil {
		return ptySetup{}, closeWithJoin(err, inputConsole, outputConsole, restore)
	}

	master := &virtualPTY{reader: pipes.outputReader, writer: pipes.inputWriter, consoleFD: outputConsole.Fd()}
	input := &virtualPTY{reader: pipes.inputReader, consoleFD: inputConsole.Fd()}
	output := &virtualPTY{writer: pipes.outputWriter, consoleFD: outputConsole.Fd()}

	extraClosers := appendConsoleClosers(inputConsole, outputConsole, restore)
	return ptySetup{
		master:       master,
		input:        input,
		output:       output,
		extraClosers: extraClosers,
	}, nil
}

func setPTYSizeDefault(file terminalFile, size terminal.Size) error {
	if err := validateSize(size); err != nil {
		return err
	}
	if err := validateWindowsPTYSize(size); err != nil {
		return err
	}

	handle := windows.Handle(file.Fd())
	currentSize, err := currentConsoleSize(handle)
	if err != nil {
		return err
	}

	return resizeConsole(handle, currentSize, size)
}

func validateWindowsPTYSize(size terminal.Size) error {
	const maxInt16 = int(^uint16(0) >> 1)
	if size.Columns > maxInt16 || size.Rows > maxInt16 {
		return fmt.Errorf("terminal size out of range: %s", size)
	}
	return nil
}

func currentConsoleSize(handle windows.Handle) (terminal.Size, error) {
	var info windows.ConsoleScreenBufferInfo
	if err := getConsoleScreenBufferInfoFunc(handle, &info); err != nil {
		return terminal.Size{}, err
	}

	return terminal.Size{
		Columns: int(info.Window.Right-info.Window.Left) + 1,
		Rows:    int(info.Window.Bottom-info.Window.Top) + 1,
	}, nil
}

func resizeConsole(handle windows.Handle, currentSize terminal.Size, targetSize terminal.Size) error {
	coord := consoleCoord(targetSize)
	window := consoleWindow(targetSize)

	if isConsoleShrink(currentSize, targetSize) {
		return resizeConsoleShrink(handle, window, coord)
	}

	if isConsoleExpand(currentSize, targetSize) {
		return resizeConsoleExpand(handle, window, coord)
	}

	return resizeConsoleMixed(handle, currentSize, targetSize, window, coord)
}

func consoleCoord(size terminal.Size) windows.Coord {
	//nolint:gosec // Bounds checked above.
	return windows.Coord{X: int16(size.Columns), Y: int16(size.Rows)}
}

func consoleWindow(size terminal.Size) windows.SmallRect {
	return windows.SmallRect{
		Left: 0,
		Top:  0,
		//nolint:gosec // Bounds checked above.
		Right: int16(size.Columns - 1),
		//nolint:gosec // Bounds checked above.
		Bottom: int16(size.Rows - 1),
	}
}

func isConsoleShrink(currentSize terminal.Size, targetSize terminal.Size) bool {
	return targetSize.Columns <= currentSize.Columns && targetSize.Rows <= currentSize.Rows
}

func isConsoleExpand(currentSize terminal.Size, targetSize terminal.Size) bool {
	return targetSize.Columns >= currentSize.Columns && targetSize.Rows >= currentSize.Rows
}

func resizeConsoleShrink(handle windows.Handle, window windows.SmallRect, coord windows.Coord) error {
	if err := setConsoleWindowInfo(handle, &window); err != nil {
		return err
	}
	if err := setConsoleScreenBufferSize(handle, coord); err != nil {
		return err
	}
	return nil
}

func resizeConsoleExpand(handle windows.Handle, window windows.SmallRect, coord windows.Coord) error {
	if err := setConsoleScreenBufferSize(handle, coord); err != nil {
		return err
	}
	if err := setConsoleWindowInfo(handle, &window); err != nil {
		return err
	}
	return nil
}

func resizeConsoleMixed(handle windows.Handle, currentSize terminal.Size, targetSize terminal.Size, window windows.SmallRect, coord windows.Coord) error {
	// Mixed resize: adjust the window within the current buffer before resizing it.
	interimColumns := targetSize.Columns
	if interimColumns > currentSize.Columns {
		interimColumns = currentSize.Columns
	}
	interimRows := targetSize.Rows
	if interimRows > currentSize.Rows {
		interimRows = currentSize.Rows
	}
	interimWindow := consoleWindow(terminal.Size{Columns: interimColumns, Rows: interimRows})

	if err := setConsoleWindowInfo(handle, &interimWindow); err != nil {
		return err
	}
	if err := setConsoleScreenBufferSize(handle, coord); err != nil {
		return err
	}
	if err := setConsoleWindowInfo(handle, &window); err != nil {
		return err
	}
	return nil
}

func allocConsole() error {
	result, err := allocConsoleCall()
	if result == 0 {
		return err
	}
	return nil
}

func createConsoleScreenBuffer() (windows.Handle, error) {
	const (
		accessMode            = windows.GENERIC_READ | windows.GENERIC_WRITE
		shareMode             = windows.FILE_SHARE_READ | windows.FILE_SHARE_WRITE
		consoleTextModeBuffer = 1
	)
	result, err := createConsoleScreenBufferCall(
		uintptr(accessMode),
		uintptr(shareMode),
		uintptr(consoleTextModeBuffer),
	)
	if result == 0 {
		return 0, err
	}
	return windows.Handle(result), nil
}

func setConsoleActiveScreenBuffer(handle windows.Handle) error {
	result, err := setConsoleActiveScreenBufferCall(uintptr(handle))
	if result == 0 {
		return err
	}
	return nil
}

func setConsoleScreenBufferSize(handle windows.Handle, coord windows.Coord) error {
	//nolint:gosec // Required conversion for Windows API call.
	coordValue := *(*uint32)(unsafe.Pointer(&coord))
	result, err := setConsoleScreenBufferSizeCall(uintptr(handle), uintptr(coordValue))
	if result == 0 {
		return err
	}
	return nil
}

func setConsoleWindowInfo(handle windows.Handle, window *windows.SmallRect) error {
	//nolint:gosec // Required conversion for Windows API call.
	result, err := setConsoleWindowInfoCall(uintptr(handle), uintptr(1), uintptr(unsafe.Pointer(window)))
	if result == 0 {
		return err
	}
	return nil
}

func isConsoleHandle(handle windows.Handle) bool {
	var mode uint32
	return getConsoleModeFunc(handle, &mode) == nil
}

type ptyPipes struct {
	inputReader  *os.File
	inputWriter  *os.File
	outputReader *os.File
	outputWriter *os.File
}

func openPTYPipes() (ptyPipes, error) {
	inputReader, inputWriter, err := pipeFunc()
	if err != nil {
		return ptyPipes{}, err
	}

	outputReader, outputWriter, err := pipeFunc()
	if err != nil {
		return ptyPipes{}, closeWithJoin(err, inputReader, inputWriter)
	}

	return ptyPipes{
		inputReader:  inputReader,
		inputWriter:  inputWriter,
		outputReader: outputReader,
		outputWriter: outputWriter,
	}, nil
}

func openConsoleFiles(bufferHandle windows.Handle) (input *os.File, output *os.File, err error) {
	inputConsole, err := openFileFunc("CONIN$", os.O_RDWR, 0o644)
	if err != nil {
		closeErr := windows.CloseHandle(bufferHandle)
		return nil, nil, errors.Join(err, closeErr)
	}

	outputConsole := os.NewFile(uintptr(bufferHandle), "pty-console")
	return inputConsole, outputConsole, nil
}

func appendConsoleClosers(inputConsole *os.File, outputConsole *os.File, restore consoleRestore) []io.Closer {
	extraClosers := []io.Closer{inputConsole, outputConsole}
	if restore.restore {
		extraClosers = append(extraClosers, restore)
	}
	return extraClosers
}

func closeWithJoin(err error, closers ...io.Closer) error {
	joinErr := err
	for _, closer := range closers {
		if closer == nil {
			continue
		}
		if closeErr := closer.Close(); closeErr != nil {
			joinErr = errors.Join(joinErr, closeErr)
		}
	}
	return joinErr
}

func closePipeFile(file *os.File) error {
	if file == nil {
		return nil
	}
	if err := closePipeFunc(file); err != nil && !errors.Is(err, os.ErrClosed) {
		return err
	}
	return nil
}

func allocConsoleProc() (uintptr, error) {
	result, _, err := procAllocConsole.Call()
	return result, err
}

func createConsoleScreenBufferProc(accessMode uintptr, shareMode uintptr, consoleTextModeBuffer uintptr) (uintptr, error) {
	result, _, err := procCreateConsoleBuffer.Call(accessMode, shareMode, 0, consoleTextModeBuffer, 0)
	return result, err
}

func setConsoleActiveScreenBufferProc(handle uintptr) (uintptr, error) {
	result, _, err := procSetActiveConsoleBuffer.Call(handle)
	return result, err
}

func setConsoleScreenBufferSizeProc(handle uintptr, coordValue uintptr) (uintptr, error) {
	result, _, err := procSetConsoleScreenBuffer.Call(handle, coordValue)
	return result, err
}

func setConsoleWindowInfoProc(handle uintptr, absolute uintptr, window uintptr) (uintptr, error) {
	result, _, err := procSetConsoleWindowInfo.Call(handle, absolute, window)
	return result, err
}
