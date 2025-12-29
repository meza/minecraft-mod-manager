package clierrors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type exitCodeStub struct {
	code int
}

func (stub exitCodeStub) Error() string {
	return "exit code stub"
}

func (stub exitCodeStub) ExitCode() int {
	return stub.code
}

func TestMarkHandledReturnsNilForNil(t *testing.T) {
	assert.Nil(t, MarkHandled(nil))
}

func TestMarkHandledWrapsError(t *testing.T) {
	baseErr := errors.New("boom")
	wrapped := MarkHandled(baseErr)

	assert.ErrorIs(t, wrapped, baseErr)
	assert.True(t, IsHandled(wrapped))
}

func TestMarkHandledKeepsHandledError(t *testing.T) {
	baseErr := errors.New("boom")
	wrapped := MarkHandled(baseErr)

	assert.Same(t, wrapped, MarkHandled(wrapped))
}

func TestIsHandledReturnsFalseForNil(t *testing.T) {
	assert.False(t, IsHandled(nil))
}

func TestHandledErrorExitCodeFallsBackToDefault(t *testing.T) {
	baseErr := errors.New("boom")
	handled := MarkHandled(baseErr)

	assert.Equal(t, "boom", handled.Error())

	var exitCoder interface {
		ExitCode() int
	}
	assert.True(t, errors.As(handled, &exitCoder))
	assert.Equal(t, 1, exitCoder.ExitCode())
}

func TestHandledErrorExitCodeDelegates(t *testing.T) {
	baseErr := exitCodeStub{code: 7}
	handled := MarkHandled(baseErr)

	var exitCoder interface {
		ExitCode() int
	}
	assert.True(t, errors.As(handled, &exitCoder))
	assert.Equal(t, 7, exitCoder.ExitCode())
}
