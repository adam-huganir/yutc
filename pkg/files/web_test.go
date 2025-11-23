package files

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func Test_ReadURL(t *testing.T) {
	type args struct {
		templatePath string
	}
	tests := []struct {
		name         string
		args         args
		wantFilename string
		wantData     []byte
		wantMimetype string
		wantErr      error
	}{
		{
			name: "Get a template",
			args: args{
				templatePath: "https://raw.githubusercontent.com/adam-huganir/yutc/main/testFiles/templates/simpleTemplate.tmpl",
			},
			wantFilename: "simpleTemplate.tmpl",
			wantData:     []byte("JSON representation of the input:\n\n```json\n{{ . | toPrettyJson}}\n```\n\nor yaml\n\n```yaml\n{{ . | toYaml }}\n```\n"),
			wantMimetype: "text/plain",
			wantErr:      nil,
		},
		{
			name: "Test url 2",
			args: args{
				templatePath: "https://raw.githubusercontent.com/adam-huganir/yutc/main/testFiles/templates",
			},
			wantFilename: "templates",
			wantData:     []byte{0x34, 0x30, 0x34, 0x3a, 0x20, 0x4e, 0x6f, 0x74, 0x20, 0x46, 0x6f, 0x75, 0x6e, 0x64},
			wantMimetype: "text/plain",
			wantErr:      &HTTPStatusError{Status: "404 Not Found"},
		},
	}
	logger := zerolog.Nop()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename, data, mimetype, err := ReadURL(tt.args.templatePath, &logger)
			if !assert.IsType(t, tt.wantErr, err) {
				return
			}
			if err == nil {
				assert.Equalf(t, tt.wantFilename, filename, "ReadURL(%v)", tt.args.templatePath)
				assert.Equalf(t, tt.wantData, data, "ReadURL(%v)", tt.args.templatePath)
				assert.Equalf(t, tt.wantMimetype, mimetype, "ReadURL(%v)", tt.args.templatePath)
			} else {
				assert.Equalf(t, tt.wantErr.Error(), err.Error(), "ReadURL(%v)", tt.args.templatePath)
			}
		})
	}
}
