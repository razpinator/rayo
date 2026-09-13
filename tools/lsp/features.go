package lsp

import (
	"encoding/json"
	"sort"
	"strings"

	"rayo/internal/ast"
	"rayo/internal/diag"
	"rayo/internal/lex"
	"rayo/internal/parse"
)

// symbol describes a named declaration discovered in a document.
type symbol struct {
	name string
	kind int // SymbolKind*
	span diag.Span
}

// collectSymbols parses content and returns every top-level and nested
// declaration (functions, their parameters, variables, and loop bindings).
// It tolerates parse errors, walking whatever AST the parser produced.
func collectSymbols(content string) []symbol {
	parser := parse.NewParser(content)
	mod := parser.ParseModule()
	if mod == nil {
		return nil
	}
	c := &symbolCollector{}
	ast.Walk(c, mod)
	return c.symbols
}

type symbolCollector struct {
	symbols []symbol
}

func (c *symbolCollector) Visit(n ast.Node) bool {
	switch x := n.(type) {
	case *ast.FuncDef:
		c.symbols = append(c.symbols, symbol{name: x.Name, kind: SymbolKindFunction, span: x.Span()})
	case *ast.Param:
		if x.Name != "" {
			c.symbols = append(c.symbols, symbol{name: x.Name, kind: SymbolKindVariable, span: x.Span()})
		}
	case *ast.VarStmt:
		c.symbols = append(c.symbols, symbol{name: x.Name, kind: SymbolKindVariable, span: x.Span()})
	case *ast.ForStmt:
		if x.Var != "" {
			c.symbols = append(c.symbols, symbol{name: x.Var, kind: SymbolKindVariable, span: x.Span()})
		}
	}
	return true
}

// handleCompletion returns keyword completions, in-scope names collected from
// the document, and snippet shapes for def/if/try. When the client did not
// advertise snippet support, snippets fall back to plain-text insertions.
func handleCompletion(store *docStore, params *any, snippetSupport bool) *CompletionList {
	list := &CompletionList{IsIncomplete: false, Items: []CompletionItem{}}
	if params == nil {
		return list
	}
	b, _ := json.Marshal(params)
	var p CompletionParams
	if json.Unmarshal(b, &p) != nil {
		return list
	}
	content := store.get(p.TextDocument.URI)

	// Keywords (kept in sync with the lexer).
	for _, kw := range lex.Keywords() {
		list.Items = append(list.Items, CompletionItem{
			Label:    kw,
			Kind:     CompletionKindKeyword,
			Detail:   "keyword",
			SortText: "2_" + kw,
		})
	}

	// Names in scope: declared symbols in this document, de-duplicated.
	seen := map[string]bool{}
	for _, s := range collectSymbols(content) {
		if s.name == "" || seen[s.name] {
			continue
		}
		seen[s.name] = true
		kind := CompletionKindVariable
		detail := "variable"
		if s.kind == SymbolKindFunction {
			kind = CompletionKindFunction
			detail = "function"
		}
		list.Items = append(list.Items, CompletionItem{
			Label:    s.name,
			Kind:     kind,
			Detail:   detail,
			SortText: "1_" + s.name,
		})
	}

	// Snippet shapes.
	list.Items = append(list.Items, snippetItems(snippetSupport)...)
	return list
}

func snippetItems(snippetSupport bool) []CompletionItem {
	type snip struct {
		label, detail, snippet, plain string
	}
	snips := []snip{
		{
			label:   "def",
			detail:  "function definition",
			snippet: "def ${1:name}(${2:params}) {\n\t${0}\n}",
			plain:   "def name(params) {\n\t\n}",
		},
		{
			label:   "if",
			detail:  "if statement",
			snippet: "if ${1:cond} {\n\t${0}\n}",
			plain:   "if cond {\n\t\n}",
		},
		{
			label:   "try",
			detail:  "try/except block",
			snippet: "try {\n\t${1}\n} except ${2:e} {\n\t${0}\n}",
			plain:   "try {\n\t\n} except e {\n\t\n}",
		},
	}
	out := make([]CompletionItem, 0, len(snips))
	for _, s := range snips {
		item := CompletionItem{
			Label:    s.label,
			Kind:     CompletionKindSnippet,
			Detail:   s.detail,
			SortText: "0_" + s.label,
		}
		if snippetSupport {
			item.InsertText = s.snippet
			item.InsertTextFormat = InsertFormatSnippet
		} else {
			item.InsertText = s.plain
			item.InsertTextFormat = InsertFormatPlainText
		}
		out = append(out, item)
	}
	return out
}

