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

		for _, dir := range s.pathDirs {

			pathToCheck := filepath.Join(dir, name)

			if info, err := os.Stat(pathToCheck); err == nil {
				if info.Mode().IsRegular() && info.Mode()&0111 != 0 {
					fmt.Fprintln(s.Out, name, "is", pathToCheck)
					return nil
				}
			}
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
}
