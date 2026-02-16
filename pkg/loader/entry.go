package loader

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/rs/zerolog"
)

const (
	dataPreallocate = 1024 * 8
)

// SourceKind identifies where a file comes from.
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

// FileContent holds the raw bytes and metadata for a loaded file.
type FileContent struct {
	Filename string // name of file, either from path or url, or '-' for stdin
	Mimetype string // mimetype if known
	Data     []byte // contents of file gathered during load/download
	Read     bool   // whether the file has been read into memory
}

// NewFileContent creates a FileContent with a pre-allocated buffer.
func NewFileContent() *FileContent {
	b := make([]byte, 0, dataPreallocate)
	return &FileContent{Data: b}
}

// RemoteInfo holds HTTP response metadata for URL-sourced files.
type RemoteInfo struct {
	URL      *url.URL       // URL for http call if the source is a url
	Response *http.Response // Response from http call if the source is a url
}

// AuthInfo holds authentication credentials for URL-sourced files.
type AuthInfo struct {
	BearerToken string // Bearer token for http call. just token, not "Bearer "
	BasicAuth   string // Basic auth for http call in username:password format
	Disabled    bool   // If true, authentication is explicitly disabled for this entry
	Lazy        bool   // If true, authentication is only sent if the server returns 401
}

// ParseAuthString interprets a combined auth string as either Basic Auth (user:pass) or Bearer Token.
func ParseAuthString(authStr string) AuthInfo {
	if authStr == "" {
		return AuthInfo{}
	}
	if strings.EqualFold(authStr, "false") {
		return AuthInfo{Disabled: true}
	}
	if strings.Contains(authStr, ":") {
		return AuthInfo{BasicAuth: authStr}
	}
	return AuthInfo{BearerToken: authStr}
}

// FileEntry is the shared loading base for all file inputs (data and templates).
// It handles source detection, reading bytes from file/url/stdin, and path normalization.
type FileEntry struct {
	Name    string     // File path, URL, or "-" for stdin
	Source  SourceKind // Source of data: "file", "url", "stdin", or "stdout"
	Content *FileContent
	Auth    AuthInfo
	Remote  RemoteInfo
	logger  *zerolog.Logger
}

// FileEntryOption is a functional option for configuring a FileEntry.
type FileEntryOption func(*FileEntry)

// WithSource sets the SourceKind.
func WithSource(source SourceKind) FileEntryOption {
	return func(fe *FileEntry) {
		fe.Source = source
	}
}

// WithContent sets the FileContent.
func WithContent(content *FileContent) FileEntryOption {
	return func(fe *FileEntry) {
		fe.Content = content
	}
}

// WithContentBytes sets the content from raw bytes and marks it as read.
func WithContentBytes(data []byte) FileEntryOption {
	return func(fe *FileEntry) {
		fe.Content = NewFileContent()
		fe.Content.Data = data
		fe.Content.Read = true
	}
}

// WithAuth sets auth info on the FileEntry.
func WithAuth(auth AuthInfo) FileEntryOption {
	return func(fe *FileEntry) {
		fe.Auth = auth
	}
}

// WithLogger sets the logger on the FileEntry.
func WithLogger(logger *zerolog.Logger) FileEntryOption {
	return func(fe *FileEntry) {
		fe.logger = logger
	}
}

// NewFileEntry creates a FileEntry with the given name and functional options.
// Defaults: Content=NewFileContent(), logger=nop.
// Source is auto-detected from the name if not provided via WithSource.
func NewFileEntry(name string, opts ...FileEntryOption) *FileEntry {
	nop := zerolog.Nop()
	fe := &FileEntry{
		Name:    name,
		Content: NewFileContent(),
		logger:  &nop,
	}
	for _, opt := range opts {
		opt(fe)
	}
	// Auto-detect source from name if not explicitly set
	if fe.Source == "" {
		if name == "-" {
			fe.Source = SourceKindStdin
		} else {
			detected, err := ParseFileStringSource(name)
			if err == nil {
				fe.Source = detected
			}
		}
	}
	if fe.Source == SourceKindFile {
		fe.NormalizePath()
	}
	return fe
}

func (f *FileEntry) SetLogger(logger *zerolog.Logger) {
	if logger == nil {
		nop := zerolog.Nop()
		logger = &nop
	}
	f.logger = logger
}

// Logger returns the logger for this FileEntry.
func (f *FileEntry) Logger() *zerolog.Logger {
	return f.logger
}

func (f *FileEntry) String() string {
	return fmt.Sprintf("FileEntry{Name: %s, Source: %s, Auth: %+v, Content: %v}", f.Name, f.Source, f.Auth, f.Content)
}

func (f *FileEntry) NormalizePath() {
	f.Name = NormalizeFilepath(f.Name)
}

// Load reads the file content based on its source. Returns ErrIsContainer if the entry is a directory or archive.
func (f *FileEntry) Load() (err error) {
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

func (f *FileEntry) ReadStdin() (err error) {
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

func (f *FileEntry) ReadFile() (err error) {
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
func (f *FileEntry) ReadURL() (err error) {

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

	// First attempt without auth if Lazy is true
	var resp *http.Response
	if f.Auth.Lazy {
		resp, err = GetURL(f.Remote.URL, "", "")
		if err == nil && resp.StatusCode == http.StatusUnauthorized {
			_ = resp.Body.Close()
			// Retry with auth
			resp, err = GetURL(f.Remote.URL, f.Auth.BasicAuth, f.Auth.BearerToken)
		}
	} else {
		resp, err = GetURL(f.Remote.URL, f.Auth.BasicAuth, f.Auth.BearerToken)
	}

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

func (f *FileEntry) IsDir() (bool, error) {
	if f.Source == SourceKindURL {
		return false, nil
	}
	return IsDir(f.Name)
}

func (f *FileEntry) IsFile() (bool, error) {
	if f.Source == SourceKindStdin {
		return false, nil
	}
	return IsFile(f.Name)
}

func (f *FileEntry) IsArchive() (bool, error) {
	if f.Source == SourceKindStdin {
		return false, nil
	}
	if f.Source == SourceKindURL {
		// TODO: support archives from urls
		if err := AssertRead(f); err != nil {
			return false, err
		}
		return false, nil
	}
	return IsArchive(f.Name), nil
}

func (f *FileEntry) IsContainer() (bool, error) {
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

func (f *FileEntry) IsText() (bool, error) {
	if f.Content.Mimetype == "" {
		if err := AssertRead(f); err != nil {
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

func AssertRead(f *FileEntry) (err error) {
	if !f.Content.Read {
		return fmt.Errorf("file %s: %w", f.Name, ErrNotLoaded)
	}
	return nil
}
