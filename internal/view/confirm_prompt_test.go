package view

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

type errorAfterFirstWrite struct {
	count int
}

func (writer *errorAfterFirstWrite) Write(p []byte) (int, error) {
	writer.count++
	if writer.count > 1 {
		return 0, errors.New("write failed")
	}
	return len(p), nil
}

func TestRunConfirmPromptReturnsTrueForYesShort(t *testing.T) {
	var out bytes.Buffer
	confirmed, err := RunConfirmPrompt(stringsReader("y\n"), &out, yesNoPrompt("Proceed?"))

	assert.NoError(t, err)
	assert.True(t, confirmed)
	assert.Equal(t, "Proceed? (y/n) [default: n]: ", out.String())
}

func TestRunConfirmPromptReturnsTrueForYesLabel(t *testing.T) {
	var out bytes.Buffer
	confirmed, err := RunConfirmPrompt(stringsReader("Yes\n"), &out, yesNoPrompt("Proceed?"))

	assert.NoError(t, err)
	assert.True(t, confirmed)
}

func TestRunConfirmPromptReturnsFalseForNoWithPrefix(t *testing.T) {
	var out bytes.Buffer
	prompt := yesNoPrompt("Delete files?")
	prompt.Prefix = "?"

	confirmed, err := RunConfirmPrompt(stringsReader("n\n"), &out, prompt)

	assert.NoError(t, err)
	assert.False(t, confirmed)
	assert.Equal(t, "? Delete files? (y/n) [default: n]: ", out.String())
}

func TestRunConfirmPromptReturnsErrorOnWriteFailure(t *testing.T) {
	confirmed, err := RunConfirmPrompt(stringsReader("y\n"), errorWriter{}, yesNoPrompt("Proceed?"))

	assert.Error(t, err)
	assert.False(t, confirmed)
}

func TestRunConfirmPromptReturnsErrorOnInvalidMessageWriteFailure(t *testing.T) {
	writer := &errorAfterFirstWrite{}
	confirmed, err := RunConfirmPrompt(stringsReader("maybe\n"), writer, yesNoPrompt("Proceed?"))

	assert.Error(t, err)
	assert.False(t, confirmed)
}

func TestRunConfirmPromptReturnsErrorOnReadFailure(t *testing.T) {
	var out bytes.Buffer
	confirmed, err := RunConfirmPrompt(errorReader{}, &out, yesNoPrompt("Proceed?"))

	assert.Error(t, err)
	assert.False(t, confirmed)
}

func TestRunConfirmPromptReturnsErrorOnEOF(t *testing.T) {
	var out bytes.Buffer
	confirmed, err := RunConfirmPrompt(stringsReader(""), &out, yesNoPrompt("Proceed?"))

	assert.ErrorIs(t, err, io.EOF)
	assert.False(t, confirmed)
}

func TestRunConfirmPromptReturnsDefaultOnEmptyResponse(t *testing.T) {
	var out bytes.Buffer
	confirmed, err := RunConfirmPrompt(stringsReader("\n"), &out, yesNoPrompt("Proceed?"))

	assert.NoError(t, err)
	assert.False(t, confirmed)
}

func TestRunConfirmPromptRepromptsOnInvalidResponse(t *testing.T) {
	var out bytes.Buffer
	confirmed, err := RunConfirmPrompt(stringsReader("maybe\nn\n"), &out, yesNoPrompt("Proceed?"))

	assert.NoError(t, err)
	assert.False(t, confirmed)
	output := out.String()
	assert.Contains(t, output, "Invalid choice. Use y or n.")
	assert.Equal(t, 2, strings.Count(output, "Proceed? (y/n) [default: n]: "))
}

func TestRunConfirmPromptSkipsInvalidMessageWhenBlank(t *testing.T) {
	var out bytes.Buffer
	prompt := yesNoPrompt("Proceed?")
	prompt.InvalidMessage = ""

	confirmed, err := RunConfirmPrompt(stringsReader("maybe\nn\n"), &out, prompt)

	assert.NoError(t, err)
	assert.False(t, confirmed)
	output := out.String()
	assert.NotContains(t, output, "Invalid choice")
	assert.Equal(t, 2, strings.Count(output, "Proceed? (y/n) [default: n]: "))
}

