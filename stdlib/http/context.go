package http

import (
    "net/http"
    "encoding/json"
)

type Context struct {
    Writer  http.ResponseWriter
    Request *http.Request
}

func NewContext(w http.ResponseWriter, r *http.Request) *Context {
    return &Context{Writer: w, Request: r}
}

func (c *Context) JSON(status int, v any) {
    c.Writer.Header().Set("Content-Type", "application/json")
    c.Writer.WriteHeader(status)
    json.NewEncoder(c.Writer).Encode(v)
}

func (c *Context) Text(status int, s string) {
    c.Writer.Header().Set("Content-Type", "text/plain")
    c.Writer.WriteHeader(status)
    c.Writer.Write([]byte(s))
}
