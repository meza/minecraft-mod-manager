// Package main provides the packaging helper binary.
package main

import (
	"archive/zip"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	executableName  = "mmm"
	metadataDirName = "metadata"
	noticesFileName = "THIRD_PARTY_NOTICES.txt"
	sbomFileName    = "mmm-sbom.json"
)

type buildArtifact struct {
	goos   string
	goarch string
	path   string
}

type logger interface {
	Printf(format string, args ...any)
}

type distTool struct {
	repoRoot string
	logger   logger
}

var getWorkingDirectory = os.Getwd
var newDistToolFunc = newDistTool
var exit = os.Exit
var removeAll = os.RemoveAll
var mkdirAll = os.MkdirAll
var zipFileInfoHeader = zip.FileInfoHeader
var zipCreateHeader = func(writer *zip.Writer, header *zip.FileHeader) (io.Writer, error) {
	return writer.CreateHeader(header)
}
var copyFile = io.Copy
var openFile = os.Open
var createFile = func(path string, mode os.FileMode) (*os.File, error) {
	//nolint:gosec // output path is rooted in dist metadata dir with controlled filenames.
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
}
var closeInputFile = func(file *os.File) error { return file.Close() }
var closeOutputFile = func(file *os.File) error { return file.Close() }
var closeZipWriter = func(writer *zip.Writer) error { return writer.Close() }
var glob = filepath.Glob
var statFile = os.Stat
var stderrWriter io.Writer = os.Stderr

func main() {
	exit(runMain())
}

func runMain() int {
	versionFlag := flag.String("version", "dev", "version suffix for dist artifacts")
	flag.Parse()

	tool, err := newDistToolFunc()
	if err != nil {
		if _, writeErr := fmt.Fprintln(stderrWriter, err); writeErr != nil {
			return 1
		}
		return 1
	}

	if err := tool.run(*versionFlag); err != nil {
		if _, writeErr := fmt.Fprintln(stderrWriter, err); writeErr != nil {
			return 1
		}
		return 1
	}
	return 0
}

func newDistTool() (*distTool, error) {
	workingDirectory, err := getWorkingDirectory()
	if err != nil {
		return nil, fmt.Errorf("error: failed to determine working directory: %w", err)
	}

	repoRoot, err := findRepoRoot(workingDirectory)
	if err != nil {
		return nil, err
	}

	return &distTool{
		repoRoot: repoRoot,
		logger:   log.New(os.Stdout, "dist: ", 0),
	}, nil
}

func (tool *distTool) run(version string) error {
	normalizedVersion := normalizeVersion(version)
	buildDir := filepath.Join(tool.repoRoot, "build")
	distDir := filepath.Join(tool.repoRoot, "dist")

	artifacts, err := findBuildArtifacts(buildDir)
	if err != nil {
		return err
	}
	metadataFiles, err := findMetadataFiles(buildDir)
	if err != nil {
		return err
	}

	if err := resetDistDir(distDir); err != nil {
		return err
	}
	if err := copyMetadataFiles(distDir, metadataFiles); err != nil {
		return err
	}

	for _, artifact := range artifacts {
		outputName := fmt.Sprintf("mmm-%s-%s-%s.zip", artifact.goos, artifact.goarch, normalizedVersion)
		outputPath := filepath.Join(distDir, outputName)
		if err := writeZip(outputPath, artifact.path); err != nil {
			return err
		}
		tool.logger.Printf("created %s", outputPath)
	}

	return nil
}

func normalizeVersion(version string) string {
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return "dev"
	}
	sanitized := strings.NewReplacer("/", "-", "\\", "-", ":", "-").Replace(trimmed)
	return sanitized
}

func resetDistDir(distDir string) error {
	if err := removeAll(distDir); err != nil {
		return fmt.Errorf("error: clean dist dir: %w", err)
	}
	if err := mkdirAll(distDir, 0o755); err != nil {
		return fmt.Errorf("error: create dist dir: %w", err)
	}
	return nil
}

func findBuildArtifacts(buildDir string) ([]buildArtifact, error) {
	patterns := []string{
		filepath.Join(buildDir, "*", "*", executableName),
		filepath.Join(buildDir, "*", "*", executableName+".exe"),
	}

	artifactByPath := make(map[string]buildArtifact)
	for _, pattern := range patterns {
		matches, err := glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("error: invalid build glob %q: %w", pattern, err)
		}
		for _, match := range matches {
			info, err := statFile(match)
			if err != nil {
				return nil, fmt.Errorf("error: stat build output %s: %w", match, err)
			}
			if !info.Mode().IsRegular() {
				continue
			}
			artifact, err := buildArtifactFromPath(match)
			if err != nil {
				return nil, err
			}
			artifactByPath[match] = artifact
		}
	}

	if len(artifactByPath) == 0 {
		return nil, fmt.Errorf("error: no build outputs found in %s", buildDir)
	}

	artifacts := make([]buildArtifact, 0, len(artifactByPath))
	for _, artifact := range artifactByPath {
		artifacts = append(artifacts, artifact)
	}

	sort.Slice(artifacts, func(left, right int) bool {
		if artifacts[left].goos == artifacts[right].goos {
			return artifacts[left].goarch < artifacts[right].goarch
		}
		return artifacts[left].goos < artifacts[right].goos
	})

	return artifacts, nil
}

