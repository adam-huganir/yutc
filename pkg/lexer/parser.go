package lexer

import "errors"

type Arg struct {
	Path   string
	Fields map[string]Field
}

type Field struct {
	Value string
	Args  map[string]string
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
		"src":  true,
		"path": true,
		"auth": true,
		"type": true,
	}
	if !allowedKeys[key] {
		return &ValidationError{
			Message: "invalid key '" + key + "': allowed keys are src, path, auth, type",
			Key:     key,
		}
	}
	return nil
}

func DefaultFunctionValidator(key string, functionName string, args map[string]string) error {
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

	arg := &Arg{
		Fields: make(map[string]Field),
	}

	firstToken := p.current()
	if firstToken.Type == EOF {
		return arg, nil
	}

	// Handle quoted path
	if firstToken.Type == QUOTE_ENTER {
		p.advance()
		if p.current().Type == VALUE_LITERAL {
			arg.Path = p.current().Literal
			p.advance()
			if _, err := p.expect(QUOTE_EXIT); err != nil {
				return nil, err
			}
			if p.current().Type == EOF {
				return arg, nil
			}
			if p.current().Type == FIELD_SEP {
				p.advance()
			}
		}
	} else if firstToken.Type == KEY {
		nextToken := p.peek()
		if nextToken.Type == EOF {
			arg.Path = firstToken.Literal
			p.advance()
			return arg, nil
		}

		if nextToken.Type == FIELD_SEP {
			arg.Path = firstToken.Literal
			p.advance()
			p.advance()
		}
	}

	for p.current().Type != EOF {
		if err := p.parseField(arg); err != nil {
			return nil, err
		}

		if p.current().Type == FIELD_SEP {
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
			if valErr, ok := err.(*ValidationError); ok {
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

	// Handle quoted values
	if p.current().Type == QUOTE_ENTER {
		p.advance()
		if p.current().Type != VALUE_LITERAL {
			return &ParseError{
				Expected: VALUE_LITERAL,
				Got:      p.current().Type,
				Position: p.current().Start,
			}
		}
		field.Value = p.current().Literal
		p.advance()
		if _, err := p.expect(QUOTE_EXIT); err != nil {
			return err
		}
		arg.Fields[keyToken.Literal] = field
		return nil
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

	if p.current().Type == PAREN_ENTER_CALL {
		p.advance()
		if err := p.parseArgs(&field); err != nil {
			return err
		}
		if _, err := p.expect(PAREN_EXIT_CALL); err != nil {
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
	arg.Fields[keyToken.Literal] = field

	return nil
}

func (p *Parser) parseArgs(field *Field) error {
	for p.current().Type != PAREN_EXIT_CALL && p.current().Type != EOF {
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

		if p.current().Type == FIELD_SEP {
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
	return "parse error at position " + string(rune(e.Position)) + ": expected " + e.Expected.String() + ", got " + e.Got.String()
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
