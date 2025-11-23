package files

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/adam-huganir/yutc/pkg/types"
)

// ParseDataFileArg parses a data file argument which can be in two formats:
// 1. Simple path: "./my_file.yaml"
// 2. With key: "key=Secrets,src=./my_secrets.yaml"
func ParseDataFileArg(arg string) (*types.DataFileArg, error) {
	// Check if the argument contains the structured format
	hasKey := strings.Contains(arg, "key=")
	hasSrc := strings.Contains(arg, "src=")

	// If either key= or src= is present, we expect the structured format. if an equals is in there otherwise we just
	// take that as the filename
	if hasKey || hasSrc {
		// Use CSV reader to properly parse comma-separated key=value pairs
		dataArg := &types.DataFileArg{}
		data, err := mapFromKeyValueOption(arg)
		if err != nil {
			return nil, err
		}

		for key, value := range data {
			switch key {
			case "key":
				dataArg.Key = value
			case "src":
				dataArg.Path = value
			default:
				return nil, fmt.Errorf("invalid data argument format with unknown parameter %s: %s", key, arg)
			}
		}
		if dataArg.Path == "" {
			return nil, fmt.Errorf("missing 'src' parameter in data argument: %s", arg)
		}

		return dataArg, nil
	}

	// Simple format - just a path
	return &types.DataFileArg{
		Key:  "",
		Path: arg,
	}, nil
}

func mapFromKeyValueOption(arg string) (map[string]string, error) {
	reader := csv.NewReader(strings.NewReader(arg))
	reader.TrimLeadingSpace = true

	records, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to parse data argument: %w", err)
	}

	data := make(map[string]string)

	for _, part := range records {
		if !strings.Contains(part, "=") {
			return nil, fmt.Errorf("invalid data argument format, no argument provided in %s: %s", part, arg)
		}
		part = strings.TrimSpace(part)
		prefix := part[:strings.Index(part, "=")]  //nolint: gocritic // we already know "=" exists
		value := part[strings.Index(part, "=")+1:] //nolint: gocritic // we already know "=" exists
		data[prefix] = value
	}
	return data, nil
}
