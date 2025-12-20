package modpath

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestResolveWritablePath_NonOsFsReturnsDestination(t *testing.T) {
	fs := afero.NewMemMapFs()
	destination := filepath.FromSlash("mods/example.jar")

	resolved, err := ResolveWritablePath(fs, "mods", destination)

	assert.NoError(t, err)
	assert.Equal(t, destination, resolved)
}

func TestResolveWritablePath_AllowsSymlinkedRoot(t *testing.T) {
	fs := afero.NewOsFs()
	root := t.TempDir()
	modsRoot := filepath.Join(root, "mods")
	assert.NoError(t, os.MkdirAll(modsRoot, 0755))

	linkRoot := filepath.Join(root, "mods-link")
	if err := os.Symlink(modsRoot, linkRoot); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	destination := filepath.Join(linkRoot, "example.jar")
	resolved, err := ResolveWritablePath(fs, linkRoot, destination)
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(modsRoot, "example.jar"), resolved)
}

func TestResolveWritablePath_AllowsSymlinkedFileInsideRoot(t *testing.T) {
	fs := afero.NewOsFs()
	root := t.TempDir()
	modsRoot := filepath.Join(root, "mods")
	assert.NoError(t, os.MkdirAll(modsRoot, 0755))

	target := filepath.Join(modsRoot, "target.jar")
	assert.NoError(t, os.WriteFile(target, []byte("data"), 0644))

	linkPath := filepath.Join(modsRoot, "link.jar")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	resolved, err := ResolveWritablePath(fs, modsRoot, linkPath)
	assert.NoError(t, err)
	assert.Equal(t, target, resolved)
}

func TestResolveWritablePath_ResolvesRelativeSymlinkTarget(t *testing.T) {
	fs := afero.NewOsFs()
	root := t.TempDir()
	modsRoot := filepath.Join(root, "mods")
	assert.NoError(t, os.MkdirAll(modsRoot, 0755))

	target := filepath.Join(modsRoot, "target.jar")
	assert.NoError(t, os.WriteFile(target, []byte("data"), 0644))

	linkPath := filepath.Join(modsRoot, "link.jar")
	if err := os.Symlink("target.jar", linkPath); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	resolved, err := ResolveWritablePath(fs, modsRoot, linkPath)
	assert.NoError(t, err)
	assert.Equal(t, target, resolved)
}

func TestResolveWritablePath_ReturnsErrorWhenSymlinkTargetDirMissing(t *testing.T) {
	fs := afero.NewOsFs()
	root := t.TempDir()
	modsRoot := filepath.Join(root, "mods")
	assert.NoError(t, os.MkdirAll(modsRoot, 0755))

	linkPath := filepath.Join(modsRoot, "link.jar")
	if err := os.Symlink(filepath.Join("missing", "target.jar"), linkPath); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	_, err := ResolveWritablePath(fs, modsRoot, linkPath)
	assert.Error(t, err)
}

func TestResolveWritablePath_ReturnsErrorWhenDestinationDirMissing(t *testing.T) {
	fs := afero.NewOsFs()
	root := t.TempDir()
	modsRoot := filepath.Join(root, "mods")
	assert.NoError(t, os.MkdirAll(modsRoot, 0755))

	destination := filepath.Join(modsRoot, "missing", "file.jar")
	_, err := ResolveWritablePath(fs, modsRoot, destination)
	assert.Error(t, err)
}

func TestResolveWritablePath_RejectsSymlinkedFileOutsideRoot(t *testing.T) {
	fs := afero.NewOsFs()
	root := t.TempDir()
	modsRoot := filepath.Join(root, "mods")
	assert.NoError(t, os.MkdirAll(modsRoot, 0755))

	outside := t.TempDir()
	target := filepath.Join(outside, "target.jar")
	assert.NoError(t, os.WriteFile(target, []byte("data"), 0644))

	linkPath := filepath.Join(modsRoot, "link.jar")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	_, err := ResolveWritablePath(fs, modsRoot, linkPath)
	assert.Error(t, err)
	assert.IsType(t, OutsideRootError{}, err)
	assert.Contains(t, err.Error(), "outside root")
}

func TestResolveWritablePath_RejectsSymlinkedDirOutsideRoot(t *testing.T) {
	fs := afero.NewOsFs()
	root := t.TempDir()
	modsRoot := filepath.Join(root, "mods")
	assert.NoError(t, os.MkdirAll(modsRoot, 0755))

	outside := t.TempDir()
	linkDir := filepath.Join(modsRoot, "linked")
	if err := os.Symlink(outside, linkDir); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	destination := filepath.Join(linkDir, "example.jar")
	_, err := ResolveWritablePath(fs, modsRoot, destination)
	assert.Error(t, err)
	assert.IsType(t, OutsideRootError{}, err)
	assert.Contains(t, err.Error(), "outside root")
}

