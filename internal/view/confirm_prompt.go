package view

import (
	"bufio"
	"io"
	"strings"
)

// ConfirmPrompt defines the prompt text and optional prefix/default hint.
type ConfirmPrompt struct {
	Prefix      string
	Question    string
	DefaultHint string
}

// RunConfirmPrompt writes the prompt, reads a single line, and returns true for yes answers.
// It treats "y" and "yes" (case-insensitive, trimmed) as confirmation.
func RunConfirmPrompt(in io.Reader, out io.Writer, prompt ConfirmPrompt) (bool, error) {
	promptText := prompt.Question
	if prompt.Prefix != "" {
		promptText = prompt.Prefix + " " + prompt.Question
	}

	suffix := " "
	if prompt.DefaultHint != "" {
		suffix = " (" + prompt.DefaultHint + "): "
	}

	if _, err := writeString(out, promptText+suffix); err != nil {
		return false, err
	}

	answer, err := readLine(in)
	if err != nil {
		return false, err
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes", nil
}

func readLine(reader io.Reader) (string, error) {
	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return scanner.Text(), nil
}

func writeString(writer io.Writer, value string) (int, error) {
	return writer.Write([]byte(value))
}
