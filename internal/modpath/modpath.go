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

	lstater, ok := fs.(afero.Lstater)
	if !ok {
		return destination, nil
	}

	resolvedRoot, err := resolveRootPath(root, evalSymlinksFunc, absFunc)
	if err != nil {
		return "", err
	}

	info, _, err := lstater.LstatIfPossible(destination)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		resolvedTarget, resolveErr := resolveSymlinkTarget(destination, linkReader, evalSymlinksFunc, absFunc)
		if resolveErr != nil {
			return "", resolveErr
		}
		return ensureWithinRoot(resolvedRoot, destination, resolvedTarget)
	}

	resolvedDestination, err := resolveDestinationPath(destination, evalSymlinksFunc, absFunc)
	if err != nil {
		return "", err
	}

	return ensureWithinRoot(resolvedRoot, destination, resolvedDestination)
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

func resolveRootPath(root string, evalSymlinksFunc func(string) (string, error), absFunc func(string) (string, error)) (string, error) {
	resolvedRoot, err := evalSymlinksFunc(root)
	if err != nil {
		return "", err
	}
	resolvedRoot, err = absFunc(resolvedRoot)
	if err != nil {
		return "", err
	}
	return resolvedRoot, nil
}

func resolveDestinationPath(destination string, evalSymlinksFunc func(string) (string, error), absFunc func(string) (string, error)) (string, error) {
	resolvedDir, err := evalSymlinksFunc(filepath.Dir(destination))
	if err != nil {
		return "", err
	}
	resolvedDestination := filepath.Join(resolvedDir, filepath.Base(destination))
	resolvedDestination, err = absFunc(resolvedDestination)
	if err != nil {
		return "", err
	}
	return resolvedDestination, nil
}

func resolveSymlinkTarget(
	destination string,
	linkReader afero.LinkReader,
	evalSymlinksFunc func(string) (string, error),
	absFunc func(string) (string, error),
) (string, error) {
	target, readlinkErr := linkReader.ReadlinkIfPossible(destination)
	if readlinkErr != nil {
		return "", readlinkErr
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(destination), target)
	}

	resolvedTargetDir, resolveDirErr := evalSymlinksFunc(filepath.Dir(target))
	if resolveDirErr != nil {
		return "", resolveDirErr
	}
	resolvedTarget := filepath.Join(resolvedTargetDir, filepath.Base(target))
	resolvedTarget, resolveErr := absFunc(resolvedTarget)
	if resolveErr != nil {
		return "", resolveErr
	}
	return resolvedTarget, nil
}

func ensureWithinRoot(root string, originalPath string, resolvedPath string) (string, error) {
	if !pathWithinRoot(root, resolvedPath) {
		return "", OutsideRootError{Path: originalPath, ResolvedPath: resolvedPath, Root: root}
	}
	return resolvedPath, nil
}
