package data

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"dario.cat/mergo"
	"github.com/adam-huganir/yutc/pkg/schema"
	"github.com/goccy/go-yaml"
	"github.com/pelletier/go-toml/v2"
	"github.com/rs/zerolog"
	"github.com/theory/jsonpath"
)

// NormalizeFilepath cleans and normalizes a file path to use forward slashes.
func NormalizeFilepath(file string) string {
	return filepath.ToSlash(filepath.Clean(path.Join(file)))
}

// MergeData merges data from a list of data and returns a map of the merged data.
// The data is merged in the order of the data, with later data overriding earlier ones.
func MergeData(dataFiles []*FileArg, helmMode bool, logger *zerolog.Logger) (data map[string]any, err error) {
	// since some of helms data structures are go structs, when the chart file is accessed through templates
	// it uses the struct casing rather than the yaml casing. this adjusts for that. for right now we only do this
	// for Chart
	specialHelmKeys := []string{"Chart"}

	// order data and schema files so that schemas are processed last, and can be applied
	// to the fully merged data
	toProcessData := make([]*FileArg, 0, len(dataFiles))
	toProcessSchema := make([]*FileArg, 0, len(dataFiles))
	for _, dataArg := range dataFiles {
		switch dataArg.Kind {
		case "schema":
			toProcessSchema = append(toProcessSchema, dataArg)
		default:
			toProcessData = append(toProcessData, dataArg)
		}
	}
	toProcess := slices.Concat(toProcessData, toProcessSchema)

	for _, dataArg := range toProcess {
		isDir, err := IsDir(dataArg.Path)
		if err != nil {
			return data, err
		}
		if isDir {
			continue
		}
		source, err := ParseFileStringSource(dataArg.Path)
		if err != nil {
			return data, err
		}
		logger.Debug().Msgf("Loading from %s data file %s with type %s", source, dataArg.Path, dataArg.Kind)

		if !dataArg.Content.Read {
			err = dataArg.Load()
			if err != nil {
				return data, err
			}
		}
		dataPartial := make(map[string]any)
		switch strings.ToLower(path.Ext(dataArg.Path)) {
		case ".toml":
			err = toml.Unmarshal(dataArg.Content.Data, &dataPartial)
		// originally i had used yaml to parse the json, but then thought that the expected behavior for giving invalid
		// json would be to fail, even if it was valid yaml
		case ".json":
			err = json.Unmarshal(dataArg.Content.Data, &dataPartial)
		default:
			err = yaml.Unmarshal(dataArg.Content.Data, &dataPartial)
		}
		if err != nil {
			return data, fmt.Errorf("unable to load data file %s: %w", dataArg.Path, err)
		}

		// If a top-level key is specified, nest the data under that key
		if dataArg.JSONPath != nil && dataArg.JSONPath.String() != "$" && dataArg.Kind != "schema" {
			_, err := jsonpath.Parse(checkPathPrefix(dataArg.JSONPath.String()))
			if err != nil {
				logger.Debug().Msg(fmt.Sprintf("Nesting data for %s under top-level key: %s", dataArg.Path, dataArg.JSONPath.String()))
				if helmMode && slices.Contains(specialHelmKeys, dataArg.JSONPath.String()) {
					logger.Debug().Msg(fmt.Sprintf("Applying helm key transformation for %s", dataArg.JSONPath.String()))
					dataPartial = KeysToPascalCase(dataPartial)
				}
				dataPartial = map[string]any{dataArg.JSONPath.String(): dataPartial}
			} else {
				logger.Debug().Msg(fmt.Sprintf("Nesting data for %s under path: %s", dataArg.Path, dataArg.JSONPath.String()))
				var dataPartialAny any
				dataPartialAny = dataPartial
				err = SetPath(&dataPartialAny, checkPathPrefix(dataArg.JSONPath.String()), dataPartial)
				if err != nil {
					return data, fmt.Errorf("unable to set path for %s: %w", dataArg.Path, err)
				}
			}
		}

		if dataArg.Kind == "schema" {
			schemaBytes, err := json.Marshal(dataPartial)
			if err != nil {
				return data, fmt.Errorf("unable to marshal schema %s: %w", dataArg.Path, err)
			}
			s, err := schema.LoadSchema(schemaBytes)
			if err != nil {
				return data, fmt.Errorf("unable to load schema %s: %w", dataArg.Path, err)
			}
			if dataArg.JSONPath.String() != "$" {
				s = schema.NestSchema(s, dataArg.JSONPath.String())
			}
			resolvedSchema, err := schema.ApplyDefaults(data, s)
			if err != nil {
				return data, fmt.Errorf("unable to resolve schema %s: %w", dataArg.Path, err)
			}
			err = resolvedSchema.Validate(data)
			if err != nil {
				return data, fmt.Errorf("unable to validate schema %s: %w", dataArg.Path, err)
			}
		} else {
			err = mergo.Merge(&data, dataPartial, mergo.WithOverride)
			if err != nil {
				return data, err
			}
		}
	}
	return data, nil
}

