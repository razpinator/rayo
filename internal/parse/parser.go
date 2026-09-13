package parse

import (
	"fmt"
	"strconv"
	"strings"

	"rayo/internal/ast"
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
	kwTok := p.tok
	p.expect(lex.TokenKeyword) // 'def'

	// Function name
	if p.tok.Kind != lex.TokenIdent {
		err := &ParseError{Msg: "expected function name", Span: TokenSpan(p.tok), Expected: []string{"identifier"}, Excerpt: p.tok.Value}
		p.errors = append(p.errors, err)
		return nil
	}

	name := p.tok.Value
	nameTok := p.tok
	p.next()

	// Optional generic type parameters: def name[T, U](...)
	var typeParams []string
	if p.tok.Kind == lex.TokenLBracket {
		typeParams = p.parseTypeParams()
	}

	// Parameters '(' ... ')'
	if p.tok.Kind != lex.TokenLParen {
		err := &ParseError{Msg: "expected '(' after function name", Span: TokenSpan(p.tok), Expected: []string{"("}, Excerpt: p.tok.Value}
		p.errors = append(p.errors, err)
		return nil
	}
	p.next()
	params := p.parseParams()
	if p.tok.Kind == lex.TokenRParen {
		p.next()
	}

	// Optional return type: -> type
	var retType ast.Type
	if p.tok.Kind == lex.TokenOp && p.tok.Value == "->" {
		p.next()
		retType = p.parseType()
	}

	// Function body '{ ... }'
	body := p.parseBlock()

	fn := &ast.FuncDef{
		Name:       name,
		TypeParams: typeParams,
		Params:     params,
		RetType:    retType,
		Body:       body,
	}
	fn.SetSpan(MergeSpans(TokenSpan(kwTok), TokenSpan(nameTok)))
	return fn
}

// parseDecorated parses one or more `@decorator` lines followed by a `def`, and
// attaches the decorators (outermost first, as written) to the resulting
// FuncDef. A decorator is an expression: a bare name (`@log`) or a call
// (`@retry(3)`).
func (p *Parser) parseDecorated() ast.Stmt {
	var decorators []ast.Expr
	for p.tok.Kind == lex.TokenAt {
		p.next() // consume '@'
		expr := p.parseExpr()
		if expr != nil {
			decorators = append(decorators, expr)
		}
	}
	if !(p.tok.Kind == lex.TokenKeyword && p.tok.Value == "def") {
		p.errors = append(p.errors, &ParseError{Msg: "expected 'def' after decorator", Span: TokenSpan(p.tok), Expected: []string{"def"}, Excerpt: p.tok.Value})
		return nil
	}
	fn, ok := p.parseFuncDef().(*ast.FuncDef)
	if !ok || fn == nil {
		return nil
	}
	fn.Decorators = decorators
	return fn
}

// parseTypeParams parses a generic type-parameter list `[T, U]`. The opening
// '[' is the current token on entry.
func (p *Parser) parseTypeParams() []string {
	p.expect(lex.TokenLBracket)
	var params []string
	for p.tok.Kind != lex.TokenRBracket && p.tok.Kind != lex.TokenEOF {
		if p.tok.Kind == lex.TokenIdent {
			params = append(params, p.tok.Value)
			p.next()
		}
		if p.tok.Kind == lex.TokenComma {
			p.next()
			continue
		}
		if p.tok.Kind != lex.TokenRBracket && p.tok.Kind != lex.TokenIdent {
			// Unexpected token inside the type-param list; break to recover.
			break
		}
	}
	if p.tok.Kind == lex.TokenRBracket {
		p.next()
	}
	return params
}

// parseParams parses a comma-separated parameter list until (but not consuming)
// the closing ')'. Each parameter is `name [: type] [= default]`; the default
// value is parsed and discarded for now (the generator emits dynamic
// signatures), but accepting it keeps annotated signatures parseable.
func (p *Parser) parseParams() []*ast.Param {
	params := []*ast.Param{}
	for p.tok.Kind != lex.TokenRParen && p.tok.Kind != lex.TokenEOF {
		if p.tok.Kind != lex.TokenIdent {
			// Skip stray tokens to avoid an infinite loop.
			p.next()
			continue
		}
		nameTok := p.tok
		param := &ast.Param{Name: nameTok.Value}
		p.next()
		// Optional type annotation.
		if p.tok.Kind == lex.TokenColon {
			p.next()
			param.Type = p.parseType()
		}
		// Optional default value.
		if p.tok.Kind == lex.TokenOp && p.tok.Value == "=" {
			p.next()
			_ = p.parseExpr()
		}
		param.SetSpan(TokenSpan(nameTok))
		params = append(params, param)
		if p.tok.Kind == lex.TokenComma {
			p.next()
		}
	}
	return params
}

