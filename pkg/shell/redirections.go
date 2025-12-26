package shell

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// FileOpener abstracts file system operations for I/O redirection.
//
// This interface allows the redirection system to be tested without
// touching the real file system.  Implementations can provide mock
// file systems, in-memory buffers, or other I/O abstractions.
//
// The interface separates read and write operations to allow different
// permissions and flags for each direction.
//
// Example mock implementation for testing:
//
//	type MockFileOpener struct {
//	    files map[string]*bytes.Buffer
//	}
//
//	func (m *MockFileOpener) OpenWrite(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
//	    buf := &bytes.Buffer{}
//	    m.files[name] = buf
//	    return &mockWriteCloser{buf}, nil
//	}
type FileOpener interface {
	// OpenRead opens a file for reading.
	//
	// Parameters:
	//   - name:  Path to the file to open
	//
	// Returns:
	//   - io.ReadCloser: Reader for the file contents
	//   - error: os.ErrNotExist if file doesn't exist, or permission errors
	OpenRead(name string) (io.ReadCloser, error)

	// OpenWrite opens a file for writing with specified flags and permissions.
	//
	// Parameters:
	//   - name: Path to the file to open/create
	//   - flag:  File opening flags (os.O_CREATE, os.O_TRUNC, os.O_APPEND, etc.)
	//   - perm: File permissions if created (typically 0644)
	//
	// Returns:
	//   - io.WriteCloser: Writer for the file
	//   - error: Permission errors, path errors, or other I/O failures
	OpenWrite(name string, flag int, perm os.FileMode) (io.WriteCloser, error)
}

// ErrMissingRedirectDestination is returned when a redirection operator
// is encountered without a following target file path.
//
// This error occurs during argument parsing when a command ends with a
// redirection operator that requires a target.
//
// Example command that triggers this error:
//
//	$ echo hello >
//	parse error: missing target for redirection '>' at position 2
var ErrMissingRedirectDestination = errors.New("missing redirect destination")

// DefaultFileOpener implements FileOpener using the real file system.
//
// This is the production implementation used by the shell for actual
// I/O redirection.  It directly wraps os.Open and os.OpenFile.
//
// Example:
//
//	opener := &DefaultFileOpener{}
//	file, err := opener.OpenWrite("output.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
type DefaultFileOpener struct{}

// OpenRead opens a file for reading using os.Open.
//
// Parameters:
//   - name: Path to the file to open
//
// Returns:
//   - io.ReadCloser: File handle for reading
//   - error:  os.ErrNotExist if file not found, permission errors, etc.
//
// Example:
//
//	opener := &DefaultFileOpener{}
//	file, err := opener.OpenRead("input.txt")
//	if err != nil {
//	    return err
//	}
//	defer file.Close()
func (fp *DefaultFileOpener) OpenRead(name string) (io.ReadCloser, error) {
	return os.Open(name)
}

// OpenWrite opens a file for writing using os.OpenFile.
//
// Parameters:
//   - name: Path to the file to open/create
//   - flag: File opening flags (os.O_CREATE, os. O_WRONLY, os.O_TRUNC, os.O_APPEND)
//   - perm: File permissions if created (e.g., 0644 for rw-r--r--)
//
// Returns:
//   - io.WriteCloser: File handle for writing
//   - error: Permission errors, invalid path, disk full, etc.
//
// Example (overwrite mode):
//
//	file, err := opener.OpenWrite("out.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
//
// Example (append mode):
//
//	file, err := opener.OpenWrite("out.txt", os.O_CREATE|os. O_WRONLY|os. O_APPEND, 0644)
func (fp *DefaultFileOpener) OpenWrite(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	return os.OpenFile(name, flag, perm)
}

