package shell

import (
	"errors"
	"io"
	"strings"
	"unicode"
)

type Parser interface {
	Parse(line string) ([]string, error)
}

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
	builder *strings.Builder
}

func newTokenBuffer(builder *strings.Builder) *tokenBuffer {
	tokenBuffer := &tokenBuffer{
		builder: builder,
	}

	return tokenBuffer
}

func (tokenBuffer *tokenBuffer) isEmpty() bool {
	return tokenBuffer.builder.Len() == 0
}

func (tokenBuffer *tokenBuffer) appendRune(r rune) {
	tokenBuffer.builder.WriteRune(r)
}

func (tokenBuffer *tokenBuffer) flushIfNotEmpty(args []string) []string {
	if !tokenBuffer.isEmpty() {
		s := tokenBuffer.builder.String()
		tokenBuffer.builder.Reset()
		args = append(args, s)
	}

	return args

}

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

func handleStateSingleQuote(ch rune, currState parseState, tokenBuffer *tokenBuffer, isEscaping bool, args []string) (parseState, bool, []string) {

	if ch == '\'' {
		currState = stateOutside

	} else {
		tokenBuffer.appendRune(ch)
	}

	return currState, isEscaping, args

}

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
