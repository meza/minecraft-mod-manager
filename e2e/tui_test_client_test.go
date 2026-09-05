//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const requiredTUITestVersion = "0.1.0-beta.2"

var tuiTestSessionSequence atomic.Uint64

type tuiTestEnvelope struct {
	OK      bool            `json:"ok"`
	Data    json.RawMessage `json:"data"`
	Kind    string          `json:"kind"`
	Message string          `json:"message"`
}

type tuiTestMetadata struct {
	SchemaVersion string `json:"schema_version"`
	Version       string `json:"version"`
}

type tuiTestRunData struct {
	Recording string `json:"recording"`
}

type tuiTestTextData struct {
	Text string `json:"text"`
}

type tuiTestState struct {
	Columns int    `json:"cols"`
	Rows    int    `json:"rows"`
	Text    string `json:"text"`
	Exited  *int   `json:"exited"`
}

type tuiTestCommandError struct {
	Arguments      []string
	ExitCode       int
	Kind           string
	Message        string
	StandardOutput string
	StandardError  string
}

func (commandError *tuiTestCommandError) Error() string {
	return fmt.Sprintf(
		"tui-test command failed: args=%q exit=%d kind=%q message=%q stdout=%q stderr=%q",
		commandError.Arguments,
		commandError.ExitCode,
		commandError.Kind,
		commandError.Message,
		commandError.StandardOutput,
		commandError.StandardError,
	)
}

type tuiTestLaunch struct {
	Program          string
	Arguments        []string
	WorkingDirectory string
	Environment      []string
	Columns          int
	Rows             int
}

type terminalDriver interface {
	Run(tuiTestLaunch) error
	WaitText(string, time.Duration) error
	WaitIdle(time.Duration) error
	WaitExit(time.Duration) error
	Type(string) error
	Press(...string) error
	Resize(int, int) error
	Click(int, int) error
	Scroll(string, int) error
	Text(bool) (string, error)
	State() (tuiTestState, error)
	ExpectSnapshot(string, bool, bool) error
	Diagnostics() string
	Close() error
}

type tuiTestClient struct {
	binaryPath       string
	sessionName      string
	commandDirectory string
	recordingPath    string
}

func newTUITestClient(commandDirectory string) (*tuiTestClient, error) {
	binaryPath, err := resolveTUITestBinary()
	if err != nil {
		return nil, err
	}
	if err := validateTUITest(binaryPath); err != nil {
		return nil, err
	}

	return &tuiTestClient{
		binaryPath:       binaryPath,
		sessionName:      fmt.Sprintf("mmm-e2e-%d-%d", os.Getpid(), tuiTestSessionSequence.Add(1)),
		commandDirectory: commandDirectory,
	}, nil
}

func resolveTUITestBinary() (string, error) {
	if override := strings.TrimSpace(os.Getenv("TUI_TEST_BIN")); override != "" {
		return override, nil
	}
	if resolved, err := exec.LookPath("tui-test"); err == nil {
		return resolved, nil
	}

	for _, candidate := range defaultTUITestLocations() {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	return "", fmt.Errorf(
		"tui-test %s is required; install it from https://github.com/microsoft/tui-test or set TUI_TEST_BIN",
		requiredTUITestVersion,
	)
}

func defaultTUITestLocations() []string {
	if runtime.GOOS == "windows" {
		localAppData := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
		if localAppData == "" {
			return nil
		}
		return []string{filepath.Join(localAppData, "Programs", "tui-test", "bin", "tui-test.exe")}
	}

	locations := []string{"/usr/local/bin/tui-test"}
	if userHome, err := os.UserHomeDir(); err == nil {
		locations = append([]string{filepath.Join(userHome, ".local", "bin", "tui-test")}, locations...)
	}
	return locations
}

func validateTUITest(binaryPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//nolint:gosec // The executable path is explicitly configured or discovered from trusted install locations.
	command := exec.CommandContext(ctx, binaryPath, "agent-context", "--json")
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("inspect tui-test installation at %q: %w", binaryPath, err)
	}

	var metadata tuiTestMetadata
	if err := json.Unmarshal(output, &metadata); err != nil {
		return fmt.Errorf("decode tui-test metadata from %q: %w", binaryPath, err)
	}
	if metadata.SchemaVersion != "1" || metadata.Version != requiredTUITestVersion {
		return fmt.Errorf(
			"tui-test %s with schema 1 is required, found version %q schema %q at %q; set TUI_TEST_BIN to the pinned executable",
			requiredTUITestVersion,
			metadata.Version,
			metadata.SchemaVersion,
			binaryPath,
		)
	}
	return nil
}

