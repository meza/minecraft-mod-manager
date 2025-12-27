package privacy

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const redactedUsername = "<user>"

var (
	usernamesMu     sync.Mutex
	usernamesLoaded bool
	cachedUsernames []string
	loadUsernames   = defaultKnownUsernames
)

// KnownUsernames returns a best-effort list of local usernames for redaction.
func KnownUsernames() []string {
	usernamesMu.Lock()
	defer usernamesMu.Unlock()

	if !usernamesLoaded {
		cachedUsernames = loadUsernames()
		usernamesLoaded = true
	}

	cloned := make([]string, len(cachedUsernames))
	copy(cloned, cachedUsernames)
	return cloned
}

// ResetForTesting clears cached usernames to allow deterministic tests.
func ResetForTesting() {
	usernamesMu.Lock()
	defer usernamesMu.Unlock()
	usernamesLoaded = false
	cachedUsernames = nil
}

// RedactPathUsernames replaces known usernames in path segments with "<user>".
func RedactPathUsernames(value string) string {
	if strings.TrimSpace(value) == "" {
		return value
	}
	matcher := newUsernameMatcher(KnownUsernames(), usernameMatchOptions{caseInsensitive: runtime.GOOS == "windows"})
	return redactPathSegments(value, matcher)
}

type usernameMatcher struct {
	names           map[string]struct{}
	caseInsensitive bool
}

type usernameMatchOptions struct {
	caseInsensitive bool
}

func newUsernameMatcher(usernames []string, options usernameMatchOptions) usernameMatcher {
	unique := make(map[string]struct{}, len(usernames))
	for _, name := range usernames {
		normalized := normalizeUsername(name)
		if normalized == "" || normalized == redactedUsername {
			continue
		}
		if options.caseInsensitive {
			normalized = strings.ToLower(normalized)
		}
		unique[normalized] = struct{}{}
	}
	return usernameMatcher{names: unique, caseInsensitive: options.caseInsensitive}
}

func (matcher usernameMatcher) matches(segment string) bool {
	if segment == "" {
		return false
	}
	if matcher.caseInsensitive {
		segment = strings.ToLower(segment)
	}
	_, ok := matcher.names[segment]
	return ok
}

func redactPathSegments(value string, matcher usernameMatcher) string {
	var output strings.Builder
	output.Grow(len(value))

	var segment strings.Builder
	flush := func(separator string) {
		segmentText := segment.String()
		if matcher.matches(segmentText) {
			writeString(&output, redactedUsername)
		} else {
			writeString(&output, segmentText)
		}
		writeString(&output, separator)
		segment.Reset()
	}

	for _, runeValue := range value {
		if runeValue == '/' || runeValue == '\\' {
			flush(string(runeValue))
			continue
		}
		writeRune(&segment, runeValue)
	}

	if segment.Len() > 0 {
		segmentText := segment.String()
		if matcher.matches(segmentText) {
			writeString(&output, redactedUsername)
		} else {
			writeString(&output, segmentText)
		}
	}

	return output.String()
}

func writeString(builder *strings.Builder, value string) {
	_, _ = builder.WriteString(value)
}

func writeRune(builder *strings.Builder, value rune) {
	_, _ = builder.WriteRune(value)
}

func defaultKnownUsernames() []string {
	return defaultKnownUsernamesWithSources(runtime.GOOS, os.Getenv, user.Current, os.ReadDir, defaultHomeRoots)
}

func defaultKnownUsernamesWithSources(
	goos string,
	envLookup func(string) string,
	currentUser func() (*user.User, error),
	readDir func(string) ([]os.DirEntry, error),
	homeRoots func(string) []string,
) []string {
	unique := map[string]struct{}{}
	names := []string{}

	add := func(value string) {
		normalized := normalizeUsername(value)
		if normalized == "" || normalized == redactedUsername {
			return
		}
		if _, exists := unique[normalized]; exists {
			return
		}
		unique[normalized] = struct{}{}
		names = append(names, normalized)
	}

	for _, envVar := range []string{"USER", "USERNAME", "LOGNAME"} {
		add(strings.TrimSpace(envLookup(envVar)))
	}

	current, err := currentUser()
	if err == nil {
		add(current.Username)
	}

	for _, root := range homeRoots(goos) {
		entries, err := readDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				add(entry.Name())
			}
		}
	}

	return names
}

func defaultHomeRoots(goos string) []string {
	switch goos {
	case "windows":
		drive := strings.TrimSpace(os.Getenv("SystemDrive"))
		if drive == "" {
			drive = "C:"
		}
		return []string{filepath.Join(drive, "Users")}
	case "darwin":
		return []string{"/Users"}
	case "linux":
		return []string{"/home"}
	default:
		return nil
	}
}

func normalizeUsername(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if idx := strings.LastIndexAny(trimmed, "\\/"); idx >= 0 && idx+1 < len(trimmed) {
		trimmed = trimmed[idx+1:]
	}
	return strings.TrimSpace(trimmed)
}
