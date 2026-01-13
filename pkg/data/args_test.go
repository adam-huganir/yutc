package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/theory/jsonpath"
)

func TestParseFileArg(t *testing.T) {
	root := jsonpath.MustParse("$")
	tests := []struct {
		name         string
		input        string
		expectedKey  *jsonpath.Path
		expectedPath string
		expectError  string
		kind         string
	}{
		{
			name:         "simple path",
			input:        "./my_file.yaml",
			expectedKey:  root,
			expectedPath: "my_file.yaml",
			expectError:  "",
		},
		{
			name:         "path with key",
			input:        "jsonpath=.Secrets,src=./my_secrets.yaml",
			expectedKey:  jsonpath.MustParse("$.Secrets"),
			expectedPath: "my_secrets.yaml",
			expectError:  "",
		},
		{
			name:         "path with key (reversed order)",
			input:        "src=./my_secrets.yaml,jsonpath=.Secrets",
			expectedKey:  jsonpath.MustParse("$.Secrets"),
			expectedPath: "my_secrets.yaml",
			expectError:  "",
		},
		{
			name:         "path with key and spaces",
			input:        "jsonpath=.Secrets, src=./my_secrets.yaml",
			expectedKey:  jsonpath.MustParse("$.Secrets"),
			expectedPath: "my_secrets.yaml",
			expectError:  "",
		},
		{
			name:         "missing src parameter",
			input:        "jsonpath=.Secrets",
			expectedKey:  root,
			expectedPath: "",
			expectError:  "missing or empty 'src' parameter in argument",
		},
		{
			name:         "stdin",
			input:        "-",
			expectedKey:  root,
			expectedPath: "-",
			expectError:  "",
		},
		{
			name:         "url",
			input:        "https://example.com/data.yaml",
			expectedKey:  root,
			expectedPath: "https://example.com/data.yaml",
			expectError:  "",
		},
		{
			name:         "url with key",
			input:        "jsonpath=.Remote,src=https://example.com/data.yaml",
			expectedKey:  jsonpath.MustParse("$.Remote"),
			expectedPath: "https://example.com/data.yaml",
			expectError:  "",
		},
		{
			name:         "invalid key",
			input:        "jsonpath=.Secrets,source=./my_secrets.yaml",
			expectedKey:  root,
			expectedPath: "",
			expectError:  "invalid key 'source': allowed keys are src, jsonpath, auth, type",
		},
		{
			name:         "partial no key in entry",
			input:        "jsonpath=.Secrets,./my_file.yaml",
			expectedKey:  root,
			expectedPath: "",
			expectError:  "invalid key './my_file.yaml': allowed keys are src, jsonpath, auth, type",
		},
		{
			name:         "file named src=dumb_filename.yaml",
			input:        "jsonpath=.Secrets2,src=src=dumb_filename.yaml",
			expectedKey:  jsonpath.MustParse("$.Secrets2"),
			expectedPath: "src=dumb_filename.yaml",
			expectError:  "",
		},
		{
			name:         "template and a jsonpath (error)",
			input:        "jsonpath=.test,src=something.tmpl",
			expectedKey:  jsonpath.MustParse("$.Secrets2"),
			expectedPath: "something.tmpl",
			expectError:  "key parameter is not supported for template arguments",
			kind:         "template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ParseFileArg(tt.input, FileKind(tt.kind))

			if tt.expectError != "" {
				assert.Errorf(t, err, "expected error but got none")
				assert.ErrorContains(t, err, tt.expectError)
				return
			}

			assert.NoErrorf(t, err, "unexpected error: %v", err)
			if !assert.NotEmptyf(t, results, "expected at least one result for %q", tt.input) {
				return
			}
			result := results[0]

			assert.Equalf(t, tt.expectedKey.String(), result.JSONPath.String(),
				"expected key %q but got %q", tt.expectedKey, result.JSONPath)

			assert.Equalf(t, result.Path, tt.expectedPath,
				"expected path %q but got %q", tt.expectedPath, result.Path)

		})
	}
}