// RedirectionSpec represents a parsed I/O redirection from a command line.
//
// A redirection spec is created during argument parsing when an operator
// like ">" or "2>>" is encountered, and contains all information needed
// to apply the redirection later.
//
// Example:
//
// From command: "ls -l > output.txt 2> errors.log"
// Creates two specs:
//   - {Operator: ">", Target:  "output.txt", Index: 2}
//   - {Operator: "2>", Target: "errors.log", Index: 4}
type RedirectionSpec struct {
	Operator string // Redirection operator (>, >>, 1>, 1>>, 2>, 2>>)
	Target   string // Target file path
	Index    int    // Position in original arguments (for error reporting)
}

// ParsedCommand represents a command line split into regular arguments
// and redirection specifications.
//
// This separation allows the shell to:
//  1. Execute the command with clean arguments (no redirection operators)
//  2. Apply redirections before execution
//  3. Validate redirections before opening files
//
// Example transformation:
//
// Input:   ["ls", "-l", ">", "output.txt", "src/"]
//
//	Result: ParsedCommand{
//	    Args:         []string{"ls", "-l", "src/"},
//	    Redirections: []RedirectionSpec{{Operator: ">", Target:  "output.txt", Index: 2}},
//	}
type ParsedCommand struct {
	Args         []string          // Command arguments without redirection operators
	Redirections []RedirectionSpec // Parsed redirection specifications
}

// RedirectionHandler defines the interface for implementing specific
// redirection types (stdout, stderr, stdin, etc.).
//
// Each handler is responsible for:
//  1. Recognizing its operator(s)
//  2. Validating redirection specifications
//  3. Opening files and modifying I/O bindings
//  4. Providing cleanup functions
//
// This design follows the Strategy pattern, allowing new redirection
// types to be added without modifying existing code.
//
// Example implementation for a custom handler:
//
//	type StdinHandler struct{}
//
//	func (h *StdinHandler) CanHandle(op string) bool {
//	    return op == "<"
//	}
//
//	func (h *StdinHandler) Validate(spec RedirectionSpec) error {
//	    return checkFileExists(spec.Target)
//	}
//
//	func (h *StdinHandler) Apply(spec RedirectionSpec, bindings *IOBindings, opener FileOpener) (func(), error) {
//	    file, err := opener.OpenRead(spec.Target)
//	    if err != nil {
//	        return nil, err
//	    }
//	    bindings.Stdin = file
//	    return func() { file.Close() }, nil
//	}
type RedirectionHandler interface {
	// CanHandle returns true if this handler can process the given operator.
	//
	// Multiple operators may map to the same handler.  For example,
	// ">" and "1>" both redirect stdout and can be handled by the same handler.
	//
	// Parameters:
	//   - operator: The redirection operator to check (e.g., ">", ">>", "2>")
	//
	// Returns:
	//   - bool: true if this handler can process the operator
	CanHandle(operator string) bool

	// Validate checks if a redirection specification is valid.
	//
	// This is called before any files are opened, allowing fast failure
	// for invalid redirections without side effects.
	//
	// Parameters:
	//   - spec: The redirection specification to validate
	//
	// Returns:
	//   - error: ErrMissingRedirectDestination or other validation errors,
	//            nil if valid
	Validate(spec RedirectionSpec) error

	// Apply executes the redirection by opening files and modifying I/O bindings.
	//
	// This method opens the target file with appropriate flags and updates
	// the bindings to point to the opened file.  It returns a cleanup function
	// that must be called to close the file when done.
	//
	// Parameters:
	//   - spec:  The redirection specification to apply
	//   - ioBindings: I/O bindings to modify (Stdin, Stdout, Stderr)
	//   - opener: File opener for creating file handles
	//
	// Returns:
	//   - cleanup: Function to close opened files (must be called by caller)
	//   - error: File opening errors, permission errors, etc.
	Apply(spec RedirectionSpec, ioBindings *IOBindings, opener FileOpener) (cleanup func(), err error)
}

