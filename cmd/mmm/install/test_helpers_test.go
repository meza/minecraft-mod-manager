package install

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/afero"
)

type errorWriter struct {
	err error
}

func (writer errorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

func sha1Hex(data string) string {
	sum := sha1.Sum([]byte(data))
	return hex.EncodeToString(sum[:])
}

type statErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (fs statErrorFs) Stat(name string) (os.FileInfo, error) {
	if filepath.Clean(name) == filepath.Clean(fs.failPath) {
		return nil, fs.err
	}
	return fs.Fs.Stat(name)
}

type symlinkStubFs struct {
	afero.Fs
	symlinks map[string]string
}

func (filesystem symlinkStubFs) LstatIfPossible(path string) (os.FileInfo, bool, error) {
	cleanPath := filepath.Clean(path)
	if _, ok := filesystem.symlinks[cleanPath]; ok {
		return fakeFileInfo{name: filepath.Base(cleanPath), mode: os.ModeSymlink}, true, nil
	}
	return nil, true, os.ErrNotExist
}

func (filesystem symlinkStubFs) ReadlinkIfPossible(path string) (string, error) {
	cleanPath := filepath.Clean(path)
	target, ok := filesystem.symlinks[cleanPath]
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
