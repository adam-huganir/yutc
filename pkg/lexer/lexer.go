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
	"slices"
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
	EQ
	QUOTE_ENTER
	QUOTE_EXIT
	KEY
	KEY_LITERAL
	VALUE
	VALUE_LITERAL
	PAREN_ENTER_CALL
	PAREN_EXIT_CALL
	INVALID
)

func (t TokenType) String() string {
	switch t {
	case START:
		return "START"
	case EOF:
		return "EOF"
	case FIELD_SEP:
		return "FIELD_SEP"
	case EQ:
		return "EQ"
	case QUOTE_ENTER:
		return "QUOTE_ENTER"
	case QUOTE_EXIT:
		return "QUOTE_EXIT"
	case KEY:
		return "KEY"
	case KEY_LITERAL:
		return "KEY_LITERAL"
	case VALUE:
		return "VALUE"
	case VALUE_LITERAL:
		return "VALUE_LITERAL"
	case PAREN_ENTER_CALL:
		return "PAREN_ENTER_CALL"
	case PAREN_EXIT_CALL:
		return "PAREN_EXIT_CALL"
	case INVALID:
		return "INVALID"
	default:
		return fmt.Sprintf("TokenType(%d)", t)
	}
}

var (
	quotations = []rune{'\'', '"'}
)

type Lexer struct {
	input      string
	start      int
	pos        int
	width      int
	quoteLevel []rune
	parenLevel []rune
	lexed      chan Token
}

func NewLexer(input string) *Lexer {
	return &Lexer{
		input: input,
		start: 0,
		pos:   0,
		width: 0,
		lexed: make(chan Token),
	}
}

func (l *Lexer) next() (r rune) {
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		r, width := utf8.DecodeRuneInString(l.input[l.pos:])
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			break
		}
		l.pos += width
	}
	l.start = l.pos
}

func (l *Lexer) trimTrailingWhitespace(end int) int {
	for end > l.start {
		r, width := utf8.DecodeLastRuneInString(l.input[l.start:end])
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			break
		}
		end -= width
	}
	return end
}

// peek i is a rune not byte, 0 is the rune that will be returned by next()
func (l *Lexer) peek(i int) rune {
	pos := l.pos
	count := 0
	r, width := utf8.DecodeRuneInString(l.input[pos:])
	pos += width
	if pos >= len(l.input) {
		return -1
	}
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
	l.lexed <- Token{Type: START, Literal: l.input[l.start:l.pos]}
	l.skipWhitespace()
	if l.pos+1 > len(l.input) {
		l.lexed <- Token{Type: EOF, Literal: l.input[l.start:l.pos]}
		return nil
	}
	return lexKey
}

func lexSep(l *Lexer) lexFunc {
	l.start = l.pos
	if l.pos >= len(l.input) {
		l.lexed <- Token{Type: EOF, Literal: "", Start: l.pos, End: l.pos}
		return nil
	}
	r := l.next()
	if r != ',' {
		l.lexed <- Token{Type: INVALID, Literal: string(r), Start: l.start, End: l.pos}
		return nil
	}
	l.lexed <- Token{Type: FIELD_SEP, Literal: ",", Start: l.start, End: l.pos}
	l.skipWhitespace()
	return lexKey
}

