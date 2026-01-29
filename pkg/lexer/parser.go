package lexer

import (
	"errors"
	"strconv"
)

type Arg struct {
	Source   *Field
	JSONPath *Field
	Type     *Field
	Auth     *Field
}

func (a *Arg) Map() map[string]*Field {
	return map[string]*Field{
		"src":      a.Source,
		"jsonpath": a.JSONPath,
		"type":     a.Type,
		"auth":     a.Auth,
	}
}

type Field struct {
	Value string
	Args  map[string]string
}

func NewField(value string, valueArgs map[string]string) *Field {
	return &Field{Value: value, Args: valueArgs}
}

type KeyValidator func(key string) error

type ValueValidator func(key string, value string) error
type FunctionValidator func(key string, functionName string, args map[string]string) error

type ValidationConfig struct {
	ValidateKey      KeyValidator
	ValidateValue    ValueValidator
	ValidateFunction FunctionValidator
}

type Parser struct {
	lexer      *Lexer
	tokens     []Token
	pos        int
	validation *ValidationConfig
}

func DefaultKeyValidator(key string) error {
	allowedKeys := map[string]bool{
		"src":      true,
		"jsonpath": true,
		"auth":     true,
		"type":     true,
	}
	if !allowedKeys[key] {
		return &ValidationError{
			Message: "invalid key '" + key + "': allowed keys are src, jsonpath, auth, type",
			Key:     key,
		}
	}
	return nil
}

func DefaultFunctionValidator(key, functionName string, _ map[string]string) error {
	if key == "type" && functionName == "schema" {
		return nil
	}
	return &ValidationError{
		Message: "function '" + functionName + "' not allowed on key '" + key + "': only schema() is allowed on type",
		Key:     key,
		Value:   functionName,
	}
}

var DefaultValidation = &ValidationConfig{
	ValidateKey:      DefaultKeyValidator,
	ValidateFunction: DefaultFunctionValidator,
}

func NewParser(input string) *Parser {
	return NewParserWithValidation(input, DefaultValidation)
}

func NewParserWithValidation(input string, validation *ValidationConfig) *Parser {
	l := NewLexer(input)
	return &Parser{
		lexer:      l,
		tokens:     []Token{},
		pos:        0,
		validation: validation,
	}
}

func (p *Parser) Parse() (*Arg, error) {
	go p.lexer.run()
	for token := range p.lexer.lexed {
		p.tokens = append(p.tokens, token)
	}
	return p.parseArg()
}

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: EOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) peek() Token {
	if p.pos+1 >= len(p.tokens) {
		return Token{Type: EOF}
	}
	return p.tokens[p.pos+1]
}

func (p *Parser) advance() {
	p.pos++
}

func (p *Parser) expect(tokenType TokenType) (Token, error) {
	token := p.current()
	if token.Type != tokenType {
		return token, &ParseError{
			Expected: tokenType,
			Got:      token.Type,
			Position: token.Start,
		}
	}
	p.advance()
	return token, nil
}

func (p *Parser) parseArg() (*Arg, error) {
	if _, err := p.expect(START); err != nil {
		return nil, err
	}

	arg := &Arg{}

	firstToken := p.current()
	if firstToken.Type == EOF {
		return arg, nil
	}

	if firstToken.Type == KEY {
		nextToken := p.peek()
		if nextToken.Type == EOF {
			// Non-keyed value, treat as src
			arg.Source = &Field{
				Value: firstToken.Literal,
				Args:  make(map[string]string),
			}
			p.advance()
			return arg, nil
		}

		if nextToken.Type == FieldSep {
			// Non-keyed value followed by more fields, treat as src
			arg.Source = &Field{
				Value: firstToken.Literal,
				Args:  make(map[string]string),
			}
			p.advance()
			p.advance()
		}
	}

	for p.current().Type != EOF {
		if err := p.parseField(arg); err != nil {
			return nil, err
		}

		if p.current().Type == FieldSep {
			p.advance()
		}
	}

	return arg, nil
}

func (p *Parser) parseField(arg *Arg) error {
	keyToken, err := p.expect(KEY)
	if err != nil {
		return err
	}

	if p.validation != nil && p.validation.ValidateKey != nil {
		if err := p.validation.ValidateKey(keyToken.Literal); err != nil {
			var valErr *ValidationError
			if errors.As(err, &valErr) {
				valErr.Position = keyToken.Start
			}
			return err
		}
	}

	if _, err := p.expect(EQ); err != nil {
		return err
	}

	field := Field{
		Args: make(map[string]string),
	}

	if p.current().Type != VALUE {
		return &ParseError{
			Expected: VALUE,
			Got:      p.current().Type,
			Position: p.current().Start,
		}
	}

	valueToken := p.current()
	p.advance()

	if p.current().Type == ParenEnterCall {
		p.advance()
		if err := p.parseArgs(&field); err != nil {
			return err
		}
		if _, err := p.expect(ParenExitCall); err != nil {
			return err
		}

		if p.validation != nil && p.validation.ValidateFunction != nil {
			if err := p.validation.ValidateFunction(keyToken.Literal, valueToken.Literal, field.Args); err != nil {
				var valErr *ValidationError
				if errors.As(err, &valErr) {
					valErr.Position = valueToken.Start
				}
				return err
			}
		}
	}

	field.Value = valueToken.Literal

	// Set the appropriate field on Arg based on key name
	switch keyToken.Literal {
	case "src":
		arg.Source = &field
	case "jsonpath":
		arg.JSONPath = &field
	case "type":
		arg.Type = &field
	case "auth":
		arg.Auth = &field
	default:
		// Unknown key - only error if validation is enabled
		if p.validation != nil {
			return &ValidationError{
				Message:  "unknown key: " + keyToken.Literal,
				Key:      keyToken.Literal,
				Position: keyToken.Start,
			}
		}
		// If no validation, silently ignore unknown keys
	}

	return nil
}

func (p *Parser) parseArgs(field *Field) error {
	for p.current().Type != ParenExitCall && p.current().Type != EOF {
		keyToken, err := p.expect(KEY)
		if err != nil {
			return err
		}

		// Check if there's an EQ token (optional for function i)
		if p.current().Type == EQ {
			p.advance()
			if p.current().Type == VALUE {
				valueToken := p.current()
				p.advance()
				field.Args[keyToken.Literal] = valueToken.Literal
			} else {
				field.Args[keyToken.Literal] = ""
			}
		} else {
			// No EQ means key without value
			field.Args[keyToken.Literal] = ""
		}

		if p.current().Type == FieldSep {
			p.advance()
		}
	}

	return nil
}

type ParseError struct {
	Expected TokenType
	Got      TokenType
	Position int
}

func (e *ParseError) Error() string {
	return "parse error at position " + strconv.Itoa(e.Position) + ": expected " + e.Expected.String() + ", got " + e.Got.String()
}

type ValidationError struct {
	Message  string
	Key      string
	Value    string
	Position int
}

func (e *ValidationError) Error() string {
	return e.Message
}
