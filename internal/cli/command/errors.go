package command

import "errors"

type exitError struct {
	code int
	err  error
}

func (e exitError) Error() string {
	return e.err.Error()
}

func (e exitError) Unwrap() error {
	return e.err
}

func (e exitError) ExitCode() int {
	return e.code
}

func checkFailure(message string) error {
	return exitError{code: 2, err: errors.New(message)}
}
