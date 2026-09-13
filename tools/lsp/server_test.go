package lsp

import (
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"
)

func TestInitializeResult(t *testing.T) {
	sess := &session{}
	result := handleInitialize(sess, nil)
	if result.Capabilities.HoverProvider != true {
		t.Error("expected hoverProvider true")
	}
	if result.Capabilities.DefinitionProvider != true {
		t.Error("expected definitionProvider true")
	}
	if result.Capabilities.CompletionProvider == nil {
		t.Error("expected completionProvider to be advertised")
	}
	if result.Capabilities.ReferencesProvider != true {
		t.Error("expected referencesProvider true")
	}
	if result.Capabilities.DocumentSymbolProvider != true {
		t.Error("expected documentSymbolProvider true")
	}
	if result.Capabilities.WorkspaceSymbolProvider != true {
		t.Error("expected workspaceSymbolProvider true")
	}
	if result.ServerInfo.Name != "rayo" {
		t.Errorf("expected server name rayo, got %q", result.ServerInfo.Name)
	}
}

func TestInitializeNegotiatesSnippetSupport(t *testing.T) {
	sess := &session{}
	raw := `{"capabilities":{"textDocument":{"completion":{"completionItem":{"snippetSupport":true}}}}}`
	var params any
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	handleInitialize(sess, &params)
	if !sess.snippetSupport {
		t.Error("expected snippetSupport to be negotiated to true")
	}
}

func TestGetDiagnostics(t *testing.T) {
	// Valid code: no errors
	diags := getDiagnostics("file:///test.ryo", `import "fmt"
def main() {
    print(1)
}`)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics for valid code, got %d: %v", len(diags), diags)
	}

	// Invalid: missing function name after def
	diags = getDiagnostics("file:///bad.ryo", "def ()\n{ }")
	if len(diags) < 1 {
		t.Errorf("expected at least one diagnostic for invalid code, got %d", len(diags))
	}
}

func TestRunServer(t *testing.T) {
	// Test that the server can start and accept connections
	addr := "127.0.0.1:9999" // Use a fixed port for testing
	go func() {
		if err := RunServer(addr); err != nil {
			t.Errorf("RunServer failed: %v", err)
		}
	}()

	// Wait a bit for the server to start
	time.Sleep(100 * time.Millisecond)

	// Try to connect to the server
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Errorf("Failed to connect to server: %v", err)
		return
	}
	defer conn.Close()

	// Send a simple JSON-RPC request
	request := `{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}`
	_, err = conn.Write([]byte(request + "\n"))
	if err != nil {
		t.Errorf("Failed to send request: %v", err)
		return
	}

	// Read the response
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		t.Errorf("Failed to read response: %v", err)
		return
	}

	response := string(buffer[:n])
	if response == "" {
		t.Errorf("Received empty response")
	}

	// Server responds with initialize result (capabilities)
	if response == "" {
		t.Errorf("Received empty response")
	}
	if !strings.Contains(response, "hoverProvider") {
		t.Errorf("Expected initialize result with capabilities; got %q", response)
	}
}
