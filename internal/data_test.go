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

func Test_updateData(t *testing.T) {
	type args struct {
		contentBuffer *bytes.Buffer
		currentData   any
		lastType      reflect.Kind
		appendMode    bool
	}
	tests := []struct {
		name     string
		args     args
		want     any
		wantType reflect.Kind
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name: "first item is map",
			args: args{
				bytes.NewBuffer([]byte("{\"a\": 1}")),
				nil,
				reflect.Invalid,
				false,
			},
			want: map[string]any{
				"a": 1,
			},
			wantType: reflect.Map,
			wantErr:  assert.NoError,
		},
		{
			name: "first item is list",
			args: args{
				bytes.NewBuffer([]byte("[\"a\", 1]")),
				nil,
				reflect.Invalid,
				false,
			},
			want: []any{
				"a", 1,
			},
			wantType: reflect.Slice,
			wantErr:  assert.NoError,
		},
		{
			name: "2 maps",
			args: args{
				bytes.NewBuffer([]byte("{\"a\": 1}")),
				map[string]any{"b": 2},
				reflect.Map,
				false,
			},
			want: map[string]any{
				"a": 1,
				"b": 2,
			},
			wantType: reflect.Map,
			wantErr:  assert.NoError,
		},
		{
			name: "2 slices with append",
			args: args{
				bytes.NewBuffer([]byte("[1]")),
				[]any{2, 3},
				reflect.Slice,
				true,
			},
			want:     []any{2, 3, 1},
			wantType: reflect.Slice,
			wantErr:  assert.NoError,
		},
		{
			name: "2 slices without append",
			args: args{
				bytes.NewBuffer([]byte("[1]")),
				[]any{"hello", "goodbye"},
				reflect.Slice,
				false,
			},
			want:     nil,
			wantType: reflect.Invalid,
			wantErr:  assert.Error,
		},
		{
			name: "2 scalars",
			args: args{
				bytes.NewBuffer([]byte("1")),
				5,
				reflect.Int,
				false,
			},
			want:     nil,
			wantType: reflect.Invalid,
			wantErr:  assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			T := assert.TestingT(t)
			got, gotType, err := updateData(tt.args.contentBuffer, &tt.args.currentData, tt.args.lastType, tt.args.appendMode)
			if !tt.wantErr(T, err, fmt.Sprintf("updateData(%v, %v, %v, %v)", tt.args.contentBuffer, tt.args.currentData, tt.args.lastType, tt.args.appendMode)) {
				return
			}
			assert.Equalf(T, tt.wantType, gotType, "updateData(%v, %v, %v, %v) -> type", tt.args.contentBuffer, tt.args.currentData, tt.args.lastType, tt.args.appendMode)
			if err == nil {
				assert.Equalf(T, tt.want, got, "updateData(%v, %v, %v, %v) -> data", tt.args.contentBuffer, tt.args.currentData, tt.args.lastType, tt.args.appendMode)
			}
		})
	}
}

func TestCollateData(t *testing.T) {
	type args struct {
		dataFiles  []string
		appendMode bool
	}
	tests := []struct {
		name    string
		args    args
		want    any
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "single file",
			args: args{
				[]string{"../testFiles/data/data1.yaml"},
				false,
			},
			want:    map[string]interface{}{"dogs": []interface{}{map[string]interface{}{"breed": "Labrador", "name": "Fido", "owner": map[string]interface{}{"name": "John Doe"}, "vaccinations": []interface{}{"rabies"}}}, "thisWillMerge": map[string]interface{}{"value23": "not 23", "value24": 24}},
			wantErr: assert.NoError,
		}, {
			name: "2 maps",
			args: args{
				[]string{"../testFiles/data/data1.yaml", "../testFiles/data/data2.yaml"},
				false,
			},
			want:    map[string]interface{}{"ditto": []interface{}{"woohooo", "yipeee"}, "dogs": []interface{}{}, "thisIsNew": 1000, "thisWillMerge": map[string]interface{}{"value23": 23, "value24": 24}},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := CollateData(tt.args.dataFiles, tt.args.appendMode)
			if !tt.wantErr(t, err, fmt.Sprintf("CollateData(%v, %v)", tt.args.dataFiles, tt.args.appendMode)) {
				return
			}
			assert.Equalf(t, tt.want, got, "CollateData(%v, %v)", tt.args.dataFiles, tt.args.appendMode)
		})
	}
}
