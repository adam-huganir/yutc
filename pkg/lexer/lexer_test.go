package lexer

import (
	"fmt"
	"reflect"
	"testing"
)

func testRenderTokens(c chan Token) string {
	t := ""
	for token := range c {
		t = fmt.Sprintf("%s%s(%s) ", t, token.Type, token.Literal)
	}
	if len(t) > 0 {
		t = t[:len(t)-1]
	}
	return t
}

func TestLexing_pprint(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{
			name:   "basic test",
			input:  "./my_file.yaml",
			output: "START() KEY(./my_file.yaml) EOF()",
		},
		{
			name:   "basic test 2",
			input:  "src=./my_file.yaml",
			output: "START() KEY(src) EQ(=) VALUE(./my_file.yaml) EOF()",
		},
		{
			name:   "test whitespace",
			input:  "src = ./my_file.yaml",
			output: "START() KEY(src) EQ(=) VALUE(./my_file.yaml) EOF()",
		},
		{
			name:   "test whitespace",
			input:  "src = ./my_file.yaml",
			output: "START() KEY(src) EQ(=) VALUE(./my_file.yaml) EOF()",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer(tt.input)
			go l.Run()
			if got := testRenderTokens(l.lexed); got != tt.output {
				t.Errorf("testRenderTokens() = \n%s\nwant\n%s", got, tt.output)
			}
		})
	}
}

func TestLexer_peek(t *testing.T) {
	tests := []struct {
		name  string
		input string
		i     int
		want  rune
	}{
		{name: "peek first character", input: "hello", i: 0, want: 'h'},
		{name: "peek second character", input: "hello", i: 1, want: 'e'},
		{name: "peek unicode character", input: "h的llo", i: 1, want: '的'},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer(tt.input)
			if got := l.peek(tt.i); got != tt.want {
				t.Errorf("peek() = %c, want %c", got, tt.want)
			}
		})
	}
}

func Test_lexStart(t *testing.T) {
	tests := []struct {
		name string
		l    *Lexer
		want lexFunc
	}{
		{
			name: "empty input returns nil",
			l:    NewLexer(""),

			want: nil,
		},
		{
			name: "text input returns lexText",
			l:    NewLexer("hello world"),
			want: lexKey,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Consume tokens in a goroutine to prevent deadlock
			go func() {
				for range tt.l.lexed {
					// Drain the channel
				}
			}()

			got := lexStart(tt.l)

			// Close the channel to stop the goroutine
			close(tt.l.lexed)

			// Compare function pointers by checking if both are nil or both are non-nil
			if (got == nil) != (tt.want == nil) {
				t.Errorf("lexStart() = %v, want %v", reflect.ValueOf(got), reflect.ValueOf(tt.want))
			} else if got != nil && tt.want != nil {
				// Both are non-nil, compare function pointers
				gotPtr := reflect.ValueOf(got).Pointer()
				wantPtr := reflect.ValueOf(tt.want).Pointer()
				if gotPtr != wantPtr {
					t.Errorf("lexStart() = %v, want %v", reflect.ValueOf(got), reflect.ValueOf(tt.want))
				}
			}
		})
	}
}

func Test_lexKey(t *testing.T) {
	type args struct {
		l *Lexer
	}
	tests := []struct {
		name       string
		args       args
		want       lexFunc
		wantTokens []Token
	}{
		{
			name: "basic test",
			args: args{
				l: NewLexer(`hello world`),
			},
			want: nil,
			wantTokens: []Token{
				Token{Type: KEY, Literal: `hello world`, Start: 0, End: 11},
				Token{Type: EOF, Literal: "", Start: 11, End: 11},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Consume tokens in a goroutine to prevent deadlock
			var out []Token
			done := make(chan bool)
			go func() {
				for i := range tt.args.l.lexed {
					out = append(out, i)
				}
				done <- true
			}()

			got := lexKey(tt.args.l)

			// Close the channel to stop the goroutine
			close(tt.args.l.lexed)
			<-done

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("lexKey() = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(out, tt.wantTokens) {
				t.Errorf("lexKey() = %v, want %v", out, tt.wantTokens)
			}
		})
	}
}
