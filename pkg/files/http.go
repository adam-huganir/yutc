package files

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// ReadURL fetches a file from a URL and returns the filename, data, MIME type, and any error.
// It attempts to extract the filename from Content-Disposition header or falls back to the URL path.
func ReadURL(f *FileArg, logger *zerolog.Logger) (err error) {
	var mediaKV map[string]string
	var mimetype string

	f.Url, err = url.Parse(f.Path)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("GET", f.Url.String(), nil)
	if err != nil {
		return err
	}

	// note: this will override any basicauth int he url
	// note: basicauth and bearer tokens are mutually exclusive, and basicauth will take precedence over bearer tokens
	if f.BasicAuth != "" {
		auth := strings.Split(f.BasicAuth, ":")
		if len(auth) != 2 {
			return fmt.Errorf("basic auth must be in username:password format")
		}
		req.SetBasicAuth(auth[0], auth[1])
	} else if f.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+f.BearerToken)
	}
	client := http.Client{Timeout: time.Second * 30}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return NewHTTPStatusError(resp)
	}
	contentDisposition := resp.Header.Get("Content-Disposition")
	if contentDisposition != "" {
		mimetype, mediaKV, err = mime.ParseMediaType(contentDisposition)
		if err != nil {
			logger.Fatal().Msg(err.Error())
		}
		if _, ok := mediaKV["filename"]; ok {
			f.Content.Filename = mediaKV["filename"]
		}
	} else {
		mimetype = resp.Header.Get("Content-Type")
		mimetype, mediaKV, err = mime.ParseMediaType(mimetype)
		if _, ok := mediaKV["filename"]; ok {
			f.Content.Filename = mediaKV["filename"]
		} else {
			f.Content.Filename = filepath.Base(f.Path)
		}
	}

	f.Content.Data, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if mimetype == "" {
		mimetype = http.DetectContentType(f.Content.Data[:512]) // 512 is max of function anyways
		mimetype, _, err = mime.ParseMediaType(mimetype)
		if err != nil {
			return err
		}
	}
	f.Content.Mimetype = mimetype

	return err
}
