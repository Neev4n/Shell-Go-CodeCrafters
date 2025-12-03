package shell

import (
	"bufio"
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

		fields := strings.Fields(line)
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

		// not found
		fmt.Fprintln(s.Out, cmd+": command not found")

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
