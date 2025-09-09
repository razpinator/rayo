package http

import (
    "net/http"
)

type App struct {
    mux *http.ServeMux
}

func NewApp() *App {
    return &App{mux: http.NewServeMux()}
}

func (a *App) Get(path string, handler func(*Context)) {
    a.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodGet {
            handler(NewContext(w, r))
        } else {
            http.NotFound(w, r)
        }
    })
}

func (a *App) Post(path string, handler func(*Context)) {
    a.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodPost {
            handler(NewContext(w, r))
        } else {
            http.NotFound(w, r)
        }
    })
}

func (a *App) Listen(addr string) error {
    return http.ListenAndServe(addr, a.mux)
}
