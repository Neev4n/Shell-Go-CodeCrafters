package shell

import (
	"fmt"
	"io"
	"os"
)

type FileOpener interface {
	OpenRead(name string) (io.ReadCloser, error)
	OpenWrite(name string, flag int, perm os.FileMode) (io.WriteCloser, error)
}

// Default file opener uses real file system in device
type DefaultFileOpener struct{}

func (fp *DefaultFileOpener) OpenRead(name string) (io.ReadCloser, error) {
	return os.Open(name)
}

func (fp *DefaultFileOpener) OpenWrite(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	return os.OpenFile(name, flag, perm)
}

type RedirectionSpec struct {
	Operator string // operators such as (>, 1>, >>, 1>>, 2>, 2>>)
	Target   string // target path
	Index    int    //relative index in args
}

// cleaned up arguments after parsed through for redirection
type ParsedCommand struct {
	Args         []string
	Redirections []RedirectionSpec
}

// handles each type of redirection
type RedirectionHandler interface {
	CanHandle(operator string) bool                                                                           // check for operator
	Validate(redirection RedirectionSpec) error                                                               // check if this redirection is possible
	Apply(redirection RedirectionSpec, ioBindings *IOBindings, opener FileOpener) (cleanup func(), err error) // apply redirection to bindings

}

// handle stdout redirections
type StdoutRedirectionHandler struct {
	Overwrite bool
}

func (handler *StdoutRedirectionHandler) CanHandle(operator string) bool {
	if handler.Overwrite {
		return operator == ">" || operator == "1>"
	}

	return operator == ">>" || operator == "1>>"
}

func (handler *StdoutRedirectionHandler) Validate(redirection RedirectionSpec) error {
	if redirection.Target == "" {
		return ErrMissingRedirectDestination
	}

	return nil
}

func (handler *StdoutRedirectionHandler) Apply(redirection RedirectionSpec, ioBindings *IOBindings, opener FileOpener) (cleanup func(), err error) {

	flag := os.O_CREATE | os.O_WRONLY

	if handler.Overwrite {
		flag |= os.O_TRUNC
	} else {
		flag |= os.O_APPEND
	}

	file, err := opener.OpenWrite(redirection.Target, flag, 0644)

	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", redirection.Target, err)
	}

	ioBindings.Stdout = file
	return func() { file.Close() }, nil

}
