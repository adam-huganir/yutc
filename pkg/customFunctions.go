package yutc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/isbm/textwrap"
	"github.com/pelletier/go-toml/v2"
)

type YamlEncodeOptions struct {
	Indent                     int
	IndentSequence             bool
	Flow                       bool
	UseLiteralStyleIfMultiline bool
	UseSingleQuote             bool
}

func DefaultYamlEncodeOptions() YamlEncodeOptions {
	return YamlEncodeOptions{
		Indent:                     4,
		IndentSequence:             false,
		Flow:                       false,
		UseLiteralStyleIfMultiline: false,
		UseSingleQuote:             false,
	}
}

type RuntimeOptions struct {
	YamlEncodeOptions YamlEncodeOptions
}

func NewRuntimeOptions() *RuntimeOptions {
	return &RuntimeOptions{
		YamlEncodeOptions: DefaultYamlEncodeOptions(),
	}
}

var runtimeOptions = NewRuntimeOptions()

// SetYamlEncodeOptions sets the global yaml encode options
func SetYamlEncodeOptions(opts map[string]any) (string, error) {
	if indentVal, exists := opts["indent"]; exists {
		var indent int
		switch val := indentVal.(type) {
		case int:
			indent = val
		case int64:
			indent = int(val)
		case float64:
			indent = int(val)
		case uint:
			indent = int(val)
		case uint64:
			indent = int(val)
		default:
			return "", errors.New("indent must be an integer")
		}
		if indent < 0 {
			return "", errors.New("indent must be a positive integer")
		}
		runtimeOptions.YamlEncodeOptions.Indent = indent
	}
	if flowVal, exists := opts["flow"]; exists {
		var flow bool
		switch val := flowVal.(type) {
		case bool:
			flow = val
		default:
			return "", errors.New("flow must be a boolean")
		}
		runtimeOptions.YamlEncodeOptions.Flow = flow
	}
	if indentSequenceVal, exists := opts["indentSequence"]; exists {
		var indentSequence bool
		switch val := indentSequenceVal.(type) {
		case bool:
			indentSequence = val
		default:
			return "", errors.New("indentSequence must be a boolean")
		}
		runtimeOptions.YamlEncodeOptions.IndentSequence = indentSequence
	}
	if useLiteralStyleIfMultilineVal, exists := opts["useLiteralStyleIfMultiline"]; exists {
		var useLiteralStyleIfMultiline bool
		switch val := useLiteralStyleIfMultilineVal.(type) {
		case bool:
			useLiteralStyleIfMultiline = val
		default:
			return "", errors.New("useLiteralStyleIfMultiline must be a boolean")
		}
		runtimeOptions.YamlEncodeOptions.UseLiteralStyleIfMultiline = useLiteralStyleIfMultiline
	}
	if useSingleQuoteVal, exists := opts["useSingleQuote"]; exists {
		var useSingleQuote bool
		switch val := useSingleQuoteVal.(type) {
		case bool:
			useSingleQuote = val
		default:
			return "", errors.New("useSingleQuote must be a boolean")
		}
		runtimeOptions.YamlEncodeOptions.UseSingleQuote = useSingleQuote
	}
	return "", nil
}

// MustToYaml converts an interface to a yaml string or returns an error
func MustToYaml(v interface{}) (string, error) {
	var err error
	var out []byte
	opts := []yaml.EncodeOption{
		yaml.Indent(runtimeOptions.YamlEncodeOptions.Indent),
		yaml.Flow(runtimeOptions.YamlEncodeOptions.Flow),
		yaml.IndentSequence(runtimeOptions.YamlEncodeOptions.IndentSequence),
		yaml.UseLiteralStyleIfMultiline(runtimeOptions.YamlEncodeOptions.UseLiteralStyleIfMultiline),
		yaml.UseSingleQuote(runtimeOptions.YamlEncodeOptions.UseSingleQuote),
	}
	if out, err = yaml.MarshalWithOptions(v, opts...); err != nil {
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

func MustToToml(v interface{}) (string, error) {
	var err error
	var out []byte
	if out, err = toml.Marshal(v); err != nil {
		return "", err
	}
	return string(out), nil
}
func ToToml(v interface{}) string {
	out, _ := MustToToml(v)
	return out
}

func MustFromToml(s string) (interface{}, error) {
	var err error
	var out interface{}
	if err = toml.Unmarshal([]byte(s), &out); err != nil {
		return "", err
	}
	return out, nil
}

func FromToml(s string) interface{} {
	out, _ := MustFromToml(s)
	return out

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

func PathAbsolute(path string) string {
	path = pathCommonClean(path)
	path, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return path
}

func PathGlob(path string) []string {
	path = pathCommonClean(path)
	files, err := filepath.Glob(path)
	if err != nil {
		panic(err)
	}
	return files
}

func PathStat(path string) map[string]interface{} {
	path = pathCommonClean(path)
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			panic(errors.Join(fmt.Errorf("file not found: %s", path)))
		}
		if os.IsPermission(err) {
			panic(errors.Join(fmt.Errorf("permission denied: %s", path)))
		}
		panic(errors.Join(fmt.Errorf("unknown error %v: %s", err, path)))

	}
	return map[string]interface{}{
		"Name":    stat.Name(),
		"Size":    stat.Size(),
		"Mode":    stat.Mode().String(),
		"ModTime": stat.ModTime(),
		"IsDir":   stat.IsDir(),
		"Sys":     stat.Sys(),
	}
}

func pathCommonClean(path string) string {
	return filepath.Clean(os.ExpandEnv(path))
}

func PathIsDir(path string) bool {
	path = pathCommonClean(path)
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return stat.IsDir()
}

func PathIsFile(path string) bool {
	path = pathCommonClean(path)
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !stat.IsDir()
}

func PathExists(path string) bool {
	path = pathCommonClean(path)
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func FileRead(path string) string {
	path = pathCommonClean(path)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		} else {
			panic(fmt.Errorf("file not found: %s", path))
		}
	}
	if info.IsDir() {
		panic(fmt.Errorf("cannot read a directory: %s", path))
	}
	nBytes := int(info.Size())
	return FileReadN(nBytes, path)
}

func FileReadN(nBytes int, path string) string {
	path = pathCommonClean(path)
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	data := make([]byte, nBytes)
	n, err := f.Read(data)
	if err != nil {
		panic(err)
	}
	return string(data[:n])
}

func TypeOf(v interface{}) string {
	return fmt.Sprintf("%T", v)
}

var recursionMaxNums = 10