func TestRunConfirmPromptReturnsErrorOnNilOutput(t *testing.T) {
	confirmed, err := RunConfirmPrompt(stringsReader("y\n"), nil, yesNoPrompt("Proceed?"))

	assert.Error(t, err)
	assert.False(t, confirmed)
}

func TestRunConfirmPromptValidationErrors(t *testing.T) {
	cases := []struct {
		name   string
		prompt ConfirmPrompt
	}{
		{
			name:   "missing options",
			prompt: ConfirmPrompt{Question: "Proceed?", DefaultID: "no", ConfirmID: "yes"},
		},
		{
			name: "missing default id",
			prompt: ConfirmPrompt{
				Question:  "Proceed?",
				Options:   []ConfirmPromptOption{{ID: "yes", Label: "yes", Short: "y"}},
				ConfirmID: "yes",
			},
		},
		{
			name: "duplicate shorts",
			prompt: ConfirmPrompt{
				Question:  "Proceed?",
				Options:   []ConfirmPromptOption{{ID: "yes", Label: "yes", Short: "y"}, {ID: "no", Label: "no", Short: "y"}},
				DefaultID: "no",
				ConfirmID: "yes",
			},
		},
		{
			name: "missing default option",
			prompt: ConfirmPrompt{
				Question:  "Proceed?",
				Options:   []ConfirmPromptOption{{ID: "yes", Label: "yes", Short: "y"}},
				DefaultID: "no",
				ConfirmID: "yes",
			},
		},
		{
			name: "missing confirm id",
			prompt: ConfirmPrompt{
				Question:  "Proceed?",
				Options:   []ConfirmPromptOption{{ID: "yes", Label: "yes", Short: "y"}},
				DefaultID: "yes",
			},
		},
		{
			name: "missing option label",
			prompt: ConfirmPrompt{
				Question:  "Proceed?",
				Options:   []ConfirmPromptOption{{ID: "yes", Short: "y"}},
				DefaultID: "yes",
				ConfirmID: "yes",
			},
		},
		{
			name: "missing option short",
			prompt: ConfirmPrompt{
				Question:  "Proceed?",
				Options:   []ConfirmPromptOption{{ID: "yes", Label: "yes"}},
				DefaultID: "yes",
				ConfirmID: "yes",
			},
		},
		{
			name: "duplicate ids",
			prompt: ConfirmPrompt{
				Question:  "Proceed?",
				Options:   []ConfirmPromptOption{{ID: "yes", Label: "yes", Short: "y"}, {ID: "yes", Label: "no", Short: "n"}},
				DefaultID: "yes",
				ConfirmID: "yes",
			},
		},
		{
			name: "missing confirm option",
			prompt: ConfirmPrompt{
				Question:  "Proceed?",
				Options:   []ConfirmPromptOption{{ID: "no", Label: "no", Short: "n"}},
				DefaultID: "no",
				ConfirmID: "yes",
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			confirmed, err := RunConfirmPrompt(stringsReader("y\n"), &bytes.Buffer{}, testCase.prompt)
			assert.Error(t, err)
			assert.False(t, confirmed)
		})
	}
}

func TestBuildConfirmPromptLineWithoutSuffix(t *testing.T) {
	line := buildConfirmPromptLine(ConfirmPrompt{Question: "Proceed?"})
	assert.Equal(t, "Proceed? ", line)
}

func TestBuildConfirmPromptSuffixWithoutDefault(t *testing.T) {
	options := []ConfirmPromptOption{
		{ID: "yes", Label: "yes", Short: "y"},
		{ID: "no", Label: "no", Short: "n"},
	}
	suffix := buildConfirmPromptSuffix(options, "missing")
	assert.Equal(t, "(y/n):", suffix)
}

func yesNoPrompt(question string) ConfirmPrompt {
	yes := ConfirmPromptOption{ID: "yes", Label: "yes", Short: "y"}
	no := ConfirmPromptOption{ID: "no", Label: "no", Short: "n"}
	return ConfirmPrompt{
		Question:       question,
		Options:        []ConfirmPromptOption{yes, no},
		DefaultID:      no.ID,
		ConfirmID:      yes.ID,
		InvalidMessage: "Invalid choice. Use y or n.",
	}
}

func stringsReader(value string) io.Reader {
	return bytes.NewBufferString(value)
}
