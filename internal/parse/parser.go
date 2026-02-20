package parse

import (
	"rayo/internal/ast"
	"rayo/internal/diag"
	"rayo/internal/lex"
	"strconv"
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
	body := p.parseBlock()

	// Return a basic function definition
	return &ast.FuncDef{
		Name:   name,
		Params: []*ast.Param{}, // Empty for now
		Body:   body,
	}
}

func (p *Parser) parseBlock() []ast.Stmt {
	p.expect(lex.TokenLBrace)
	var stmts []ast.Stmt
	for p.tok.Kind != lex.TokenRBrace && p.tok.Kind != lex.TokenEOF {
		// Skip semicolons or newlines if we had them, but for now just whitespace loop in next() handles it.
		// However, we might want to be robust.
		if p.tok.Kind == lex.TokenWhitespace {
			p.next()
			continue
		}
		stmt := p.parseStmt()
		if stmt != nil {
			stmts = append(stmts, stmt)
		} else {
			// If we can't parse a statement, we should probably skip a token to avoid infinite loop
			// or break if it looks like end of block
			if p.tok.Kind == lex.TokenRBrace {
				break
			}
			p.next()
		}
	}
	p.expect(lex.TokenRBrace)
	return stmts
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
	// Skip whitespace
	for p.tok.Kind == lex.TokenWhitespace {
		p.next()
	}

	if p.tok.Kind == lex.TokenEOF || p.tok.Kind == lex.TokenRBrace {
		return nil
	}

	// Function definition: def name() { ... }
	if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "def" {
		return p.parseFuncDef()
	}

	// Return statement
	if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "return" {
		p.next()
		var val ast.Expr
		// If next token is not newline/semicolon/RBrace, parse expression
		// For now assume if not RBrace, try parse expression
		if p.tok.Kind != lex.TokenRBrace && p.tok.Kind != lex.TokenEOF {
			val = p.parseExpr()
		}
		return &ast.ReturnStmt{Value: val}
	}

	// If statement
	if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "if" {
		p.next()
		cond := p.parseExpr()
		body := p.parseBlock()
		var elseBody []ast.Stmt
		// Check for else
		// For simplicity, skip whitespace
		for p.tok.Kind == lex.TokenWhitespace {
			p.next()
		}
		if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "else" {
			p.next()
			elseBody = p.parseBlock()
		}
		return &ast.IfStmt{Cond: cond, Then: body, Else: elseBody}
	}

	// Var statement (if kept)
	if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "var" {
		p.next()
		// Expect identifier
		if p.tok.Kind != lex.TokenIdent {
			return nil
		}
		nameTok := p.tok
		p.next()
		// Expect '='
		if p.tok.Kind != lex.TokenOp || p.tok.Value != "=" {
			return nil
		}
		p.next()
		val := p.parseExpr()
		return &ast.VarStmt{Name: nameTok.Value, Value: val}
	}

	// Assignment or Expression Statement
	// Peek ahead or parse expression and check if followed by '='
	// Since we don't have good backtracking, let's parse Left Hand Side expression first.
	// In Rayo, lvalues are usually identifiers, index expressions, or attributes.
	// But `parseExpr` parses a full expression.
	// A simple approach without backtracking: parseExpr(). If next token is '=', treat as assignment target.

	expr := p.parseExpr()
	if expr == nil {
		// Could not parse expression, so not a statement
		return nil
	}

	// Check for assignment
	if p.tok.Kind == lex.TokenOp && p.tok.Value == "=" {
		p.next()
		rhs := p.parseExpr()
		return &ast.AssignStmt{Target: expr, Value: rhs}
	}

	// Otherwise it's an expression statement
	return &ast.ExprStmt{Expr: expr}
}

func (p *Parser) parseExpr() ast.Expr {
	return p.parseComparison()
}

func (p *Parser) parseComparison() ast.Expr {
	expr := p.parseTerm()
	for p.tok.Kind == lex.TokenOp && (p.tok.Value == "<" || p.tok.Value == ">" || p.tok.Value == "==" || p.tok.Value == "!=") {
		op := p.tok.Value
		p.next()
		right := p.parseTerm()
		expr = &ast.BinaryOp{Op: op, Left: expr, Right: right}
	}
	return expr
}

func (p *Parser) parseTerm() ast.Expr {
	expr := p.parsePrimary()
	// Loop for +, - (and maybe string concatenation)
	for p.tok.Kind == lex.TokenOp && (p.tok.Value == "+" || p.tok.Value == "-") {
		op := p.tok.Value
		p.next()
		right := p.parsePrimary()
		expr = &ast.BinaryOp{Op: op, Left: expr, Right: right}
	}
	return expr
}

func (p *Parser) parsePrimary() ast.Expr {
	var expr ast.Expr

	switch p.tok.Kind {
	case lex.TokenNumber:
		tok := p.tok
		p.next()
		i, _ := strconv.Atoi(tok.Value)
		expr = &ast.Literal{Value: i}
	case lex.TokenString:
		tok := p.tok
		val := tok.Value
		// Remove quotes
		if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') {
			val = val[1 : len(val)-1]
		}
		p.next()
		expr = &ast.Literal{Value: val}
	case lex.TokenIdent:
		tok := p.tok
		p.next()
		expr = &ast.Name{Ident: tok.Value}
	case lex.TokenLParen:
		p.next()
		expr = p.parseExpr()
		if p.tok.Kind == lex.TokenRParen {
			p.next()
		}
	default:
		// Try to recover or return error
		// For now return nil, creating invalid AST but preventing panic?
		// Better append error
		// We should not panic.
		// Returning nil causes trouble upstream.
		// Let's create an ErrorExpr or similar? Or just return nil.
		return nil
	}

	// Handle Postfix: Calls, Index, Attributes
	for {
		if p.tok.Kind == lex.TokenLParen {
			// Call
			p.next()
			var args []ast.Expr
			if p.tok.Kind != lex.TokenRParen {
				for {
					arg := p.parseExpr()
					if arg != nil {
						args = append(args, arg)
					}
					if p.tok.Kind == lex.TokenComma {
						p.next()
					} else {
						break
					}
				}
			}
			if p.tok.Kind == lex.TokenRParen {
				p.next()
			}
			expr = &ast.Call{Func: expr, Args: args}
		} else if p.tok.Kind == lex.TokenLBracket {
			// Index
			p.next()
			idx := p.parseExpr()
			if p.tok.Kind == lex.TokenRBracket {
				p.next()
			}
			expr = &ast.Index{Target: expr, Index: idx}
		} else if p.tok.Kind == lex.TokenDot {
			// Attribute or Method Call
			p.next()
			if p.tok.Kind == lex.TokenIdent {
				attrName := p.tok.Value
				p.next()
				expr = &ast.Attr{Target: expr, Attr: attrName}
			}
		} else {
			break
		}
	}

	return expr
}
