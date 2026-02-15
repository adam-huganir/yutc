package data

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/theory/jsonpath"
)

// NormalizeFilepath cleans and normalizes a file path to use forward slashes.
func NormalizeFilepath(file string) string {
	return filepath.ToSlash(filepath.Clean(path.Join(file)))
}

func applySetArgs(dst *map[string]any, setArgs []string, logger *zerolog.Logger) error {
	if len(setArgs) == 0 {
		return nil
	}

	mergedDataAny := any(*dst)
	for _, ss := range setArgs {
		pathExpr, value, err := SplitSetString(ss)
		if err != nil {
			return fmt.Errorf("error parsing --set value '%s': %w", ss, err)
		}
		parsed, err := jsonpath.Parse(pathExpr)
		if err != nil {
			return fmt.Errorf("error parsing --set value '%s': %w", ss, err)
		}
		if pq := parsed.Query().Singular(); pq == nil {
			return fmt.Errorf("error parsing --set value '%s': resulting path is not unique singular path", ss)
		}
		err = SetValueInData(&mergedDataAny, parsed.Query().Segments(), value, ss)
		if err != nil {
			return err
		}
		if logger != nil {
			logger.Debug().Msgf("set %s to %v", parsed, value)
		}
	}

	mergedData, ok := mergedDataAny.(map[string]any)
	if !ok {
		return fmt.Errorf("error applying --set values: expected map at root, got %T", mergedDataAny)
	}
	*dst = mergedData
	return nil
}
