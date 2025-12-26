package shell

import (
	"errors"
	"io"
	"strings"
	"unicode"
)

// Parser defines the interface for command line parsing implementations.
//
// Parsers are responsible for tokenizing a command line string into a slice
// of arguments, handling shell quoting rules, escape sequences, and whitespace.
//
// The parser must handle:
//   - Whitespace separation between tokens
//   - Single-quoted strings (preserving everything literally)
//   - Double-quoted strings (allowing escape sequences)
//   - Backslash escaping outside of quotes
//   - Empty quoted strings
//
// Example transformations:
//
//	echo hello world          → []string{"echo", "hello", "world"}
//	echo "hello world"        → []string{"echo", "hello world"}
//	echo 'hello world'        → []string{"echo", "hello world"}
//	echo hello\ world         → []string{"echo", "hello world"}
//	echo "hello \"world\""    → []string{"echo", "hello \"world\""}
type Parser interface {
	// Parse tokenizes a command line into arguments.
	//
	// Parameters:
	//   - line: The raw command line string to parse
	//
	// Returns:
	//   - []string:  Slice of parsed arguments/tokens
	//   - error: ErrUnclosedQuote if quotes aren't balanced,
	//            ErrUnescapedCharacter if line ends with backslash,
	//            or other errors for I/O failures
	Parse(line string) ([]string, error)
}

// ErrUnclosedQuote is returned when a command line contains an opening quote
// (single or double) without a corresponding closing quote.
//
// Example inputs that trigger this error:
//
//	echo "hello world
//	echo 'unterminated
//	echo "mixed 'quotes
var ErrUnclosedQuote = errors.New("unclosed quote")

// ErrUnescapedCharacter is returned when a command line ends with a backslash
// that doesn't escape any character.
//
// This prevents ambiguous parsing where a trailing backslash could be
// interpreted as either a literal backslash or an incomplete escape sequence.
//
// Example input that triggers this error:
//
//	echo hello\
var ErrUnescapedCharacter = errors.New("unescaped character")

// DefaultParser implements the Parser interface with shell-compatible
// quoting and escaping rules.
//
// The parser uses a state machine to track whether it's currently inside
// quotes and handles escape sequences according to standard shell rules:
//
// Single quotes:
//   - Everything is literal (no escaping possible)
//   - Cannot include a single quote inside single quotes
//   - Example: 'hello\nworld' → "hello\nworld" (backslash literal)
//
// Double quotes:
//   - Backslash escapes work for \" and \\
//   - Other backslashes are preserved literally
//   - Example: "hello\"world" → "hello"world"
//   - Example: "hello\nworld" → "hello\nworld" (backslash literal)
//
// Outside quotes:
//   - Backslash escapes the next character
//   - Whitespace separates tokens
//   - Example:  hello\ world → "hello world"
//
// The parser is designed for testability with injectable dependencies
// (newReader and newBuilder functions).
type DefaultParser struct {
	newReader  func(string) io.RuneReader
	newBuilder func() *strings.Builder
}

// NewDefaultParser creates a new DefaultParser with standard dependencies.
//
// The returned parser uses strings.NewReader for reading runes and
// strings.Builder for accumulating tokens.  These dependencies are
// injectable to facilitate testing.
//
// Returns:
//   - *DefaultParser: A ready-to-use parser instance
//
// Example:
//
//	parser := shell.NewDefaultParser()
//	args, err := parser.Parse(`echo "hello world"`)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(args) // Output: [echo hello world]
func NewDefaultParser() *DefaultParser {
	d := &DefaultParser{
		newReader: func(s string) io.RuneReader {
			return strings.NewReader(s)
		},
		newBuilder: func() *strings.Builder {
			return &strings.Builder{}
		},
	}

	return d
}

// parseState represents the current parsing context in the state machine.
//
// The parser transitions between states as it encounters quotes:
//   - stateOutside: Not inside any quotes (normal parsing)
//   - stateSingleQuote: Inside single quotes (literal mode)
//   - stateDoubleQuote: Inside double quotes (limited escaping)
type parseState int

const (
	stateOutside parseState = iota
	stateSingleQuote
	stateDoubleQuote
)

// tokenBuffer accumulates characters for the current token being parsed.
//
// This wrapper around strings.Builder provides token-specific operations
// like checking emptiness and flushing complete tokens to the result slice.
type tokenBuffer struct {
	builder *strings.Builder
}

