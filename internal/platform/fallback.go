package platform

import (
	"strconv"
	"strings"
)

func nextVersionDown(version string) (string, bool) {
	parts := splitVersion(version)
	if parts.patch > 1 {
		return parts.format(parts.patch - 1), true
	}

	return parts.format(parts.patch), false
}

type versionParts struct {
	major int
	minor int
	patch int
}

func (parts versionParts) format(patch int) string {
	if patch == 0 {
		return strconv.Itoa(parts.major) + "." + strconv.Itoa(parts.minor)
	}
	return strconv.Itoa(parts.major) + "." + strconv.Itoa(parts.minor) + "." + strconv.Itoa(patch)
}

func splitVersion(version string) versionParts {
	var parts versionParts
	segments := strings.Split(version, ".")

	if len(segments) > 0 {
		parts.major = parseInt(segments[0])
	}
	if len(segments) > 1 {
		parts.minor = parseInt(segments[1])
	}
	if len(segments) > 2 {
		parts.patch = parseInt(segments[2])
	}

	return parts
}

func parseInt(value string) int {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return n
}
