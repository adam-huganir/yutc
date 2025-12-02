// Package types defines the core data structures used throughout yutc.
package types // nolint:revive // make this a better name at some point?

// Arguments is a struct to hold all the settings from the CLI
type Arguments struct {
	DataFiles []string `json:"data-files"`
	SetData   []string `json:"set-data"`
	// DataMatch []string `json:"data-match"`

	CommonTemplateFiles []string `json:"common-templates"`
	// CommonTemplateMatch []string `json:"common-templates-match"`

	TemplatePaths []string `json:"template-files"`
	// TemplateMatch []string `json:"template-match"`

	Output           string `json:"output"`
	IncludeFilenames bool   `json:"include-filenames"`
	Overwrite        bool   `json:"overwrite"`

	Strict bool

	Version bool `json:"version"`
	Verbose bool `json:"verbose"`

	BearerToken string `json:"bearer-auth"`
	BasicAuth   string `json:"basic-auth"`
}

// NewCLISettings creates and returns a new Arguments struct with default values.
func NewCLISettings() *Arguments {
	return &Arguments{}
}

// DataFileArg represents a parsed data file argument with optional top-level key
type DataFileArg struct {
	Key  string // Optional top-level key to nest the data under
	Path string // File path, URL, or "-" for stdin
}

// ExitError represents an error with an associated exit code for CLI commands.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	return e.Err.Error()
}

// ValidationError represents an error that occurred during argument validation.
type ValidationError struct {
	Errors []error
}

func (e *ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "validation error"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	msg := "validation errors:"
	for _, err := range e.Errors {
		msg += "\n  - " + err.Error()
	}
	return msg
}

// RuntimeError represents a general runtime error.
type RuntimeError struct {
	Err error
}

func (e *RuntimeError) Error() string {
	return e.Err.Error()
}

func (e *RuntimeError) Unwrap() error {
	return e.Err
}

// TemplateError represents an error that occurred during template execution.
type TemplateError struct {
	TemplatePath string
	Err          error
}

func (e *TemplateError) Error() string {
	if e.TemplatePath != "" {
		return "template error in " + e.TemplatePath + ": " + e.Err.Error()
	}
	return "template error: " + e.Err.Error()
}

func (e *TemplateError) Unwrap() error {
	return e.Err
}

// RunData holds runtime data for template execution including data files and template paths.
type RunData struct {
	DataFiles           []*DataFileArg
	CommonTemplateFiles []string
	TemplatePaths       []string
}

// NewRunData creates and returns a new RunData struct.
func NewRunData() *RunData {
	return &RunData{}
}
