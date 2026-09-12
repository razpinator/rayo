package ast

import "rayo/internal/diag"

// Node is the base interface for all AST nodes.
type Node interface {
	Span() diag.Span
}

// Module represents a source file/module.
type Module struct {
	Name    string
	Imports []*Import
	Body    []Stmt
	span    diag.Span
}

func (m *Module) Span() diag.Span { return m.span }

// Import statement.
type Import struct {
	Path string
	span diag.Span
}

func (i *Import) Span() diag.Span { return i.span }

// Function definition.
type FuncDef struct {
	Name   string
	Params []*Param
	Body   []Stmt
	span   diag.Span
}

func (f *FuncDef) Span() diag.Span { return f.span }
func (f *FuncDef) isStmt()         {}

// Parameter.
type Param struct {
	Name string
	Type Type
	span diag.Span
}

func (p *Param) Span() diag.Span { return p.span }

// Statement base interface.
type Stmt interface {
	Node
	isStmt()
}

// Expression base interface.
type Expr interface {
	Node
	isExpr()
}

// Statement types

type VarStmt struct {
	Name  string
	Value Expr
	span  diag.Span
}

func (s *VarStmt) Span() diag.Span { return s.span }
func (s *VarStmt) isStmt()         {}

type AssignStmt struct {
	Target Expr
	Value  Expr
	span   diag.Span
}

func (s *AssignStmt) Span() diag.Span { return s.span }
func (s *AssignStmt) isStmt()         {}

type IfStmt struct {
	Cond  Expr
	Then  []Stmt
	Elifs []*Elif
	Else  []Stmt
	span  diag.Span
}

func (s *IfStmt) Span() diag.Span { return s.span }
func (s *IfStmt) isStmt()         {}

type Elif struct {
	Cond Expr
	Body []Stmt
	span diag.Span
}

func (e *Elif) Span() diag.Span { return e.span }
func (e *Elif) isStmt()         {}

type WhileStmt struct {
	Cond Expr
	Body []Stmt
	span diag.Span
}

func (s *WhileStmt) Span() diag.Span { return s.span }
func (s *WhileStmt) isStmt()         {}

type ForStmt struct {
	Var  string
	Iter Expr
	Body []Stmt
	span diag.Span
}

func (s *ForStmt) Span() diag.Span { return s.span }
func (s *ForStmt) isStmt()         {}

type ReturnStmt struct {
	Value Expr
	span  diag.Span
}

func (s *ReturnStmt) Span() diag.Span { return s.span }
func (s *ReturnStmt) isStmt()         {}

type TryStmt struct {
	Body    []Stmt
	Excepts []*Except
	Finally []Stmt
	span    diag.Span
}

func (s *TryStmt) Span() diag.Span { return s.span }
func (s *TryStmt) isStmt()         {}

type Except struct {
	Type Type
	Var  string
	Body []Stmt
	span diag.Span
}

func (e *Except) Span() diag.Span { return e.span }
func (e *Except) isStmt()         {}

type ExprStmt struct {
	Expr Expr
	span diag.Span
}

func (s *ExprStmt) Span() diag.Span { return s.span }
func (s *ExprStmt) isStmt()         {}

// Expression types

type Literal struct {
	Value any
	span  diag.Span
}

func (e *Literal) Span() diag.Span { return e.span }
func (e *Literal) isExpr()         {}

type Name struct {
	Ident string
	span  diag.Span
}

func (e *Name) Span() diag.Span { return e.span }
func (e *Name) isExpr()         {}

type Call struct {
	Func Expr
	Args []Expr
	span diag.Span
}

func (e *Call) Span() diag.Span { return e.span }
func (e *Call) isExpr()         {}

type Index struct {
	Target Expr
	Index  Expr
	span   diag.Span
}

func (e *Index) Span() diag.Span { return e.span }
func (e *Index) isExpr()         {}

type Attr struct {
	Target Expr
	Attr   string
	span   diag.Span
}

