package shell

import (
	"context"
	"errors"
	"io"
	"os/exec"
)

// Executor defines the interface for executing external commands.
//
// Executors are responsible for:
//  1. Locating the executable in the file system
//  2. Spawning the external process
//  3. Binding I/O streams (stdin, stdout, stderr)
//  4. Waiting for completion
//  5. Returning the exit code
//
// The interface allows for different execution strategies, such as:
//   - Real process execution (DefaultExecutor)
//   - Mock execution for testing
//   - Sandboxed execution
//   - Remote execution
//
// Example mock implementation for testing:
//
//	type MockExecutor struct {
//	    ExecuteFunc func(ctx context.Context, name string, args []string, io IOBindings) (int, error)
//	}
//
//	func (m *MockExecutor) Execute(ctx context. Context, name string, args []string, io IOBindings) (int, error) {
//	    return m.ExecuteFunc(ctx, name, args, io)
//	}
type Executor interface {
	// Execute runs an external command and waits for it to complete.
	//
	// The method locates the executable, spawns a new process, binds I/O streams,
	// and waits for the process to exit. It respects context cancellation and
	// can terminate long-running commands.
	//
	// Parameters:
	//   - ctx:   Context for cancellation and timeouts.  When cancelled, the process
	//           is terminated (SIGKILL on Unix).
	//   - name: Name of the command to execute (e.g., "ls", "grep", "cat").
	//           This is the name as it appears in the command line, not the full path.
	//   - args: Command arguments, not including the command name itself.
	//           For "ls -la /tmp", args would be []string{"-la", "/tmp"}.
	//   - io:   I/O bindings for stdin, stdout, and stderr streams.
	//
	// Returns:
	//   - int:    Exit code of the process (0 for success, non-zero for failure).
	//            Returns -1 if the command is not found or other execution errors occur.
	//   - error: ErrNotFound if the executable is not found in PATH,
	//            nil otherwise (exit codes are returned as the int, not as errors).
	//
	// Exit code behavior:
	//   - 0:      Successful execution
	//   - 1-255: Command-specific error codes
	//   - -1:    Command not found or execution failure
	//
	// Example:
	//
	//	executor := &DefaultExecutor{LookupFunc: shell. Lookup}
	//	bindings := IOBindings{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr}
	//	exitCode, err := executor.Execute(context.Background(), "ls", []string{"-la"}, bindings)
	//	if err != nil {
	//	    if errors.Is(err, ErrNotFound) {
	//	        fmt.Println("ls:  command not found")
	//	    }
	//	}
	Execute(ctx context.Context, name string, args []string, io IOBindings) (int, error)
}

// ErrNotFound is returned when an executable cannot be found in the PATH.
//
// This error is returned by Execute implementations when the command lookup
// fails, indicating that the requested program does not exist in any of the
// searched directories.
//
// Example:
//
//	exitCode, err := executor.Execute(ctx, "nonexistent", []string{}, bindings)
//	if errors.Is(err, ErrNotFound) {
//	    fmt.Fprintln(os.Stderr, "nonexistent: command not found")
//	}
var ErrNotFound = errors.New("not found")

// IOBindings represents the I/O streams for command execution.
//
// Each binding connects a standard file descriptor to an io.Reader or io.Writer:
//   - Stdin  (fd 0): Input source for the command
//   - Stdout (fd 1): Output destination for normal output
//   - Stderr (fd 2): Output destination for error messages
//
// These bindings are applied to the external process, allowing redirection
// of I/O streams without the command being aware.
//
// Example (normal execution):
//
//	bindings := IOBindings{
//	    Stdin:  os.Stdin,
//	    Stdout: os.Stdout,
//	    Stderr: os.Stderr,
//	}
//
// Example (with redirection):
//
//	outputFile, _ := os.Create("output.txt")
//	defer outputFile. Close()
//	bindings := IOBindings{
//	    Stdin:  os.Stdin,
//	    Stdout: outputFile,  // stdout redirected to file
//	    Stderr: os.Stderr,
//	}
//
// Example (capturing output):
//
//	var stdout bytes.Buffer
//	bindings := IOBindings{
//	    Stdin:  strings.NewReader("input data\n"),
//	    Stdout: &stdout,
//	    Stderr: os.Stderr,
//	}
//	// After execution, stdout. String() contains command output
type IOBindings struct {
	Stdin  io.Reader // Input stream for the command (file descriptor 0)
	Stdout io.Writer // Output stream for normal output (file descriptor 1)
	Stderr io.Writer // Output stream for error messages (file descriptor 2)
}

