package internal

import (
	"bytes"
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
		config  *CLISettings
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Test url",
			args: args{
				"https://raw.githubusercontent.com/adam-huganir/yutc/main/testFiles/templates/simpleTemplate.tmpl",
				&bytes.Buffer{},
			},
			config: &CLISettings{
				DataFiles: []string{"./testFiles/data/data1.yaml"},
			},
			want:    "JSON representation of the input:\n\n```json\n{{ . | toPrettyJson}}\n```\n\nor yaml\n\n```yaml\n{{ . | toYaml }}\n```\n",
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RunSettings = tt.config
			wantBuff := bytes.NewBuffer([]byte(tt.want))
			got, err := getUrlFile(tt.args.arg, tt.args.buff)
			if !tt.wantErr(t, err, fmt.Sprintf("getUrlFile(%v, %v)", tt.args.arg, tt.args.buff)) {
				return
			}
			assert.Equalf(t, wantBuff, got, "getUrlFile(%v, %v)", tt.args.arg, tt.args.buff)
		})
	}
}
