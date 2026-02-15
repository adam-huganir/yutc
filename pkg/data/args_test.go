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
		},
		{
			name:         "schema defaults false",
			input:        "src=./schema.yaml,type=schema(defaults=false)",
			expectedKey:  root,
			expectedPath: "schema.yaml",
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

			if tt.name == "schema defaults false" {
				assert.True(t, result.IsSchema)
				assert.True(t, result.Schema.DisableDefaults)
			}
		})
	}
}

func TestParseTemplateArg(t *testing.T) {
	tests := []struct {
		name                string
		input               string
		expectedPath        string
		expectError         string
		isCommon            bool
		expectedBearerToken string
		expectedBasicAuth   string
	}{
		{
			name:         "simple template path",
			input:        "./my_template.tmpl",
			expectedPath: "my_template.tmpl",
		},
		{
			name:        "template and a jsonpath (error)",
			input:       "jsonpath=.test,src=something.tmpl",
			expectError: "key parameter is not supported for template arguments",
		},
		{
			name:         "common template",
			input:        "./shared.tmpl",
			expectedPath: "shared.tmpl",
			isCommon:     true,
		},
		{
			name:                "template URL with bearer auth",
			input:               "src=https://example.com/template.tmpl,auth=my-secret-token",
			expectedPath:        "https://example.com/template.tmpl",
			expectedBearerToken: "my-secret-token",
		},
		{
			name:              "template URL with basic auth",
			input:             "src=https://example.com/template.tmpl,auth=user:pass",
			expectedPath:      "https://example.com/template.tmpl",
			expectedBasicAuth: "user:pass",
		},
		{
			name:                "common template URL with bearer auth",
			input:               "src=https://example.com/shared.tmpl,auth=token123",
			expectedPath:        "https://example.com/shared.tmpl",
			isCommon:            true,
			expectedBearerToken: "token123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseTemplateArg(tt.input, tt.isCommon)

			if tt.expectError != "" {
				assert.Errorf(t, err, "expected error but got none")
				assert.ErrorContains(t, err, tt.expectError)
				return
			}

			assert.NoErrorf(t, err, "unexpected error: %v", err)
			assert.NotNil(t, result)
			assert.Equalf(t, tt.expectedPath, result.Name,
				"expected path %q but got %q", tt.expectedPath, result.Name)
			assert.Equal(t, tt.isCommon, result.IsCommon)
			assert.Equal(t, tt.expectedBearerToken, result.Auth.BearerToken)
			assert.Equal(t, tt.expectedBasicAuth, result.Auth.BasicAuth)
		})
	}
}
