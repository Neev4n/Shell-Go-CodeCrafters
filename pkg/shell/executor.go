package shell

import (
	"context"
	"errors"
	"io"
	"os/exec"
)

var ErrNotFound = errors.New("not found")

type IOBindings struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type DefaultExectuor struct {
	LookupFunc func(name string) (string, bool)
}

func (e *DefaultExectuor) Execute(ctx context.Context, name string, args []string, io IOBindings) (int, error) {

	path, ok := e.LookupFunc(name)

	if !ok {
		return -1, ErrNotFound
	}

	extCmd := exec.CommandContext(ctx, path, args...)
	extCmd.Args = append([]string{name}, args...)
	extCmd.Stdout = io.Stdout
	extCmd.Stderr = io.Stderr

	if err := extCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}

		return -1, nil
	}

	return 0, nil

}
