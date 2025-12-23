// Package main provides the coverage helper binary.
package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	coverageFuncOutputName = "coverage.out"
	coverageHTMLName       = "coverage.html"
	coverageProfileName    = "coverage.profile"
)

var excludedPathFragments []string

type commandRunner interface {
	Run(*exec.Cmd) error
}

type commandOutputRunner interface {
	Output(*exec.Cmd) ([]byte, error)
}

type logger interface {
	Printf(format string, args ...any)
}

type execRunner struct{}

func (execRunner) Run(command *exec.Cmd) error {
	return command.Run()
}

type execOutputRunner struct{}

func (execOutputRunner) Output(command *exec.Cmd) ([]byte, error) {
	return command.Output()
}

type coverageTool struct {
	repoRoot      string
	goBinary      string
	commandRunner commandRunner
	commandOutput commandOutputRunner
	logger        logger
}

var getWorkingDirectory = os.Getwd
var newCoverageToolFunc = newCoverageTool
var exit = os.Exit
var closeFile = func(file *os.File) error { return file.Close() }
var writeFile = os.WriteFile

func main() {
	exit(runMain())
}

func runMain() int {
	tool, err := newCoverageToolFunc()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if err := tool.run(); err != nil {
		var coverageError coverageNotFullError
		if !errors.As(err, &coverageError) {
			fmt.Fprintln(os.Stderr, err)
		}
		return 1
	}
	return 0
}

func newCoverageTool() (*coverageTool, error) {
	workingDirectory, err := getWorkingDirectory()
	if err != nil {
		return nil, fmt.Errorf("error: failed to determine working directory: %w", err)
	}

	repoRoot, err := findRepoRoot(workingDirectory)
	if err != nil {
		return nil, err
	}

	return &coverageTool{
		repoRoot:      repoRoot,
		goBinary:      "go",
		commandRunner: execRunner{},
		commandOutput: execOutputRunner{},
		logger:        log.New(os.Stdout, "coverage: ", 0),
	}, nil
}

func (tool *coverageTool) run() error {
	coveragePath := filepath.Join(tool.repoRoot, coverageProfileName)
	htmlPath := filepath.Join(tool.repoRoot, coverageHTMLName)
	funcOutputPath := filepath.Join(tool.repoRoot, coverageFuncOutputName)

	defer func() {
		_ = os.Remove(coveragePath) // #nosec G104 -- best-effort cleanup of temporary coverage profile.
	}()

	if err := tool.runCoverageTests(coveragePath); err != nil {
		return err
	}

	filteredCoveragePath := coveragePath
	if len(excludedPathFragments) > 0 {
		filteredFile, err := os.CreateTemp(tool.repoRoot, "coverage-filtered-*.out")
		if err != nil {
			return fmt.Errorf("error: create filtered coverage file: %w", err)
		}
		if err := closeFile(filteredFile); err != nil {
			_ = filteredFile.Close() // best-effort release for Windows temp cleanup.
			return fmt.Errorf("error: finalize filtered coverage file: %w", err)
		}
		filteredCoveragePath = filteredFile.Name()
		defer func() {
			_ = os.Remove(filteredCoveragePath) // #nosec G104 -- best-effort cleanup of filtered coverage output.
		}()
		if err := filterCoverageFile(coveragePath, filteredCoveragePath, excludedPathFragments); err != nil {
			return err
		}
	}

	if err := tool.generateCoverageHTML(filteredCoveragePath, htmlPath); err != nil {
		return err
	}

	funcOutput, totalLine, totalCoverage, offendingLines, err := tool.coverageFuncOutput(filteredCoveragePath)
	if err != nil {
		return err
	}

	if err := writeFile(funcOutputPath, funcOutput, 0o644); err != nil {
		return fmt.Errorf("error: write coverage output: %w", err)
	}

	if len(offendingLines) > 0 {
		for _, line := range offendingLines {
			fmt.Println(line)
		}
		fmt.Println(totalLine)
		printCoverageHints(funcOutputPath, htmlPath)
		return coverageNotFullError{coverage: totalCoverage}
	}

	fmt.Println("coverage is 100%")
	printCoverageHints(funcOutputPath, htmlPath)
	return nil
}

type coverageNotFullError struct {
	coverage string
}

func (err coverageNotFullError) Error() string {
	return fmt.Sprintf("coverage is not 100%%: %s", err.coverage)
}

func (tool *coverageTool) runCoverageTests(coveragePath string) error {
	// #nosec G204 -- go binary and args are controlled by this tool.
	command := exec.Command(tool.goBinary, "test", "./...", "-coverprofile", coveragePath) // #nosec G204 -- go binary and args are controlled by this tool.
	command.Dir = tool.repoRoot
	outputBuffer := &bytes.Buffer{}
	command.Stdout = outputBuffer
	command.Stderr = outputBuffer
	if err := tool.commandRunner.Run(command); err != nil {
		if _, writeErr := os.Stderr.Write(outputBuffer.Bytes()); writeErr != nil {
			return fmt.Errorf("error: coverage tests failed with unreadable output: %w", err)
		}
		return fmt.Errorf("error: coverage tests failed: %w", err)
	}
	return nil
}