// StdoutRedirectionHandler handles redirection of standard output (file descriptor 1).
//
// Supported operators:
//   - > or 1>   :  Overwrite mode (truncate existing file)
//   - >> or 1>> : Append mode (append to existing file)
//
// File permissions:
//   - Created files get 0644 permissions (rw-r--r--)
//
// The Overwrite field determines the behavior when the target file exists:
//   - true:   File is truncated (os.O_TRUNC)
//   - false: Data is appended (os.O_APPEND)
//
// Example usage:
//
//	handler := &StdoutRedirectionHandler{Overwrite: true}
//	handler. CanHandle(">")   // returns true
//	handler. CanHandle("1>")  // returns true
//	handler. CanHandle(">>")  // returns false
type StdoutRedirectionHandler struct {
	Overwrite bool // true for >/1>, false for >>/1>>
}

// CanHandle returns true if this handler can process the given stdout operator.
//
// Parameters:
//   - operator: The operator to check
//
// Returns:
//   - bool: true for > or 1> (if Overwrite=true), or >> or 1>> (if Overwrite=false)
func (handler *StdoutRedirectionHandler) CanHandle(operator string) bool {
	if handler.Overwrite {
		return operator == ">" || operator == "1>"
	}

	return operator == ">>" || operator == "1>>"
}

// Validate checks that the redirection has a target file.
//
// Parameters:
//   - spec: The redirection specification to validate
//
// Returns:
//   - error: ErrMissingRedirectDestination if Target is empty, nil otherwise
func (handler *StdoutRedirectionHandler) Validate(spec RedirectionSpec) error {
	if spec.Target == "" {
		return ErrMissingRedirectDestination
	}

	return nil
}

// Apply redirects stdout to the target file.
//
// The file is opened with:
//   - os.O_CREATE: Create if it doesn't exist
//   - os.O_WRONLY: Write-only mode
//   - os.O_TRUNC or os.O_APPEND:  Depending on Overwrite flag
//   - 0644 permissions:  rw-r--r--
//
// Parameters:
//   - spec:  Redirection specification with target file path
//   - ioBindings: I/O bindings to modify (Stdout will be replaced)
//   - opener: File opener for creating the file handle
//
// Returns:
//   - cleanup: Function to close the file (must be called by caller)
//   - error: File opening errors (permission denied, disk full, etc.)
//
// Example:
//
//	handler := &StdoutRedirectionHandler{Overwrite:  true}
//	spec := RedirectionSpec{Operator: ">", Target: "output.txt"}
//	cleanup, err := handler.Apply(spec, &bindings, opener)
//	if err != nil {
//	    return err
//	}
//	defer cleanup()
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

// StderrRedirectionHandler handles redirection of standard error (file descriptor 2).
//
// Supported operators:
//   - 2>  :  Overwrite mode (truncate existing file)
//   - 2>> :  Append mode (append to existing file)
//
// File permissions:
//   - Created files get 0644 permissions (rw-r--r--)
//
// The Overwrite field determines the behavior when the target file exists:
//   - true:  File is truncated (os.O_TRUNC)
//   - false: Data is appended (os. O_APPEND)
//
// Example usage:
//
//	handler := &StderrRedirectionHandler{Overwrite:  true}
//	handler. CanHandle("2>")   // returns true
//	handler. CanHandle("2>>")  // returns false
type StderrRedirectionHandler struct {
	Overwrite bool // true for 2>, false for 2>>
}

// CanHandle returns true if this handler can process the given stderr operator.
//
// Parameters:
//   - operator: The operator to check
//
// Returns:
//   - bool: true for 2> (if Overwrite=true), or 2>> (if Overwrite=false)
func (handler *StderrRedirectionHandler) CanHandle(operator string) bool {
	if handler.Overwrite {
		return operator == "2>"
	}

	return operator == "2>>"
}

// Validate checks that the redirection has a target file.
//
// Parameters:
//   - spec: The redirection specification to validate
//
// Returns:
//   - error: ErrMissingRedirectDestination if Target is empty, nil otherwise
func (handler *StderrRedirectionHandler) Validate(spec RedirectionSpec) error {
	if spec.Target == "" {
		return ErrMissingRedirectDestination
	}

	return nil
}

