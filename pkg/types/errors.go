package types

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

type DedentError struct {
	Line   string
	Prefix string
}

func (e *DedentError) Error() string {
	return "cannot dedent line: \"" + e.Line + "\" with prefix: \"" + e.Prefix + "\""
}