// parseType parses a type annotation: a basic/named type, optionally
// parameterized (`list[int]`, `dict[str, int]`) and optionally optional (`T?`).
func (p *Parser) parseType() ast.Type {
	if p.tok.Kind != lex.TokenIdent && p.tok.Kind != lex.TokenKeyword {
		return &ast.Any{}
	}
	name := p.tok.Value
	p.next()
	named := &ast.Named{Name: name}
	// Type arguments: name[T, U]
	if p.tok.Kind == lex.TokenLBracket {
		p.next()
		for p.tok.Kind != lex.TokenRBracket && p.tok.Kind != lex.TokenEOF {
			named.Args = append(named.Args, p.parseType())
			if p.tok.Kind == lex.TokenComma {
				p.next()
			}
		}
		if p.tok.Kind == lex.TokenRBracket {
			p.next()
		}
	}
	var t ast.Type = named
	// Optional marker: T?  (the lexer tokenizes '?' as an error token today, so
	// this is a best-effort check for a following '?' operator token.)
	if p.tok.Kind == lex.TokenOp && p.tok.Value == "?" {
		p.next()
		t = &ast.Optional{Elem: t}
	}
	return t
}

// parseClassDef parses a class declaration:
//
//	class Name [ ( Base ) ] {
//	    field: type
//	    def method(self, ...) { ... }
//	    property name { get { ... } set(v) { ... } }
//	}
func (p *Parser) parseClassDef() ast.Stmt {
	kwTok := p.tok
	p.expect(lex.TokenKeyword) // 'class'

	if p.tok.Kind != lex.TokenIdent {
		p.errors = append(p.errors, &ParseError{Msg: "expected class name", Span: TokenSpan(p.tok), Expected: []string{"identifier"}, Excerpt: p.tok.Value})
		return nil
	}
	cls := &ast.ClassDef{Name: p.tok.Value}
	nameTok := p.tok
	p.next()

	// Optional single base class: class Name(Base)
	if p.tok.Kind == lex.TokenLParen {
		p.next()
		if p.tok.Kind == lex.TokenIdent {
			cls.Base = p.tok.Value
			p.next()
		}
		if p.tok.Kind == lex.TokenRParen {
			p.next()
		}
	}

	p.expect(lex.TokenLBrace)
	for p.tok.Kind != lex.TokenRBrace && p.tok.Kind != lex.TokenEOF {
		switch {
		case p.tok.Kind == lex.TokenKeyword && p.tok.Value == "def":
			if fn, ok := p.parseFuncDef().(*ast.FuncDef); ok && fn != nil {
				cls.Methods = append(cls.Methods, fn)
			}
		case p.isWordOp("property"):
			if prop := p.parseProperty(); prop != nil {
				cls.Properties = append(cls.Properties, prop)
			}
		case p.tok.Kind == lex.TokenIdent:
			// Typed field: name: type
			fieldTok := p.tok
			field := &ast.Field{Name: fieldTok.Value}
			p.next()
			if p.tok.Kind == lex.TokenColon {
				p.next()
				field.Type = p.parseType()
			}
			field.SetSpan(TokenSpan(fieldTok))
			cls.Fields = append(cls.Fields, field)
		default:
			// Skip anything unexpected to avoid an infinite loop.
			p.next()
		}
	}
	if p.tok.Kind == lex.TokenRBrace {
		p.next()
	}
	cls.SetSpan(MergeSpans(TokenSpan(kwTok), TokenSpan(nameTok)))
	return cls
}

