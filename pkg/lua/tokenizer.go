package lua

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/spf13/cast"
)

type Parser struct {
	source   string
	position uint32
}

func NewParser(source string) *Parser {
	return &Parser{source: source}
}

//func (p *Parser) Parse() {
//
//}
//
//type Token interface {
//	Name() string
//	NextStates() []Token
//	Optional() bool
//	Repeatable() bool
//}
//
//type StringLiteral struct {
//	Content string
//}
//
//func (sl *StringLiteral) Repeatable() bool {
//	return sl.max > 1
//}
//
//func (sl StringLiteral) Optional() bool {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (sl StringLiteral) Name() string        { return "StringLiteral" }
//func (sl StringLiteral) NextStates() []Token { return []Token{} }
//
//type Chunk struct{}
//
//func (n Chunk) Repeatable() bool {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (n Chunk) Optional() bool {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (n Chunk) Name() string        { return "chunk" }
//func (n Chunk) NextStates() []Token { return []Token{Stat{}, StringLiteral{";"}} }

type Stat struct{}

type TypeInfo interface {
	Name() string
	Value() int
}

type KeyType struct{}

func (k KeyType) Name() string {
	return "key"
}

func (k KeyType) Value() int {
	return 0
}

type UnknownType struct{}

func (k UnknownType) Name() string {
	return "unknown"
}

func (k UnknownType) Value() int {
	return -1
}

func JsonToLua(obj any, typeInfo TypeInfo) string {
	t := reflect.TypeOf(obj)
	lua := ""
	//pos := 0
	kind := t.Kind().String()
	println("Input kind:", kind)
	if slices.Contains([]string{"struct", "map", "array", "slice"}, kind) {
		lua = "{}"
		_ = 1
		if kind == "struct" {
			for i := 0; i < t.NumField(); i++ {
				field := t.Field(i)
				fieldRepr := field.Tag.Get("json")
				if fieldRepr == "" {
					fieldRepr = field.Name
				}
				asLua := JsonToLua(field.Tag.Get("json"), KeyType{})
				println("asLua", asLua)
			}
			f := t.FieldByIndex([]int{
				0,
			})
			print("toString", cast.ToString(f))
		}
	} else if kind == "string" {
		if typeInfo.Name() == "key" {
			lua = fmt.Sprintf("[%s]", obj)
		} else {
			lua = fmt.Sprintf("\"%s\"", obj)
		}
	}
	return lua
}

//
//func (n Stat) Repeatable() bool {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (n Stat) Optional() bool {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (n Stat) Name() string        { return "stat" }
//func (n Stat) NextStates() []Token { return []Token{VarList{}} }
//
//type Block struct{}
//
//func (n Block) Repeatable() bool {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (n Block) Optional() bool {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (n Block) Name() string        { return "block" }
//func (n Block) NextStates() []Token { return []Token{Chunk{}} }
//
//type VarList struct{}
//
//func (v VarList) Repeatable() bool {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (v VarList) Optional() bool {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (v VarList) Name() string        { return "varlist" }
//func (v VarList) NextStates() []Token { return []Token{} }
//
//type ExpList struct{}
