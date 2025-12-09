package shell

import (
	"errors"
	"io"
	"strings"
	"unicode"
)

var (
	ErrUnclosedQuote      = errors.New("unclosed quote")
	ErrUnescapedCharacter = errors.New("unescaped character")
)

type Parser interface {
	Parse(line string) ([]string, error)
}

type DefaultParser struct {
	newReader  func(string) io.RuneReader
	newBuilder func() *strings.Builder
}

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

type parseSate int

const (
	stateOutside parseSate = iota
	stateSingleQuote
	stateDoubleQuote
)

type tokenBuffer struct {
	b *strings.Builder
}

func newTokenBuffer(b *strings.Builder) *tokenBuffer {
	tb := &tokenBuffer{
		b: b,
	}

	return tb
}

func (tb *tokenBuffer) isEmpty() bool {
	return tb.b.Len() == 0
}

func (tb *tokenBuffer) appendRune(r rune) {
	tb.b.WriteRune(r)
}

func (tb *tokenBuffer) flushIfNotEmpty(args []string) []string {
	if !tb.isEmpty() {
		s := tb.b.String()
		tb.b.Reset()
		args = append(args, s)
	}

	return args

}

func (p *DefaultParser) Parse(line string) ([]string, error) {
	r := p.newReader(line)
	tb := newTokenBuffer(p.newBuilder())

	args := []string{}

	currState := stateOutside
	escaping := false

	for {
		ch, _, err := r.ReadRune()

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		switch currState {
		case stateOutside:

			if escaping {
				tb.appendRune(ch)
				escaping = false
				continue
			}

			if unicode.IsSpace(ch) {

				args = tb.flushIfNotEmpty(args)
				currState = stateOutside

			} else if ch == '\'' {
				currState = stateSingleQuote

			} else if ch == '"' {
				currState = stateDoubleQuote
			} else if ch == '\\' {
				escaping = true
			} else {
				tb.appendRune(ch)
			}

		case stateSingleQuote:
			if ch == '\'' {
				currState = stateOutside

			} else {
				tb.appendRune(ch)
			}

		case stateDoubleQuote:

			if escaping && (ch == '"' || ch == '\\') {
				tb.appendRune(ch)
				escaping = false
				continue
			}

			if ch == '"' {
				currState = stateOutside

			} else if ch == '\\' {
				escaping = true
			} else {
				tb.appendRune(ch)
			}

		}

	}

	if currState == stateSingleQuote {
		return nil, ErrUnclosedQuote
	}

	if escaping {
		return nil, ErrUnescapedCharacter
	}

	args = tb.flushIfNotEmpty(args)

	return args, nil

}
