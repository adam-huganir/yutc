package data

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func Test_getURLFile(t *testing.T) {
	type args struct {
		arg  string
		buff *bytes.Buffer
	}
	tests := []struct {
		name    string
		args    args
		config  *types.Arguments
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Test url",
			args: args{
				"https://raw.githubusercontent.com/adam-huganir/yutc/main/testFiles/templates/simpleTemplate.tmpl",
				&bytes.Buffer{},
			},
			config: &types.Arguments{
				DataFiles: []string{"../../testFiles/data/data1.yaml"},
			},
			want:    "JSON representation of the input:\n\n```json\n{{ . | toPrettyJson}}\n```\n\nor yaml\n\n```yaml\n{{ . | toYaml }}\n```\n",
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantBuff := bytes.NewBuffer([]byte(tt.want))
			u, err := url.Parse(tt.args.arg)
			if err != nil {
				assert.Failf(t, "url parse error", "url parse error: %s", err)
			}
			got, err := GetURL(u, "", "")
			if !tt.wantErr(t, err, fmt.Sprintf("getURLFile(%v, %v)", tt.args.arg, tt.args.buff)) {
				return
			}
			assert.Equalf(t, wantBuff, got, "getURLFile(%v, %v)", tt.args.arg, tt.args.buff)
		})
	}
}

func TestGetDataFromPath(t *testing.T) {
	var buffer, buffer2 *bytes.Buffer
	l := zerolog.Nop()
	// test file that does not exist
	// Test case 1: Valid file path
	f := NewFileArgFile("testdata/sample.json", "data")
	err := f.Load(&l)
	assert.Error(t, err) 

	// test file that does exist
	f = NewFileArgFile("../../testFiles/data/data1.yaml", "data")
	err = f.Load(&l)
	assert.NoError(t, err)
	assert.Equal(t, string(f.Content.Data), buffer.String())

	// test url
	f = NewFileArgURL(
		"https://raw.githubusercontent.com/adam-huganir/yutc/main/testFiles/data/data1.yaml",
		"data",
	)

	err = f.Load(&l)
	assert.NoError(t, err)
	assert.Equal(t, buffer.String(), buffer2.String())
}

func TestCheckIfDir(t *testing.T) {
	isDir, err := IsDir("../../testFiles/data")
	assert.NoError(t, err)
	assert.Equal(t, true, isDir)
	isDir, err = IsDir("../../testFiles/data/data1.yaml")
	assert.NoError(t, err)
	assert.Equal(t, false, isDir)
	_, err = IsDir("../../testFiles/NotAFile")
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestCheckIsFile(t *testing.T) {
	isFile, err := CheckIfFile("../../testFiles/data/data1.yaml")
	assert.NoError(t, err)
	assert.Equal(t, true, isFile)
	isFile, err = CheckIfFile("../../testFiles/data")
	assert.NoError(t, err)
	assert.Equal(t, false, isFile)
	_, err = CheckIfFile("../../testFiles/NotAFile")
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestGenerateTempDirName(t *testing.T) {
	name, err := GenerateTempDirName("test-*")
	assert.NoError(t, err)
	assert.Contains(t, name, "test-")
}

//func TestCountRecursables(t *testing.T) {
//	count, err := CountRecursables([]string{"../../testFiles/data", "../../testFiles/data/data1.yaml"})
//	assert.NoError(t, err)
//	assert.Equal(t, 1, count)
//
//	count, err = CountRecursables([]string{"../../testFiles/data/data1.yaml"})
//	assert.NoError(t, err)
//	assert.Equal(t, 0, count)
//}