func findMetadataFiles(buildDir string) ([]string, error) {
	metadataDir := filepath.Join(buildDir, metadataDirName)
	metadataFiles := []string{
		filepath.Join(metadataDir, noticesFileName),
		filepath.Join(metadataDir, sbomFileName),
	}

	for _, metadataPath := range metadataFiles {
		info, err := statFile(metadataPath)
		if err != nil {
			return nil, fmt.Errorf("error: stat metadata file %s: %w", metadataPath, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("error: metadata file is not a file: %s", metadataPath)
		}
	}

	return metadataFiles, nil
}

func copyMetadataFiles(distDir string, metadataFiles []string) error {
	metadataDistDir := filepath.Join(distDir, metadataDirName)
	if err := mkdirAll(metadataDistDir, 0o755); err != nil {
		return fmt.Errorf("error: create metadata dir: %w", err)
	}

	for _, metadataPath := range metadataFiles {
		if err := copyMetadataFile(metadataDistDir, metadataPath); err != nil {
			return err
		}
	}

	return nil
}

func copyMetadataFile(distDir, inputPath string) (returnErr error) {
	inputInfo, err := statFile(inputPath)
	if err != nil {
		return fmt.Errorf("error: stat metadata file %s: %w", inputPath, err)
	}
	if !inputInfo.Mode().IsRegular() {
		return fmt.Errorf("error: metadata file is not a file: %s", inputPath)
	}

	inputFile, err := openFile(inputPath)
	if err != nil {
		return fmt.Errorf("error: open metadata file %s: %w", inputPath, err)
	}
	defer func() {
		if closeErr := closeInputFile(inputFile); closeErr != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("error: close metadata file %s: %w", inputPath, closeErr))
		}
	}()

	outputPath := filepath.Join(distDir, filepath.Base(inputPath))
	outputFile, err := createFile(outputPath, inputInfo.Mode())
	if err != nil {
		return fmt.Errorf("error: create metadata file %s: %w", outputPath, err)
	}
	defer func() {
		if closeErr := closeOutputFile(outputFile); closeErr != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("error: close metadata output %s: %w", outputPath, closeErr))
		}
	}()

	if _, err := copyFile(outputFile, inputFile); err != nil {
		return fmt.Errorf("error: copy metadata file %s: %w", inputPath, err)
	}

	return nil
}

func buildArtifactFromPath(path string) (buildArtifact, error) {
	archDir := filepath.Base(filepath.Dir(path))
	osDir := filepath.Base(filepath.Dir(filepath.Dir(path)))
	if archDir == "." || archDir == string(filepath.Separator) || osDir == "." || osDir == string(filepath.Separator) {
		return buildArtifact{}, fmt.Errorf("error: unexpected build output path %s", path)
	}
	if osDir == "build" {
		return buildArtifact{}, fmt.Errorf("error: missing os/arch in build output path %s", path)
	}

	return buildArtifact{
		goos:   osDir,
		goarch: archDir,
		path:   path,
	}, nil
}

func buildZipHeader(inputInfo os.FileInfo, inputPath string) (*zip.FileHeader, error) {
	header, err := zipFileInfoHeader(inputInfo)
	if err != nil {
		return nil, fmt.Errorf("error: create zip header for %s: %w", inputPath, err)
	}
	header.Name = filepath.Base(inputPath)
	header.Method = zip.Deflate
	header.SetMode(inputInfo.Mode())
	return header, nil
}

func writeZipContents(zipEntryWriter io.Writer, inputPath string) (returnErr error) {
	inputFile, err := openFile(inputPath)
	if err != nil {
		return fmt.Errorf("error: open build output %s: %w", inputPath, err)
	}
	defer func() {
		if closeErr := closeInputFile(inputFile); closeErr != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("error: close build output %s: %w", inputPath, closeErr))
		}
	}()

	if _, err := copyFile(zipEntryWriter, inputFile); err != nil {
		return fmt.Errorf("error: write zip contents for %s: %w", inputPath, err)
	}

	return nil
}

func writeZip(outputPath, inputPath string) (returnErr error) {
	//nolint:gosec // output path is rooted in dist dir with a sanitized version string.
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error: create zip %s: %w", outputPath, err)
	}
	defer func() {
		if closeErr := closeOutputFile(outputFile); closeErr != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("error: close zip output %s: %w", outputPath, closeErr))
		}
	}()

	zipWriter := zip.NewWriter(outputFile)
	defer func() {
		if closeErr := closeZipWriter(zipWriter); closeErr != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("error: close zip writer %s: %w", outputPath, closeErr))
		}
	}()

	inputInfo, err := statFile(inputPath)
	if err != nil {
		return fmt.Errorf("error: stat build output %s: %w", inputPath, err)
	}
	if !inputInfo.Mode().IsRegular() {
		return fmt.Errorf("error: build output is not a file: %s", inputPath)
	}

	header, err := buildZipHeader(inputInfo, inputPath)
	if err != nil {
		return err
	}

	zipEntryWriter, err := zipCreateHeader(zipWriter, header)
	if err != nil {
		return fmt.Errorf("error: write zip header for %s: %w", inputPath, err)
	}

	return writeZipContents(zipEntryWriter, inputPath)
}

func findRepoRoot(startDir string) (string, error) {
	current := startDir
	for {
		if hasGoMod(current) {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", errors.New("error: failed to locate repo root (missing go.mod); run from repo root")
		}
		current = parent
	}
}

func hasGoMod(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil
}
