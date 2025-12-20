package modinstall

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modpath"
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
		Hash:        sha1Hex("data"),
		DownloadUrl: "https://example.com/x.jar",
	}, nil, noopSender{})

	assert.NoError(t, err)
	assert.True(t, downloadCalled)
	assert.True(t, result.Downloaded)
	assert.Equal(t, EnsureReasonMissing, result.Reason)
}

func TestEnsureLockedFile_MissingFileMkdirFailureReturnsError(t *testing.T) {
	base := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{ModsFolder: "mods"}

	service := NewService(failingMkdirFs{Fs: base}, func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
		return nil
	})

	_, err := service.EnsureLockedFile(context.Background(), meta, cfg, models.ModInstall{
		FileName:    "x.jar",
		Hash:        sha1Hex("data"),
		DownloadUrl: "https://example.com/x.jar",
	}, nil, noopSender{})
	assert.Error(t, err)
}

func TestEnsureLockedFile_MissingFileMissingDownloaderReturnsError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{ModsFolder: "mods"}

	service := NewService(fs, nil)
	_, err := service.EnsureLockedFile(context.Background(), meta, cfg, models.ModInstall{
		FileName:    "x.jar",
		Hash:        sha1Hex("data"),
		DownloadUrl: "https://example.com/x.jar",
	}, nil, noopSender{})
	assert.Error(t, err)
}

func TestEnsureLockedFile_MissingFileNameReturnsError(t *testing.T) {
	service := NewService(afero.NewMemMapFs(), nil)
	_, err := service.EnsureLockedFile(context.Background(), config.NewMetadata("modlist.json"), models.ModsJson{ModsFolder: "mods"}, models.ModInstall{
		DownloadUrl: "https://example.com/x.jar",
	}, nil, noopSender{})
	assert.Error(t, err)
}

func TestEnsureLockedFile_InvalidFileNameReturnsError(t *testing.T) {
	service := NewService(afero.NewMemMapFs(), nil)
	_, err := service.EnsureLockedFile(context.Background(), config.NewMetadata("modlist.json"), models.ModsJson{ModsFolder: "mods"}, models.ModInstall{
		FileName:    "mods/x.jar",
		Hash:        sha1Hex("data"),
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
		Hash:        sha1Hex("new"),
		DownloadUrl: "https://example.com/x.jar",
	}, nil, noopSender{})

	assert.NoError(t, err)
	assert.True(t, result.Downloaded)
	assert.Equal(t, EnsureReasonHashMismatch, result.Reason)
}

func TestEnsureLockedFile_ResolvesSymlinkTargetInsideRoot(t *testing.T) {
	fs := afero.NewOsFs()
	root := t.TempDir()
	meta := config.NewMetadata(filepath.Join(root, "modlist.json"))
	cfg := models.ModsJson{ModsFolder: "mods"}

	modsDir := meta.ModsFolderPath(cfg)
	assert.NoError(t, os.MkdirAll(modsDir, 0755))

	target := filepath.Join(modsDir, "target.jar")
	assert.NoError(t, os.WriteFile(target, []byte("old"), 0644))

	link := filepath.Join(modsDir, "link.jar")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	service := NewService(fs, func(_ context.Context, _ string, dst string, _ httpClient.Doer, _ httpClient.Sender, filesystem ...afero.Fs) error {
		return afero.WriteFile(filesystem[0], dst, []byte("data"), 0644)
	})

	result, err := service.EnsureLockedFile(context.Background(), meta, cfg, models.ModInstall{
		FileName:    "link.jar",
		Hash:        sha1Hex("data"),
		DownloadUrl: "https://example.com/x.jar",
	}, nil, noopSender{})

	assert.NoError(t, err)
	assert.True(t, result.Downloaded)
	assert.Equal(t, EnsureReasonHashMismatch, result.Reason)

	content, readErr := os.ReadFile(target)
	assert.NoError(t, readErr)
	assert.Equal(t, []byte("data"), content)

	linkInfo, statErr := os.Lstat(link)
	assert.NoError(t, statErr)
	assert.True(t, linkInfo.Mode()&os.ModeSymlink != 0)
}

