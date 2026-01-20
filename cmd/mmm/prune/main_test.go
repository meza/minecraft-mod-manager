package prune

import (
	"os"
	"testing"

	"github.com/muesli/termenv"

	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestMain(m *testing.M) {
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	code := m.Run()
	restoreUnicode()
	restoreColor()
	os.Exit(code)
}
