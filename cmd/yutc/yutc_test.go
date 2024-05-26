package main

import (
	"github.com/adam-huganir/yutc/internal"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"slices"
	"strings"
	"testing"
)

func newCmdTest(settings *internal.YutcSettings, args []string) *cobra.Command {
	cmd := newRootCommand()
	runSettings = settings
	initRoot(cmd, settings)
	cmd.SetArgs(args)
	return cmd
}

func Must(result any, err error) any {
	if err != nil {
		panic(err)
	}
	return result
}

func getTestTempfile(deleteFile bool, extension string) *os.File {
	tempfile, err := os.CreateTemp("", "yutc-test-*"+extension)
	if err != nil {
		panic(err)
	}
	if deleteFile {
		_ = tempfile.Close()
		err = os.Remove(tempfile.Name())
		if err != nil {
			panic(err)
		}
	}
	return tempfile
}

func getTempDir(delete bool) string {
	tempDir := Must(os.MkdirTemp("", "yutc-test-*")).(string)
	if delete {
		_ = os.RemoveAll(tempDir)
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
	tempfile := *getTestTempfile(true, ".go")
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
	_ = os.Remove(tempfile.Name())

	// test that if file exists we fail:
	tempfile = *getTestTempfile(false, ".go")
	// internal.InitLogger("trace")
	cmd = newCmdTest(&internal.YutcSettings{}, []string{
		"-d", "../../testFiles/data/data1.yaml",
		"-o", tempfile.Name(),
		"../../testFiles/templates/verbatim.tmpl",
	})
	_, err = CaptureStdoutWithError(cmd.Execute)
	assert.ErrorContains(t, err, "exists and `overwrite` is not set")
	_ = os.Remove(tempfile.Name())
}

func TestRecursiveFolderTree(t *testing.T) {
	var cmd *cobra.Command
	for _, templateFilename := range []bool{false, true} {
		tempdir := internal.NormalizeFilepath(getTempDir(false))
		YutcLog.Debug().Msg("tempdir: " + tempdir)
		// internal.InitLogger("trace")
		inputDir := internal.NormalizeFilepath("../../testFiles/poetry-init/from-dir")
		inputData := internal.NormalizeFilepath("../../testFiles/poetry-init/data.yaml")
		if templateFilename {
			cmd = newCmdTest(&internal.YutcSettings{}, []string{
				"-d", inputData,
				"--include-filenames",
				"-o", tempdir,
				inputDir,
			})
		} else {
			cmd = newCmdTest(&internal.YutcSettings{}, []string{
				"-d", inputData,
				"-o", tempdir,
				inputDir,
			})
		}
		currentDir, _ := os.Getwd()
		YutcLog.Debug().Msg("currentDir: " + currentDir)
		_, err := CaptureStdoutWithError(cmd.Execute)
		assert.NoError(t, err)
		assert.Equal(t, internal.ExitCodeMap["ok"], *internal.ExitCode)
		sourcePaths := internal.WalkDir(inputDir)
		for i, sourcePath := range sourcePaths {
			sourcePaths[i] = strings.TrimPrefix(strings.TrimPrefix(sourcePath, inputDir), "/") // make relative
		}
		outputPaths := internal.WalkDir(tempdir)
		for i, outputPath := range outputPaths {
			outputPaths[i] = strings.TrimPrefix(strings.TrimPrefix(outputPath, tempdir), "/") // make relative

		}
		slices.SortFunc(sourcePaths, internal.CmpStringLength)
		slices.SortFunc(outputPaths, internal.CmpStringLength)

		for i, sourcePath := range sourcePaths {
			if templateFilename && strings.Contains(sourcePath, "{{") {
				assert.NotEqual(t, sourcePath, outputPaths[i])
			} else {
				assert.Equal(t, sourcePath, outputPaths[i])
			}
		}
		_ = os.RemoveAll(tempdir)
	}
}
