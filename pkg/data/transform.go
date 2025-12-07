package data

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ToPascalCase converts a string to PascalCase.
func ToPascalCase(s string) string {
	if s == "" {
		return ""
	}
	// Replace common separators with spaces, then title case, then remove spaces
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	s = cases.Title(language.English, cases.NoLower).String(s)
	return strings.ReplaceAll(s, " ", "")
}

// KeysToPascalCase recursively transforms all keys in a map to PascalCase.
func KeysToPascalCase(data map[string]interface{}) map[string]interface{} {
	newData := make(map[string]interface{})
	for k, v := range data {
		newKey := ToPascalCase(k)
		switch v := v.(type) {
		case map[string]interface{}:
			newData[newKey] = KeysToPascalCase(v)
		case []interface{}:
			var newSlice []interface{}
			for _, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					newSlice = append(newSlice, KeysToPascalCase(m))
				} else {
					newSlice = append(newSlice, item)
				}
			}
			newData[newKey] = newSlice
		default:
			newData[newKey] = v
		}
	}
	return newData
}
