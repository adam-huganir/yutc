package data

import (
	"fmt"
	"os"
	"path"
	"strings"
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
		wantErr       assert.ErrorAssertionFunc
		errorContains string
	}{
		{
			name: "no op single file",
			args: args{
				paths: []string{"set-test.tmpl"},
			},
			wantErr: assert.NoError,
		},
		{
			name: "merge 2 data",
			args: args{
				paths: []string{"set-test.tmpl", "yamlOpts.tmpl"},
			},
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
			wantErr: assert.NoError,
		},
		{
			name: "duplicate data merged output",
			args: args{
				paths: []string{"set-test.tmpl", "yamlOpts.tmpl", "set-test.tmpl"},
			},
			wantErr: assert.NoError,
		},
		{
			name: "empty list",
			args: args{
				paths: []string{},
			},
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
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var paths []string
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

			outfile := path.Join("../../testFiles/tailmerge", strings.Replace(tt.name, " ", "_", -1)+".out")
			if _, err := os.Stat(outfile); err != nil {
				err = os.WriteFile(outfile, []byte(gotOut), 0o644)
				assert.Nil(t, err) //
			}
			compareFile, err := os.ReadFile(outfile)
			// Use this to update expected outputs when something changes
			assert.Equal(t, string(compareFile), gotOut, "TailMergeFiles(%v)", paths)

		})
	}
}
