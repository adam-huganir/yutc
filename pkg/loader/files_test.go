package loader

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getURLFile(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "Test url",
			url:     "/templates/simpleTemplate.tmpl",
			want:    "JSON representation of the input:\n\n```json\n{{ . | toPrettyJson}}\n```\n\nor yaml\n\n```yaml\n{{ . | toYaml }}\n```\n",
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newAuthFileServer(t, testAuthConfig{bearerToken: "secret"})
			wantBuff := bytes.NewBuffer([]byte(tt.want))
			u, err := url.Parse(srv.URL + tt.url)
			if err != nil {
				assert.Failf(t, "url parse error", "url parse error: %s", err)
			}
			gotReq, err := GetURL(u, "", "secret")
			if err != nil {
				assert.Failf(t, "url get error", "url get error: %s", err)
			}
			defer func() { _ = gotReq.Body.Close() }()
			got, err := io.ReadAll(gotReq.Body)
			if err != nil {
				assert.Failf(t, "url read error", "url read error: %s", err)
			}
			if !tt.wantErr(t, err, fmt.Sprintf("getURLFile(%v)", tt.url)) {
				return
			}
			assert.Equalf(t, wantBuff.Bytes(), got, "getURLFile(%v)", tt.url)
		})
	}
}

func TestGetDataFromPath(t *testing.T) {
	// test file that does not exist
	f := NewFileEntry("testdata/sample.json")
	err := f.Load()
	assert.Error(t, err)

	// Test case 2: Valid file path and valid url
	localPath := "../../testFiles/data/data1.yaml"
	srv := newAuthFileServer(t, testAuthConfig{bearerToken: "secret"})
	urlPath := srv.URL + "/data/data1.yaml"

	buffer, err := os.ReadFile(localPath)
	if err != nil {
		assert.Failf(t, "file read error", "file read error: %s", err)
	}

	// test file that does exist
	f = NewFileEntry(localPath)
	err = f.Load()
	assert.NoError(t, err)
	assert.Equal(t, string(buffer), string(f.Content.Data))

	// test url same as the above file
	f2 := NewFileEntry(urlPath, WithSource(SourceKindURL))
	f2.Auth.BearerToken = "secret"

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
	isFile, err := IsFile("../../testFiles/data/data1.yaml")
	assert.NoError(t, err)
	assert.Equal(t, true, isFile)
	isFile, err = IsFile("../../testFiles/data")
	assert.NoError(t, err)
	assert.Equal(t, false, isFile)
	_, err = IsFile("../../testFiles/NotAFile")
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestGenerateTempDirName(t *testing.T) {
	name, err := GenerateTempDirName("test-*")
	assert.NoError(t, err)
	assert.Contains(t, name, "test-")
}

func TestExists(t *testing.T) {
	tempFile, err := os.CreateTemp("", "exists-test")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	exists, err := Exists(tempFile.Name())
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = Exists(tempFile.Name() + "nonexistent")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestGetDataFromReadCloser(t *testing.T) {
	content := "hello world"
	rc := io.NopCloser(bytes.NewBufferString(content))
	buf, err := GetDataFromReadCloser(rc)
	assert.NoError(t, err)
	assert.Equal(t, content, buf.String())
}

func TestFiles_ErrorPaths(t *testing.T) {
	// Test GenerateTempDirName error path (invalid pattern)
	_, err := GenerateTempDirName("invalid/pattern")
	assert.Error(t, err)

	// Test IsDir error path (non-existent)
	_, err = IsDir("/non/existent/path/that/should/never/exist")
	assert.Error(t, err)

	// Test IsFile error path (non-existent)
	_, err = IsFile("/non/existent/path/that/should/never/exist")
	assert.Error(t, err)
}

func TestGetDataFromReadCloser_Error(t *testing.T) {
	errReader := &errorReader{}
	_, err := GetDataFromReadCloser(io.NopCloser(errReader))
	assert.Error(t, err)
}

type errorReader struct{}

func (e *errorReader) Read(_ []byte) (n int, err error) {
	return 0, fmt.Errorf("read error")
}
