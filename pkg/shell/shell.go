// Package shell provides a customizable command-line shell implementation with support
// for built-in commands, external command execution, and I/O redirection.
//
// The shell implements a REPL (Read-Eval-Print Loop) that reads commands from an input
// stream, executes them, and writes results to output streams. It supports both built-in
// commands (echo, exit, type, pwd, cd) and external commands found in the system PATH.
//
// # I/O Redirection
//
// The shell supports standard I/O redirection operators:
//   - >   or 1>   :  Redirect stdout (overwrite)
//   - >>  or 1>>  : Redirect stdout (append)
//   - 2>          : Redirect stderr (overwrite)
//   - 2>>         : Redirect stderr (append)
//
// # Basic Usage
//
// Create a shell with standard I/O streams and run it:
//
//	sh := shell.New(os.Stdin, os.Stdout, os. Stderr)
//	if err := sh.Run(); err != nil {
//	    log.Fatal(err)
//	}
//
// # Testing with Custom Streams
//
// For testing, you can provide custom I/O streams:
//
//	input := strings.NewReader("echo hello\nexit\n")
//	var stdout, stderr bytes.Buffer
//	sh := shell.New(input, &stdout, &stderr)
//	sh.Run()
//	fmt.Println(stdout.String()) // Output: $ hello\n$
//
// # Architecture
//
// The shell uses a modular architecture with pluggable components:
//   - Parser: Tokenizes command lines with quote and escape handling
//   - ArgumentParser: Separates regular arguments from redirection operators
//   - RedirectionManager: Manages file opening and I/O stream binding
//   - Executor:  Executes external commands via os/exec
//
// # Thread Safety
//
// Shell instances are not thread-safe and should be used by a single goroutine.
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

// ErrExit is returned by built-in commands to signal that the shell should
// terminate gracefully.  When a built-in function returns this error, the
// shell's Run method will return nil, indicating a successful exit.
//
// Example usage in a custom builtin:
//
//	shell.builtins["quit"] = func(args []string, s *Shell) error {
//	    return shell.ErrExit
//	}
var ErrExit = errors.New("exit")

// Builtin is the function signature for implementing custom built-in commands.
//
// Built-in commands are executed directly by the shell without spawning
// external processes. They have access to the shell's internal state and
// can perform operations that external commands cannot, such as changing
// the shell's working directory.
//
// Parameters:
//   - args: Command arguments excluding the command name itself.
//     For "echo hello world", args would be []string{"hello", "world"}.
//   - s:  Pointer to the shell instance, providing access to I/O streams
//     (s.Out, s.Err) and other shell facilities.
//
// Returns:
//   - error: Return nil for success, ErrExit to terminate the shell gracefully,
//     or any other error to indicate command failure.  Errors are printed to
//     stderr but do not terminate the shell.
//
// The shell automatically manages I/O redirection for built-in commands,
// temporarily replacing s.Out and s.Err before calling the builtin and
// restoring them afterward.
//
// Example custom builtin:
//
//	myBuiltin := func(args []string, s *Shell) error {
//	    if len(args) == 0 {
//	        fmt.Fprintln(s. Err, "error: missing argument")
//	        return nil // Continue shell even on error
//	    }
//	    fmt.Fprintln(s. Out, "Processing:", args[0])
//	    return nil
//	}
type Builtin func(args []string, s *Shell) error

// Shell represents a command-line shell instance with configurable I/O streams
// and pluggable components for parsing, execution, and redirection.
//
// The shell maintains state for command execution including:
//   - I/O streams for input, output, and error messages
//   - PATH directories for executable lookup
//   - Registry of built-in commands
//   - Parsers for command tokenization and argument processing
//   - Redirection manager for file I/O
//   - Executor for external commands
//
// Shell instances are not safe for concurrent use.  Each instance should be
// used by a single goroutine.
//
// Fields are unexported to maintain encapsulation and prevent external
// modification of internal state.  Use the New constructor to create instances.
type Shell struct {
	in                 *bufio.Reader       // Buffered command input reader
	Out                io.Writer           // Standard output stream (exported for builtin access)
	Err                io.Writer           // Standard error stream (exported for builtin access)
	pathDirs           []string            // Directories from PATH environment variable
	builtins           map[string]Builtin  // Registry of built-in command implementations
	executor           Executor            // External command executor
	parser             Parser              // Command line tokenizer
	argumentParser     *ArgumentParser     // Separates args from redirection operators
	redirectionManager *RedirectionManager // Manages file I/O for redirections
}