func TestResolveWritablePath_RejectsSymlinkedDirOutsideRootWithStubFs(t *testing.T) {
	root := absolutePath(t, "mods")
	destination := filepath.Join(root, "linked", "example.jar")
	outsideDir := absolutePath(t, "outside")

	fs := linkStubFs{
		Fs:       afero.NewMemMapFs(),
		symlinks: map[string]string{},
	}
	evalSymlinks := func(path string) (string, error) {
		if path == root {
			return root, nil
		}
		if path == filepath.Dir(destination) {
			return outsideDir, nil
		}
		return path, nil
	}

	_, err := resolveWritablePathWithFuncs(fs, root, destination, evalSymlinks, func(path string) (string, error) {
		return path, nil
	})
	assert.Error(t, err)
	assert.IsType(t, OutsideRootError{}, err)
}

func TestResolveWritablePath_ReturnsErrorWhenRootMissing(t *testing.T) {
	fs := afero.NewOsFs()
	root := filepath.Join(t.TempDir(), "missing")

	_, err := ResolveWritablePath(fs, root, filepath.Join(root, "file.jar"))
	assert.Error(t, err)
}

func TestResolveWritablePath_ReturnsErrorWhenAbsFails(t *testing.T) {
	fs := linkStubFs{
		Fs:       afero.NewMemMapFs(),
		symlinks: map[string]string{},
	}
	root := absolutePath(t, "mods")
	destination := filepath.Join(root, "file.jar")

	_, err := resolveWritablePathWithFuncs(fs, root, destination, func(path string) (string, error) {
		return path, nil
	}, func(string) (string, error) {
		return "", errors.New("abs failed")
	})
	assert.Error(t, err)
}

func TestResolveWritablePath_ReturnsErrorWhenAbsFailsOnResolvedDestination(t *testing.T) {
	fs := linkStubFs{
		Fs:       afero.NewMemMapFs(),
		symlinks: map[string]string{},
	}
	root := absolutePath(t, "mods")
	destination := filepath.Join(root, "file.jar")

	_, err := resolveWritablePathWithFuncs(fs, root, destination, func(path string) (string, error) {
		return path, nil
	}, func(path string) (string, error) {
		if path == root {
			return path, nil
		}
		return "", errors.New("abs failed")
	})
	assert.Error(t, err)
}

func TestResolveWritablePath_ReturnsErrorWhenAbsFailsOnSymlinkTarget(t *testing.T) {
	root := absolutePath(t, "mods")
	destination := filepath.Join(root, "link.jar")
	target := filepath.Join(root, "target.jar")
	fs := linkStubFs{
		Fs: afero.NewMemMapFs(),
		symlinks: map[string]string{
			destination: target,
		},
	}

	_, err := resolveWritablePathWithFuncs(fs, root, destination, func(path string) (string, error) {
		return path, nil
	}, func(path string) (string, error) {
		if path == root {
			return path, nil
		}
		return "", errors.New("abs failed")
	})
	assert.Error(t, err)
}

func TestResolveWritablePath_ReturnsErrorOnLstatFailure(t *testing.T) {
	root := t.TempDir()
	modsRoot := filepath.Join(root, "mods")
	assert.NoError(t, os.MkdirAll(modsRoot, 0755))

	fs := lstatErrorFs{OsFs: &afero.OsFs{}, err: os.ErrPermission}
	_, err := ResolveWritablePath(fs, modsRoot, filepath.Join(modsRoot, "file.jar"))
	assert.Error(t, err)
}

func TestResolveWritablePath_ReturnsErrorOnReadlinkFailure(t *testing.T) {
	root := t.TempDir()
	modsRoot := filepath.Join(root, "mods")
	assert.NoError(t, os.MkdirAll(modsRoot, 0755))

	target := filepath.Join(modsRoot, "target.jar")
	assert.NoError(t, os.WriteFile(target, []byte("data"), 0644))

	linkPath := filepath.Join(modsRoot, "link.jar")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	fs := readlinkErrorFs{OsFs: &afero.OsFs{}, err: os.ErrPermission}
	_, err := ResolveWritablePath(fs, modsRoot, linkPath)
	assert.Error(t, err)
}