func TestEnsureLockedFile_RejectsSymlinkTargetOutsideRoot(t *testing.T) {
	fs := afero.NewOsFs()
	root := t.TempDir()
	meta := config.NewMetadata(filepath.Join(root, "modlist.json"))
	cfg := models.ModsJson{ModsFolder: "mods"}

	modsDir := meta.ModsFolderPath(cfg)
	assert.NoError(t, os.MkdirAll(modsDir, 0755))

	outside := t.TempDir()
	target := filepath.Join(outside, "target.jar")
	assert.NoError(t, os.WriteFile(target, []byte("data"), 0644))

	link := filepath.Join(modsDir, "link.jar")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	service := NewService(fs, func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
		return errors.New("unexpected download")
	})

	_, err := service.EnsureLockedFile(context.Background(), meta, cfg, models.ModInstall{
		FileName:    "link.jar",
		Hash:        sha1Hex("data"),
		DownloadUrl: "https://example.com/x.jar",
	}, nil, noopSender{})

	var outsideErr modpath.OutsideRootError
	assert.ErrorAs(t, err, &outsideErr)
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

func TestEnsureLockedFile_ReturnsErrorWhenExistsCheckFails(t *testing.T) {
	base := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{ModsFolder: "mods"}

	destination := filepath.Join(meta.ModsFolderPath(cfg), "x.jar")
	service := NewService(failingStatFs{Fs: base, failPath: destination}, nil)
	_, err := service.EnsureLockedFile(context.Background(), meta, cfg, models.ModInstall{
		FileName:    "x.jar",
		Hash:        sha1Hex("data"),
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
		Hash:        sha1Hex("data"),
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
		Hash:        sha1Hex("data"),
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
		Hash:        sha1Hex("data"),
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
		Hash:        sha1Hex("data"),
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
		Hash:        sha1Hex("data"),
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
		Hash:        sha1Hex("data"),
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
		Hash:        sha1Hex("data"),
		DownloadUrl: "https://example.com/x.jar",
	}, nil, noopSender{})
	assert.Error(t, err)
}

func TestEnsureLockedFile_MissingHashReturnsError(t *testing.T) {
	service := NewService(afero.NewMemMapFs(), func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
		return errors.New("unexpected download")
	})

	_, err := service.EnsureLockedFile(context.Background(), config.NewMetadata("modlist.json"), models.ModsJson{ModsFolder: "mods"}, models.ModInstall{
		FileName:    "x.jar",
		DownloadUrl: "https://example.com/x.jar",
	}, nil, noopSender{})
	var missingHash MissingHashError
	assert.ErrorAs(t, err, &missingHash)
}

func TestEnsureLockedFile_DownloadedFileHashMismatchReturnsError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{ModsFolder: "mods"}

	service := NewService(fs, func(_ context.Context, _ string, dst string, _ httpClient.Doer, _ httpClient.Sender, filesystem ...afero.Fs) error {
		return afero.WriteFile(filesystem[0], dst, []byte("actual"), 0644)
	})

	_, err := service.EnsureLockedFile(context.Background(), meta, cfg, models.ModInstall{
		FileName:    "x.jar",
		Hash:        sha1Hex("expected"),
		DownloadUrl: "https://example.com/x.jar",
	}, nil, noopSender{})
	var mismatch HashMismatchError
	assert.ErrorAs(t, err, &mismatch)
}

func TestMissingHashErrorMessage(t *testing.T) {
	err := MissingHashError{FileName: "x.jar"}
	assert.Contains(t, err.Error(), "missing expected hash")
}

func TestHashMismatchErrorMessage(t *testing.T) {
	err := HashMismatchError{FileName: "x.jar", Expected: "a", Actual: "b"}
	assert.Contains(t, err.Error(), "hash mismatch")
}

func TestDownloadAndVerifyWritesFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	service := NewService(fs, func(_ context.Context, _ string, dst string, _ httpClient.Doer, _ httpClient.Sender, filesystem ...afero.Fs) error {
		return afero.WriteFile(filesystem[0], dst, []byte("data"), 0644)
	})

	err := service.DownloadAndVerify(context.Background(), "https://example.com/x.jar", filepath.FromSlash("/mods/x.jar"), sha1Hex("data"), nil, nil)
	assert.NoError(t, err)

	content, readErr := afero.ReadFile(fs, filepath.FromSlash("/mods/x.jar"))
	assert.NoError(t, readErr)
	assert.Equal(t, []byte("data"), content)
}

