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

// Import statement. Alias is the local name an import is bound to when the
// source writes `import "path" as name`; it is empty for a plain import.
type Import struct {
	Path  string
	Alias string
	span  diag.Span
}

func (i *Import) Span() diag.Span { return i.span }

// Function definition.
//
// TypeParams holds generic type parameter names (e.g. ["T", "U"] for
// `def name[T, U](...)`); it is empty for non-generic functions. RetType is the
// declared return type from a `-> type` annotation, or nil when omitted (the
// generator then falls back to the dynamic `any` return).
type FuncDef struct {
	Name       string
	TypeParams []string
	Params     []*Param
	RetType    Type
	Body       []Stmt
	// Decorators are `@expr` markers written above the def, outermost first as
	// written in source. Each is a Name (`@log`) or a Call (`@retry(3)`). The
	// generator applies them innermost-first (nearest the def), matching Python.
	Decorators []Expr
	span       diag.Span
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

// ClassDef is a class declaration. It maps to a Go struct plus a constructor
// (from the `__init__` method), receiver methods, and getter/setter methods for
// each property. Base is the optional single base-class name (`class C(Base)`).
type ClassDef struct {
	Name       string
	Base       string
	Fields     []*Field
	Methods    []*FuncDef
	Properties []*Property
	span       diag.Span
}

func (c *ClassDef) Span() diag.Span { return c.span }
func (c *ClassDef) isStmt()         {}

// Field is a typed class attribute (`name: type`).
type Field struct {
	Name string
	Type Type
	span diag.Span
}

func (f *Field) Span() diag.Span { return f.span }

// Property is a computed attribute with a getter and optional setter:
//
//	property full_name {
//	    get { return self.first + self.last }
//	    set(value) { self.first = value }
//	}
//
// Get holds the getter body; Set holds the setter body (empty when read-only);
// SetParam is the setter's value parameter name (default "value").
type Property struct {
	Name     string
	Type     Type
	Get      []Stmt
	Set      []Stmt
	SetParam string
	span     diag.Span
}

func (p *Property) Span() diag.Span { return p.span }

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

// MatchStmt is `match subject { case ... }`. The subject is evaluated once and
// compared against each case in order; the first matching case runs.
type MatchStmt struct {
	Subject Expr
	Cases   []*Case
	span    diag.Span
}

func (s *MatchStmt) Span() diag.Span { return s.span }
func (s *MatchStmt) isStmt()         {}

// Case is one arm of a match. Exactly one of these shapes applies:
//   - literal/expression pattern: Pattern != nil, matched by equality with the
//     subject;
//   - capture: Binding != "" (and not "_"), which always matches and binds the
//     subject to that name inside Body;
//   - wildcard: IsWildcard is true (written `case _`), the default arm.
type Case struct {
	Pattern    Expr
	Binding    string
	IsWildcard bool
	Body       []Stmt
	span       diag.Span
}

func (c *Case) Span() diag.Span { return c.span }

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
//
// The AST's type representation is intentionally lightweight; it mirrors the
// surface syntax of a type annotation so the generator can map it to Go. The
// semantic analyzer has its own richer inference types in internal/sem.

type Type interface{}

// Optional is `Elem?`.
type Optional struct {
	Elem Type
}

// Any is the dynamic `any` type (also the implicit type of unannotated values).
type Any struct{}

// Named is a basic or user/type-parameter-named type such as `int`, `str`, or a
// generic parameter `T`. Args carries type arguments for parameterized types
// like `list[int]` (Name="list", Args=[int]) or `dict[str, int]`.
type Named struct {
	Name string
	Args []Type
}

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
func (c *ClassDef) SetSpan(s diag.Span)    { c.span = s }
func (f *Field) SetSpan(s diag.Span)       { f.span = s }
func (p *Property) SetSpan(s diag.Span)    { p.span = s }
func (s *VarStmt) SetSpan(sp diag.Span)    { s.span = sp }
func (s *AssignStmt) SetSpan(sp diag.Span) { s.span = sp }
func (s *IfStmt) SetSpan(sp diag.Span)     { s.span = sp }
func (e *Elif) SetSpan(s diag.Span)        { e.span = s }
func (s *WhileStmt) SetSpan(sp diag.Span)  { s.span = sp }
func (s *ForStmt) SetSpan(sp diag.Span)    { s.span = sp }
func (s *ReturnStmt) SetSpan(sp diag.Span) { s.span = sp }
func (s *TryStmt) SetSpan(sp diag.Span)    { s.span = sp }
func (e *Except) SetSpan(s diag.Span)      { e.span = s }
func (s *MatchStmt) SetSpan(sp diag.Span)  { s.span = sp }
func (c *Case) SetSpan(s diag.Span)        { c.span = s }
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
