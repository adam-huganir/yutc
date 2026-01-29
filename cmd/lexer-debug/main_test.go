package main

import (
	"strings"
	"testing"

	"github.com/adam-huganir/yutc/pkg/lexer"
)

func TestFormatTokensAsPython_Exact(t *testing.T) {
	tokens := []lexer.Token{
		{Type: lexer.KEY, Literal: "src", Start: 0, End: 3},
		{Type: lexer.EQ, Literal: "=", Start: 3, End: 4},
		{Type: lexer.VALUE, Literal: "./file\"name.yaml", Start: 4, End: 20},
		{Type: lexer.EOF, Literal: "", Start: 20, End: 20},
	}

	got := formatTokensAsPython(tokens)

	want := "tokens = [\n" +
		"    Token(type=\"KEY\", literal=\"src\", span=(0, 3)),\n" +
		"    Token(type=\"EQ\", literal=\"=\", span=(3, 4)),\n" +
		"    Token(type=\"VALUE\", literal=\"./file\\\"name.yaml\", span=(4, 20)),\n" +
		"    Token(type=\"EOF\", literal=\"\", span=(20, 20))\n" +
		"]\n"

	if got != want {
		t.Fatalf("unexpected formatted tokens\n--- want ---\n%q\n--- got ---\n%q", want, got)
	}
}

func TestFormatTokensAsPython_FromLexer(t *testing.T) {
	input := "src=./here.json,type=schema(defaults=false)"

	l := lexer.NewLexer(input)
	go l.Run()

	tokens := make([]lexer.Token, 0, 32)
	for tok := range l.Lexed() {
		tokens = append(tokens, tok)
	}

	out := formatTokensAsPython(tokens)

	if !strings.HasPrefix(out, "tokens = [\n") {
		t.Fatalf("output missing expected header: %q", out)
	}
	if !strings.HasSuffix(out, "]\n") {
		t.Fatalf("output missing expected footer: %q", out)
	}

	// Spot-check a couple key things without over-specifying the entire token stream.
	if !strings.Contains(out, "Token(type=\"KEY\", literal=\"src\"") {
		t.Fatalf("output missing src key token: %q", out)
	}
	if !strings.Contains(out, "Token(type=\"VALUE\", literal=\"./here.json\"") {
		t.Fatalf("output missing value token: %q", out)
	}
	if !strings.Contains(out, "Token(type=\"PAREN_ENTER_CALL\", literal=\"(\"") {
		t.Fatalf("output missing paren enter token: %q", out)
	}

	// Ensure we printed one line per token (plus header/footer).
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	wantLines := len(tokens) + 2
	if len(lines) != wantLines {
		t.Fatalf("unexpected number of lines: got=%d want=%d", len(lines), wantLines)
	}
}
