package init

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestSelectedReleaseTypeIconUsesUnicodeWhenAvailable(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	assert.Equal(t, "\u2713", selectedReleaseTypeIcon())
}

func TestSelectedReleaseTypeIconFallsBackToAsciiWhenUnicodeUnsupported(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	assert.Equal(t, "*", selectedReleaseTypeIcon())
}
