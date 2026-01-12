//go:build !windows

package pty

import (
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeReadErrorEIO(t *testing.T) {
	require.NoError(t, normalizeReadError(syscall.EIO))
}
