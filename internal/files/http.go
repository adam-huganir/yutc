package files

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/rs/zerolog"
)

func ReadUrl(templatePath string, logger zerolog.Logger) (string, []byte, string, error) {
	var filename, mimetype string
	var mediaKV map[string]string
	resp, err := http.Get(templatePath)
	if err != nil {
		logger.Fatal().Msg(err.Error())
	} else if resp.StatusCode != http.StatusOK {
		return "", nil, "", NewHttpStatusError(resp)
	}
	contentDisposition := resp.Header.Get("Content-Disposition")
	if contentDisposition != "" {
		mimetype, mediaKV, err = mime.ParseMediaType(contentDisposition)
		if err != nil {
			logger.Fatal().Msg(err.Error())
		}
		if _, ok := mediaKV["filename"]; ok {
			filename = mediaKV["filename"]
		}
	} else {
		mimetype = resp.Header.Get("Content-Type")
		mimetype, mediaKV, err = mime.ParseMediaType(mimetype)
		if _, ok := mediaKV["filename"]; ok {
			filename = mediaKV["filename"]
		} else {
			filename = filepath.Base(templatePath)
		}
	}
	data, errRaw := io.ReadAll(resp.Body)
	if errRaw != nil {
		return "", nil, "", err
	}
	if mimetype == "" {
		mimetype = http.DetectContentType(data[:512]) // 512 is max of function anyways
		mimetype, _, _ = mime.ParseMediaType(mimetype)
	}
	_ = resp.Body.Close()
	return filename, data, mimetype, nil
}