// Apply redirects stderr to the target file.
//
// The file is opened with:
//   - os.O_CREATE: Create if it doesn't exist
//   - os.O_WRONLY:  Write-only mode
//   - os.O_TRUNC or os.O_APPEND:  Depending on Overwrite flag
//   - 0644 permissions:  rw-r--r--
//
// Parameters:
//   - spec: Redirection specification with target file path
//   - ioBindings: I/O bindings to modify (Stderr will be replaced)
//   - opener: File opener for creating the file handle
//
// Returns:
//   - cleanup: Function to close the file (must be called by caller)
//   - error: File opening errors (permission denied, disk full, etc.)
//
// Example:
//
//	handler := &StderrRedirectionHandler{Overwrite: true}
//	spec := RedirectionSpec{Operator: "2>", Target:  "errors.log"}
//	cleanup, err := handler.Apply(spec, &bindings, opener)
//	if err != nil {
//	    return err
//	}
//	defer cleanup()
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

// RedirectionManager coordinates multiple redirection handlers and manages
// the I/O redirection lifecycle.
//
// The manager maintains:
//   - A registry of handlers for different operator types
//   - A file opener for abstracting file system operations
//   - A list of known operators for argument parsing
//
// Responsibilities:
//  1. Route operators to appropriate handlers
//  2. Validate all redirections before opening files (fail-fast)
//  3. Apply redirections in order
//  4. Clean up opened files if any redirection fails
//  5. Provide combined cleanup function for caller
//
// The manager follows the Registry and Strategy patterns, making it
// easy to add new redirection types without modifying existing code.
//
// Example:
//
//	manager := NewRedirectionManager(&DefaultFileOpener{})
//	specs := []RedirectionSpec{
//	    {Operator: ">", Target: "out.txt"},
//	    {Operator: "2>", Target: "err.log"},
//	}
//	bindings, cleanup, err := manager.ApplyRedirections(specs, baseBindings)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer cleanup()
type RedirectionManager struct {
	handlers   []RedirectionHandler // Registered handlers for different operators
	fileOpener FileOpener           // Abstraction for file system operations
	knownOps   []string             // List of recognized operators
}

// GetHandler finds the appropriate handler for a redirection operator.
//
// The method searches through registered handlers in order until it finds
// one that can handle the operator. If multiple handlers can handle the
// same operator, the first registered handler is used.
//
// Parameters:
//   - operator:  The redirection operator to find a handler for
//
// Returns:
//   - RedirectionHandler: Handler that can process this operator
//   - error: Error if no handler supports this operator
//
// Example:
//
//	handler, err := manager.GetHandler(">")
//	if err != nil {
//	    return fmt.Errorf("unsupported operator: %w", err)
//	}
func (rManager *RedirectionManager) GetHandler(operator string) (RedirectionHandler, error) {

	for _, handler := range rManager.handlers {
		if handler.CanHandle(operator) {
			return handler, nil
		}
	}

	return nil, fmt.Errorf("unsupported redirection operator: %s", operator)

}

// NewRedirectionManager creates a new redirection manager with default handlers.
//
// The manager is initialized with handlers for:
//   - > and 1>   : Stdout overwrite
//   - >> and 1>> : Stdout append
//   - 2>         : Stderr overwrite
//   - 2>>        : Stderr append
//
// Parameters:
//   - fileOpener: Implementation of FileOpener for file operations.
//     Use &DefaultFileOpener{} for production,
//     or a mock implementation for testing.
//
// Returns:
//   - *RedirectionManager: Fully initialized manager ready to use
//
// Example (production):
//
//	manager := NewRedirectionManager(&DefaultFileOpener{})
//
// Example (testing):
//
//	mockOpener := &MockFileOpener{files: make(map[string]*bytes.Buffer)}
//	manager := NewRedirectionManager(mockOpener)
func NewRedirectionManager(fileOpener FileOpener) *RedirectionManager {

	rManager := &RedirectionManager{
		handlers:   []RedirectionHandler{},
		fileOpener: fileOpener,
		knownOps:   []string{},
	}

	// > , 1>
	rManager.RegisterHandler(&StdoutRedirectionHandler{Overwrite: true})
	rManager.RegisterKnownOperator(">")
	rManager.RegisterKnownOperator("1>")

	// >> , 1>>
	rManager.RegisterHandler(&StdoutRedirectionHandler{Overwrite: false})
	rManager.RegisterKnownOperator(">>")
	rManager.RegisterKnownOperator("1>>")

	// 2>
	rManager.RegisterHandler(&StderrRedirectionHandler{Overwrite: true})
	rManager.RegisterKnownOperator("2>")

	// 2>>
	rManager.RegisterHandler(&StderrRedirectionHandler{Overwrite: false})
	rManager.RegisterKnownOperator("2>>")

	return rManager

}

