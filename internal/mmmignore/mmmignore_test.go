package mmmignore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestListPatternsAlwaysIncludesDisabled(t *testing.T) {
	fs := afero.NewMemMapFs()
	rootDir := filepath.FromSlash("/cfg")
	assert.NoError(t, fs.MkdirAll(rootDir, 0755))

	patterns, err := ListPatterns(fs, rootDir)
	assert.NoError(t, err)
	assert.Equal(t, []string{disabledPattern}, patterns)
}

func TestListPatternsReadsAndTrimsIgnoreFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	rootDir := filepath.FromSlash("/cfg")
	assert.NoError(t, fs.MkdirAll(rootDir, 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(rootDir, ".mmmignore"), []byte("\n mods/*.jar \n\n"), 0644))

	patterns, err := ListPatterns(fs, rootDir)
	assert.NoError(t, err)
	assert.Equal(t, []string{disabledPattern, "mods/*.jar"}, patterns)
}

func TestIgnoredFilesBuildsAbsolutePathSet(t *testing.T) {
	fs := afero.NewMemMapFs()
	ignoreDir := filepath.FromSlash("/cfg")
	matchRoot := filepath.FromSlash("/data/mods")
	assert.NoError(t, fs.MkdirAll(ignoreDir, 0755))
	assert.NoError(t, fs.MkdirAll(matchRoot, 0755))

	ignored := filepath.Join(matchRoot, "ignored.jar")
	kept := filepath.Join(matchRoot, "kept.jar")
	assert.NoError(t, afero.WriteFile(fs, ignored, []byte("x"), 0644))
	assert.NoError(t, afero.WriteFile(fs, kept, []byte("x"), 0644))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(ignoreDir, ".mmmignore"), []byte("ignored.jar\n"), 0644))

	set, err := IgnoredFiles(fs, ignoreDir, matchRoot)
	assert.NoError(t, err)
	assert.True(t, set[ignored])
	assert.False(t, set[kept])
}

func TestGlobMatchSupportsDoubleStar(t *testing.T) {
	assert.True(t, globMatch("**/*.disabled", "mods/a.jar.disabled"))
	assert.False(t, globMatch("**/*.disabled", "mods/a.jar"))
}

func TestIsIgnored_MatchesRelativePaths(t *testing.T) {
	matchRoot := filepath.FromSlash("/mods")
	target := filepath.Join(matchRoot, "ignored.jar")
	assert.True(t, IsIgnored(matchRoot, target, []string{"*.jar"}))
	assert.False(t, IsIgnored(matchRoot, target, []string{"*.zip"}))
}

func TestIsIgnored_DoesNotMatchPathsOutsideRoot(t *testing.T) {
	matchRoot := filepath.FromSlash("/mods")
	target := filepath.FromSlash("/external/mods/ignored.jar")
	assert.False(t, IsIgnored(matchRoot, target, []string{"*.jar"}))
	assert.False(t, IsIgnored(matchRoot, target, []string{"**/*"}))
}

func TestPathRelativeToRoot_ReturnsRelativePath(t *testing.T) {
	matchRoot := filepath.FromSlash("/mods")
	target := filepath.Join(matchRoot, "nested", "ignored.jar")

	rel, ok := pathRelativeToRoot(matchRoot, target)
	assert.True(t, ok)
	assert.Equal(t, filepath.Join("nested", "ignored.jar"), rel)
}

func TestPathRelativeToRoot_AllowsRootItself(t *testing.T) {
	matchRoot := filepath.FromSlash("/mods")

	rel, ok := pathRelativeToRoot(matchRoot, matchRoot)
	assert.True(t, ok)
	assert.Equal(t, "", rel)
}

func TestPathRelativeToRoot_RejectsOutsideRoot(t *testing.T) {
	matchRoot := filepath.FromSlash("/mods")
	target := filepath.FromSlash("/other/mods/ignored.jar")

	rel, ok := pathRelativeToRoot(matchRoot, target)
	assert.False(t, ok)
	assert.Equal(t, "", rel)
}

func TestPathRelativeToRoot_RejectsParentDirectory(t *testing.T) {
	matchRoot := filepath.FromSlash("/mods/child")
	target := filepath.FromSlash("/mods")

	rel, ok := pathRelativeToRoot(matchRoot, target)
	assert.False(t, ok)
	assert.Equal(t, "", rel)
}

func TestPathRelativeToRoot_ReturnsFalseOnAbsError(t *testing.T) {
	originalAbsPath := absPath
	absPath = func(string) (string, error) {
		return "", errors.New("abs failed")
	}
	t.Cleanup(func() {
		absPath = originalAbsPath
	})

	rel, ok := pathRelativeToRoot(filepath.FromSlash("/mods"), filepath.FromSlash("/mods/ignored.jar"))
	assert.False(t, ok)
	assert.Equal(t, "", rel)
}

