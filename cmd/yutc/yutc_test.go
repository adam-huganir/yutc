package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/adam-huganir/yutc/pkg/files"
	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

var expectedOutputs = map[string]string{
	"data1Verbatim":   "map[dogs:[map[breed:Labrador name:Fido owner:map[name:John Doe] vaccinations:[rabies]]] thisWillMerge:map[value23:not 23 value24:24]]\n",
	"data2":           "Unmerged data from data 1: {\"dogs\":[{\"breed\":\"Labrador\",\"name\":\"Fido\",\"owner\":{\"name\":\"John Doe\"},\"vaccinations\":[\"rabies\"]}],\"thisWillMerge\":{\"value23\":\"not 23\",\"value24\":24}}\nUnmerged data from data 2: {\"ditto\":[\"woohooo\",\"yipeee\"],\"dogs\":[],\"thisIsNew\":1000,\"thisWillMerge\":{\"value23\":23}}\n",
	"dataYamlOptions": "just testing things\naLongString: |-\n    this is a long string that should be split into multiple lines.\n    it is long enough that we should wrap it.\n    this is a long string that should be split into multiple lines.\n    it is long enough that we should wrap it.\n    this is a long string that should be split into multiple lines.\naString: a:b\nanotherMap:\n    a: \"\"\nnestedMap:\n    a:\n    - b\n    - c\nsomeList:\n- 1\n- 2\n\naLongString: |-\n this is a long string that should be split into multiple lines.\n it is long enough that we should wrap it.\n this is a long string that should be split into multiple lines.\n it is long enough that we should wrap it.\n this is a long string that should be split into multiple lines.\naString: a:b\nanotherMap:\n a: \"\"\nnestedMap:\n a:\n - b\n - c\nsomeList:\n- 1\n- 2\n",
	"strictSuccess1":  "1 and <no value>",
	"include1":        "version: \"3.7\"\n\nservices:\n  my-service:\n    restart: always\n    env_file:\n    - common.env\n    image: 1234\n",
}

func newCmdTest(settings *types.Arguments, args []string) (*cobra.Command, context.Context) {
	runData := types.RunData{}
	cmd := newRootCommand(settings, &runData, &logger)
	cmd.SetArgs(args)
	initRoot(cmd, settings)

	ctx := context.Background()
	return cmd, ctx
}

func CaptureStdoutWithError(ctx context.Context, f func(context.Context) error) (bStdOut []byte, err error) {
	r, w, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = w

	outC := make(chan []byte)
	go func() { // don't block the pipes
		b, _ := io.ReadAll(r)
		outC <- b
	}()

	defer func() {
		os.Stdout = stdout
		_ = w.Close()
	}()

	err = f(ctx)
	_ = w.Close()

	bStdOut = <-outC
	return bStdOut, err
}

func TestBasicStdout(t *testing.T) {
	runTest(t, TestCase{
		Name: "Basic Stdout",
		Args: func(_ string) []string {
			return []string{
				"-d", "../../testFiles/data/data1.yaml",
				"-o", "-",
				"../../testFiles/templates/verbatim.tmpl",
			}
		},
		ExpectedStdout: expectedOutputs["data1Verbatim"],
	})
}

func TestStrict(t *testing.T) {
	runTest(t, TestCase{
		Name: "Strict Mode - Success",
		InputFiles: map[string]string{
			"data.yaml": "test:\n  data_1: 1",
			"tmpl.txt":  "{{ .test.data_1 }} and {{ .test.data_2 }}",
		},
		Args: func(rootDir string) []string {
			return []string{
				"-d", filepath.Join(rootDir, "data.yaml"),
				"-o", "-",
				filepath.Join(rootDir, "tmpl.txt"),
			}
		},
		ExpectedStdout: expectedOutputs["strictSuccess1"],
	})

	runTest(t, TestCase{
		Name: "Strict Mode - Failure",
		InputFiles: map[string]string{
			"data.yaml": "test:\n  data_1: 1",
			"tmpl.txt":  "{{ .test.data_1 }} and {{ .test.data_2 }}",
		},
		Args: func(rootDir string) []string {
			return []string{
				"-d", filepath.Join(rootDir, "data.yaml"),
				"-o", "-",
				"--strict",
				filepath.Join(rootDir, "tmpl.txt"),
			}
		},
		WantPanic: true,
	})
}

