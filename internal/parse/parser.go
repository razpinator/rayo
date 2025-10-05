package parse

import (
	"rayo/internal/ast"
	"rayo/internal/diag"
	"rayo/internal/lex"
)

// Parser implements a recursive-descent parser for Rayo.
type Parser struct {
	lx     *lex.Lexer
	tok    lex.Token
	errors []error
}

func NewParser(src string) *Parser {
	lx := lex.NewLexer(src)
	p := &Parser{lx: lx}
	p.next()
	return p
}

func (p *Parser) Errors() []error {
	return p.errors
}

func (p *Parser) parseFuncDef() ast.Stmt {
	p.expect(lex.TokenKeyword) // 'def'

	// Skip whitespace
	for p.tok.Kind == lex.TokenWhitespace {
		p.next()
	}

	// Function name
	if p.tok.Kind != lex.TokenIdent {
		err := &ParseError{Msg: "expected function name", Span: diag.Span{}, Expected: []string{"identifier"}, Excerpt: p.tok.Value}
		p.errors = append(p.errors, err)
		return nil
	}

	name := p.tok.Value
	p.next()

	// Skip whitespace
	for p.tok.Kind == lex.TokenWhitespace {
		p.next()
	}

	// Parameters '(' ... ')'
	if p.tok.Kind != lex.TokenLParen {
		err := &ParseError{Msg: "expected '(' after function name", Span: diag.Span{}, Expected: []string{"("}, Excerpt: p.tok.Value}
		p.errors = append(p.errors, err)
		return nil
	}
	p.next()

	// For now, skip parameter parsing and just expect ')'
	for p.tok.Kind != lex.TokenRParen && p.tok.Kind != 0 { // 0 is EOF
		p.next()
	}

	if p.tok.Kind == lex.TokenRParen {
		p.next()
	}

	// Skip whitespace
	for p.tok.Kind == lex.TokenWhitespace {
		p.next()
	}

	// Function body '{ ... }'
	if p.tok.Kind != lex.TokenLBrace {
		err := &ParseError{Msg: "expected '{' for function body", Span: diag.Span{}, Expected: []string{"{"}, Excerpt: p.tok.Value}
		p.errors = append(p.errors, err)
		return nil
	}
	p.next()

	// For now, skip body parsing and just expect '}'
	braceCount := 1
	for braceCount > 0 && p.tok.Kind != 0 { // 0 is EOF
		if p.tok.Kind == lex.TokenLBrace {
			braceCount++
		} else if p.tok.Kind == lex.TokenRBrace {
			braceCount--
		}
		if braceCount > 0 {
			p.next()
		}
	}

	if p.tok.Kind == lex.TokenRBrace {
		p.next()
	}

	// Return a basic function definition
	return &ast.FuncDef{
		Name:   name,
		Params: []*ast.Param{}, // Empty for now
		Body:   []ast.Stmt{},   // Empty for now
	}
}

func (p *Parser) next() {
	for {
		p.tok = p.lx.Next()
		if p.tok.Kind != lex.TokenWhitespace {
			break
		}
	}
}

func (p *Parser) expect(kind lex.TokenKind) lex.Token {
	if p.tok.Kind != kind {
		err := &ParseError{Msg: "unexpected token", Span: diag.Span{}, Expected: []string{kindToString(kind)}, Excerpt: p.tok.Value}
		p.errors = append(p.errors, err)
	}
	tok := p.tok
	p.next()
	return tok
}

func kindToString(kind lex.TokenKind) string {
	switch kind {
	case lex.TokenIdent:
		return "identifier"
	case lex.TokenKeyword:
		return "keyword"
	case lex.TokenNumber:
		return "number"
	case lex.TokenString:
		return "string"
	case lex.TokenLBrace:
		return "{"
	case lex.TokenRBrace:
		return "}"
	// ...extend as needed...
	default:
		return "token"
	}
}

// ParseModule parses a module.
func (p *Parser) ParseModule() *ast.Module {
	mod := &ast.Module{Imports: []*ast.Import{}, Body: []ast.Stmt{}}
	// Example: parse imports and body
	for p.tok.Kind != lex.TokenEOF {
		// Skip any whitespace tokens between statements
		for p.tok.Kind == lex.TokenWhitespace {
			p.next()
		}
		if p.tok.Kind == lex.TokenEOF {
			break
		}
		// Handle imports
		if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "import" {
			imp := p.parseImport()
			mod.Imports = append(mod.Imports, imp)
			continue
		}
		// Try to parse a statement
		if p.tok.Kind != lex.TokenWhitespace {
			stmt := p.parseStmt()
			if stmt != nil {
				mod.Body = append(mod.Body, stmt)
				continue
			}
		}
		// If not a statement or failed to parse, advance token
		p.next()
	}
	return mod
}

func (p *Parser) parseImport() *ast.Import {
	p.expect(lex.TokenKeyword) // 'import'
	pathTok := p.expect(lex.TokenString)
	val := pathTok.Value
	if len(val) >= 2 && (val[0] == '\'' || val[0] == '"') && val[len(val)-1] == val[0] {
		val = val[1 : len(val)-1]
	}
	return &ast.Import{Path: val}
}

func (p *Parser) parseStmt() ast.Stmt {
	// Parse function definition
	if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "def" {
		return p.parseFuncDef()
	}

	// Example: parse var statement
	if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "var" {
		p.next()
		// Skip whitespace after 'var'
		for p.tok.Kind == lex.TokenWhitespace {
			p.next()
		}
		// Expect identifier
		if p.tok.Kind != lex.TokenIdent {
			err := &ParseError{Msg: "expected identifier after 'var'", Span: diag.Span{}, Expected: []string{"identifier"}, Excerpt: p.tok.Value}
			p.errors = append(p.errors, err)
			return nil
		}
		nameTok := p.tok
		p.next()
		// Skip whitespace after identifier
		for p.tok.Kind == lex.TokenWhitespace {
			p.next()
		}
		// Expect '=' operator
		if p.tok.Kind != lex.TokenOp || p.tok.Value != "=" {
			err := &ParseError{Msg: "expected '=' after variable name", Span: diag.Span{}, Expected: []string{"="}, Excerpt: p.tok.Value}
			p.errors = append(p.errors, err)
			return nil
		}
		p.next()
		// Skip whitespace after '='
		for p.tok.Kind == lex.TokenWhitespace {
			p.next()
		}
		val := p.parseExpr()
		return &ast.VarStmt{Name: nameTok.Value, Value: val}
	}
	p.next() // advance token if not a statement start
	// ...extend for other statements...
	return nil
}

func (p *Parser) parseExpr() ast.Expr {
	// Example: parse literal or name
	switch p.tok.Kind {
	case lex.TokenNumber:
		tok := p.tok
		p.next()
		return &ast.Literal{Value: tok.Value}
	case lex.TokenIdent:
		tok := p.tok
		p.next()
		return &ast.Name{Ident: tok.Value}
	case lex.TokenString:
		tok := p.tok
		val := tok.Value
		if len(val) >= 2 && (val[0] == '\'' || val[0] == '"') && val[len(val)-1] == val[0] {
			val = val[1 : len(val)-1]
		}
		p.next()
		return &ast.Literal{Value: val}
	}
	return nil
}