// New creates and initializes a new Shell instance with the specified I/O streams.
//
// The constructor sets up all necessary components and initializes the shell's
// state from the environment. The PATH is captured at creation time; subsequent
// changes to the PATH environment variable will not affect this shell instance.
//
// Parameters:
//   - reader: Input stream for reading commands. Use os.Stdin for interactive shells
//     or strings.NewReader/bytes.NewReader for testing and scripting.
//   - out: Output stream for normal command output. Typically os.Stdout.
//   - errw: Output stream for error messages. Typically os.Stderr.
//
// Returns:
//   - *Shell:  Fully initialized shell ready to execute commands via Run().
//
// Initialization steps:
//  1. Reads and parses the PATH environment variable
//  2. Registers built-in commands:  echo, exit, type, pwd, cd
//  3. Initializes command parser with quote and escape handling
//  4. Configures redirection manager with operators:  >, >>, 1>, 1>>, 2>, 2>>
//  5. Sets up default executor for external command execution
//
// Example for interactive shell:
//
//	sh := shell.New(os. Stdin, os.Stdout, os.Stderr)
//	if err := sh.Run(); err != nil {
//	    log.Fatal(err)
//	}
//
// Example for testing:
//
//	input := strings.NewReader("pwd\nls -l\nexit\n")
//	var stdout, stderr bytes.Buffer
//	sh := shell.New(input, &stdout, &stderr)
//	sh.Run()
//	output := stdout.String()
//
// Example for script execution:
//
//	script, _ := os.Open("script.sh")
//	defer script.Close()
//	sh := shell.New(script, os.Stdout, os.Stderr)
//	sh.Run()
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
	return shell
}

