package lsp

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"rayo/internal/diag"
	"rayo/internal/lex"
	"rayo/internal/parse"
	"rayo/internal/sem"
)

// docStore holds open document contents by URI.
type docStore struct {
	mu   sync.RWMutex
	docs map[string]string
}

func newDocStore() *docStore {
	return &docStore{docs: make(map[string]string)}
}

func (s *docStore) set(uri string, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[uri] = content
}

func (s *docStore) get(uri string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.docs[uri]
}

func (s *docStore) remove(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.docs, uri)
}

// all returns a snapshot of every open document keyed by URI. Callers get a
// copy so they can iterate without holding the lock.
func (s *docStore) all() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.docs))
	for k, v := range s.docs {
		out[k] = v
	}
	return out
}

// RunServer starts the LSP server on the given TCP address.
func RunServer(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	fmt.Println("LSP server listening on", addr)
	store := newDocStore()
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleConn(conn, store)
	}
}

// RunServerStdio runs a single LSP session over stdin/stdout. This transport
// is preferred by many editors (Neovim, Emacs, Helix) that spawn the language
// server as a child process rather than connecting over TCP.
func RunServerStdio() error {
	store := newDocStore()
	serve(os.Stdin, os.Stdout, store)
	return nil
}

func handleConn(conn net.Conn, store *docStore) {
	defer conn.Close()
	serve(conn, conn, store)
}

// serve runs the JSON-RPC read/dispatch loop over an arbitrary reader/writer.
// It backs both the TCP transport (handleConn) and the stdio transport
// (RunServerStdio), so the two paths share identical protocol behavior.
func serve(r io.Reader, w io.Writer, store *docStore) {
	dec := json.NewDecoder(r)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	sess := &session{enc: enc, store: store}

	for {
		var req jsonRPCRequest
		if err := dec.Decode(&req); err != nil {
			// EOF or malformed stream terminates the connection.
			return
		}
		if req.JSONRPC != "" && req.JSONRPC != "2.0" {
			if req.ID != nil {
				sendResponse(enc, req.ID, nil, &jsonRPCError{Code: codeInvalidRequest, Message: "unsupported jsonrpc version"})
			}
			continue
		}
		if dispatch(sess, &req) {
			return // exit requested
		}
	}
}

// JSON-RPC / LSP error codes.
const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603
	codeServerNotInit  = -32002
)

// dispatch handles a single request. It returns true when the server should
// stop serving (in response to the LSP "exit" notification).
func dispatch(sess *session, req *jsonRPCRequest) (exit bool) {
	enc := sess.enc
	store := sess.store
	isNotification := req.ID == nil

	// Guard: only "initialize" and "exit" are allowed before initialization.
	if !sess.initialized && req.Method != "initialize" && req.Method != "exit" {
		if !isNotification {
			sendResponse(enc, req.ID, nil, &jsonRPCError{Code: codeServerNotInit, Message: "server not initialized"})
		}
		return false
	}

	switch req.Method {
	case "initialize":
		result := handleInitialize(sess, req.Params)
		sess.initialized = true
		if !isNotification {
			sendResponse(enc, req.ID, result, nil)
		}
	case "initialized":
		// Notification, no response.
	case "shutdown":
		sess.shutdown = true
		if !isNotification {
			sendResponse(enc, req.ID, nil, nil) // result: null
		}
	case "exit":
		return true
	case "textDocument/didOpen":
		handleDidOpen(sess, req.Params)
	case "textDocument/didChange":
		handleDidChange(sess, req.Params)
	case "textDocument/didClose":
		handleDidClose(store, req.Params)
	case "textDocument/hover":
		result := handleHover(store, req.Params)
		if !isNotification {
			sendResponse(enc, req.ID, result, nil)
		}
	case "textDocument/definition":
		result := handleDefinition(store, req.Params)
		if !isNotification {
			sendResponse(enc, req.ID, result, nil)
		}
	case "textDocument/completion":
		result := handleCompletion(store, req.Params, sess.snippetSupport)
		if !isNotification {
			sendResponse(enc, req.ID, result, nil)
		}
	case "textDocument/references":
		result := handleReferences(store, req.Params)
		if !isNotification {
			sendResponse(enc, req.ID, emptyLocations(result), nil)
		}
	case "textDocument/documentSymbol":
		result := handleDocumentSymbol(store, req.Params)
		if !isNotification {
			sendResponse(enc, req.ID, emptyDocSymbols(result), nil)
		}
	case "workspace/symbol":
		result := handleWorkspaceSymbol(store, req.Params)
		if !isNotification {
			sendResponse(enc, req.ID, emptySymInfo(result), nil)
		}
	default:
		if !isNotification {
			sendResponse(enc, req.ID, nil, &jsonRPCError{Code: codeMethodNotFound, Message: "method not found: " + req.Method})
		}
	}
	return false
}

// emptyLocations guarantees a non-nil JSON array ([] rather than null) so
// clients that treat null as an error are satisfied.
func emptyLocations(in []Location) []Location {
	if in == nil {
		return []Location{}
	}
	return in
}

