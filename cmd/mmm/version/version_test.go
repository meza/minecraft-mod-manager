package version

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

type mockCommand struct {
	mockPrintln func(args ...interface{})
}

func (command *mockCommand) Println(args ...interface{}) {
	command.mockPrintln(args)
}

func TestVersion(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	command := Command()

	assert.Equal(t, "version", command.Use)
	assert.Equal(t, "cmd.version.short, Arg 1: {Count: 0, Data: &map[appName:minecraft-mod-manager]}", command.Short)
}

func TestVersionOutput(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	b := bytes.NewBufferString("")
	command := Command()
	command.SetOut(b)
	err := command.Execute()
	assert.NoError(t, err)

	out, err := io.ReadAll(b)
	assert.NoError(t, err)

	assert.Equal(t, "REPL_VERSION\n", string(out))

}
