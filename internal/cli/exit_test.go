package cli

import (
	"errors"
	"testing"
)

type testExitError struct{}

func (testExitError) Error() string {
	return "check failed"
}

func (testExitError) ExitCode() int {
	return 2
}

func TestExitCode(t *testing.T) {
	if got := ExitCode(nil); got != 0 {
		t.Fatalf("nil exit code: got %d, want 0", got)
	}
	if got := ExitCode(errors.New("failure")); got != 1 {
		t.Fatalf("generic exit code: got %d, want 1", got)
	}
	if got := ExitCode(testExitError{}); got != 2 {
		t.Fatalf("coded exit code: got %d, want 2", got)
	}
}