func emptyDocSymbols(in []DocumentSymbol) []DocumentSymbol {
	if in == nil {
		return []DocumentSymbol{}
	}
	return in
}

func emptySymInfo(in []SymbolInformation) []SymbolInformation {
	if in == nil {
		return []SymbolInformation{}
	}
	return in
}

// session holds encoder and store for sending notifications (e.g. publishDiagnostics).
type session struct {
	enc            *json.Encoder
	store          *docStore
	snippetSupport bool
	initialized    bool
	shutdown       bool
}

func (s *session) publishDiagnostics(uri string, diagnostics []Diagnostic) {
	notif := struct {
		JSONRPC string                   `json:"jsonrpc"`
		Method  string                   `json:"method"`
		Params  PublishDiagnosticsParams `json:"params"`
	}{"2.0", "textDocument/publishDiagnostics", PublishDiagnosticsParams{URI: uri, Diagnostics: diagnostics}}
	_ = s.enc.Encode(notif)
}

func sendResponse(enc *json.Encoder, id *any, result any, err *jsonRPCError) {
	if id == nil {
		return
	}
	var idVal any
	if id != nil {
		idVal = *id
	}
	resp := jsonRPCResponse{JSONRPC: "2.0", ID: idVal, Result: result}
	if err != nil {
		resp.Result = nil
		resp.Error = &struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    any    `json:"data,omitempty"`
		}{Code: err.Code, Message: err.Message, Data: err.Data}
	}
	_ = enc.Encode(resp)
}

type jsonRPCError struct {
	Code    int
	Message string
	Data    any
}

func handleInitialize(sess *session, params *any) InitializeResult {
	// Capability negotiation: record what the client supports so responses can
	// adapt (e.g. plain-text vs snippet completions).
	if params != nil {
		b, _ := json.Marshal(params)
		var p InitializeParams
		if json.Unmarshal(b, &p) == nil {
			sess.snippetSupport = p.Capabilities.TextDocument.Completion.CompletionItem.SnippetSupport
		}
	}

	fullSync := 1
	return InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync:   &fullSync,
			HoverProvider:      true,
			DefinitionProvider: true,
			CompletionProvider: &CompletionOptions{
				TriggerCharacters: []string{"."},
				ResolveProvider:   false,
			},
			ReferencesProvider:      true,
			DocumentSymbolProvider:  true,
			WorkspaceSymbolProvider: true,
		},
		ServerInfo: struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}{Name: "rayo", Version: "0.2.0"},
	}
}

func handleDidOpen(session *session, params *any) {
	if params == nil {
		return
	}
	b, _ := json.Marshal(params)
	var p DidOpenTextDocumentParams
	if json.Unmarshal(b, &p) != nil {
		return
	}
	session.store.set(p.TextDocument.URI, p.TextDocument.Text)
	diags := getDiagnostics(p.TextDocument.URI, p.TextDocument.Text)
	session.publishDiagnostics(p.TextDocument.URI, diags)
}

func handleDidChange(session *session, params *any) {
	if params == nil {
		return
	}
	b, _ := json.Marshal(params)
	var p DidChangeTextDocumentParams
	if json.Unmarshal(b, &p) != nil {
		return
	}
	// Full sync: contentChanges may contain a single full-document change (no range)
	for _, c := range p.ContentChanges {
		if c.Range == nil {
			session.store.set(p.TextDocument.URI, c.Text)
			diags := getDiagnostics(p.TextDocument.URI, c.Text)
			session.publishDiagnostics(p.TextDocument.URI, diags)
			break
		}
	}
}

func handleDidClose(store *docStore, params *any) {
	if params == nil {
		return
	}
	b, _ := json.Marshal(params)
	var p DidCloseTextDocumentParams
	if json.Unmarshal(b, &p) != nil {
		return
	}
	store.remove(p.TextDocument.URI)
}

// spanToLSPRange converts 1-based diag.Span to 0-based LSP Range.
func spanToLSPRange(s diag.Span) Range {
	return Range{
		Start: Position{Line: s.Start.Line - 1, Character: s.Start.Col - 1},
		End:   Position{Line: s.End.Line - 1, Character: s.End.Col - 1},
	}
}

func handleHover(store *docStore, params *any) *Hover {
	if params == nil {
		return nil
	}
	b, _ := json.Marshal(params)
	var p TextDocumentPositionParams
	if json.Unmarshal(b, &p) != nil {
		return nil
	}
	content := store.get(p.TextDocument.URI)
	if content == "" {
		return nil
	}
	// Find token at position (LSP: 0-based line, 0-based char)
	line := p.Position.Line + 1
	col := p.Position.Character + 1
	offset := lineColToOffset(content, line, col)
	lx := lex.NewLexer(content)
	for {
		tok := lx.Next()
		if tok.Kind == lex.TokenEOF {
			break
		}
		if tok.Kind == lex.TokenWhitespace || tok.Kind == lex.TokenComment {
			continue
		}
		start := tok.Offset
		end := tok.Offset + len(tok.Value)
		if offset >= start && offset < end {
			// Token under cursor
			msg := tok.Value
			if tok.Kind == lex.TokenKeyword {
				msg = "keyword: " + tok.Value
			} else if tok.Kind == lex.TokenIdent {
				msg = "identifier: " + tok.Value
			} else if tok.Kind == lex.TokenNumber {
				msg = "number: " + tok.Value
			} else if tok.Kind == lex.TokenString {
				msg = "string: " + tok.Value
			}
			r := spanToLSPRange(parse.TokenSpan(tok))
			return &Hover{Contents: msg, Range: &r}
		}
	}
	return nil
}

