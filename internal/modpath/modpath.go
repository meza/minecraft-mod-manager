// Package modpath resolves mod-related filesystem paths.
package modpath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

type OutsideRootError struct {
	Path         string
	ResolvedPath string
	Root         string
}

func (err OutsideRootError) Error() string {
	return fmt.Sprintf("resolved path %s for %s is outside root %s", err.ResolvedPath, err.Path, err.Root)
}

func ResolveWritablePath(fs afero.Fs, root string, destination string) (string, error) {
	return resolveWritablePathWithFuncs(fs, root, destination, filepath.EvalSymlinks, filepath.Abs)
}

func resolveWritablePathWithFuncs(
	fs afero.Fs,
	root string,
	destination string,
	evalSymlinksFunc func(string) (string, error),
	absFunc func(string) (string, error),
) (string, error) {
	linkReader, ok := fs.(afero.LinkReader)
	if !ok {
		return destination, nil
	}

	resolvedRoot, err := evalSymlinksFunc(root)
	if err != nil {
		return "", err
	}
	resolvedRoot, err = absFunc(resolvedRoot)
	if err != nil {
		return "", err
	}

	lstater, ok := fs.(afero.Lstater)
	if !ok {
		return destination, nil
	}

	info, _, err := lstater.LstatIfPossible(destination)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		target, err := linkReader.ReadlinkIfPossible(destination)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(destination), target)
		}

		resolvedTargetDir, err := evalSymlinksFunc(filepath.Dir(target))
		if err != nil {
			return "", err
		}
		resolvedTarget := filepath.Join(resolvedTargetDir, filepath.Base(target))
		resolvedTarget, err = absFunc(resolvedTarget)
		if err != nil {
			return "", err
		}
		if !pathWithinRoot(resolvedRoot, resolvedTarget) {
			return "", OutsideRootError{Path: destination, ResolvedPath: resolvedTarget, Root: resolvedRoot}
		}
		return resolvedTarget, nil
	}

	resolvedDir, err := evalSymlinksFunc(filepath.Dir(destination))
	if err != nil {
		return "", err
	}
	resolvedDestination := filepath.Join(resolvedDir, filepath.Base(destination))
	resolvedDestination, err = absFunc(resolvedDestination)
	if err != nil {
		return "", err
	}
	if !pathWithinRoot(resolvedRoot, resolvedDestination) {
		return "", OutsideRootError{Path: destination, ResolvedPath: resolvedDestination, Root: resolvedRoot}
	}

	return resolvedDestination, nil
}

func pathWithinRoot(root string, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == ".." {
		return false
	}
	if strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return false
	}
	return !filepath.IsAbs(rel)
}
