package data

import (
	"bytes"
	"encoding/json"
	"errors"
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
	"text/template"
	"time"
	"unicode/utf8"

	"dario.cat/mergo"
	"github.com/adam-huganir/yutc/pkg/schema"
	"github.com/goccy/go-yaml"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/pelletier/go-toml/v2"
	"github.com/rs/zerolog"
	"github.com/theory/jsonpath"
)

// NormalizeFilepath cleans and normalizes a file path to use forward slashes.
func NormalizeFilepath(file string) string {
	return filepath.ToSlash(filepath.Clean(path.Join(file)))
}

func applySetArgs(dst *map[string]any, setArgs []string, logger *zerolog.Logger) error {
	if len(setArgs) == 0 {
		return nil
	}

	mergedDataAny := any(*dst)
	for _, ss := range setArgs {
		pathExpr, value, err := SplitSetString(ss)
		if err != nil {
			return fmt.Errorf("error parsing --set value '%s': %w", ss, err)
		}
		parsed, err := jsonpath.Parse(pathExpr)
		if err != nil {
			return fmt.Errorf("error parsing --set value '%s': %w", ss, err)
		}
		if pq := parsed.Query().Singular(); pq == nil {
			return fmt.Errorf("error parsing --set value '%s': resulting path is not unique singular path", ss)
		}
		err = SetValueInData(&mergedDataAny, parsed.Query().Segments(), value, ss)
		if err != nil {
			return err
		}
		if logger != nil {
			logger.Debug().Msgf("set %s to %v", parsed, value)
		}
	}

	mergedData, ok := mergedDataAny.(map[string]any)
	if !ok {
		return fmt.Errorf("error applying --set values: expected map at root, got %T", mergedDataAny)
	}
	*dst = mergedData
	return nil
}

