package internal

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
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

//func (sf *StringFlag) GetType() types.Type {
//	return types.Type(types.String)
//}

type CLIOptions struct {
	Stdin           bool     `json:"stdin"`
	DataFiles       []string `json:"data-files"`
	TemplateFiles   []string `json:"template-files"`
	Output          string   `json:"output"`
	Overwrite       bool     `json:"overwrite"`
	SharedTemplates []string `json:"shared-templates"`
	StdinFirst      bool     `json:"stdin-first"`
}

var HelpMessages = map[string]string{
	"stdin":       "Read data from stdin",
	"stdin-first": "Read data from stdin before merging with specified data files",
	"overwrite":   "Overwrite existing files",
	"data":        "Data file to parse and merge. Can be a file or a URL. Can be specified multiple times and the inputs will be merged.",
	"shared":      "Templates to be shared across all arguments in template list. Can be a file or a URL. Can be specified multiple times.",
	"template":    "Template file to execute, Can be a file or a URL. Can be specified multiple times for multiple file outputs",
	"output":      "Output file/directory, defaults to stdout",
	"version":     "Print the version and exit",
}

const targetTerminalWidth = 100

func OverComplicatedHelp() {
	println("Usage: yutc [flags] <template ...>\n")
	println("  Options:")
	flag.VisitAll(func(f *flag.Flag) {
		isBool := f.Value.String() == "true" || f.Value.String() == "false"
		skip := strings.HasPrefix(f.Name, "no-") && isBool
		if skip {
			return
		}
		textIndent := 16
		flagIndent := 2
		flagPrefix := "--"

		flagString := fmt.Sprintf("%-*s", flagIndent, "")
		flagString += fmt.Sprintf("%s%-*s", flagPrefix, textIndent-len(flagPrefix)-flagIndent, f.Name)
		helpTokens := strings.Split(HelpMessages[f.Name], " ")
		if f.DefValue != "[]" {
			def := f.DefValue
			if f.Name == "output" {
				def = "stdout"
			}
			defaultTokens := strings.Split(fmt.Sprintf(" (default is %v)", def), " ")
			helpTokens = slices.Concat(helpTokens, defaultTokens)
		}
		for len(helpTokens) > 0 {
			currentWidth := len(flagString)
			if len(helpTokens[0]) > targetTerminalWidth-currentWidth {
				println(flagString)
				flagString = fmt.Sprintf("%-*s%s ", textIndent, "", helpTokens[0])
			} else {
				flagString += helpTokens[0] + " "
			}
			helpTokens = helpTokens[1:]
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

func ValidateArguments(
	stdin,
	stdinFirst,
	overwrite bool,
	sharedTemplates,
	dataFiles,
	templateFiles []string,
	output string,
) {
	var err error
	var errs []error
	var code, v int64

	if len(templateFiles) == 0 {
		err = errors.New("must provide at least one template file")
		v, _ = strconv.ParseInt("1", 2, 64)
		code += v
		errs = append(errs, err)
	}

	outputFiles := output != ""
	if !outputFiles && len(templateFiles) > 1 {
		err = errors.New("cannot use `stdout` with multiple template files")
		v, _ = strconv.ParseInt("100", 2, 64)
		code += v
		errs = append(errs, err)
	}

	if !outputFiles {
		_, err = os.Stat(output)
		if err != nil {
			if os.IsNotExist(err) && len(templateFiles) > 1 {
				err = errors.New("folder " + output + " does not exist to generate multiple templates")
				v, _ = strconv.ParseInt("1000", 2, 64)
				code += v
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		for _, err := range errs {
			logger.Error(err.Error())
		}
		os.Exit(int(code))
	}
}