func TestInclude(t *testing.T) {
	runTest(t, TestCase{
		Name: "Include Function",
		Args: func(_ string) []string {
			return []string{
				"-c", "../../testFiles/functions/fn.tmpl",
				"-o", "-",
				"../../testFiles/functions/docker-compose.yaml.tmpl",
			}
		},
		ExpectedStdout: expectedOutputs["include1"],
	})
}

func TestBasicFile(t *testing.T) {
	runTest(t, TestCase{
		Name: "Basic File Output",
		Args: func(rootDir string) []string {
			return []string{
				"-d", "../../testFiles/data/data1.yaml",
				"-o", filepath.Join(rootDir, "output.go"),
				"../../testFiles/templates/verbatim.tmpl",
			}
		},
		ExpectedFiles: map[string]string{
			"output.go": expectedOutputs["data1Verbatim"],
		},
	})

	runTest(t, TestCase{
		Name: "File Exists Failure",
		InputFiles: map[string]string{
			"output.go": "existing content",
		},
		Args: func(rootDir string) []string {
			return []string{
				"-d", "../../testFiles/data/data1.yaml",
				"-o", filepath.Join(rootDir, "output.go"),
				"../../testFiles/templates/verbatim.tmpl",
			}
		},
		ExpectedError: "exists and `overwrite` is not set",
	})
}

func TestTopLevelKeys(t *testing.T) {
	runTest(t, TestCase{
		Name: "Top Level Keys",
		Args: func(rootDir string) []string {
			return []string{
				"-d", "key=data1,src=../../testFiles/data/data1.yaml",
				"-d", "key=data2,src=../../testFiles/data/data2.yaml",
				"-o", filepath.Join(rootDir, "output.go"),
				"../../testFiles/templates/templateWithKeys.tmpl",
			}
		},
		ExpectedFiles: map[string]string{
			"output.go": expectedOutputs["data2"],
		},
	})
}

func TestRecursiveFolderTree(t *testing.T) {
	inputDir := files.NormalizeFilepath("../../testFiles/poetry-init/from-dir")
	inputData := files.NormalizeFilepath("../../testFiles/poetry-init/data.yaml")

	runTest(t, TestCase{
		Name: "Recursive Folder Tree - No Template Filenames",
		Args: func(rootDir string) []string {
			return []string{
				"-d", inputData,
				"-o", rootDir,
				inputDir,
			}
		},
		Verify: func(t *testing.T, rootDir string) {
			verifyRecursiveFolderTree(t, inputDir, rootDir, false)
		},
	})

	runTest(t, TestCase{
		Name: "Recursive Folder Tree - With Template Filenames",
		Args: func(rootDir string) []string {
			return []string{
				"-d", inputData,
				"--include-filenames",
				"-o", rootDir,
				inputDir,
			}
		},
		Verify: func(t *testing.T, rootDir string) {
			verifyRecursiveFolderTree(t, inputDir, rootDir, true)
		},
	})
}

func verifyRecursiveFolderTree(t *testing.T, inputDir, outputDir string, templateFilename bool) {
	logger := zerolog.Nop()
	sourcePaths := files.WalkDir(inputDir, &logger)
	for i, sourcePath := range sourcePaths {
		sourcePaths[i] = strings.TrimPrefix(strings.TrimPrefix(sourcePath, inputDir), "/") // make relative
	}
	outputPaths := files.WalkDir(outputDir, &logger)
	for i, outputPath := range outputPaths {
		outputPaths[i] = strings.TrimPrefix(strings.TrimPrefix(outputPath, outputDir), "/") // make relative
	}
	slices.SortFunc(sourcePaths, files.CmpStringLength)
	slices.SortFunc(outputPaths, files.CmpStringLength)

	for i, sourcePath := range sourcePaths {
		if templateFilename && strings.Contains(sourcePath, "{{") {
			assert.NotEqual(t, sourcePath, outputPaths[i])
		} else {
			assert.Equal(t, sourcePath, outputPaths[i])
		}
	}
}

