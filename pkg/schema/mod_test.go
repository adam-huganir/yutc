package schema

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func jsonNormalize(d any) (string, error) {
	b, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	var d2 any
	err = json.Unmarshal(b, &d2)
	if err != nil {
		return "", err
	}
	out, err := json.Marshal(d2)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func mustJSONNormalize(d any) string {
	dOut, err := jsonNormalize(d)
	if err != nil {
		panic(err)
	}
	return dOut
}

func TestLoadSchema(t *testing.T) {

	type args struct {
		schema []byte
	}
	tests := []struct {
		name         string
		expectedType string // proxy for empty
		args         args
	}{
		{
			name:         "valid",
			expectedType: "object",
			args: args{
				schema: []byte(`
					{
						"type": "object",
						"properties": {
							"name": {"type": "string"},
							"age": {"type": "integer", "default": 21}
						},
						"required": ["name"]
					}
			`),
			},
		}, {
			name:         "empty",
			expectedType: "",
			args: args{
				schema: []byte(`
					{}
			`),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := LoadSchema(tt.args.schema)
			if err != nil {
				t.Errorf("LoadSchema() error = %v", err)
			}
			assert.Equal(t, tt.expectedType, s.Type)
		})
	}
}

func TestResolveSchema(t *testing.T) {
	type args struct {
		data   any
		schema string
	}
	tests := []struct {
		name string
		args args
		want any
		err  string
	}{
		{
			name: "test simple validation success",
			args: args{
				data:   map[string]any{},
				schema: `{"type": "object"}`,
			},
			want: map[string]any{},
			err:  "",
		},
		{
			name: "test simple validation on something more complex",
			args: args{
				data: map[string]any{
					"name": "adam",
				},
				schema: `{
					"type": "object",
					"description": "something or other",
					"properties": {
						"name": {"type": "string"},
						"age": {"type": "integer", "default": 21}
					},
					"required": ["name"]
				}`,
			},
			want: map[string]any{
				"name": "adam",
				"age":  21,
			},
			err: "",
		},
		{
			name: "test validation failure with age type mismatch",
			args: args{
				data: map[string]any{
					"name": "adam",
					"age":  "38",
				},
				schema: `{
					"type": "object",
					"description": "something or other",
					"properties": {
						"name": {"type": "string"},
						"age": {"type": "integer", "default": 21}
					},
					"required": ["name"]
				}`,
			},
			want: nil,
			err:  `validate error: validating root: validating /properties/age: type: 38 has type "string", want "integer"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := ResolveSchema(tt.args.data, []byte(tt.args.schema))
			if tt.err != "" {
				assert.Error(t, err, "ResolveSchema() expected err but returned err = nil)")
				assert.Equal(t, tt.err, strings.TrimSpace(err.Error()))
				return
			}
			if err != nil {
				t.Errorf("ResolveSchema() error = %v", err)
			}
			assert.Equal(t, mustJSONNormalize(tt.want), mustJSONNormalize(d))
		})
	}
}