func lexValue(l *Lexer) lexFunc {
	if l.pos < len(l.input) && l.input[l.pos] == '=' {
		l.pos++
	}
	l.skipWhitespace()
	for l.pos < len(l.input) {
		r := l.next()
		if slices.Contains(quotations, r) {
			l.quoteLevel = append(l.quoteLevel, r)
			l.lexed <- Token{Type: QUOTE_ENTER, Literal: string(r), Start: l.start, End: l.pos}
			l.start = l.pos

			if l.peek(1) == l.quoteLevel[len(l.quoteLevel)-1] {
				l.quoteLevel = l.quoteLevel[:len(l.quoteLevel)-1]
				r = l.next()
				l.lexed <- Token{Type: QUOTE_EXIT, Literal: string(r), Start: l.start, End: l.pos}
				l.start = l.pos
			}
		}
		switch r {
		case '(':
			if len(l.quoteLevel) == 0 {

				if len(l.parenLevel) > 0 {
					l.lexed <- Token{Type: INVALID, Literal: string(r), Start: l.start, End: l.pos}
				}
				if l.start < l.pos-l.width {
					l.lexed <- Token{Type: VALUE, Literal: l.input[l.start : l.pos-l.width], Start: l.start, End: l.pos - l.width}
				}
				l.lexed <- Token{Type: PAREN_ENTER_CALL, Literal: "(", Start: l.pos - l.width, End: l.pos}
				l.start = l.pos
				return lexInsideParens
			}
		case ',':
			if len(l.quoteLevel) == 0 {
				l.pos -= l.width
				if l.start < l.pos {
					end := l.trimTrailingWhitespace(l.pos)
					l.lexed <- Token{Type: VALUE, Literal: l.input[l.start:end], Start: l.start, End: end}
				}
				return lexSep
			} else {

			}
		}
	}
	if l.start < l.pos {
		end := l.trimTrailingWhitespace(l.pos)
		l.lexed <- Token{Type: VALUE, Literal: l.input[l.start:end], Start: l.start, End: end}
	}
	l.lexed <- Token{Type: EOF, Literal: "", Start: l.pos, End: l.pos}
	return nil
}

func lexInsideParens(l *Lexer) lexFunc {
	l.skipWhitespace()
	inParenValue := false
	for l.pos < len(l.input) {
		r := l.next()
		switch r {
		case ')':
			if l.start < l.pos-l.width {
				tokenType := KEY
				if inParenValue {
					tokenType = VALUE
				}
				end := l.trimTrailingWhitespace(l.pos - l.width)
				l.lexed <- Token{Type: tokenType, Literal: l.input[l.start:end], Start: l.start, End: end}
			}
			l.lexed <- Token{Type: PAREN_EXIT_CALL, Literal: ")", Start: l.pos - l.width, End: l.pos}
			l.start = l.pos
			rNext := l.peek(0)
			if rNext < 0 {
				l.lexed <- Token{Type: EOF, Literal: "", Start: l.pos, End: l.pos}
				return nil
			} else if rNext == ',' {
				return lexSep
			}
			l.lexed <- Token{Type: INVALID, Literal: string(rNext), Start: l.pos, End: l.pos}
			return nil
		case ',':
			if l.start < l.pos-l.width {
				tokenType := KEY
				if inParenValue {
					tokenType = VALUE
				}
				end := l.trimTrailingWhitespace(l.pos - l.width)
				l.lexed <- Token{Type: tokenType, Literal: l.input[l.start:end], Start: l.start, End: end}
			}
			l.lexed <- Token{Type: FIELD_SEP, Literal: ",", Start: l.pos - l.width, End: l.pos}
			l.skipWhitespace()
			inParenValue = false
		case '=':
			if l.start < l.pos-l.width {
				end := l.trimTrailingWhitespace(l.pos - l.width)
				l.lexed <- Token{Type: KEY, Literal: l.input[l.start:end], Start: l.start, End: end}
			}
			l.lexed <- Token{Type: EQ, Literal: "=", Start: l.pos - l.width, End: l.pos}
			l.skipWhitespace()
			inParenValue = true
		}
	}
	l.lexed <- Token{Type: EOF, Literal: "", Start: l.pos, End: l.pos}
	return nil
}

