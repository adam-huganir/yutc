package data

import (
	"testing"

	"github.com/adam-huganir/yutc/pkg/files"
)

func TestParseDataFileArg(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedKey  string
		expectedPath string
		expectError  bool
	}{
		{
			name:         "simple path",
			input:        "./my_file.yaml",
			expectedKey:  "",
			expectedPath: "./my_file.yaml",
			expectError:  false,
		},
		{
			name:         "path with key",
			input:        "key=Secrets,src=./my_secrets.yaml",
			expectedKey:  "Secrets",
			expectedPath: "./my_secrets.yaml",
			expectError:  false,
		},
		{
			name:         "path with key (reversed order)",
			input:        "src=./my_secrets.yaml,key=Secrets",
			expectedKey:  "Secrets",
			expectedPath: "./my_secrets.yaml",
			expectError:  false,
		},
		{
			name:         "path with key and spaces",
			input:        "key=Secrets, src=./my_secrets.yaml",
			expectedKey:  "Secrets",
			expectedPath: "./my_secrets.yaml",
			expectError:  false,
		},
		{
			name:         "missing src parameter",
			input:        "key=Secrets",
			expectedKey:  "",
			expectedPath: "",
			expectError:  true,
		},
		{
			name:         "stdin",
			input:        "-",
			expectedKey:  "",
			expectedPath: "-",
			expectError:  false,
		},
		{
			name:         "url",
			input:        "https://example.com/data.yaml",
			expectedKey:  "",
			expectedPath: "https://example.com/data.yaml",
			expectError:  false,
		},
		{
			name:         "url with key",
			input:        "key=Remote,src=https://example.com/data.yaml",
			expectedKey:  "Remote",
			expectedPath: "https://example.com/data.yaml",
			expectError:  false,
		},
		{
			name:         "invalid key",
			input:        "key=Secrets,source=./my_secrets.yaml",
			expectedKey:  "",
			expectedPath: "",
			expectError:  true,
		},
		{
			name:         "partial no key in entry",
			input:        "key=Secrets,./my_file.yaml",
			expectedKey:  "",
			expectedPath: "",
			expectError:  true,
		},
		{
			name:         "file named src=dumb_filename.yaml",
			input:        "key=Secrets2,src=src=dumb_filename.yaml",
			expectedKey:  "Secrets2",
			expectedPath: "src=dumb_filename.yaml",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := files.ParseDataFileArg(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Key != tt.expectedKey {
				t.Errorf("expected key %q but got %q", tt.expectedKey, result.Key)
			}

			if result.Path != tt.expectedPath {
				t.Errorf("expected path %q but got %q", tt.expectedPath, result.Path)
			}
		})
	}
}
