package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
