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

// type Builtin
type Builtin func(args []string, s *Shell) error

// type Shell
type Shell struct {
	in       *bufio.Reader
	Out      io.Writer
	Err      io.Writer
	pathDirs []string
	builtins map[string]Builtin
	executor Executor
	parser   Parser
}

// func New
func New(reader io.Reader, out, errw io.Writer) *Shell {
	path := os.Getenv("PATH")
	var dirs []string

	if path != "" {
		dirs = strings.Split(path, string(os.PathListSeparator))
	}

	s := &Shell{
		in:       bufio.NewReader(reader),
		Out:      out,
		Err:      errw,
		pathDirs: dirs,
		builtins: make(map[string]Builtin),
	}

	s.executor = &DefaultExecutuor{LookupFunc: s.Lookup}
	s.parser = NewDefaultParser()
	s.registerBuiltins()
	return s
}

//func Run

func (s *Shell) Run() error {
	for {
		fmt.Fprint(s.Out, "$ ")

		line, err := s.in.ReadString('\n')

		if err != nil {
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields, err := s.parser.Parse(line)

		if err != nil {
			return err
		}

		cmd := fields[0]
		args := []string{}
		if len(fields) > 1 {
			args = fields[1:]
		}

		// check built ins
		if fn, ok := s.builtins[cmd]; ok {
			if err := fn(args, s); err != nil {
				if errors.Is(err, ErrExit) {
					return nil
				}

				fmt.Fprintln(s.Err, "builtin error:", err)
			}
			continue
		}

		ioBinding := IOBindings{
			Stdin:  nil,
			Stdout: s.Out,
			Stderr: s.Err,
		}

		exitCode, err := s.executor.Execute(context.Background(), cmd, args, ioBinding)

		if errors.Is(err, ErrNotFound) {
			fmt.Fprintln(s.Out, cmd+": command not found")
			continue
		}

		if err != nil {
			fmt.Fprintln(s.Err, "error running command:", err)
			continue
		}

		_ = exitCode

	}

}

// func Lookup
func (s *Shell) Lookup(name string) (string, bool) {

	for _, dir := range s.pathDirs {

		pathToCheck := filepath.Join(dir, name)

		if info, err := os.Stat(pathToCheck); err == nil {
			if info.Mode().IsRegular() && info.Mode()&0111 != 0 {
				return pathToCheck, true
			}
		}
	}

	return "", false

}

func (s *Shell) registerBuiltins() {

	s.builtins["echo"] = func(args []string, s *Shell) error {
		fmt.Fprintln(s.Out, strings.Join(args, " "))
		return nil
	}

	s.builtins["exit"] = func(args []string, s *Shell) error {
		return ErrExit
	}

	s.builtins["type"] = func(args []string, s *Shell) error {

		if len(args) == 0 {
			fmt.Fprintln(s.Out, "type: usage: type NAME")
			return nil
		}

		name := args[0]

		// check builts in
		if _, ok := s.builtins[name]; ok {
			fmt.Fprintln(s.Out, name, "is a shell builtin")
			return nil
		}

		if path, ok := s.Lookup(name); ok {
			fmt.Fprintln(s.Out, name, "is", path)
			return nil
		}

		fmt.Fprintln(s.Out, name+": not found")
		return nil
	}

	s.builtins["pwd"] = func(args []string, s *Shell) error {
		dir, err := os.Getwd()
		if err == nil {
			fmt.Fprintln(s.Out, dir)
		} else {
			fmt.Fprintln(s.Err, "error finding directory:", err)
		}

		return nil
	}

	s.builtins["cd"] = func(args []string, s *Shell) error {

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
				fmt.Fprintln(s.Err, "cd: HOME not set")
				return nil
			}

			if target == "~" {
				target = home
			} else if strings.HasSuffix(target, "~/") {
				target = filepath.Join(home, target[2:])
			} else {
				fmt.Fprintf(s.Err, "cd: unsupported user expansion: %s\n", target)
				return nil
			}
		}

		if err := os.Chdir(target); err != nil {

			if os.IsNotExist(err) {
				fmt.Fprintf(s.Err, "cd: %s: No such file or directory\n", target)
			} else if os.IsPermission(err) {
				fmt.Fprintf(s.Err, "cd: %s: Permission denied\n", target)
			} else {
				fmt.Fprintf(s.Err, "cd: %s: %v", target, err)
			}

		}

		return nil

	}
}
