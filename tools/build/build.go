package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joho/godotenv"
)

const (
	executableName = "mmm"

	modrinthEnvVar   = "MODRINTH_API_KEY"
	curseforgeEnvVar = "CURSEFORGE_API_KEY"
	posthogEnvVar    = "POSTHOG_API_KEY"

	modrinthLdflag   = "github.com/meza/minecraft-mod-manager/internal/environment.modrinthApiKeyDefault"
	curseforgeLdflag = "github.com/meza/minecraft-mod-manager/internal/environment.curseforgeApiKeyDefault"
	posthogLdflag    = "github.com/meza/minecraft-mod-manager/internal/environment.posthogApiKeyDefault"
)

var buildTargets = []buildTarget{
	{goos: "darwin", goarch: "amd64"},
	{goos: "darwin", goarch: "arm64"},
	{goos: "linux", goarch: "amd64"},
	{goos: "linux", goarch: "arm64"},
	{goos: "windows", goarch: "amd64"},
	{goos: "windows", goarch: "arm64"},
}

type buildTarget struct {
	goos   string
	goarch string
}

type commandRunner interface {
	Run(*exec.Cmd) error
}

type logger interface {
	Printf(format string, args ...any)
}

type execRunner struct{}

func (execRunner) Run(command *exec.Cmd) error {
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

type envFileReader func(string) (map[string]string, error)

type buildTool struct {
	repoRoot      string
	baseEnv       []string
	goBinary      string
	commandRunner commandRunner
	envFileReader envFileReader
	logger        logger
}

var getWorkingDirectory = os.Getwd
var newBuildToolFunc = newBuildTool
var exit = os.Exit

func newBuildTool() (*buildTool, error) {
	workingDirectory, err := getWorkingDirectory()
	if err != nil {
		return nil, fmt.Errorf("error: failed to determine working directory: %w", err)
	}

	repoRoot, err := findRepoRoot(workingDirectory)
	if err != nil {
		return nil, err
	}

	return &buildTool{
		repoRoot:      repoRoot,
		baseEnv:       os.Environ(),
		goBinary:      "go",
		commandRunner: execRunner{},
		envFileReader: readEnvFile,
		logger:        log.New(os.Stdout, "build: ", 0),
	}, nil
}

func main() {
	exit(runMain())
}

func runMain() int {
	tool, err := newBuildToolFunc()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if err := tool.run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func (tool *buildTool) run() error {
	envFilePath := filepath.Join(tool.repoRoot, ".env")
	tool.logger.Printf("loading %s", envFilePath)
	envFileValues, err := tool.envFileReader(envFilePath)
	if err != nil {
		return fmt.Errorf("error: failed to read .env: %w", err)
	}

	envMap := buildEnvMap(tool.baseEnv, envFileValues)
	missing := missingTokenNames(envMap)
	if len(missing) > 0 {
		return fmt.Errorf("%s\n%s", missingTokensErrorLine(missing), missingTokensHintLine())
	}

	ldflags := ldflagsFromTokens(envMap)

	for _, target := range buildTargets {
		tool.logger.Printf("building %s/%s", target.goos, target.goarch)
		if err := tool.buildTarget(target, envMap, ldflags); err != nil {
			return err
		}
		tool.logger.Printf("finished %s/%s", target.goos, target.goarch)
	}

	tool.logger.Printf("build complete")
	return nil
}

func (tool *buildTool) buildTarget(target buildTarget, envMap map[string]string, ldflags string) error {
	outputDir := filepath.Join(tool.repoRoot, "build", target.goos, target.goarch)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("error: create build directory: %w", err)
	}

	outputName := executableName
	if target.goos == "windows" {
		outputName = outputName + ".exe"
	}
	outputPath := filepath.Join(outputDir, outputName)

	environment := copyEnvMap(envMap)
	environment["GOOS"] = target.goos
	environment["GOARCH"] = target.goarch
	environment["CGO_ENABLED"] = "0"

	command := exec.Command(tool.goBinary, "build", "-ldflags", ldflags, "-o", outputPath, "main.go")
	command.Env = envMapToSlice(environment)

	tool.logger.Printf("output %s", outputPath)
	if err := tool.commandRunner.Run(command); err != nil {
		return fmt.Errorf("build %s/%s: %w", target.goos, target.goarch, err)
	}

	return nil
}

func readEnvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	values, err := godotenv.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return values, nil
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

func buildEnvMap(baseEnv []string, envFileValues map[string]string) map[string]string {
	envMap := envSliceToMap(baseEnv)
	for _, tokenName := range requiredTokenNames() {
		if _, exists := envMap[tokenName]; exists {
			continue
		}
		if value, ok := envFileValues[tokenName]; ok {
			envMap[tokenName] = value
		}
	}
	return envMap
}

func envSliceToMap(baseEnv []string) map[string]string {
	envMap := make(map[string]string, len(baseEnv))
	for _, entry := range baseEnv {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		envMap[key] = value
	}
	return envMap
}

func envMapToSlice(envMap map[string]string) []string {
	keys := make([]string, 0, len(envMap))
	for key := range envMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	entries := make([]string, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, key+"="+envMap[key])
	}
	return entries
}

func copyEnvMap(envMap map[string]string) map[string]string {
	clone := make(map[string]string, len(envMap))
	for key, value := range envMap {
		clone[key] = value
	}
	return clone
}

func requiredTokenNames() []string {
	return []string{modrinthEnvVar, curseforgeEnvVar, posthogEnvVar}
}

func missingTokenNames(envMap map[string]string) []string {
	var missing []string
	for _, name := range requiredTokenNames() {
		value, ok := envMap[name]
		if !ok || value == "" {
			missing = append(missing, name)
		}
	}
	return missing
}

func missingTokensErrorLine(missing []string) string {
	return fmt.Sprintf("error: missing build token(s): %s", strings.Join(missing, " "))
}

func missingTokensHintLine() string {
	return "hint: set them as environment variables or add them to ./.env (repo root) before running make"
}

func ldflagsFromTokens(envMap map[string]string) string {
	modrinthToken := envMap[modrinthEnvVar]
	curseforgeToken := envMap[curseforgeEnvVar]
	posthogToken := envMap[posthogEnvVar]

	return fmt.Sprintf(
		"-X %s=%s -X %s=%s -X %s=%s",
		modrinthLdflag,
		modrinthToken,
		curseforgeLdflag,
		curseforgeToken,
		posthogLdflag,
		posthogToken,
	)
}
