package terminal

import "sync/atomic"

var nextDeviceID atomic.Int64

// Device wraps a buffer and exposes a stable file descriptor for terminal checks.
type Device struct {
	buffer *Buffer
	fd     int
}

// NewDevice returns a Device with a unique file descriptor.
func NewDevice() *Device {
	id := int(nextDeviceID.Add(1))
	return NewDeviceWithFD(id)
}

// NewDeviceWithFD returns a Device with the provided file descriptor.
func NewDeviceWithFD(fd int) *Device {
	return &Device{buffer: NewBuffer(), fd: fd}
}

// Read reads from the underlying buffer.
func (device *Device) Read(data []byte) (int, error) {
	return device.buffer.Read(data)
}

// Write writes to the underlying buffer.
func (device *Device) Write(data []byte) (int, error) {
	return device.buffer.Write(data)
}

// Fd returns the device file descriptor.
func (device *Device) Fd() uintptr {
	return uintptr(device.fd)
}

// Bytes returns a copy of the device buffer contents.
func (device *Device) Bytes() []byte {
	return device.buffer.Bytes()
}

// String returns the device buffer contents as a string.
func (device *Device) String() string {
	return device.buffer.String()
}

// Reset clears the device buffer.
func (device *Device) Reset() {
	device.buffer.Reset()
}

// Buffer returns the underlying buffer.
func (device *Device) Buffer() *Buffer {
	return device.buffer
}
