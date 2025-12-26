package shell

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// exit error
var ErrExit = errors.New("exit")
var ErrMissingRedirectDestination = errors.New("missing redirect destination")

// type Builtin
type Builtin func(args []string, s *Shell) error

type redirectType int

const (
	in redirectType = iota
	out
	err
)

// type Redirection struct {
// 	redirectType redirectType
// 	flag         int
// }

// type Shell
type Shell struct {
	in                 *bufio.Reader
	Out                io.Writer
	Err                io.Writer
	pathDirs           []string
	builtins           map[string]Builtin
	executor           Executor
	parser             Parser
	argumentParser     *ArgumentParser
	redirectionManager *RedirectionManager
}

// func New
func New(reader io.Reader, out, errw io.Writer) *Shell {
	path := os.Getenv("PATH")
	var dirs []string

	if path != "" {
		dirs = strings.Split(path, string(os.PathListSeparator))
	}

	shell := &Shell{
		in:       bufio.NewReader(reader),
		Out:      out,
		Err:      errw,
		pathDirs: dirs,
		builtins: make(map[string]Builtin),
	}

	shell.executor = &DefaultExecutuor{LookupFunc: shell.Lookup}
	shell.parser = NewDefaultParser()
	shell.redirectionManager = NewRedirectionManager(&DefaultFileOpener{})
	shell.argumentParser = NewArgumentParser(shell.redirectionManager)
	shell.registerBuiltins()
	//shell.registerRedirections()
	return shell
}

//func Run

func (shell *Shell) Run() error {
	for {

		fmt.Fprint(shell.Out, "$ ")

		line, err := shell.in.ReadString('\n')

		if err != nil {
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parsedArgs, err := shell.parser.Parse(line)

		if err != nil {
			return err
		}

		command := parsedArgs[0]
		args := []string{}
		if len(parsedArgs) > 1 {
			args = parsedArgs[1:]
		}

		parsedCommand, err := shell.argumentParser.Parse(args)

		baseBindings := &IOBindings{
			Stdin:  shell.in,
			Stdout: shell.Out,
			Stderr: shell.Err,
		}

		ioBindings, cleanup, err := shell.redirectionManager.ApplyRedirections(parsedCommand.Redirections, *baseBindings)

		if err != nil {
			fmt.Fprintln(shell.Err, "redirection error:", err)
			continue
		}

		if cleanup != nil {
			defer cleanup()
		}

		// Execute builtin or external command
		if builtinFunc, ok := shell.builtins[command]; ok {
			// Temporarily swap shell I/O for builtins
			prevOut := shell.Out
			prevErr := shell.Err

			shell.Out = ioBindings.Stdout
			shell.Err = ioBindings.Stderr

			err := builtinFunc(parsedCommand.Args, shell)

			// Restore original I/O
			shell.Out = prevOut
			shell.Err = prevErr

			if err != nil {

				// if exiting run clean up
				if errors.Is(err, ErrExit) {
					if cleanup != nil {
						cleanup()
					}
					return nil
				}

				// else it is a built in error
				fmt.Fprintln(shell.Err, "builtin error:", err)
			}

			if cleanup != nil {
				cleanup()
			}
			continue
		}

		//execute command
		exitCode, err := shell.executor.Execute(context.Background(), command, parsedCommand.Args, ioBindings)

		if errors.Is(err, ErrNotFound) {
			fmt.Fprintln(shell.Err, command+": command not found")
			continue
		}

		if err != nil {
			fmt.Fprintln(shell.Err, "error running command:", err)
			continue
		}

		if cleanup != nil {
			cleanup()
		}

		_ = exitCode

	}

}

// func Lookup
func (shell *Shell) Lookup(name string) (string, bool) {

	for _, directory := range shell.pathDirs {

		pathToCheck := filepath.Join(directory, name)

		if info, err := os.Stat(pathToCheck); err == nil {
			if info.Mode().IsRegular() && info.Mode()&0111 != 0 {
				return pathToCheck, true
			}
		}
	}

	return "", false

}

// registerBuiltins initializes the shell's built-in commands map with the following commands:
//
// - echo: prints the arguments separated by spaces to stdout
// - exit: terminates the shell by returning ErrExit
// - type: displays information about a command (whether it's a builtin or external command)
// - pwd: prints the current working directory
// - cd: changes the current working directory, supports ~ expansion for home directory
//
// Each builtin is registered as a function that takes a slice of arguments and a shell pointer,
// and returns an error if the operation fails.
func (shell *Shell) registerBuiltins() {

	shell.builtins["echo"] = func(args []string, shell *Shell) error {
		fmt.Fprintln(shell.Out, strings.Join(args, " "))
		return nil
	}

	shell.builtins["exit"] = func(args []string, shell *Shell) error {
		return ErrExit
	}

	shell.builtins["type"] = func(args []string, shell *Shell) error {

		if len(args) == 0 {
			fmt.Fprintln(shell.Out, "type: usage: type NAME")
			return nil
		}

		name := args[0]

		// check builts in
		if _, ok := shell.builtins[name]; ok {
			fmt.Fprintln(shell.Out, name, "is a shell builtin")
			return nil
		}

		if path, ok := shell.Lookup(name); ok {
			fmt.Fprintln(shell.Out, name, "is", path)
			return nil
		}

		fmt.Fprintln(shell.Out, name+": not found")
		return nil
	}

	shell.builtins["pwd"] = func(args []string, shell *Shell) error {
		dir, err := os.Getwd()
		if err == nil {
			fmt.Fprintln(shell.Out, dir)
		} else {
			fmt.Fprintln(shell.Err, "error finding directory:", err)
		}

		return nil
	}

	shell.builtins["cd"] = func(args []string, shell *Shell) error {

		var target string

		if len(args) == 0 {
			target = os.Getenv("HOME")
			if target == "" {
				return nil //no home variable set
			}

		} else {
			target = args[0]
		}

		if strings.HasSuffix(target, "~") {
			home := os.Getenv("HOME")
			if home == "" {
				fmt.Fprintln(shell.Err, "cd: HOME not set")
				return nil
			}

			if target == "~" {
				target = home
			} else if strings.HasSuffix(target, "~/") {
				target = filepath.Join(home, target[2:])
			} else {
				fmt.Fprintf(shell.Err, "cd: unsupported user expansion: %s\n", target)
				return nil
			}
		}

		if err := os.Chdir(target); err != nil {

			if os.IsNotExist(err) {
				fmt.Fprintf(shell.Err, "cd: %s: No such file or directory\n", target)
			} else if os.IsPermission(err) {
				fmt.Fprintf(shell.Err, "cd: %s: Permission denied\n", target)
			} else {
				fmt.Fprintf(shell.Err, "cd: %s: %v", target, err)
			}

		}

		return nil

	}
}
