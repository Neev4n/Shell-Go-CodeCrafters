package shell

import (
	"errors"
	"testing"
)

func TestParser_Parse(t *testing.T) {

	// Table-driven test:  each test case has a name, input, expected output, and expected error.
	tests := []struct {
		name        string
		input       string
		expected    []string
		expectedErr error
	}{
		{
			name:        "simple command",
			input:       "echo hello",
			expected:    []string{"echo", "hello"},
			expectedErr: nil,
		},
		{
			name:        "command with multiple arguments",
			input:       "ls -la /home/user",
			expected:    []string{"ls", "-la", "/home/user"},
			expectedErr: nil,
		},
		{
			name:        "single quoted string",
			input:       "echo 'hello world'",
			expected:    []string{"echo", "hello world"},
			expectedErr: nil,
		},
		{
			name:        "double quoted string",
			input:       `echo "hello world"`,
			expected:    []string{"echo", "hello world"},
			expectedErr: nil,
		},
		{
			name:        "mixed quotes",
			input:       `echo "hello" 'world'`,
			expected:    []string{"echo", "hello", "world"},
			expectedErr: nil,
		},
		{
			name:        "escaped characters outside quotes",
			input:       `echo hello\ world`,
			expected:    []string{"echo", "hello world"},
			expectedErr: nil,
		},
		{
			name:        "escaped quote in double quotes",
			input:       `echo "hello \"world\""`,
			expected:    []string{"echo", `hello "world"`},
			expectedErr: nil,
		},
		{
			name:        "escaped backslash in double quotes",
			input:       `echo "hello\\world"`,
			expected:    []string{"echo", `hello\world`},
			expectedErr: nil,
		},
		{
			name:        "single quotes preserve everything literally",
			input:       `echo 'hello\nworld'`,
			expected:    []string{"echo", `hello\nworld`},
			expectedErr: nil,
		},
		{
			name:        "empty input",
			input:       "",
			expected:    []string{},
			expectedErr: nil,
		},
		{
			name:        "only whitespace",
			input:       "   \t  \n  ",
			expected:    []string{},
			expectedErr: nil,
		},
		{
			name:        "multiple spaces between arguments",
			input:       "echo    hello     world",
			expected:    []string{"echo", "hello", "world"},
			expectedErr: nil,
		},
		{
			name:        "unclosed single quote",
			input:       "echo 'hello",
			expected:    nil,
			expectedErr: ErrUnclosedQuote,
		},
		{
			name:        "unclosed double quote",
			input:       `echo "hello`,
			expected:    nil,
			expectedErr: ErrUnclosedQuote,
		},
		{
			name:        "trailing backslash",
			input:       `echo hello\`,
			expected:    nil,
			expectedErr: ErrUnescapedCharacter,
		},
		{
			name:        "empty quotes",
			input:       `echo "" ''`,
			expected:    []string{"echo"},
			expectedErr: nil,
		},
		{
			name:        "adjacent quoted strings",
			input:       `echo "hello"'world'`,
			expected:    []string{"echo", "helloworld"},
			expectedErr: nil,
		},
		{
			name:        "command with special characters",
			input:       `grep "pattern" file.txt`,
			expected:    []string{"grep", "pattern", "file.txt"},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			parser := NewDefaultParser()
			res, err := parser.Parse(tt.input)

			// Check if the error matches expectations
			if tt.expectedErr != nil {
				if err == nil {
					t.Errorf("Expected error: %v got nil", tt.expectedErr)
					return
				} else if !errors.Is(err, tt.expectedErr) {
					t.Errorf("Expected error: %v got %v", tt.expectedErr, err)
					return
				}

				return

			}

			// Check if we got an unexpected error

			if err != nil {
				t.Errorf("Expected no error got %v", err)
				return
			}

			// Check if the output matches expectations
			if !equalStringSlices(res, tt.expected) {
				t.Errorf("input:  %q\nexpected: %v\ngot:       %v", tt.input, tt.expected, res)
			}

		})

	}

}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
