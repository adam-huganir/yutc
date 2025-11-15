package lua

import (
	"testing"
)

func TestNewParser(t *testing.T) {
	example := struct {
		Key  string `json:"key"`
		Key2 string
	}{
		Key:  "test",
		Key2: "test2",
	}
	t.Run("simple test", func(t *testing.T) {
		JsonToLua(example, UnknownType{})
	})
}

//
//func TestParser_Parse(t *testing.T)
//	{
//		type fields struct {
//			source string
//		}
//		tests := []struct {
//			name string
//			text string
//		}{
//			{
//				name: "test simple",
//				text: "local a = 10",
//			},
//		}
//		for _, tt := range tests {
//			t.Run(tt.name, func(t *testing.T) {
//				p := NewParser(tt.text)
//				p.Parse()
//			})
//		}
//	}
