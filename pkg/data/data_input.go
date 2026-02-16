package data

import (
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strings"

	"dario.cat/mergo"
	"github.com/adam-huganir/yutc/pkg/loader"
	"github.com/adam-huganir/yutc/pkg/schema"
	"github.com/goccy/go-yaml"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/pelletier/go-toml/v2"
	"github.com/rs/zerolog"
	"github.com/theory/jsonpath"
)

// SchemaInfo holds configuration for schema validation.
type SchemaInfo struct {
	DisableDefaults bool // For schema files: skip applying defaults but still validate
}

// Input represents a data file (yaml/json/toml) or schema file for template merging.
type Input struct {
	*loader.FileEntry
	JSONPath *jsonpath.Path // Optional top-level key to nest the data under
	Schema   SchemaInfo
	IsSchema bool // true if this is a schema file rather than a data file
}

// InputOption is a functional option for configuring an Input.
type InputOption func(*Input)

// WithJSONPath sets the JSONPath for data nesting.
func WithJSONPath(jp *jsonpath.Path) InputOption {
	return func(di *Input) {
		di.JSONPath = jp
	}
}

// WithDefaultJSONPath sets the JSONPath to "$" if not already set.
func WithDefaultJSONPath() InputOption {
	return func(di *Input) {
		if di.JSONPath == nil {
			di.JSONPath = jsonpath.MustParse("$")
		}
	}
}

// AsSchema marks this Input as a schema file.
func AsSchema() InputOption {
	return func(di *Input) {
		di.IsSchema = true
	}
}

// NewInput creates an Input with the given name, FileEntry options, and Input options.
// By default, sets JSONPath to "$".
func NewInput(name string, entryOpts []loader.FileEntryOption, dataOpts ...InputOption) *Input {
	fe := loader.NewFileEntry(name, entryOpts...)
	di := &Input{
		FileEntry: fe,
		JSONPath:  jsonpath.MustParse("$"),
	}
	for _, opt := range dataOpts {
		opt(di)
	}
	return di
}

func unmarshalToMap(name string, data []byte) (map[string]any, error) {
	fileData := make(map[string]any)
	switch strings.ToLower(path.Ext(name)) {
	case ".toml":
		if err := toml.Unmarshal(data, &fileData); err != nil {
			return nil, err
		}
	case ".json":
		if err := json.Unmarshal(data, &fileData); err != nil {
			return nil, err
		}
	default:
		if err := yaml.Unmarshal(data, &fileData); err != nil {
			return nil, err
		}
	}
	return fileData, nil
}

// MergeInto loads and merges this data file into the destination map.
func (di *Input) MergeInto(dst map[string]any, helmMode bool, specialHelmKeys []string, logger *zerolog.Logger) error {
	if di.Content == nil || !di.Content.Read {
		err := di.Load()
		if err != nil {
			return err
		}
	}
	fileData, err := unmarshalToMap(di.Name, di.Content.Data)
	if err != nil {
		return fmt.Errorf("unable to load data file %s: %w", di.Name, err)
	}

	dataPartial := fileData
	if di.JSONPath != nil && di.JSONPath.String() != "$" {
		q := di.JSONPath.Query()
		segments := di.JSONPath.Query().Segments()
		firstKey := ""
		if err = json.Unmarshal([]byte(segments[0].Selectors()[0].String()), &firstKey); err != nil {
			return fmt.Errorf("unable to parse first key for %s: %w", di.Name, err)
		}

		logger.Debug().Msg(fmt.Sprintf("Nesting data for %s under top-level key: %s", di.Name, q.String()))
		if helmMode && len(segments) == 1 && slices.Contains(specialHelmKeys, firstKey) {
			logger.Debug().Msg(fmt.Sprintf("Applying helm key transformation for %s", di.Name))
			fileData = KeysToPascalCase(fileData)
		}
		partial := make(map[string]any)
		partialAny := any(partial)
		err = SetPath(&partialAny, di.JSONPath.String(), fileData)
		if err != nil {
			return fmt.Errorf("unable to set path for %s: %w", di.Name, err)
		}
		var ok bool
		dataPartial, ok = partialAny.(map[string]any)
		if !ok {
			return fmt.Errorf("unable to set path for %s: expected map at root, got %T", di.Name, partialAny)
		}
	}

	err = mergo.Merge(&dst, dataPartial, mergo.WithOverride)
	if err != nil {
		return err
	}
	return nil
}

