package main

import (
	"github.com/adam-huganir/yutc/internal"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
)

func Must(result any, err error) any {
	if err != nil {
		panic(err)
	}
	return result
}

func getTestTempfile(deleteFile bool) *os.File {
	tempfile, err := os.CreateTemp("", "yutc-test-*.yaml")
	if err != nil {
		panic(err)
	}
	if deleteFile {
		_ = tempfile.Close()
		defer func() {
			_ = os.Remove(tempfile.Name())
		}()

	}
	return tempfile
}

func getTempDir(delete bool) string {
	tempDir := Must(os.MkdirTemp("", "yutc-test-*")).(string)
	if delete {
		defer func() {
			_ = os.RemoveAll(tempDir)
		}()
	}
	return tempDir
}

var data1Verbatim = "map[dogs:[map[breed:Labrador name:Fido owner:map[name:John Doe] vaccinations:[rabies]]] thisWillMerge:map[value23:not 23 value24:24]]\n"

func CaptureStdoutWithError(f func() error) (bStdOut []byte, err error) {
	var readErr error
	r, w, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = w
	err = f()
	_ = w.Close()
	os.Stdout = stdout
	bStdOut, readErr = io.ReadAll(r)
	defer func() { _ = r.Close() }()
	if readErr != nil {
		panic("fix me")
	}
	return bStdOut, err
}

func TestBasicStdout(t *testing.T) {
	println("Current working directory: ", Must(os.Getwd()).(string))

	// internal.InitLogger("trace")
	cmd := newCmdTest(&internal.YutcSettings{}, []string{
		"-d", "../../testFiles/data/data1.yaml",
		"-o", "-",
		"../../testFiles/templates/verbatim.tmpl",
	})
	bStdOut, err := CaptureStdoutWithError(cmd.Execute)
	stdOut := string(bStdOut)
	assert.NoError(t, err)
	assert.Equal(t, internal.ExitCodeMap["ok"], *internal.ExitCode)
	assert.Equal(
		t,
		data1Verbatim,
		stdOut,
	)
}

func TestBasicFile(t *testing.T) {
	tempfile := *getTestTempfile(true)
	// internal.InitLogger("trace")
	cmd := newCmdTest(&internal.YutcSettings{}, []string{
		"-d", "../../testFiles/data/data1.yaml",
		"-o", tempfile.Name(),
		"../../testFiles/templates/verbatim.tmpl",
	})
	_, err := CaptureStdoutWithError(cmd.Execute)
	assert.NoError(t, err)
	assert.Equal(t, internal.ExitCodeMap["ok"], *internal.ExitCode)
	output, err := os.ReadFile(tempfile.Name())
	assert.NoError(t, err)
	assert.Equal(t, data1Verbatim, string(output))

	// test that if file exists we fail:
	tempfile = *getTestTempfile(false)
	// internal.InitLogger("trace")
	cmd = newCmdTest(&internal.YutcSettings{}, []string{
		"-d", "../../testFiles/data/data1.yaml",
		"-o", tempfile.Name(),
		"../../testFiles/templates/verbatim.tmpl",
	})
	_, err = CaptureStdoutWithError(cmd.Execute)
	assert.ErrorContains(t, err, "exists and `overwrite` is not set")
}

func TestWIP(t *testing.T) {
	tempdir := getTempDir(true)
	YutcLog.Debug().Msg("tempdir: " + tempdir)
	// internal.InitLogger("trace")
	cmd := newCmdTest(&internal.YutcSettings{}, []string{
		"-d", "../../testFiles/poetry-init/data.yaml",
		"-o", tempdir,
		"../../testFiles/poetry-init/from-dir",
	})
	currentDir, _ := os.Getwd()
	YutcLog.Debug().Msg("currentDir: " + currentDir)
	_, err := CaptureStdoutWithError(cmd.Execute)
	assert.NoError(t, err)
	assert.Equal(t, internal.ExitCodeMap["ok"], *internal.ExitCode)
	output, err := os.ReadFile(tempdir)
	assert.NoError(t, err)
	assert.Equal(t, data1Verbatim, string(output))
}
