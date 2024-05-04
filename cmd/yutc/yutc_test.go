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

func TestBasic(t *testing.T) {
	err := os.Chdir("../../")
	assert.NoError(t, err)

	println("Current working directory: ", Must(os.Getwd()).(string))

	// internal.InitLogger("trace")
	cmd := newCmdTest(&internal.YutcSettings{}, []string{
		"-d", "./testFiles/data/data1.yaml",
		"-o", "-",
		"./testFiles/templates/verbatim.tmpl",
	})
	bStdOut, err := CaptureStdoutWithError(cmd.Execute)
	stdOut := string(bStdOut)
	assert.NoError(t, err)
	assert.Equal(t, internal.ExitCodeMap["ok"], *internal.ExitCode)
	assert.Equal(
		t,
		"map[dogs:[map[breed:Labrador name:Fido owner:map[name:John Doe] vaccinations:[rabies]]] thisWillMerge:map[value23:not 23 value24:24]]\n",
		stdOut,
	)
}
