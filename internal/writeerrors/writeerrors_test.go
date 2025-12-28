package writeerrors

import (
	"errors"
	"io"
	"testing"
)

func TestIsBrokenPipeReturnsTrueForClosedPipe(t *testing.T) {
	if !IsBrokenPipe(io.ErrClosedPipe) {
		t.Fatal("expected io.ErrClosedPipe to be treated as broken pipe")
	}
}

func TestIsBrokenPipeReturnsTrueForPlatformPipe(t *testing.T) {
	if !IsBrokenPipe(brokenPipeErr()) {
		t.Fatal("expected platform broken pipe error to be treated as broken pipe")
	}
}

func TestIsBrokenPipeReturnsFalseForOtherErrors(t *testing.T) {
	if IsBrokenPipe(errors.New("not a pipe")) {
		t.Fatal("unexpected broken pipe detection for generic error")
	}
}
