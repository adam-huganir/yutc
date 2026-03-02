package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/theory/jsonpath"
)

func TestParseDataArg(t *testing.T) {
	root := jsonpath.MustParse("$")
	tests := []struct {
		name         string
		input        string
		expectedKey  *jsonpath.Path
		expectedPath string
		expectError  string
	}{
		{
			name:         "simple path",
			input:        "./my_file.yaml",
			expectedKey:  root,
			expectedPath: "my_file.yaml",
		},
		{
			name:         "path with key",
			input:        "jsonpath=.Secrets,src=./my_secrets.yaml",
			expectedKey:  jsonpath.MustParse("$.Secrets"),
			expectedPath: "my_secrets.yaml",
		},
		{
			name:         "path with key (reversed order)",
			input:        "src=./my_secrets.yaml,jsonpath=.Secrets",
			expectedKey:  jsonpath.MustParse("$.Secrets"),
			expectedPath: "my_secrets.yaml",
		},
		{
			name:         "path with key and spaces",
			input:        "jsonpath=.Secrets, src=./my_secrets.yaml",
			expectedKey:  jsonpath.MustParse("$.Secrets"),
			expectedPath: "my_secrets.yaml",
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
		},
		{
			name:         "url",
			input:        "https://example.com/data.yaml",
			expectedKey:  root,
			expectedPath: "https://example.com/data.yaml",
		},
		{
			name:         "url with key",
			input:        "jsonpath=.Remote,src=https://example.com/data.yaml",
			expectedKey:  jsonpath.MustParse("$.Remote"),
			expectedPath: "https://example.com/data.yaml",
		},
		{
			name:         "invalid key",
			input:        "jsonpath=.Secrets,bogus=./my_secrets.yaml",
			expectedKey:  root,
			expectedPath: "",
			expectError:  "invalid key 'bogus': allowed keys are auth, jsonpath, kind, path, ref, src, type",
		},
		{
			name:         "partial no key in entry",
			input:        "jsonpath=.Secrets,./my_file.yaml",
			expectedKey:  root,
			expectedPath: "",
			expectError:  "invalid key './my_file.yaml': allowed keys are auth, jsonpath, kind, path, ref, src, type",
		},
		{
			name:         "file named src=dumb_filename.yaml",
			input:        "jsonpath=.Secrets2,src=src=dumb_filename.yaml",
			expectedKey:  jsonpath.MustParse("$.Secrets2"),
			expectedPath: "src=dumb_filename.yaml",
		},
		{
			name:         "schema defaults false",
			input:        "src=./schema.yaml,kind=schema(defaults=false)",
			expectedKey:  root,
			expectedPath: "schema.yaml",
		},
		{
			name:         "invalid kind",
			input:        "src=./schema.yaml,kind=not-schema",
			expectedKey:  root,
			expectedPath: "",
			expectError:  "invalid kind \"not-schema\": only 'schema' is supported",
		},
		{
			name:         "schema with invalid argument",
			input:        "src=./schema.yaml,kind=schema(invalid=true)",
			expectedKey:  root,
			expectedPath: "",
			expectError:  "invalid argument \"invalid\" for kind=schema(): only 'defaults' is allowed",
		},
		{
			name:         "schema with invalid defaults value",
			input:        "src=./schema.yaml,kind=schema(defaults=maybe)",
			expectedKey:  root,
			expectedPath: "",
			expectError:  "invalid value for 'defaults' argument: must be 'true' or 'false'",
		},
		{
			name:         "explicit source kind",
			input:        "src=./my_file.yaml,type=file",
			expectedKey:  root,
			expectedPath: "my_file.yaml",
		},
		{
			name:         "git known host source",
			input:        "src=github.com/org/repo",
			expectedKey:  root,
			expectedPath: "https://github.com/org/repo",
		},
		{
			name:         "git source with ref and path",
			input:        "src=github.com/org/repo,ref=main,path=values.yaml",
			expectedKey:  root,
			expectedPath: "https://github.com/org/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ParseDataArg(tt.input)

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

			assert.Equalf(t, result.Name, tt.expectedPath,
				"expected path %q but got %q", tt.expectedPath, result.Name)

			if tt.name == "git known host source" || tt.name == "git source with ref and path" {
				assert.Equal(t, "git", result.Source.String())
				assert.NotNil(t, result.Git)
			}

			if tt.name == "schema defaults false" {
				assert.True(t, result.IsSchema)
				assert.True(t, result.Schema.DisableDefaults)
			}
		})
	}
}