// Run starts the shell's read-eval-print loop (REPL).
//
// The method runs an infinite loop that reads commands from the input stream,
// parses and executes them, and displays results.  The loop continues until
// the exit command is executed or a fatal error occurs.
//
// Execution flow for each command:
//  1. Display prompt "$ "
//  2. Read a line of input
//  3. Parse command and arguments (handling quotes and escapes)
//  4. Separate redirection operators from regular arguments
//  5. Apply I/O redirections (open files as needed)
//  6. Execute built-in command or external program
//  7. Clean up resources (close opened files)
//  8. Repeat
//
// Returns:
//   - error: nil on graceful exit via the exit command, or non-nil for
//     fatal errors (I/O failures, parse errors).
//
// Error handling behavior:
//   - I/O read errors:  Returns immediately with the error
//   - Parse errors: Returns immediately with the error
//   - Redirection errors: Prints to stderr and continues to next command
//   - Command not found: Prints to stderr and continues to next command
//   - Command execution errors: Prints to stderr and continues to next command
//   - Exit command (ErrExit): Returns nil for graceful shutdown
//
// Resource management:
//
// The method ensures proper cleanup of file descriptors opened for redirection.
// Cleanup functions are deferred and execute even if command execution fails.
// However, note that defer in loops can accumulate; in practice, cleanup is
// called explicitly before continuing to the next iteration.
//
// Example interactive session:
//
//	$ echo hello
//	hello
//	$ ls > files.txt
//	$ cat files.txt
//	main.go
//	shell.go
//	$ pwd 2> /dev/null
//	/home/user/project
//	$ exit
//
// Example programmatic usage:
//
//	sh := shell.New(os.Stdin, os.Stdout, os. Stderr)
//	if err := sh.Run(); err != nil {
//	    log. Fatalf("Shell terminated with error:  %v", err)
//	}
func (shell *Shell) Run() error {
	for {

		// print $ for user to type in
		fmt.Fprint(shell.Out, "$ ")

		// get user input
		line, err := shell.in.ReadString('\n')

		if err != nil {
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// parse user input command to list of arguments
		parsedArgs, err := shell.parser.Parse(line)

		if err != nil {
			return err
		}

		command := parsedArgs[0]
		args := []string{}
		if len(parsedArgs) > 1 {
			args = parsedArgs[1:]
		}

		// parse list of arguments to obtain proper redirections
		parsedCommand, err := shell.argumentParser.Parse(args)

		baseBindings := &IOBindings{
			Stdin:  shell.in,
			Stdout: shell.Out,
			Stderr: shell.Err,
		}

		// aply redirections to ioBindings for use in builtin and execution commands
		ioBindings, cleanup, err := shell.redirectionManager.ApplyRedirections(parsedCommand.Redirections, *baseBindings)

		if err != nil {
			fmt.Fprintln(shell.Err, "redirection error:", err)
			continue
		}

		if cleanup != nil {
			defer cleanup()
		}

		// execute builtin or external command
		if builtinFunc, ok := shell.builtins[command]; ok {
			// temporarily swap shell I/O for builtins
			prevOut := shell.Out
			prevErr := shell.Err

			shell.Out = ioBindings.Stdout
			shell.Err = ioBindings.Stderr

			err := builtinFunc(parsedCommand.Args, shell)

			// restore original I/O
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

// Lookup searches for an executable in the shell's PATH directories.
//
// The method searches each directory in the PATH (captured during shell
// initialization) for a file matching the given name. It verifies that
// the file exists, is a regular file, and has execute permissions.
//
// Parameters:
//   - name: The name of the executable to find (e.g., "ls", "grep", "cat").
//     This should be just the filename, not a path.
//
// Returns:
//   - path: The full absolute path to the executable if found, empty string otherwise.
//   - found: true if an executable was found and is executable, false otherwise.
//
// Search behavior:
//   - Searches directories in the order they appear in PATH
//   - Returns the first match found (does not search remaining directories)
//   - Checks that the file is a regular file (not a directory or special file)
//   - Verifies execute permission bits (0111) are set
//   - Does not follow symbolic links beyond what filepath.Join provides
//
// The method does not search the current directory unless "." is explicitly
// in the PATH.
//
// Example:
//
//	if path, ok := sh.Lookup("ls"); ok {
//	    fmt. Printf("ls found at: %s\n", path)
//	} else {
//	    fmt. Println("ls not found in PATH")
//	}
//
// Example checking before execution:
//
//	name := "mycommand"
//	if _, found := sh.Lookup(name); ! found {
//	    return fmt.Errorf("%s: command not found", name)
//	}
//
// Note: This method is used internally by the default executor to locate
// external commands before executing them.
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

// registerBuiltins initializes the shell's built-in commands map.
//
// This method is called during shell initialization (in New) and registers
// all default built-in commands. Built-in commands are executed directly
// by the shell without spawning external processes.
//
// Registered built-ins:
//
//   - echo: Prints arguments separated by spaces to stdout.
//     Syntax: echo [args...]
//     Example: echo hello world → "hello world"
//
//   - exit: Terminates the shell gracefully by returning ErrExit.
//     Syntax: exit [code]
//     Note: Exit code parameter is currently ignored.
//
//   - type: Displays information about how a command would be interpreted.
//     Syntax: type <command>
//     Shows whether a command is a builtin or external program.
//     Example: type echo → "echo is a shell builtin"
//     Example: type ls → "ls is /bin/ls"
//
//   - pwd:  Prints the current working directory to stdout.
//     Syntax: pwd
//     Example: pwd → "/home/user/project"
//
//   - cd: Changes the current working directory.
//     Syntax: cd [directory]
//     Supports tilde expansion for home directory.
//     With no args, changes to $HOME.
//     Example: cd /tmp
//     Example: cd ~
//     Example: cd ~/Documents
//
// Error handling:
//
// All built-ins return nil on success or non-fatal errors. They print
// error messages to the shell's Err stream but allow the shell to continue.
// Only the exit command returns ErrExit to terminate the shell.
//
// This method is not exported as built-in registration is handled
// automatically during shell initialization.  Future versions may expose
// a public RegisterBuiltin method for custom extensions.
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