// newTokenBuffer creates a new token accumulator.
//
// Parameters:
//   - builder: The strings.Builder to use for accumulation
//
// Returns:
//   - *tokenBuffer: A new token buffer instance
func newTokenBuffer(builder *strings.Builder) *tokenBuffer {
	tokenBuffer := &tokenBuffer{
		builder: builder,
	}

	return tokenBuffer
}

// isEmpty returns true if the buffer contains no characters.
//
// This is used to determine whether a token should be flushed.
// Empty quoted strings ("" or ”) do not trigger a flush.
func (tokenBuffer *tokenBuffer) isEmpty() bool {
	return tokenBuffer.builder.Len() == 0
}

// appendRune adds a single rune to the current token.
//
// Parameters:
//   - r: The rune to append
func (tokenBuffer *tokenBuffer) appendRune(r rune) {
	tokenBuffer.builder.WriteRune(r)
}

// flushIfNotEmpty finalizes the current token and adds it to the arguments slice.
//
// If the buffer is empty, no token is added.  After flushing, the buffer
// is reset for the next token.
//
// Parameters:
//   - args: The current slice of parsed arguments
//
// Returns:
//   - []string: Updated arguments slice with the flushed token (if any)
func (tokenBuffer *tokenBuffer) flushIfNotEmpty(args []string) []string {
	if !tokenBuffer.isEmpty() {
		s := tokenBuffer.builder.String()
		tokenBuffer.builder.Reset()
		args = append(args, s)
	}

	return args

}

// handleStateOutside processes a character when the parser is outside quotes.
//
// In this state:
//   - Whitespace triggers token completion
//   - ' begins a single-quoted string
//   - " begins a double-quoted string
//   - \ begins an escape sequence
//   - Other characters are added to the current token
//
// Parameters:
//   - ch:  The current character being processed
//   - currState:  The current parsing state (should be stateOutside)
//   - tokenBuffer: Buffer for the current token
//   - isEscaping: Whether the previous character was an unprocessed backslash
//   - args: Current slice of completed arguments
//
// Returns:
//   - parseState: The new parsing state
//   - bool: Whether an escape is in progress
//   - []string: Updated arguments slice
func handleStateOutside(ch rune, currState parseState, tokenBuffer *tokenBuffer, isEscaping bool, args []string) (parseState, bool, []string) {

	if isEscaping {
		tokenBuffer.appendRune(ch)
		isEscaping = false
		return currState, isEscaping, args
	}

	if unicode.IsSpace(ch) {

		args = tokenBuffer.flushIfNotEmpty(args)

	} else if ch == '\'' {
		currState = stateSingleQuote

	} else if ch == '"' {
		currState = stateDoubleQuote
	} else if ch == '\\' {
		isEscaping = true
	} else {
		tokenBuffer.appendRune(ch)
	}

	return currState, isEscaping, args

}

// handleStateSingleQuote processes a character when inside single quotes.
//
// In single-quoted strings:
//   - Everything is literal (no escape sequences)
//   - Only a closing ' exits the quoted string
//   - Backslashes, double quotes, and whitespace are all literal
//
// Parameters:
//   - ch: The current character being processed
//   - currState: The current parsing state (should be stateSingleQuote)
//   - tokenBuffer: Buffer for the current token
//   - isEscaping:  Escape flag (ignored in single quotes)
//   - args: Current slice of completed arguments
//
// Returns:
//   - parseState: The new parsing state
//   - bool: Whether an escape is in progress (always false)
//   - []string: Updated arguments slice
//
// Example:
//
//	'hello\nworld' → "hello\nworld" (backslash literal, not newline)
func handleStateSingleQuote(ch rune, currState parseState, tokenBuffer *tokenBuffer, isEscaping bool, args []string) (parseState, bool, []string) {

	if ch == '\'' {
		currState = stateOutside

	} else {
		tokenBuffer.appendRune(ch)
	}

	return currState, isEscaping, args

}

