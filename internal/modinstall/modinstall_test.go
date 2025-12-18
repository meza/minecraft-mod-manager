package modinstall

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

func TestEnsureLockedFile_MissingFileDownloads(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{ModsFolder: "mods"}

	downloadCalled := false
	service := NewService(fs, func(_ context.Context, url string, path string, _ httpClient.Doer, _ httpClient.Sender, filesystem ...afero.Fs) error {
		downloadCalled = true
		assert.Equal(t, "https://example.com/x.jar", url)
		return afero.WriteFile(filesystem[0], path, []byte("data"), 0644)
	})

	result, err := service.EnsureLockedFile(context.Background(), meta, cfg, models.ModInstall{
		FileName:    "x.jar",
		Hash:        "ignored",
		DownloadUrl: "https://example.com/x.jar",
	}, nil, noopSender{})

	assert.NoError(t, err)
	assert.True(t, downloadCalled)
	assert.True(t, result.Downloaded)
	assert.Equal(t, EnsureReasonMissing, result.Reason)
}

func TestEnsureLockedFile_MissingFileNameReturnsError(t *testing.T) {
	service := NewService(afero.NewMemMapFs(), nil)
	_, err := service.EnsureLockedFile(context.Background(), config.NewMetadata("modlist.json"), models.ModsJson{ModsFolder: "mods"}, models.ModInstall{
		DownloadUrl: "https://example.com/x.jar",
	}, nil, noopSender{})
	assert.Error(t, err)
}

func TestEnsureLockedFile_ExistingFileWithMatchingHashDoesNothing(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{ModsFolder: "mods"}

	data := []byte("data")
	sum := sha1.Sum(data)
	hash := hex.EncodeToString(sum[:])

	path := filepath.Join(meta.ModsFolderPath(cfg), "x.jar")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(fs, path, data, 0644))

	service := NewService(fs, func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
		return errors.New("unexpected download")
	})

	result, err := service.EnsureLockedFile(context.Background(), meta, cfg, models.ModInstall{
		FileName:    "x.jar",
		Hash:        hash,
		DownloadUrl: "https://example.com/x.jar",
	}, nil, noopSender{})

	assert.NoError(t, err)
	assert.False(t, result.Downloaded)
	assert.Equal(t, EnsureReasonAlreadyPresent, result.Reason)
}

func TestEnsureLockedFile_ExistingFileWithMismatchedHashDownloads(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{ModsFolder: "mods"}

	path := filepath.Join(meta.ModsFolderPath(cfg), "x.jar")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(fs, path, []byte("old"), 0644))

	service := NewService(fs, func(_ context.Context, _ string, dst string, _ httpClient.Doer, _ httpClient.Sender, filesystem ...afero.Fs) error {
		return afero.WriteFile(filesystem[0], dst, []byte("new"), 0644)
	})

	result, err := service.EnsureLockedFile(context.Background(), meta, cfg, models.ModInstall{
		FileName:    "x.jar",
		Hash:        "does-not-match",
		DownloadUrl: "https://example.com/x.jar",
	}, nil, noopSender{})

	assert.NoError(t, err)
	assert.True(t, result.Downloaded)
	assert.Equal(t, EnsureReasonHashMismatch, result.Reason)
}

func TestEnsureLockedFile_ExistingFileWithMismatchedHashDownloadErrorReturnsError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{ModsFolder: "mods"}

	path := filepath.Join(meta.ModsFolderPath(cfg), "x.jar")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(fs, path, []byte("old"), 0644))

	service := NewService(fs, func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
		return errors.New("download failed")
	})

	_, err := service.EnsureLockedFile(context.Background(), meta, cfg, models.ModInstall{
		FileName:    "x.jar",
		Hash:        "does-not-match",
		DownloadUrl: "https://example.com/x.jar",
	}, nil, noopSender{})

	assert.Error(t, err)
}

func TestEnsureLockedFile_ExistingFileWithMismatchedHashMissingDownloaderReturnsError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{ModsFolder: "mods"}

	path := filepath.Join(meta.ModsFolderPath(cfg), "x.jar")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(fs, path, []byte("old"), 0644))

	service := NewService(fs, nil)
	_, err := service.EnsureLockedFile(context.Background(), meta, cfg, models.ModInstall{
		FileName:    "x.jar",
		Hash:        "does-not-match",
		DownloadUrl: "https://example.com/x.jar",
	}, nil, noopSender{})

	assert.Error(t, err)
}

func TestEnsureLockedFile_ExistingFileWithMismatchedHashMkdirFailureReturnsError(t *testing.T) {
	base := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{ModsFolder: "mods"}

	path := filepath.Join(meta.ModsFolderPath(cfg), "x.jar")
	assert.NoError(t, base.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(base, path, []byte("old"), 0644))

	service := NewService(failingMkdirFs{Fs: base}, func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
		return nil
	})

	_, err := service.EnsureLockedFile(context.Background(), meta, cfg, models.ModInstall{
		FileName:    "x.jar",
		Hash:        "does-not-match",
		DownloadUrl: "https://example.com/x.jar",
	}, nil, noopSender{})
	assert.Error(t, err)
}

func TestEnsureLockedFile_UsesNoopSenderWhenNil(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{ModsFolder: "mods"}

	service := NewService(fs, func(_ context.Context, _ string, dst string, _ httpClient.Doer, sender httpClient.Sender, filesystem ...afero.Fs) error {
		assert.NotNil(t, sender)
		return afero.WriteFile(filesystem[0], dst, []byte("data"), 0644)
	})

	_, err := service.EnsureLockedFile(context.Background(), meta, cfg, models.ModInstall{
		FileName:    "x.jar",
		Hash:        "ignored",
		DownloadUrl: "https://example.com/x.jar",
	}, nil, nil)
	assert.NoError(t, err)
}

