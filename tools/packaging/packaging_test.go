package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNormalizeVersionDefaultsToDev(t *testing.T) {
	if normalizeVersion("") != "dev" {
		t.Fatalf("expected dev for empty version")
	}
	if normalizeVersion("   ") != "dev" {
		t.Fatalf("expected dev for whitespace version")
	}
	if normalizeVersion("1.2.3") != "1.2.3" {
		t.Fatalf("expected version to remain unchanged")
	}
}

func TestNormalizeVersionStripsPathSeparators(t *testing.T) {
	version := normalizeVersion(" ../1.2.3/../../evil ")
	if strings.ContainsAny(version, `/\:`) {
		t.Fatalf("expected sanitized version, got %q", version)
	}
	if version == "dev" {
		t.Fatalf("expected non-dev version after sanitizing")
	}
}

func TestRunMainGetwdFailure(t *testing.T) {
	originalNewDistToolFunc := newDistToolFunc
	t.Cleanup(func() {
		newDistToolFunc = originalNewDistToolFunc
	})

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	newDistToolFunc = func() (*distTool, error) {
		return nil, errors.New("boom")
	}

	if exitCode := runMain(); exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestMainSuccessUsesExit(t *testing.T) {
	originalNewDistToolFunc := newDistToolFunc
	originalExit := exit
	t.Cleanup(func() {
		newDistToolFunc = originalNewDistToolFunc
		exit = originalExit
	})

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	tempDir := t.TempDir()
	buildDir := filepath.Join(tempDir, "build", "linux", "amd64")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatalf("failed to create build dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(buildDir, executableName), []byte("linux-binary"), 0o755); err != nil {
		t.Fatalf("failed to write binary: %v", err)
	}

	newDistToolFunc = func() (*distTool, error) {
		return &distTool{
			repoRoot: tempDir,
			logger:   log.New(&bytes.Buffer{}, "dist: ", 0),
		}, nil
	}

	exitCode := 99
	exit = func(code int) {
		exitCode = code
	}

	main()

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
}

func TestRunMainToolRunError(t *testing.T) {
	originalNewDistToolFunc := newDistToolFunc
	t.Cleanup(func() {
		newDistToolFunc = originalNewDistToolFunc
	})

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	newDistToolFunc = func() (*distTool, error) {
		return &distTool{
			repoRoot: t.TempDir(),
			logger:   log.New(&bytes.Buffer{}, "dist: ", 0),
		}, nil
	}

	if exitCode := runMain(); exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestNewDistToolFindsRepoRoot(t *testing.T) {
	originalGetWorkingDirectory := getWorkingDirectory
	t.Cleanup(func() {
		getWorkingDirectory = originalGetWorkingDirectory
	})

	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	nestedDir := filepath.Join(repoRoot, "tools", "dist")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/test"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	getWorkingDirectory = func() (string, error) {
		return nestedDir, nil
	}

	tool, err := newDistTool()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tool.repoRoot != repoRoot {
		t.Fatalf("expected repo root %q, got %q", repoRoot, tool.repoRoot)
	}
}

func TestNewDistToolMissingRepoRoot(t *testing.T) {
	originalGetWorkingDirectory := getWorkingDirectory
	t.Cleanup(func() {
		getWorkingDirectory = originalGetWorkingDirectory
	})

	tempDir := t.TempDir()
	getWorkingDirectory = func() (string, error) {
		return tempDir, nil
	}

	tool, err := newDistTool()
	if err == nil {
		t.Fatalf("expected error, got nil and tool %v", tool)
	}
}

func TestNewDistToolGetwdFailure(t *testing.T) {
	originalGetWorkingDirectory := getWorkingDirectory
	t.Cleanup(func() {
		getWorkingDirectory = originalGetWorkingDirectory
	})

	getWorkingDirectory = func() (string, error) {
		return "", errors.New("boom")
	}

	tool, err := newDistTool()
	if err == nil {
		t.Fatalf("expected error, got nil and tool %v", tool)
	}
}

func TestResetDistDirError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based permission test is not reliable on Windows")
	}
	parentDir := t.TempDir()
	if err := os.Chmod(parentDir, 0o500); err != nil {
		t.Fatalf("failed to chmod parent dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(parentDir, 0o700); err != nil {
			t.Fatalf("failed to restore parent dir perms: %v", err)
		}
	})

	distDir := filepath.Join(parentDir, "dist")
	if err := resetDistDir(distDir); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResetDistDirRemoveAllError(t *testing.T) {
	originalRemoveAll := removeAll
	t.Cleanup(func() {
		removeAll = originalRemoveAll
	})

	removeAll = func(string) error {
		return errors.New("remove failed")
	}

	if err := resetDistDir(filepath.Join(t.TempDir(), "dist")); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResetDistDirMkdirError(t *testing.T) {
	originalMkdirAll := mkdirAll
	t.Cleanup(func() {
		mkdirAll = originalMkdirAll
	})

	mkdirAll = func(string, os.FileMode) error {
		return errors.New("mkdir failed")
	}

	if err := resetDistDir(filepath.Join(t.TempDir(), "dist")); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFindBuildArtifactsNoOutputs(t *testing.T) {
	tempDir := t.TempDir()
	_, err := findBuildArtifacts(tempDir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFindBuildArtifacts(t *testing.T) {
	tempDir := t.TempDir()
	linuxDir := filepath.Join(tempDir, "linux", "amd64")
	windowsDir := filepath.Join(tempDir, "windows", "arm64")
	if err := os.MkdirAll(linuxDir, 0o755); err != nil {
		t.Fatalf("failed to create linux dir: %v", err)
	}
	if err := os.MkdirAll(windowsDir, 0o755); err != nil {
		t.Fatalf("failed to create windows dir: %v", err)
	}

	linuxBinary := filepath.Join(linuxDir, executableName)
	windowsBinary := filepath.Join(windowsDir, executableName+".exe")
	if err := os.WriteFile(linuxBinary, []byte("linux-binary"), 0o755); err != nil {
		t.Fatalf("failed to write linux binary: %v", err)
	}
	if err := os.WriteFile(windowsBinary, []byte("windows-binary"), 0o755); err != nil {
		t.Fatalf("failed to write windows binary: %v", err)
	}

	artifacts, err := findBuildArtifacts(tempDir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(artifacts))
	}
}

func TestFindBuildArtifactsSortsByArchWhenOSMatches(t *testing.T) {
	tempDir := t.TempDir()
	amdDir := filepath.Join(tempDir, "linux", "amd64")
	armDir := filepath.Join(tempDir, "linux", "arm64")
	if err := os.MkdirAll(amdDir, 0o755); err != nil {
		t.Fatalf("failed to create amd dir: %v", err)
	}
	if err := os.MkdirAll(armDir, 0o755); err != nil {
		t.Fatalf("failed to create arm dir: %v", err)
	}

	amdBinary := filepath.Join(amdDir, executableName)
	armBinary := filepath.Join(armDir, executableName)
	if err := os.WriteFile(amdBinary, []byte("amd-binary"), 0o755); err != nil {
		t.Fatalf("failed to write amd binary: %v", err)
	}
	if err := os.WriteFile(armBinary, []byte("arm-binary"), 0o755); err != nil {
		t.Fatalf("failed to write arm binary: %v", err)
	}

	artifacts, err := findBuildArtifacts(tempDir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(artifacts))
	}
	if artifacts[0].goarch != "amd64" || artifacts[1].goarch != "arm64" {
		t.Fatalf("unexpected order: %v", artifacts)
	}
}

func TestFindBuildArtifactsSkipsNonRegular(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "linux", "amd64", executableName)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	if _, err := findBuildArtifacts(tempDir); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFindBuildArtifactsInvalidGlob(t *testing.T) {
	originalGlob := glob
	t.Cleanup(func() {
		glob = originalGlob
	})

	glob = func(string) ([]string, error) {
		return nil, errors.New("glob fail")
	}

	if _, err := findBuildArtifacts(t.TempDir()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFindBuildArtifactsInvalidPathSegments(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "build", "linux")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}
	targetPath := filepath.Join(targetDir, executableName)
	if err := os.WriteFile(targetPath, []byte("binary"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if _, err := findBuildArtifacts(tempDir); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFindBuildArtifactsBuildArtifactError(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "build", "amd64")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}
	targetPath := filepath.Join(targetDir, executableName)
	if err := os.WriteFile(targetPath, []byte("binary"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if _, err := findBuildArtifacts(tempDir); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBuildArtifactFromPathMissingSegments(t *testing.T) {
	_, err := buildArtifactFromPath(filepath.Join("build", executableName))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBuildArtifactFromPathMissingOS(t *testing.T) {
	_, err := buildArtifactFromPath(filepath.Join("build", "linux", executableName))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFindBuildArtifactsStatError(t *testing.T) {
	originalStatFile := statFile
	t.Cleanup(func() {
		statFile = originalStatFile
	})
	statFile = func(string) (os.FileInfo, error) {
		return nil, errors.New("stat fail")
	}

	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "linux", "amd64")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}
	targetPath := filepath.Join(targetDir, executableName)
	if err := os.WriteFile(targetPath, []byte("binary"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if _, err := findBuildArtifacts(tempDir); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWriteZipWithDirectoryInput(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "output.zip")
	if err := writeZip(outputPath, tempDir); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWriteZipMissingInput(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "output.zip")
	if err := writeZip(outputPath, filepath.Join(tempDir, "missing")); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWriteZipCreateError(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "missing-dir", "output.zip")
	inputPath := filepath.Join(tempDir, "input.bin")
	if err := os.WriteFile(inputPath, []byte("data"), 0o644); err != nil {
		t.Fatalf("failed to write input: %v", err)
	}

	if err := writeZip(outputPath, inputPath); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWriteZipHeaderError(t *testing.T) {
	originalZipFileInfoHeader := zipFileInfoHeader
	t.Cleanup(func() {
		zipFileInfoHeader = originalZipFileInfoHeader
	})

	zipFileInfoHeader = func(os.FileInfo) (*zip.FileHeader, error) {
		return nil, errors.New("header failed")
	}

	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "input.bin")
	if err := os.WriteFile(inputPath, []byte("data"), 0o644); err != nil {
		t.Fatalf("failed to write input: %v", err)
	}

	if err := writeZip(filepath.Join(tempDir, "output.zip"), inputPath); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWriteZipCreateHeaderError(t *testing.T) {
	originalZipCreateHeader := zipCreateHeader
	t.Cleanup(func() {
		zipCreateHeader = originalZipCreateHeader
	})

	zipCreateHeader = func(writer *zip.Writer, header *zip.FileHeader) (io.Writer, error) {
		return nil, errors.New("create header failed")
	}

	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "input.bin")
	if err := os.WriteFile(inputPath, []byte("data"), 0o644); err != nil {
		t.Fatalf("failed to write input: %v", err)
	}

	if err := writeZip(filepath.Join(tempDir, "output.zip"), inputPath); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWriteZipCopyError(t *testing.T) {
	originalCopyFile := copyFile
	t.Cleanup(func() {
		copyFile = originalCopyFile
	})

	copyFile = func(io.Writer, io.Reader) (int64, error) {
		return 0, errors.New("copy failed")
	}

	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "input.bin")
	if err := os.WriteFile(inputPath, []byte("data"), 0o644); err != nil {
		t.Fatalf("failed to write input: %v", err)
	}

	if err := writeZip(filepath.Join(tempDir, "output.zip"), inputPath); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWriteZipOpenError(t *testing.T) {
	originalOpenFile := openFile
	t.Cleanup(func() {
		openFile = originalOpenFile
	})

	openFile = func(string) (*os.File, error) {
		return nil, errors.New("open failed")
	}

	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "input.bin")
	if err := os.WriteFile(inputPath, []byte("data"), 0o644); err != nil {
		t.Fatalf("failed to write input: %v", err)
	}

	if err := writeZip(filepath.Join(tempDir, "output.zip"), inputPath); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDistToolRunCreatesZipsAndCleansDist(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/test"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	buildDir := filepath.Join(tempDir, "build", "linux", "amd64")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatalf("failed to create build dir: %v", err)
	}
	linuxBinary := filepath.Join(buildDir, executableName)
	if err := os.WriteFile(linuxBinary, []byte("linux-binary"), 0o755); err != nil {
		t.Fatalf("failed to write linux binary: %v", err)
	}

	distDir := filepath.Join(tempDir, "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatalf("failed to create dist dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "stale.txt"), []byte("stale"), 0o644); err != nil {
		t.Fatalf("failed to write stale file: %v", err)
	}

	logBuffer := &bytes.Buffer{}
	tool := &distTool{
		repoRoot: tempDir,
		logger:   log.New(logBuffer, "dist: ", 0),
	}

	if err := tool.run("1.2.3"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedZip := filepath.Join(distDir, "mmm-linux-amd64-1.2.3.zip")
	if _, err := os.Stat(expectedZip); err != nil {
		t.Fatalf("expected zip to exist: %v", err)
	}

	if _, err := os.Stat(filepath.Join(distDir, "stale.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected stale file removed, got %v", err)
	}

	zipReader, err := zip.OpenReader(expectedZip)
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	t.Cleanup(func() {
		if err := zipReader.Close(); err != nil {
			t.Fatalf("failed to close zip reader: %v", err)
		}
	})

	if len(zipReader.File) != 1 {
		t.Fatalf("expected one file in zip, got %d", len(zipReader.File))
	}
	if zipReader.File[0].Name != executableName {
		t.Fatalf("expected zip entry %q, got %q", executableName, zipReader.File[0].Name)
	}
}

func TestDistToolRunDoesNotCleanDistWhenNoArtifacts(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/test"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	distDir := filepath.Join(tempDir, "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatalf("failed to create dist dir: %v", err)
	}
	stalePath := filepath.Join(distDir, "stale.txt")
	if err := os.WriteFile(stalePath, []byte("stale"), 0o644); err != nil {
		t.Fatalf("failed to write stale file: %v", err)
	}

	logBuffer := &bytes.Buffer{}
	tool := &distTool{
		repoRoot: tempDir,
		logger:   log.New(logBuffer, "dist: ", 0),
	}

	if err := tool.run("1.2.3"); err == nil {
		t.Fatal("expected error, got nil")
	}

	if _, err := os.Stat(stalePath); err != nil {
		t.Fatalf("expected dist contents to remain, got %v", err)
	}
}

func TestDistToolRunResetDistDirError(t *testing.T) {
	originalRemoveAll := removeAll
	t.Cleanup(func() {
		removeAll = originalRemoveAll
	})
	removeAll = func(string) error {
		return errors.New("remove failed")
	}

	tempDir := t.TempDir()
	buildDir := filepath.Join(tempDir, "build", "linux", "amd64")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatalf("failed to create build dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(buildDir, executableName), []byte("linux-binary"), 0o755); err != nil {
		t.Fatalf("failed to write binary: %v", err)
	}

	tool := &distTool{
		repoRoot: tempDir,
		logger:   log.New(&bytes.Buffer{}, "dist: ", 0),
	}

	if err := tool.run("dev"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDistToolRunWriteZipError(t *testing.T) {
	originalZipFileInfoHeader := zipFileInfoHeader
	t.Cleanup(func() {
		zipFileInfoHeader = originalZipFileInfoHeader
	})

	zipFileInfoHeader = func(os.FileInfo) (*zip.FileHeader, error) {
		return nil, errors.New("header failed")
	}

	tempDir := t.TempDir()
	buildDir := filepath.Join(tempDir, "build", "linux", "amd64")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatalf("failed to create build dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(buildDir, executableName), []byte("linux-binary"), 0o755); err != nil {
		t.Fatalf("failed to write binary: %v", err)
	}

	tool := &distTool{
		repoRoot: tempDir,
		logger:   log.New(&bytes.Buffer{}, "dist: ", 0),
	}

	if err := tool.run("dev"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDistToolRunNoArtifactsReturnsError(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/test"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	tool := &distTool{
		repoRoot: tempDir,
		logger:   log.New(&bytes.Buffer{}, "dist: ", 0),
	}

	if err := tool.run("dev"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