// parseProperty parses a property block:
//
//	property name [: type] { get { ... } [set(param) { ... }] }
//
// The `property` word is the current token on entry (it is lexed as an
// identifier, not a reserved keyword).
func (p *Parser) parseProperty() *ast.Property {
	kwTok := p.tok
	p.next() // consume 'property'
	prop := &ast.Property{SetParam: "value"}
	if p.tok.Kind == lex.TokenIdent {
		prop.Name = p.tok.Value
		p.next()
	}
	if p.tok.Kind == lex.TokenColon {
		p.next()
		prop.Type = p.parseType()
	}
	p.expect(lex.TokenLBrace)
	for p.tok.Kind != lex.TokenRBrace && p.tok.Kind != lex.TokenEOF {
		switch {
		case p.isWordOp("get"):
			p.next()
			prop.Get = p.parseBlock()
		case p.isWordOp("set"):
			p.next()
			// Optional `(param)` naming the assigned value.
			if p.tok.Kind == lex.TokenLParen {
				p.next()
				if p.tok.Kind == lex.TokenIdent {
					prop.SetParam = p.tok.Value
					p.next()
				}
				if p.tok.Kind == lex.TokenRParen {
					p.next()
				}
			}
			prop.Set = p.parseBlock()
		default:
			p.next()
		}
	}
	if p.tok.Kind == lex.TokenRBrace {
		p.next()
	}
	prop.SetSpan(TokenSpan(kwTok))
	return prop
}

// parseTry parses `try { ... } (except [Type [as name]] { ... })* [finally { ... }]`.
func (p *Parser) parseTry() ast.Stmt {
	kwTok := p.tok
	p.expect(lex.TokenKeyword) // 'try'
	body := p.parseBlock()

	var excepts []*ast.Except
	for p.tok.Kind == lex.TokenKeyword && p.tok.Value == "except" {
		excKw := TokenSpan(p.tok)
		p.next()
		exc := &ast.Except{}
		// Optional exception type name.
		if p.tok.Kind == lex.TokenIdent {
			exc.Type = &ast.Named{Name: p.tok.Value}
			p.next()
		}
		// Optional `as name` binding.
		if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "as" {
			p.next()
			if p.tok.Kind == lex.TokenIdent {
				exc.Var = p.tok.Value
				p.next()
			}
		}
		exc.Body = p.parseBlock()
		exc.SetSpan(excKw)
		excepts = append(excepts, exc)
	}

	var finally []ast.Stmt
	if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "finally" {
		p.next()
		finally = p.parseBlock()
	}

	try := &ast.TryStmt{Body: body, Excepts: excepts, Finally: finally}
	try.SetSpan(TokenSpan(kwTok))
	return try
}

// parseMatch parses `match subject { case pattern { body } ... }`. A case
// pattern is either a literal/expression (matched by equality), a bare
// identifier (a capture binding), or `_` (the wildcard/default arm).
func (p *Parser) parseMatch() ast.Stmt {
	kwTok := p.tok
	p.expect(lex.TokenKeyword) // 'match'
	subject := p.parseExpr()

	p.expect(lex.TokenLBrace)
	var cases []*ast.Case
	for p.tok.Kind != lex.TokenRBrace && p.tok.Kind != lex.TokenEOF {
		if !(p.tok.Kind == lex.TokenKeyword && p.tok.Value == "case") {
			// Skip stray tokens to avoid an infinite loop.
			p.next()
			continue
		}
		caseKw := TokenSpan(p.tok)
		p.next() // consume 'case'

		cs := &ast.Case{}
		// A bare identifier pattern is a capture binding; `_` is the wildcard.
		if p.tok.Kind == lex.TokenIdent && p.peekIsBlockStart() {
			if p.tok.Value == "_" {
				cs.IsWildcard = true
			} else {
				cs.Binding = p.tok.Value
			}
			p.next()
		} else {
			cs.Pattern = p.parseExpr()
		}
		cs.Body = p.parseBlock()
		cs.SetSpan(caseKw)
		cases = append(cases, cs)
	}
	if p.tok.Kind == lex.TokenRBrace {
		p.next()
	}
	m := &ast.MatchStmt{Subject: subject, Cases: cases}
	m.SetSpan(TokenSpan(kwTok))
	return m
}

