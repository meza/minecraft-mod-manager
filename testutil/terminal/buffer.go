package terminal

import (
	"bytes"
	"io"
	"sync"
)

// Buffer is a threadsafe byte buffer for terminal I/O capture.
type Buffer struct {
	mu     sync.RWMutex
	buffer bytes.Buffer
}

// NewBuffer returns an empty Buffer.
func NewBuffer() *Buffer {
	return &Buffer{}
}

// Read reads from the buffer.
func (buffer *Buffer) Read(data []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buffer.Read(data)
}

// Write writes to the buffer.
func (buffer *Buffer) Write(data []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buffer.Write(data)
}

// Bytes returns a copy of the buffer contents.
func (buffer *Buffer) Bytes() []byte {
	buffer.mu.RLock()
	defer buffer.mu.RUnlock()
	data := buffer.buffer.Bytes()
	return append([]byte(nil), data...)
}

// String returns the buffer contents as a string.
func (buffer *Buffer) String() string {
	buffer.mu.RLock()
	defer buffer.mu.RUnlock()
	return buffer.buffer.String()
}

// Reset clears the buffer.
func (buffer *Buffer) Reset() {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	buffer.buffer.Reset()
}

// SnapshotReader reads a fresh snapshot of the buffer on each read cycle.
type SnapshotReader struct {
	source   func() []byte
	snapshot []byte
	offset   int
}

// NewSnapshotReader returns a SnapshotReader that pulls from the provided source.
func NewSnapshotReader(source func() []byte) *SnapshotReader {
	return &SnapshotReader{source: source}
}

// Read implements io.Reader.
func (reader *SnapshotReader) Read(data []byte) (int, error) {
	if reader.source == nil {
		return 0, io.EOF
	}
	if reader.snapshot == nil || reader.offset >= len(reader.snapshot) {
		reader.snapshot = reader.source()
		reader.offset = 0
	}
	if len(reader.snapshot) == 0 {
		return 0, io.EOF
	}

	remaining := reader.snapshot[reader.offset:]
	bytesCopied := copy(data, remaining)
	reader.offset += bytesCopied
	if reader.offset >= len(reader.snapshot) {
		return bytesCopied, io.EOF
	}
	return bytesCopied, nil
}
