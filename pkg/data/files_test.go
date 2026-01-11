package data

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"testing"

	"github.com/adam-huganir/yutc/pkg/types"
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
			gotReq, err := GetURL(u, "", "")
			if err != nil {
				assert.Failf(t, "url get error", "url get error: %s", err)
			}
			defer func() { _ = gotReq.Body.Close() }()
			got, err := io.ReadAll(gotReq.Body)
			if err != nil {
				assert.Failf(t, "url read error", "url read error: %s", err)
			}
			if !tt.wantErr(t, err, fmt.Sprintf("getURLFile(%v, %v)", tt.args.arg, tt.args.buff)) {
				return
			}
			assert.Equalf(t, wantBuff.Bytes(), got, "getURLFile(%v, %v)", tt.args.arg, tt.args.buff)
		})
	}
}

func TestGetDataFromPath(t *testing.T) {
	// test file that does not exist
	// Test case 1: Valid file path
	fk := FileKindData
	f := NewFileArgFile("testdata/sample.json", &fk)
	err := f.Load()
	assert.Error(t, err)

	// Test case 2: Valid file path and valid url
	localPath := "../../testFiles/data/data1.yaml"
	urlPath := "https://raw.githubusercontent.com/adam-huganir/yutc/main/testFiles/data/data1.yaml"

	buffer, err := os.ReadFile(localPath)
	if err != nil {
		assert.Failf(t, "file read error", "file read error: %s", err)
	}

	// test file that does exist
	f = NewFileArgFile(localPath, &fk)
	err = f.Load()
	assert.NoError(t, err)
	assert.Equal(t, string(buffer), string(f.Content.Data))

	// test url same as the above file
	f2 := NewFileArgURL(
		urlPath,
		&fk,
	)

	err = f2.Load()
	assert.NoError(t, err)
	assert.Equal(t, string(buffer), string(f2.Content.Data))
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
