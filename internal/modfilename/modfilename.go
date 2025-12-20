package modfilename

import "strings"

type ErrorReason string

const (
	ReasonEmpty       ErrorReason = "empty"
	ReasonDriveLetter ErrorReason = "drive_letter"
	ReasonUNCPath     ErrorReason = "unc_path"
	ReasonSeparator   ErrorReason = "path_separator"
	ReasonExtension   ErrorReason = "extension"
)

type Error struct {
	Value  string
	Reason ErrorReason
}

func (err Error) Error() string {
	if err.Value == "" {
		return "invalid mod filename: " + string(err.Reason)
	}
	return "invalid mod filename " + err.Value + ": " + string(err.Reason)
}

func Display(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "(empty)"
	}
	return trimmed
}

func hasUNCPath(value string) bool {
	return strings.HasPrefix(value, `\\`) || strings.HasPrefix(value, "//")
}

func hasDriveLetter(value string) bool {
	if len(value) < 2 {
		return false
	}
	first := value[0]
	if !isASCIIAlpha(first) {
		return false
	}
	return value[1] == ':'
}

func isASCIIAlpha(value byte) bool {
	return (value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z')
}
