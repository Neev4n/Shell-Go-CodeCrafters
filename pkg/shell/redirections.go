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
	Validate(spec RedirectionSpec) error                                                                      // check if this redirection is possible
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

func (handler *StdoutRedirectionHandler) Validate(spec RedirectionSpec) error {
	if spec.Target == "" {
		return ErrMissingRedirectDestination
	}

	return nil
}

func (handler *StdoutRedirectionHandler) Apply(spec RedirectionSpec, ioBindings *IOBindings, opener FileOpener) (cleanup func(), err error) {

	flag := os.O_CREATE | os.O_WRONLY

	if handler.Overwrite {
		flag |= os.O_TRUNC
	} else {
		flag |= os.O_APPEND
	}

	file, err := opener.OpenWrite(spec.Target, flag, 0644)

	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", spec.Target, err)
	}

	ioBindings.Stdout = file
	return func() { file.Close() }, nil

}

// handle stderr redirections
type StderrRedirectionHandler struct {
	Overwrite bool
}

func (handler *StderrRedirectionHandler) CanHandle(operator string) bool {
	if handler.Overwrite {
		return operator == "2>"
	}

	return operator == "2>>"
}

func (handler *StderrRedirectionHandler) Validate(spec RedirectionSpec) error {
	if spec.Target == "" {
		return ErrMissingRedirectDestination
	}

	return nil
}

func (handler *StderrRedirectionHandler) Apply(spec RedirectionSpec, ioBindings *IOBindings, opener FileOpener) (cleanup func(), err error) {

	flag := os.O_CREATE | os.O_WRONLY

	if handler.Overwrite {
		flag |= os.O_TRUNC
	} else {
		flag |= os.O_APPEND
	}

	file, err := opener.OpenWrite(spec.Target, flag, 0644)

	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", spec.Target, err)
	}

	ioBindings.Stderr = file
	return func() { file.Close() }, nil

}

// shell holds a redirection manager
type RedirectionManager struct {
	handlers   []RedirectionHandler
	fileOpener FileOpener
}

// find the handler for a given operator
func (rManager *RedirectionManager) GetHandler(operator string) (RedirectionHandler, error) {

	for _, handler := range rManager.handlers {
		if handler.CanHandle(operator) {
			return handler, nil
		}
	}

	return nil, fmt.Errorf("unsupported redirection operator: %s", operator)

}

func NewRedirectionManager(fileOpener FileOpener) *RedirectionManager {

	rManager := &RedirectionManager{
		handlers:   []RedirectionHandler{},
		fileOpener: fileOpener,
	}

	rManager.RegisterHandler(&StdoutRedirectionHandler{Overwrite: true})
	rManager.RegisterHandler(&StdoutRedirectionHandler{Overwrite: false})
	rManager.RegisterHandler(&StderrRedirectionHandler{Overwrite: true})
	rManager.RegisterHandler(&StderrRedirectionHandler{Overwrite: false})

	return rManager

}

func (rManager *RedirectionManager) RegisterHandler(handler RedirectionHandler) {
	rManager.handlers = append(rManager.handlers, handler)
}

func (rManager *RedirectionManager) ValidateSpecs(specs []RedirectionSpec) error {

	for _, spec := range specs {

		handler, err := rManager.GetHandler(spec.Operator)

		if err != nil {
			return err
		}

		if err := handler.Validate(spec); err != nil {
			return fmt.Errorf("invalid redirection '%s %s': %w", spec.Operator, spec.Target, err)
		}

	}

	return nil

}

func (rManager *RedirectionManager) ApplyRedirections(specs []RedirectionSpec, baseBindings IOBindings) (IOBindings, func(), error) {

	if err := rManager.ValidateSpecs(specs); err != nil {
		return baseBindings, nil, err
	}

	cleanupFuncs := []func(){}

	bindings := IOBindings{
		Stdin:  baseBindings.Stdin,
		Stdout: baseBindings.Stdout,
		Stderr: baseBindings.Stderr,
	}

	for _, spec := range specs {

		handler, _ := rManager.GetHandler(spec.Operator)

		fn, err := handler.Apply(spec, &bindings, rManager.fileOpener)

		if err != nil {

			// clean up already existing functions
			for _, c := range cleanupFuncs {
				c()
			}

			return baseBindings, nil, err
		}

		if fn != nil {
			cleanupFuncs = append(cleanupFuncs, fn)
		}

	}

	cleanup := func() {
		for _, c := range cleanupFuncs {
			c()
		}
	}

	return bindings, cleanup, nil

}
