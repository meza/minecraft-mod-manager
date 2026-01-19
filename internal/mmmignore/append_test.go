package mmmignore

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendPatternsCreatesFileWhenMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	rootDir := filepath.FromSlash("/cfg")
	require.NoError(t, fs.MkdirAll(rootDir, 0o755))

	err := AppendPatterns(fs, rootDir, []string{"alpha.jar"})
	require.NoError(t, err)

	data, readErr := afero.ReadFile(fs, filepath.Join(rootDir, ".mmmignore"))
	require.NoError(t, readErr)
	assert.Equal(t, "alpha.jar\n", string(data))
}

func TestAppendPatternsSkipsDuplicatesAndPreservesContent(t *testing.T) {
	fs := afero.NewMemMapFs()
	rootDir := filepath.FromSlash("/cfg")
	require.NoError(t, fs.MkdirAll(rootDir, 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(rootDir, ".mmmignore"), []byte("alpha.jar\n\n"), 0o644))

	err := AppendPatterns(fs, rootDir, []string{"alpha.jar", "beta.jar"})
	require.NoError(t, err)

	data, readErr := afero.ReadFile(fs, filepath.Join(rootDir, ".mmmignore"))
	require.NoError(t, readErr)
	assert.Equal(t, "alpha.jar\n\nbeta.jar\n", string(data))
}

func TestAppendPatternsNoopWhenOnlyDuplicates(t *testing.T) {
	fs := afero.NewMemMapFs()
	rootDir := filepath.FromSlash("/cfg")
	require.NoError(t, fs.MkdirAll(rootDir, 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(rootDir, ".mmmignore"), []byte("alpha.jar\n"), 0o644))

	err := AppendPatterns(fs, rootDir, []string{"alpha.jar"})
	require.NoError(t, err)

	data, readErr := afero.ReadFile(fs, filepath.Join(rootDir, ".mmmignore"))
	require.NoError(t, readErr)
	assert.Equal(t, "alpha.jar\n", string(data))
}

func TestAppendPatternsNoopOnEmptyInput(t *testing.T) {
	fs := afero.NewMemMapFs()
	rootDir := filepath.FromSlash("/cfg")
	require.NoError(t, fs.MkdirAll(rootDir, 0o755))

	err := AppendPatterns(fs, rootDir, []string{"", "  "})
	require.NoError(t, err)

	_, readErr := afero.ReadFile(fs, filepath.Join(rootDir, ".mmmignore"))
	assert.Error(t, readErr)
}

func TestAppendPatternsAddsMissingNewline(t *testing.T) {
	fs := afero.NewMemMapFs()
	rootDir := filepath.FromSlash("/cfg")
	require.NoError(t, fs.MkdirAll(rootDir, 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(rootDir, ".mmmignore"), []byte("alpha.jar"), 0o644))

	err := AppendPatterns(fs, rootDir, []string{"beta.jar"})
	require.NoError(t, err)

	data, readErr := afero.ReadFile(fs, filepath.Join(rootDir, ".mmmignore"))
	require.NoError(t, readErr)
	assert.Equal(t, "alpha.jar\nbeta.jar\n", string(data))
}

func TestBuildIgnoreFileContentsWithEmptyExistingData(t *testing.T) {
	contents, err := buildIgnoreFileContents("", []string{"alpha.jar"})
	require.NoError(t, err)
	assert.Equal(t, "alpha.jar\n", contents)
}

func TestBuildIgnoreFileContentsPreservesTrailingNewline(t *testing.T) {
	contents, err := buildIgnoreFileContents("alpha.jar\n", []string{"beta.jar"})
	require.NoError(t, err)
	assert.Equal(t, "alpha.jar\nbeta.jar\n", contents)
}