// peekIsBlockStart reports whether the token after the current one is a '{',
// which distinguishes a bare-identifier capture pattern (`case x {`) from an
// expression pattern that merely starts with an identifier (`case x + 1 {`).
func (p *Parser) peekIsBlockStart() bool {
	// The lexer is cheap to re-run from the current position via a lookahead
	// lexer seeded with the remaining source. We approximate by scanning the
	// next non-whitespace token from a cloned lexer.
	save := p.lx.Clone()
	for {
		t := save.Next()
		if t.Kind == lex.TokenWhitespace {
			continue
		}
		return t.Kind == lex.TokenLBrace
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
		err := &ParseError{Msg: "unexpected token", Span: TokenSpan(p.tok), Expected: []string{kindToString(kind)}, Excerpt: p.tok.Value}
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
	kwTok := p.tok
	p.expect(lex.TokenKeyword) // 'import'
	pathTok := p.expect(lex.TokenString)
	val := pathTok.Value
	if len(val) >= 2 && (val[0] == '\'' || val[0] == '"') && val[len(val)-1] == val[0] {
		val = val[1 : len(val)-1]
	}
	imp := &ast.Import{Path: val}
	endTok := pathTok
	// Optional alias: import "path" as name
	if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "as" {
		p.next()
		if p.tok.Kind == lex.TokenIdent {
			imp.Alias = p.tok.Value
			endTok = p.tok
			p.next()
		} else {
			err := &ParseError{Msg: "expected identifier after 'as'", Span: TokenSpan(p.tok), Expected: []string{"identifier"}, Excerpt: p.tok.Value}
			p.errors = append(p.errors, err)
		}
	}
	imp.SetSpan(MergeSpans(TokenSpan(kwTok), TokenSpan(endTok)))
	return imp
}

func (p *Parser) parseStmt() ast.Stmt {
	// Skip whitespace
	for p.tok.Kind == lex.TokenWhitespace {
		p.next()
	}

	if p.tok.Kind == lex.TokenEOF || p.tok.Kind == lex.TokenRBrace {
		return nil
	}

	// Decorators: one or more `@decorator` lines preceding a def.
	if p.tok.Kind == lex.TokenAt {
		return p.parseDecorated()
	}

	// Function definition: def name() { ... }
	if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "def" {
		return p.parseFuncDef()
	}

	// Class definition: class Name [ (Base) ] { ... }
	if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "class" {
		return p.parseClassDef()
	}

	// Return statement
	if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "return" {
		kwSpan := TokenSpan(p.tok)
		p.next()
		var val ast.Expr
		// If next token is not newline/semicolon/RBrace, parse expression
		// For now assume if not RBrace, try parse expression
		if p.tok.Kind != lex.TokenRBrace && p.tok.Kind != lex.TokenEOF {
			val = p.parseExpr()
		}
		ret := &ast.ReturnStmt{Value: val}
		if val != nil {
			ret.SetSpan(MergeSpans(kwSpan, val.Span()))
		} else {
			ret.SetSpan(kwSpan)
		}
		return ret
	}

	// If statement, with elif/else chain.
	if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "if" {
		kwSpan := TokenSpan(p.tok)
		p.next()
		cond := p.parseExpr()
		body := p.parseBlock()

		var elifs []*ast.Elif
		var elseBody []ast.Stmt
		for p.tok.Kind == lex.TokenKeyword && p.tok.Value == "elif" {
			elifKw := TokenSpan(p.tok)
			p.next()
			ec := p.parseExpr()
			eb := p.parseBlock()
			elif := &ast.Elif{Cond: ec, Body: eb}
			if ec != nil {
				elif.SetSpan(MergeSpans(elifKw, ec.Span()))
			} else {
				elif.SetSpan(elifKw)
			}
			elifs = append(elifs, elif)
		}
		if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "else" {
			p.next()
			elseBody = p.parseBlock()
		}
		ifStmt := &ast.IfStmt{Cond: cond, Then: body, Elifs: elifs, Else: elseBody}
		if cond != nil {
			ifStmt.SetSpan(MergeSpans(kwSpan, cond.Span()))
		} else {
			ifStmt.SetSpan(kwSpan)
		}
		return ifStmt
	}

	// While loop.
	if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "while" {
		kwSpan := TokenSpan(p.tok)
		p.next()
		cond := p.parseExpr()
		body := p.parseBlock()
		ws := &ast.WhileStmt{Cond: cond, Body: body}
		if cond != nil {
			ws.SetSpan(MergeSpans(kwSpan, cond.Span()))
		} else {
			ws.SetSpan(kwSpan)
		}
		return ws
	}

	// For-in loop: for x in iter { ... }
	if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "for" {
		kwSpan := TokenSpan(p.tok)
		p.next()
		loopVar := ""
		if p.tok.Kind == lex.TokenIdent {
			loopVar = p.tok.Value
			p.next()
		}
		// A second loop variable (`for k, v in ...`) is accepted syntactically;
		// only the first is modeled today.
		if p.tok.Kind == lex.TokenComma {
			p.next()
			if p.tok.Kind == lex.TokenIdent {
				p.next()
			}
		}
		if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "in" {
			p.next()
		}
		iter := p.parseExpr()
		body := p.parseBlock()
		fs := &ast.ForStmt{Var: loopVar, Iter: iter, Body: body}
		fs.SetSpan(kwSpan)
		return fs
	}

	// Try/except/finally.
	if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "try" {
		return p.parseTry()
	}

	// Match / case.
	if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "match" {
		return p.parseMatch()
	}

	// Simple keyword statements: break / continue / pass.
	if p.tok.Kind == lex.TokenKeyword && (p.tok.Value == "break" || p.tok.Value == "continue" || p.tok.Value == "pass") {
		kw := p.tok
		p.next()
		// Model these as expression statements over a Name, matching how flow
		// analysis already recognizes `break` (see internal/sem/flow.go).
		return ast.SetSpan(&ast.ExprStmt{Expr: ast.SetSpan(&ast.Name{Ident: kw.Value}, TokenSpan(kw))}, TokenSpan(kw))
	}

	// Raise statement: raise Expr  (modeled as a call to `raise` so must-return
	// analysis treats it as a terminator).
	if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "raise" {
		kw := p.tok
		p.next()
		arg := p.parseExpr()
		call := &ast.Call{Func: ast.SetSpan(&ast.Name{Ident: "raise"}, TokenSpan(kw))}
		if arg != nil {
			call.Args = []ast.Expr{arg}
		}
		call.SetSpan(TokenSpan(kw))
		return ast.SetSpan(&ast.ExprStmt{Expr: call}, TokenSpan(kw))
	}

	// Var statement (if kept)
	if p.tok.Kind == lex.TokenKeyword && p.tok.Value == "var" {
		kwSpan := TokenSpan(p.tok)
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
		varStmt := &ast.VarStmt{Name: nameTok.Value, Value: val}
		if val != nil {
			varStmt.SetSpan(MergeSpans(kwSpan, val.Span()))
		} else {
			varStmt.SetSpan(MergeSpans(kwSpan, TokenSpan(nameTok)))
		}
		return varStmt
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
		assign := &ast.AssignStmt{Target: expr, Value: rhs}
		if rhs != nil {
			assign.SetSpan(MergeSpans(expr.Span(), rhs.Span()))
		} else {
			assign.SetSpan(expr.Span())
		}
		return assign
	}

	// Otherwise it's an expression statement
	return ast.SetSpan(&ast.ExprStmt{Expr: expr}, expr.Span())
}

func (p *Parser) parseExpr() ast.Expr {
	return p.parseOr()
}

// parseOr handles the lowest-precedence logical `or`.
func (p *Parser) parseOr() ast.Expr {
	expr := p.parseAnd()
	for p.isWordOp("or") || (p.tok.Kind == lex.TokenOp && p.tok.Value == "||") {
		p.next()
		right := p.parseAnd()
		expr = p.binary("or", expr, right)
	}
	return expr
}

// parseAnd handles logical `and`.
func (p *Parser) parseAnd() ast.Expr {
	expr := p.parseComparison()
	for p.isWordOp("and") || (p.tok.Kind == lex.TokenOp && p.tok.Value == "&&") {
		p.next()
		right := p.parseComparison()
		expr = p.binary("and", expr, right)
	}
	return expr
}

func (p *Parser) parseComparison() ast.Expr {
	expr := p.parseTerm()
	for p.tok.Kind == lex.TokenOp && (p.tok.Value == "<" || p.tok.Value == ">" ||
		p.tok.Value == "==" || p.tok.Value == "!=" || p.tok.Value == "<=" || p.tok.Value == ">=") {
		op := p.tok.Value
		p.next()
		right := p.parseTerm()
		expr = p.binary(op, expr, right)
	}
	return expr
}

func (p *Parser) parseTerm() ast.Expr {
	expr := p.parseFactor()
	for p.tok.Kind == lex.TokenOp && (p.tok.Value == "+" || p.tok.Value == "-") {
		op := p.tok.Value
		p.next()
		right := p.parseFactor()
		expr = p.binary(op, expr, right)
	}
	return expr
}

// parseFactor handles `*`, `/`, `%` (and floor-division `//`).
func (p *Parser) parseFactor() ast.Expr {
	expr := p.parseUnary()
	for p.tok.Kind == lex.TokenOp && (p.tok.Value == "*" || p.tok.Value == "/" || p.tok.Value == "%" || p.tok.Value == "//") {
		op := p.tok.Value
		p.next()
		right := p.parseUnary()
		expr = p.binary(op, expr, right)
	}
	return expr
}

// parseUnary handles prefix `not`, unary `-`, and unary `+`.
func (p *Parser) parseUnary() ast.Expr {
	if p.isWordOp("not") || (p.tok.Kind == lex.TokenOp && (p.tok.Value == "-" || p.tok.Value == "+" || p.tok.Value == "!")) {
		opTok := p.tok
		op := opTok.Value
		if op == "!" {
			op = "not"
		}
		p.next()
		operand := p.parseUnary()
		u := &ast.UnaryOp{Op: op, Right: operand}
		if operand != nil {
			u.SetSpan(MergeSpans(TokenSpan(opTok), operand.Span()))
		} else {
			u.SetSpan(TokenSpan(opTok))
		}
		return u
	}
	return p.parsePrimary()
}

// isWordOp reports whether the current token is a word operator (and/or/not/in)
// regardless of whether the lexer classified it as a keyword or identifier.
func (p *Parser) isWordOp(word string) bool {
	return (p.tok.Kind == lex.TokenKeyword || p.tok.Kind == lex.TokenIdent) && p.tok.Value == word
}

// binary builds a BinaryOp node with a merged span.
func (p *Parser) binary(op string, left, right ast.Expr) ast.Expr {
	bin := &ast.BinaryOp{Op: op, Left: left, Right: right}
	if left != nil && right != nil {
		bin.SetSpan(MergeSpans(left.Span(), right.Span()))
	} else if left != nil {
		bin.SetSpan(left.Span())
	}
	return bin
}

func (p *Parser) parsePrimary() ast.Expr {
	var expr ast.Expr

	switch p.tok.Kind {
	case lex.TokenNumber:
		tok := p.tok
		p.next()
		if strings.Contains(tok.Value, ".") {
			f, _ := strconv.ParseFloat(tok.Value, 64)
			expr = ast.SetSpan(&ast.Literal{Value: f}, TokenSpan(tok))
		} else {
			i, _ := strconv.Atoi(tok.Value)
			expr = ast.SetSpan(&ast.Literal{Value: i}, TokenSpan(tok))
		}
	case lex.TokenString:
		tok := p.tok
		val := tok.Value
		// Remove quotes
		if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') {
			val = val[1 : len(val)-1]
		}
		p.next()
		expr = ast.SetSpan(&ast.Literal{Value: val}, TokenSpan(tok))
	case lex.TokenKeyword:
		// Literal keywords: True / False / None, and the lambda form.
		switch p.tok.Value {
		case "True", "False":
			tok := p.tok
			p.next()
			expr = ast.SetSpan(&ast.Literal{Value: tok.Value == "True"}, TokenSpan(tok))
		case "None":
			tok := p.tok
			p.next()
			expr = ast.SetSpan(&ast.Literal{Value: nil}, TokenSpan(tok))
		case "lambda":
			return p.parseLambda()
		default:
			return nil
		}
	case lex.TokenIdent:
		tok := p.tok
		p.next()
		// `func(params) { body }` anonymous function form used in examples.
		if tok.Value == "func" && p.tok.Kind == lex.TokenLParen {
			return p.parseFuncLambda(tok)
		}
		expr = ast.SetSpan(&ast.Name{Ident: tok.Value}, TokenSpan(tok))
	case lex.TokenLBracket:
		expr = p.parseListLit()
	case lex.TokenLBrace:
		expr = p.parseDictLit()
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
			base := expr
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
			closeSpan := TokenSpan(p.tok)
			if p.tok.Kind == lex.TokenRParen {
				p.next()
			}
			call := &ast.Call{Func: base, Args: args}
			if base != nil {
				call.SetSpan(MergeSpans(base.Span(), closeSpan))
			}
			expr = call
		} else if p.tok.Kind == lex.TokenLBracket {
			// Index
			base := expr
			p.next()
			idx := p.parseExpr()
			closeSpan := TokenSpan(p.tok)
			if p.tok.Kind == lex.TokenRBracket {
				p.next()
			}
			ix := &ast.Index{Target: base, Index: idx}
			if base != nil {
				ix.SetSpan(MergeSpans(base.Span(), closeSpan))
			}
			expr = ix
		} else if p.tok.Kind == lex.TokenDot {
			// Attribute or Method Call
			base := expr
			p.next()
			if p.tok.Kind == lex.TokenIdent {
				attrTok := p.tok
				p.next()
				attr := &ast.Attr{Target: base, Attr: attrTok.Value}
				if base != nil {
					attr.SetSpan(MergeSpans(base.Span(), TokenSpan(attrTok)))
				}
				expr = attr
			}
		} else {
			break
		}
	}

	return expr
}

