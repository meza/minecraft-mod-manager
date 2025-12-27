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
	fs := linkStubFs{
		Fs:       afero.NewMemMapFs(),
		symlinks: map[string]string{},
	}
	linkRoot := absolutePath(t, "mods-link")
	modsRoot := absolutePath(t, "mods")
	destination := filepath.Join(linkRoot, "example.jar")

	resolved, err := resolveWritablePathWithFuncs(fs, linkRoot, destination, func(path string) (string, error) {
		if path == linkRoot || path == filepath.Dir(destination) {
			return modsRoot, nil
		}
		return path, nil
	}, func(path string) (string, error) {
		return path, nil
	})
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(modsRoot, "example.jar"), resolved)
}

func TestResolveWritablePath_AllowsSymlinkedFileInsideRoot(t *testing.T) {
	modsRoot := absolutePath(t, "mods")
	target := filepath.Join(modsRoot, "target.jar")
	linkPath := filepath.Join(modsRoot, "link.jar")
	fs := linkStubFs{
		Fs: afero.NewMemMapFs(),
		symlinks: map[string]string{
			linkPath: target,
		},
	}

	resolved, err := resolveWritablePathWithFuncs(fs, modsRoot, linkPath, func(path string) (string, error) {
		return path, nil
	}, func(path string) (string, error) {
		return path, nil
	})
	assert.NoError(t, err)
	assert.Equal(t, target, resolved)
}

func TestResolveWritablePath_ResolvesRelativeSymlinkTarget(t *testing.T) {
	modsRoot := absolutePath(t, "mods")
	linkPath := filepath.Join(modsRoot, "link.jar")
	fs := linkStubFs{
		Fs: afero.NewMemMapFs(),
		symlinks: map[string]string{
			linkPath: "target.jar",
		},
	}

	resolved, err := resolveWritablePathWithFuncs(fs, modsRoot, linkPath, func(path string) (string, error) {
		return path, nil
	}, func(path string) (string, error) {
		return path, nil
	})
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(modsRoot, "target.jar"), resolved)
}

func TestResolveWritablePath_ReturnsErrorWhenSymlinkTargetDirMissing(t *testing.T) {
	modsRoot := absolutePath(t, "mods")
	linkPath := filepath.Join(modsRoot, "link.jar")
	fs := linkStubFs{
		Fs: afero.NewMemMapFs(),
		symlinks: map[string]string{
			linkPath: filepath.Join("missing", "target.jar"),
		},
	}

	_, err := resolveWritablePathWithFuncs(fs, modsRoot, linkPath, func(path string) (string, error) {
		if path == filepath.Join(modsRoot, "missing") {
			return "", os.ErrNotExist
		}
		return path, nil
	}, func(path string) (string, error) {
		return path, nil
	})
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
	modsRoot := absolutePath(t, "mods")
	linkPath := filepath.Join(modsRoot, "link.jar")
	target := absolutePath(t, "outside", "target.jar")
	fs := linkStubFs{
		Fs: afero.NewMemMapFs(),
		symlinks: map[string]string{
			linkPath: target,
		},
	}

	_, err := resolveWritablePathWithFuncs(fs, modsRoot, linkPath, func(path string) (string, error) {
		return path, nil
	}, func(path string) (string, error) {
		return path, nil
	})
	assert.Error(t, err)
	assert.IsType(t, OutsideRootError{}, err)
	assert.Contains(t, err.Error(), "outside root")
}

func TestResolveWritablePath_RejectsSymlinkedDirOutsideRoot(t *testing.T) {
	root := absolutePath(t, "mods")
	destination := filepath.Join(root, "linked", "example.jar")
	outsideDir := absolutePath(t, "outside")

	fs := linkStubFs{
		Fs:       afero.NewMemMapFs(),
		symlinks: map[string]string{},
	}

	_, err := resolveWritablePathWithFuncs(fs, root, destination, func(path string) (string, error) {
		if path == root {
			return root, nil
		}
		if path == filepath.Dir(destination) {
			return outsideDir, nil
		}
		return path, nil
	}, func(path string) (string, error) {
		return path, nil
	})
	assert.Error(t, err)
	assert.IsType(t, OutsideRootError{}, err)
	assert.Contains(t, err.Error(), "outside root")
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
	root := absolutePath(t, "mods")
	linkPath := filepath.Join(root, "link.jar")
	fs := readlinkErrorFs{
		Fs:       afero.NewMemMapFs(),
		symlinks: map[string]string{linkPath: filepath.Join(root, "target.jar")},
		err:      os.ErrPermission,
	}

	_, err := resolveWritablePathWithFuncs(fs, root, linkPath, func(path string) (string, error) {
		return path, nil
	}, func(path string) (string, error) {
		return path, nil
	})
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
	t.Cleanup(func() { assert.NoError(t, os.Chdir(previousWorkingDir)) })

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
	afero.Fs
	symlinks map[string]string
	err      error
}

func (filesystem readlinkErrorFs) LstatIfPossible(path string) (os.FileInfo, bool, error) {
	if _, ok := filesystem.symlinks[path]; ok {
		return fakeFileInfo{name: filepath.Base(path), mode: os.ModeSymlink}, true, nil
	}
	return nil, true, os.ErrNotExist
}

func (filesystem readlinkErrorFs) ReadlinkIfPossible(string) (string, error) {
	return "", filesystem.err
}

type lstatErrorFs struct {
	*afero.OsFs
	err error
}

func (filesystem lstatErrorFs) LstatIfPossible(string) (os.FileInfo, bool, error) {
	return nil, true, filesystem.err
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
