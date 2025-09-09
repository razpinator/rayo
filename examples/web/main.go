package main

import (
    "functure/stdlib/http"
)

func main() {
    app := http.NewApp()
    app.Get("/health", func(ctx *http.Context) {
        ctx.Text(200, "ok")
    })
    app.Post("/echo", func(ctx *http.Context) {
        var data map[string]any
        err := ctx.Request.ParseForm()
        if err != nil {
            ctx.JSON(400, map[string]any{"error": "bad request"})
            return
        }
        ctx.JSON(200, map[string]any{"ok": true, "echo": data})
    })
    app.Listen(":8080")
}
