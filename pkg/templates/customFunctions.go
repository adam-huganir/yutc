package templates

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/isbm/textwrap"
	"github.com/pelletier/go-toml/v2"
)

// YamlEncodeOptions configures YAML encoding behavior for template functions.
type YamlEncodeOptions struct {
	Indent                     int
	IndentSequence             bool
	Flow                       bool
	UseLiteralStyleIfMultiline bool
	UseSingleQuote             bool
	FinalNewline               bool
}

// DefaultYamlEncodeOptions returns the default YAML encoding options.
func DefaultYamlEncodeOptions() YamlEncodeOptions {
	return YamlEncodeOptions{
		Indent:                     4,
		IndentSequence:             false,
		Flow:                       false,
		UseLiteralStyleIfMultiline: false,
		UseSingleQuote:             false,
		FinalNewline:               false,
	}
}

// RuntimeOptions holds global runtime configuration for template functions.
type RuntimeOptions struct {
	YamlEncodeOptions YamlEncodeOptions
}

// NewRuntimeOptions creates a new RuntimeOptions with default YAML encoding options.
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
			panic("indent must be an integer")
		}
		if indent < 0 {
			panic("indent must be a positive integer")
		}
		runtimeOptions.YamlEncodeOptions.Indent = indent
	}
	if flowVal, exists := opts["flow"]; exists {
		var flow bool
		switch val := flowVal.(type) {
		case bool:
			flow = val
		default:
			panic("flow must be a boolean")
		}
		runtimeOptions.YamlEncodeOptions.Flow = flow
	}
	if indentSequenceVal, exists := opts["indentSequence"]; exists {
		var indentSequence bool
		switch val := indentSequenceVal.(type) {
		case bool:
			indentSequence = val
		default:
			panic("indentSequence must be a boolean")
		}
		runtimeOptions.YamlEncodeOptions.IndentSequence = indentSequence
	}
	if useLiteralStyleIfMultilineVal, exists := opts["useLiteralStyleIfMultiline"]; exists {
		var useLiteralStyleIfMultiline bool
		switch val := useLiteralStyleIfMultilineVal.(type) {
		case bool:
			useLiteralStyleIfMultiline = val
		default:
			panic("useLiteralStyleIfMultiline must be a boolean")
		}
		runtimeOptions.YamlEncodeOptions.UseLiteralStyleIfMultiline = useLiteralStyleIfMultiline
	}
	if useSingleQuoteVal, exists := opts["useSingleQuote"]; exists {
		var useSingleQuote bool
		switch val := useSingleQuoteVal.(type) {
		case bool:
			useSingleQuote = val
		default:
			panic("useSingleQuote must be a boolean")
		}
		runtimeOptions.YamlEncodeOptions.UseSingleQuote = useSingleQuote
	}
	if finalNewlineVal, exists := opts["finalNewline"]; exists {
		var finalNewline bool
		switch val := finalNewlineVal.(type) {
		case bool:
			finalNewline = val
		default:
			panic("finalNewline must be a boolean")
		}
		runtimeOptions.YamlEncodeOptions.FinalNewline = finalNewline
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
	outStr := strings.TrimRight(string(out), "\n")
	if runtimeOptions.YamlEncodeOptions.FinalNewline {
		outStr += "\n"
	}
	return outStr, nil
}

// ToYaml converts an interface to a yaml string
func ToYaml(v interface{}) string {
	out, err := MustToYaml(v)
	if err != nil {
		return ""
	}
	return out
}

// MustFromYaml converts a yaml string to an interface or returns an error
func MustFromYaml(s string) (interface{}, error) {
	var out interface{}
	if err := yaml.Unmarshal([]byte(s), &out); err != nil {
		return "", err
	}
	return out, nil
}

// FromYaml converts a yaml string to an interface
func FromYaml(s string) interface{} {
	out, err := MustFromYaml(s)
	if err != nil {
		return ""
	}
	return out
}

// MustToToml converts an interface to a TOML string or returns an error.
func MustToToml(v interface{}) (string, error) {
	var err error
	var out []byte
	if out, err = toml.Marshal(v); err != nil {
		return "", err
	}
	return string(out), nil
}

// ToToml converts an interface to a TOML string.
func ToToml(v interface{}) string {
	out, err := MustToToml(v)
	if err != nil {
		return ""
	}
	return out
}

// MustFromToml converts a TOML string to an interface or returns an error.
func MustFromToml(s string) (interface{}, error) {
	var out interface{}
	if err := toml.Unmarshal([]byte(s), &out); err != nil {
		panic(err) // as a Must function, we should panic on error
	}
	return out, nil
}

