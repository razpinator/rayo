package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoutingByMethod(t *testing.T) {
	app := NewApp()
	app.Get("/ping", func(c *Context) { c.Text(200, "pong") })

	srv := httptest.NewServer(app.mux)
	defer srv.Close()

	// GET matches.
	resp, err := http.Get(srv.URL + "/ping")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("GET /ping: got %d", resp.StatusCode)
	}

	// POST to a GET-only route should not match (405/404).
	resp2, err := http.Post(srv.URL+"/ping", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode == 200 {
		t.Errorf("POST /ping should not hit the GET handler")
	}
}

func TestQueryAndParam(t *testing.T) {
	app := NewApp()
	app.Get("/users/{id}", func(c *Context) {
		c.Text(200, "id="+c.Param("id")+" q="+c.QueryDefault("q", "none"))
	})
	srv := httptest.NewServer(app.mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/users/42?q=hi")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	buf := make([]byte, 64)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])
	if !strings.Contains(body, "id=42") || !strings.Contains(body, "q=hi") {
		t.Errorf("param/query not read: %q", body)
	}
}

func TestBindJSON(t *testing.T) {
	app := NewApp()
	app.Post("/echo", func(c *Context) {
		var in map[string]any
		if err := c.BindJSON(&in); err != nil {
			c.JSON(400, map[string]any{"error": "bad json"})
			return
		}
		c.JSON(200, in)
	})
	srv := httptest.NewServer(app.mux)
	defer srv.Close()

	// Valid JSON.
	resp, err := http.Post(srv.URL+"/echo", "application/json", strings.NewReader(`{"name":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("valid JSON body: got %d", resp.StatusCode)
	}

	// Malformed JSON -> handler returns 400.
	resp2, err := http.Post(srv.URL+"/echo", "application/json", strings.NewReader(`{not json`))
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != 400 {
		t.Errorf("malformed JSON should yield 400, got %d", resp2.StatusCode)
	}
}

func TestStatusHelper(t *testing.T) {
	app := NewApp()
	app.Delete("/x", func(c *Context) { c.Status(204) })
	srv := httptest.NewServer(app.mux)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/x", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 204 {
		t.Errorf("Status(204): got %d", resp.StatusCode)
	}
}