func TestEnsureLockedFile_MissingDownloadUrlReturnsError(t *testing.T) {
	service := NewService(afero.NewMemMapFs(), nil)
	_, err := service.EnsureLockedFile(context.Background(), config.NewMetadata("modlist.json"), models.ModsJson{ModsFolder: "mods"}, models.ModInstall{
		FileName: "x.jar",
	}, nil, noopSender{})
	assert.Error(t, err)
}

func TestEnsureLockedFile_MissingDownloaderReturnsError(t *testing.T) {
	service := NewService(afero.NewMemMapFs(), nil)
	_, err := service.EnsureLockedFile(context.Background(), config.NewMetadata("modlist.json"), models.ModsJson{ModsFolder: "mods"}, models.ModInstall{
		FileName:    "x.jar",
		DownloadUrl: "https://example.com/x.jar",
		Hash:        "abc",
	}, nil, noopSender{})
	assert.Error(t, err)
}

func TestEnsureLockedFile_ExistsCheckErrorReturnsError(t *testing.T) {
	fs := failingStatFs{Fs: afero.NewMemMapFs(), failPath: filepath.FromSlash("/cfg/mods/x.jar")}
	service := NewService(fs, nil)

	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	_, err := service.EnsureLockedFile(context.Background(), meta, models.ModsJson{ModsFolder: "mods"}, models.ModInstall{
		FileName:    "x.jar",
		DownloadUrl: "https://example.com/x.jar",
		Hash:        "abc",
	}, nil, noopSender{})
	assert.Error(t, err)
}

func TestEnsureLockedFile_Sha1OpenErrorReturnsError(t *testing.T) {
	base := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{ModsFolder: "mods"}

	path := filepath.Join(meta.ModsFolderPath(cfg), "x.jar")
	assert.NoError(t, base.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(base, path, []byte("data"), 0644))

	service := NewService(failingOpenFs{Fs: base, failPath: path}, nil)
	_, err := service.EnsureLockedFile(context.Background(), meta, cfg, models.ModInstall{
		FileName:    "x.jar",
		Hash:        "abc",
		DownloadUrl: "https://example.com/x.jar",
	}, nil, noopSender{})
	assert.Error(t, err)
}

func TestEnsureLockedFile_Sha1ReadErrorReturnsError(t *testing.T) {
	base := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{ModsFolder: "mods"}

	path := filepath.Join(meta.ModsFolderPath(cfg), "x.jar")
	assert.NoError(t, base.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(base, path, []byte("data"), 0644))

	service := NewService(failingReadFs{Fs: base, failPath: path}, nil)
	_, err := service.EnsureLockedFile(context.Background(), meta, cfg, models.ModInstall{
		FileName:    "x.jar",
		Hash:        "abc",
		DownloadUrl: "https://example.com/x.jar",
	}, nil, noopSender{})
	assert.Error(t, err)
}

func TestEnsureLockedFile_MkdirFailureReturnsError(t *testing.T) {
	base := afero.NewMemMapFs()
	readOnly := afero.NewReadOnlyFs(base)

	service := NewService(readOnly, func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
		return nil
	})
	_, err := service.EnsureLockedFile(context.Background(), config.NewMetadata(filepath.FromSlash("/cfg/modlist.json")), models.ModsJson{ModsFolder: "mods"}, models.ModInstall{
		FileName:    "x.jar",
		Hash:        "abc",
		DownloadUrl: "https://example.com/x.jar",
	}, nil, noopSender{})
	assert.Error(t, err)
}

func TestEnsureLockedFile_DownloadErrorReturnsError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	service := NewService(fs, func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
		return errors.New("download failed")
	})

	_, err := service.EnsureLockedFile(context.Background(), meta, models.ModsJson{ModsFolder: "mods"}, models.ModInstall{
		FileName:    "x.jar",
		Hash:        "abc",
		DownloadUrl: "https://example.com/x.jar",
	}, nil, noopSender{})
	assert.Error(t, err)
}

func TestNoopSender_SendCovered(t *testing.T) {
	var sender noopSender
	sender.Send(nil)
}

type failingStatFs struct {
	afero.Fs
	failPath string
}

func (f failingStatFs) Stat(name string) (os.FileInfo, error) {
	if filepath.Clean(name) == filepath.Clean(f.failPath) {
		return nil, errors.New("stat failed")
	}
	return f.Fs.Stat(name)
}

type failingOpenFs struct {
	afero.Fs
	failPath string
}

func (f failingOpenFs) Open(name string) (afero.File, error) {
	if filepath.Clean(name) == filepath.Clean(f.failPath) {
		return nil, errors.New("open failed")
	}
	return f.Fs.Open(name)
}

type failingReadFs struct {
	afero.Fs
	failPath string
}

func (f failingReadFs) Open(name string) (afero.File, error) {
	file, err := f.Fs.Open(name)
	if err != nil {
		return nil, err
	}
	if filepath.Clean(name) == filepath.Clean(f.failPath) {
		return failingReaderFile{File: file}, nil
	}
	return file, nil
}

type failingReaderFile struct {
	afero.File
}

func (f failingReaderFile) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

var _ io.Reader = failingReaderFile{}

type failingMkdirFs struct {
	afero.Fs
}

func (f failingMkdirFs) MkdirAll(string, os.FileMode) error {
	return errors.New("mkdir failed")
}
