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

type parseState int

const (
	stateOutside parseState = iota
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

func handleStateOutside(ch rune, currState parseState, tb *tokenBuffer, escaping bool, args []string) (parseState, bool, []string) {

	if escaping {
		tb.appendRune(ch)
		escaping = false
		return currState, escaping, args
	}

	if unicode.IsSpace(ch) {

		args = tb.flushIfNotEmpty(args)

	} else if ch == '\'' {
		currState = stateSingleQuote

	} else if ch == '"' {
		currState = stateDoubleQuote
	} else if ch == '\\' {
		escaping = true
	} else {
		tb.appendRune(ch)
	}

	return currState, escaping, args

}

func handleStateSingleQuote(ch rune, currState parseState, tb *tokenBuffer, escaping bool, args []string) (parseState, bool, []string) {

	if ch == '\'' {
		currState = stateOutside

	} else {
		tb.appendRune(ch)
	}

	return currState, escaping, args

}

func handleStateDoubleQuote(ch rune, currState parseState, tb *tokenBuffer, escaping bool, args []string) (parseState, bool, []string) {

	if escaping {
		if ch != '\\' && ch != '"' {
			tb.appendRune('\\')
		}

		tb.appendRune(ch)

		escaping = false
		return currState, escaping, args

	}

	if ch == '"' {
		currState = stateOutside

	} else if ch == '\\' {
		escaping = true
	} else {
		tb.appendRune(ch)
	}

	return currState, escaping, args

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
			currState, escaping, args = handleStateOutside(ch, currState, tb, escaping, args)

		case stateSingleQuote:
			currState, escaping, args = handleStateSingleQuote(ch, currState, tb, escaping, args)

		case stateDoubleQuote:
			currState, escaping, args = handleStateDoubleQuote(ch, currState, tb, escaping, args)
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
