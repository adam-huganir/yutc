package main

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/adam-huganir/yutc/internal"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func newCmdTest(settings *internal.YutcSettings, args []string) *cobra.Command {
	cmd := newRootCommand()
	runSettings = settings
	initRoot(cmd, settings)
	cmd.SetArgs(args)
	runData = &internal.RunData{
		YutcSettings: settings,
	}
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
var data2 = "Unmerged data from data 1: {\"dogs\":[{\"breed\":\"Labrador\",\"name\":\"Fido\",\"owner\":{\"name\":\"John Doe\"},\"vaccinations\":[\"rabies\"]}],\"thisWillMerge\":{\"value23\":\"not 23\",\"value24\":24}}\nUnmerged data from data 2: {\"ditto\":[\"woohooo\",\"yipeee\"],\"dogs\":[],\"thisIsNew\":1000,\"thisWillMerge\":{\"value23\":23}}\n"
var dataYamlOptions = "just testing things\naLongString: |-\n    this is a long string that should be split into multiple lines.\n    it is long enough that we should wrap it.\n    this is a long string that should be split into multiple lines.\n    it is long enough that we should wrap it.\n    this is a long string that should be split into multiple lines.\naString: a:b\nanotherMap:\n    a: \"\"\nnestedMap:\n    a:\n    - b\n    - c\nsomeList:\n- 1\n- 2\n\n\naLongString: |-\n this is a long string that should be split into multiple lines.\n it is long enough that we should wrap it.\n this is a long string that should be split into multiple lines.\n it is long enough that we should wrap it.\n this is a long string that should be split into multiple lines.\naString: a:b\nanotherMap:\n a: \"\"\nnestedMap:\n a:\n - b\n - c\nsomeList:\n- 1\n- 2\n\n"

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

func RunTestShortHandArgsEquals(t *testing.T, args []string, expected string) {
	// internal.InitLogger("trace")
	cmd := newCmdTest(&internal.YutcSettings{}, args)
	bStdOut, err := CaptureStdoutWithError(cmd.Execute)
	stdOut := string(bStdOut)
	assert.NoError(t, err)
	assert.Equal(t, internal.ExitCodeMap["ok"], *internal.ExitCode)
	assert.Equal(
		t,
		expected,
		stdOut,
	)
}

func TestBasicStdout(t *testing.T) {
	RunTestShortHandArgsEquals(t, []string{
		"-d", "../../testFiles/data/data1.yaml",
		"-o", "-",
		"../../testFiles/templates/verbatim.tmpl",
	}, data1Verbatim)
}

func TestStrict(t *testing.T) {
	tempData := *getTestTempfile(false, ".yaml")
	defer tempData.Close()
	data := "test:\n  data_1: 1"
	_, err := tempData.Write([]byte(data))

	tempTemplate1 := *getTestTempfile(false, ".txt")
	defer tempTemplate1.Close()
	template := "{{ .test.data_1 }} and {{ .test.data_2 }}"
	_, err = tempTemplate1.Write([]byte(template))
	assert.NoError(t, err)

	cmd := newCmdTest(&internal.YutcSettings{}, []string{
		"-d", tempData.Name(),
		"-o", "-",
		tempTemplate1.Name(),
	})
	bStdOut, err := CaptureStdoutWithError(cmd.Execute)
	stdOut := string(bStdOut)
	assert.NoError(t, err)
	assert.Equal(t, internal.ExitCodeMap["ok"], *internal.ExitCode)
	assert.Equal(
		t,
		"1 and <no value>",
		stdOut,
	)

	cmd = newCmdTest(&internal.YutcSettings{}, []string{
		"-d", tempData.Name(),
		"-o", "-",
		"--strict",
		tempTemplate1.Name(),
	})
	assert.Panics(t, func() {
		_ = cmd.Execute()
	})
}

func TestInclude(t *testing.T) {
	RunTestShortHandArgsEquals(t, []string{
		"-c", "../../testFiles/functions/fn.tmpl",
		"-o", "-",
		"../../testFiles/functions/docker-compose.yaml.tmpl",
	}, "version: \"3.7\"\n\nservices:\n  my-service:\n    restart: always\n    env_file:\n    - common.env\n    image: 1234\n")
}

func TestSet(t *testing.T) {
	tests := [][]string{
		{"$.thisWillMerge = {\"hello\":[1,2,[1,2,3]]}", "map[dogs:[map[breed:Labrador name:Fido owner:map[name:John Doe] vaccinations:[rabies]]] thisWillMerge:map[hello:[1 2 [1 2 3]]]]\n"},
		{"$.thisWillMerge.value23 = 23", "map[dogs:[map[breed:Labrador name:Fido owner:map[name:John Doe] vaccinations:[rabies]]] thisWillMerge:map[value23:23 value24:24]]\n"},
	}
	for _, tt := range tests {
		set_string := tt[0]
		expected := tt[1]
		RunTestShortHandArgsEquals(t, []string{
			"-d", "../../testFiles/data/data1.yaml",
			"-o", "-",
			"../../testFiles/templates/template2.tmpl",
			"--set", set_string,
		}, expected)
	}
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

func TestTopLevelKeys(t *testing.T) {
	tempfile := *getTestTempfile(true, ".go")
	// internal.InitLogger("trace")
	cmd := newCmdTest(&internal.YutcSettings{}, []string{
		"-d", "key=data1,src=../../testFiles/data/data1.yaml",
		"-d", "key=data2,src=../../testFiles/data/data2.yaml",
		"-o", tempfile.Name(),
		"../../testFiles/templates/templateWithKeys.tmpl",
	})
	_, err := CaptureStdoutWithError(cmd.Execute)
	assert.NoError(t, err)
	assert.Equal(t, internal.ExitCodeMap["ok"], *internal.ExitCode)
	output, err := os.ReadFile(tempfile.Name())
	assert.NoError(t, err)
	assert.Equal(t, data2, string(output))
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

func TestYamlOptions(t *testing.T) {
	tempfile := *getTestTempfile(true, ".go")
	// internal.InitLogger("trace")
	cmd := newCmdTest(&internal.YutcSettings{}, []string{
		"-d", "../../testFiles/data/yamlOptions.yaml",
		"-o", tempfile.Name(),
		"../../testFiles/yamlOpts.tmpl",
	})
	_, err := CaptureStdoutWithError(cmd.Execute)
	assert.NoError(t, err)
	assert.Equal(t, internal.ExitCodeMap["ok"], *internal.ExitCode)
	output, err := os.ReadFile(tempfile.Name())
	assert.NoError(t, err)
	assert.Equal(t, dataYamlOptions, string(output))
	_ = os.Remove(tempfile.Name())
}

func TestYamlOptionsBad(t *testing.T) {
	tempfile := *getTestTempfile(true, ".go")
	// internal.InitLogger("trace")

	// we expect a panic here, so gotta check
	defer func() {
		if r := recover(); r != nil {
			// Verify the panic message contains expected text
			panicMsg := fmt.Sprintf("%v", r)
			assert.Contains(t, panicMsg, "indent must be an integer")
		}
	}()
	cmd := newCmdTest(&internal.YutcSettings{}, []string{
		"-d", "../../testFiles/data/yamlOptionsBad.yaml",
		"-o", tempfile.Name(),
		"../../testFiles/yamlOpts.tmpl",
	})
	_, err := CaptureStdoutWithError(cmd.Execute)
	assert.Error(t, err)
	_ = os.Remove(tempfile.Name())
}
