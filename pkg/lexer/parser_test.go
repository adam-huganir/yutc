package lexer

import (
	"reflect"
	"testing"
)

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Arg
		wantErr bool
	}{
		{
			name:  "simple path",
			input: "./my_file.yaml",
			want: &Arg{
				Source: &Field{
					Value: "./my_file.yaml",
					Args:  map[string]string{},
				},
			},
			wantErr: false,
		},
		{
			name:  "single key=value",
			input: "jsonpath=.Secrets",
			want: &Arg{
				JSONPath: &Field{
					Value: ".Secrets",
					Args:  map[string]string{},
				},
			},
			wantErr: false,
		},
		{
			name:  "multiple key=value pairs",
			input: "jsonpath=.Secrets,src=./my_secrets.yaml",
			want: &Arg{
				JSONPath: &Field{
					Value: ".Secrets",
					Args:  map[string]string{},
				},
				Source: &Field{
					Value: "./my_secrets.yaml",
					Args:  map[string]string{},
				},
			},
			wantErr: false,
		},
		{
			name:  "path with key=value pairs",
			input: "./file.yaml,jsonpath=.Secrets",
			want: &Arg{
				Source: &Field{
					Value: "./file.yaml",
					Args:  map[string]string{},
				},
				JSONPath: &Field{
					Value: ".Secrets",
					Args:  map[string]string{},
				},
			},
			wantErr: false,
		},
		{
			name:  "value with function call",
			input: "type=schema(defaults=false)",
			want: &Arg{
				Type: &Field{
					Value: "schema",
					Args: map[string]string{
						"defaults": "false",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "value with multiple i in function call",
			input: "type=schema(a=b,c)",
			want: &Arg{
				Type: &Field{
					Value: "schema",
					Args: map[string]string{
						"a": "b",
						"c": "",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "complex example",
			input: "jsonpath=.Secrets,src=https://example.com/my_secrets.yaml,auth=username:password",
			want: &Arg{
				JSONPath: &Field{
					Value: ".Secrets",
					Args:  map[string]string{},
				},
				Source: &Field{
					Value: "https://example.com/my_secrets.yaml",
					Args:  map[string]string{},
				},
				Auth: &Field{
					Value: "username:password",
					Args:  map[string]string{},
				},
			},
			wantErr: false,
		},
		{
			name:  "mixed path and function call",
			input: "src=./here.json,type=schema(defaults=false)",
			want: &Arg{
				Source: &Field{
					Value: "./here.json",
					Args:  map[string]string{},
				},
				Type: &Field{
					Value: "schema",
					Args: map[string]string{
						"defaults": "false",
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			got, err := p.Parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_Parse_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "invalid key",
			input:   "invalid=value",
			wantErr: "invalid key 'invalid': allowed keys are src, jsonpath, auth, type",
		},
		{
			name:    "invalid key with valid keys",
			input:   "jsonpath=.Secrets,invalid=value",
			wantErr: "invalid key 'invalid': allowed keys are src, jsonpath, auth, type",
		},
		{
			name:    "function call on non-schema value",
			input:   "type=other(arg=val)",
			wantErr: "function 'other' not allowed on key 'type': only schema() is allowed on type",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			_, err := p.Parse()
			if err == nil {
				t.Errorf("Parser.Parse() expected error, got nil")
				return
			}
			if err.Error() != tt.wantErr {
				t.Errorf("Parser.Parse() error = %v, want %v", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestParser_Parse_NoValidation(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "any key allowed",
			input: "customkey=value",
		},
		{
			name:  "any function allowed",
			input: "type=customfunc(arg=val)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParserWithValidation(tt.input, nil)
			_, err := p.Parse()
			if err != nil {
				t.Errorf("Parser.Parse() with no validation should not error, got %v", err)
			}
		})
	}
}
