package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func BenchmarkEchoServerThroughput(b *testing.B) {
	app := NewApp()
	app.Post("/echo", func(ctx *Context) {
		var data map[string]any
		err := ctx.Request.ParseForm()
		if err != nil {
			ctx.JSON(400, map[string]any{"error": "bad request"})
			return
		}
		ctx.JSON(200, map[string]any{"ok": true, "echo": data})
	})
	server := httptest.NewServer(app.mux)
	defer server.Close()

	client := &http.Client{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := strings.NewReader("key=value")
		req, _ := http.NewRequest("POST", server.URL+"/echo", body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}
