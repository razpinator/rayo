package lsp

import (
    "encoding/json"
    "fmt"
    "net"
)

// Minimal LSP server skeleton
func RunServer(addr string) error {
    ln, err := net.Listen("tcp", addr)
    if err != nil {
        return err
    }
    fmt.Println("LSP server listening on", addr)
    for {
        conn, err := ln.Accept()
        if err != nil {
            continue
        }
        go handleConn(conn)
    }
}

func handleConn(conn net.Conn) {
    defer conn.Close()
    dec := json.NewDecoder(conn)
    enc := json.NewEncoder(conn)
    for {
        var req map[string]any
        if err := dec.Decode(&req); err != nil {
            break
        }
        // TODO: handle LSP requests (textDocument/didChange, hover, definition, diagnostics)
        resp := map[string]any{"result": "not implemented"}
        enc.Encode(resp)
    }
}
