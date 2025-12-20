//go:build windows

package modfilename

import (
	"path"
	"path/filepath"
	"strings"
)

func Normalize(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", Error{Value: trimmed, Reason: ReasonEmpty}
	}
	if hasUNCPath(trimmed) {
		return "", Error{Value: trimmed, Reason: ReasonUNCPath}
	}
	if hasDriveLetter(trimmed) {
		return "", Error{Value: trimmed, Reason: ReasonDriveLetter}
	}
	if filepath.Base(trimmed) != trimmed || path.Base(trimmed) != trimmed {
		return "", Error{Value: trimmed, Reason: ReasonSeparator}
	}
	if !strings.EqualFold(filepath.Ext(trimmed), ".jar") {
		return "", Error{Value: trimmed, Reason: ReasonExtension}
	}
	return trimmed, nil
}
