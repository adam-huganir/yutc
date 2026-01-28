package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/adam-huganir/yutc/pkg/lexer"
)

func formatTokensAsPython(tokens []lexer.Token) string {
	var b strings.Builder
	b.WriteString("tokens = [\n")
	for i, token := range tokens {
		b.WriteString("    Token(")
		b.WriteString("type=")
		b.WriteString(strconv.Quote(token.Type.String()))
		b.WriteString(", literal=")
		b.WriteString(strconv.Quote(token.Literal))
		b.WriteString(", span=(")
		b.WriteString(strconv.Itoa(token.Start))
		b.WriteString(", ")
		b.WriteString(strconv.Itoa(token.End))
		b.WriteString(")")
		b.WriteString(")")
		if i != len(tokens)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("]\n")
	return b.String()
}

func main() {
	var (
		showTokens  bool
		showTokens2 bool
		showAST     bool
		noValidate  bool
	)

	flag.BoolVar(&showTokens, "tokens", false, "Show lexer tokens")
	flag.BoolVar(&showTokens2, "tokens2", true, "Show lexer tokens")
	flag.BoolVar(&showAST, "ast", true, "Show parsed AST (default)")
	flag.BoolVar(&noValidate, "no-validate", false, "Disable validation")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: lexer-debug [flags] <statement>")
		fmt.Fprintln(os.Stderr, "\nFlags:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintln(os.Stderr, "  lexer-debug './my_file.yaml'")
		fmt.Fprintln(os.Stderr, "  lexer-debug 'path=.Secrets,src=./my_secrets.yaml'")
		fmt.Fprintln(os.Stderr, "  lexer-debug -tokens 'type=schema(defaults=false)'")
		os.Exit(1)
	}

	input := args[0]

	// Show tokens if requested
	if showTokens {
		fmt.Println("=== TOKENS ===")
		l := lexer.NewLexer(input)
		go l.Run()
		for token := range l.Lexed() {
			fmt.Printf("%-20s %q\n", token.Type, token.Literal)
		}
		fmt.Println()
	}

	if showTokens2 {
		l := lexer.NewLexer(input)
		go l.Run()
		tokens := make([]lexer.Token, 0, 32)
		for token := range l.Lexed() {
			tokens = append(tokens, token)
		}
		fmt.Print(formatTokensAsPython(tokens))
	}

	// Parse and show AST
	if showAST {
		fmt.Println("=== AST ===")
		var p *lexer.Parser
		if noValidate {
			p = lexer.NewParserWithValidation(input, nil)
		} else {
			p = lexer.NewParser(input)
		}

		arg, err := p.Parse()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
			os.Exit(1)
		}

		// Pretty print the AST
		output, err := json.MarshalIndent(arg, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "JSON marshal error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(output))
	}
}
