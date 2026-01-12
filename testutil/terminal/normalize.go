package terminal

import (
	"regexp"
	"strings"
)

// NormalizeOptions controls terminal output normalization.
type NormalizeOptions struct {
	StripControlSequences  bool
	TrimTrailingWhitespace bool
	TrimTrailingEmptyLines bool
	TrimSpace              bool
	RowLimit               int
	PadRows                bool
}

// NormalizeOutput normalizes terminal output for snapshot testing.
func NormalizeOutput(value string, options NormalizeOptions) string {
	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	if options.StripControlSequences {
		normalized = StripControlSequences(normalized)
	}

	lines := strings.Split(normalized, "\n")
	if options.TrimTrailingWhitespace {
		for index, line := range lines {
			lines[index] = strings.TrimRight(line, " \t")
		}
	}
	if options.TrimTrailingEmptyLines {
		lines = trimTrailingEmptyLines(lines)
	}
	if options.RowLimit > 0 {
		if len(lines) > options.RowLimit {
			lines = lines[len(lines)-options.RowLimit:]
		} else if options.PadRows && len(lines) < options.RowLimit {
			padding := make([]string, options.RowLimit-len(lines))
			lines = append(lines, padding...)
		}
	}

	output := strings.Join(lines, "\n")
	if options.TrimSpace {
		output = strings.TrimSpace(output)
	}
	return output
}

// StripControlSequences removes OSC and CSI sequences from output.
func StripControlSequences(value string) string {
	value = stripOSCSequences(value)
	value = stripCSISequences(value)
	return value
}

var oscSequence = regexp.MustCompile(`\x1b\][^\x07]*(\x07|\x1b\\)`)

func stripOSCSequences(value string) string {
	return oscSequence.ReplaceAllString(value, "")
}

var csiSequence = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

func stripCSISequences(value string) string {
	return csiSequence.ReplaceAllString(value, "")
}

func trimTrailingEmptyLines(lines []string) []string {
	for len(lines) > 0 {
		if strings.TrimSpace(lines[len(lines)-1]) != "" {
			return lines
		}
		lines = lines[:len(lines)-1]
	}
	return lines
}