func (e *Attr) Span() diag.Span { return e.span }
func (e *Attr) isExpr()         {}

type UnaryOp struct {
	Op    string
	Right Expr
	span  diag.Span
}

func (e *UnaryOp) Span() diag.Span { return e.span }
func (e *UnaryOp) isExpr()         {}

type BinaryOp struct {
	Op    string
	Left  Expr
	Right Expr
	span  diag.Span
}

func (e *BinaryOp) Span() diag.Span { return e.span }
func (e *BinaryOp) isExpr()         {}

type DictLit struct {
	Keys []Expr
	Vals []Expr
	span diag.Span
}

func (e *DictLit) Span() diag.Span { return e.span }
func (e *DictLit) isExpr()         {}

type ListLit struct {
	Elems []Expr
	span  diag.Span
}

func (e *ListLit) Span() diag.Span { return e.span }
func (e *ListLit) isExpr()         {}

type Lambda struct {
	Params []*Param
	Body   Expr
	span   diag.Span
}

func (e *Lambda) Span() diag.Span { return e.span }
func (e *Lambda) isExpr()         {}

// Types

type Type interface{}

type Optional struct {
	Elem Type
}

type Any struct{}

// Constructors for common nodes (examples)
func NewName(ident string, span diag.Span) *Name {
	return &Name{Ident: ident, span: span}
}
func NewLiteral(val any, span diag.Span) *Literal {
	return &Literal{Value: val, span: span}
}

// SpanSetter is implemented by AST nodes that can have their source span
// assigned after construction. The parser uses this to attach spans uniformly.
type SpanSetter interface {
	Node
	SetSpan(diag.Span)
}

func (m *Module) SetSpan(s diag.Span)      { m.span = s }
func (i *Import) SetSpan(s diag.Span)      { i.span = s }
func (f *FuncDef) SetSpan(s diag.Span)     { f.span = s }
func (p *Param) SetSpan(s diag.Span)       { p.span = s }
func (s *VarStmt) SetSpan(sp diag.Span)    { s.span = sp }
func (s *AssignStmt) SetSpan(sp diag.Span) { s.span = sp }
func (s *IfStmt) SetSpan(sp diag.Span)     { s.span = sp }
func (e *Elif) SetSpan(s diag.Span)        { e.span = s }
func (s *WhileStmt) SetSpan(sp diag.Span)  { s.span = sp }
func (s *ForStmt) SetSpan(sp diag.Span)    { s.span = sp }
func (s *ReturnStmt) SetSpan(sp diag.Span) { s.span = sp }
func (s *TryStmt) SetSpan(sp diag.Span)    { s.span = sp }
func (e *Except) SetSpan(s diag.Span)      { e.span = s }
func (s *ExprStmt) SetSpan(sp diag.Span)   { s.span = sp }
func (e *Literal) SetSpan(s diag.Span)     { e.span = s }
func (e *Name) SetSpan(s diag.Span)        { e.span = s }
func (e *Call) SetSpan(s diag.Span)        { e.span = s }
func (e *Index) SetSpan(s diag.Span)       { e.span = s }
func (e *Attr) SetSpan(s diag.Span)        { e.span = s }
func (e *UnaryOp) SetSpan(s diag.Span)     { e.span = s }
func (e *BinaryOp) SetSpan(s diag.Span)    { e.span = s }
func (e *DictLit) SetSpan(s diag.Span)     { e.span = s }
func (e *ListLit) SetSpan(s diag.Span)     { e.span = s }
func (e *Lambda) SetSpan(s diag.Span)      { e.span = s }

// SetSpan assigns a span to n if n supports it, returning n unchanged. It is a
// convenience for the parser so callers can write `ast.SetSpan(node, span)`
// inline regardless of the concrete type.
func SetSpan[T Node](n T, s diag.Span) T {
	if ss, ok := any(n).(SpanSetter); ok {
		ss.SetSpan(s)
	}
	return n
}
