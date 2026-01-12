package terminal

import (
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/view"
)

// ApplyTerminalDetection overrides terminal detection for the provided devices.
func ApplyTerminalDetection(test testing.TB, inputDevice *Device, outputDevice *Device, capabilities Capabilities) {
	if inputDevice == nil && outputDevice == nil {
		return
	}

	inputFD := -1
	outputFD := -1
	if inputDevice != nil {
		inputFD = int(inputDevice.Fd())
	}
	if outputDevice != nil {
		outputFD = int(outputDevice.Fd())
	}

	ApplyTerminalDetectionWithFDs(test, inputFD, outputFD, capabilities)
}

// ApplyTerminalDetectionWithFDs overrides terminal detection for the provided file descriptors.
func ApplyTerminalDetectionWithFDs(test testing.TB, inputFD int, outputFD int, capabilities Capabilities) {
	if test == nil {
		return
	}
	restore := view.SetIsTerminalFuncForTesting(func(fd int) bool {
		switch fd {
		case inputFD:
			return capabilities.InputIsTerminal
		case outputFD:
			return capabilities.OutputIsTerminal
		default:
			return false
		}
	})

	test.Cleanup(restore)
}
