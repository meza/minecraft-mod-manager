// Package mmmignore parses .mmmignore patterns.
package mmmignore

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

const disabledPattern = "**/*.disabled"

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

func IsIgnored(rootDir string, absolutePath string, patterns []string) bool {
	cleanRoot := filepath.Clean(rootDir)
	cleanPath := filepath.Clean(absolutePath)

	if cleanPath != cleanRoot && !strings.HasPrefix(cleanPath, cleanRoot+string(filepath.Separator)) {
		return false
	}

	rel := strings.TrimPrefix(cleanPath, cleanRoot)
	rel = strings.TrimPrefix(rel, string(filepath.Separator))
	rel = filepath.ToSlash(rel)

	for _, pattern := range patterns {
		pattern = filepath.ToSlash(strings.TrimSpace(pattern))
		if pattern == "" {
			continue
		}
		if globMatch(pattern, rel) {
			return true
		}
	}

	return false
}

func IgnoredFiles(fs afero.Fs, rootDir string) (map[string]bool, error) {
	patterns, err := ListPatterns(fs, rootDir)
	if err != nil {
		return nil, err
	}
	return buildIgnoredSet(fs, rootDir, patterns)
}

func buildIgnoredSet(fs afero.Fs, rootDir string, patterns []string) (map[string]bool, error) {
	ignored := make(map[string]bool)

	walkErr := afero.Walk(fs, rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return err
		}

		if IsIgnored(rootDir, path, patterns) {
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