// parseListLit parses `[e1, e2, ...]`. The opening '[' is current.
func (p *Parser) parseListLit() ast.Expr {
	openTok := p.tok
	p.next()
	var elems []ast.Expr
	for p.tok.Kind != lex.TokenRBracket && p.tok.Kind != lex.TokenEOF {
		e := p.parseExpr()
		if e != nil {
			elems = append(elems, e)
		}
		if p.tok.Kind == lex.TokenComma {
			p.next()
			continue
		}
		break
	}
	if p.tok.Kind == lex.TokenRBracket {
		p.next()
	}
	return ast.SetSpan(&ast.ListLit{Elems: elems}, TokenSpan(openTok))
}

// parseDictLit parses `{k1: v1, k2: v2, ...}`. The opening '{' is current.
func (p *Parser) parseDictLit() ast.Expr {
	openTok := p.tok
	p.next()
	var keys, vals []ast.Expr
	for p.tok.Kind != lex.TokenRBrace && p.tok.Kind != lex.TokenEOF {
		k := p.parseExpr()
		if p.tok.Kind == lex.TokenColon {
			p.next()
		}
		v := p.parseExpr()
		if k != nil {
			keys = append(keys, k)
			vals = append(vals, v)
		}
		if p.tok.Kind == lex.TokenComma {
			p.next()
			continue
		}
		break
	}
	if p.tok.Kind == lex.TokenRBrace {
		p.next()
	}
	return ast.SetSpan(&ast.DictLit{Keys: keys, Vals: vals}, TokenSpan(openTok))
}