// RegisterKnownOperator adds an operator to the list of known operators.
//
// This is used by ArgumentParser to recognize redirection operators during
// command line parsing.  Operators must be registered here to be recognized.
//
// Parameters:
//   - operator: The operator string to register (e.g., ">", "2>>")
//
// Example:
//
//	manager.RegisterKnownOperator("<")   // for stdin redirection
//	manager.RegisterKnownOperator("2>&1") // for fd duplication
func (rManager *RedirectionManager) RegisterKnownOperator(operator string) {
	rManager.knownOps = append(rManager.knownOps, operator)
}

// RegisterHandler adds a new redirection handler to the manager.
//
// Handlers are checked in registration order. If you register a handler
// that can handle the same operators as an existing handler, the first
// registered handler will be used.
//
// Parameters:
//   - handler: The RedirectionHandler implementation to register
//
// Example:
//
//	stdinHandler := &StdinRedirectionHandler{}
//	manager.RegisterHandler(stdinHandler)
func (rManager *RedirectionManager) RegisterHandler(handler RedirectionHandler) {
	rManager.handlers = append(rManager.handlers, handler)
}

// ValidateSpecs validates all redirection specifications before execution.
//
// This method implements fail-fast validation by checking all redirections
// before opening any files. This prevents partial execution where some files
// are opened successfully before an error occurs.
//
// Validation checks:
//  1. Each operator has a registered handler
//  2. Each redirection passes its handler's validation
//  3. All required fields are present (e.g., target paths)
//
// Parameters:
//   - specs:  Slice of redirection specifications to validate
//
// Returns:
//   - error:  First validation error encountered, or nil if all valid
//
// Example:
//
//	specs := []RedirectionSpec{
//	    {Operator: ">", Target: "out.txt"},
//	    {Operator: "2>", Target: ""},  // Invalid:  empty target
//	}
//	err := manager.ValidateSpecs(specs)
//	// err:  "invalid redirection '2> ':  missing redirect destination"
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

// ApplyRedirections validates and applies all redirection specifications.
//
// This is the main entry point for the redirection system. It:
//  1. Validates all specs (fail-fast if any are invalid)
//  2. Creates a copy of base I/O bindings
//  3. Applies each redirection in order
//  4. Collects cleanup functions
//  5. Returns modified bindings and combined cleanup
//
// Error handling:
//   - If validation fails:  Returns base bindings unchanged, no cleanup needed
//   - If any Apply fails:  Closes already-opened files and returns base bindings
//
// Redirection order matters:
//   - Redirections are applied in the order they appear
//   - Later redirections can override earlier ones
//   - Example: "cmd > a.txt > b.txt" results in output to b.txt
//
// Parameters:
//   - specs: Redirection specifications to apply
//   - baseBindings: Original I/O bindings (Stdin, Stdout, Stderr)
//
// Returns:
//   - IOBindings: Modified bindings with redirections applied
//   - cleanup:  Function to close all opened files (MUST be called by caller)
//   - error:  Validation or file opening errors
//
// Example:
//
//	specs := []RedirectionSpec{
//	    {Operator: ">", Target: "output.txt"},
//	    {Operator: "2>", Target: "errors.log"},
//	}
//	baseBindings := IOBindings{
//	    Stdin:  os.Stdin,
//	    Stdout: os.Stdout,
//	    Stderr: os.Stderr,
//	}
//	bindings, cleanup, err := manager.ApplyRedirections(specs, baseBindings)
//	if err != nil {
//	    return err
//	}
//	defer cleanup()
//	// Use bindings. Stdout and bindings.Stderr (now point to files)
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

