package data

import (
	"fmt"
	"net/http"
)

// HTTPStatusError represents an HTTP error response with status code and response details.
type HTTPStatusError struct {
	StatusCode int
	Status     string
	Response   http.Response
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("HTTP status %s", e.Status)
}

// StatusText returns the HTTP status text for this error's status code.
func (e *HTTPStatusError) StatusText() string {
	return http.StatusText(e.StatusCode)
}

// NewHTTPStatusError creates a new HTTPStatusError from an HTTP response.
func NewHTTPStatusError(resp *http.Response) *HTTPStatusError {
	return &HTTPStatusError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Response:   *resp,
	}
}
