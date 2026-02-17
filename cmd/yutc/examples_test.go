package main

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/adam-huganir/yutc/pkg/loader"
	"github.com/adam-huganir/yutc/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestUvPythonExample(t *testing.T) {
	expected := util.MustDedent(`
	==> ../../examples/uv-python-project/build/my-python-project/__init__.py <==
	# my-python-project

	__version__ = "0.1.0"

	==> ../../examples/uv-python-project/build/pyproject.toml <==
	[project]
	name = "my-python-project"
	version = "0.1.0"
	description = "A sample project generated with yutc"
	authors = [
	    { name = "adam", email = "you@example.com" }
	]
	dependencies = [
	    "fastapi",
	]
	requires-python = ">=3.8"
	readme = "README.md"
	license = { text = "MIT" }

	[build-system]
	requires = ["hatchling"]
	build-backend = "hatchling.build"

	[tool.rye]
	managed = true
	dev-dependencies = []

	[tool.hatch.metadata]
	allow-direct-references = true
	`)

	rootDir := "../../examples/uv-python-project"
	buildDir := path.Join(rootDir, "build")
	err := os.RemoveAll(buildDir) // delete previous build output if it exists
	if err != nil {
		t.Fatalf("failed to remove previous build dir: %v", err)
		return
	}
	err = os.Mkdir(buildDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create build dir: %v", err)
		return
	}
	defer func(path string) { _ = os.RemoveAll(path) }(buildDir)

	runTest(t, &TestCase{
		Name: "Build UV python example",
		Args: func(_ string) []string {
			return []string{
				"-d", path.Join(rootDir, "data.yaml"),
				"-o", buildDir,
				"--overwrite",
				"--include-filenames",
				path.Join(rootDir, "src"),
			}
		},
		Verify: func(t *testing.T, _ string) {
			output, err := tailMergeDir(buildDir)
			assert.NoError(t, err, "failed to merge build output data")
			assert.Equal(t, expected, output, fmt.Sprintf("merged build output did not match expected:\n%s", output))
		},
	})
}

func tailMergeDir(buildDir string) (string, error) {
	var f []string
	err := fs.WalkDir(os.DirFS("../.."), strings.TrimPrefix(buildDir, "../../"), func(fpath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			f = append(f, path.Join("../../", fpath))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return loader.TailMergeFiles(f)
}
