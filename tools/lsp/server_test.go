package lsp

import (
	"net"
	"testing"
	"time"
)

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

	// The server currently just responds with "not implemented", so we check for that
	expected := `{"result":"not implemented"}`
	if response != expected+"\n" && response != expected+"\r\n" {
		t.Errorf("Unexpected response: got %q, want %q", response, expected)
	}
}