// handleReferences returns every occurrence of the identifier under the cursor
// within the current document. When includeDeclaration is false the declaring
// occurrence(s) are omitted.
func handleReferences(store *docStore, params *any) []Location {
	if params == nil {
		return nil
	}
	b, _ := json.Marshal(params)
	var p ReferenceParams
	if json.Unmarshal(b, &p) != nil {
		return nil
	}
	content := store.get(p.TextDocument.URI)
	if content == "" {
		return nil
	}

	ident := identAt(content, p.Position)
	if ident == "" {
		return nil
	}

	declOffsets := map[int]bool{}
	if !p.Context.IncludeDeclaration {
		declOffsets = declarationNameOffsets(content, ident)
	}

	var locs []Location
	lx := lex.NewLexer(content)
	for {
		tok := lx.Next()
		if tok.Kind == lex.TokenEOF {
			break
		}
		if tok.Kind != lex.TokenIdent || tok.Value != ident {
			continue
		}
		if declOffsets[tok.Offset] {
			continue
		}
		locs = append(locs, Location{URI: p.TextDocument.URI, Range: spanToLSPRange(parse.TokenSpan(tok))})
	}
	return locs
}

// handleDocumentSymbol returns the hierarchical symbol tree for a document:
// functions at the top level with their parameters/locals as children.
func handleDocumentSymbol(store *docStore, params *any) []DocumentSymbol {
	if params == nil {
		return nil
	}
	b, _ := json.Marshal(params)
	var p DocumentSymbolParams
	if json.Unmarshal(b, &p) != nil {
		return nil
	}
	content := store.get(p.TextDocument.URI)
	if content == "" {
		return nil
	}

	parser := parse.NewParser(content)
	mod := parser.ParseModule()
	if mod == nil {
		return nil
	}

	var out []DocumentSymbol
	for _, stmt := range mod.Body {
		switch s := stmt.(type) {
		case *ast.FuncDef:
			fn := DocumentSymbol{
				Name:           s.Name,
				Kind:           SymbolKindFunction,
				Range:          spanToLSPRange(s.Span()),
				SelectionRange: spanToLSPRange(s.Span()),
			}
			child := &symbolCollector{}
			for _, param := range s.Params {
				ast.Walk(child, param)
			}
			for _, bs := range s.Body {
				ast.Walk(child, bs)
			}
			for _, sym := range child.symbols {
				if sym.name == "" {
					continue
				}
				fn.Children = append(fn.Children, DocumentSymbol{
					Name:           sym.name,
					Kind:           sym.kind,
					Range:          spanToLSPRange(sym.span),
					SelectionRange: spanToLSPRange(sym.span),
				})
			}
			out = append(out, fn)
		case *ast.VarStmt:
			out = append(out, DocumentSymbol{
				Name:           s.Name,
				Kind:           SymbolKindVariable,
				Range:          spanToLSPRange(s.Span()),
				SelectionRange: spanToLSPRange(s.Span()),
			})
		}
	}
	return out
}

// handleWorkspaceSymbol searches declarations across all open documents,
// filtering by a case-insensitive substring query. An empty query returns all.
func handleWorkspaceSymbol(store *docStore, params *any) []SymbolInformation {
	if params == nil {
		return nil
	}
	b, _ := json.Marshal(params)
	var p WorkspaceSymbolParams
	if json.Unmarshal(b, &p) != nil {
		return nil
	}
	query := strings.ToLower(p.Query)

	var out []SymbolInformation
	docs := store.all()
	// Deterministic ordering by URI then position.
	uris := make([]string, 0, len(docs))
	for uri := range docs {
		uris = append(uris, uri)
	}
	sort.Strings(uris)

	for _, uri := range uris {
		for _, s := range collectSymbols(docs[uri]) {
			if s.name == "" {
				continue
			}
			if query != "" && !strings.Contains(strings.ToLower(s.name), query) {
				continue
			}
			out = append(out, SymbolInformation{
				Name:     s.name,
				Kind:     s.kind,
				Location: Location{URI: uri, Range: spanToLSPRange(s.span)},
			})
		}
	}
	return out
}

// declarationNameOffsets returns the byte offsets of identifier tokens that
// introduce a binding named ident (the name token following def/var/for).
// These are the "declaration" occurrences that find-references omits when
// includeDeclaration is false.
func declarationNameOffsets(content, ident string) map[int]bool {
	out := map[int]bool{}
	lx := lex.NewLexer(content)
	expectName := false
	for {
		tok := lx.Next()
		if tok.Kind == lex.TokenEOF {
			break
		}
		if tok.Kind == lex.TokenWhitespace || tok.Kind == lex.TokenComment {
			continue
		}
		if expectName {
			if tok.Kind == lex.TokenIdent && tok.Value == ident {
				out[tok.Offset] = true
			}
			expectName = false
			continue
		}
		if tok.Kind == lex.TokenKeyword {
			switch tok.Value {
			case "def", "var", "for":
				expectName = true
			}
		}
	}
	return out
}

// identAt returns the identifier token covering the given LSP position, or "".
func identAt(content string, pos Position) string {
	offset := lineColToOffset(content, pos.Line+1, pos.Character+1)
	lx := lex.NewLexer(content)
	for {
		tok := lx.Next()
		if tok.Kind == lex.TokenEOF {
			break
		}
		if tok.Kind != lex.TokenIdent {
			continue
		}
		if offset >= tok.Offset && offset < tok.Offset+len(tok.Value) {
			return tok.Value
		}
	}
	return ""
}