func (tool *coverageTool) generateCoverageHTML(coveragePath, htmlPath string) error {
	// #nosec G204 -- go binary and args are controlled by this tool.
	command := exec.Command(tool.goBinary, "tool", "cover", "-html", coveragePath, "-o", htmlPath) // #nosec G204 -- go binary and args are controlled by this tool.
	command.Dir = tool.repoRoot
	if err := tool.commandRunner.Run(command); err != nil {
		return fmt.Errorf("error: coverage html generation failed: %w", err)
	}
	return nil
}

func (tool *coverageTool) coverageFuncOutput(coveragePath string) ([]byte, string, string, []string, error) {
	// #nosec G204 -- go binary and args are controlled by this tool.
	command := exec.Command(tool.goBinary, "tool", "cover", "-func", coveragePath) // #nosec G204 -- go binary and args are controlled by this tool.
	command.Dir = tool.repoRoot
	output, err := tool.commandOutput.Output(command)
	if err != nil {
		return nil, "", "", nil, fmt.Errorf("error: coverage summary failed: %w", err)
	}

	totalLine, coverage, err := parseTotalCoverage(bytes.NewReader(output))
	if err != nil {
		return nil, "", "", nil, err
	}

	offendingLines, err := filterOffendingCoverageLines(bytes.NewReader(output))
	if err != nil {
		return nil, "", "", nil, err
	}

	return output, totalLine, coverage, offendingLines, nil
}

func parseTotalCoverage(reader io.Reader) (string, string, error) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "total:") {
			fields := strings.Fields(line)
			if len(fields) < 3 {
				return "", "", errors.New("error: malformed total line in coverage output")
			}
			return line, fields[2], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("error: reading coverage output: %w", err)
	}
	return "", "", errors.New("error: no total line found in coverage output")
}

func filterCoverageFile(sourcePath, filteredPath string, exclusions []string) error {
	inputFile, err := os.Open(sourcePath) // #nosec G304 -- paths are derived from repo-root coverage output.
	if err != nil {
		return fmt.Errorf("error: open coverage profile: %w", err)
	}
	defer func() {
		_ = inputFile.Close() // #nosec G104 -- best-effort cleanup for read-only coverage input.
	}()

	outputFile, err := os.Create(filteredPath) // #nosec G304 -- filtered output path is created by this tool in repo root.
	if err != nil {
		return fmt.Errorf("error: create filtered coverage file: %w", err)
	}
	defer func() {
		_ = outputFile.Close() // #nosec G104 -- best-effort cleanup for filtered coverage output.
	}()

	if err := filterCoverageContent(inputFile, outputFile, exclusions); err != nil {
		return err
	}

	return nil
}

func filterCoverageContent(reader io.Reader, writer io.Writer, exclusions []string) error {
	scanner := bufio.NewScanner(reader)
	writerBuffer := bufio.NewWriter(writer)

	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if strings.HasPrefix(line, "mode:") {
			if _, err := writerBuffer.WriteString(line + "\n"); err != nil {
				return fmt.Errorf("error: write coverage mode line: %w", err)
			}
			continue
		}

		filePath, err := coverageLinePath(line)
		if err != nil {
			return fmt.Errorf("error: parse coverage line %d: %w", lineNumber, err)
		}
		if shouldExcludePath(filePath, exclusions) {
			continue
		}
		if _, err := writerBuffer.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("error: write coverage line: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error: read coverage profile: %w", err)
	}
	return writerBuffer.Flush()
}

func coverageLinePath(line string) (string, error) {
	separatorIndex := strings.LastIndex(line, ":")
	if separatorIndex == -1 {
		return "", errors.New("missing path separator")
	}
	return line[:separatorIndex], nil
}

func shouldExcludePath(path string, exclusions []string) bool {
	for _, fragment := range exclusions {
		normalizedPath := strings.ReplaceAll(path, "\\", "/")
		normalizedFragment := strings.ReplaceAll(fragment, "\\", "/")
		if strings.Contains(normalizedPath, normalizedFragment) {
			return true
		}
	}
	return false
}

func filterOffendingCoverageLines(reader io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(reader)
	offending := make([]string, 0)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "total:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		coverage := fields[len(fields)-1]
		if !strings.HasSuffix(coverage, "%") {
			return nil, fmt.Errorf("error: malformed coverage line: %s", line)
		}
		if coverage != "100.0%" {
			offending = append(offending, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error: read coverage output: %w", err)
	}
	return offending, nil
}

func printCoverageHints(outPath, htmlPath string) {
	fmt.Printf("details: %s and %s were generated\n", outPath, htmlPath)
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
