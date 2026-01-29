package data

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/theory/jsonpath"
	"github.com/theory/jsonpath/spec"
)

// SplitSetString parses a --set flag string in the format "path=value" and returns the JSONPath and value.
// The value is automatically unmarshalled from JSON if possible, otherwise returned as a string.
// Convenience feature: paths starting with '.' are auto-prefixed with '$'.
func SplitSetString(s string) (path string, interfaceValue any, err error) {
	var value string
	switch strings.Count(s, "=") {
	case 0:
		return "", "", fmt.Errorf("no '=' found in set string: %s", s)
	case 1:
		parts := strings.SplitN(s, "=", 2)
		path = parts[0]
		value = parts[1]
	default:
		// may handle this differently in the future if it becomes an issue that
		// people want to have '=' in their keys we can implement escaping or something
		parts := strings.SplitN(s, "=", 2)
		path = parts[0]
		value = parts[1]
	}

	path = checkPathPrefix(path)

	if err = json.Unmarshal([]byte(value), &interfaceValue); err != nil {
		// if we can't unmarshal, just return the string value
		interfaceValue = value
	}
	return path, interfaceValue, nil
}

func checkPathPrefix(path string) string {
	// Convenience: auto-prefix with $ if path starts with .
	path = strings.TrimSpace(path)
	if path != "" && path[0] == '.' {
		path = "$" + path
	}
	return path
}

// SetPath sets a value in a nested data structure using JSONPath. A shorthand for SetValueInData direct from a string path.
func SetPath(data *any, path string, value any) error {
	path = checkPathPrefix(path)
	parsed, err := jsonpath.Parse(path)
	if err != nil {
		return err
	}
	segments := parsed.Query().Segments()
	return SetValueInData(data, segments, value, fmt.Sprintf("%v", value))
}

// SetValueInData sets a value in a nested data structure using JSONPath segments.
// It creates intermediate maps/arrays as needed and supports both map keys and array indices.
func SetValueInData(data *any, segments []*spec.Segment, value any, setString string) error {
	current := *data
	writeBack := func(v any) { *data = v } // write back to the original data or parent container
	for i, segment := range segments {
		selector := segment.Selectors()[0]
		isLast := i == len(segments)-1

		switch sel := selector.(type) {
		case spec.Name:
			var key string
			if err := json.Unmarshal([]byte(sel.String()), &key); err != nil {
				return fmt.Errorf("error decoding map key '%s': %w", sel.String(), err)
			}

			m, ok := current.(map[string]any)
			if !ok {
				return fmt.Errorf("error setting --set value '%s': expected map at path segment %v, but found %T", setString, selector, current)
			}
			if isLast {
				m[key] = value
				return nil
			}

			next, exists := m[key]
			if !exists || next == nil {
				next = createNextContainer(segments[i+1].Selectors()[0])
				m[key] = next
			}

			current = next
			writeBack = func(v any) { m[key] = v }

		case spec.Index:
			idx := int(sel)
			arr, ok := current.([]any)
			if !ok {
				return fmt.Errorf("error setting --set value '%s': expected array at path segment %v, but found %T", setString, selector, current)
			}
			if idx < 0 {
				return fmt.Errorf("array index '%d' out of bounds", idx)
			}
			if len(arr) == 0 {
				if idx != 0 {
					return fmt.Errorf("array index '%d' out of bounds", idx)
				}
				arr = append(arr, nil)
				writeBack(arr)
			} else if idx >= len(arr) {
				return fmt.Errorf("array index '%d' out of bounds", idx)
			}

			if isLast {
				arr[idx] = value
				return nil
			}

			next := arr[idx]
			if next == nil {
				next = createNextContainer(segments[i+1].Selectors()[0])
				arr[idx] = next
				writeBack(arr)
			}

			current = next
			writeBack = func(v any) { arr[idx] = v }

		default:
			return fmt.Errorf("unsupported path segment type '%T'", sel)
		}
	}

	return nil
}

func createNextContainer(selector spec.Selector) any {
	if _, isIndex := selector.(spec.Index); isIndex {
		return make([]any, 0)
	}
	return make(map[string]any)
}
