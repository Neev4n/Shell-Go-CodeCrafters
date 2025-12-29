# Shell-Go-CodeCrafters

[![Go Version](https://img.shields.io/badge/Go-1.19+-00ADD8? style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![CodeCrafters](https://img.shields.io/badge/CodeCrafters-Shell_Challenge-orange)](https://codecrafters.io/)

A feature-rich, modular command-line shell implementation in Go, built as part of the CodeCrafters Shell Challenge. This shell provides a REPL interface with support for built-in commands, external program execution, I/O redirection, and robust command parsing.

## ✨ Features

### 🔧 Built-in Commands

- **`echo`** - Print arguments to stdout
- **`exit`** - Terminate the shell gracefully
- **`type`** - Display command type information (builtin vs external)
- **`pwd`** - Print current working directory
- **`cd`** - Change directory with tilde (`~`) expansion support

### 🚀 External Command Execution

- PATH-based executable lookup
- Full argument passing with exit code handling
- Context-aware execution with timeout support

### 📁 I/O Redirection

| Operator | Description | Example |
|----------|-------------|---------|
| `>`, `1>` | Redirect stdout (overwrite) | `echo hello > file.txt` |
| `>>`, `1>>` | Redirect stdout (append) | `echo world >> file.txt` |
| `2>` | Redirect stderr (overwrite) | `cmd 2> errors.log` |
| `2>>` | Redirect stderr (append) | `cmd 2>> errors.log` |

### 🎯 Advanced Parsing

- **Single quotes** - Literal strings:  `'hello\nworld'` → `hello\nworld`
- **Double quotes** - Escape sequences: `"hello\"world"` → `hello"world`
- **Backslash escaping** - Outside quotes: `hello\ world` → `hello world`
- **Unicode support** - Full UTF-8 rune handling

## 📦 Installation

### Prerequisites

- Go 1.19 or higher
- Unix-like environment (Linux, macOS) or WSL on Windows

### Build from Source

```bash
# Clone the repository
git clone https://github.com/Neev4n/Shell-Go-CodeCrafters. git
cd Shell-Go-CodeCrafters

# Build the executable
go build -o shell ./app

# Run the shell
./shell
```

### Run Directly

```bash
go run ./app/main.go
```

### Install to PATH

```bash
# Build and install to $GOPATH/bin
go install ./app

# Or copy the binary to a directory in your PATH
sudo cp shell /usr/local/bin/
```

## 🎮 Usage

### Interactive Mode

```bash
$ ./shell
$ echo "Hello, World!"
Hello, World!
$ pwd
/home/user/projects/Shell-Go-CodeCrafters
$ cd ~/Documents
$ ls -la > files.txt
$ cat files.txt
total 24
drwxr-xr-x  4 user  group   128 Jan 15 10:30 . 
drwxr-xr-x  8 user  group   256 Jan 15 10:00 ..
-rw-r--r--  1 user  group  1024 Jan 15 10:30 files.txt
$ exit
```

### Script Execution

Create a script file `script.sh`:

```bash
echo "Running script..."
pwd
ls -l
echo "Done!"
exit
```

Execute it:

```bash
./shell < script.sh
```

### Command Examples

#### Basic Commands

```bash
# Echo with multiple arguments
$ echo hello world from shell
hello world from shell

# Print working directory
$ pwd
/home/user/projects

# Change directory (absolute path)
$ cd /tmp
$ pwd
/tmp

# Change to home directory
$ cd ~
$ pwd
/home/user

# Change to subdirectory of home
$ cd ~/Documents/projects
```

#### Quoting and Escaping

```bash
# Single quotes (literal)
$ echo 'hello\nworld'
hello\nworld

# Double quotes (escape sequences)
$ echo "hello\nworld"
hello\nworld

# Escape quote inside double quotes
$ echo "say \"hello\""
say "hello"

# Escape space outside quotes
$ echo hello\ world
hello world

# Mixed quoting
$ echo "hello" 'world' test
hello world test
```

#### I/O Redirection

```bash
# Redirect stdout to file (overwrite)
$ echo "First line" > output.txt
$ cat output.txt
First line

# Redirect stdout to file (append)
$ echo "Second line" >> output.txt
$ cat output. txt
First line
Second line

# Redirect stderr
$ ls /nonexistent 2> errors.log
$ cat errors.log
ls: /nonexistent: No such file or directory

# Multiple redirections
$ command arg1 > output. txt 2> errors.log
```

#### Command Type Checking

```bash
# Check built-in command
$ type echo
echo is a shell builtin

# Check external command
$ type ls
ls is /bin/ls

# Check non-existent command
$ type nonexistent
nonexistent:  not found
```

## 🏗️ Architecture

The shell follows a modular, extensible architecture with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────┐
│                         main.go                             │
│                  (Application Entry Point)                  │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                      Shell (REPL)                           │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  1. Display prompt                                  │   │
│  │  2. Read command line                               │   │
│  │  3. Parse & tokenize                                │   │
│  │  4. Separate arguments from redirections            │   │
│  │  5. Apply I/O redirections                          │   │
│  │  6. Execute builtin or external command             │   │
│  │  7. Cleanup resources                               │   │
│  │  8. Repeat                                          │   │
│  └─────────────────────────────────────────────────────┘   │
└────┬──────────┬──────────────┬──────────────┬──────────────┘
     │          │              │              │
     ▼          ▼              ▼              ▼
┌─────────┐ ┌────────┐ ┌──────────────┐ ┌────────────┐
│ Parser  │ │Executor│ │ Redirection  │ │  Builtins  │
│         │ │        │ │   Manager    │ │            │
└─────────┘ └────────┘ └──────────────┘ └────────────┘
```

### Core Components

#### 1. **Parser** (`parser.go`)
- Tokenizes command lines into arguments
- Handles single quotes, double quotes, and escape sequences
- State machine implementation for robust parsing
- Unicode/UTF-8 support

#### 2. **ArgumentParser** (`redirections.go`)
- Separates regular arguments from redirection operators
- Validates redirection syntax
- Preserves argument order

#### 3. **RedirectionManager** (`redirections.go`)
- Manages file opening for redirections
- Strategy pattern for different redirection types
- Automatic resource cleanup
- Supports custom file openers for testing

#### 4. **Executor** (`executor.go`)
- Executes external commands via `os/exec`
- PATH-based command lookup
- Context-aware execution
- Exit code handling

#### 5. **Shell** (`shell.go`)
- Coordinates all components
- REPL implementation
- Built-in command registry
- Error handling and recovery

### Design Patterns

- **Strategy Pattern**: RedirectionHandler interface for extensible I/O redirection
- **Registry Pattern**: RedirectionManager for operator-to-handler mapping
- **Dependency Injection**: FileOpener interface for testable file operations
- **State Machine**: Parser for quote and escape handling

## 🧪 Testing

### Run Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./pkg/shell

# Run specific test
go test -run TestParser_Parse ./pkg/shell
```

### Test with Custom I/O

```go
package main

import (
    "bytes"
    "strings"
    "testing"
    "github.com/Neev4n/CodeCrafters-Shell-GO/codecrafters-shell-go/pkg/shell"
)

func TestShellEcho(t *testing.T) {
    input := strings.NewReader("echo hello\nexit\n")
    var stdout, stderr bytes.Buffer
    
    sh := shell.New(input, &stdout, &stderr)
    err := sh.Run()
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    
    output := stdout.String()
    if ! strings.Contains(output, "hello") {
        t.Errorf("expected 'hello' in output, got: %s", output)
    }
}
```

### Mock File Opener Example

```go
type MockFileOpener struct {
    files map[string]*bytes.Buffer
}

func (m *MockFileOpener) OpenWrite(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
    buf := &bytes.Buffer{}
    m. files[name] = buf
    return &mockWriteCloser{buf}, nil
}

// Test redirection without touching filesystem
func TestRedirection(t *testing.T) {
    mock := &MockFileOpener{files: make(map[string]*bytes.Buffer)}
    manager := shell.NewRedirectionManager(mock)
    
    specs := []shell.RedirectionSpec{
        {Operator: ">", Target: "output.txt"},
    }
    
    bindings, cleanup, err := manager.ApplyRedirections(specs, baseBindings)
    defer cleanup()
    
    fmt.Fprintln(bindings. Stdout, "test output")
    
    content := mock.files["output.txt"].String()
    if content != "test output\n" {
        t. Errorf("unexpected content: %s", content)
    }
}
```

## 📚 API Documentation

### Generate Documentation

```bash
# View package documentation
go doc github.com/Neev4n/CodeCrafters-Shell-GO/codecrafters-shell-go/pkg/shell

# View specific function
go doc shell.New
go doc shell.Shell. Run

# Serve HTML documentation locally
godoc -http=:6060
# Visit: http://localhost:6060/pkg/github.com/Neev4n/CodeCrafters-Shell-GO/codecrafters-shell-go/pkg/shell/
```

### Quick Reference

#### Creating a Shell

```go
import "github.com/Neev4n/CodeCrafters-Shell-GO/codecrafters-shell-go/pkg/shell"

// Standard I/O
sh := shell.New(os.Stdin, os. Stdout, os.Stderr)

// Custom I/O (for testing)
input := strings.NewReader("echo test\nexit\n")
var stdout, stderr bytes.Buffer
sh := shell.New(input, &stdout, &stderr)
```

#### Running Commands

```go
// Start REPL
if err := sh.Run(); err != nil {
    log.Fatal(err)
}
```

#### Custom Built-ins (requires code modification)

```go
shell.builtins["hello"] = func(args []string, s *shell.Shell) error {
    fmt.Fprintln(s.Out, "Hello from custom builtin!")
    return nil
}
```

## 🔌 Extensibility

### Adding New Redirection Types

Implement the `RedirectionHandler` interface:

```go
type StdinHandler struct{}

func (h *StdinHandler) CanHandle(operator string) bool {
    return operator == "<"
}

func (h *StdinHandler) Validate(spec RedirectionSpec) error {
    if spec.Target == "" {
        return ErrMissingRedirectDestination
    }
    return nil
}

func (h *StdinHandler) Apply(spec RedirectionSpec, bindings *IOBindings, opener FileOpener) (func(), error) {
    file, err := opener.OpenRead(spec.Target)
    if err != nil {
        return nil, err
    }
    bindings.Stdin = file
    return func() { file.Close() }, nil
}

// Register the handler
manager.RegisterHandler(&StdinHandler{})
manager.RegisterKnownOperator("<")
```

### Adding New Built-in Commands

Modify `registerBuiltins()` in `shell.go`:

```go
shell.builtins["history"] = func(args []string, shell *Shell) error {
    // Implementation here
    fmt.Fprintln(shell. Out, "Command history:")
    // ... display history
    return nil
}
```

### Custom Executors

Implement the `Executor` interface for specialized execution:

```go
type SandboxExecutor struct {
    allowedCommands map[string]bool
}

func (e *SandboxExecutor) Execute(ctx context.Context, name string, args []string, io IOBindings) (int, error) {
    if !e.allowedCommands[name] {
        return -1, fmt.Errorf("command not allowed: %s", name)
    }
    // Execute with restrictions... 
}
```

## ⚠️ Known Limitations

The following features are not currently supported:

- ❌ **Pipes** (`|`) - Command chaining
- ❌ **Background jobs** (`&`) - Asynchronous execution
- ❌ **Command substitution** (`` `cmd` `` or `$(cmd)`)
- ❌ **Wildcards/globbing** (`*`, `?`, `[abc]`)
- ❌ **Environment variable expansion** (except `~` in `cd`)
- ❌ **Job control** (`fg`, `bg`, `jobs`)
- ❌ **Signal handling** (Ctrl+C, Ctrl+Z)
- ❌ **Command history** (up/down arrows)
- ❌ **Tab completion**
- ❌ **Aliases**
- ❌ **Shell scripting** (if/else, loops, functions)

## 🐛 Troubleshooting

### Command Not Found

```bash
$ mycommand
mycommand: command not found
```

**Solution**: Ensure the command is in your `PATH`:
```bash
$ echo $PATH
$ which mycommand
```

### Permission Denied

```bash
$ cd /root
cd: /root: Permission denied
```

**Solution**: Check directory permissions or use `sudo` if appropriate. 

### Unclosed Quote Error

```bash
$ echo "hello
[shell exits with parse error]
```

**Solution**: Ensure all quotes are properly closed: 
```bash
$ echo "hello"
```

### Redirection Error

```bash
$ echo test >
parse error: missing target for redirection '>' at position 2
```

**Solution**: Provide a target file:
```bash
$ echo test > output.txt
```

## 📖 Environment Variables

| Variable | Purpose | Example |
|----------|---------|---------|
| `PATH` | Directories to search for executables | `/usr/local/bin:/usr/bin:/bin` |
| `HOME` | User's home directory (for `~` expansion) | `/home/username` |

## 🚦 Exit Codes

| Code | Meaning | Trigger |
|------|---------|---------|
| `0` | Success | `exit` command or normal termination |
| `1` | Fatal error | I/O error, parse error |

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Development Setup

1. Fork the repository
2. Clone your fork: 
   ```bash
   git clone https://github.com/YOUR_USERNAME/Shell-Go-CodeCrafters. git
   ```
3. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```
4. Make your changes and add tests
5. Run tests:
   ```bash
   go test ./...
   ```
6. Commit and push:
   ```bash
   git add .
   git commit -m "Add your feature"
   git push origin feature/your-feature-name
   ```
7. Open a Pull Request

### Code Style

- Follow standard Go formatting (`gofmt`)
- Add GoDoc comments for all exported types and functions
- Write unit tests for new functionality
- Keep functions focused and modular

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built as part of the [CodeCrafters](https://codecrafters.io/) Shell Challenge
- Inspired by traditional Unix shells (bash, sh, zsh)
- Thanks to the Go community for excellent tooling and libraries

## 📞 Contact

- **Author**: Neev4n
- **Repository**: [github.com/Neev4n/Shell-Go-CodeCrafters](https://github.com/Neev4n/Shell-Go-CodeCrafters)
- **Issues**: [GitHub Issues](https://github.com/Neev4n/Shell-Go-CodeCrafters/issues)

## 🗺️ Roadmap

Future enhancements being considered: 

- [ ] Pipe support (`|`)
- [ ] Input redirection (`<`)
- [ ] Here-documents (`<<`)
- [ ] Command history with persistence
- [ ] Tab completion
- [ ] Signal handling (Ctrl+C)
- [ ] Scripting support (conditionals, loops)
- [ ] Environment variable expansion
- [ ] Alias support
- [ ] Configuration file (`.shellrc`)
- [ ] Plugin system for custom commands

## 📊 Project Stats

```bash
# Lines of code
$ find . -name '*.go' | xargs wc -l

# Test coverage
$ go test -cover ./... 
```

---

**Happy Shelling!  🐚**

Made with ❤️ using Go