func TestYamlOptions(t *testing.T) {
	runTest(t, TestCase{
		Name: "Yaml Options",
		Args: func(rootDir string) []string {
			return []string{
				"-d", "../../testFiles/data/yamlOptions.yaml",
				"-o", filepath.Join(rootDir, "output.go"),
				"../../testFiles/yamlOpts.tmpl",
			}
		},
		ExpectedFiles: map[string]string{
			"output.go": expectedOutputs["dataYamlOptions"],
		},
	})
}

func TestYamlOptionsBad(t *testing.T) {
	runTest(t, TestCase{
		Name: "Yaml Options Bad",
		Args: func(rootDir string) []string {
			return []string{
				"-d", "../../testFiles/data/yamlOptionsBad.yaml",
				"-o", filepath.Join(rootDir, "output.go"),
				"../../testFiles/yamlOpts.tmpl",
			}
		},
		WantPanic:     true,
		ExpectedPanic: "indent must be an integer",
	})
}

func TestSetFeature(t *testing.T) {
	runTest(t, TestCase{
		Name: "Set Simple String",
		Args: func(_ string) []string {
			return []string{
				"--set", "$.foo=hello",
				"-o", "-",
				"../../testFiles/set-test.tmpl",
			}
		},
		ExpectedStdout: "hello\n<no value>\n<no value>\n<no value>\n<no value>\n<no value>\n",
	})

	runTest(t, TestCase{
		Name: "Set Nested Value",
		Args: func(_ string) []string {
			return []string{
				"--set", "$.bar.baz=world",
				"-o", "-",
				"../../testFiles/set-test.tmpl",
			}
		},
		ExpectedStdout: "<no value>\nworld\n<no value>\n<no value>\n<no value>\n<no value>\n",
	})

	runTest(t, TestCase{
		Name: "Set Number",
		Args: func(_ string) []string {
			return []string{
				"--set", "$.num=42",
				"-o", "-",
				"../../testFiles/set-test.tmpl",
			}
		},
		ExpectedStdout: "<no value>\n<no value>\n<no value>\n<no value>\n42\n<no value>\n",
	})

	runTest(t, TestCase{
		Name: "Set Boolean",
		Args: func(_ string) []string {
			return []string{
				"--set", "$.bool=true",
				"-o", "-",
				"../../testFiles/set-test.tmpl",
			}
		},
		ExpectedStdout: "<no value>\n<no value>\n<no value>\n<no value>\n<no value>\ntrue\n",
	})

	runTest(t, TestCase{
		Name: "Set Array Values",
		Args: func(_ string) []string {
			return []string{
				"--set", `$.arr=["first","second"]`,
				"-o", "-",
				"../../testFiles/set-test.tmpl",
			}
		},
		ExpectedStdout: "<no value>\n<no value>\nfirst\nsecond\n<no value>\n<no value>\n",
	})

	runTest(t, TestCase{
		Name: "Set Multiple Values",
		Args: func(_ string) []string {
			return []string{
				"--set", "$.foo=test",
				"--set", "$.bar.baz=nested",
				"--set", "$.num=123",
				"-o", "-",
				"../../testFiles/set-test.tmpl",
			}
		},
		ExpectedStdout: "test\nnested\n<no value>\n<no value>\n123\n<no value>\n",
	})

	runTest(t, TestCase{
		Name: "Set With Data File",
		Args: func(_ string) []string {
			return []string{
				"-d", "../../testFiles/data/data1.yaml",
				"--set", "$.dogs[0].name=Buddy",
				"-o", "-",
				"../../testFiles/templates/verbatim.tmpl",
			}
		},
		ExpectedStdout: "map[dogs:[map[breed:Labrador name:Buddy owner:map[name:John Doe] vaccinations:[rabies]]] thisWillMerge:map[value23:not 23 value24:24]]\n",
	})

	// Test convenience feature: auto-prefix $ for paths starting with . or [
	runTest(t, TestCase{
		Name: "Set With Dot Prefix",
		Args: func(_ string) []string {
			return []string{
				"--set", ".foo=convenience",
				"-o", "-",
				"../../testFiles/set-test.tmpl",
			}
		},
		ExpectedStdout: "convenience\n<no value>\n<no value>\n<no value>\n<no value>\n<no value>\n",
	})

	runTest(t, TestCase{
		Name: "Set With Nested Dot Prefix",
		Args: func(_ string) []string {
			return []string{
				"--set", ".bar.baz=dottest",
				"-o", "-",
				"../../testFiles/set-test.tmpl",
			}
		},
		ExpectedStdout: "<no value>\ndottest\n<no value>\n<no value>\n<no value>\n<no value>\n",
	})
}

