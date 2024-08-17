package curseforgeFingerprint

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestHashing(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	fixturePath := filepath.Join(currentDir, "__fixtures__", "test1.md")
	fixturePath2 := filepath.Join(currentDir, "__fixtures__", "test2.md")

	actualHash1 := GetFingerprintFor(fixturePath)
	actualHash2 := GetFingerprintFor(fixturePath2)

	assert.Equal(t, "3608199863", actualHash1, "Hashes should match")
	assert.Equal(t, "3493718775", actualHash2, "Hashes should match")
}
