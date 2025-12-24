package lexer

// lexer implements a lexer for the arg grammar
//
// valid examples
// ./my_file.yaml
// path=.Secrets,src=./my_secrets.yaml
// path=.Secrets,src=https://example.com/my_secrets.yaml
// path=.Secrets,src=https://example.com/my_secrets.yaml,auth=username:password
// src=https://example.com/my_secrets.yaml
// src=https://example.com/my_secrets.tgz,decompress
// src=./here.json,type=schema(defaults=false)
// src=./here.json,type=schema
//
//

import (
	"fmt"
	"unicode/utf8"
)

type Token struct {
	Type    TokenType
	Literal string
	Start   int
	End     int
}

func (t *Token) String() string {
	return fmt.Sprintf("%s (%v)", t.Literal, t.Type)
}

type TokenType int

const (
	START TokenType = iota
	EOF
	FIELD_SEP
	KEY
	VALUE
	VALUE_OPEN_CALL
	VALUE_CLOSE_CALL
	INVALID
)

type Lexer struct {
	input string
	start int
	pos   int
	width int
	items chan Token
}

func NewLexer(input string) *Lexer {
	return &Lexer{
		input: input,
		start: 0,
		pos:   0,
		width: 0,
		items: make(chan Token),
	}
}

func (l *Lexer) next() (r rune) {
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

// peek i is a rune not byte
func (l *Lexer) peek(i int) rune {
	pos := l.pos
	count := 0
	r, width := utf8.DecodeRuneInString(l.input[pos:])
	pos += width
	for count < i {
		r, width = utf8.DecodeRuneInString(l.input[pos:])
		count++
		if count == i {
			return r
		}
		pos += width
	}
	return r
}

type lexFunc func(*Lexer) lexFunc

func lexStart(l *Lexer) lexFunc {
	l.items <- Token{Type: START, Literal: l.input[l.start:l.pos]}
	if l.pos+1 > len(l.input) {
		l.items <- Token{Type: EOF, Literal: l.input[l.start:l.pos]}
		return nil
	}
	return lexKey
}

func lexSep(l *Lexer) lexFunc {
	l.items <- Token{Type: FIELD_SEP, Literal: l.input[l.start:l.pos]}
	return lexKey
}

func lexValue(l *Lexer) lexFunc {
	l.items <- Token{Type: VALUE, Literal: l.input[l.start:l.pos]}
	return lexSep
}

func lexKey(l *Lexer) lexFunc {
	start := l.pos
	for r := l.next(); int(r) < l.width; {
		switch r {
		case ',':
			l.items <- Token{Type: KEY, Literal: l.input[start:l.pos]}
			return lexSep
		case '=':
			l.items <- Token{Type: KEY, Literal: l.input[start:l.pos]}
			return lexValue
		default:
			l.pos++
		}
	}
	l.items <- Token{Type: KEY, Literal: l.input[start:l.pos]}
	l.items <- Token{Type: EOF, Literal: ""}
	return nil
}

func (l *Lexer) run() {
	var state lexFunc
	for state = lexStart; state != nil; {
		state = state(l)
	}
	close(l.items)
}
