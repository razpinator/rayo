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
