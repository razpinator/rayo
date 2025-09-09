package parse

import (
    "functure/internal/lex"
    "functure/internal/ast"
    "functure/internal/diag"
)

// Parser implements a recursive-descent parser for Functure.
type Parser struct {
    lx    *lex.Lexer
    tok   lex.Token
    errors []error
}

func NewParser(src string) *Parser {
    lx := lex.NewLexer(src)
    p := &Parser{lx: lx}
    p.next()
    return p
}

func (p *Parser) next() {
    p.tok = p.lx.Next()
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
        if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "import" {
            imp := p.parseImport()
            mod.Imports = append(mod.Imports, imp)
        } else {
            stmt := p.parseStmt()
            if stmt != nil {
                mod.Body = append(mod.Body, stmt)
            } else {
                p.next() // error recovery
            }
        }
    }
    return mod
}

func (p *Parser) parseImport() *ast.Import {
    p.expect(lex.TokenKeyword) // 'import'
    pathTok := p.expect(lex.TokenString)
    return &ast.Import{Path: pathTok.Value}
}

func (p *Parser) parseStmt() ast.Stmt {
    // Example: parse var statement
    if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "var" {
        p.next()
        nameTok := p.expect(lex.TokenIdent)
        p.expect(lex.TokenOp) // '='
        val := p.parseExpr()
        return &ast.VarStmt{Name: nameTok.Value, Value: val}
    }
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
    }
    return nil
}