func TestDownloadAndVerifyReturnsErrorOnDownloadFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	service := NewService(fs, func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
		return errors.New("download failed")
	})

	err := service.DownloadAndVerify(context.Background(), "https://example.com/x.jar", filepath.FromSlash("/mods/x.jar"), sha1Hex("data"), nil, noopSender{})
	assert.Error(t, err)
}

func TestDownloadAndVerifyReturnsErrorOnTempFileFailure(t *testing.T) {
	base := afero.NewMemMapFs()
	assert.NoError(t, base.MkdirAll(filepath.FromSlash("/mods"), 0755))

	service := NewService(openFileErrorFs{Fs: base, err: errors.New("open failed")}, func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
		return nil
	})

	err := service.DownloadAndVerify(context.Background(), "https://example.com/x.jar", filepath.FromSlash("/mods/x.jar"), sha1Hex("data"), nil, noopSender{})
	assert.Error(t, err)
}

func TestDownloadAndVerifyReturnsMissingHashError(t *testing.T) {
	fs := afero.NewMemMapFs()
	service := NewService(fs, func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
		return errors.New("unexpected download")
	})

	err := service.DownloadAndVerify(context.Background(), "https://example.com/x.jar", filepath.FromSlash("/mods/x.jar"), "", nil, noopSender{})
	var missingHash MissingHashError
	assert.ErrorAs(t, err, &missingHash)
}

func TestDownloadAndVerifyReturnsErrorWhenDownloaderMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	service := NewService(fs, nil)

	err := service.DownloadAndVerify(context.Background(), "https://example.com/x.jar", filepath.FromSlash("/mods/x.jar"), sha1Hex("data"), nil, noopSender{})
	assert.Error(t, err)
}

func TestDownloadAndVerifyReturnsErrorOnHashReadFailure(t *testing.T) {
	base := afero.NewMemMapFs()
	fs := failingOpenFs{Fs: base, failContains: ".mmm."}
	service := NewService(fs, func(_ context.Context, _ string, dst string, _ httpClient.Doer, _ httpClient.Sender, filesystem ...afero.Fs) error {
		return afero.WriteFile(filesystem[0], dst, []byte("data"), 0644)
	})

	err := service.DownloadAndVerify(context.Background(), "https://example.com/x.jar", filepath.FromSlash("/mods/x.jar"), sha1Hex("data"), nil, noopSender{})
	assert.Error(t, err)
}

func TestDownloadAndVerifyReturnsErrorOnReplaceFailure(t *testing.T) {
	base := afero.NewMemMapFs()
	fs := renameErrorFs{Fs: base, failOldContains: ".mmm.", failNew: filepath.FromSlash("/mods/x.jar"), err: errors.New("rename failed")}
	service := NewService(fs, func(_ context.Context, _ string, dst string, _ httpClient.Doer, _ httpClient.Sender, filesystem ...afero.Fs) error {
		return afero.WriteFile(filesystem[0], dst, []byte("data"), 0644)
	})

	err := service.DownloadAndVerify(context.Background(), "https://example.com/x.jar", filepath.FromSlash("/mods/x.jar"), sha1Hex("data"), nil, noopSender{})
	assert.Error(t, err)
}

func TestReplaceExistingFileRenamesWhenDestinationMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	source := filepath.FromSlash("/mods/source.jar")
	destination := filepath.FromSlash("/mods/dest.jar")

	assert.NoError(t, afero.WriteFile(fs, source, []byte("source"), 0644))
	assert.NoError(t, replaceExistingFile(fs, source, destination))

	exists, _ := afero.Exists(fs, destination)
	assert.True(t, exists)
	exists, _ = afero.Exists(fs, source)
	assert.False(t, exists)
}

func TestReplaceExistingFileReturnsErrorOnStatFailure(t *testing.T) {
	fs := failingStatFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.FromSlash("/mods/dest.jar"),
	}

	err := replaceExistingFile(fs, filepath.FromSlash("/mods/source.jar"), filepath.FromSlash("/mods/dest.jar"))
	assert.Error(t, err)
}

func TestReplaceExistingFileReturnsErrorOnBackupRenameFailure(t *testing.T) {
	base := afero.NewMemMapFs()
	destination := filepath.FromSlash("/mods/dest.jar")
	source := filepath.FromSlash("/mods/source.jar")
	backup := destination + ".mmm.bak"

	assert.NoError(t, afero.WriteFile(base, destination, []byte("old"), 0644))
	assert.NoError(t, afero.WriteFile(base, source, []byte("new"), 0644))

	fs := renameErrorFs{Fs: base, failOld: destination, failNew: backup, err: errors.New("rename failed")}
	err := replaceExistingFile(fs, source, destination)
	assert.Error(t, err)
}

