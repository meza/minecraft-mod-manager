package test

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestOutputProgramOptionsUsesCommandOutputWhenWriterNil(t *testing.T) {
	command := &cobra.Command{}
	output := &bytes.Buffer{}
	command.SetOut(output)

	options := outputProgramOptions(command, nil)
	assert.Len(t, options, 3)
}
