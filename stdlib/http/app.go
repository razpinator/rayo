package http

import (
	"context"
	"net/http"
	"time"
)

// App is a small HTTP application built on net/http's ServeMux. It uses Go
// 1.22 method-aware pattern routing so handlers only fire for their verb and
// path parameters (`/users/{id}`) are available via Context.Param.
type App struct {
	mux    *http.ServeMux
	server *http.Server
}

func NewApp() *App {
	return &App{mux: http.NewServeMux()}
}

// handle registers handler for the given method and path pattern. The pattern
// may contain `{name}` segments whose values are read with Context.Param.
func (a *App) handle(method, pattern string, handler func(*Context)) {
	a.mux.HandleFunc(method+" "+pattern, func(w http.ResponseWriter, r *http.Request) {
		handler(NewContext(w, r))
	})
}

// Get registers a GET handler.
func (a *App) Get(path string, handler func(*Context)) { a.handle(http.MethodGet, path, handler) }

// Post registers a POST handler.
func (a *App) Post(path string, handler func(*Context)) { a.handle(http.MethodPost, path, handler) }

// Put registers a PUT handler.
func (a *App) Put(path string, handler func(*Context)) { a.handle(http.MethodPut, path, handler) }

// Delete registers a DELETE handler.
func (a *App) Delete(path string, handler func(*Context)) {
	a.handle(http.MethodDelete, path, handler)
}

// Patch registers a PATCH handler.
func (a *App) Patch(path string, handler func(*Context)) { a.handle(http.MethodPatch, path, handler) }

// Listen starts the server on addr and blocks. Read/write timeouts guard
// against slow-client resource exhaustion, a common production edge case.
func (a *App) Listen(addr string) error {
	a.server = &http.Server{
		Addr:              addr,
		Handler:           a.mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return a.server.ListenAndServe()
}

// Shutdown gracefully stops a running server.
func (a *App) Shutdown(ctx context.Context) error {
	if a.server == nil {
		return nil
	}
	return a.server.Shutdown(ctx)
}