type TestCase struct {
	Name           string
	Args           func(rootDir string) []string
	InputFiles     map[string]string // filename (relative to rootDir) -> content
	ExpectedStdout string
	ExpectedFiles  map[string]string // filename (relative to rootDir) -> content
	ExpectedError  string            // substring match
	WantPanic      bool
	ExpectedPanic  string // substring match
	Verify         func(t *testing.T, rootDir string)
}

func runTest(t *testing.T, tc TestCase) {
	t.Run(tc.Name, func(t *testing.T) {
		rootDir := files.NormalizeFilepath(getTempDir(false))
		defer func() { _ = os.RemoveAll(rootDir) }()

		for filename, content := range tc.InputFiles {
			fullPath := filepath.Join(rootDir, filename)
			err := os.MkdirAll(filepath.Dir(fullPath), 0755)
			assert.NoError(t, err)
			err = os.WriteFile(fullPath, []byte(content), 0644)
			assert.NoError(t, err)
		}

		var args []string
		if tc.Args != nil {
			args = tc.Args(rootDir)
		}

		cmd, ctx := newCmdTest(&types.Arguments{}, args)

		var bStdOut []byte
		var err error

		if tc.WantPanic {
			defer func() {
				r := recover()
				if r == nil {
					t.Errorf("expected panic but did not panic")
				} else if tc.ExpectedPanic != "" {
					assert.Contains(t, fmt.Sprintf("%v", r), tc.ExpectedPanic)
				}
			}()
			_, _ = CaptureStdoutWithError(ctx, cmd.ExecuteContext)
			return
		}

		if tc.ExpectedStdout == "" && !tc.WantPanic {
			err = cmd.ExecuteContext(ctx)
		} else {
			bStdOut, err = CaptureStdoutWithError(ctx, cmd.ExecuteContext)
		}
		stdOut := string(bStdOut)

		if tc.WantPanic {
			t.Errorf("expected panic but did not panic")
		}

		if tc.ExpectedError != "" {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.ExpectedError)
		} else if err != nil {
			t.Errorf("Command failed: %v", err)
		}

		if tc.ExpectedStdout != "" {
			normalizedStdOut := strings.ReplaceAll(stdOut, "\r\n", "\n")
			assert.Equal(t, tc.ExpectedStdout, normalizedStdOut)
		}

		for filename, expectedContent := range tc.ExpectedFiles {
			fullPath := filepath.Join(rootDir, filename)
			content, err := os.ReadFile(fullPath)
			assert.NoError(t, err)
			normalizedContent := strings.ReplaceAll(string(content), "\r\n", "\n")
			assert.Equal(t, expectedContent, normalizedContent)
		}

		if tc.Verify != nil {
			tc.Verify(t, rootDir)
		}
	})
}

func Must(result any, err error) any {
	if err != nil {
		panic(err)
	}
	return result
}

func getTempDir(deleteOnCreate bool) string {
	tempDir := Must(os.MkdirTemp("", "yutc-test-*")).(string)
	if deleteOnCreate {
		_ = os.RemoveAll(tempDir)
	}
	return tempDir
}
