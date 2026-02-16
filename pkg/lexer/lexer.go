package lexer

// lexer implements a lexer for the arg grammar
//
// valid examples
// ./my_file.yaml
// jsonpath=.Secrets,src=./my_secrets.yaml
// jsonpath=.Secrets,src=https://example.com/my_secrets.yaml
// jsonpath=.Secrets,src=https://example.com/my_secrets.yaml,auth=username:password
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
	FieldSep
	EQ
	KEY
	KeyLiteral
	VALUE
	ValueLiteral
	ParenEnterCall
	ParenExitCall
	INVALID
)

func (t TokenType) String() string {
	switch t {
	case START:
		return "START"
	case EOF:
		return "EOF"
	case FieldSep:
		return "FIELD_SEP"
	case EQ:
		return "EQ"
	case KEY:
		return "KEY"
	case KeyLiteral:
		return "KEY_LITERAL"
	case VALUE:
		return "VALUE"
	case ValueLiteral:
		return "VALUE_LITERAL"
	case ParenEnterCall:
		return "PAREN_ENTER_CALL"
	case ParenExitCall:
		return "PAREN_EXIT_CALL"
	case INVALID:
		return "INVALID"
	default:
		return fmt.Sprintf("TokenType(%d)", t)
	}
}

type Lexer struct {
	input string
	start int
	pos   int
	width int
	lexed chan Token
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

func (l *Lexer) nextEscaped() (r rune, escaped bool) {
	if l.pos >= len(l.input) {
		return -1, false
	}

	if l.input[l.pos] == '\\' {
		// Escape character found, get the next character
		l.pos++
		if l.pos < len(l.input) {
			r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
			l.pos += l.width
			return r, true
		}
		// Backslash at end of input, treat as literal backslash
		return '\\', false
	}

	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r, false
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
	l.lexed <- Token{Type: FieldSep, Literal: ",", Start: l.start, End: l.pos}
	l.skipWhitespace()
	return lexKey
}

func lexValue(l *Lexer) lexFunc {
	if l.pos < len(l.input) && l.input[l.pos] == '=' {
		l.pos++
	}
	l.skipWhitespace()

	var valueBuilder []rune
	valueStart := l.pos

	for l.pos < len(l.input) {
		r, escaped := l.nextEscaped()
		if r == -1 {
			break
		}

		if !escaped && r == ',' {
			// Unescaped comma ends the value
			// Need to rollback the position since nextEscaped already moved past the comma
			if escaped {
				l.pos -= l.width
			} else {
				// For unescaped comma, we need to find the actual byte width
				commaPos := l.pos - 1
				for commaPos > valueStart && l.input[commaPos] != ',' {
					commaPos--
				}
				if l.input[commaPos] == ',' {
					l.pos = commaPos
				}
			}
			break
		}

		if !escaped && r == '(' {
			// Unescaped parenthesis starts a function call
			if len(valueBuilder) > 0 {
				// Emit the value before the parenthesis
				literal := string(valueBuilder)
				l.lexed <- Token{Type: VALUE, Literal: literal, Start: valueStart, End: l.pos - l.width}
				l.start = l.pos - l.width
			}
			l.lexed <- Token{Type: ParenEnterCall, Literal: "(", Start: l.pos - l.width, End: l.pos}
			l.start = l.pos
			return lexInsideParens
		}

		// Add the character to the value (escaped or not)
		valueBuilder = append(valueBuilder, r)
	}

	if len(valueBuilder) > 0 {
		literal := string(valueBuilder)
		l.lexed <- Token{Type: VALUE, Literal: literal, Start: valueStart, End: l.pos}
	}

	l.lexed <- Token{Type: EOF, Literal: "", Start: l.pos, End: l.pos}
	return nil
}

func lexInsideParens(l *Lexer) lexFunc {
	l.skipWhitespace()
	inParenValue := false
	var argBuilder []rune
	argStart := l.pos

	for l.pos < len(l.input) {
		r, escaped := l.nextEscaped()
		if r == -1 {
			break
		}

		if !escaped && r == ')' {
			// End of function call
			if len(argBuilder) > 0 {
				tokenType := KEY
				if inParenValue {
					tokenType = VALUE
				}
				literal := string(argBuilder)
				l.lexed <- Token{Type: tokenType, Literal: literal, Start: argStart, End: l.pos - l.width}
			}
			l.lexed <- Token{Type: ParenExitCall, Literal: ")", Start: l.pos - l.width, End: l.pos}
			l.start = l.pos

			// Check what comes next
			rNext := l.peek(0)
			if rNext < 0 {
				l.lexed <- Token{Type: EOF, Literal: "", Start: l.pos, End: l.pos}
				return nil
			} else if rNext == ',' {
				return lexSep
			}
			l.lexed <- Token{Type: INVALID, Literal: string(rNext), Start: l.pos, End: l.pos}
			return nil
		}

		if !escaped && r == ',' {
			// Argument separator
			if len(argBuilder) > 0 {
				tokenType := KEY
				if inParenValue {
					tokenType = VALUE
				}
				literal := string(argBuilder)
				l.lexed <- Token{Type: tokenType, Literal: literal, Start: argStart, End: l.pos - l.width}
			}
			l.lexed <- Token{Type: FieldSep, Literal: ",", Start: l.pos - l.width, End: l.pos}
			l.skipWhitespace()
			inParenValue = false
			argBuilder = nil
			argStart = l.pos
			continue
		}

		if !escaped && r == '=' {
			// Key-value separator
			if len(argBuilder) > 0 {
				literal := string(argBuilder)
				l.lexed <- Token{Type: KEY, Literal: literal, Start: argStart, End: l.pos - l.width}
			}
			l.lexed <- Token{Type: EQ, Literal: "=", Start: l.pos - l.width, End: l.pos}
			l.skipWhitespace()
			inParenValue = true
			argBuilder = nil
			argStart = l.pos
			continue
		}

		// Add the character to the current argument
		argBuilder = append(argBuilder, r)
	}

	l.lexed <- Token{Type: EOF, Literal: "", Start: l.pos, End: l.pos}
	return nil
}

func lexLiteralValue(l *Lexer) lexFunc {
	if l.pos < len(l.input) && l.input[l.pos] == '=' {
		l.pos++
	}
	l.skipWhitespace()

	var valueBuilder []rune
	valueStart := l.pos

	for l.pos < len(l.input) {
		r, escaped := l.nextEscaped()
		if r == -1 {
			break
		}

		if !escaped && r == ',' {
			// Unescaped comma ends the value
			// Need to rollback the position since nextEscaped already moved past the comma
			if escaped {
				l.pos -= l.width
			} else {
				// For unescaped comma, we need to find the actual byte width
				commaPos := l.pos - 1
				for commaPos > valueStart && l.input[commaPos] != ',' {
					commaPos--
				}
				if l.input[commaPos] == ',' {
					l.pos = commaPos
				}
			}
			break
		}

		// Add the character to the value (escaped or not)
		valueBuilder = append(valueBuilder, r)
	}

	if len(valueBuilder) > 0 {
		literal := string(valueBuilder)
		l.lexed <- Token{Type: VALUE, Literal: literal, Start: valueStart, End: l.pos}
	}

	// Check if we're at EOF or need to continue with separator
	if l.pos >= len(l.input) {
		l.lexed <- Token{Type: EOF, Literal: "", Start: l.pos, End: l.pos}
		return nil
	}

	return lexSep
}

func lexKey(l *Lexer) lexFunc {
	l.start = l.pos

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
			if keyLiteral == "src" || keyLiteral == "jsonpath" {
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
