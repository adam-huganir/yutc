package quote

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/adam-huganir/yutc/pkg/files"
	"github.com/adam-huganir/yutc/pkg/util"
)

func TestLuaQuote(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name       string
		args       args
		want       string
		expectFail bool
	}{
		{
			name: "empty string",
			args: args{s: ""},
			want: `""`,
		},
		{
			name: "simple string",
			args: args{s: "hello"},
			want: `"hello"`,
		},
		{
			name: "string with spaces",
			args: args{s: "hello world"},
			want: `"hello world"`,
		},
		{
			name: "string with double quotes",
			args: args{s: `he"llo`},
			want: `"he\"llo"`,
		},
		{
			name: "string with single quotes",
			args: args{s: `he'llo`},
			want: `"he'llo"`,
		},
		{
			name: "string with backslash",
			args: args{s: `he\llo`},
			want: `"he\\llo"`,
		},
		{
			name: "string with newline",
			args: args{s: "hello\nworld"},
			want: `"hello\nworld"`,
		},
		{
			name: "string with tab",
			args: args{s: "hello\tworld"},
			want: `"hello\tworld"`,
		},
		{
			name: "string with mixed special characters",
			args: args{s: `he"llo\nworld's "best"`},
			want: `"he\"llo\\nworld's \"best\""`,
		},
		{
			name:       "string with unicode characters",
			args:       args{s: "😀"},
			want:       `"\240\159\152\128"`,
			expectFail: false,
		},
		{
			name: "string with null character",
			args: args{s: "hello\x00world"},
			want: `"hello\000world"`,
		},
		{
			name: "string with unicode characters 2",
			args: args{s: "こんにちは"},
			want: `"\227\129\147\227\130\147\227\129\171\227\129\161\227\129\175"`,
		},
		{
			name: "string with mixed unicode and ascii",
			args: args{s: "hello 世界 (keeping track of width)"},
			want: `"hello \228\184\150\231\149\140 (keeping track of width)"`,
		},
		{
			name: "string with unicode euro sign",
			args: args{s: "€"},
			want: `"\226\130\172"`,
		},
		{
			name: "multiline string",
			args: args{s: util.MustDedent(`
				This is the first line
				this is the second

				oops missed one
			`)},
			want: `"This is the first line\nthis is the second\n\noops missed one\n"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LuaQuote(tt.args.s); got != tt.want {
				if tt.expectFail {
					return
				}
				t.Errorf("LuaQuote() = %v, want %v", got, tt.want)
			} else if tt.expectFail {
				t.Errorf("LuaQuote() = %v, want fail", got)
			}

			// Verify that the quoted string is valid Lua by executing it.
			// The `print` function in Lua adds a newline to the output.
			luaCode := fmt.Sprintf("print(%s)", LuaQuote(tt.args.s))
			tempfile := filepath.Join(t.TempDir(), "test.lua")
			err := os.WriteFile(tempfile, []byte(luaCode), 0644)
			cmd := exec.Command("lua", tempfile)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("lua execution failed: %v\noutput: %s", err, string(out))
			}

			// The output from `print` has a trailing newline. We need to remove it
			// to compare with the original string.
			outString := string(out)
			if runtime.GOOS == "windows" {
				outString = strings.ReplaceAll(outString, "\r\n", "\n")
			}
			outString = outString[:len(outString)-1]
			if outString != tt.args.s {
				t.Errorf("Lua execution stdout = %q, want %q", outString, tt.args.s)
			}
		})
	}
}

func TestShellQuote(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name       string
		args       args
		want       string
		expectFail bool
	}{
		{
			name: "empty string",
			args: args{s: ""},
			want: `''`,
		},
		{
			name: "simple string",
			args: args{s: "hello"},
			want: `hello`,
		},
		{
			name: "string with spaces",
			args: args{s: "hello world"},
			want: `'hello world'`,
		},
		{
			name: "string with single quotes",
			args: args{s: `it's a trap`},
			want: `'it'"'"'s a trap'`,
		},
		{
			name: "string with double quotes",
			args: args{s: `hello "world"`},
			want: `'hello "world"'`,
		},
		{
			name: "string with dollar sign",
			args: args{s: `hello $world`},
			want: `'hello $world'`,
		},
		{
			name: "string with backticks",
			args: args{s: "hello `world`"},
			want: "'hello `world`'",
		},
		{
			name: "string with unicode characters",
			args: args{s: "こんにちは"},
			want: `'こんにちは'`,
		},
		{
			name: "string with mixed unicode and special chars",
			args: args{s: "hello 世界"},
			want: `'hello 世界'`,
		},
		{
			name: "multiline string",
			args: args{s: util.MustDedent(`
				This is the first line
				this is the second

				oops missed one
			`)},
			want: util.MustDedent(`
				'This is the first line
				this is the second

				oops missed one
				'`),
		},
		{
			name: "shellspecific",
			args: args{s: `\e[0;31m[\u@\h \W]\$ \e[m`},
			want: `'\e[0;31m[\u@\h \W]\$ \e[m'`,
		},
		{
			name:       "shell specific more",
			args:       args{s: "hello \for \roger \tales \versus \\066 \\Unit \\characters \\elf \\unguent or \\ as well"},
			want:       "'hello \for \roger \tales \versus \\066 \\Unit \\characters \\elf \\unguent or \\ as well'",
			expectFail: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShellQuote(tt.args.s)
			if got != tt.want {
				t.Errorf("ShellQuote() = %v, want %v", got, tt.want)
			}

			tmpDir := t.TempDir()
			scriptFile := filepath.Join(tmpDir, "test.sh")
			_ = os.WriteFile(scriptFile, []byte("echo "+got), 0644)

			linuxPath := scriptFile
			if runtime.GOOS == "windows" {
				n, _ := regexp.Compile(`^\w:`)
				linuxPath = "/mnt/c" + n.ReplaceAllString(files.NormalizeFilepath(scriptFile), "")
			}

			// Verify that the quoted string is valid shell by executing it.
			var cmd *exec.Cmd
			if runtime.GOOS == "windows" {
				cmd = exec.Command("bash", linuxPath)
			} else {
				cmd = exec.Command("bash", linuxPath)
			}

			out, err := cmd.CombinedOutput()

			if err != nil {
				t.Fatalf("sh execution failed: %v\noutput: %s", err, string(out))
			}

			outString := string(out)
			// `echo` adds a trailing newline. We need to remove it.
			if runtime.GOOS == "windows" {
				outString = strings.ReplaceAll(outString, "\r\n", "\n")
			}
			outString = strings.TrimSuffix(outString, "\n")

			if !tt.expectFail && outString != tt.args.s {
				t.Errorf("Shell execution stdout = %q, want %q", outString, tt.args.s)
			}
			return
		})
	}
}
