package data

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/theory/jsonpath/spec"
)

func SplitSetString(s string) (string, any, error) {
	var path, value string
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

	// Convenience: auto-prefix with $ if path starts with .
	path = strings.TrimSpace(path)
	if len(path) > 0 && path[0] == '.' {
		path = "$" + path
	}

	var interfaceValue any
	err := json.Unmarshal([]byte(value), &interfaceValue)
	if err != nil {
		// if we can't unmarshal, just return the string value
		interfaceValue = value
	}
	return path, interfaceValue, nil
}

func SetValueInData(data map[string]any, segments []*spec.Segment, value any, setString string) error {
	current := any(data)

	for i, segment := range segments {
		selector := segment.Selectors()[0]
		isLast := i == len(segments)-1

		switch sel := selector.(type) {
		case spec.Name:
			var key string
			if err := json.Unmarshal([]byte(sel.String()), &key); err != nil {
				return fmt.Errorf("error decoding map key '%s': %v", sel.String(), err)
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
			if !exists {
				next = createNextContainer(segments[i+1].Selectors()[0])
				m[key] = next
			}
			current = next

		case spec.Index:
			idx := int(sel)
			arr, ok := current.([]any)
			if !ok {
				return fmt.Errorf("error setting --set value '%s': expected array at path segment %v, but found %T", setString, selector, current)
			}
			if idx < 0 || (len(arr) > 0 && idx >= len(arr)) {
				return fmt.Errorf("array index '%d' out of bounds", idx)
			}
			if len(arr) == 0 && idx == 0 {
				arr = append(arr, nil)
			}

			if isLast {
				arr[idx] = value
				return nil
			}

			next := arr[idx]
			if next == nil {
				next = createNextContainer(segments[i+1].Selectors()[0])
				arr[idx] = next
			}
			current = next

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
