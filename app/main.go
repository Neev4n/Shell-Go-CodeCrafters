// Command shell is an interactive command-line shell implementation.
//
// The shell provides a REPL (Read-Eval-Print Loop) interface for executing
// commands, both built-in and external. It supports standard shell features
// including I/O redirection, PATH-based command lookup, and common built-in
// commands.
//
// # Features
//
// Built-in Commands:
//   - echo:   Print arguments to stdout
//   - exit:  Terminate the shell
//   - type: Display command type information
//   - pwd:  Print working directory
//   - cd:   Change directory (with tilde expansion)
//
// External Commands:
//   - Any executable found in PATH
//   - Full argument and quoting support
//
// I/O Redirection:
//   - >   or 1>   :  Redirect stdout (overwrite)
//   - >>  or 1>>  : Redirect stdout (append)
//   - 2>          : Redirect stderr (overwrite)
//   - 2>>         : Redirect stderr (append)
//
// Command Parsing:
//   - Single-quoted strings (literal)
//   - Double-quoted strings (with escape sequences)
//   - Backslash escaping
//   - Whitespace handling
//
// # Installation
//
// Build the shell:
//
//	go build -o shell ./app
//
// Or run directly:
//
//	go run ./app/main.go
//
// # Usage
//
// Start the shell:
//
//	$ ./shell
//	$ echo "Hello, World!"
//	Hello, World!
//	$ ls -la > files.txt
//	$ cat files.txt
//	total 24
//	drwxr-xr-x  4 user  group   128 Jan 15 10:30 .
//	drwxr-xr-x  8 user  group   256 Jan 15 10:00 ..
//	-rw-r--r--  1 user  group  1024 Jan 15 10:30 main.go
//	$ pwd
//	/home/user/project
//	$ cd ~/Documents
//	$ exit
//
// # Environment
//
// The shell reads the following environment variables:
//   - PATH: Colon-separated list of directories to search for executables
//   - HOME: User's home directory (used for tilde expansion in cd)
//
// # Exit Codes
//
//   - 0:   Normal termination (exit command)
//   - 1:   Fatal error (I/O error, parse error)
//
// # Examples
//
// Redirect output to a file:
//
//	$ echo "Log entry" > log.txt
//
// Append to a file:
//
//	$ echo "Another entry" >> log.txt
//
// Redirect stderr:
//
//	$ command-that-fails 2> errors.log
//
// Multiple redirections:
//
//	$ command > output.txt 2> errors.log
//
// Use quotes for arguments with spaces:
//
//	$ echo "hello world"
//	$ echo 'single quotes work too'
//
// Escape special characters:
//
//	$ echo hello\ world
//	$ echo "use \"quotes\" inside"
//
// Check command type:
//
//	$ type echo
//	echo is a shell builtin
//	$ type ls
//	ls is /bin/ls
//
// Change directory:
//
//	$ cd /tmp
//	$ cd ~
//	$ cd ~/projects/myapp
//
// # Architecture
//
// The shell uses a modular architecture:
//
//	┌──────────────────────────────────────┐
//	│           main.go                    │
//	│  (Application Entry Point)           │
//	└────────────┬─────────────────────────┘
//	             │
//	             ▼
//	┌──────────────────────────────────────┐
//	│         Shell (REPL)                 │
//	│  - Read commands                     │
//	│  - Parse input                       │
//	│  - Execute commands                  │
//	│  - Handle errors                     │
//	└──┬───────┬──────────┬────────────┬───┘
//	   │       │          │            │
//	   ▼       ▼          ▼            ▼
//	Parser  Executor  Redirections  Builtins
//
// # Limitations
//
// The following features are not currently supported:
//   - Pipes (|)
//   - Background jobs (&)
//   - Command substitution (`cmd` or $(cmd))
//   - Wildcards/globbing (*, ?, [])
//   - Environment variable expansion (except ~ in cd)
//   - Job control (fg, bg, jobs)
//   - Signal handling (Ctrl+C, Ctrl+Z)
//   - Command history
//   - Tab completion
//
// # See Also
//
// Package documentation:  https://pkg.go.dev/github.com/Neev4n/CodeCrafters-Shell-GO/codecrafters-shell-go/pkg/shell
//
// # Naveen
//
// Created as part of CodeCrafters Shell Challenge

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Neev4n/CodeCrafters-Shell-GO/codecrafters-shell-go/pkg/shell"
)

// Ensures gofmt doesn't remove the "fmt" and "os" imports in stage 1 (feel free to remove this!)
var _ = fmt.Fprint
var _ = os.Stdout

// main is the entry point for the shell application.
//
// The function initializes a new shell with standard I/O streams (stdin,
// stdout, stderr) and starts the interactive REPL loop.  If the shell
// encounters a fatal error, the program exits with status code 1.
//
// Execution flow:
//  1. Create shell instance with os.Stdin, os.Stdout, os.Stderr
//  2. Start the REPL with shell.Run()
//  3. Run continues until:
//     - User executes 'exit' command (normal termination, exit code 0)
//     - Fatal I/O error occurs (abnormal termination, exit code 1)
//     - Fatal parse error occurs (abnormal termination, exit code 1)
//
// The shell runs in the foreground and blocks until termination.
//
// Exit behavior:
//   - exit command:       shell.Run() returns nil, main exits normally (code 0)
//   - I/O error:         shell.Run() returns error, log.Fatal exits with code 1
//   - Parse error:       shell.Run() returns error, log.Fatal exits with code 1
//
// Standard streams:
//   - os.Stdin:  Used for reading user commands
//   - os.Stdout: Used for command output and prompts
//   - os. Stderr: Used for error messages
//
// Example normal session:
//
//	$ ./shell
//	$ echo hello
//	hello
//	$ exit
//	[process exits with code 0]
//
// Example with error:
//
//	$ ./shell
//	$ echo "unclosed quote
//	[shell terminates with parse error, code 1]
//
// For non-interactive use (scripting), redirect stdin from a file:
//
//	$ ./shell < script.sh
//
// For debugging, redirect stderr to a log file:
//
//	$ ./shell 2> debug.log
func main() {

	s := shell.New(os.Stdin, os.Stdout, os.Stderr)

	if err := s.Run(); err != nil {
		log.Fatal(err)
	}

}
