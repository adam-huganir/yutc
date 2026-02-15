// This file re-exports loader types so that existing pkg/data consumers
// can continue to use data.FileEntry, data.SourceKindFile, etc.
// New code should import pkg/loader directly.
package data

import (
	"github.com/adam-huganir/yutc/pkg/loader"
)

// Type aliases re-exported from pkg/loader.
type FileEntry = loader.FileEntry
type FileEntryOption = loader.FileEntryOption
type FileContent = loader.FileContent
type SourceKind = loader.SourceKind
type RemoteInfo = loader.RemoteInfo
type AuthInfo = loader.AuthInfo
type HTTPStatusError = loader.HTTPStatusError

// Constant aliases.
const (
	SourceKindFile   = loader.SourceKindFile
	SourceKindURL    = loader.SourceKindURL
	SourceKindStdin  = loader.SourceKindStdin
	SourceKindStdout = loader.SourceKindStdout
)

// Constructor / option aliases.
var (
	NewFileEntry     = loader.NewFileEntry
	NewFileContent   = loader.NewFileContent
	WithSource       = loader.WithSource
	WithContent      = loader.WithContent
	WithContentBytes = loader.WithContentBytes
	WithAuth         = loader.WithAuth
	WithLogger       = loader.WithLogger
)

// Function aliases.
var (
	GetURL             = loader.GetURL
	NewHTTPStatusError = loader.NewHTTPStatusError
)