func (client *tuiTestClient) Run(launch tuiTestLaunch) error {
	arguments := []string{
		"run",
		"--cols", strconv.Itoa(launch.Columns),
		"--rows", strconv.Itoa(launch.Rows),
		"--cwd", launch.WorkingDirectory,
	}
	for _, environmentEntry := range launch.Environment {
		arguments = append(arguments, "--env", environmentEntry)
	}
	arguments = append(arguments, launch.Program)
	arguments = append(arguments, launch.Arguments...)

	data, err := client.executeWithDiagnostics(arguments...)
	if err != nil {
		return err
	}
	var runData tuiTestRunData
	if len(data) > 0 {
		if err := json.Unmarshal(data, &runData); err != nil {
			return fmt.Errorf("decode tui-test run response: %w", err)
		}
	}
	client.recordingPath = runData.Recording
	return nil
}

func (client *tuiTestClient) WaitText(value string, timeout time.Duration) error {
	_, err := client.executeWithDiagnosticsTimeout(timeout+5*time.Second, "wait", "text", value, "--timeout", timeoutMilliseconds(timeout))
	return err
}

func (client *tuiTestClient) WaitIdle(timeout time.Duration) error {
	_, err := client.executeWithDiagnosticsTimeout(timeout+5*time.Second, "wait", "idle", "--timeout", timeoutMilliseconds(timeout))
	return err
}

func (client *tuiTestClient) WaitExit(timeout time.Duration) error {
	_, err := client.executeWithDiagnosticsTimeout(timeout+5*time.Second, "wait", "exit", "--timeout", timeoutMilliseconds(timeout))
	return err
}

func (client *tuiTestClient) Type(value string) error {
	_, err := client.executeWithDiagnostics("type", value)
	return err
}

func (client *tuiTestClient) Press(keys ...string) error {
	arguments := append([]string{"key", "press"}, keys...)
	_, err := client.executeWithDiagnostics(arguments...)
	return err
}

func (client *tuiTestClient) Resize(columns int, rows int) error {
	_, err := client.executeWithDiagnostics("resize", strconv.Itoa(columns), strconv.Itoa(rows))
	return err
}

func (client *tuiTestClient) Click(column int, row int) error {
	_, err := client.executeWithDiagnostics("mouse", "click", strconv.Itoa(column), strconv.Itoa(row))
	return err
}

func (client *tuiTestClient) Scroll(direction string, amount int) error {
	_, err := client.executeWithDiagnostics("mouse", "scroll", direction, "--amount", strconv.Itoa(amount))
	return err
}

func (client *tuiTestClient) Text(full bool) (string, error) {
	arguments := []string{"text"}
	if full {
		arguments = append(arguments, "--full")
	}
	data, err := client.executeWithDiagnostics(arguments...)
	if err != nil {
		return "", err
	}
	var textData tuiTestTextData
	if err := json.Unmarshal(data, &textData); err != nil {
		return "", fmt.Errorf("decode tui-test text response: %w", err)
	}
	return textData.Text, nil
}

