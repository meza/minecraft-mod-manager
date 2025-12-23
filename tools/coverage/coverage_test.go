package main

import (
	"bytes"
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTotalCoverage(t *testing.T) {
	input := "example\t0.0%\n" +
		"total:\t(statements)\t100.0%\n"
	totalLine, coverage, err := parseTotalCoverage(strings.NewReader(input))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if coverage != "100.0%" {
		t.Fatalf("expected 100.0%%, got %q", coverage)
	}
	if totalLine == "" {
		t.Fatal("expected total line")
	}
}

func TestParseTotalCoverageMalformedLine(t *testing.T) {
	input := "total:\t(statements)\n"
	_, _, err := parseTotalCoverage(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseTotalCoverageMissingTotal(t *testing.T) {
	_, _, err := parseTotalCoverage(strings.NewReader("example\t0.0%\n"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseTotalCoverageReadError(t *testing.T) {
	if _, _, err := parseTotalCoverage(errorReader{}); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunMainGetwdFailure(t *testing.T) {
	originalNewCoverageToolFunc := newCoverageToolFunc
	t.Cleanup(func() {
		newCoverageToolFunc = originalNewCoverageToolFunc
	})

	newCoverageToolFunc = func() (*coverageTool, error) {
		return nil, os.ErrNotExist
	}

	output := captureStderr(t, func() {
		if exitCode := runMain(); exitCode != 1 {
			t.Fatalf("expected exit code 1, got %d", exitCode)
		}
	})
	if !strings.Contains(output, "file does not exist") {
		t.Fatalf("expected stderr to include error, got %q", output)
	}
}

func TestRunMainToolRunError(t *testing.T) {
	originalNewCoverageToolFunc := newCoverageToolFunc
	t.Cleanup(func() {
		newCoverageToolFunc = originalNewCoverageToolFunc
	})

	tempDir := t.TempDir()
	coverageProfilePath := filepath.Join(tempDir, coverageProfileName)
	if err := os.WriteFile(coverageProfilePath, []byte("mode: set\n"), 0o644); err != nil {
		t.Fatalf("failed to write coverage profile: %v", err)
	}

	newCoverageToolFunc = func() (*coverageTool, error) {
		return &coverageTool{
			repoRoot:      tempDir,
			goBinary:      "go",
			commandRunner: &recordingRunner{},
			commandOutput: outputRunner{
				output: []byte("github.com/meza/minecraft-mod-manager/internal/config/config.go:1.2 3.4 1 99.0%\n" +
					"total:\t(statements)\t99.0%\n"),
			},
			logger: log.New(&bytes.Buffer{}, "coverage: ", 0),
		}, nil
	}

	if exitCode := runMain(); exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestRunMainToolRunUnexpectedErrorWritesStderr(t *testing.T) {
	originalNewCoverageToolFunc := newCoverageToolFunc
	t.Cleanup(func() {
		newCoverageToolFunc = originalNewCoverageToolFunc
	})

	tempDir := t.TempDir()
	coverageProfilePath := filepath.Join(tempDir, coverageProfileName)
	if err := os.WriteFile(coverageProfilePath, []byte("mode: set\n"), 0o644); err != nil {
		t.Fatalf("failed to write coverage profile: %v", err)
	}

	newCoverageToolFunc = func() (*coverageTool, error) {
		return &coverageTool{
			repoRoot: tempDir,
			goBinary: "go",
			commandRunner: selectiveRunner{
				failArgs: []string{"tool", "cover", "-html"},
				err:      errors.New("html fail"),
			},
			commandOutput: outputRunner{
				output: []byte("total:\t(statements)\t100.0%\n"),
			},
			logger: log.New(&bytes.Buffer{}, "coverage: ", 0),
		}, nil
	}

	output := captureStderr(t, func() {
		if exitCode := runMain(); exitCode != 1 {
			t.Fatalf("expected exit code 1, got %d", exitCode)
		}
	})
	if !strings.Contains(output, "coverage html generation failed") {
		t.Fatalf("expected stderr to include html error, got %q", output)
	}
}

func TestRunMainFindRepoRootFailureWritesError(t *testing.T) {
	originalNewCoverageToolFunc := newCoverageToolFunc
	originalGetWorkingDirectory := getWorkingDirectory
	t.Cleanup(func() {
		newCoverageToolFunc = originalNewCoverageToolFunc
		getWorkingDirectory = originalGetWorkingDirectory
	})

	newCoverageToolFunc = newCoverageTool
	getWorkingDirectory = func() (string, error) {
		return t.TempDir(), nil
	}

	output := captureStderr(t, func() {
		if exitCode := runMain(); exitCode != 1 {
			t.Fatalf("expected exit code 1, got %d", exitCode)
		}
	})
	if !strings.Contains(output, "failed to locate repo root") {
		t.Fatalf("expected stderr to include repo root error, got %q", output)
	}
}

func TestNewCoverageToolGetwdFailure(t *testing.T) {
	originalGetWorkingDirectory := getWorkingDirectory
	t.Cleanup(func() {
		getWorkingDirectory = originalGetWorkingDirectory
	})

	getWorkingDirectory = func() (string, error) {
		return "", errors.New("boom")
	}

	tool, err := newCoverageTool()
	if err == nil {
		t.Fatalf("expected error, got nil and tool %v", tool)
	}
}

func TestNewCoverageToolFindsRepoRoot(t *testing.T) {
	originalGetWorkingDirectory := getWorkingDirectory
	t.Cleanup(func() {
		getWorkingDirectory = originalGetWorkingDirectory
	})

	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	nestedDir := filepath.Join(repoRoot, "tools", "coverage")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/test"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	getWorkingDirectory = func() (string, error) {
		return nestedDir, nil
	}

	tool, err := newCoverageTool()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tool.repoRoot != repoRoot {
		t.Fatalf("expected repo root %q, got %q", repoRoot, tool.repoRoot)
	}
}

func TestNewCoverageToolMissingRepoRoot(t *testing.T) {
	originalGetWorkingDirectory := getWorkingDirectory
	t.Cleanup(func() {
		getWorkingDirectory = originalGetWorkingDirectory
	})

	tempDir := t.TempDir()
	getWorkingDirectory = func() (string, error) {
		return tempDir, nil
	}

	tool, err := newCoverageTool()
	if err == nil {
		t.Fatalf("expected error, got nil and tool %v", tool)
	}
}

func TestCoverageLinePath(t *testing.T) {
	path, err := coverageLinePath("github.com/meza/minecraft-mod-manager/internal/foo.go:1.2 3.4 1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if path != "github.com/meza/minecraft-mod-manager/internal/foo.go" {
		t.Fatalf("unexpected path %q", path)
	}
}

func TestCoverageLinePathWindowsDrive(t *testing.T) {
	path, err := coverageLinePath(`C:\repo\internal\foo.go:1.2 3.4 1`)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if path != `C:\repo\internal\foo.go` {
		t.Fatalf("unexpected path %q", path)
	}
}

func TestCoverageLinePathMissingSeparator(t *testing.T) {
	_, err := coverageLinePath("missing-separator")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestShouldExcludePath(t *testing.T) {
	if !shouldExcludePath("github.com/meza/minecraft-mod-manager/tools/build/build.go", []string{"/tools/"}) {
		t.Fatal("expected tools path to be excluded")
	}
	if shouldExcludePath("github.com/meza/minecraft-mod-manager/internal/config/config.go", []string{"/tools/"}) {
		t.Fatal("expected non-tools path to be kept")
	}
}

func TestShouldExcludePathWindows(t *testing.T) {
	if !shouldExcludePath(`C:\repo\tools\coverage\coverage.go`, []string{"/tools/"}) {
		t.Fatal("expected windows tools path to be excluded")
	}
}

func TestFilterCoverageContent(t *testing.T) {
	input := strings.Join([]string{
		"mode: set",
		"github.com/meza/minecraft-mod-manager/internal/config/config.go:1.2 3.4 1",
		"github.com/meza/minecraft-mod-manager/tools/build/build.go:1.2 3.4 1",
	}, "\n") + "\n"

	var output bytes.Buffer
	if err := filterCoverageContent(strings.NewReader(input), &output, []string{"/tools/"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := strings.Join([]string{
		"mode: set",
		"github.com/meza/minecraft-mod-manager/internal/config/config.go:1.2 3.4 1",
		"",
	}, "\n")
	if output.String() != expected {
		t.Fatalf("unexpected output: %q", output.String())
	}
}

func TestFilterCoverageContentMalformedLine(t *testing.T) {
	input := "mode: set\nmissing-separator\n"
	var output bytes.Buffer
	if err := filterCoverageContent(strings.NewReader(input), &output, []string{"/tools/"}); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFilterCoverageFileWritesFilteredOutput(t *testing.T) {
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "coverage.profile")
	filteredPath := filepath.Join(tempDir, "coverage.filtered")

	input := strings.Join([]string{
		"mode: set",
		"github.com/meza/minecraft-mod-manager/internal/config/config.go:1.2 3.4 1",
		"github.com/meza/minecraft-mod-manager/tools/build/build.go:1.2 3.4 1",
	}, "\n") + "\n"

	if err := os.WriteFile(sourcePath, []byte(input), 0o644); err != nil {
		t.Fatalf("failed to write source profile: %v", err)
	}

	if err := filterCoverageFile(sourcePath, filteredPath, []string{"/tools/"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// #nosec G304 -- test reads temp file path.
	output, err := os.ReadFile(filteredPath) // #nosec G304 -- test reads temp file path.
	if err != nil {
		t.Fatalf("failed to read filtered profile: %v", err)
	}

	expected := strings.Join([]string{
		"mode: set",
		"github.com/meza/minecraft-mod-manager/internal/config/config.go:1.2 3.4 1",
		"",
	}, "\n")
	if string(output) != expected {
		t.Fatalf("unexpected output: %q", string(output))
	}
}

func TestFilterCoverageFileOpenError(t *testing.T) {
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "missing.profile")
	filteredPath := filepath.Join(tempDir, "coverage.filtered")
	if err := filterCoverageFile(sourcePath, filteredPath, []string{"/tools/"}); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFilterCoverageFileCreateError(t *testing.T) {
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "coverage.profile")
	if err := os.WriteFile(sourcePath, []byte("mode: set\n"), 0o644); err != nil {
		t.Fatalf("failed to write source profile: %v", err)
	}
	filteredPath := filepath.Join(tempDir, "missing-dir", "coverage.filtered")
	if err := filterCoverageFile(sourcePath, filteredPath, []string{"/tools/"}); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFilterCoverageFileContentError(t *testing.T) {
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "coverage.profile")
	if err := os.WriteFile(sourcePath, []byte("mode: set\nmissing-separator\n"), 0o644); err != nil {
		t.Fatalf("failed to write source profile: %v", err)
	}
	filteredPath := filepath.Join(tempDir, "coverage.filtered")
	if err := filterCoverageFile(sourcePath, filteredPath, []string{"/tools/"}); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFilterCoverageContentWriteError(t *testing.T) {
	input := "mode: set\n"
	if err := filterCoverageContent(strings.NewReader(input), errorWriter{}, []string{"/tools/"}); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFilterCoverageContentReadError(t *testing.T) {
	if err := filterCoverageContent(errorReader{}, &bytes.Buffer{}, []string{"/tools/"}); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFilterCoverageContentWriteStringError(t *testing.T) {
	longLine := "mode: " + strings.Repeat("a", 5000) + "\n"
	if err := filterCoverageContent(strings.NewReader(longLine), errorWriter{}, []string{"/tools/"}); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFilterCoverageContentWriteStringErrorNonMode(t *testing.T) {
	longLine := "github.com/meza/minecraft-mod-manager/internal/config/config.go:" +
		strings.Repeat("1", 5000) + " 3.4 1\n"
	if err := filterCoverageContent(strings.NewReader(longLine), errorWriter{}, []string{"/tools/"}); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFilterOffendingCoverageLines(t *testing.T) {
	input := strings.Join([]string{
		"github.com/meza/minecraft-mod-manager/internal/config/config.go:1.2 3.4 1 100.0%",
		"github.com/meza/minecraft-mod-manager/tools/build/build.go:1.2 3.4 1 75.0%",
		"total:\t(statements)\t97.3%",
	}, "\n") + "\n"

	lines, err := filterOffendingCoverageLines(strings.NewReader(input))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(lines) != 1 {
		t.Fatalf("expected 1 offending line, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "tools/build/build.go") {
		t.Fatalf("unexpected offending line %q", lines[0])
	}
}

func TestFilterOffendingCoverageLinesSkipsNonPercent(t *testing.T) {
	input := "github.com/meza/minecraft-mod-manager/internal/config/config.go:1.2 3.4 1 missing\n"
	if _, err := filterOffendingCoverageLines(strings.NewReader(input)); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFilterOffendingCoverageLinesSkipsEmpty(t *testing.T) {
	lines, err := filterOffendingCoverageLines(strings.NewReader("\n"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(lines) != 0 {
		t.Fatalf("expected no offending lines, got %d", len(lines))
	}
}

func TestFilterOffendingCoverageLinesReadError(t *testing.T) {
	if _, err := filterOffendingCoverageLines(errorReader{}); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCoverageFuncOutputParsesResults(t *testing.T) {
	output := strings.Join([]string{
		"github.com/meza/minecraft-mod-manager/internal/config/config.go:1.2 3.4 1 100.0%",
		"github.com/meza/minecraft-mod-manager/tools/build/build.go:1.2 3.4 1 75.0%",
		"total:\t(statements)\t97.3%",
	}, "\n") + "\n"

	tool := &coverageTool{
		repoRoot: "unused",
		goBinary: "go",
		commandOutput: outputRunner{
			output: []byte(output),
		},
	}

	funcOutput, totalLine, total, offending, err := tool.coverageFuncOutput("coverage.profile")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != "97.3%" {
		t.Fatalf("unexpected total %q", total)
	}
	if len(offending) != 1 {
		t.Fatalf("expected 1 offending line, got %d", len(offending))
	}
	if !strings.Contains(offending[0], "tools/build/build.go") {
		t.Fatalf("unexpected offending line %q", offending[0])
	}
	if string(funcOutput) != output {
		t.Fatalf("unexpected func output: %q", string(funcOutput))
	}
	if totalLine == "" {
		t.Fatal("expected total line")
	}
}

func TestCoverageFuncOutputMissingTotal(t *testing.T) {
	tool := &coverageTool{
		repoRoot: "unused",
		goBinary: "go",
		commandOutput: outputRunner{
			output: []byte("example\t0.0%\n"),
		},
	}

	if _, _, _, _, err := tool.coverageFuncOutput("coverage.profile"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCoverageFuncOutputMalformedLine(t *testing.T) {
	tool := &coverageTool{
		repoRoot: "unused",
		goBinary: "go",
		commandOutput: outputRunner{
			output: []byte("github.com/meza/minecraft-mod-manager/internal/config/config.go:1.2 3.4 1 missing\n" +
				"total:\t(statements)\t100.0%\n"),
		},
	}

	if _, _, _, _, err := tool.coverageFuncOutput("coverage.profile"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCoverageFuncOutputError(t *testing.T) {
	tool := &coverageTool{
		repoRoot: "unused",
		goBinary: "go",
		commandOutput: outputRunner{
			err: os.ErrNotExist,
		},
	}

	if _, _, _, _, err := tool.coverageFuncOutput("coverage.profile"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestMainSuccessUsesExit(t *testing.T) {
	originalNewCoverageToolFunc := newCoverageToolFunc
	originalExit := exit
	t.Cleanup(func() {
		newCoverageToolFunc = originalNewCoverageToolFunc
		exit = originalExit
	})

	tempDir := t.TempDir()
	coverageProfilePath := filepath.Join(tempDir, coverageProfileName)
	if err := os.WriteFile(coverageProfilePath, []byte("mode: set\n"), 0o644); err != nil {
		t.Fatalf("failed to write coverage profile: %v", err)
	}

	newCoverageToolFunc = func() (*coverageTool, error) {
		return &coverageTool{
			repoRoot: tempDir,
			goBinary: "go",
			commandRunner: &recordingRunner{
				commands: []*exec.Cmd{},
			},
			commandOutput: outputRunner{
				output: []byte("total:\t(statements)\t100.0%\n"),
			},
			logger: log.New(&bytes.Buffer{}, "coverage: ", 0),
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

func TestCoverageToolRunWithExclusions(t *testing.T) {
	originalExclusions := excludedPathFragments
	t.Cleanup(func() {
		excludedPathFragments = originalExclusions
	})
	excludedPathFragments = []string{"/tools/"}

	tempDir := t.TempDir()
	coverageProfilePath := filepath.Join(tempDir, coverageProfileName)
	content := strings.Join([]string{
		"mode: set",
		"github.com/meza/minecraft-mod-manager/internal/config/config.go:1.2 3.4 1",
		"github.com/meza/minecraft-mod-manager/tools/build/build.go:1.2 3.4 1",
		"",
	}, "\n")
	if err := os.WriteFile(coverageProfilePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write coverage profile: %v", err)
	}

	tool := &coverageTool{
		repoRoot:      tempDir,
		goBinary:      "go",
		commandRunner: &recordingRunner{},
		commandOutput: outputRunner{
			output: []byte("total:\t(statements)\t100.0%\n"),
		},
		logger: log.New(&bytes.Buffer{}, "coverage: ", 0),
	}

	if err := tool.run(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(tempDir, coverageFuncOutputName)); err != nil {
		t.Fatalf("expected coverage output to exist: %v", err)
	}
}

func TestCoverageNotFullErrorMessage(t *testing.T) {
	coverageError := coverageNotFullError{coverage: "89.0%"}
	if coverageError.Error() != "coverage is not 100%: 89.0%" {
		t.Fatalf("unexpected error message: %q", coverageError.Error())
	}
}

func TestCoverageToolRunCoverageTestsError(t *testing.T) {
	tool := &coverageTool{
		repoRoot:      t.TempDir(),
		goBinary:      "go",
		commandRunner: errorRunner{err: errors.New("tests failed")},
		commandOutput: outputRunner{
			output: []byte("total:\t(statements)\t100.0%\n"),
		},
		logger: log.New(&bytes.Buffer{}, "coverage: ", 0),
	}

	if err := tool.run(); err == nil {
		t.Fatal("expected error, got nil")
	}
}
func TestCoverageToolRunCreateFilteredFileError(t *testing.T) {
	originalExclusions := excludedPathFragments
	t.Cleanup(func() {
		excludedPathFragments = originalExclusions
	})
	excludedPathFragments = []string{"/tools/"}

	tempDir := t.TempDir()
	tool := &coverageTool{
		repoRoot:      filepath.Join(tempDir, "missing-root"),
		goBinary:      "go",
		commandRunner: &recordingRunner{},
		commandOutput: outputRunner{
			output: []byte("total:\t(statements)\t100.0%\n"),
		},
		logger: log.New(&bytes.Buffer{}, "coverage: ", 0),
	}

	if err := tool.run(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCoverageToolRunOutputFailure(t *testing.T) {
	tempDir := t.TempDir()
	coverageProfilePath := filepath.Join(tempDir, coverageProfileName)
	if err := os.WriteFile(coverageProfilePath, []byte("mode: set\n"), 0o644); err != nil {
		t.Fatalf("failed to write coverage profile: %v", err)
	}

	tool := &coverageTool{
		repoRoot:      tempDir,
		goBinary:      "go",
		commandRunner: &recordingRunner{},
		commandOutput: outputRunner{
			output: []byte("github.com/meza/minecraft-mod-manager/tools/build/build.go:1.2 3.4 1 90.0%\n" +
				"total:\t(statements)\t99.0%\n"),
		},
		logger: log.New(&bytes.Buffer{}, "coverage: ", 0),
	}

	output := captureStdout(t, func() {
		if err := tool.run(); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	if !strings.Contains(output, "tools/build/build.go") {
		t.Fatalf("expected offending line, got %q", output)
	}
	if !strings.Contains(output, "total:\t(statements)\t99.0%") {
		t.Fatalf("expected total line, got %q", output)
	}
	if strings.Contains(output, "coverage is 100%") {
		t.Fatalf("unexpected success line in output: %q", output)
	}
}

func TestCoverageToolRunOutputSuccess(t *testing.T) {
	tempDir := t.TempDir()
	coverageProfilePath := filepath.Join(tempDir, coverageProfileName)
	if err := os.WriteFile(coverageProfilePath, []byte("mode: set\n"), 0o644); err != nil {
		t.Fatalf("failed to write coverage profile: %v", err)
	}

	tool := &coverageTool{
		repoRoot:      tempDir,
		goBinary:      "go",
		commandRunner: &recordingRunner{},
		commandOutput: outputRunner{
			output: []byte("total:\t(statements)\t100.0%\n"),
		},
		logger: log.New(&bytes.Buffer{}, "coverage: ", 0),
	}

	output := captureStdout(t, func() {
		if err := tool.run(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	if !strings.Contains(output, "coverage is 100%") {
		t.Fatalf("expected success line, got %q", output)
	}
	if !strings.Contains(output, "coverage.out") {
		t.Fatalf("expected details line, got %q", output)
	}
}

func TestCoverageToolRunFilterCoverageError(t *testing.T) {
	originalExclusions := excludedPathFragments
	t.Cleanup(func() {
		excludedPathFragments = originalExclusions
	})
	excludedPathFragments = []string{"/tools/"}

	tempDir := t.TempDir()
	tool := &coverageTool{
		repoRoot:      tempDir,
		goBinary:      "go",
		commandRunner: &recordingRunner{},
		commandOutput: outputRunner{
			output: []byte("total:\t(statements)\t100.0%\n"),
		},
		logger: log.New(&bytes.Buffer{}, "coverage: ", 0),
	}

	if err := tool.run(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCoverageToolRunWriteOutputError(t *testing.T) {
	originalWriteFile := writeFile
	t.Cleanup(func() {
		writeFile = originalWriteFile
	})
	writeFile = func(string, []byte, os.FileMode) error {
		return errors.New("write failed")
	}

	tempDir := t.TempDir()
	coverageProfilePath := filepath.Join(tempDir, coverageProfileName)
	if err := os.WriteFile(coverageProfilePath, []byte("mode: set\n"), 0o644); err != nil {
		t.Fatalf("failed to write coverage profile: %v", err)
	}

	tool := &coverageTool{
		repoRoot:      tempDir,
		goBinary:      "go",
		commandRunner: &recordingRunner{},
		commandOutput: outputRunner{
			output: []byte("total:\t(statements)\t100.0%\n"),
		},
		logger: log.New(&bytes.Buffer{}, "coverage: ", 0),
	}

	if err := tool.run(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCoverageToolRunFinalizeFilteredFileError(t *testing.T) {
	originalExclusions := excludedPathFragments
	originalCloseFile := closeFile
	t.Cleanup(func() {
		excludedPathFragments = originalExclusions
		closeFile = originalCloseFile
	})
	excludedPathFragments = []string{"/tools/"}

	tempDir := t.TempDir()
	coverageProfilePath := filepath.Join(tempDir, coverageProfileName)
	if err := os.WriteFile(coverageProfilePath, []byte("mode: set\n"), 0o644); err != nil {
		t.Fatalf("failed to write coverage profile: %v", err)
	}

	closeFile = func(file *os.File) error {
		return errors.New("close failed")
	}

	tool := &coverageTool{
		repoRoot:      tempDir,
		goBinary:      "go",
		commandRunner: &recordingRunner{},
		commandOutput: outputRunner{
			output: []byte("total:\t(statements)\t100.0%\n"),
		},
		logger: log.New(&bytes.Buffer{}, "coverage: ", 0),
	}

	if err := tool.run(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCoverageToolRunGenerateHTMLError(t *testing.T) {
	tempDir := t.TempDir()
	coverageProfilePath := filepath.Join(tempDir, coverageProfileName)
	if err := os.WriteFile(coverageProfilePath, []byte("mode: set\n"), 0o644); err != nil {
		t.Fatalf("failed to write coverage profile: %v", err)
	}

	tool := &coverageTool{
		repoRoot: tempDir,
		goBinary: "go",
		commandRunner: selectiveRunner{
			failArgs: []string{"tool", "cover", "-html"},
			err:      errors.New("html fail"),
		},
		commandOutput: outputRunner{
			output: []byte("total:\t(statements)\t100.0%\n"),
		},
		logger: log.New(&bytes.Buffer{}, "coverage: ", 0),
	}

	if err := tool.run(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCoverageToolRunCoverageFuncOutputError(t *testing.T) {
	tempDir := t.TempDir()
	coverageProfilePath := filepath.Join(tempDir, coverageProfileName)
	if err := os.WriteFile(coverageProfilePath, []byte("mode: set\n"), 0o644); err != nil {
		t.Fatalf("failed to write coverage profile: %v", err)
	}

	tool := &coverageTool{
		repoRoot:      tempDir,
		goBinary:      "go",
		commandRunner: &recordingRunner{},
		commandOutput: outputRunner{
			err: errors.New("func fail"),
		},
		logger: log.New(&bytes.Buffer{}, "coverage: ", 0),
	}

	if err := tool.run(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCoverageToolRunNotFullCoverage(t *testing.T) {
	tempDir := t.TempDir()
	coverageProfilePath := filepath.Join(tempDir, coverageProfileName)
	if err := os.WriteFile(coverageProfilePath, []byte("mode: set\n"), 0o644); err != nil {
		t.Fatalf("failed to write coverage profile: %v", err)
	}

	tool := &coverageTool{
		repoRoot:      tempDir,
		goBinary:      "go",
		commandRunner: &recordingRunner{},
		commandOutput: outputRunner{
			output: []byte("github.com/meza/minecraft-mod-manager/internal/config/config.go:1.2 3.4 1 98.0%\n" +
				"total:\t(statements)\t98.0%\n"),
		},
		logger: log.New(&bytes.Buffer{}, "coverage: ", 0),
	}

	if err := tool.run(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCoverageToolRunNotFullCoverageWithExclusions(t *testing.T) {
	originalExclusions := excludedPathFragments
	t.Cleanup(func() {
		excludedPathFragments = originalExclusions
	})
	excludedPathFragments = []string{"/tools/"}

	tempDir := t.TempDir()
	coverageProfilePath := filepath.Join(tempDir, coverageProfileName)
	content := strings.Join([]string{
		"mode: set",
		"github.com/meza/minecraft-mod-manager/internal/config/config.go:1.2 3.4 1",
		"",
	}, "\n")
	if err := os.WriteFile(coverageProfilePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write coverage profile: %v", err)
	}

	tool := &coverageTool{
		repoRoot:      tempDir,
		goBinary:      "go",
		commandRunner: &recordingRunner{},
		commandOutput: outputRunner{
			output: []byte("github.com/meza/minecraft-mod-manager/internal/config/config.go:1.2 3.4 1 98.0%\n" +
				"total:\t(statements)\t98.0%\n"),
		},
		logger: log.New(&bytes.Buffer{}, "coverage: ", 0),
	}

	if err := tool.run(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunCoverageTestsUsesCommandRunner(t *testing.T) {
	tempDir := t.TempDir()
	tool := &coverageTool{
		repoRoot:      tempDir,
		goBinary:      "go",
		commandRunner: &recordingRunner{},
	}

	if err := tool.runCoverageTests(filepath.Join(tempDir, "coverage.profile")); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRunCoverageTestsError(t *testing.T) {
	tempDir := t.TempDir()
	tool := &coverageTool{
		repoRoot:      tempDir,
		goBinary:      "go",
		commandRunner: errorRunner{err: os.ErrInvalid},
	}

	if err := tool.runCoverageTests(filepath.Join(tempDir, "coverage.profile")); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunCoverageTestsErrorWriteFailure(t *testing.T) {
	tempDir := t.TempDir()
	tool := &coverageTool{
		repoRoot:      tempDir,
		goBinary:      "go",
		commandRunner: errorRunner{err: os.ErrInvalid},
	}

	originalStderr := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}
	os.Stderr = writer
	t.Cleanup(func() {
		os.Stderr = originalStderr
		if err := reader.Close(); err != nil {
			t.Fatalf("failed to close reader: %v", err)
		}
	})

	if err := tool.runCoverageTests(filepath.Join(tempDir, "coverage.profile")); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGenerateCoverageHTMLUsesCommandRunner(t *testing.T) {
	tempDir := t.TempDir()
	tool := &coverageTool{
		repoRoot:      tempDir,
		goBinary:      "go",
		commandRunner: &recordingRunner{},
	}

	if err := tool.generateCoverageHTML("coverage.profile", filepath.Join(tempDir, "coverage.html")); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestGenerateCoverageHTMLError(t *testing.T) {
	tempDir := t.TempDir()
	tool := &coverageTool{
		repoRoot:      tempDir,
		goBinary:      "go",
		commandRunner: errorRunner{err: os.ErrInvalid},
	}

	if err := tool.generateCoverageHTML("coverage.profile", filepath.Join(tempDir, "coverage.html")); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPrintCoverageHints(t *testing.T) {
	var output bytes.Buffer
	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	printCoverageHints("coverage.out", "coverage.html")

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}
	os.Stdout = originalStdout
	if _, err := output.ReadFrom(r); err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	if !strings.Contains(output.String(), "coverage.out") {
		t.Fatalf("expected output to mention coverage.out, got %q", output.String())
	}
}

func TestExecRunnerRun(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS_RUN") == "1" {
		os.Exit(0)
	}

	// #nosec G204 -- test helper executes the current test binary.
	cmd := exec.Command(os.Args[0], "-test.run=TestExecRunnerRun", "--") // #nosec G204 -- test helper executes the current test binary.
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS_RUN=1")
	if err := (execRunner{}).Run(cmd); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestExecOutputRunner(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS_OUTPUT") == "1" {
		os.Exit(0)
	}

	// #nosec G204 -- test helper executes the current test binary.
	cmd := exec.Command(os.Args[0], "-test.run=TestExecOutputRunner", "--") // #nosec G204 -- test helper executes the current test binary.
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS_OUTPUT=1")
	if _, err := (execOutputRunner{}).Output(cmd); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

type recordingRunner struct {
	commands []*exec.Cmd
}

func (runner *recordingRunner) Run(command *exec.Cmd) error {
	runner.commands = append(runner.commands, command)
	return nil
}

type outputRunner struct {
	output []byte
	err    error
}

func (runner outputRunner) Output(command *exec.Cmd) ([]byte, error) {
	return runner.output, runner.err
}

type errorRunner struct {
	err error
}

func (runner errorRunner) Run(command *exec.Cmd) error {
	return runner.err
}

type selectiveRunner struct {
	failArgs []string
	err      error
}

func (runner selectiveRunner) Run(command *exec.Cmd) error {
	if len(command.Args) >= len(runner.failArgs) {
		match := true
		for index, arg := range runner.failArgs {
			if command.Args[index+1] != arg {
				match = false
				break
			}
		}
		if match {
			return runner.err
		}
	}
	return nil
}

type errorWriter struct{}

func (errorWriter) Write(data []byte) (int, error) {
	return 0, errors.New("write failed")
}

type errorReader struct{}

func (errorReader) Read(data []byte) (int, error) {
	return 0, errors.New("read failed")
}

func TestFindRepoRoot(t *testing.T) {
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	nestedDir := filepath.Join(repoRoot, "tools", "coverage")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/test"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	found, err := findRepoRoot(nestedDir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found != repoRoot {
		t.Fatalf("expected repo root %q, got %q", repoRoot, found)
	}
}

func captureStdout(t *testing.T, run func()) string {
	t.Helper()

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = writer

	run()

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}
	os.Stdout = originalStdout

	var output bytes.Buffer
	if _, err := output.ReadFrom(reader); err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	return output.String()
}

func captureStderr(t *testing.T, run func()) string {
	t.Helper()

	originalStderr := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = writer

	run()

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}
	os.Stderr = originalStderr

	var output bytes.Buffer
	if _, err := output.ReadFrom(reader); err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	return output.String()
}

func TestFindRepoRootMissing(t *testing.T) {
	if _, err := findRepoRoot(t.TempDir()); err == nil {
		t.Fatal("expected error, got nil")
	}
}
