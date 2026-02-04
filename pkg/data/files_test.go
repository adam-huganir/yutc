package data

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"text/template"

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
			srv := newAuthFileServer(t, testAuthConfig{bearerToken: "secret"})
			wantBuff := bytes.NewBuffer([]byte(tt.want))
			u, err := url.Parse(srv.URL + "/templates/simpleTemplate.tmpl")
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
	f := NewFileArgFile("testdata/sample.json", FileKindData)
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
	f = NewFileArgFile(localPath, FileKindData)
	err = f.Load()
	assert.NoError(t, err)
	assert.Equal(t, string(buffer), string(f.Content.Data))

	// test url same as the above file
	f2 := NewFileArgURL(
		urlPath,
		FileKindData,
	)
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

func TestTemplateFilenames(t *testing.T) {
	tmpl, err := template.New("test").Parse("{{ .project_name }}")
	assert.NoError(t, err)

	fa := NewFileArgWithContent("{{ .project_name }}/init.py", FileKindTemplate, "file", []byte("content"))
	fas := []*FileArg{fa}

	data := map[string]any{"project_name": "my-project"}
	err = TemplateFilenames(fas, tmpl, data)
	assert.NoError(t, err)
	assert.Equal(t, "my-project/init.py", fa.Template.NewName)
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

func TestCountRecursables(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0o755)
	assert.NoError(t, err)

	file1 := filepath.Join(tempDir, "file1.txt")
	err = os.WriteFile(file1, []byte("content"), 0o644)
	assert.NoError(t, err)

	faDir := NewFileArgFile(subDir, FileKindData)
	faFile := NewFileArgFile(file1, FileKindData)

	count, err := CountRecursables([]*FileArg{&faDir, &faFile})
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// Test URL archive (mocked by extension)
	faURL := NewFileArgURL("http://example.com/test.zip", FileKindData)
	count, err = CountRecursables([]*FileArg{&faURL})
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestResolvePaths_Complex(t *testing.T) {
	tempDir := t.TempDir()

	// Single file
	file1 := filepath.Join(tempDir, "file1.yaml")
	err := os.WriteFile(file1, []byte("key: value"), 0o644)
	assert.NoError(t, err)

	outFiles, err := ResolvePaths([]string{file1}, FileKindData, tempDir, nil)
	assert.NoError(t, err)
	assert.Len(t, outFiles, 1)

	// Directory
	subDir := filepath.Join(tempDir, "mysubdir")
	err = os.Mkdir(subDir, 0o755)
	assert.NoError(t, err)
	file2 := filepath.Join(subDir, "file2.yaml")
	err = os.WriteFile(file2, []byte("key2: value2"), 0o644)
	assert.NoError(t, err)

	outFiles, err = ResolvePaths([]string{subDir}, FileKindData, tempDir, nil)
	assert.NoError(t, err)
	assert.True(t, len(outFiles) >= 1)

	// Error path: non-existent file
	_, err = ResolvePaths([]string{filepath.Join(tempDir, "nonexistent.yaml")}, FileKindData, tempDir, nil)
	assert.Error(t, err)
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

func TestTemplateFilenames_Error(t *testing.T) {
	tmpl := template.Must(template.New("test").Parse("{{ .project_name }}"))
	faInvalid := NewFileArgWithContent("{{ .Unclosed", FileKindTemplate, "file", []byte("content"))
	err := TemplateFilenames([]*FileArg{faInvalid}, tmpl, nil)
	assert.Error(t, err)
}

func TestGetDataFromReadCloser_Error(t *testing.T) {
	// Custom reader that returns an error
	errReader := &errorReader{}
	_, err := GetDataFromReadCloser(io.NopCloser(errReader))
	assert.Error(t, err)
}

type errorReader struct{}

func (e *errorReader) Read(_ []byte) (n int, err error) {
	return 0, fmt.Errorf("read error")
}

func TestMakeDirExist_Error(t *testing.T) {
	tempFile, err := os.CreateTemp("", "mkdir-error-test")
	assert.NoError(t, err)
	_ = os.Remove(tempFile.Name())
}