func TestPathRelativeToRoot_ReturnsFalseOnPathAbsError(t *testing.T) {
	originalAbsPath := absPath
	callCount := 0
	absPath = func(path string) (string, error) {
		callCount++
		if callCount == 1 {
			return path, nil
		}
		return "", errors.New("abs failed")
	}
	t.Cleanup(func() {
		absPath = originalAbsPath
	})

	rel, ok := pathRelativeToRoot(filepath.FromSlash("/mods"), filepath.FromSlash("/mods/ignored.jar"))
	assert.False(t, ok)
	assert.Equal(t, "", rel)
}

func TestPathRelativeToRoot_ReturnsFalseOnRelError(t *testing.T) {
	originalRelPath := relPath
	relPath = func(string, string) (string, error) {
		return "", errors.New("rel failed")
	}
	t.Cleanup(func() {
		relPath = originalRelPath
	})

	rel, ok := pathRelativeToRoot(filepath.FromSlash("/mods"), filepath.FromSlash("/mods/ignored.jar"))
	assert.False(t, ok)
	assert.Equal(t, "", rel)
}

func TestListPatterns_ReturnsErrorWhenExistsFails(t *testing.T) {
	fs := statErrorFs{Fs: afero.NewMemMapFs(), err: errors.New("stat failed")}
	_, err := ListPatterns(fs, filepath.FromSlash("/cfg"))
	assert.Error(t, err)
}

func TestListPatterns_ReturnsErrorWhenIgnoreFileUnreadable(t *testing.T) {
	fs := afero.NewMemMapFs()
	rootDir := filepath.FromSlash("/cfg")
	assert.NoError(t, fs.MkdirAll(rootDir, 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(rootDir, ".mmmignore"), []byte("mods/*.jar\n"), 0644))

	_, err := ListPatterns(openErrorFs{Fs: fs, failPath: filepath.Join(rootDir, ".mmmignore")}, rootDir)
	assert.Error(t, err)
}

func TestIgnoredFiles_ReturnsErrorWhenRootDirMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	ignoreDir := filepath.FromSlash("/cfg")
	assert.NoError(t, fs.MkdirAll(ignoreDir, 0755))

	_, err := IgnoredFiles(fs, ignoreDir, filepath.FromSlash("/missing"))
	assert.Error(t, err)
}

func TestIgnoredFiles_ReturnsErrorWhenListPatternsFails(t *testing.T) {
	fs := statErrorFs{Fs: afero.NewMemMapFs(), err: errors.New("stat failed")}
	_, err := IgnoredFiles(fs, filepath.FromSlash("/cfg"), filepath.FromSlash("/mods"))
	assert.Error(t, err)
}

func TestBuildIgnoredSet_IgnoresEmptyPatterns(t *testing.T) {
	fs := afero.NewMemMapFs()
	matchRoot := filepath.FromSlash("/mods")
	assert.NoError(t, fs.MkdirAll(matchRoot, 0755))
	target := filepath.Join(matchRoot, "ignored.jar")
	assert.NoError(t, afero.WriteFile(fs, target, []byte("x"), 0644))

	set, err := buildIgnoredSet(fs, matchRoot, []string{"", "ignored.jar"})
	assert.NoError(t, err)
	assert.True(t, set[target])
}

func TestBuildIgnoredSet_StopsOnWalkError(t *testing.T) {
	fs := afero.NewMemMapFs()
	matchRoot := filepath.FromSlash("/mods")
	assert.NoError(t, fs.MkdirAll(matchRoot, 0755))
	badPath := filepath.Join(matchRoot, "bad.jar")
	assert.NoError(t, afero.WriteFile(fs, badPath, []byte("x"), 0644))

	_, err := buildIgnoredSet(walkStatErrorFs{Fs: fs, failPath: badPath}, matchRoot, []string{"*.jar"})
	assert.Error(t, err)
}

type statErrorFs struct {
	afero.Fs
	err error
}

func (filesystem statErrorFs) Stat(name string) (os.FileInfo, error) { return nil, filesystem.err }

type openErrorFs struct {
	afero.Fs
	failPath string
}

func (filesystem openErrorFs) Open(name string) (afero.File, error) {
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return nil, errors.New("open failed")
	}
	return filesystem.Fs.Open(name)
}

type walkStatErrorFs struct {
	afero.Fs
	failPath string
}

func (filesystem walkStatErrorFs) Stat(name string) (os.FileInfo, error) {
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return nil, errors.New("stat failed")
	}
	return filesystem.Fs.Stat(name)
}
