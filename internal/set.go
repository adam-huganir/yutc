package internal

import (
	"encoding/json"
	"fmt"
	"strings"
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
	var interfaceValue any
	err := json.Unmarshal([]byte(value), &interfaceValue)
	if err != nil {
		// if we can't unmarshal, just return the string value
		interfaceValue = value
	}
	return strings.TrimSpace(path), interfaceValue, nil
}