func lexLiteralValue(l *Lexer) lexFunc {
	if l.pos < len(l.input) && l.input[l.pos] == '=' {
		l.pos++
	}
	l.skipWhitespace()

	// Check for quoted value
	if l.pos < len(l.input) && slices.Contains(quotations, rune(l.input[l.pos])) {
		return lexQuotedValue
	}

	for l.pos < len(l.input) {
		r := l.next()
		if r == ',' {
			l.pos -= l.width
			if l.start < l.pos {
				end := l.trimTrailingWhitespace(l.pos)
				l.lexed <- Token{Type: VALUE, Literal: l.input[l.start:end], Start: l.start, End: end}
			}
			return lexSep
		}
	}
	if l.start < l.pos {
		end := l.trimTrailingWhitespace(l.pos)
		l.lexed <- Token{Type: VALUE, Literal: l.input[l.start:end], Start: l.start, End: end}
	}
	l.lexed <- Token{Type: EOF, Literal: "", Start: l.pos, End: l.pos}
	return nil
}

func lexQuotedValue(l *Lexer) lexFunc {
	quoteChar := rune(l.input[l.pos])
	l.lexed <- Token{Type: QUOTE_ENTER, Literal: string(quoteChar), Start: l.pos, End: l.pos + 1}
	l.pos++
	l.start = l.pos

	// Scan until we find the closing quote
	for l.pos < len(l.input) {
		r := l.next()
		if r == quoteChar {
			// Found closing quote
			if l.start < l.pos-l.width {
				l.lexed <- Token{Type: VALUE_LITERAL, Literal: l.input[l.start : l.pos-l.width], Start: l.start, End: l.pos - l.width}
			}
			l.lexed <- Token{Type: QUOTE_EXIT, Literal: string(quoteChar), Start: l.pos - l.width, End: l.pos}
			l.start = l.pos

			// Check what comes next
			if l.pos >= len(l.input) {
				l.lexed <- Token{Type: EOF, Literal: "", Start: l.pos, End: l.pos}
				return nil
			}

			nextChar := l.peek(0)
			if nextChar == ',' {
				return lexSep
			}

			// If nothing follows or EOF, we're done
			l.lexed <- Token{Type: EOF, Literal: "", Start: l.pos, End: l.pos}
			return nil
		}
	}

	// Unclosed quote - emit what we have as literal
	if l.start < l.pos {
		l.lexed <- Token{Type: VALUE_LITERAL, Literal: l.input[l.start:l.pos], Start: l.start, End: l.pos}
	}
	l.lexed <- Token{Type: EOF, Literal: "", Start: l.pos, End: l.pos}
	return nil
}

func lexKey(l *Lexer) lexFunc {
	l.start = l.pos

	// Check for quoted key/path
	if l.pos < len(l.input) && slices.Contains(quotations, rune(l.input[l.pos])) {
		return lexQuotedValue
	}

	for l.pos < len(l.input) {
		r := l.next()
		switch r {
		case ',':
			l.pos -= l.width
			end := l.trimTrailingWhitespace(l.pos)
			l.lexed <- Token{Type: KEY, Literal: l.input[l.start:end], Start: l.start, End: end}
			return lexSep
		case '=':
			l.pos -= l.width
			end := l.trimTrailingWhitespace(l.pos)
			keyLiteral := l.input[l.start:end]
			l.lexed <- Token{Type: KEY, Literal: keyLiteral, Start: l.start, End: end}
			l.lexed <- Token{Type: EQ, Literal: "=", Start: l.pos - l.width, End: l.pos}
			if keyLiteral == "src" || keyLiteral == "path" {
				return lexLiteralValue
			}
			return lexValue
		}
	}
	end := l.trimTrailingWhitespace(l.pos)
	l.lexed <- Token{Type: KEY, Literal: l.input[l.start:end], Start: l.start, End: end}
	l.lexed <- Token{Type: EOF, Literal: "", Start: l.pos, End: l.pos}
	return nil
}

func (l *Lexer) run() {
	var state lexFunc
	for state = lexStart; state != nil; {
		state = state(l)
	}
	close(l.lexed)
}

func (l *Lexer) Run() {
	l.run()
}

func (l *Lexer) Lexed() <-chan Token {
	return l.lexed
}

/*
KEY = QUOTE KEYSTRING QUOTE
VALUE = QUOTE VALUESTRING QUOTE

KEYSTRING= '[^=]'
VALUESTRING = '[^,]'
Q= '["'']'
*/
