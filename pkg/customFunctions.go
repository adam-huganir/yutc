package yutc

import (
	"errors"
	"fmt"
	"github.com/isbm/textwrap"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
	"text/template"
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

// IncludeFun is an initializer for the include function
func IncludeFun(t *template.Template, includedNames map[string]int) func(string, interface{}) (string, error) {
	// see https://github.com/helm/helm/blob/47529bbffb1d92314373d5df236e87f704357e7f/pkg/engine/engine.go#L144
	return func(name string, data interface{}) (string, error) {
		var buf strings.Builder
		if v, ok := includedNames[name]; ok {
			if v > recursionMaxNums {
				return "", fmt.Errorf(
					"rendering template has a nested reference name: %s: %w",
					name, errors.New("unable to execute template"))
			}
			includedNames[name]++
		} else {
			includedNames[name] = 1
		}
		err := t.ExecuteTemplate(&buf, name, data)
		includedNames[name]--
		return buf.String(), err
	}
}
