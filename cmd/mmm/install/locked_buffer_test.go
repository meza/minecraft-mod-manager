package install

import (
	"bytes"
	"sync"
)

type lockedBuffer struct {
	mutex  sync.Mutex
	buffer bytes.Buffer
}

func (buffer *lockedBuffer) Write(p []byte) (int, error) {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()
	return buffer.buffer.Write(p)
}

func (buffer *lockedBuffer) String() string {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()
	return buffer.buffer.String()
}