func TestBuildIgnoreFileContentsReturnsErrorOnExistingWriteFailure(t *testing.T) {
	restore := writeString
	t.Cleanup(func() { writeString = restore })
	writeString = func(*strings.Builder, string) error {
		return errors.New("write failed")
	}

	_, err := buildIgnoreFileContents("alpha.jar", []string{"beta.jar"})
	assert.Error(t, err)
}

func TestBuildIgnoreFileContentsReturnsErrorOnAppendFailure(t *testing.T) {
	restore := writeString
	t.Cleanup(func() { writeString = restore })
	callCount := 0
	writeString = func(builder *strings.Builder, value string) error {
		callCount++
		if callCount == 2 {
			return errors.New("write failed")
		}
		return restore(builder, value)
	}

	_, err := buildIgnoreFileContents("alpha.jar\n", []string{"beta.jar"})
	assert.Error(t, err)
}

func TestBuildIgnoreFileContentsReturnsErrorOnFinalNewline(t *testing.T) {
	restore := writeString
	t.Cleanup(func() { writeString = restore })
	callCount := 0
	writeString = func(builder *strings.Builder, value string) error {
		callCount++
		if callCount == 3 {
			return errors.New("write failed")
		}
		return restore(builder, value)
	}

	_, err := buildIgnoreFileContents("alpha.jar\n", []string{"beta.jar"})
	assert.Error(t, err)
}

func TestAppendExistingDataPreservesTrailingNewline(t *testing.T) {
	var builder strings.Builder
	err := appendExistingData(&builder, "alpha.jar\n")
	require.NoError(t, err)
	assert.Equal(t, "alpha.jar\n", builder.String())
}

func TestAppendExistingDataReturnsErrorOnWriteFailure(t *testing.T) {
	restore := writeString
	t.Cleanup(func() { writeString = restore })
	writeString = func(*strings.Builder, string) error {
		return errors.New("write failed")
	}

	var builder strings.Builder
	err := appendExistingData(&builder, "alpha.jar")
	assert.Error(t, err)
}

func TestAppendPatternsReturnsErrorWhenBuildFails(t *testing.T) {
	fs := afero.NewMemMapFs()
	rootDir := filepath.FromSlash("/cfg")
	require.NoError(t, fs.MkdirAll(rootDir, 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(rootDir, ".mmmignore"), []byte("alpha.jar\n"), 0o644))

	restore := writeString
	t.Cleanup(func() { writeString = restore })
	writeString = func(*strings.Builder, string) error {
		return errors.New("write failed")
	}

	err := AppendPatterns(fs, rootDir, []string{"beta.jar"})
	assert.Error(t, err)
}

func TestAppendPatternsReturnsErrorOnExistsFailure(t *testing.T) {
	expectedErr := errors.New("stat failed")
	fs := appendStatErrorFs{Fs: afero.NewMemMapFs(), err: expectedErr}
	err := AppendPatterns(fs, filepath.FromSlash("/cfg"), []string{"alpha.jar"})
	assert.ErrorIs(t, err, expectedErr)
}

func TestAppendPatternsReturnsErrorOnReadFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	rootDir := filepath.FromSlash("/cfg")
	require.NoError(t, fs.MkdirAll(rootDir, 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(rootDir, ".mmmignore"), []byte("alpha.jar\n"), 0o644))

	expectedErr := errors.New("read failed")
	failFs := appendOpenErrorFs{Fs: fs, failPath: filepath.Join(rootDir, ".mmmignore"), err: expectedErr}
	err := AppendPatterns(failFs, rootDir, []string{"beta.jar"})
	assert.ErrorIs(t, err, expectedErr)
}

type appendStatErrorFs struct {
	afero.Fs
	err error
}

func (fs appendStatErrorFs) Stat(string) (os.FileInfo, error) {
	return nil, fs.err
}

type appendOpenErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (fs appendOpenErrorFs) Open(name string) (afero.File, error) {
	if filepath.Clean(name) == filepath.Clean(fs.failPath) {
		return nil, fs.err
	}
	return fs.Fs.Open(name)
}