// LoadSharedTemplates reads from a list of shared template data and returns a list of buffers with the contents
//func LoadSharedTemplates(templates []string, logger *zerolog.Logger) ([]*bytes.Buffer, error) {
//	var sharedTemplateBuffers []*bytes.Buffer
//	for _, template := range templates {
//		isDir, err := afero.IsDir(Fs, template)
//		if err != nil {
//			return nil, err
//		}
//		if isDir {
//			continue
//		}
//		source, err := ParseFileStringSource(template)
//		if err != nil {
//			return nil, err
//		}
//		logger.Debug().Msg("Loading from " + source + " shared template file " + template)
//		contentBuffer, err := GetDataFromPath(source, template, "", "")
//		if err != nil {
//			return nil, err
//		}
//		sharedTemplateBuffers = append(sharedTemplateBuffers, contentBuffer)
//	}
//	return sharedTemplateBuffers, nil
//}

//// LoadTemplates resolves template paths and returns a sorted list of template file paths.
//// It resolves directories, archives, and URLs to actual file paths and sorts them.
//func LoadTemplates(
//	templatePaths []string,
//	tempDir string,
//	logger *zerolog.Logger,
//) (
//	[]string,
//	error,
//) {
//	templateFiles, err := ResolvePaths("", templatePaths, tempDir, logger)
//	if err != nil {
//		return nil, err
//	}
//	// this sort will help us later when we make assumptions about if folders already exist
//	slices.Sort(templateFiles)
//
//	logger.Debug().Msg(fmt.Sprintf("Found %d template data", len(templateFiles)))
//	for _, templateFile := range templateFiles {
//		logger.Trace().Msg("  - " + templateFile)
//	}
//	return templateFiles, nil
//}
//
//// LoadFiles resolves data file paths (directories, archives, URLs) to actual file paths.
//// Returns an updated list of FileArg with resolved paths.
//func LoadFiles(files []string, kind, tempDir string, logger *zerolog.Logger) ([]*FileArg, error) {
//	fileArgs, err := ResolvePaths(files, kind, tempDir, logger)
//	if err != nil {
//		return nil, err
//	}
//	return fileArgs, nil
//}

const (
	dataPreallocate = 1024 * 8
)

type FileContent struct {
	Filename string // name of file, either from path or url, or '-' for stdin
	Mimetype string // mimetype if known
	Data     []byte // contents of file gathered during load/download
	Read     bool   // whether the file has been read into memory
}

func NewFileContent() *FileContent {
	// keep a few k around to start, this may end up being an issue at scale but probably not for most use cases
	b := make([]byte, 0, dataPreallocate)
	return &FileContent{Data: b}
}

// FileArg represents a parsed data file argument with optional top-level key
type FileArg struct {
	JSONPath    *jsonpath.Path // Optional top-level key to nest the data under
	Path        string         // File path, URL, or "-" for stdin
	Url         *url.URL       // URL for http call if the source is a url
	Kind        string         // Optional type of data, either "schema" or "data" or "template" or not provided
	Source      string         // Optional source of data, either "file", "url", or "stdin"
	BearerToken string         // Bearer token for http call. just token, not "Bearer "
	BasicAuth   string         // Basic auth for http call in username:password format
	Content     *FileContent   // Content of the file
	logger      *zerolog.Logger
}

func NewFileArgFile(path, kind string) FileArg {
	nop := zerolog.Nop()
	fa := FileArg{
		Path:    path,
		Kind:    kind,
		Source:  "file",
		Content: NewFileContent(),
		logger:  &nop,
	}
	fa.NormalizePath()
	return fa
}

func NewFileArgURL(path, kind string) FileArg {
	nop := zerolog.Nop()
	return FileArg{
		Path:    path,
		Kind:    kind,
		Source:  "url",
		Content: NewFileContent(),
		logger:  &nop,
	}
}

func NewFileArgStdin(kind string) FileArg {
	nop := zerolog.Nop()
	return FileArg{
		Path:    "-",
		Kind:    kind,
		Source:  "stdin",
		Content: NewFileContent(),
		logger:  &nop,
	}
}

