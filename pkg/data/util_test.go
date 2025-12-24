package data

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTailMergeFiles(t *testing.T) {
	type args struct {
		paths []string
	}
	tests := []struct {
		name          string
		args          args
		wantSum       string
		wantErr       assert.ErrorAssertionFunc
		errorContains string
	}{
		{
			name: "no op single file",
			args: args{
				paths: []string{"set-test.tmpl"},
			},
			wantSum: "bf9bde859d6d066bbd88da034c2bbe0febe4d1c3519bc7ea10bc26877c22c703",
			wantErr: assert.NoError,
		},
		{
			name: "merge 2 data",
			args: args{
				paths: []string{"set-test.tmpl", "yamlOpts.tmpl"},
			},
			wantSum: "a8c5b25c3eab238d3e5c5aaec61cb10c73ebbb290b87dfc4a995e8363a16d0ae",
			wantErr: assert.NoError,
		},
		{
			name: "file does not exist",
			args: args{
				paths: []string{"set-test.tmpl", "nonexistent.tmpl"},
			},
			wantErr:       assert.Error,
			errorContains: "does not exist",
		},
		{
			name: "duplicate data single file output",
			args: args{
				paths: []string{"set-test.tmpl", "set-test.tmpl"},
			},
			wantSum: "bf9bde859d6d066bbd88da034c2bbe0febe4d1c3519bc7ea10bc26877c22c703",
			wantErr: assert.NoError,
		},
		{
			name: "duplicate data merged output",
			args: args{
				paths: []string{"set-test.tmpl", "yamlOpts.tmpl", "set-test.tmpl"},
			},
			wantSum: "a8c5b25c3eab238d3e5c5aaec61cb10c73ebbb290b87dfc4a995e8363a16d0ae",
			wantErr: assert.NoError,
		},
		{
			name: "empty list",
			args: args{
				paths: []string{},
			},
			wantSum: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", // sha256 of empty string
			wantErr: assert.NoError,
		},
		{
			name: "large number of data including directories",
			args: args{
				paths: []string{
					"./data/yamlOptionsBad.yaml",
					"./data/yamlOptions.yaml",
					"./functions/docker-compose.yaml.tmpl",
					"./functions/fn.tmpl",
					"./ls-like.tmpl",
				},
			},
			wantSum: "b9cae26147625c22ca64b6fc060265b957e5f124c38d8a525ab951dd68d1fe58",
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := []string{}
			for _, p := range tt.args.paths {
				paths = append(paths, path.Join("../../testFiles", p))
			}
			gotOut, err := TailMergeFiles(paths)
			if !tt.wantErr(t, err, fmt.Sprintf("TailMergeFiles(%v)", paths)) {
				t.Logf("unexpected error: %v", err)
				t.FailNow()
			}
			if tt.errorContains != "" {
				assert.Contains(t, err.Error(), tt.errorContains)
				return
			}
			assert.Nil(t, err) //
			sha := sha256.New()
			sha.Write([]byte(gotOut))
			sum := hex.EncodeToString(sha.Sum(nil))
			// Use this to update expected outputs when something changes
			if !assert.Equal(t, tt.wantSum, sum, "TailMergeFiles(%v)", paths) {
				t.Logf("SHA256 does not match, check if source data have been changed and update test with actual sum: %s", sum)
				t.Logf("actual output:\n%s", gotOut)
				t.FailNow()
			}
		})
	}
}
