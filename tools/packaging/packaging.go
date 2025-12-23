package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const executableName = "mmm"

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
var glob = filepath.Glob
var statFile = os.Stat

func main() {
	exit(runMain())
}

func runMain() int {
	versionFlag := flag.String("version", "dev", "version suffix for dist artifacts")
	flag.Parse()

	tool, err := newDistToolFunc()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if err := tool.run(*versionFlag); err != nil {
		fmt.Fprintln(os.Stderr, err)
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

	if err := resetDistDir(distDir); err != nil {
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

func writeZip(outputPath, inputPath string) error {
	inputInfo, err := os.Stat(inputPath)
	if err != nil {
		return fmt.Errorf("error: stat build output %s: %w", inputPath, err)
	}
	if !inputInfo.Mode().IsRegular() {
		return fmt.Errorf("error: build output is not a file: %s", inputPath)
	}

	// #nosec G304 -- output path is rooted in dist dir with a sanitized version string.
	outputFile, err := os.Create(outputPath) // #nosec G304 -- output path is rooted in dist dir with a sanitized version string.
	if err != nil {
		return fmt.Errorf("error: create zip %s: %w", outputPath, err)
	}
	defer func() {
		_ = outputFile.Close() // #nosec G104 -- best-effort cleanup for zip output.
	}()

	zipWriter := zip.NewWriter(outputFile)
	defer func() {
		_ = zipWriter.Close() // #nosec G104 -- best-effort cleanup for zip writer.
	}()

	header, err := zipFileInfoHeader(inputInfo)
	if err != nil {
		return fmt.Errorf("error: create zip header for %s: %w", inputPath, err)
	}
	header.Name = filepath.Base(inputPath)
	header.Method = zip.Deflate
	header.SetMode(inputInfo.Mode())

	zipEntryWriter, err := zipCreateHeader(zipWriter, header)
	if err != nil {
		return fmt.Errorf("error: write zip header for %s: %w", inputPath, err)
	}

	inputFile, err := openFile(inputPath)
	if err != nil {
		return fmt.Errorf("error: open build output %s: %w", inputPath, err)
	}
	defer func() {
		_ = inputFile.Close() // #nosec G104 -- best-effort cleanup for zip input file.
	}()

	if _, err := copyFile(zipEntryWriter, inputFile); err != nil {
		return fmt.Errorf("error: write zip contents for %s: %w", inputPath, err)
	}

	return nil
}

func findRepoRoot(startDir string) (string, error) {
	current := startDir
	for {
		if hasGoMod(current) {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("error: failed to locate repo root (missing go.mod); run from repo root")
		}
		current = parent
	}
}

func hasGoMod(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil
}
