package lsp

// LSP types (JSON-RPC 2.0 and Language Server Protocol 3.16)

// JSON-RPC
type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      *any   `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  *any   `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    any    `json:"data,omitempty"`
	} `json:"error,omitempty"`
}

// LSP Position (0-based line, 0-based character)
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// Initialize
type InitializeParams struct {
	ProcessID    int                `json:"processId,omitempty"`
	RootURI      string             `json:"rootUri,omitempty"`
	Capabilities ClientCapabilities `json:"capabilities,omitempty"`
}

// ClientCapabilities carries the subset of client capabilities the server
// inspects during negotiation. Unknown fields are ignored by the JSON decoder.
type ClientCapabilities struct {
	TextDocument struct {
		Completion struct {
			CompletionItem struct {
				SnippetSupport bool `json:"snippetSupport"`
			} `json:"completionItem"`
		} `json:"completion"`
		Hover struct {
			ContentFormat []string `json:"contentFormat"`
		} `json:"hover"`
		DocumentSymbol struct {
			HierarchicalDocumentSymbolSupport bool `json:"hierarchicalDocumentSymbolSupport"`
		} `json:"documentSymbol"`
	} `json:"textDocument"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

type ServerCapabilities struct {
	TextDocumentSync        *int               `json:"textDocumentSync,omitempty"` // 1 = full
	HoverProvider           bool               `json:"hoverProvider"`
	DefinitionProvider      bool               `json:"definitionProvider"`
	CompletionProvider      *CompletionOptions `json:"completionProvider,omitempty"`
	ReferencesProvider      bool               `json:"referencesProvider"`
	DocumentSymbolProvider  bool               `json:"documentSymbolProvider"`
	WorkspaceSymbolProvider bool               `json:"workspaceSymbolProvider"`
}

// CompletionOptions advertises completion capabilities during negotiation.
type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
	ResolveProvider   bool     `json:"resolveProvider"`
}

// TextDocument
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version,omitempty"`
}

type TextDocumentContentChangeEvent struct {
	Range       *Range `json:"range,omitempty"`
	RangeLength *int   `json:"rangeLength,omitempty"`
	Text        string `json:"text"`
}

type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// Hover
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type Hover struct {
	Contents string `json:"contents"` // string or MarkedString; we use string
	Range    *Range `json:"range,omitempty"`
}

// Definition
type DefinitionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// PublishDiagnostics (notification from server to client)
type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type Diagnostic struct {
	Range    Range  `json:"range"`
	Message  string `json:"message"`
	Severity int    `json:"severity,omitempty"` // 1=Error, 2=Warning
	Source   string `json:"source,omitempty"`
}

// Completion

// CompletionItemKind values (LSP 3.16).
const (
	CompletionKindText     = 1
	CompletionKindFunction = 3
	CompletionKindVariable = 6
	CompletionKindKeyword  = 14
	CompletionKindSnippet  = 15
)

// InsertTextFormat values.
const (
	InsertFormatPlainText = 1
	InsertFormatSnippet   = 2
)

type CompletionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type CompletionItem struct {
	Label            string `json:"label"`
	Kind             int    `json:"kind,omitempty"`
	Detail           string `json:"detail,omitempty"`
	Documentation    string `json:"documentation,omitempty"`
	InsertText       string `json:"insertText,omitempty"`
	InsertTextFormat int    `json:"insertTextFormat,omitempty"`
	SortText         string `json:"sortText,omitempty"`
}

type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

// References

type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

type ReferenceParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      ReferenceContext       `json:"context"`
}

// Document / workspace symbols

// SymbolKind values (LSP 3.16, subset used here).
const (
	SymbolKindFunction = 12
	SymbolKindVariable = 13
)

type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DocumentSymbol is the hierarchical form returned for textDocument/documentSymbol.
type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           int              `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

type WorkspaceSymbolParams struct {
	Query string `json:"query"`
}

// SymbolInformation is the flat form returned for workspace/symbol.
type SymbolInformation struct {
	Name          string   `json:"name"`
	Kind          int      `json:"kind"`
	Location      Location `json:"location"`
	ContainerName string   `json:"containerName,omitempty"`
}
