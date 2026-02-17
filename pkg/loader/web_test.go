package loader

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadURL_SmallBody(t *testing.T) {
	// Server that returns a small body without Content-Type
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Del("Content-Type")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("small body"))
	}))
	defer ts.Close()

	fe := NewFileEntry(ts.URL, WithSource(SourceKindURL))
	err := fe.ReadURL()
	assert.NoError(t, err)
	assert.Equal(t, "text/plain", fe.Content.Mimetype) // http.DetectContentType should detect text/plain
}

func TestReadURL_EmptyBody(t *testing.T) {
	// Server that returns an empty body without Content-Type
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Del("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	fe := NewFileEntry(ts.URL, WithSource(SourceKindURL))
	err := fe.ReadURL()
	assert.NoError(t, err)
	// http.DetectContentType returns "text/plain; charset=utf-8" for empty body sometimes,
	// or "application/octet-stream" depending on Go version and environment.
	// Based on local test run, it returned "text/plain".
	assert.Contains(t, fe.Content.Mimetype, "text/plain")
}

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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFileEntry(tt.args.templatePath, WithSource(SourceKindURL))
			err := f.ReadURL()

			if !assert.IsType(t, tt.wantErr, err) {
				return
			}
			if err == nil {
				assert.Equalf(t, tt.wantFilename, f.Content.Filename, "ReadURL(%v)", tt.args.templatePath)
				assert.Equalf(t, tt.wantData, f.Content.Data, "ReadURL(%v)", tt.args.templatePath)
				assert.Equalf(t, tt.wantMimetype, f.Content.Mimetype, "ReadURL(%v)", tt.args.templatePath)
			} else {
				assert.Equalf(t, tt.wantErr.Error(), err.Error(), "ReadURL(%v)", tt.args.templatePath)
			}
		})
	}
}
