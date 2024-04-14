package yutc

import (
	"errors"
	"fmt"
	"github.com/isbm/textwrap"
	"gopkg.in/yaml.v3"
	"strings"
)

func MustToYaml(v interface{}) (string, error) {
	var err error
	var out []byte
	if out, err = yaml.Marshal(v); err != nil {
		return "", err
	}
	return string(out), nil
}

func ToYaml(v interface{}) string {
	out, _ := MustToYaml(v)
	return out
}

func MustFromYaml(s string) (interface{}, error) {
	var err error
	var out interface{}
	if err = yaml.Unmarshal([]byte(s), &out); err != nil {
		return "", err
	}
	return out, nil
}

func FromYaml(s string) interface{} {
	out, _ := MustFromYaml(s)
	return out
}

func StringMap(v interface{}) (map[string]interface{}, error) {
	// i don't feel like writing a recursive function right now
	return nil, errors.New("not implemented")
}

func WrapText(width int, text string) []string {
	wrapper := textwrap.NewTextWrap()
	wrapper.SetWidth(width)
	return wrapper.Wrap(text)
}

func WrapComment(prefix string, width int, comment string) string {
	var wrapped []string
	for _, line := range WrapText(width, comment) {
		wrapped = append(wrapped, fmt.Sprintf("%s %s", prefix, line))
	}
	return strings.Join(wrapped, "\n")
}
