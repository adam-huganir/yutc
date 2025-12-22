package schema

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func jsonEqual(a, b any) bool {
	aBuf := &bytes.Buffer{}
	bBuf := &bytes.Buffer{}

	aEnc := json.NewEncoder(aBuf)
	aEnc.SetIndent("", "")
	bEnc := json.NewEncoder(bBuf)
	bEnc.SetIndent("", "")

	if err := aEnc.Encode(a); err != nil {
		return false
	}
	if err := bEnc.Encode(b); err != nil {
		return false
	}

	return aBuf.String() == bBuf.String()
}

func TestLoadSchema(t *testing.T) {

	type args struct {
		schema []byte
		url    *url.URL
	}
	tests := []struct {
		name          string
		expected_type string // proxy for empty
		args          args
	}{
		{
			name:          "valid",
			expected_type: "object",
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
			name:          "empty",
			expected_type: "",
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
			assert.Equal(t, tt.expected_type, s.Type)
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
			name: "test simple validation on something more complex",
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
			want: map[string]any{
				"name": "adam",
				"age":  38,
			},
			err: `Validate error: validating root: validating /properties/age: type: 38 has type "string", want "integer"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := ResolveSchema(&tt.args.data, []byte(tt.args.schema))
			if tt.err != "" {
				assert.Equal(t, tt.err, strings.TrimSpace(err.Error()))
				return
			}
			if err != nil {
				t.Errorf("ResolveSchema() error = %v", err)
			}
			assert.True(t, jsonEqual(d, tt.want))
		})
	}
}