// ArgumentParser separates regular command arguments from redirection operators.
//
// The parser recognizes redirection operators and extracts them along with
// their target paths, leaving only the clean command arguments.  This allows
// commands to be executed without redirection syntax in their argument lists.
//
// Recognition:
//   - Operators are recognized based on the RedirectionManager's known operators
//   - Each operator must be followed by a target path
//   - Operators can appear anywhere in the argument list
//
// Example transformation:
//
// Input:  ["ls", "-l", ">", "output.txt", "src/", "2>", "errors.log"]
//
//	Output: ParsedCommand{
//	    Args:  []string{"ls", "-l", "src/"},
//	    Redirections: []RedirectionSpec{
//	        {Operator: ">", Target: "output.txt", Index: 2},
//	        {Operator: "2>", Target: "errors.log", Index: 5},
//	    },
//	}
type ArgumentParser struct {
	operators map[string]bool
}

// NewArgumentParser creates a parser initialized with known operators.
//
// The parser is configured with operators from the RedirectionManager,
// ensuring consistent recognition between parsing and execution.
//
// Parameters:
//   - rManager: RedirectionManager to get known operators from
//
// Returns:
//   - *ArgumentParser: Parser ready to process arguments
//
// Example:
//
//	manager := NewRedirectionManager(&DefaultFileOpener{})
//	parser := NewArgumentParser(manager)
//	parsed, err := parser.Parse([]string{"echo", "hello", ">", "out.txt"})
func NewArgumentParser(rManager *RedirectionManager) *ArgumentParser {

	argParser := &ArgumentParser{
		operators: make(map[string]bool),
	}

	for _, op := range rManager.knownOps {
		argParser.operators[op] = true
	}

	return argParser

}

// Parse separates arguments into commands and redirection specifications.
//
// The parser scans the argument list, extracting redirection operators and
// their targets while preserving the order of regular arguments.
//
// Processing:
//  1. Iterate through arguments
//  2. If argument is a known operator:
//     - Check that a target follows
//     - Create RedirectionSpec
//     - Skip both operator and target
//  3. If argument is not an operator:
//     - Add to regular arguments
//
// Error conditions:
//   - Operator at end of list without target
//
// Parameters:
//   - args: Command arguments including redirection operators
//
// Returns:
//   - ParsedCommand: Separated arguments and redirection specs
//   - error: Error if operator is missing target
//
// Examples:
//
//	Parse([]string{"echo", "hello", ">", "out.txt"})
//	  → ParsedCommand{Args: []string{"echo", "hello"}, Redirections: [... ]}
//
//	Parse([]string{"ls", "-l"})
//	  → ParsedCommand{Args: []string{"ls", "-l"}, Redirections: []}, nil
//
//	Parse([]string{"echo", "test", ">"})
//	  → ParsedCommand{}, error("missing target for redirection '>' at position 2")
func (argumentParser *ArgumentParser) Parse(args []string) (ParsedCommand, error) {

	parsedCommand := ParsedCommand{
		Args:         []string{},
		Redirections: []RedirectionSpec{},
	}

	i := 0

	for i < len(args) {

		arg := args[i]
		// is a known operator -> append
		if argumentParser.operators[arg] {

			// if there is no target then return error
			if i == len(args)-1 {
				return parsedCommand, fmt.Errorf("missing target for redirection '%s' at position %d", arg, i)
			}

			spec := RedirectionSpec{
				Operator: arg,
				Target:   args[i+1],
				Index:    i,
			}

			parsedCommand.Redirections = append(parsedCommand.Redirections, spec)
			i += 2
			continue

		}

		parsedCommand.Args = append(parsedCommand.Args, arg)
		i++

	}

	return parsedCommand, nil

}
