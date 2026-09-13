package lsp

import (
	"bytes"
	"encoding/json"
	"testing"
)

// toParams marshals v into the *any shape the dispatch layer passes to handlers.
func toParams(t *testing.T, v any) *any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	var p any
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	return &p
}

const sampleDoc = `import "fmt"
def greet(name) {
    var msg = 1
    print(msg)
}
def main() {
    greet(2)
}`

func TestCompletionIncludesKeywordsNamesAndSnippets(t *testing.T) {
	store := newDocStore()
	uri := "file:///a.ryo"
	store.set(uri, sampleDoc)

	params := toParams(t, CompletionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     Position{Line: 3, Character: 4},
	})
	list := handleCompletion(store, params, true)
	if list == nil || len(list.Items) == 0 {
		t.Fatal("expected completion items")
	}

	labels := map[string]CompletionItem{}
	for _, it := range list.Items {
		labels[it.Label] = it
	}

	// Keyword present.
	if _, ok := labels["return"]; !ok {
		t.Error("expected keyword 'return' in completions")
	}
	// In-scope declared names present.
	if it, ok := labels["greet"]; !ok || it.Kind != CompletionKindFunction {
		t.Error("expected function 'greet' in completions")
	}
	if it, ok := labels["msg"]; !ok || it.Kind != CompletionKindVariable {
		t.Error("expected variable 'msg' in completions")
	}
	// Snippets present with snippet insert format when supported.
	for _, want := range []string{"def", "if", "try"} {
		it, ok := labels[want]
		if !ok || it.Kind != CompletionKindSnippet {
			t.Errorf("expected snippet %q", want)
			continue
		}
		if it.InsertTextFormat != InsertFormatSnippet {
			t.Errorf("snippet %q: expected snippet insert format", want)
		}
	}
}

func TestCompletionFallsBackToPlainSnippets(t *testing.T) {
	store := newDocStore()
	uri := "file:///a.ryo"
	store.set(uri, sampleDoc)

	params := toParams(t, CompletionParams{TextDocument: TextDocumentIdentifier{URI: uri}})
	list := handleCompletion(store, params, false)
	for _, it := range list.Items {
		if it.Kind == CompletionKindSnippet && it.InsertTextFormat == InsertFormatSnippet {
			t.Errorf("snippet %q should be plain text when snippetSupport=false", it.Label)
		}
	}
}

func TestReferencesFindsAllOccurrences(t *testing.T) {
	store := newDocStore()
	uri := "file:///a.ryo"
	store.set(uri, sampleDoc)

	// Cursor on "greet" in the definition (line 1, 0-based).
	params := toParams(t, ReferenceParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     Position{Line: 1, Character: 4},
		Context:      ReferenceContext{IncludeDeclaration: true},
	})
	locs := handleReferences(store, params)
	// "greet" appears in def and in the call inside main() => 2 occurrences.
	if len(locs) != 2 {
		t.Fatalf("expected 2 references to greet, got %d: %v", len(locs), locs)
	}

	// Without declaration, the def occurrence is dropped.
	params = toParams(t, ReferenceParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     Position{Line: 1, Character: 4},
		Context:      ReferenceContext{IncludeDeclaration: false},
	})
	locs = handleReferences(store, params)
	if len(locs) != 1 {
		t.Fatalf("expected 1 reference to greet without declaration, got %d", len(locs))
	}
}

func TestDocumentSymbolHierarchy(t *testing.T) {
	store := newDocStore()
	uri := "file:///a.ryo"
	store.set(uri, sampleDoc)

	params := toParams(t, DocumentSymbolParams{TextDocument: TextDocumentIdentifier{URI: uri}})
	syms := handleDocumentSymbol(store, params)

	names := map[string]DocumentSymbol{}
	for _, s := range syms {
		names[s.Name] = s
	}
	if _, ok := names["greet"]; !ok {
		t.Error("expected top-level symbol 'greet'")
	}
	if _, ok := names["main"]; !ok {
		t.Error("expected top-level symbol 'main'")
	}
	if names["greet"].Kind != SymbolKindFunction {
		t.Error("expected greet to be a function symbol")
	}
	// 'msg' is declared inside greet -> should be a child.
	foundChild := false
	for _, c := range names["greet"].Children {
		if c.Name == "msg" {
			foundChild = true
		}
	}
	if !foundChild {
		t.Error("expected 'msg' as child symbol of greet")
	}
}

func TestWorkspaceSymbolAcrossDocs(t *testing.T) {
	store := newDocStore()
	store.set("file:///a.ryo", `def alpha() { }`)
	store.set("file:///b.ryo", `def beta() { }`)

	// Query filters by substring.
	params := toParams(t, WorkspaceSymbolParams{Query: "alp"})
	syms := handleWorkspaceSymbol(store, params)
	if len(syms) != 1 || syms[0].Name != "alpha" {
		t.Fatalf("expected only 'alpha' for query 'alp', got %v", syms)
	}

	// Empty query returns all.
	params = toParams(t, WorkspaceSymbolParams{Query: ""})
	syms = handleWorkspaceSymbol(store, params)
	if len(syms) != 2 {
		t.Fatalf("expected 2 symbols for empty query, got %d", len(syms))
	}
}

// newBufferedSession returns a session whose encoder writes into buf, so tests
// can decode the last emitted JSON-RPC response.
func newBufferedSession() (*session, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	return &session{enc: enc, store: newDocStore()}, buf
}

func decodeResponse(t *testing.T, buf *bytes.Buffer) jsonRPCResponse {
	t.Helper()
	var resp jsonRPCResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("decode response %q: %v", buf.String(), err)
	}
	return resp
}

func TestDispatchRejectsBeforeInitialize(t *testing.T) {
	sess, buf := newBufferedSession()
	id := any(float64(1))
	req := &jsonRPCRequest{JSONRPC: "2.0", ID: &id, Method: "textDocument/completion"}
	if dispatch(sess, req) {
		t.Fatal("dispatch should not request exit")
	}
	resp := decodeResponse(t, buf)
	if resp.Error == nil || resp.Error.Code != codeServerNotInit {
		t.Fatalf("expected server-not-initialized error, got %+v", resp)
	}
}

func TestDispatchUnknownMethod(t *testing.T) {
	sess, buf := newBufferedSession()
	sess.initialized = true
	id := any(float64(7))
	req := &jsonRPCRequest{JSONRPC: "2.0", ID: &id, Method: "textDocument/bogus"}
	dispatch(sess, req)
	resp := decodeResponse(t, buf)
	if resp.Error == nil || resp.Error.Code != codeMethodNotFound {
		t.Fatalf("expected method-not-found error, got %+v", resp)
	}
}

func TestDispatchExitStopsServing(t *testing.T) {
	sess, _ := newBufferedSession()
	sess.initialized = true
	req := &jsonRPCRequest{JSONRPC: "2.0", Method: "exit"}
	if !dispatch(sess, req) {
		t.Fatal("expected exit to request stopping the serve loop")
	}
}
