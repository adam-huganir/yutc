package data

import (
	"errors"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetURL_BearerAuth(t *testing.T) {
	srv := newAuthFileServer(t, testAuthConfig{bearerToken: "secret"})

	u, err := url.Parse(srv.URL + "/templates/simpleTemplate.tmpl")
	assert.NoError(t, err)

	resp, err := GetURL(u, "", "secret")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	_ = resp.Body.Close()

	resp, err = GetURL(u, "", "wrong")
	assert.NotNil(t, resp)
	var httpErr *HTTPStatusError
	assert.True(t, errors.As(err, &httpErr))
	assert.Equal(t, 401, httpErr.StatusCode)
}

func TestGetURL_BasicAuth(t *testing.T) {
	srv := newAuthFileServer(t, testAuthConfig{basicUser: "user", basicPass: "pass"})

	u, err := url.Parse(srv.URL + "/templates/simpleTemplate.tmpl")
	assert.NoError(t, err)

	resp, err := GetURL(u, "user:pass", "")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	_ = resp.Body.Close()

	resp, err = GetURL(u, "user:wrong", "")
	assert.NotNil(t, resp)
	var httpErr *HTTPStatusError
	assert.True(t, errors.As(err, &httpErr))
	assert.Equal(t, 401, httpErr.StatusCode)
}

func TestGetURL_BasicTakesPrecedenceOverBearer(t *testing.T) {
	// Server allows bearer auth only.
	srv := newAuthFileServer(t, testAuthConfig{bearerToken: "secret"})

	u, err := url.Parse(srv.URL + "/templates/simpleTemplate.tmpl")
	assert.NoError(t, err)

	// Client behavior: if basicAuth is set, it will NOT send bearer even if provided.
	resp, err := GetURL(u, "user:wrong", "secret")
	assert.NotNil(t, resp)
	var httpErr *HTTPStatusError
	assert.True(t, errors.As(err, &httpErr))
	assert.Equal(t, 401, httpErr.StatusCode)
}
