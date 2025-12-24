package files

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"slices"
	"strings"

	"dario.cat/mergo"
	"github.com/adam-huganir/yutc/pkg/schema"
	"github.com/goccy/go-yaml"
	"github.com/pelletier/go-toml/v2"
	"github.com/rs/zerolog"
	"github.com/spf13/afero"
	"github.com/theory/jsonpath"
)

// MergeData merges data from a list of data files and returns a map of the merged data.
// The data is merged in the order of the data files, with later files overriding earlier ones.
// Supports files supported by ParseFileStringSource.
func MergeData(dataFiles []*FileArg, helmMode bool, logger *zerolog.Logger) (map[string]any, error) {
	var err error
	data := make(map[string]any)
	err = mergePaths(dataFiles, data, helmMode, logger)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func mergePaths(dataFiles []*FileArg, data map[string]any, helmMode bool, logger *zerolog.Logger) error {
	// since some of helms data structures are go structs, when the chart file is accessed through templates
	// it uses the struct casing rather than the yaml casing. this adjusts for that. for right now we only do this
	// for Chart
	specialHelmKeys := []string{"Chart"}

	toProcessData := make([]*FileArg, 0, len(dataFiles))
	toProcessSchema := make([]*FileArg, 0, len(dataFiles))
	for _, dataArg := range dataFiles {
		if dataArg.Kind == "schema" {
			toProcessSchema = append(toProcessSchema, dataArg)
		} else {
			toProcessData = append(toProcessData, dataArg)
		}
	}
	toProcess := slices.Concat(toProcessData, toProcessSchema)
	for _, dataArg := range toProcess {
		if dataArg.BasicAuth != "" || dataArg.BearerToken != "" {
			return fmt.Errorf("basic auth and bearer tokens are not yet implemented")
		}

		isDir, err := afero.IsDir(Fs, dataArg.Path)
		if err != nil {
			return err
		}
		if isDir {
			continue
		}
		source, err := ParseFileStringSource(dataArg.Path)
		if err != nil {
			return err
		}
		logger.Debug().Msg("Loading from " + source + " data file " + dataArg.Path + " with type " + dataArg.Kind)
		contentBuffer, err := GetDataFromPath(source, dataArg.Path, dataArg.BearerToken, dataArg.BasicAuth)
		if err != nil {
			return err
		}
		dataPartial := make(map[string]any)

		switch strings.ToLower(path.Ext(dataArg.Path)) {
		case ".toml":
			err = toml.Unmarshal(contentBuffer.Bytes(), &dataPartial)
		// originally i had used yaml to parse the json, but then thought that the expected behavior for giving invalid
		// json would be to fail, even if it was valid yaml
		case ".json":
			err = json.Unmarshal(contentBuffer.Bytes(), &dataPartial)
		default:
			err = yaml.Unmarshal(contentBuffer.Bytes(), &dataPartial)
		}
		if err != nil {
			return fmt.Errorf("unable to load data file %s: %w", dataArg.Path, err)
		}

		// If a top-level key is specified, nest the data under that key
		if dataArg.Key != "" && dataArg.Type != "schema" {
			_, err := jsonpath.Parse(checkPathPrefix(dataArg.Key))
			if err != nil {
				logger.Debug().Msg(fmt.Sprintf("Nesting data for %s under top-level key: %s", dataArg.Path, dataArg.Key))
				if helmMode && slices.Contains(specialHelmKeys, dataArg.Key) {
					logger.Debug().Msg(fmt.Sprintf("Applying helm key transformation for %s", dataArg.Key))
					dataPartial = KeysToPascalCase(dataPartial)
				}
				dataPartial = map[string]any{dataArg.Key: dataPartial}
			} else {
				logger.Debug().Msg(fmt.Sprintf("Nesting data for %s under path: %s", dataArg.Path, dataArg.Key))
				var dataPartialAny any
				dataPartialAny = dataPartial
				err = SetPath(&dataPartialAny, checkPathPrefix(dataArg.Key), dataPartial)
				if err != nil {
					return fmt.Errorf("unable to set path for %s: %w", dataArg.Path, err)
				}
			}
		}

		if dataArg.Kind == "schema" {
			schemaBytes, err := json.Marshal(dataPartial)
			if err != nil {
				return fmt.Errorf("unable to marshal schema %s: %w", dataArg.Path, err)
			}
			s, err := schema.LoadSchema(schemaBytes)
			if err != nil {
				return fmt.Errorf("unable to load schema %s: %w", dataArg.Path, err)
			}
			if dataArg.JSONPath != "" {
				s = schema.NestSchema(s, dataArg.JSONPath)
			}
			resolvedSchema, err := schema.ApplyDefaults(data, s)
			if err != nil {
				return fmt.Errorf("unable to resolve schema %s: %w", dataArg.Path, err)
			}
			err = resolvedSchema.Validate(data)
			if err != nil {
				return fmt.Errorf("unable to validate schema %s: %w", dataArg.Path, err)
			}
		} else {
			err = mergo.Merge(&data, dataPartial, mergo.WithOverride)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// LoadSharedTemplates reads from a list of shared template files and returns a list of buffers with the contents
func LoadSharedTemplates(templates []string, logger *zerolog.Logger) ([]*bytes.Buffer, error) {
	var sharedTemplateBuffers []*bytes.Buffer
	for _, template := range templates {
		isDir, err := afero.IsDir(Fs, template)
		if err != nil {
			return nil, err
		}
		if isDir {
			continue
		}
		source, err := ParseFileStringSource(template)
		if err != nil {
			return nil, err
		}
		logger.Debug().Msg("Loading from " + source + " shared template file " + template)
		contentBuffer, err := GetDataFromPath(source, template, "", "")
		if err != nil {
			return nil, err
		}
		sharedTemplateBuffers = append(sharedTemplateBuffers, contentBuffer)
	}
	return sharedTemplateBuffers, nil
}

// LoadTemplates resolves template paths and returns a sorted list of template file paths.
// It resolves directories, archives, and URLs to actual file paths and sorts them.
func LoadTemplates(
	templatePaths []string,
	tempDir string,
	logger *zerolog.Logger,
) (
	[]string,
	error,
) {
	templateFiles, err := ResolvePaths("", templatePaths, tempDir, logger)
	if err != nil {
		return nil, err
	}
	// this sort will help us later when we make assumptions about if folders already exist
	slices.Sort(templateFiles)

	logger.Debug().Msg(fmt.Sprintf("Found %d template files", len(templateFiles)))
	for _, templateFile := range templateFiles {
		logger.Trace().Msg("  - " + templateFile)
	}
	return templateFiles, nil
}

// LoadFiles resolves data file paths (directories, archives, URLs) to actual file paths.
// Returns an updated list of FileArg with resolved paths.
func LoadFiles(dataFiles []*FileArg, tempDir string, logger *zerolog.Logger) ([]*FileArg, error) {
	dataPathsOnly := make([]string, len(dataFiles))
	for idx, dataFile := range dataFiles {
		dataPathsOnly[idx] = dataFile.Path
	}
	paths, err := ResolvePaths("", dataPathsOnly, tempDir, logger)
	if err != nil {
		return nil, err
	}
	for idx, newPath := range paths {
		dataFiles[idx].Path = newPath
	}

	return dataFiles, nil
}

// ParseDataFiles parses raw data file arguments and populates the RunData structure.
func ParseDataFiles(files []*FileArg, dataFiles []string) ([]*FileArg, error) {
	df := make([]*FileArg, 0, len(dataFiles))
	for i, dataFileArg := range dataFiles {
		dataArg, err := ParseFileArg(dataFileArg)
		if err != nil {
			return nil, err
		}
		files[i] = dataArg
	}
	return df, nil
}

type FileContent struct {
	Filename string // name of file, either from path or url, or '-' for stdin
	Mimetype string // mimetype if known
	Data     []byte // contents of file gathered during load/download, nil'd once on disk
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
