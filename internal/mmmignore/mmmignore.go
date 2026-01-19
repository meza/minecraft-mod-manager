// Package mmmignore parses .mmmignore patterns.
package mmmignore

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

const disabledPattern = "**/*.disabled"

var (
	absPath = filepath.Abs
	relPath = filepath.Rel
)

// ListPatterns loads .mmmignore patterns from rootDir and always includes
// the default disabled pattern. Blank lines are ignored.
//
// Example:
//
//	patterns, err := mmmignore.ListPatterns(fs, configDir)
func ListPatterns(fs afero.Fs, rootDir string) ([]string, error) {
	ignoreFile := filepath.Join(rootDir, ".mmmignore")
	exists, err := afero.Exists(fs, ignoreFile)
	if err != nil {
		return nil, err
	}

	patterns := []string{disabledPattern}
	if !exists {
		return patterns, nil
	}

	data, err := afero.ReadFile(fs, ignoreFile)
	if err != nil {
		return nil, err
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		patterns = append(patterns, line)
	}

	return patterns, nil
}

// AppendPatterns appends new patterns to the .mmmignore file when missing.
// It preserves existing contents and avoids duplicating trimmed patterns.
func AppendPatterns(fs afero.Fs, rootDir string, patterns []string) error {
	cleaned := normalizePatterns(patterns)
	if len(cleaned) == 0 {
		return nil
	}

	ignoreFile := filepath.Join(rootDir, ".mmmignore")
	existingData, existing, err := readExistingIgnoreFile(fs, ignoreFile)
	if err != nil {
		return err
	}

	additions := filterNewPatterns(cleaned, existing)
	if len(additions) == 0 {
		return nil
	}

	contents, err := buildIgnoreFileContents(existingData, additions)
	if err != nil {
		return err
	}

	return afero.WriteFile(fs, ignoreFile, []byte(contents), 0o644)
}

func normalizePatterns(patterns []string) []string {
	cleaned := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		trimmed := strings.TrimSpace(pattern)
		if trimmed == "" {
			continue
		}
		cleaned = append(cleaned, trimmed)
	}
	return cleaned
}

func readExistingIgnoreFile(fs afero.Fs, ignoreFile string) (string, map[string]struct{}, error) {
	exists, err := afero.Exists(fs, ignoreFile)
	if err != nil {
		return "", nil, err
	}

	existing := make(map[string]struct{})
	if !exists {
		return "", existing, nil
	}

	data, err := afero.ReadFile(fs, ignoreFile)
	if err != nil {
		return "", nil, err
	}

	existingData := string(data)
	for _, line := range strings.Split(existingData, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		existing[trimmed] = struct{}{}
	}

	return existingData, existing, nil
}

func filterNewPatterns(cleaned []string, existing map[string]struct{}) []string {
	additions := make([]string, 0, len(cleaned))
	for _, pattern := range cleaned {
		if _, ok := existing[pattern]; ok {
			continue
		}
		existing[pattern] = struct{}{}
		additions = append(additions, pattern)
	}
	return additions
}

func buildIgnoreFileContents(existingData string, additions []string) (string, error) {
	var builder strings.Builder
	if err := appendExistingData(&builder, existingData); err != nil {
		return "", err
	}
	if err := writeString(&builder, strings.Join(additions, "\n")); err != nil {
		return "", err
	}
	if err := writeString(&builder, "\n"); err != nil {
		return "", err
	}
	return builder.String(), nil
}

func appendExistingData(builder *strings.Builder, existingData string) error {
	if existingData == "" {
		return nil
	}
	if err := writeString(builder, existingData); err != nil {
		return err
	}
	if strings.HasSuffix(existingData, "\n") {
		return nil
	}
	return writeString(builder, "\n")
}

var writeString = func(builder *strings.Builder, value string) error {
	_, err := builder.WriteString(value)
	return err
}

func IsIgnored(matchRoot string, absolutePath string, patterns []string) bool {
	relativePath, ok := pathRelativeToRoot(matchRoot, absolutePath)
	if !ok {
		return false
	}
	relativePath = filepath.ToSlash(relativePath)

	for _, pattern := range patterns {
		pattern = filepath.ToSlash(strings.TrimSpace(pattern))
		if pattern == "" {
			continue
		}
		if globMatch(pattern, relativePath) {
			return true
		}
	}

	return false
}

func IgnoredFiles(fs afero.Fs, ignoreDir string, matchRoot string) (map[string]bool, error) {
	patterns, err := ListPatterns(fs, ignoreDir)
	if err != nil {
		return nil, err
	}
	return buildIgnoredSet(fs, matchRoot, patterns)
}

func buildIgnoredSet(fs afero.Fs, matchRoot string, patterns []string) (map[string]bool, error) {
	ignored := make(map[string]bool)

	walkErr := afero.Walk(fs, matchRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return err
		}

		if IsIgnored(matchRoot, path, patterns) {
			ignored[path] = true
			return nil
		}

		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	return ignored, nil
}

func pathRelativeToRoot(rootDir string, absolutePath string) (string, bool) {
	rootAbs, err := absPath(rootDir)
	if err != nil {
		return "", false
	}
	pathAbs, err := absPath(absolutePath)
	if err != nil {
		return "", false
	}

	rel, err := relPath(rootAbs, pathAbs)
	if err != nil {
		return "", false
	}
	if rel == "." {
		return "", true
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return rel, true
}

func globMatch(pattern string, target string) bool {
	pattern = strings.TrimPrefix(pattern, "./")
	target = strings.TrimPrefix(target, "./")

	patternParts := strings.Split(pattern, "/")
	targetParts := strings.Split(target, "/")

	var match func(pi, ti int) bool
	match = func(pi, ti int) bool {
		if pi == len(patternParts) {
			return ti == len(targetParts)
		}

		part := patternParts[pi]
		if part == "**" {
			for skip := ti; skip <= len(targetParts); skip++ {
				if match(pi+1, skip) {
					return true
				}
			}
			return false
		}

		if ti >= len(targetParts) {
			return false
		}

		ok, err := filepath.Match(part, targetParts[ti])
		if err != nil || !ok {
			return false
		}
		return match(pi+1, ti+1)
	}

	return match(0, 0)
}