func TestResolveWritablePath_ResolvesRelativeRootAgainstCwd(t *testing.T) {
	fs := afero.NewOsFs()
	root := t.TempDir()
	workingDir := filepath.Join(root, "work")
	modsRoot := filepath.Join(workingDir, "mods")
	assert.NoError(t, os.MkdirAll(modsRoot, 0755))

	previousWorkingDir, err := os.Getwd()
	assert.NoError(t, err)
	assert.NoError(t, os.Chdir(workingDir))
	t.Cleanup(func() { _ = os.Chdir(previousWorkingDir) })

	resolved, err := ResolveWritablePath(fs, "mods", filepath.Join("mods", "example.jar"))
	assert.NoError(t, err)
	assert.True(t, filepath.IsAbs(resolved))
	assert.Equal(t, filepath.Join(modsRoot, "example.jar"), resolved)
}

func TestResolveWritablePath_AllowsSymlinkedFileInsideRootWithStubFs(t *testing.T) {
	root := absolutePath(t, "mods")
	destination := filepath.Join(root, "link.jar")
	target := filepath.Join(root, "target.jar")
	fs := linkStubFs{
		Fs: afero.NewMemMapFs(),
		symlinks: map[string]string{
			destination: target,
		},
	}

	resolved, err := resolveWritablePathWithFuncs(fs, root, destination, func(path string) (string, error) {
		return path, nil
	}, func(path string) (string, error) {
		return path, nil
	})
	assert.NoError(t, err)
	assert.Equal(t, target, resolved)
}

func TestResolveWritablePath_RejectsSymlinkedFileOutsideRootWithStubFs(t *testing.T) {
	root := absolutePath(t, "mods")
	destination := filepath.Join(root, "link.jar")
	target := absolutePath(t, "outside", "target.jar")
	fs := linkStubFs{
		Fs: afero.NewMemMapFs(),
		symlinks: map[string]string{
			destination: target,
		},
	}

	_, err := resolveWritablePathWithFuncs(fs, root, destination, func(path string) (string, error) {
		return path, nil
	}, func(path string) (string, error) {
		return path, nil
	})
	assert.Error(t, err)
	assert.IsType(t, OutsideRootError{}, err)
}

func TestResolveWritablePath_ReturnsDestinationWhenLstaterMissing(t *testing.T) {
	fs := linkReaderOnlyFs{Fs: &afero.OsFs{}}
	root := t.TempDir()
	destination := filepath.Join(root, "mods", "file.jar")
	assert.NoError(t, os.MkdirAll(filepath.Dir(destination), 0755))

	resolved, err := ResolveWritablePath(fs, filepath.Dir(destination), destination)
	assert.NoError(t, err)
	assert.Equal(t, destination, resolved)
}

func TestPathWithinRootCoversBranches(t *testing.T) {
	root := absolutePath(t, "mods")
	assert.True(t, pathWithinRoot(root, root))
	assert.False(t, pathWithinRoot(root, filepath.Dir(root)))
	assert.False(t, pathWithinRoot(root, filepath.Join(root, "..", "other")))
	assert.False(t, pathWithinRoot("", filepath.Join(string(os.PathSeparator), "abs")))
}

type readlinkErrorFs struct {
	*afero.OsFs
	err error
}

func (r readlinkErrorFs) ReadlinkIfPossible(string) (string, error) {
	return "", r.err
}

type lstatErrorFs struct {
	*afero.OsFs
	err error
}

func (l lstatErrorFs) LstatIfPossible(string) (os.FileInfo, bool, error) {
	return nil, true, l.err
}

type linkReaderOnlyFs struct {
	afero.Fs
}

func (linkReaderOnlyFs) ReadlinkIfPossible(string) (string, error) {
	return "", os.ErrInvalid
}

func absolutePath(t *testing.T, parts ...string) string {
	t.Helper()

	root := string(os.PathSeparator)
	volume := filepath.VolumeName(os.TempDir())
	if volume != "" {
		root = volume + string(os.PathSeparator)
	}
	return filepath.Join(append([]string{root}, parts...)...)
}

type linkStubFs struct {
	afero.Fs
	symlinks map[string]string
}

func (fs linkStubFs) LstatIfPossible(path string) (os.FileInfo, bool, error) {
	if _, ok := fs.symlinks[path]; ok {
		return fakeFileInfo{name: filepath.Base(path), mode: os.ModeSymlink}, true, nil
	}
	return nil, true, os.ErrNotExist
}

func (fs linkStubFs) ReadlinkIfPossible(path string) (string, error) {
	target, ok := fs.symlinks[path]
	if !ok {
		return "", os.ErrNotExist
	}
	return target, nil
}

type fakeFileInfo struct {
	name string
	mode os.FileMode
}

func (info fakeFileInfo) Name() string       { return info.name }
func (info fakeFileInfo) Size() int64        { return 0 }
func (info fakeFileInfo) Mode() os.FileMode  { return info.mode }
func (info fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (info fakeFileInfo) IsDir() bool        { return false }
func (info fakeFileInfo) Sys() interface{}   { return nil }
