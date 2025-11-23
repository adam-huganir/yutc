package types

// Arguments is a struct to hold all the settings from the CLI
type Arguments struct {
	DataFiles []string `json:"data-files"`
	SetData   []string `json:"set-data"`
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

func NewCLISettings() *Arguments {
	return &Arguments{}
}

// DataFileArg represents a parsed data file argument with optional top-level key
type DataFileArg struct {
	Key  string // Optional top-level key to nest the data under
	Path string // File path, URL, or "-" for stdin
}
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	return e.Err.Error()
}

type RunData struct {
	DataFiles           []*DataFileArg
	CommonTemplateFiles []string
	TemplatePaths       []string
}

func NewRunData() *RunData {
	return &RunData{}
}