// FromToml converts a TOML string to an interface.
func FromToml(s string) interface{} {
	out, err := MustFromToml(s)
	if err != nil {
		return ""
	}
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

// PathAbsolute returns the absolute path of a file after cleaning and expanding environment variables.
func PathAbsolute(path string) string {
	path = pathCommonClean(path)
	path, err := filepath.Abs(path)
	if err != nil {
		panic(err) // panic as a file not existing means we have an issue with our inputs
	}
	return path
}

// PathGlob returns all file paths matching a glob pattern.
func PathGlob(path string) []string {
	path = pathCommonClean(path)
	files, err := filepath.Glob(path)
	if err != nil {
		panic(err) // panic as a file not existing means we have an issue with our inputs
	}
	return files
}

// PathStat returns file information as a map including name, size, mode, modification time, and is_dir flag.
func PathStat(path string) map[string]interface{} {
	path = pathCommonClean(path)
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			panic(errors.Join(fmt.Errorf("file not found: %s", path))) // panic as a file not existing means we have an issue with our inputs
		}
		if os.IsPermission(err) {
			panic(errors.Join(fmt.Errorf("permission denied: %s", path)))
		}
		panic(errors.Join(fmt.Errorf("unknown error %w: %s", err, path)))

	}
	return map[string]interface{}{
		"Name":    stat.Name(),
		"Items":   stat.Size(),
		"Mode":    stat.Mode().String(),
		"ModTime": stat.ModTime(),
		"IsDir":   stat.IsDir(),
		"Sys":     stat.Sys(),
	}
}

func pathCommonClean(path string) string {
	return filepath.Clean(os.ExpandEnv(path))
}

// PathIsDir checks if a path is a directory.
func PathIsDir(path string) bool {
	path = pathCommonClean(path)
	stat, err := os.Stat(path)
	if err != nil {
		panic(err) // panic as a file not existing means we have an issue with our inputs
	}
	return stat.IsDir()
}

// PathIsFile checks if a path is a file (not a directory).
func PathIsFile(path string) bool {
	path = pathCommonClean(path)
	stat, err := os.Stat(path)
	if err != nil {
		panic(err) // panic as a file not existing means we have an issue with our inputs
	}
	return !stat.IsDir()
}

// PathExists checks if a path exists.
func PathExists(path string) bool {
	path = pathCommonClean(path)
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	} else if err == nil {
		return true
	}
	panic(err)
}

// FileRead reads an entire file and returns its contents as a string.
func FileRead(path string) string {
	path = pathCommonClean(path)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			panic(fmt.Errorf("file not found: %s", path))
		}
		panic(err)
	}
	if info.IsDir() {
		panic(fmt.Errorf("cannot read a directory: %s", path))
	}
	nBytes := int(info.Size())
	return FileReadN(nBytes, path)
}

// FileReadN reads the first nBytes from a file and returns them as a string.
func FileReadN(nBytes int, path string) string {
	path = pathCommonClean(path)
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer func() { _ = f.Close() }()
	data := make([]byte, nBytes)
	n, err := f.Read(data)
	if err != nil {
		panic(err)
	}
	return string(data[:n])
}

func SortList(v []any) []any {
	sorted := make([]any, len(v))
	copy(sorted, v)
	// check types of all elements
	if len(v) == 0 {
		return sorted
	}
	switch sorted[0].(type) {
	case string:
		strs := make([]string, len(sorted))
		for i, val := range sorted {
			strs[i] = val.(string)
		}
		sort.Strings(strs)
		for i, val := range strs {
			sorted[i] = val
		}
	case int:
		ints := make([]int, len(sorted))
		for i, val := range sorted {
			ints[i] = val.(int)
		}
		sort.Ints(ints)
		for i, val := range ints {
			sorted[i] = val
		}
	case float64:
		floats := make([]float64, len(sorted))
		for i, val := range sorted {
			floats[i] = val.(float64)
		}
		sort.Float64s(floats)
		for i, val := range floats {
			sorted[i] = val
		}
	default:
		panic("unsupported type for sorting")
	}
	return sorted
}

// SortKeys returns a new map with the keys sorted in ascending order, not recursive
func SortKeys(m map[string]any) map[string]any {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sorted := make(map[string]any)
	for _, k := range keys {
		sorted[k] = m[k]
	}
	return sorted
}

// TypeOf returns the type of a value as a string.
func TypeOf(v interface{}) string {
	return fmt.Sprintf("%T", v)
}

var recursionMaxNums = 10
