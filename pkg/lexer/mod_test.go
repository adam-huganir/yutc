package lexer

import (
	"reflect"
	"testing"
)

func TestLexer_peek(t *testing.T) {
	type fields struct {
		input string
	}
	type args struct {
		i int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   rune
	}{
		{
			name: "peek first character",
			fields: fields{
				input: "hello",
			},
			args: args{i: 0},
			want: 'h',
		},
		{
			name: "peek second character",
			fields: fields{
				input: "hello",
			},
			args: args{i: 1},
			want: 'e',
		},
		{
			name: "peek unicode character",
			fields: fields{
				input: "h的llo",
			},
			args: args{i: 1},
			want: '的',
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer(tt.fields.input)
			if got := l.peek(tt.args.i); got != tt.want {
				t.Errorf("peek() = %c, want %c", got, tt.want)
			}
		})
	}
}

func Test_lexStart(t *testing.T) {
	type args struct {
		l *Lexer
	}
	tests := []struct {
		name string
		args args
		want lexFunc
	}{
		{
			name: "empty input returns nil",
			args: args{
				l: NewLexer(""),
			},
			want: nil,
		},
		{
			name: "text input returns lexText",
			args: args{
				l: NewLexer("hello world"),
			},
			want: lexKey,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Consume tokens in a goroutine to prevent deadlock
			go func() {
				for range tt.args.l.items {
					// Drain the channel
				}
			}()

			got := lexStart(tt.args.l)

			// Close the channel to stop the goroutine
			close(tt.args.l.items)

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
				Token{Type: KEY, Literal: `hello world`},
				Token{Type: EOF, Literal: ""},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Consume tokens in a goroutine to prevent deadlock
			var out []Token
			go func() {
				for i := range tt.args.l.items {
					out = append(out, i)
				}
			}()


			got := lexKey(tt.args.l)

			// Close the channel to stop the goroutine
			close(tt.args.l.items)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("lexKey() = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(out, tt.wantTokens) {
				t.Errorf("lexKey() = %v, want %v", out, tt.wantTokens)
			}
		})
	}
}
