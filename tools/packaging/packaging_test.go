package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"log"
	"os"
	"path/filepath"
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
	defer func() {
		_ = zipReader.Close()
	}()

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
