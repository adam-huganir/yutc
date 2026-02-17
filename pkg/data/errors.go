package data

import (
	"github.com/adam-huganir/yutc/pkg/loader"
)

// Sentinel errors re-exported from pkg/loader.
var (
	ErrIsContainer = loader.ErrIsContainer
	ErrNotLoaded   = loader.ErrNotLoaded
)
