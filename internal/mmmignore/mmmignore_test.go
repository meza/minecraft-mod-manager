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
	rootDir := filepath.FromSlash("/cfg")
	assert.NoError(t, fs.MkdirAll(filepath.Join(rootDir, "mods"), 0755))

	ignored := filepath.Join(rootDir, "mods", "ignored.jar")
	kept := filepath.Join(rootDir, "mods", "kept.jar")
	assert.NoError(t, afero.WriteFile(fs, ignored, []byte("x"), 0644))
	assert.NoError(t, afero.WriteFile(fs, kept, []byte("x"), 0644))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(rootDir, ".mmmignore"), []byte("mods/ignored.jar\n"), 0644))

	set, err := IgnoredFiles(fs, rootDir)
	assert.NoError(t, err)
	assert.True(t, set[ignored])
	assert.False(t, set[kept])
}

func TestGlobMatchSupportsDoubleStar(t *testing.T) {
	assert.True(t, globMatch("**/*.disabled", "mods/a.jar.disabled"))
	assert.False(t, globMatch("**/*.disabled", "mods/a.jar"))
}

func TestIsIgnored_MatchesRelativePaths(t *testing.T) {
	rootDir := filepath.FromSlash("/cfg")
	target := filepath.Join(rootDir, "mods", "ignored.jar")
	assert.True(t, IsIgnored(rootDir, target, []string{"mods/*.jar"}))
	assert.False(t, IsIgnored(rootDir, target, []string{"mods/*.zip"}))
}

func TestIsIgnored_DoesNotMatchPathsOutsideRoot(t *testing.T) {
	rootDir := filepath.FromSlash("/cfg")
	target := filepath.FromSlash("/external/mods/ignored.jar")
	assert.False(t, IsIgnored(rootDir, target, []string{"mods/*.jar"}))
	assert.False(t, IsIgnored(rootDir, target, []string{"**/*"}))
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
	_, err := IgnoredFiles(fs, filepath.FromSlash("/missing"))
	assert.Error(t, err)
}

func TestIgnoredFiles_ReturnsErrorWhenListPatternsFails(t *testing.T) {
	fs := statErrorFs{Fs: afero.NewMemMapFs(), err: errors.New("stat failed")}
	_, err := IgnoredFiles(fs, filepath.FromSlash("/cfg"))
	assert.Error(t, err)
}

func TestBuildIgnoredSet_IgnoresEmptyPatterns(t *testing.T) {
	fs := afero.NewMemMapFs()
	rootDir := filepath.FromSlash("/cfg")
	assert.NoError(t, fs.MkdirAll(filepath.Join(rootDir, "mods"), 0755))
	target := filepath.Join(rootDir, "mods", "ignored.jar")
	assert.NoError(t, afero.WriteFile(fs, target, []byte("x"), 0644))

	set, err := buildIgnoredSet(fs, rootDir, []string{"", "mods/ignored.jar"})
	assert.NoError(t, err)
	assert.True(t, set[target])
}

func TestBuildIgnoredSet_StopsOnWalkError(t *testing.T) {
	fs := afero.NewMemMapFs()
	rootDir := filepath.FromSlash("/cfg")
	assert.NoError(t, fs.MkdirAll(rootDir, 0755))
	badPath := filepath.Join(rootDir, "bad.jar")
	assert.NoError(t, afero.WriteFile(fs, badPath, []byte("x"), 0644))

	_, err := buildIgnoredSet(walkStatErrorFs{Fs: fs, failPath: badPath}, rootDir, []string{"*.jar"})
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