// DefaultExecutor executes external commands using os/exec.
//
// This is the production implementation that spawns real processes on the
// operating system. It uses the provided LookupFunc to locate executables
// in the PATH and exec. CommandContext to spawn processes.
//
// Features:
//   - Context-aware execution (respects cancellation)
//   - Proper exit code handling
//   - I/O stream binding
//   - PATH-based executable lookup
//
// The executor sets up the process with:
//   - The full path to the executable
//   - Command arguments (with argv[0] set to the command name)
//   - I/O stream bindings from IOBindings
//
// Exit code semantics:
//   - Returns actual exit code for normal termination
//   - Returns -1 for abnormal termination or execution errors
//   - Returns -1 with ErrNotFound if executable not in PATH
//
// Note: The struct name has a typo ("Executuor" instead of "Executor").
// This is maintained for backward compatibility but may be fixed in a future version.
//
// Example:
//
//	executor := &DefaultExecutor{LookupFunc: shell.Lookup}
//	bindings := IOBindings{
//	    Stdin:  os.Stdin,
//	    Stdout: os.Stdout,
//	    Stderr: os.Stderr,
//	}
//	exitCode, err := executor.Execute(context.Background(), "ls", []string{"-la", "/tmp"}, bindings)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Command exited with code: %d\n", exitCode)
type DefaultExecutor struct {
	// LookupFunc locates an executable by name, returning its full path.
	//
	// The function should search the PATH and return:
	//   - Full path to the executable if found
	//   - true if found, false otherwise
	//
	// Typically this is set to shell.Lookup, which searches PATH directories
	// and verifies execute permissions.
	//
	// Example:
	//
	//	executor := &DefaultExecutuor{
	//	    LookupFunc: func(name string) (string, bool) {
	//	        return exec.LookPath(name) // Alternative using standard library
	//	    },
	//	}
	LookupFunc func(name string) (string, bool)
}

// Execute runs an external command using os/exec.
//
// The method performs the following steps:
//  1. Lookup the executable using LookupFunc
//  2. Create a CommandContext with the provided context
//  3. Set command arguments (with argv[0] as the command name)
//  4. Bind I/O streams from IOBindings
//  5. Run the command and wait for completion
//  6. Extract and return the exit code
//
// Context handling:
//   - If ctx is cancelled, the process is killed (SIGKILL on Unix)
//   - Use context.WithTimeout for commands with time limits
//   - Use context. Background for commands without timeouts
//
// Argument handling:
//   - The args parameter should NOT include the command name
//   - The method sets argv[0] to the command name automatically
//   - This matches shell behavior where argv[0] is the program name
//
// Exit code extraction:
//   - Normal exit (status 0):     Returns 0, nil
//   - Normal exit (status N):     Returns N, nil
//   - Abnormal termination:         Returns -1, nil
//   - Command not found:           Returns -1, ErrNotFound
//
// I/O binding:
//   - Only Stdout and Stderr are bound (Stdin could be added if needed)
//   - Streams are connected directly to the process
//   - No buffering is added by the executor
//
// Parameters:
//   - ctx:  Context for cancellation/timeout
//   - name: Command name (e.g., "ls", "grep")
//   - args: Command arguments (e.g., []string{"-la", "/tmp"})
//   - io:   I/O stream bindings
//
// Returns:
//   - int:   Exit code (-1 if not found or error, 0-255 for normal exit)
//   - error: ErrNotFound if executable not in PATH, nil otherwise
//
// Examples:
//
// Basic execution:
//
//	executor := &DefaultExecutuor{LookupFunc: shell.Lookup}
//	bindings := IOBindings{Stdout: os.Stdout, Stderr: os.Stderr}
//	exitCode, err := executor. Execute(context.Background(), "ls", []string{"-la"}, bindings)
//
// With timeout:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//	exitCode, err := executor.Execute(ctx, "sleep", []string{"10"}, bindings)
//	// Command is killed after 5 seconds
//
// With output capture:
//
//	var stdout bytes.Buffer
//	bindings := IOBindings{Stdout: &stdout, Stderr:  os.Stderr}
//	exitCode, err := executor.Execute(ctx, "echo", []string{"hello"}, bindings)
//	output := stdout.String() // "hello\n"
//
// Handling not found:
//
//	exitCode, err := executor.Execute(ctx, "nonexistent", []string{}, bindings)
//	if errors.Is(err, ErrNotFound) {
//	    fmt.Fprintln(os.Stderr, "nonexistent: command not found")
//	    return
//	}
//
// Checking exit code:
//
//	exitCode, err := executor.Execute(ctx, "grep", []string{"pattern", "file. txt"}, bindings)
//	if err != nil {
//	    return err
//	}
//	if exitCode != 0 {
//	    fmt.Printf("grep exited with code %d\n", exitCode)
//	}
func (e *DefaultExecutor) Execute(ctx context.Context, name string, args []string, io IOBindings) (int, error) {

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
