package internal

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_getUrlFile(t *testing.T) {
	type args struct {
		arg  string
		buff *bytes.Buffer
	}
	tests := []struct {
		name    string
		args    args
		config  *YutcSettings
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Test url",
			args: args{
				"https://raw.githubusercontent.com/adam-huganir/yutc/main/testFiles/templates/simpleTemplate.tmpl",
				&bytes.Buffer{},
			},
			config: &YutcSettings{
				DataFiles: []string{"./testFiles/data/data1.yaml"},
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
	dummySettings := &YutcSettings{}

	// test file that does not exist
	f := "testinggggg"
	_, err := GetDataFromPath("file", f, dummySettings)
	assert.Equal(t, errors.New(fmt.Sprintf("file does not exist: %s", f)), err)

	// test file that does exist
	f = "../testFiles/data/data1.yaml"
	buffer, err = GetDataFromPath("file", f, dummySettings)
	assert.NoError(t, err)
	assert.Equal(t,
		"dogs:\n  - name: Fido\n    breed: Labrador\n    vaccinations:\n      - rabies\n    owner:\n      name: John Doe\nthisWillMerge:\n  value23: \"not 23\"\n  value24: 24\n",
		buffer.String(),
	)

	// test url
	f = "https://raw.githubusercontent.com/adam-huganir/yutc/main/testFiles/data/data1.yaml"
	buffer2, err = GetDataFromPath("url", f, dummySettings)
	assert.NoError(t, err)
	assert.Equal(t, buffer.String(), buffer2.String())
}

func TestCheckIfDir(t *testing.T) {
	isDir, _ := CheckIfDir("../testFiles/data")
	assert.Equal(t, true, *isDir)
	isDir, _ = CheckIfDir("../testFiles/data/data1.yaml")
	assert.Equal(t, false, *isDir)
	_, err := CheckIfDir("../testFiles/NotAFile")
	assert.ErrorContains(t, err, "The system cannot find the file specified")
}

func TestCheckIsFile(t *testing.T) {
	isFile, _ := CheckIsFile("../testFiles/data/data1.yaml")
	assert.Equal(t, true, *isFile)
	isFile, _ = CheckIsFile("../testFiles/data")
	assert.Equal(t, false, *isFile)
	_, err := CheckIsFile("../testFiles/NotAFile")
	assert.ErrorContains(t, err, "The system cannot find the file specified")
}