// ApplySchemaTo validates and optionally applies defaults from this schema to the data.
func (di *Input) ApplySchemaTo(data map[string]any) error {
	if di.Content == nil || !di.Content.Read {
		err := di.Load()
		if err != nil {
			return err
		}
	}
	fileData, err := unmarshalToMap(di.Name, di.Content.Data)
	if err != nil {
		return fmt.Errorf("unable to load data file %s: %w", di.Name, err)
	}
	schemaBytes, err := json.Marshal(fileData)
	if err != nil {
		return fmt.Errorf("unable to marshal schema %s: %w", di.Name, err)
	}
	s, err := schema.LoadSchema(schemaBytes)
	if err != nil {
		return fmt.Errorf("unable to load schema %s: %w", di.Name, err)
	}
	if di.JSONPath != nil && di.JSONPath.String() != "$" {
		s = schema.NestSchema(s, di.JSONPath.String())
	}
	var resolvedSchema *jsonschema.Resolved
	if di.Schema.DisableDefaults {
		resolvedSchema, err = s.Resolve(&jsonschema.ResolveOptions{ValidateDefaults: false})
		if err != nil {
			return fmt.Errorf("unable to resolve schema %s: %w", di.Name, err)
		}
	} else {
		resolvedSchema, err = schema.ApplyDefaults(data, s)
		if err != nil {
			return fmt.Errorf("unable to resolve schema %s: %w", di.Name, err)
		}
	}

	err = resolvedSchema.Validate(data)
	if err != nil {
		return fmt.Errorf("unable to validate schema %s: %w", di.Name, err)
	}
	return nil
}

// MergeDataFiles merges data from a list of Input and returns a map of the merged data.
// The data is merged in the order of the inputs, with later data overriding earlier ones.
// Schema inputs are applied after all data and --set args are merged.
func MergeDataFiles(dataFiles []*Input, setArgs []string, helmMode bool, logger *zerolog.Logger) (data map[string]any, err error) {
	data = make(map[string]any)
	// since some of helms data structures are go structs, when the chart file is accessed through templates
	// it uses the struct casing rather than the yaml casing. this adjusts for that. for right now we only do this
	// for Chart
	specialHelmKeys := []string{"Chart"}

	// order data and schema files so that schemas are processed last, and can be applied
	// to the fully merged data
	toProcessData := make([]*Input, 0, len(dataFiles))
	toProcessSchema := make([]*Input, 0, len(dataFiles))
	for _, dataArg := range dataFiles {
		if dataArg.IsSchema {
			toProcessSchema = append(toProcessSchema, dataArg)
		} else {
			toProcessData = append(toProcessData, dataArg)
		}
	}

	processDataInput := func(dataArg *Input) error {
		isDir, err := IsDir(dataArg.Name)
		if err != nil {
			return err
		}
		if isDir {
			return nil
		}
		source := dataArg.Source
		if source == "" {
			source, err = ParseFileStringSource(dataArg.Name)
			if err != nil {
				return err
			}
		}
		logger.Debug().Msgf("Loading from %s data file %s (schema=%v)", source, dataArg.Name, dataArg.IsSchema)

		if dataArg.IsSchema {
			err = dataArg.ApplySchemaTo(data)
			if err != nil {
				return err
			}
		} else {
			err = dataArg.MergeInto(data, helmMode, specialHelmKeys, logger)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, dataArg := range toProcessData {
		err = processDataInput(dataArg)
		if err != nil {
			return data, err
		}
	}

	err = applySetArgs(data, setArgs, logger)
	if err != nil {
		return data, err
	}

	for _, dataArg := range toProcessSchema {
		err = processDataInput(dataArg)
		if err != nil {
			return data, err
		}
	}
	return data, nil
}