// handleStateDoubleQuote processes a character when inside double quotes.
//
// In double-quoted strings:
//   - Backslash escapes \" and \\
//   - Other backslashes are preserved literally
//   - Closing " exits the quoted string
//   - Whitespace is included in the token
//
// Escape behavior:
//   - \" → "   (escaped quote)
//   - \\ → \   (escaped backslash)
//   - \x → \x  (backslash + x, for any other character x)
//
// Parameters:
//   - ch: The current character being processed
//   - currState: The current parsing state (should be stateDoubleQuote)
//   - tokenBuffer: Buffer for the current token
//   - isEscaping: Whether the previous character was an unprocessed backslash
//   - args: Current slice of completed arguments
//
// Returns:
//   - parseState: The new parsing state
//   - bool: Whether an escape is in progress
//   - []string: Updated arguments slice
//
// Examples:
//
//	"hello\"world" → hello"world
//	"hello\\world" → hello\world
//	"hello\nworld" → hello\nworld (backslash literal)
func handleStateDoubleQuote(ch rune, currState parseState, tokenBuffer *tokenBuffer, isEscaping bool, args []string) (parseState, bool, []string) {

	if isEscaping {
		if ch != '\\' && ch != '"' {
			tokenBuffer.appendRune('\\')
		}

		tokenBuffer.appendRune(ch)

		isEscaping = false
		return currState, isEscaping, args

	}

	if ch == '"' {
		currState = stateOutside

	} else if ch == '\\' {
		isEscaping = true
	} else {
		tokenBuffer.appendRune(ch)
	}

	return currState, isEscaping, args

}

// Parse tokenizes a command line string into arguments using shell quoting rules.
//
// The method implements a state machine that tracks whether it's inside quotes
// and handles escape sequences appropriately for each context.  It processes
// the input character-by-character (rune-by-rune for Unicode support).
//
// Parsing rules:
//
// Whitespace (outside quotes):
//   - Separates tokens
//   - Multiple consecutive spaces are treated as one separator
//   - Tabs, newlines, and other Unicode spaces are all separators
//
// Single quotes:
//   - Everything inside is literal (no escaping)
//   - Example: 'hello world' → "hello world"
//   - Example: 'hello\n' → "hello\n" (backslash literal)
//
// Double quotes:
//   - Allows \" and \\ escape sequences
//   - Other backslashes are literal
//   - Example: "hello world" → "hello world"
//   - Example: "say \"hi\"" → "say "hi""
//   - Example: "path\\to\\file" → "path\to\file"
//
// Backslash (outside quotes):
//   - Escapes the next character
//   - Example: hello\ world → "hello world"
//   - Example: \$ → "$"
//
// Empty input:
//   - Returns empty slice (not an error)
//
// Parameters:
//   - line: The command line string to parse
//
// Returns:
//   - []string: Slice of parsed arguments/tokens
//   - error: ErrUnclosedQuote if quotes aren't balanced,
//     ErrUnescapedCharacter if line ends with backslash,
//     or I/O errors from the rune reader
//
// Examples:
//
//	Parse("echo hello")           → ["echo", "hello"], nil
//	Parse("echo 'hello world'")   → ["echo", "hello world"], nil
//	Parse(`echo "a b" c`)         → ["echo", "a b", "c"], nil
//	Parse("echo hello\\ world")   → ["echo", "hello world"], nil
//	Parse("echo 'unterminated)    → nil, ErrUnclosedQuote
//	Parse("echo trailing\\")      → nil, ErrUnescapedCharacter
//
// The parser maintains no state between calls - each invocation is independent.
func (p *DefaultParser) Parse(line string) ([]string, error) {
	runeReader := p.newReader(line)
	tokenBuffer := newTokenBuffer(p.newBuilder())

	args := []string{}

	currState := stateOutside
	isEscaping := false

	for {
		ch, _, err := runeReader.ReadRune()

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		switch currState {
		case stateOutside:
			currState, isEscaping, args = handleStateOutside(ch, currState, tokenBuffer, isEscaping, args)

		case stateSingleQuote:
			currState, isEscaping, args = handleStateSingleQuote(ch, currState, tokenBuffer, isEscaping, args)

		case stateDoubleQuote:
			currState, isEscaping, args = handleStateDoubleQuote(ch, currState, tokenBuffer, isEscaping, args)
		}

	}

	if currState == stateSingleQuote || currState == stateDoubleQuote {
		return nil, ErrUnclosedQuote
	}

	if isEscaping {
		return nil, ErrUnescapedCharacter
	}

	args = tokenBuffer.flushIfNotEmpty(args)

	return args, nil

}
