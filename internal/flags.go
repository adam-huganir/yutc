package internal

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

type RepeatedStringFlag []string

func (rs *RepeatedStringFlag) String() string {
	return fmt.Sprintf("%v", []string(*rs))
}

func (rs *RepeatedStringFlag) Set(value string) error {
	*rs = append(*rs, value)
	return nil
}

type CLIOptions struct {
	Stdin         bool     `json:"stdin"`
	NoStdin       bool     `json:"no-stdin"`
	DataFiles     []string `json:"data-files"`
	TemplateFiles []string `json:"template-files"`
	Output        string   `json:"output"`
	Overwrite     bool     `json:"overwrite"`
	NoOverwrite   bool     `json:"no-overwrite"`
	StdinFirst    bool     `json:"stdin-first"`
	NoStdinFirst  bool     `json:"no-stdin-first"`
}

var HelpMessages = map[string]string{
	"stdin":       "Read data from stdin",
	"stdin-first": "Read data from stdin before merging with specified data files",
	"overwrite":   "Overwrite existing files",
	"data":        "Data file to parse and merge. Can be a file or a URL. Can be specified multiple times and the inputs will be merged.",
	"template":    "Template file to parse and merge, Can be a file or a URL. Can be specified multiple times.",
	"output":      "Output file/directory, defaults to stdout",
	"version":     "Print the version and exit",
}

func OverComplicatedHelp() {
	println("Usage: yutc [flags] <template ...>\n")
	println("  Options:")
	flag.VisitAll(func(f *flag.Flag) {
		isBool := f.Value.String() == "true" || f.Value.String() == "false"
		skip := strings.HasPrefix(f.Name, "no-") && isBool
		if skip {
			return
		}
		totalWidth := 0
		requiredIndent := 36
		indent, flagWidth, noFlagWidth := 4, 12, 12
		yesPrefixLength, noPrefixLength := 2, 5

		flagString := fmt.Sprintf("%-*s", indent, "")
		flagString += fmt.Sprintf("--%-*s", flagWidth, f.Name)
		totalWidth += indent + yesPrefixLength + flagWidth
		if isBool {
			flagString += fmt.Sprintf("--no-%-*s", noFlagWidth, f.Name)
			totalWidth += noPrefixLength + noFlagWidth
		}
		remainingWidth := requiredIndent - totalWidth
		flagString += fmt.Sprintf("%-*s%s", remainingWidth, " ", HelpMessages[f.Name])
		if f.DefValue != "[]" {
			def := f.DefValue
			if f.Name == "output" {
				def = "stdout"
			}

			flagString += fmt.Sprintf(" (default is %v)", def)
		}
		println(flagString)
	})

}

func ParseFileStringFlag(v string) (*url.URL, error) {
	// TODO: actually handle windows paths as a separate thing, since this will only work
	// if run on the same mount as the file (or maybe only if run on C? not sure)
	if v == "" {
		return nil, errors.New("TODO: handle this case")
	}
	if !strings.Contains(v, "://") {
		absPath, err := filepath.Abs(v)
		volume := filepath.VolumeName(absPath)
		if volume != "" {
			// unix-like path, since the above is only true on win
			absPath = regexp.MustCompile(`^`+volume).ReplaceAllString(absPath, "")
			absPath = regexp.MustCompile(`\\`).ReplaceAllString(absPath, `/`)
		}
		if err != nil {
			return nil, err
		}
		v = "file://" + absPath
	}

	// url of some type maybe, let's try to parse it
	urlParsed, err := url.Parse(v)
	if err != nil {
		return nil, err
	}

	scheme := strings.ToLower(urlParsed.Scheme)
	if scheme == "" {
		urlParsed.Scheme = "file"
	}

	// supported schemes at the moment, file, http, https
	if slices.Contains([]string{"file", "http", "https"}, scheme) {
		return urlParsed, nil
	}
	return nil, errors.New("unsupported scheme, " + scheme + ", for url: " + v)
}
