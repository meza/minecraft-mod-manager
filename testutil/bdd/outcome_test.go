package bdd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewOutcome(t *testing.T) {
	outcome := NewOutcome("placeholder", "details")

	require.Equal(t, "placeholder", outcome.Name)
	require.Equal(t, "details", outcome.Details)
}
