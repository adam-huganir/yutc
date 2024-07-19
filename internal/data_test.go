package internal

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func Test_coerceToStringMap(t *testing.T) {
	type args struct {
		dataAny any
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "nil",
			args: args{
				map[any]any{
					"test": 10,
					10:     "test",
				},
			},
			want: map[string]any{
				"test": 10,
				"10":   "test",
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := coerceToStringMap(tt.args.dataAny)
			if !tt.wantErr(t, err, fmt.Sprintf("coerceToStringMap(%v)", tt.args.dataAny)) {
				return
			}
			assert.Equalf(t, tt.want, got, "coerceToStringMap(%v)", tt.args.dataAny)
		})
	}
}

func Test_initAndUpdateMap(t *testing.T) {
	data := initData(map[string]any{"a": 2})
	assert.Equal(t, map[string]any{"a": 2}, data)

	data, _, _ = updateData(bytes.NewBufferString("{\"b\": \"10\"}"), data, reflect.Map, false)
	assert.Equal(t, map[string]any{"a": 2, "b": "10"}, data)
}

func Test_initAppendArray(t *testing.T) {
	var err error
	data := initData([]any{1, 2, "a"})
	assert.Equal(t, []any{1, 2, "a"}, data)

	toUpdate := bytes.NewBufferString("[5,4,3]")
	_, _, err = updateData(toUpdate, data, reflect.Map, false)
	assert.ErrorContains(t, err, "cannot merge data of different types map and slice")

	_, _, err = updateData(toUpdate, data, reflect.Slice, false)
	assert.ErrorContains(t, err, "cannot merge lists without append mode")

	data, _, err = updateData(toUpdate, data, reflect.Slice, true)
	assert.Equal(t, data, []any{1, 2, "a", 5, 4, 3})
}

func Test_initMergeScalars(t *testing.T) {
	var err error
	data := initData("a literal string")
	assert.Equal(t, "a literal string", data)

	_, _, err = updateData(bytes.NewBufferString("{\"b\": \"10\"}"), data, reflect.String, false)
	assert.ErrorContains(t, err, "cannot merge data of different types string and map")

	_, _, err = updateData(bytes.NewBufferString("\"another string literal\""), data, reflect.String, false)
	assert.ErrorContains(t, err, "cannot merge data of type string")
}