func (client *tuiTestClient) State() (tuiTestState, error) {
	data, err := client.executeWithDiagnostics("state")
	if err != nil {
		return tuiTestState{}, err
	}
	var state tuiTestState
	if err := json.Unmarshal(data, &state); err != nil {
		return tuiTestState{}, fmt.Errorf("decode tui-test state response: %w", err)
	}
	return state, nil
}

func (client *tuiTestClient) ExpectSnapshot(name string, update bool, includeColors bool) error {
	arguments := []string{"expect", "snapshot", name}
	if update {
		arguments = append(arguments, "--update")
	}
	if includeColors {
		arguments = append(arguments, "--include-colors")
	}
	_, err := client.executeWithDiagnostics(arguments...)
	return err
}

func (client *tuiTestClient) Diagnostics() string {
	var sections []string
	if client.recordingPath != "" {
		sections = append(sections, "recording: "+client.recordingPath)
	}
	if data, err := client.execute("state"); err == nil {
		sections = append(sections, "state: "+string(data))
	}
	if data, err := client.execute("text", "--full"); err == nil {
		sections = append(sections, "full text: "+string(data))
	}
	return strings.Join(sections, "\n")
}

func (client *tuiTestClient) Close() error {
	_, err := client.execute("close")
	var commandError *tuiTestCommandError
	if errors.As(err, &commandError) && commandError.Kind == "no_session" {
		return nil
	}
	return err
}

func (client *tuiTestClient) executeWithDiagnostics(arguments ...string) (json.RawMessage, error) {
	return client.executeWithDiagnosticsTimeout(45*time.Second, arguments...)
}

func (client *tuiTestClient) executeWithDiagnosticsTimeout(timeout time.Duration, arguments ...string) (json.RawMessage, error) {
	data, err := client.executeWithTimeout(timeout, arguments...)
	if err == nil {
		return data, nil
	}
	diagnostics := client.Diagnostics()
	if diagnostics == "" {
		return nil, err
	}
	return nil, fmt.Errorf("%w\n%s", err, diagnostics)
}

func (client *tuiTestClient) execute(arguments ...string) (json.RawMessage, error) {
	return client.executeWithTimeout(45*time.Second, arguments...)
}

func (client *tuiTestClient) executeWithTimeout(timeout time.Duration, arguments ...string) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	commandArguments := []string{"--session", client.sessionName, "--json"}
	commandArguments = append(commandArguments, arguments...)

	//nolint:gosec // The executable path is explicitly configured or discovered from trusted install locations.
	command := exec.CommandContext(ctx, client.binaryPath, commandArguments...)
	command.Dir = client.commandDirectory
	var standardOutput bytes.Buffer
	var standardError bytes.Buffer
	command.Stdout = &standardOutput
	command.Stderr = &standardError
	runErr := command.Run()

	var envelope tuiTestEnvelope
	decodeErr := json.Unmarshal(standardOutput.Bytes(), &envelope)
	if decodeErr != nil {
		if runErr != nil {
			return nil, fmt.Errorf(
				"run tui-test %q: %w; stdout=%q stderr=%q",
				arguments,
				runErr,
				standardOutput.String(),
				standardError.String(),
			)
		}
		return nil, fmt.Errorf("decode tui-test response for %q: %w; stdout=%q", arguments, decodeErr, standardOutput.String())
	}
	if runErr != nil || !envelope.OK {
		exitCode := 0
		var exitError *exec.ExitError
		if errors.As(runErr, &exitError) {
			exitCode = exitError.ExitCode()
		}
		return nil, &tuiTestCommandError{
			Arguments:      append([]string(nil), arguments...),
			ExitCode:       exitCode,
			Kind:           envelope.Kind,
			Message:        envelope.Message,
			StandardOutput: standardOutput.String(),
			StandardError:  standardError.String(),
		}
	}
	return envelope.Data, nil
}

func timeoutMilliseconds(timeout time.Duration) string {
	return strconv.FormatInt(timeout.Milliseconds(), 10)
}