func (f *FileArg) SetLogger(logger *zerolog.Logger) {
	if logger == nil {
		nop := zerolog.Nop()
		logger = &nop
	}
	f.logger = logger
}

func (f *FileArg) String() string {
	return fmt.Sprintf("FileArg{Path: %s, Source: %s, BearerToken: %s, BasicAuth: %s, Content: %v}", f.Path, f.Source, f.BearerToken, f.BasicAuth, f.Content)
}

// GetContents returns the contents of the file, reading from disk if necessary
func (f *FileArg) GetContents() ([]byte, error) {
	if f.Source != "file" {
		return nil, fmt.Errorf("file %s is not a file", f.Path)
	}
	if f.Content.Data != nil {
		return f.Content.Data, nil
	}
	exists, err := Exists(f.Path)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("file %s does not exist", f.Path)
	}
	contents, err := os.ReadFile(f.Path)
	if err != nil {
		return nil, err
	}
	return contents, nil
}

func (f *FileArg) NormalizePath() {
	f.Path = NormalizeFilepath(f.Path)
}

func (f *FileArg) Load() (err error) {
	if f.Content.Read {
		return nil
	}
	switch f.Source {
	case "file":
		err := f.ReadFile()
		if err != nil {
			return err
		}
	case "url":
		err = f.ReadURL(f.logger)
		if err != nil {
			return err
		}
	case "stdin":
		err = f.ReadStdin()
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown source %s", f.Source)
	}
	return nil
}

func (f *FileArg) ReadStdin() (err error) {
	buf, err := GetDataFromReadCloser(os.Stdin)
	if err != nil {
		return err
	}
	f.Content.Filename = "-"
	f.Content.Data = buf.Bytes()
	f.Content.Read = true
	if f.Path != "-" {
		panic("a bug yo")
	}
	mimetype, err := getMimetype(f.Content.Data)
	if err != nil {
		return err
	}
	f.Content.Mimetype = mimetype
	return nil
}

func (f *FileArg) ReadFile() (err error) {
	f.Content.Data, err = os.ReadFile(f.Path)
	if err != nil {
		return err
	}
	f.Content.Read = true
	f.Content.Filename = filepath.Base(f.Path)
	mimetype, err := getMimetype(f.Content.Data)
	// TODO: mimetype from file extension
	if err != nil {
		return err
	}
	f.Content.Mimetype = mimetype
	return nil
}

func getMimetype(data []byte) (mimetype string, err error) {
	mimetype = http.DetectContentType(data[:512]) // 512 is max of function anyways
	mimetype, _, err = mime.ParseMediaType(mimetype)
	if err != nil {
		return "", err
	}
	return mimetype, err
}

// ReadURL fetches a file from a URL and returns the filename, data, MIME type, and any error.
// It attempts to extract the filename from Content-Disposition header or falls back to the URL path.
func (f *FileArg) ReadURL(logger *zerolog.Logger) (err error) {

	if f.Source != "url" {
		return fmt.Errorf("file %s is not a url", f.Path)
	}

	var mediaKV map[string]string
	var mimetype string
	resp, err := GetURL(f.Url, f.BasicAuth, f.BearerToken)
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if err != nil {
		return err
	}
	f.Content.Data, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	f.Content.Read = true
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

func (f *FileArg) IsDir() (bool, error) {
	return IsDir(f.Path)
}

func (f *FileArg) IsArchive() bool {
	return IsArchive(f.Path)
}

func (f *FileArg) IsText() bool {
	if f.Content.Mimetype == "" {
		err := f.Load()
		if err != nil {
			f.logger.Error().Err(err).Msgf("Failed to load file %s with %v", f.Path, err)
			return false
		}
		// TODO: add support for other some other less common encodings.
		if !utf8.Valid(f.Content.Data) {
			return false
		}
	}
	return strings.Contains(f.Content.Mimetype, "text")
}

func GetURL(url *url.URL, basicAuth, bearerToken string) (data *http.Response, err error) {
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}

	// note: this will override any basicauth int he url
	// note: basicauth and bearer tokens are mutually exclusive, and basicauth will take precedence over bearer tokens
	if basicAuth != "" {
		auth := strings.Split(basicAuth, ":")
		if len(auth) != 2 {
			return nil, fmt.Errorf("basic auth must be in username:password format")
		}
		req.SetBasicAuth(auth[0], auth[1])
	} else if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	client := http.Client{Timeout: time.Second * 30}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return resp, NewHTTPStatusError(resp)
	}
	return resp, nil
}
