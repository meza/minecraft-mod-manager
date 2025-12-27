package privacy

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeUsernameStripsDomain(t *testing.T) {
	assert.Equal(t, "alice", normalizeUsername("DOMAIN\\alice"))
	assert.Equal(t, "alice", normalizeUsername("domain/alice"))
	assert.Equal(t, "alice", normalizeUsername(" alice "))
}

func TestRedactPathSegmentsReplacesUsername(t *testing.T) {
	matcher := newUsernameMatcher([]string{"alice"}, usernameMatchOptions{})
	input := "/Users/alice/mods/a.jar"
	assert.Equal(t, "/Users/<user>/mods/a.jar", redactPathSegments(input, matcher))
}

func TestRedactPathSegmentsPreservesWindowsSeparators(t *testing.T) {
	matcher := newUsernameMatcher([]string{"alice"}, usernameMatchOptions{})
	input := `C:\Users\alice\mods`
	assert.Equal(t, `C:\Users\<user>\mods`, redactPathSegments(input, matcher))
}

func TestRedactPathSegmentsMatchesCaseInsensitiveWhenConfigured(t *testing.T) {
	matcher := newUsernameMatcher([]string{"ALICE"}, usernameMatchOptions{caseInsensitive: true})
	input := `C:\Users\alice\mods`
	assert.Equal(t, `C:\Users\<user>\mods`, redactPathSegments(input, matcher))
}

func TestRedactPathSegmentsLeavesUnmatchedSegments(t *testing.T) {
	matcher := newUsernameMatcher([]string{"alice"}, usernameMatchOptions{})
	input := "/Users/bob/mods/a.jar"
	assert.Equal(t, input, redactPathSegments(input, matcher))
}

func TestRedactPathSegmentsHandlesTrailingSeparator(t *testing.T) {
	matcher := newUsernameMatcher([]string{"alice"}, usernameMatchOptions{})
	input := "/Users/alice/"
	assert.Equal(t, "/Users/<user>/", redactPathSegments(input, matcher))
}

func TestRedactPathSegmentsRedactsFinalSegment(t *testing.T) {
	matcher := newUsernameMatcher([]string{"alice"}, usernameMatchOptions{})
	input := "/Users/alice"
	assert.Equal(t, "/Users/<user>", redactPathSegments(input, matcher))
}

func TestKnownUsernamesCachesAndResets(t *testing.T) {
	ResetForTesting()
	originalLoader := loadUsernames
	defer func() { loadUsernames = originalLoader }()

	loadUsernames = func() []string { return []string{"alice"} }
	assert.Equal(t, []string{"alice"}, KnownUsernames())

	loadUsernames = func() []string { return []string{"bob"} }
	assert.Equal(t, []string{"alice"}, KnownUsernames())

	ResetForTesting()
	assert.Equal(t, []string{"bob"}, KnownUsernames())
}

func TestRedactPathUsernamesUsesKnownUsernames(t *testing.T) {
	ResetForTesting()
	originalLoader := loadUsernames
	defer func() { loadUsernames = originalLoader }()

	loadUsernames = func() []string { return []string{"alice"} }

	var input string
	var expected string
	switch runtime.GOOS {
	case "windows":
		input = `C:\Users\alice\mods`
		expected = `C:\Users\<user>\mods`
	case "darwin":
		input = "/Users/alice/mods"
		expected = "/Users/<user>/mods"
	default:
		input = "/home/alice/mods"
		expected = "/home/<user>/mods"
	}

	assert.Equal(t, expected, RedactPathUsernames(input))
}

func TestRedactPathUsernamesReturnsBlankInput(t *testing.T) {
	assert.Equal(t, "", RedactPathUsernames(""))
	assert.Equal(t, "   ", RedactPathUsernames("   "))
}

func TestNewUsernameMatcherIgnoresEmptyAndRedacted(t *testing.T) {
	matcher := newUsernameMatcher([]string{"", " <user> ", "alice"}, usernameMatchOptions{})
	assert.Equal(t, 1, len(matcher.names))
	assert.False(t, matcher.matches("<user>"))
	assert.True(t, matcher.matches("alice"))
}

func TestDefaultKnownUsernamesWithSources(t *testing.T) {
	envLookup := func(key string) string {
		if key == "USER" {
			return "alice"
		}
		return ""
	}

	currentUser := func() (*user.User, error) {
		return &user.User{Username: "bob"}, nil
	}

	readDir := func(path string) ([]os.DirEntry, error) {
		if path == "/home" {
			return []os.DirEntry{
				fakeDirEntry{name: "charlie", isDir: true},
				fakeDirEntry{name: "not-a-dir", isDir: false},
			}, nil
		}
		return nil, errors.New("missing")
	}

	homeRoots := func(goos string) []string { return []string{"/home"} }

	names := defaultKnownUsernamesWithSources("linux", envLookup, currentUser, readDir, homeRoots)
	assert.ElementsMatch(t, []string{"alice", "bob", "charlie"}, names)
}

func TestDefaultKnownUsernamesWithSourcesHandlesErrors(t *testing.T) {
	envLookup := func(key string) string {
		if key == "USER" {
			return "alice"
		}
		return ""
	}

	currentUser := func() (*user.User, error) {
		return nil, errors.New("no user")
	}

	readDir := func(path string) ([]os.DirEntry, error) {
		return nil, errors.New("missing")
	}

	homeRoots := func(goos string) []string { return []string{"/home"} }

	names := defaultKnownUsernamesWithSources("linux", envLookup, currentUser, readDir, homeRoots)
	assert.ElementsMatch(t, []string{"alice"}, names)
}

func TestDefaultKnownUsernamesReturnsSlice(t *testing.T) {
	names := defaultKnownUsernames()
	assert.NotNil(t, names)
}

func TestDefaultHomeRoots(t *testing.T) {
	t.Setenv("SystemDrive", "D:")
	assert.Equal(t, []string{filepath.Join("D:", "Users")}, defaultHomeRoots("windows"))

	t.Setenv("SystemDrive", "")
	assert.Equal(t, []string{filepath.Join("C:", "Users")}, defaultHomeRoots("windows"))
	assert.Equal(t, []string{"/Users"}, defaultHomeRoots("darwin"))
	assert.Equal(t, []string{"/home"}, defaultHomeRoots("linux"))
	assert.Nil(t, defaultHomeRoots("plan9"))
}

type fakeDirEntry struct {
	name  string
	isDir bool
}

func (entry fakeDirEntry) Name() string {
	return entry.name
}

func (entry fakeDirEntry) IsDir() bool {
	return entry.isDir
}

func (entry fakeDirEntry) Type() os.FileMode {
	if entry.isDir {
		return os.ModeDir
	}
	return 0
}

func (entry fakeDirEntry) Info() (os.FileInfo, error) {
	return nil, errors.New("not implemented")
}
