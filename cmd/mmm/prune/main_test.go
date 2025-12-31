package prune

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	originalAbs := absPath
	absPath = func(path string) (string, error) {
		return path, nil
	}
	code := m.Run()
	absPath = originalAbs
	os.Exit(code)
}
