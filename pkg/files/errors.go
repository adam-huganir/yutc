package files

import (
	"fmt"
	"net/http"
)

type HttpStatusError struct {
	StatusCode int
	Status     string
	Response   http.Response
}

func (e *HttpStatusError) Error() string {
	return fmt.Sprintf("HTTP status %s", e.Status)
}

func (e *HttpStatusError) StatusText() string {
	return http.StatusText(e.StatusCode)
}

func NewHttpStatusError(resp *http.Response) *HttpStatusError {
	return &HttpStatusError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Response:   *resp,
	}
}
