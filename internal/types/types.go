package types

// YutcSettings is a struct to hold all the settings from the CLI
type YutcSettings struct {
	DataFiles []string `json:"data-files"`
	//DataMatch []string `json:"data-match"`

	CommonTemplateFiles []string `json:"common-templates"`
	//CommonTemplateMatch []string `json:"common-templates-match"`

	TemplatePaths []string `json:"template-files"`
	//TemplateMatch []string `json:"template-match"`

	Output           string `json:"output"`
	IncludeFilenames bool   `json:"include-filenames"`
	Overwrite        bool   `json:"overwrite"`

	Strict bool

	Version bool `json:"version"`
	Verbose bool `json:"verbose"`

	BearerToken string `json:"bearer-auth"`
	BasicAuth   string `json:"basic-auth"`
}

func NewCLISettings() *YutcSettings {
	return &YutcSettings{}
}

// DataFileArg represents a parsed data file argument with optional top-level key
type DataFileArg struct {
	Key  string // Optional top-level key to nest the data under
	Path string // File path, URL, or "-" for stdin
}