// parseLambda parses Python-style `lambda a, b: expr`.
func (p *Parser) parseLambda() ast.Expr {
	kwTok := p.tok
	p.expect(lex.TokenKeyword) // 'lambda'
	var params []*ast.Param
	for p.tok.Kind == lex.TokenIdent {
		params = append(params, ast.SetSpan(&ast.Param{Name: p.tok.Value}, TokenSpan(p.tok)))
		p.next()
		if p.tok.Kind == lex.TokenComma {
			p.next()
		}
	}
	if p.tok.Kind == lex.TokenColon {
		p.next()
	}
	body := p.parseExpr()
	return ast.SetSpan(&ast.Lambda{Params: params, Body: body}, TokenSpan(kwTok))
}

// parseFuncLambda parses the `func(params) { return expr }` anonymous-function
// form. The `func` identifier token is already consumed; '(' is current.
func (p *Parser) parseFuncLambda(funcTok lex.Token) ast.Expr {
	p.expect(lex.TokenLParen)
	params := p.parseParams()
	if p.tok.Kind == lex.TokenRParen {
		p.next()
	}
	body := p.parseBlock()
	// Model the body's first `return expr` as the lambda body; fall back to nil.
	var bodyExpr ast.Expr
	for _, st := range body {
		if ret, ok := st.(*ast.ReturnStmt); ok && ret.Value != nil {
			bodyExpr = ret.Value
			break
		}
	}
	if bodyExpr == nil {
		bodyExpr = ast.SetSpan(&ast.Literal{Value: nil}, TokenSpan(funcTok))
	}
	return ast.SetSpan(&ast.Lambda{Params: params, Body: bodyExpr}, TokenSpan(funcTok))
}

// ParseExpr parses a single expression from source (for example REPL :type).
func ParseExpr(src string) (ast.Expr, []error) {
	p := NewParser(src)
	expr := p.parseExpr()
	for p.tok.Kind == lex.TokenWhitespace {
		p.next()
	}
	if p.tok.Kind != lex.TokenEOF {
		p.errors = append(p.errors, fmt.Errorf("unexpected trailing input %q", p.tok.Value))
	}
	if expr == nil && len(p.errors) == 0 {
		p.errors = append(p.errors, fmt.Errorf("expected expression"))
	}
	return expr, p.errors
}
