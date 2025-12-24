package main

import (
	"bytes"
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type recordingRunner struct {
	commands []*execCmdSnapshot
	runError error
}

type execCmdSnapshot struct {
	args []string
	env  []string
}

func (runner *recordingRunner) Run(command *exec.Cmd) error {
	runner.commands = append(runner.commands, &execCmdSnapshot{
		args: append([]string{}, command.Args...),
		env:  append([]string{}, command.Env...),
	})
	return runner.runError
}

func TestNewBuildToolGetwdFailure(t *testing.T) {
	originalGetWorkingDirectory := getWorkingDirectory
	t.Cleanup(func() {
		getWorkingDirectory = originalGetWorkingDirectory
	})

	getWorkingDirectory = func() (string, error) {
		return "", errors.New("boom")
	}

	tool, err := newBuildTool()
	if err == nil {
		t.Fatalf("expected error, got nil and tool %v", tool)
	}
}

func TestRunMainGetwdFailure(t *testing.T) {
	originalNewBuildToolFunc := newBuildToolFunc
	t.Cleanup(func() {
		newBuildToolFunc = originalNewBuildToolFunc
	})

	newBuildToolFunc = func() (*buildTool, error) {
		return nil, errors.New("boom")
	}

	if exitCode := runMain(); exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestMainSuccessUsesExit(t *testing.T) {
	originalNewBuildToolFunc := newBuildToolFunc
	originalExit := exit
	t.Cleanup(func() {
		newBuildToolFunc = originalNewBuildToolFunc
		exit = originalExit
	})

	tempDir := t.TempDir()
	newBuildToolFunc = func() (*buildTool, error) {
		return &buildTool{
			repoRoot: tempDir,
			baseEnv: []string{
				modrinthEnvVar + "=token",
				curseforgeEnvVar + "=token",
				posthogEnvVar + "=token",
			},
			goBinary:      "go",
			commandRunner: &recordingRunner{},
			envFileReader: func(string) (map[string]string, error) {
				return map[string]string{}, nil
			},
			logger: log.New(&bytes.Buffer{}, "build: ", 0),
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
	originalNewBuildToolFunc := newBuildToolFunc
	t.Cleanup(func() {
		newBuildToolFunc = originalNewBuildToolFunc
	})

	newBuildToolFunc = func() (*buildTool, error) {
		return &buildTool{
			repoRoot:      t.TempDir(),
			baseEnv:       nil,
			goBinary:      "go",
			commandRunner: &recordingRunner{},
			envFileReader: func(string) (map[string]string, error) {
				return map[string]string{}, nil
			},
			logger: log.New(&bytes.Buffer{}, "build: ", 0),
		}, nil
	}

	if exitCode := runMain(); exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestNewBuildToolFindsRepoRoot(t *testing.T) {
	originalGetWorkingDirectory := getWorkingDirectory
	t.Cleanup(func() {
		getWorkingDirectory = originalGetWorkingDirectory
	})

	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	nestedDir := filepath.Join(repoRoot, "tools", "build")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/test"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	getWorkingDirectory = func() (string, error) {
		return nestedDir, nil
	}

	tool, err := newBuildTool()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tool.repoRoot != repoRoot {
		t.Fatalf("expected repo root %q, got %q", repoRoot, tool.repoRoot)
	}
}

func TestNewBuildToolMissingRepoRoot(t *testing.T) {
	originalGetWorkingDirectory := getWorkingDirectory
	t.Cleanup(func() {
		getWorkingDirectory = originalGetWorkingDirectory
	})

	tempDir := t.TempDir()
	getWorkingDirectory = func() (string, error) {
		return tempDir, nil
	}

	tool, err := newBuildTool()
	if err == nil {
		t.Fatalf("expected error, got nil and tool %v", tool)
	}
	if !strings.Contains(err.Error(), "failed to locate repo root") {
		t.Fatalf("expected repo root error, got %q", err.Error())
	}
}

func TestReadEnvFileMissing(t *testing.T) {
	tempDir := t.TempDir()
	values, err := readEnvFile(filepath.Join(tempDir, "missing.env"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(values) != 0 {
		t.Fatalf("expected empty env map, got %v", values)
	}
}

func TestReadEnvFileDirectoryError(t *testing.T) {
	tempDir := t.TempDir()
	_, err := readEnvFile(tempDir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestReadEnvFileParseError(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(envPath, []byte("BAD-VAR=1"), 0o644); err != nil {
		t.Fatalf("failed to write env file: %v", err)
	}

	_, err := readEnvFile(envPath)
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestReadEnvFileSuccess(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(envPath, []byte("FOO=bar\nBAZ=qux\n"), 0o644); err != nil {
		t.Fatalf("failed to write env file: %v", err)
	}

	values, err := readEnvFile(envPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if values["FOO"] != "bar" || values["BAZ"] != "qux" {
		t.Fatalf("unexpected env values: %v", values)
	}
}

func TestBuildEnvMapUsesEnvFileForMissingTokens(t *testing.T) {
	envFileValues := map[string]string{
		modrinthEnvVar:   "modrinth-token",
		curseforgeEnvVar: "curseforge-token",
		posthogEnvVar:    "posthog-token",
	}

	envMap := buildEnvMap(nil, envFileValues)
	if envMap[modrinthEnvVar] != "modrinth-token" {
		t.Fatalf("expected modrinth token from env file, got %q", envMap[modrinthEnvVar])
	}
	if envMap[curseforgeEnvVar] != "curseforge-token" {
		t.Fatalf("expected curseforge token from env file, got %q", envMap[curseforgeEnvVar])
	}
	if envMap[posthogEnvVar] != "posthog-token" {
		t.Fatalf("expected posthog token from env file, got %q", envMap[posthogEnvVar])
	}
}

func TestBuildEnvMapDoesNotOverrideSetTokens(t *testing.T) {
	baseEnv := []string{
		modrinthEnvVar + "=",
	}
	envFileValues := map[string]string{
		modrinthEnvVar:   "modrinth-token",
		curseforgeEnvVar: "curseforge-token",
		posthogEnvVar:    "posthog-token",
	}

	envMap := buildEnvMap(baseEnv, envFileValues)
	if envMap[modrinthEnvVar] != "" {
		t.Fatalf("expected modrinth token to stay empty, got %q", envMap[modrinthEnvVar])
	}
	if envMap[curseforgeEnvVar] != "curseforge-token" {
		t.Fatalf("expected curseforge token from env file, got %q", envMap[curseforgeEnvVar])
	}
	if envMap[posthogEnvVar] != "posthog-token" {
		t.Fatalf("expected posthog token from env file, got %q", envMap[posthogEnvVar])
	}
}

func TestEnvSliceToMapSkipsMalformedEntry(t *testing.T) {
	envMap := envSliceToMap([]string{"GOOD=1", "MALFORMED"})
	if envMap["GOOD"] != "1" {
		t.Fatalf("expected GOOD=1, got %q", envMap["GOOD"])
	}
	if _, exists := envMap["MALFORMED"]; exists {
		t.Fatal("expected malformed entry to be skipped")
	}
}

func TestBuildToolRunMissingTokens(t *testing.T) {
	tempDir := t.TempDir()
	logBuffer := &bytes.Buffer{}
	tool := &buildTool{
		repoRoot:      tempDir,
		baseEnv:       nil,
		goBinary:      "go",
		commandRunner: &recordingRunner{},
		envFileReader: func(string) (map[string]string, error) {
			return map[string]string{}, nil
		},
		logger: log.New(logBuffer, "build: ", 0),
	}

	err := tool.run()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expected := "error: missing build token(s): MODRINTH_API_KEY CURSEFORGE_API_KEY POSTHOG_API_KEY\n" +
		"hint: set them as environment variables or add them to ./.env (repo root) before running make"
	if err.Error() != expected {
		t.Fatalf("expected error %q, got %q", expected, err.Error())
	}
}

func TestBuildToolRunEnvFileError(t *testing.T) {
	tempDir := t.TempDir()
	logBuffer := &bytes.Buffer{}
	tool := &buildTool{
		repoRoot:      tempDir,
		baseEnv:       nil,
		goBinary:      "go",
		commandRunner: &recordingRunner{},
		envFileReader: func(string) (map[string]string, error) {
			return nil, errors.New("env fail")
		},
		logger: log.New(logBuffer, "build: ", 0),
	}

	err := tool.run()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read .env") {
		t.Fatalf("expected .env failure, got %q", err.Error())
	}
}

func TestBuildToolRunBuildsAllTargets(t *testing.T) {
	tempDir := t.TempDir()
	baseEnv := []string{
		modrinthEnvVar + "=modrinth-token",
		curseforgeEnvVar + "=curseforge-token",
		posthogEnvVar + "=posthog-token",
	}
	runner := &recordingRunner{}
	logBuffer := &bytes.Buffer{}
	tool := &buildTool{
		repoRoot:      tempDir,
		baseEnv:       baseEnv,
		goBinary:      "go",
		commandRunner: runner,
		envFileReader: func(string) (map[string]string, error) {
			return map[string]string{}, nil
		},
		logger: log.New(logBuffer, "build: ", 0),
	}

	if err := tool.run(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(runner.commands) != len(buildTargets) {
		t.Fatalf("expected %d commands, got %d", len(buildTargets), len(runner.commands))
	}

	if !strings.Contains(logBuffer.String(), "build complete") {
		t.Fatalf("expected build complete log, got %q", logBuffer.String())
	}

	envMap := buildEnvMap(baseEnv, nil)
	expectedLdflags := ldflagsFromTokens(envMap)

	for index, target := range buildTargets {
		snapshot := runner.commands[index]
		outputName := executableName
		if target.goos == "windows" {
			outputName = outputName + ".exe"
		}
		outputPath := filepath.Join(tempDir, "build", target.goos, target.goarch, outputName)
		expectedArgs := []string{"go", "build", "-ldflags", expectedLdflags, "-o", outputPath, "main.go"}
		if !reflect.DeepEqual(snapshot.args, expectedArgs) {
			t.Fatalf("unexpected args for %s/%s: %v", target.goos, target.goarch, snapshot.args)
		}

		commandEnv := envSliceToMap(snapshot.env)
		if commandEnv["GOOS"] != target.goos {
			t.Fatalf("expected GOOS %q, got %q", target.goos, commandEnv["GOOS"])
		}
		if commandEnv["GOARCH"] != target.goarch {
			t.Fatalf("expected GOARCH %q, got %q", target.goarch, commandEnv["GOARCH"])
		}
		if commandEnv["CGO_ENABLED"] != "0" {
			t.Fatalf("expected CGO_ENABLED=0, got %q", commandEnv["CGO_ENABLED"])
		}
		if commandEnv[modrinthEnvVar] != "modrinth-token" {
			t.Fatalf("expected modrinth token, got %q", commandEnv[modrinthEnvVar])
		}
		if commandEnv[curseforgeEnvVar] != "curseforge-token" {
			t.Fatalf("expected curseforge token, got %q", commandEnv[curseforgeEnvVar])
		}
		if commandEnv[posthogEnvVar] != "posthog-token" {
			t.Fatalf("expected posthog token, got %q", commandEnv[posthogEnvVar])
		}

		if _, err := os.Stat(filepath.Dir(outputPath)); err != nil {
			t.Fatalf("expected build directory for %s/%s to exist: %v", target.goos, target.goarch, err)
		}
	}
}

func TestBuildToolRunBuildTargetError(t *testing.T) {
	tempDir := t.TempDir()
	baseEnv := []string{
		modrinthEnvVar + "=modrinth-token",
		curseforgeEnvVar + "=curseforge-token",
		posthogEnvVar + "=posthog-token",
	}
	runner := &recordingRunner{runError: errors.New("run fail")}
	logBuffer := &bytes.Buffer{}
	tool := &buildTool{
		repoRoot:      tempDir,
		baseEnv:       baseEnv,
		goBinary:      "go",
		commandRunner: runner,
		envFileReader: func(string) (map[string]string, error) {
			return map[string]string{}, nil
		},
		logger: log.New(logBuffer, "build: ", 0),
	}

	if err := tool.run(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBuildTargetMkdirError(t *testing.T) {
	tempDir := t.TempDir()
	buildRoot := filepath.Join(tempDir, "build")
	if err := os.WriteFile(buildRoot, []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("failed to create conflicting build file: %v", err)
	}

	logBuffer := &bytes.Buffer{}
	tool := &buildTool{
		repoRoot:      tempDir,
		baseEnv:       []string{modrinthEnvVar + "=token", curseforgeEnvVar + "=token", posthogEnvVar + "=token"},
		goBinary:      "go",
		commandRunner: &recordingRunner{},
		envFileReader: func(string) (map[string]string, error) {
			return map[string]string{}, nil
		},
		logger: log.New(logBuffer, "build: ", 0),
	}

	err := tool.buildTarget(buildTarget{goos: "linux", goarch: "amd64"}, buildEnvMap(tool.baseEnv, nil), "ldflags")
	if err == nil {
		t.Fatal("expected mkdir error, got nil")
	}
}

func TestBuildTargetRunError(t *testing.T) {
	tempDir := t.TempDir()
	runner := &recordingRunner{runError: errors.New("run fail")}
	logBuffer := &bytes.Buffer{}
	tool := &buildTool{
		repoRoot:      tempDir,
		baseEnv:       []string{modrinthEnvVar + "=token", curseforgeEnvVar + "=token", posthogEnvVar + "=token"},
		goBinary:      "go",
		commandRunner: runner,
		envFileReader: func(string) (map[string]string, error) {
			return map[string]string{}, nil
		},
		logger: log.New(logBuffer, "build: ", 0),
	}

	err := tool.buildTarget(buildTarget{goos: "linux", goarch: "amd64"}, buildEnvMap(tool.baseEnv, nil), "ldflags")
	if err == nil {
		t.Fatal("expected run error, got nil")
	}
	if !strings.Contains(err.Error(), "build linux/amd64") {
		t.Fatalf("expected build error to include target, got %q", err.Error())
	}
}

func TestExecRunnerRun(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		os.Exit(0)
	}

	//nolint:gosec // test helper executes the current test binary.
	cmd := exec.Command(os.Args[0], "-test.run=TestExecRunnerRun", "--")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	if err := (execRunner{}).Run(cmd); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