func lineColToOffset(content string, line, col int) int {
	curLine, curCol := 1, 1
	for i := 0; i < len(content); i++ {
		if curLine == line && curCol == col {
			return i
		}
		if content[i] == '\n' {
			curLine++
			curCol = 1
		} else {
			curCol++
		}
	}
	return len(content)
}

func handleDefinition(store *docStore, params *any) *Location {
	if params == nil {
		return nil
	}
	b, _ := json.Marshal(params)
	var p DefinitionParams
	if json.Unmarshal(b, &p) != nil {
		return nil
	}
	content := store.get(p.TextDocument.URI)
	if content == "" {
		return nil
	}
	// Find identifier at position
	line := p.Position.Line + 1
	col := p.Position.Character + 1
	offset := lineColToOffset(content, line, col)
	lx := lex.NewLexer(content)
	var identToken lex.Token
	found := false
	for {
		tok := lx.Next()
		if tok.Kind == lex.TokenEOF {
			break
		}
		if tok.Kind == lex.TokenIdent {
			start := tok.Offset
			end := tok.Offset + len(tok.Value)
			if offset >= start && offset < end {
				identToken = tok
				found = true
				break
			}
		}
	}
	if !found {
		return nil
	}
	ident := identToken.Value

	// Find definition: "def ident", "var ident", "for ident in"
	defSpan := findDefinitionInSource(content, ident)
	if defSpan != nil {
		r := spanToLSPRange(*defSpan)
		return &Location{URI: p.TextDocument.URI, Range: r}
	}
	return nil
}

func findDefinitionInSource(content string, ident string) *diag.Span {
	lx := lex.NewLexer(content)
	for {
		tok := lx.Next()
		if tok.Kind == lex.TokenEOF {
			break
		}
		if tok.Kind != lex.TokenKeyword {
			continue
		}
		switch tok.Value {
		case "def", "var":
			// Next non-whitespace token should be identifier
			for {
				n := lx.Next()
				if n.Kind == lex.TokenEOF {
					break
				}
				if n.Kind == lex.TokenWhitespace || n.Kind == lex.TokenComment {
					continue
				}
				if n.Kind == lex.TokenIdent && n.Value == ident {
					span := parse.TokenSpan(n)
					return &span
				}
				break
			}
		case "for":
			for {
				n := lx.Next()
				if n.Kind == lex.TokenEOF {
					break
				}
				if n.Kind == lex.TokenWhitespace || n.Kind == lex.TokenComment {
					continue
				}
				if n.Kind == lex.TokenIdent && n.Value == ident {
					span := parse.TokenSpan(n)
					return &span
				}
				break
			}
		}
	}
	return nil
}

// getDiagnostics runs parser and semantic check and returns LSP diagnostics.
func getDiagnostics(uri string, content string) []Diagnostic {
	var out []Diagnostic
	parser := parse.NewParser(content)
	mod := parser.ParseModule()
	for _, err := range parser.Errors() {
		if pe, ok := err.(*parse.ParseError); ok {
			r := spanToLSPRange(pe.Span)
			out = append(out, Diagnostic{
				Range:    r,
				Message:  pe.Msg,
				Severity: 1,
				Source:   "rayo",
			})
		} else {
			out = append(out, Diagnostic{
				Range:    Range{},
				Message:  err.Error(),
				Severity: 1,
				Source:   "rayo",
			})
		}
	}
	rep := &collectingReporter{}
	sem.CheckModule(mod, rep)
	for _, d := range rep.diags {
		out = append(out, Diagnostic{
			Range:    spanToLSPRange(d.span),
			Message:  d.msg,
			Severity: lspSeverity(d.sev),
			Source:   "rayo",
		})
	}
	return out
}

// lspSeverity maps a diag.Severity to the LSP DiagnosticSeverity numbering
// (1=Error, 2=Warning, 3=Information, 4=Hint).
func lspSeverity(s diag.Severity) int {
	switch s {
	case diag.SeverityWarning:
		return 2
	case diag.SeverityInfo:
		return 3
	default:
		return 1
	}
}

type semDiag struct {
	span diag.Span
	sev  diag.Severity
	msg  string
}

// collectingReporter implements diag.SeverityReporter so severities set by the
// semantic checker are preserved instead of being guessed from message text.
type collectingReporter struct {
	diags []semDiag
}

func (c *collectingReporter) Report(span diag.Span, msg string) {
	c.ReportAt(span, diag.SeverityError, msg)
}

func (c *collectingReporter) ReportAt(span diag.Span, sev diag.Severity, msg string) {
	c.diags = append(c.diags, semDiag{span: span, sev: sev, msg: msg})
}
