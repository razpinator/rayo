package http

import (
	"encoding/json"
	"io"
	"net/http"
)

// Context wraps a single request/response exchange with convenience helpers for
// reading input (query, form, JSON body) and writing responses.
type Context struct {
	Writer  http.ResponseWriter
	Request *http.Request
}

func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{Writer: w, Request: r}
}

// JSON writes v as a JSON response with the given status code. It reports any
// encoding error so handlers can react rather than silently dropping it.
func (c *Context) JSON(status int, v any) error {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(status)
	return json.NewEncoder(c.Writer).Encode(v)
}

// Text writes a plain-text response with the given status code.
func (c *Context) Text(status int, s string) {
	c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Writer.WriteHeader(status)
	_, _ = io.WriteString(c.Writer, s)
}

// Status writes an empty response with only a status code.
func (c *Context) Status(status int) {
	c.Writer.WriteHeader(status)
}

// Query returns the first value of a URL query parameter, or "" if absent.
func (c *Context) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

// QueryDefault returns the query parameter or def when it is missing/empty.
func (c *Context) QueryDefault(key, def string) string {
	if v := c.Query(key); v != "" {
		return v
	}
	return def
}

// Param returns a path parameter captured by the router (see App path patterns
// with `:name` segments), or "" if not present.
func (c *Context) Param(key string) string {
	if c.Request == nil {
		return ""
	}
	return c.Request.PathValue(key)
}

// Form returns a posted form value after parsing the request body.
func (c *Context) Form(key string) string {
	_ = c.Request.ParseForm()
	return c.Request.PostFormValue(key)
}

// Header returns a request header value.
func (c *Context) Header(key string) string {
	return c.Request.Header.Get(key)
}

// BindJSON decodes the request body into v. It rejects unknown fields to catch
// malformed payloads early, a common HTTP edge case.
func (c *Context) BindJSON(v any) error {
	if c.Request == nil || c.Request.Body == nil {
		return io.EOF
	}
	defer c.Request.Body.Close()
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
