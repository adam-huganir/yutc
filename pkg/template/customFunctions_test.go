package template

import (
	"bytes"
	"testing"

	"github.com/adam-huganir/yutc/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestSortListTemplate(t *testing.T) {
	template := `{{ range sortList .items }}{{ . }} {{ end }}`
	tests := []struct {
		name           string
		data           map[string]any
		expectedOutput string
		wantErr        bool
	}{
		{
			name:           "sort string list",
			data:           map[string]any{"items": []any{"banana", "apple", "cherry"}},
			expectedOutput: "apple banana cherry ",
		},
		{
			name:           "sort int list",
			data:           map[string]any{"items": []any{5, 3, 8, 1}},
			expectedOutput: "1 3 5 8 ",
		},
		{
			name:           "sort float list",
			data:           map[string]any{"items": []any{3.14, 1.41, 2.71}},
			expectedOutput: "1.41 2.71 3.14 ",
		},
		{
			name:    "unsupported type",
			data:    map[string]any{"items": []any{true, false}},
			wantErr: true,
		},
		{
			name:           "empty list",
			data:           map[string]any{"items": []any{}},
			expectedOutput: "",
		},
		{
			name:           "single element list",
			data:           map[string]any{"items": []any{"onlyone"}},
			expectedOutput: "onlyone ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := BuildTemplate(template, nil, "test", false)
			assert.NoError(t, err)

			var buf bytes.Buffer
			err = tmpl.Execute(&buf, tt.data)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, buf.String())
		})
	}
}

