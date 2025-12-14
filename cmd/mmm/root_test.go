package mmm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelpTemplateIncludesHelpURL(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cmd := Command()

	assert.Contains(t, cmd.HelpTemplate(), "REPL_HELP_URL")
}
