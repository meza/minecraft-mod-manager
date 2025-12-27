package gameversion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNextPatchDownReturnsNextPatch(t *testing.T) {
	next, ok := NextPatchDown("1.20.2")
	assert.True(t, ok)
	assert.Equal(t, "1.20.1", next)
}

func TestNextPatchDownStopsAtOne(t *testing.T) {
	next, ok := NextPatchDown("1.20.1")
	assert.False(t, ok)
	assert.Equal(t, "1.20.1", next)
}

func TestNextPatchDownStopsWithoutPatch(t *testing.T) {
	next, ok := NextPatchDown("1.20")
	assert.False(t, ok)
	assert.Equal(t, "1.20", next)
}

func TestNextPatchDownHandlesSingleSegment(t *testing.T) {
	next, ok := NextPatchDown("1")
	assert.False(t, ok)
	assert.Equal(t, "1.0", next)
}

func TestNextPatchDownHandlesInvalidNumbers(t *testing.T) {
	next, ok := NextPatchDown("bad.version")
	assert.False(t, ok)
	assert.Equal(t, "0.0", next)
}
