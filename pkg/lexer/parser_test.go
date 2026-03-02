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
				Source: &SourceField{
					Value: "./my_file.yaml",
				},
			},
			wantErr: false,
		},
		{
			name:  "single key=value",
			input: "jsonpath=.Secrets",
			want: &Arg{
				JSONPath: &JSONPathField{
					Value: ".Secrets",
				},
			},
			wantErr: false,
		},
		{
			name:  "multiple key=value pairs",
			input: "jsonpath=.Secrets,src=./my_secrets.yaml",
			want: &Arg{
				JSONPath: &JSONPathField{
					Value: ".Secrets",
				},
				Source: &SourceField{
					Value: "./my_secrets.yaml",
				},
			},
			wantErr: false,
		},
		{
			name:  "path with key=value pairs",
			input: "./file.yaml,jsonpath=.Secrets",
			want: &Arg{
				Source: &SourceField{
					Value: "./file.yaml",
				},
				JSONPath: &JSONPathField{
					Value: ".Secrets",
				},
			},
			wantErr: false,
		},
		{
			name:  "value with function call",
			input: "kind=schema(defaults=false)",
			want: &Arg{
				Kind: &KindField{
					Value: "schema",
					Args: map[string]string{
						"defaults": "false",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "value with multiple arguments in function call",
			input: "kind=schema(defaults=false)",
			want: &Arg{
				Kind: &KindField{
					Value: "schema",
					Args: map[string]string{
						"defaults": "false",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "complex example",
			input: "jsonpath=.Secrets,src=https://example.com/my_secrets.yaml,auth=username:password",
			want: &Arg{
				JSONPath: &JSONPathField{
					Value: ".Secrets",
				},
				Source: &SourceField{
					Value: "https://example.com/my_secrets.yaml",
				},
				Auth: &AuthField{
					Value: "username:password",
					Args:  map[string]string{},
				},
			},
			wantErr: false,
		},
		{
			name:  "ref field",
			input: "src=./repo,ref=main",
			want: &Arg{
				Source: &SourceField{
					Value: "./repo",
				},
				Ref: &RefField{
					Value: "main",
				},
			},
			wantErr: false,
		},
		{
			name:  "path field",
			input: "src=./repo,path=templates/app.yaml.tmpl",
			want: &Arg{
				Source: &SourceField{
					Value: "./repo",
				},
				Path: &PathField{
					Value: "templates/app.yaml.tmpl",
				},
			},
			wantErr: false,
		},
		{
			name:  "type field",
			input: "src=./repo,type=git(submodules=recurse)",
			want: &Arg{
				Source: &SourceField{
					Value: "./repo",
				},
				Type: &TypeField{
					Value: "git",
					Args: map[string]string{
						"submodules": "recurse",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "mixed path and function call",
			input: "src=./here.json,kind=schema(defaults=false)",
			want: &Arg{
				Source: &SourceField{
					Value: "./here.json",
				},
				Kind: &KindField{
					Value: "schema",
					Args: map[string]string{
						"defaults": "false",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "filename with parentheses",
			input: "src=myfile(1).docx",
			want: &Arg{
				Source: &SourceField{
					Value: "myfile(1).docx",
				},
			},
			wantErr: false,
		},
		{
			name:  "jsonpath with parentheses",
			input: "jsonpath=.Secrets(backup)",
			want: &Arg{
				JSONPath: &JSONPathField{
					Value: ".Secrets(backup)",
				},
			},
			wantErr: false,
		},
		{
			name:  "filename with escaped comma",
			input: "src=my\\,file.txt",
			want: &Arg{
				Source: &SourceField{
					Value: "my,file.txt",
				},
			},
			wantErr: false,
		},
		{
			name:  "jsonpath with escaped comma",
			input: "jsonpath=.Secrets\\,backup",
			want: &Arg{
				JSONPath: &JSONPathField{
					Value: ".Secrets,backup",
				},
			},
			wantErr: false,
		},
		{
			name:  "multiple fields with escaped characters",
			input: "src=my\\,file.txt,jsonpath=.Secrets\\,backup",
			want: &Arg{
				Source: &SourceField{
					Value: "my,file.txt",
				},
				JSONPath: &JSONPathField{
					Value: ".Secrets,backup",
				},
			},
			wantErr: false,
		},
		{
			name:  "auth field with escaped characters",
			input: "auth=user\\:password\\,123",
			want: &Arg{
				Auth: &AuthField{
					Value: "user:password,123",
					Args:  map[string]string{},
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
			wantErr: "invalid key 'invalid': allowed keys are auth, jsonpath, kind, path, ref, src, type",
		},
		{
			name:    "invalid key with valid keys",
			input:   "jsonpath=.Secrets,invalid=value",
			wantErr: "invalid key 'invalid': allowed keys are auth, jsonpath, kind, path, ref, src, type",
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
