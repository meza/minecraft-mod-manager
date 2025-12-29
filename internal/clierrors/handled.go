package clierrors

import "errors"

type handledError struct {
	err error
}

func (handled *handledError) Error() string {
	return handled.err.Error()
}

func (handled *handledError) Unwrap() error {
	return handled.err
}

func (handled *handledError) ExitCode() int {
	type exitCoder interface {
		ExitCode() int
	}

	var exitErr exitCoder
	if errors.As(handled.err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}

// MarkHandled wraps an error that has already been printed to the user.
// Cobra should not print these errors again.
func MarkHandled(err error) error {
	if err == nil {
		return nil
	}
	var handled *handledError
	if errors.As(err, &handled) {
		return err
	}
	return &handledError{err: err}
}

// IsHandled reports whether the error has already been printed to the user.
func IsHandled(err error) bool {
	if err == nil {
		return false
	}
	var handled *handledError
	return errors.As(err, &handled)
}
