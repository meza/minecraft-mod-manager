package view

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProgressPercentLineWithoutTotal(t *testing.T) {
	assert.Equal(t, "50%", ProgressPercentLine(0.5, 0, 0))
}

func TestProgressPercentLineWithTotal(t *testing.T) {
	assert.Equal(t, "50% (512 B / 1 KB)", ProgressPercentLine(0.5, 512, 1024))
}

func TestPercentFromRatioBounds(t *testing.T) {
	assert.Equal(t, 0, percentFromRatio(-0.1))
	assert.Equal(t, 0, percentFromRatio(0))
	assert.Equal(t, 100, percentFromRatio(1))
	assert.Equal(t, 100, percentFromRatio(2))
}

func TestFormatBytes(t *testing.T) {
	assert.Equal(t, "0 B", formatBytes(0))
	assert.Equal(t, "1 KB", formatBytes(1024))
	assert.Equal(t, "1 MB", formatBytes(1024*1024))
}

func TestFormatBytesAboveTerabyte(t *testing.T) {
	assert.Equal(t, "1024 TB", formatBytes(1024*1024*1024*1024*1024))
}

func TestNewProgressBarUsesUnicodeWhenSupported(t *testing.T) {
	restore := SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restore)

	bar := NewProgressBar()
	line := RenderProgressBar(bar, 0.5)
	assert.NotContains(t, line, "[")
	assert.NotContains(t, line, "]")
}

func TestNewProgressBarUsesAsciiWhenUnicodeUnsupported(t *testing.T) {
	restore := SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restore)

	bar := NewProgressBar()
	line := RenderProgressBar(bar, 0.5)
	assert.Contains(t, line, "[")
	assert.Contains(t, line, "]")
}
