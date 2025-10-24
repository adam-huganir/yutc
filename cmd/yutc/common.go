package main

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"

	"github.com/adam-huganir/yutc/internal"
	"github.com/pkg/errors"
)

func createWriter(path, outputPath string, overwrite bool) (io.Writer, error) {
	outPath := filepath.Join(outputPath, path)
	outDir := filepath.Dir(outPath)
	exists, _ := internal.Exists(outDir)
	// we may have a file in the place where our output folder should be, we respect overwrite if there is
	outDirIsDir, err := internal.IsDir(outDir)
	if !exists && err == nil || !outDirIsDir {
		if !overwrite && exists {
			return nil, fmt.Errorf("file found where output requires a folder, %s, you must use overwrite to delete existing file(s)", outDir)
		}
		if exists {
			err = internal.Fs.Remove(outDir)
			if err != nil {
				return nil, err
			}
		}
		err = internal.Fs.MkdirAll(outDir, 0755)
		if err != nil {
			return nil, err
		}
	}
	outWriter, err := internal.Fs.Create(outPath)
	if err != nil {
		return nil, err
	}
	return outWriter, nil
}

func evalTemplate(t *internal.YutcTemplate, commonTemplates []*internal.FileData, data any, outWriter io.Writer) (*bytes.Buffer, error) {
	var err error
	for _, ct := range commonTemplates {
		err = t.AddTemplate(ct.ReadWriter.String())
		if err != nil {
			return nil, errors.Wrapf(err, "error adding common template %s to %s", ct.Path, t.Path())
		}
	}
	result, err := t.Execute(data)
	if err != nil {
		return nil, errors.Wrapf(err, "error executing template %s (%s)", t.Path(), t.ID())

	}
	_, err = outWriter.Write(result.Bytes())
	return result, nil
}