// MergeDataFiles merges data from a list of data and returns a map of the merged data.
// The data is merged in the order of the data, with later data overriding earlier ones.
func MergeDataFiles(dataFiles []*FileArg, setArgs []string, helmMode bool, logger *zerolog.Logger) (data map[string]any, err error) {
	data = make(map[string]any)
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
		case FileKindSchema:
			toProcessSchema = append(toProcessSchema, dataArg)
		default:
			toProcessData = append(toProcessData, dataArg)
		}
	}

	processFileArg := func(dataArg *FileArg) error {
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
		logger.Debug().Msgf("Loading from %s data file %s with type %s", source, dataArg.Name, dataArg.Kind)

		switch dataArg.Kind {
		case FileKindSchema:
			err = dataArg.ApplySchemaTo(data)
			if err != nil {
				return err
			}
		default:
			err = dataArg.MergeInto(&data, helmMode, specialHelmKeys, logger)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, dataArg := range toProcessData {
		err = processFileArg(dataArg)
		if err != nil {
			return data, err
		}
	}

	err = applySetArgs(&data, setArgs, logger)
	if err != nil {
		return data, err
	}

	for _, dataArg := range toProcessSchema {
		err = processFileArg(dataArg)
		if err != nil {
			return data, err
		}
	}
	return data, nil
}

func unmarshalFileArgToMap(f *FileArg) (map[string]any, error) {
	fileData := make(map[string]any)
	switch strings.ToLower(path.Ext(f.Name)) {
	case ".toml":
		if err := toml.Unmarshal(f.Content.Data, &fileData); err != nil {
			return nil, err
		}
	case ".json":
		if err := json.Unmarshal(f.Content.Data, &fileData); err != nil {
			return nil, err
		}
	default:
		if err := yaml.Unmarshal(f.Content.Data, &fileData); err != nil {
			return nil, err
		}
	}
	return fileData, nil
}

func (f *FileArg) MergeInto(dst *map[string]any, helmMode bool, specialHelmKeys []string, logger *zerolog.Logger) error {
	if f.Content == nil || !f.Content.Read {
		err := f.Load()
		if err != nil {
			return err
		}
	}
	fileData, err := unmarshalFileArgToMap(f)
	if err != nil {
		return fmt.Errorf("unable to load data file %s: %w", f.Name, err)
	}

	dataPartial := fileData
	if f.JSONPath != nil && f.JSONPath.String() != "$" {
		q := f.JSONPath.Query()
		segments := f.JSONPath.Query().Segments()
		firstKey := ""
		if err = json.Unmarshal([]byte(segments[0].Selectors()[0].String()), &firstKey); err != nil {
			return fmt.Errorf("unable to parse first key for %s: %w", f.Name, err)
		}

		logger.Debug().Msg(fmt.Sprintf("Nesting data for %s under top-level key: %s", f.Name, q.String()))
		if helmMode && len(segments) == 1 && slices.Contains(specialHelmKeys, firstKey) {
			logger.Debug().Msg(fmt.Sprintf("Applying helm key transformation for %s", f.Name))
			fileData = KeysToPascalCase(fileData)
		}
		partial := make(map[string]any)
		partialAny := any(partial)
		err = SetPath(&partialAny, f.JSONPath.String(), fileData)
		if err != nil {
			return fmt.Errorf("unable to set path for %s: %w", f.Name, err)
		}
		var ok bool
		dataPartial, ok = partialAny.(map[string]any)
		if !ok {
			return fmt.Errorf("unable to set path for %s: expected map at root, got %T", f.Name, partialAny)
		}
	}

	err = mergo.Merge(dst, dataPartial, mergo.WithOverride)
	if err != nil {
		return err
	}
	return nil
}

func (f *FileArg) ApplySchemaTo(data map[string]any) error {
	if f.Content == nil || !f.Content.Read {
		err := f.Load()
		if err != nil {
			return err
		}
	}
	fileData, err := unmarshalFileArgToMap(f)
	if err != nil {
		return fmt.Errorf("unable to load data file %s: %w", f.Name, err)
	}
	schemaBytes, err := json.Marshal(fileData)
	if err != nil {
		return fmt.Errorf("unable to marshal schema %s: %w", f.Name, err)
	}
	s, err := schema.LoadSchema(schemaBytes)
	if err != nil {
		return fmt.Errorf("unable to load schema %s: %w", f.Name, err)
	}
	if f.JSONPath != nil && f.JSONPath.String() != "$" {
		s = schema.NestSchema(s, f.JSONPath.String())
	}
	var resolvedSchema *jsonschema.Resolved
	if f.Schema.DisableDefaults {
		resolvedSchema, err = s.Resolve(&jsonschema.ResolveOptions{ValidateDefaults: false})
		if err != nil {
			return fmt.Errorf("unable to resolve schema %s: %w", f.Name, err)
		}
	} else {
		resolvedSchema, err = schema.ApplyDefaults(data, s)
		if err != nil {
			return fmt.Errorf("unable to resolve schema %s: %w", f.Name, err)
		}
	}

	err = resolvedSchema.Validate(data)
	if err != nil {
		return fmt.Errorf("unable to validate schema %s: %w", f.Name, err)
	}
	return nil
}

const (
	dataPreallocate = 1024 * 8
)

type FileKind string

const (
	FileKindData           FileKind = "data"
	FileKindSchema         FileKind = "schema"
	FileKindTemplate       FileKind = "template"
	FileKindCommonTemplate FileKind = "common"
)

func (fk FileKind) String() string {
	return string(fk)
}

type SourceKind string

const (
	SourceKindFile   SourceKind = "file"
	SourceKindURL    SourceKind = "url"
	SourceKindStdin  SourceKind = "stdin"
	SourceKindStdout SourceKind = "stdout"
)

func (sk SourceKind) String() string {
	return string(sk)
}

type FileContent struct {
	Filename string // name of file, either from path or url, or '-' for stdin
	Mimetype string // mimetype if known
	Data     []byte // contents of file gathered during load/download
	Read     bool   // whether the file has been read into memory
}

type TemplateInfo struct {
	NewName string // For templates, if we are renaming the file, this is the new name
}

type ContainerInfo struct {
	Parent   *FileArg   // Parent of the file if it is a directory or archive
	Root     *FileArg   // Root of the file if it is a directory or archive
	children []*FileArg // Children of the file if it is a directory or archive
}

type RemoteInfo struct {
	URL      *url.URL       // URL for http call if the source is a url
	Response *http.Response // Response from http call if the source is a url
}

type AuthInfo struct {
	BearerToken string // Bearer token for http call. just token, not "Bearer "
	BasicAuth   string // Basic auth for http call in username:password format
}

type SchemaInfo struct {
	DisableDefaults bool // For schema files: skip applying defaults but still validate
}

func NewFileContent() *FileContent {
	// keep a few k around to start, this may end up being an issue at scale but probably not for most use cases
	b := make([]byte, 0, dataPreallocate)
	return &FileContent{Data: b}
}

// FileArg represents a parsed data file argument with optional top-level key
type FileArg struct {
	// Path variables for keeping track of where things come from, any transformations
	// applied, etc.
	Name     string // File path, URL, or "-" for stdin
	Template TemplateInfo

	Container ContainerInfo
	JSONPath  *jsonpath.Path // Optional top-level key to nest the data under
	Remote    RemoteInfo
	Kind      FileKind   // Optional type of data, either "schema" or "data", "template" / "common" or not provided
	Source    SourceKind // Optional source of data, either "file", "url", "stdin", or "stdout"
	Auth      AuthInfo
	Schema    SchemaInfo
	Content   *FileContent // Content of the file
	logger    *zerolog.Logger
}

// FileArgOption is a functional option for configuring a FileArg.
type FileArgOption func(*FileArg)

// WithKind sets the FileKind.
func WithKind(kind FileKind) FileArgOption {
	return func(fa *FileArg) {
		fa.Kind = kind
	}
}

// WithSource sets the SourceKind.
func WithSource(source SourceKind) FileArgOption {
	return func(fa *FileArg) {
		fa.Source = source
	}
}

// WithContent sets the FileContent.
func WithContent(content *FileContent) FileArgOption {
	return func(fa *FileArg) {
		fa.Content = content
	}
}

// WithContentBytes sets the content from raw bytes and marks it as read.
func WithContentBytes(data []byte) FileArgOption {
	return func(fa *FileArg) {
		fa.Content = NewFileContent()
		fa.Content.Data = data
		fa.Content.Read = true
	}
}

// WithJSONPath sets the JSONPath for data nesting.
func WithJSONPath(jp *jsonpath.Path) FileArgOption {
	return func(fa *FileArg) {
		fa.JSONPath = jp
	}
}

// WithDefaultJSONPath sets the JSONPath to "$" if not already set.
func WithDefaultJSONPath() FileArgOption {
	return func(fa *FileArg) {
		if fa.JSONPath == nil {
			fa.JSONPath = jsonpath.MustParse("$")
		}
	}
}

// WithAuth sets auth info on the FileArg.
func WithAuth(auth AuthInfo) FileArgOption {
	return func(fa *FileArg) {
		fa.Auth = auth
	}
}

// WithLogger sets the logger on the FileArg.
func WithLogger(logger *zerolog.Logger) FileArgOption {
	return func(fa *FileArg) {
		fa.logger = logger
	}
}

// NewFileArg creates a FileArg with the given name and functional options.
// Defaults: Kind=FileKindData, Content=NewFileContent(), logger=nop.
// Source is auto-detected from the name if not provided via WithSource.
func NewFileArg(name string, opts ...FileArgOption) *FileArg {
	nop := zerolog.Nop()
	fa := &FileArg{
		Name:    name,
		Kind:    FileKindData,
		Content: NewFileContent(),
		logger:  &nop,
	}
	for _, opt := range opts {
		opt(fa)
	}
	// Auto-detect source from name if not explicitly set
	if fa.Source == "" {
		if name == "-" {
			fa.Source = SourceKindStdin
		} else {
			detected, err := ParseFileStringSource(name)
			if err == nil {
				fa.Source = detected
			}
		}
	}
	if fa.Source == SourceKindFile {
		fa.NormalizePath()
	}
	return fa
}

func (f *FileArg) SetLogger(logger *zerolog.Logger) {
	if logger == nil {
		nop := zerolog.Nop()
		logger = &nop
	}
	f.logger = logger
}

func (f *FileArg) String() string {
	return fmt.Sprintf("FileArg{Name: %s, Source: %s, BearerToken: %s, BasicAuth: %s, Content: %v}", f.Name, f.Source, f.Auth.BearerToken, f.Auth.BasicAuth, f.Content)
}

func (f *FileArg) TemplateName(t *template.Template, data map[string]any) (string, error) {
	if f.Template.NewName != "" {
		return f.Template.NewName, nil
	}
	newName := bytes.NewBufferString("")
	t, err := t.New(f.Name).Parse(f.Name)
	if err != nil {
		return "", err
	}
	if err := t.ExecuteTemplate(newName, f.Name, data); err != nil {
		return "", err
	}
	f.Template.NewName = newName.String()
	return f.Template.NewName, nil
}

// RelativePath returns the relative path of the file from its root or parent.
func (f *FileArg) RelativePath() (string, error) {
	if f.Container.Root == nil || f.Container.Root == f {
		return filepath.Base(f.Name), nil
	}
	n := filepath.FromSlash(f.Name)
	rn := filepath.FromSlash(f.Container.Root.Name)
	return filepath.Rel(rn, n)
}

// RelativeNewPath returns the relative path of the file from its root or parent using NewName if available.
func (f *FileArg) RelativeNewPath() (string, error) {
	name := f.Name
	if f.Template.NewName != "" {
		name = f.Template.NewName
	}
	if f.Container.Root == nil || f.Container.Root == f {
		return filepath.Base(name), nil
	}
	n := filepath.FromSlash(name)
	rn := filepath.FromSlash(f.Container.Root.Name)
	return filepath.Rel(rn, n)
}

func (f *FileArg) NormalizePath() {
	f.Name = NormalizeFilepath(f.Name)
}

func (f *FileArg) Load() (err error) {
	if f.Content.Read {
		return nil
	}
	if isContainer, err := f.IsContainer(); err != nil {
		return err
	} else if isContainer {
		return fmt.Errorf("file %s: %w", f.Name, ErrIsContainer)
	}
	switch f.Source {
	case SourceKindFile:
		err := f.ReadFile()
		if err != nil {
			return err
		}
	case SourceKindURL:
		err = f.ReadURL()
		if err != nil {
			return err
		}
	case SourceKindStdin:
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
	if f.Name != "-" {
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
	f.Content.Data, err = os.ReadFile(f.Name)
	if err != nil {
		return err
	}
	f.Content.Read = true
	f.Content.Filename = filepath.Base(f.Name)
	mimetype, err := getMimetype(f.Content.Data)
	// TODO: mimetype from file extension
	if err != nil {
		return err
	}
	f.Content.Mimetype = mimetype
	return nil
}

func getMimetype(data []byte) (mimetype string, err error) {
	n := min(len(data), 512)
	mimetype = http.DetectContentType(data[:n]) // 512 is max of function anyways
	mimetype, _, err = mime.ParseMediaType(mimetype)
	if err != nil {
		return "", err
	}
	return mimetype, err
}

// ReadURL fetches a file from a URL and returns the filename, data, MIME type, and any error.
// It attempts to extract the filename from Content-Disposition header or falls back to the URL path.
func (f *FileArg) ReadURL() (err error) {

	if f.Source != SourceKindURL {
		return fmt.Errorf("file %s is not a url", f.Name)
	}
	if f.Remote.URL == nil {

		f.Remote.URL, err = url.Parse(f.Name)
		if err != nil {
			return fmt.Errorf("url parse error: %w", err)
		}
	}

	var mediaKV map[string]string
	var mimetype string
	resp, err := GetURL(f.Remote.URL, f.Auth.BasicAuth, f.Auth.BearerToken)
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
	} else {
		return fmt.Errorf("url get error: %w", err)
	}
	if err != nil {
		return err
	}
	f.Content.Data, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	f.Content.Read = true
	f.Remote.Response = resp
	mimetype = resp.Header.Get("Content-Type")
	if mimetype != "" {
		mimetype, mediaKV, err = mime.ParseMediaType(mimetype)

		if err != nil {
			return err
		}
	}
	if mimetype == "" {
		mimetype = http.DetectContentType(f.Content.Data[:512]) // 512 is max of function anyways
		mimetype, _, err = mime.ParseMediaType(mimetype)
		if err != nil {
			return err
		}
	}
	if mimetype == "" {
		contentDisposition := resp.Header.Get("Content-Disposition")
		if contentDisposition != "" {
			mimetype, mediaKV, err = mime.ParseMediaType(contentDisposition)
			if err != nil {
				f.logger.Error().Msg(err.Error())
				return fmt.Errorf("mimetype parse error: %w", err)
			}
		}
	}
	if _, ok := mediaKV["filename"]; ok {
		f.Content.Filename = mediaKV["filename"]
	} else {
		f.Content.Filename = filepath.Base(f.Name)
	}

	f.Content.Mimetype = mimetype

	return err
}

func (f *FileArg) IsDir() (bool, error) {
	if f.Source == SourceKindURL {
		return false, nil
	}
	return IsDir(f.Name)
}

func (f *FileArg) IsFile() (bool, error) {
	if f.Source == SourceKindStdin {
		// kind of a file, but for our purposes it's not
		return false, nil
	}
	return IsFile(f.Name)
}

func (f *FileArg) IsArchive() (bool, error) {
	if f.Source == SourceKindStdin {
		return false, nil // currently not supported for an archive through stdin
	}
	if f.Source == SourceKindURL {
		// TODO: support archives from urls
		// maybe not this, since we might have filename from the url, but i'll work that in later
		if err := assertRead(f); err != nil {
			return false, err
		}
		return false, nil
	}
	return IsArchive(f.Name), nil
}

func (f *FileArg) IsContainer() (bool, error) {
	isDir, err := f.IsDir()
	if err != nil {
		isDir = false
	}
	isArchive, err := f.IsArchive()
	if err != nil {
		if errors.Is(err, ErrNotLoaded) {
			return false, nil
		}
		return false, err
	}
	return isArchive || isDir, nil
}

func (f *FileArg) AllChildren() []*FileArg {
	if f.Container.children == nil {
		return nil
	}
	return unravelChildren(f)
}

func unravelChildren(f *FileArg) []*FileArg {
	if f.Container.children == nil {
		return []*FileArg{}
	}
	children := make([]*FileArg, 0)
	for _, child := range f.Container.children {
		children = append(children, child)
		children = append(children, unravelChildren(child)...)
	}
	return children
}

func (f *FileArg) CollectContainerChildren() error {
	if ic, err := f.IsContainer(); err != nil || !ic {
		if !ic {
			return fmt.Errorf("file %s is not a container", f.Name)
		}
		return err
	}
	if f.Container.children != nil {
		return nil
	}
	f.Container.children = make([]*FileArg, 0)
	switch f.Source {
	case SourceKindURL:
		// once you get here we know the url is an archive
		return fmt.Errorf("url %s is not implemented", f.Name)
	case SourceKindFile:
		paths, err := WalkDir(f, f.logger)
		if err != nil {
			return err
		}
		for _, p := range paths {
			if f.Name == p {
				continue
			}
			child := NewFileArg(p, WithKind(f.Kind), WithSource(SourceKindFile))
			child.Container.Parent = f
			if f.Container.Root != nil {
				child.Container.Root = f.Container.Root
			} else {
				child.Container.Root = f
			}
			f.Container.children = append(f.Container.children, child)
		}
		return nil

	default:
		return fmt.Errorf("file %s is not a file or url but a %s", f.Name, f.Source)
	}
}

func (f *FileArg) LoadContainer() error {
	err := f.CollectContainerChildren()
	if err != nil {
		return err
	}
	if f.Container.children != nil {
		for _, fi := range f.Container.children {
			if isContainer, err := fi.IsContainer(); err != nil {
				return err
			} else if isContainer {
				err = fi.LoadContainer()
				if err != nil {
					return err
				}
			} else {
				err = fi.Load()
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (f *FileArg) IsText() (bool, error) {

	if f.Content.Mimetype == "" {
		if err := assertRead(f); err != nil {
			return false, err
		}
		// TODO: add support for other some other less common encodings.
		if !utf8.Valid(f.Content.Data) {
			return false, nil
		}
	}
	return strings.Contains(f.Content.Mimetype, "text"), nil
}

func GetURL(u *url.URL, basicAuth, bearerToken string) (data *http.Response, err error) {
	req, err := http.NewRequest("GET", u.String(), http.NoBody)
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

func assertRead(f *FileArg) (err error) {
	if !f.Content.Read {
		return fmt.Errorf("file %s: %w", f.Name, ErrNotLoaded)
	}
	return nil
}