func TestReplaceExistingFileReturnsErrorOnNextBackupPathFailure(t *testing.T) {
	base := afero.NewMemMapFs()
	destination := filepath.FromSlash("/mods/dest.jar")
	source := filepath.FromSlash("/mods/source.jar")
	backup := destination + ".mmm.bak"

	assert.NoError(t, afero.WriteFile(base, destination, []byte("old"), 0644))
	assert.NoError(t, afero.WriteFile(base, source, []byte("new"), 0644))

	fs := failingStatFs{Fs: base, failPath: backup}
	err := replaceExistingFile(fs, source, destination)
	assert.Error(t, err)
}

func TestReplaceExistingFileRestoresBackupOnRenameFailure(t *testing.T) {
	base := afero.NewMemMapFs()
	destination := filepath.FromSlash("/mods/dest.jar")
	source := filepath.FromSlash("/mods/source.jar")

	assert.NoError(t, afero.WriteFile(base, destination, []byte("old"), 0644))
	assert.NoError(t, afero.WriteFile(base, source, []byte("new"), 0644))

	fs := renameErrorFs{Fs: base, failOld: source, failNew: destination, err: errors.New("rename failed")}
	err := replaceExistingFile(fs, source, destination)
	assert.Error(t, err)

	content, readErr := afero.ReadFile(fs, destination)
	assert.NoError(t, readErr)
	assert.Equal(t, []byte("old"), content)
}

func TestNextBackupPathSkipsExisting(t *testing.T) {
	fs := afero.NewMemMapFs()
	target := filepath.FromSlash("/mods/mod.jar")
	base := target + ".mmm.bak"

	assert.NoError(t, afero.WriteFile(fs, base, []byte("backup"), 0644))
	next, err := nextBackupPath(fs, target)
	assert.NoError(t, err)
	assert.Equal(t, base+".1", next)
}

func TestNextBackupPathReturnsErrorOnStatFailure(t *testing.T) {
	fs := failingStatFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.FromSlash("/mods/mod.jar.mmm.bak"),
	}

	_, err := nextBackupPath(fs, filepath.FromSlash("/mods/mod.jar"))
	assert.Error(t, err)
}

func TestNextBackupPathReturnsErrorAfterExhaustion(t *testing.T) {
	fs := afero.NewMemMapFs()
	target := filepath.FromSlash("/mods/mod.jar")
	base := target + ".mmm.bak"

	assert.NoError(t, afero.WriteFile(fs, base, []byte("backup"), 0644))
	for i := 1; i < 100; i++ {
		assert.NoError(t, afero.WriteFile(fs, base+"."+strconv.Itoa(i), []byte("backup"), 0644))
	}

	_, err := nextBackupPath(fs, target)
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
	failPath     string
	failContains string
}

func (f failingOpenFs) Open(name string) (afero.File, error) {
	if f.failPath != "" && filepath.Clean(name) == filepath.Clean(f.failPath) {
		return nil, errors.New("open failed")
	}
	if f.failContains != "" && strings.Contains(filepath.Clean(name), f.failContains) {
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

func sha1Hex(data string) string {
	sum := sha1.Sum([]byte(data))
	return hex.EncodeToString(sum[:])
}

type renameErrorFs struct {
	afero.Fs
	failOld         string
	failNew         string
	failOldContains string
	err             error
}

func (r renameErrorFs) Rename(oldname, newname string) error {
	if r.failOldContains != "" && strings.Contains(filepath.Clean(oldname), r.failOldContains) && filepath.Clean(newname) == filepath.Clean(r.failNew) {
		return r.err
	}
	if r.failOld != "" && filepath.Clean(oldname) == filepath.Clean(r.failOld) && filepath.Clean(newname) == filepath.Clean(r.failNew) {
		return r.err
	}
	return r.Fs.Rename(oldname, newname)
}

type openFileErrorFs struct {
	afero.Fs
	err error
}

func (o openFileErrorFs) OpenFile(string, int, os.FileMode) (afero.File, error) {
	return nil, o.err
}