func TestSortList(t *testing.T) {
	tests := []struct {
		name        string
		input       []any
		expected    []any
		expectPanic bool
	}{
		{
			name:     "empty list",
			input:    []any{},
			expected: []any{},
		},
		{
			name:     "string list",
			input:    []any{"zebra", "apple", "banana"},
			expected: []any{"apple", "banana", "zebra"},
		},
		{
			name:     "string list already sorted",
			input:    []any{"apple", "banana", "zebra"},
			expected: []any{"apple", "banana", "zebra"},
		},
		{
			name:     "single string",
			input:    []any{"apple"},
			expected: []any{"apple"},
		},
		{
			name:     "int list",
			input:    []any{3, 1, 4, 1, 5, 9, 2, 6},
			expected: []any{1, 1, 2, 3, 4, 5, 6, 9},
		},
		{
			name:     "int list already sorted",
			input:    []any{1, 2, 3, 4, 5},
			expected: []any{1, 2, 3, 4, 5},
		},
		{
			name:     "single int",
			input:    []any{42},
			expected: []any{42},
		},
		{
			name:     "float64 list",
			input:    []any{3.14, 1.41, 2.71, 0.57},
			expected: []any{0.57, 1.41, 2.71, 3.14},
		},
		{
			name:     "float64 list already sorted",
			input:    []any{1.1, 2.2, 3.3},
			expected: []any{1.1, 2.2, 3.3},
		},
		{
			name:     "single float64",
			input:    []any{3.14},
			expected: []any{3.14},
		},
		{
			name:        "unsupported type - bool",
			input:       []any{true, false},
			expectPanic: true,
		},
		{
			name:        "unsupported type - map",
			input:       []any{map[string]any{"key": "value"}},
			expectPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic {
				assert.Panics(t, func() {
					SortList(tt.input)
				})
			} else {
				result := SortList(tt.input)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSortKeys(t *testing.T) {
	tests := []struct {
		name         string
		input        map[string]any
		expectedKeys []string
	}{
		{
			name:         "empty map",
			input:        map[string]any{},
			expectedKeys: []string{},
		},
		{
			name: "single key",
			input: map[string]any{
				"key": "value",
			},
			expectedKeys: []string{"key"},
		},
		{
			name: "multiple keys unsorted",
			input: map[string]any{
				"zebra":  1,
				"apple":  2,
				"banana": 3,
			},
			expectedKeys: []string{"apple", "banana", "zebra"},
		},
		{
			name: "multiple keys already sorted",
			input: map[string]any{
				"alpha": 1,
				"beta":  2,
				"gamma": 3,
			},
			expectedKeys: []string{"alpha", "beta", "gamma"},
		},
		{
			name: "numeric string keys",
			input: map[string]any{
				"3": "three",
				"1": "one",
				"2": "two",
			},
			expectedKeys: []string{"1", "2", "3"},
		},
		{
			name: "mixed value types",
			input: map[string]any{
				"string": "value",
				"number": 42,
				"float":  3.14,
				"bool":   true,
			},
			expectedKeys: []string{"bool", "float", "number", "string"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SortKeys(tt.input)

			// Check that all keys are present and values match
			assert.Equal(t, len(tt.input), len(result))
			for k, v := range tt.input {
				assert.Equal(t, v, result[k], "value for key %s should match", k)
			}

			// Extract keys from result and verify they're in sorted order
			var resultKeys []string
			for k := range result {
				resultKeys = append(resultKeys, k)
			}

			// Since Go maps are unordered, we need to collect and compare
			assert.ElementsMatch(t, tt.expectedKeys, resultKeys)
		})
	}
}

func TestSortListInTemplate(t *testing.T) {
	tests := []struct {
		name           string
		template       string
		data           map[string]any
		expectedOutput string
	}{
		{
			name:     "sort string list",
			template: `{{ range sortList .items }}{{ . }} {{ end }}`,
			data: map[string]any{
				"items": []any{"zebra", "apple", "banana"},
			},
			expectedOutput: "apple banana zebra ",
		},
		{
			name:     "sort int list",
			template: `{{ range sortList .numbers }}{{ . }} {{ end }}`,
			data: map[string]any{
				"numbers": []any{3, 1, 4, 1, 5},
			},
			expectedOutput: "1 1 3 4 5 ",
		},
		{
			name:     "sort float list",
			template: `{{ range sortList .floats }}{{ . }} {{ end }}`,
			data: map[string]any{
				"floats": []any{3.14, 1.41, 2.71},
			},
			expectedOutput: "1.41 2.71 3.14 ",
		},
		{
			name:     "sort empty list",
			template: `{{ range sortList .items }}{{ . }} {{ end }}empty`,
			data: map[string]any{
				"items": []any{},
			},
			expectedOutput: "empty",
		},
		{
			name:     "sort and index",
			template: `{{ index (sortList .items) 0 }}`,
			data: map[string]any{
				"items": []any{"zebra", "apple", "banana"},
			},
			expectedOutput: "apple",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := BuildTemplate(tt.template, nil, "test", false)
			assert.NoError(t, err)

			var buf bytes.Buffer
			err = tmpl.Execute(&buf, tt.data)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, buf.String())
		})
	}
}

func TestSortKeysInTemplate(t *testing.T) {
	tests := []struct {
		name           string
		template       string
		data           map[string]any
		expectedOutput string
	}{
		{
			name:     "iterate over sorted keys",
			template: `{{ range $key, $value := sortKeys .data }}{{ $key }}:{{ $value }} {{ end }}`,
			data: map[string]any{
				"data": map[string]any{
					"zebra":  "z",
					"apple":  "a",
					"banana": "b",
				},
			},
			expectedOutput: "apple:a banana:b zebra:z ",
		},
		{
			name:     "sort keys with mixed values",
			template: `{{ range $key, $value := sortKeys .config }}{{ $key }}={{ $value }} {{ end }}`,
			data: map[string]any{
				"config": map[string]any{
					"port":    8080,
					"host":    "localhost",
					"enabled": true,
				},
			},
			expectedOutput: "enabled=true host=localhost port=8080 ",
		},
		{
			name:     "sort keys empty map",
			template: `{{ range $key, $value := sortKeys .data }}{{ $key }}:{{ $value }} {{ end }}empty`,
			data: map[string]any{
				"data": map[string]any{},
			},
			expectedOutput: "empty",
		},
		{
			name:     "nested sortKeys and sortList",
			template: `{{ range $key, $value := sortKeys .items }}{{ $key }}: {{ range sortList $value }}{{ . }} {{ end }}| {{ end }}`,
			data: map[string]any{
				"items": map[string]any{
					"fruits": []any{"banana", "apple"},
					"colors": []any{"red", "blue"},
				},
			},
			expectedOutput: "colors: blue red | fruits: apple banana | ",
		},
		{
			name: "sort keys for YAML output consistency",
			template: util.MustDedent(`
			config:
			{{ range $key, $value := sortKeys .settings }}  {{ $key }}: {{ $value }}
			{{ end }}`),
			data: map[string]any{
				"settings": map[string]any{
					"timeout": 30,
					"retries": 3,
					"debug":   false,
				},
			},
			expectedOutput: util.MustDedent(`
				config:
				  debug: false
				  retries: 3
				  timeout: 30
			`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := BuildTemplate(tt.template, nil, "test", false)
			assert.NoError(t, err)

			var buf bytes.Buffer
			err = tmpl.Execute(&buf, tt.data)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, buf.String())
		})
	}
}
