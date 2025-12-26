package shell

import (
	"context"
	"errors"
	"io"
	"os/exec"
)

type Executor interface {
	Execute(ctx context.Context, name string, args []string, io IOBindings) (int, error)
}

var ErrNotFound = errors.New("not found")

type IOBindings struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type DefaultExecutuor struct {
	LookupFunc func(name string) (string, bool)
}

func (e *DefaultExecutuor) Execute(ctx context.Context, name string, args []string, io IOBindings) (int, error) {

	path, ok := e.LookupFunc(name)

	if !ok {
		return -1, ErrNotFound
	}

	externalCmd := exec.CommandContext(ctx, path, args...)
	externalCmd.Args = append([]string{name}, args...)
	externalCmd.Stdout = io.Stdout
	externalCmd.Stderr = io.Stderr

	if err := externalCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}

		return -1, nil
	}

	return 0, nil

}
