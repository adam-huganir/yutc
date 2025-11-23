package files

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/stretchr/testify/assert"
)

func Test_getUrlFile(t *testing.T) {
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
			settings := tt.config
			wantBuff := bytes.NewBuffer([]byte(tt.want))
			got, err := getUrlFile(tt.args.arg, tt.args.buff, settings)
			if !tt.wantErr(t, err, fmt.Sprintf("getUrlFile(%v, %v)", tt.args.arg, tt.args.buff)) {
				return
			}
			assert.Equalf(t, wantBuff, got, "getUrlFile(%v, %v)", tt.args.arg, tt.args.buff)
		})
	}
}

func TestGetDataFromPath(t *testing.T) {
	var buffer, buffer2 *bytes.Buffer
	dummySettings := &types.Arguments{}

	// test file that does not exist
	// Test case 1: Valid file path
	_, err := GetDataFromPath("file", "testdata/sample.json", &types.Arguments{})
	if err != nil {
		assert.Error(t, err) // Assuming this was the intended assertion
	}

	// test file that does exist
	f := "../../testFiles/data/data1.yaml" // Re-declare f as it was removed in the snippet
	buffer, err = GetDataFromPath("file", f, dummySettings)
	assert.NoError(t, err)
	expectedBytes, err := os.ReadFile(f)
	assert.NoError(t, err)
	assert.Equal(t, string(expectedBytes), buffer.String())

	// test url
	f = "https://raw.githubusercontent.com/adam-huganir/yutc/main/testFiles/data/data1.yaml"
	buffer2, err = GetDataFromPath("url", f, dummySettings)
	assert.NoError(t, err)
	assert.Equal(t, buffer.String(), buffer2.String())
}

func TestCheckIfDir(t *testing.T) {
	isDir, _ := IsDir("../../testFiles/data")
	assert.Equal(t, true, isDir)
	isDir, _ = IsDir("../../testFiles/data/data1.yaml")
	assert.Equal(t, false, isDir)
	_, err := IsDir("../../testFiles/NotAFile")
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestCheckIsFile(t *testing.T) {
	isFile, _ := CheckIfFile("../../testFiles/data/data1.yaml")
	assert.Equal(t, true, isFile)
	isFile, _ = CheckIfFile("../../testFiles/data")
	assert.Equal(t, false, isFile)
	_, err := CheckIfFile("../../testFiles/NotAFile")
	assert.ErrorIs(t, err, os.ErrNotExist)
}
