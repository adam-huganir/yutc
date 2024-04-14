package yutc

import (
	"errors"
	"fmt"
	"github.com/isbm/textwrap"
	"gopkg.in/yaml.v3"
	"strings"
)

// MustToYaml converts an interface to a yaml string or returns an error
func MustToYaml(v interface{}) (string, error) {
	var err error
	var out []byte
	if out, err = yaml.Marshal(v); err != nil {
		return "", err
	}
	return string(out), nil
}

// ToYaml converts an interface to a yaml string
func ToYaml(v interface{}) string {
	out, _ := MustToYaml(v)
	return out
}

// MustFromYaml converts a yaml string to an interface or returns an error
func MustFromYaml(s string) (interface{}, error) {
	var err error
	var out interface{}
	if err = yaml.Unmarshal([]byte(s), &out); err != nil {
		return "", err
	}
	return out, nil
}

// FromYaml converts a yaml string to an interface
func FromYaml(s string) interface{} {
	out, _ := MustFromYaml(s)
	return out
}

func StringMap(v interface{}) (map[string]interface{}, error) {
	// i don't feel like writing a recursive function right now
	return nil, errors.New("not implemented")
}

// WrapText wraps text to a given width
func WrapText(width int, text string) []string {
	wrapper := textwrap.NewTextWrap()
	wrapper.SetWidth(width)
	return wrapper.Wrap(text)
}

// WrapComment wraps a text to a give with and then prefixes the lines (e.g. "#" for a python comment)
func WrapComment(prefix string, width int, comment string) string {
	var wrapped []string
	for _, line := range WrapText(width, comment) {
		wrapped = append(wrapped, fmt.Sprintf("%s %s", prefix, line))
	}
	return strings.Join(wrapped, "\n")
}
