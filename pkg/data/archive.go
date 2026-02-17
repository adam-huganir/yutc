package data

import (
	"github.com/adam-huganir/yutc/pkg/loader"
)

// Type and function aliases re-exported from pkg/loader.
type FilePathMap = loader.FilePathMap

var (
	IsArchive = loader.IsArchive
	ReadTar   = loader.ReadTar
)
