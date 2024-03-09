package internal

import (
	"errors"
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

func ParseStringFlag(v string) (*url.URL, error) {
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
