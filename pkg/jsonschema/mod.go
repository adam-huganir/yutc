package schema

import (
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
)

func ResolveSchema(data any, schema []byte) (any, error) {
	s, err := LoadSchema(schema)
	if err != nil {
		return nil, fmt.Errorf("Load schema error: %w\n", err)

	}

	r, err := s.Resolve(&jsonschema.ResolveOptions{ValidateDefaults: true})
	if err != nil {
		return nil, fmt.Errorf("resolve jsonschema error: %w", err)
	}
	err = r.ApplyDefaults(data)
	if err != nil {
		return nil, fmt.Errorf("Apply defaults error: %w\n", err)
	}

	err = r.Validate(data)
	if err != nil {
		return nil, fmt.Errorf("Validate error: %w\n", err)

	}
	return data, nil
}

// LoadSchema loads a schema from a byte array and returns a resolved schema.
func LoadSchema(schema []byte) (r *jsonschema.Schema, err error) {
	s := jsonschema.Schema{}
	if string(schema) == "" {
		return nil, fmt.Errorf("schema is empty")
	}

	err = s.UnmarshalJSON(schema)
	if err != nil {
		return nil, fmt.Errorf("unmarshal jsonschema error: %w", err)
	}
	return &s, err
}
