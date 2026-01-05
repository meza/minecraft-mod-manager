package view

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ConfirmPrompt describes a simple yes/no prompt rendered to a writer.
type ConfirmPrompt struct {
	Prefix         string
	Question       string
	Options        []ConfirmPromptOption
	DefaultID      string
	ConfirmID      string
	InvalidMessage string
}

// ConfirmPromptOption defines one option for a confirmation prompt.
type ConfirmPromptOption struct {
	ID    string
	Label string
	Short string
}

// RunConfirmPrompt renders a confirmation prompt, reads responses, and returns true for the confirm option.
// Empty responses select the default. Invalid responses are reported and re-prompted.
func RunConfirmPrompt(in io.Reader, out io.Writer, prompt ConfirmPrompt) (bool, error) {
	if out == nil {
		return false, errors.New("output writer is nil")
	}
	defaultOption, confirmOption, err := validateConfirmPrompt(prompt)
	if err != nil {
		return false, err
	}

	reader := bufio.NewReader(in)
	for {
		if _, err := io.WriteString(out, buildConfirmPromptLine(prompt)); err != nil {
			return false, err
		}

		response, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}

		response = strings.TrimSpace(response)
		if response == "" {
			return defaultOption.ID == confirmOption.ID, nil
		}

		if option, ok := matchConfirmOption(response, prompt.Options); ok {
			return option.ID == confirmOption.ID, nil
		}

		if strings.TrimSpace(prompt.InvalidMessage) != "" {
			if _, err := fmt.Fprintln(out, prompt.InvalidMessage); err != nil {
				return false, err
			}
		}
	}
}

func buildConfirmPromptLine(prompt ConfirmPrompt) string {
	prefix := strings.TrimSpace(prompt.Prefix)
	question := strings.TrimSpace(prompt.Question)
	if prefix != "" {
		prefix = prefix + " "
	}
	line := fmt.Sprintf("%s%s", prefix, question)
	suffix := buildConfirmPromptSuffix(prompt.Options, prompt.DefaultID)
	if suffix == "" {
		return line + " "
	}
	return fmt.Sprintf("%s %s ", line, suffix)
}

func buildConfirmPromptSuffix(options []ConfirmPromptOption, defaultID string) string {
	if len(options) == 0 {
		return ""
	}
	shorts := make([]string, 0, len(options))
	defaultShort := ""
	for _, option := range options {
		shorts = append(shorts, option.Short)
		if option.ID == defaultID {
			defaultShort = option.Short
		}
	}
	suffix := fmt.Sprintf("(%s)", strings.Join(shorts, "/"))
	if strings.TrimSpace(defaultShort) != "" {
		return fmt.Sprintf("%s [default: %s]:", suffix, defaultShort)
	}
	return suffix + ":"
}

func validateConfirmPrompt(prompt ConfirmPrompt) (defaultOption ConfirmPromptOption, confirmOption ConfirmPromptOption, err error) {
	if len(prompt.Options) == 0 {
		return ConfirmPromptOption{}, ConfirmPromptOption{}, errors.New("prompt options are required")
	}
	if strings.TrimSpace(prompt.DefaultID) == "" {
		return ConfirmPromptOption{}, ConfirmPromptOption{}, errors.New("default option id is required")
	}
	if strings.TrimSpace(prompt.ConfirmID) == "" {
		return ConfirmPromptOption{}, ConfirmPromptOption{}, errors.New("confirm option id is required")
	}

	seenIDs := map[string]bool{}
	seenShorts := map[string]bool{}
	for _, option := range prompt.Options {
		if strings.TrimSpace(option.Label) == "" {
			return ConfirmPromptOption{}, ConfirmPromptOption{}, errors.New("prompt option label is required")
		}
		if strings.TrimSpace(option.Short) == "" {
			return ConfirmPromptOption{}, ConfirmPromptOption{}, errors.New("prompt option short is required")
		}
		if seenIDs[option.ID] {
			return ConfirmPromptOption{}, ConfirmPromptOption{}, errors.New("prompt option ids must be unique")
		}
		shortKey := strings.ToLower(option.Short)
		if seenShorts[shortKey] {
			return ConfirmPromptOption{}, ConfirmPromptOption{}, errors.New("prompt option shorts must be unique")
		}
		seenIDs[option.ID] = true
		seenShorts[shortKey] = true
	}

	defaultOption, ok := findConfirmOption(prompt.DefaultID, prompt.Options)
	if !ok {
		return ConfirmPromptOption{}, ConfirmPromptOption{}, errors.New("default option id not found")
	}
	confirmOption, ok = findConfirmOption(prompt.ConfirmID, prompt.Options)
	if !ok {
		return ConfirmPromptOption{}, ConfirmPromptOption{}, errors.New("confirm option id not found")
	}
	return defaultOption, confirmOption, nil
}

func matchConfirmOption(response string, options []ConfirmPromptOption) (ConfirmPromptOption, bool) {
	for _, option := range options {
		if strings.EqualFold(response, option.Short) || strings.EqualFold(response, option.Label) {
			return option, true
		}
	}
	return ConfirmPromptOption{}, false
}

func findConfirmOption(id string, options []ConfirmPromptOption) (ConfirmPromptOption, bool) {
	for _, option := range options {
		if option.ID == id {
			return option, true
		}
	}
	return ConfirmPromptOption{}, false
}
