package lexer

import (
	"errors"
	"strconv"
)

type Arg struct {
	Source   *SourceField
	JSONPath *JSONPathField
	Type     *TypeField
	Auth     *AuthField
}

func (a *Arg) Map() map[string]FieldInterface {
	return map[string]FieldInterface{
		"src":      a.Source,
		"jsonpath": a.JSONPath,
		"type":     a.Type,
		"auth":     a.Auth,
	}
}

// Specific field types for each argument kind
type FieldInterface interface {
	GetValue() string
	GetArgs() map[string]string
}

type SourceField struct {
	Value string
}

func (f *SourceField) GetValue() string { return f.Value }
func (f *SourceField) GetArgs() map[string]string { return nil }

type JSONPathField struct {
	Value string
}

func (f *JSONPathField) GetValue() string { return f.Value }
func (f *JSONPathField) GetArgs() map[string]string { return nil }

type TypeField struct {
	Value string
	Args  map[string]string
}

func (f *TypeField) GetValue() string { return f.Value }
func (f *TypeField) GetArgs() map[string]string { return f.Args }

type AuthField struct {
	Value string
	Args  map[string]string
}

func (f *AuthField) GetValue() string { return f.Value }
func (f *AuthField) GetArgs() map[string]string { return f.Args }

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

func DefaultFunctionValidator(key, functionName string, args map[string]string) error {
	if key == "type" && functionName == "schema" {
		// Validate schema arguments
		for argName, argValue := range args {
			if argName != "defaults" {
				return &ValidationError{
					Message: "invalid argument '" + argName + "' for schema(): only 'defaults' is allowed",
					Key:     key,
					Value:   functionName,
				}
			}
			// Validate that defaults value is a boolean
			if argValue != "true" && argValue != "false" {
				return &ValidationError{
					Message: "invalid value for 'defaults' argument: must be 'true' or 'false'",
					Key:     key,
					Value:   functionName,
				}
			}
		}
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
			arg.Source = &SourceField{
				Value: firstToken.Literal,
			}
			p.advance()
			return arg, nil
		}

		if nextToken.Type == FieldSep {
			// Non-keyed value followed by more fields, treat as src
			arg.Source = &SourceField{
				Value: firstToken.Literal,
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

	// Temporary field data
	fieldValue := ""
	fieldArgs := make(map[string]string)

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
		// Only type and auth fields support function calls
		// For other fields, include the parentheses in the value
		if keyToken.Literal != "type" && keyToken.Literal != "auth" {
			// Treat parentheses as part of the value
			fieldValue = valueToken.Literal + "("
			p.advance() // consume ParenEnterCall
			
			// Collect everything until ParenExitCall
			for p.current().Type != ParenExitCall && p.current().Type != EOF {
				fieldValue += p.current().Literal
				p.advance()
			}
			
			if p.current().Type == ParenExitCall {
				fieldValue += ")"
				p.advance() // consume ParenExitCall
			}
		} else {
			// Process as function call for type and auth fields
			p.advance()
			if err := p.parseArgs(&fieldArgs); err != nil {
				return err
			}
			if _, err := p.expect(ParenExitCall); err != nil {
				return err
			}

			if p.validation != nil && p.validation.ValidateFunction != nil {
				if err := p.validation.ValidateFunction(keyToken.Literal, valueToken.Literal, fieldArgs); err != nil {
					var valErr *ValidationError
					if errors.As(err, &valErr) {
						valErr.Position = valueToken.Start
					}
					return err
				}
			}
			
			fieldValue = valueToken.Literal
		}
	} else {
		fieldValue = valueToken.Literal
	}

	// Set the appropriate field on Arg based on key name
	switch keyToken.Literal {
	case "src":
		arg.Source = &SourceField{
			Value: fieldValue,
		}
	case "jsonpath":
		arg.JSONPath = &JSONPathField{
			Value: fieldValue,
		}
	case "type":
		arg.Type = &TypeField{
			Value: fieldValue,
			Args:  fieldArgs,
		}
	case "auth":
		arg.Auth = &AuthField{
			Value: fieldValue,
			Args:  fieldArgs,
		}
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

func (p *Parser) parseArgs(args *map[string]string) error {
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
				(*args)[keyToken.Literal] = valueToken.Literal
			} else {
				(*args)[keyToken.Literal] = ""
			}
		} else {
			// No EQ means key without value
			(*args)[keyToken.Literal] = ""
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
