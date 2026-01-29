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
	IgnoreEmpty      bool   `json:"ignore-empty"`
	IncludeFilenames bool   `json:"include-filenames"`
	Overwrite        bool   `json:"overwrite"`
	Helm             bool   `json:"helm"`

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
