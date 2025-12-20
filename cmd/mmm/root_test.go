package mmm

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelpTemplateIncludesHelpURL(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cmd := Command()

	assert.Contains(t, cmd.HelpTemplate(), "REPL_HELP_URL")
}

func TestCommand_UsageTemplateUsesWrappedFlags(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cmd := Command()
	assert.Contains(t, cmd.UsageTemplate(), ".FlagUsagesWrapped")
}

func TestCommand_HelpHandlesUnknownTopic(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cmd := Command()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"help", "nope"})

	err := cmd.Execute()
	assert.NoError(t, err)
	assert.NotEmpty(t, stderr.String())
}

func TestCommand_HelpHandlesKnownTopic(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cmd := Command()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"help", "version"})

	err := cmd.Execute()
	assert.NoError(t, err)
	assert.NotEmpty(t, stdout.String())
}

func TestExecute_ReturnsNilOnHelp(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"mmm", "--help"}

	assert.NoError(t, Execute())
}
