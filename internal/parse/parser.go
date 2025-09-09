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
        if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "import" {
            imp := p.parseImport()
            mod.Imports = append(mod.Imports, imp)
            continue
        }
        // Try to parse a statement for any non-import, non-whitespace, non-EOF token
        if p.tok.Kind != lex.TokenEOF && p.tok.Kind != lex.TokenWhitespace && !(p.tok.Kind == lex.TokenKeyword && p.tok.Value == "import") {
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
        val = val[1:len(val)-1]
    }
    return &ast.Import{Path: val}
}

func (p *Parser) parseStmt() ast.Stmt {
    // Example: parse var statement
    println("parseStmt: starting with token kind=", p.tok.Kind, "value=", p.tok.Value)
    if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "var" {
        println("parseStmt: found 'var' keyword")
        println("parseStmt: before next(): kind=", p.tok.Kind, "value=", p.tok.Value)
        p.next()
        println("parseStmt: after next(): kind=", p.tok.Kind, "value=", p.tok.Value)
        // Skip whitespace after 'var'
        for p.tok.Kind == lex.TokenWhitespace {
            println("parseStmt: skipping whitespace: kind=", p.tok.Kind, "value=", p.tok.Value)
            println("parseStmt: before next(): kind=", p.tok.Kind, "value=", p.tok.Value)
            p.next()
            println("parseStmt: after next(): kind=", p.tok.Kind, "value=", p.tok.Value)
        }
        println("parseStmt: after whitespace, looking for identifier: kind=", p.tok.Kind, "value=", p.tok.Value)
        // Expect identifier
        if p.tok.Kind != lex.TokenIdent {
            println("parseStmt: ERROR: expected identifier, got kind=", p.tok.Kind, "value=", p.tok.Value)
            err := &ParseError{Msg: "expected identifier after 'var'", Span: diag.Span{}, Expected: []string{"identifier"}, Excerpt: p.tok.Value}
            p.errors = append(p.errors, err)
            return nil
        }
        println("parseStmt: found identifier: kind=", p.tok.Kind, "value=", p.tok.Value)
        nameTok := p.tok
        println("parseStmt: before next(): kind=", p.tok.Kind, "value=", p.tok.Value)
        p.next()
        println("parseStmt: after next(): kind=", p.tok.Kind, "value=", p.tok.Value)
        // Skip whitespace after identifier
        for p.tok.Kind == lex.TokenWhitespace {
            println("parseStmt: skipping whitespace: kind=", p.tok.Kind, "value=", p.tok.Value)
            println("parseStmt: before next(): kind=", p.tok.Kind, "value=", p.tok.Value)
            p.next()
            println("parseStmt: after next(): kind=", p.tok.Kind, "value=", p.tok.Value)
        }
        println("parseStmt: after whitespace, looking for =: kind=", p.tok.Kind, "value=", p.tok.Value)
        // Expect '=' operator
        if p.tok.Kind != lex.TokenOp || p.tok.Value != "=" {
            println("parseStmt: ERROR: expected '=', got kind=", p.tok.Kind, "value=", p.tok.Value)
            err := &ParseError{Msg: "expected '=' after variable name", Span: diag.Span{}, Expected: []string{"="}, Excerpt: p.tok.Value}
            p.errors = append(p.errors, err)
            return nil
        }
        println("parseStmt: found =: kind=", p.tok.Kind, "value=", p.tok.Value)
        println("parseStmt: before next(): kind=", p.tok.Kind, "value=", p.tok.Value)
        p.next()
        println("parseStmt: after next(): kind=", p.tok.Kind, "value=", p.tok.Value)
        // Skip whitespace after '='
        for p.tok.Kind == lex.TokenWhitespace {
            println("parseStmt: skipping whitespace: kind=", p.tok.Kind, "value=", p.tok.Value)
            println("parseStmt: before next(): kind=", p.tok.Kind, "value=", p.tok.Value)
            p.next()
            println("parseStmt: after next(): kind=", p.tok.Kind, "value=", p.tok.Value)
        }
        println("parseStmt: after whitespace, looking for expression: kind=", p.tok.Kind, "value=", p.tok.Value)
        val := p.parseExpr()
        println("parseStmt: parsed expression:", val)
        return &ast.VarStmt{Name: nameTok.Value, Value: val}
    }
    println("parseStmt: not a 'var' statement: kind=", p.tok.Kind, "value=", p.tok.Value)
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
            val = val[1:len(val)-1]
        }
        p.next()
        return &ast.Literal{Value: val}
    }
    return nil
}
