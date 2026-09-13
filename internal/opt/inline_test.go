package opt

import (
	"testing"

	"rayo/internal/ast"
)

// helper: a top-level `def name(params) { return body }`.
func retFunc(name string, params []string, body ast.Expr) *ast.FuncDef {
	ps := make([]*ast.Param, len(params))
	for i, p := range params {
		ps[i] = &ast.Param{Name: p}
	}
	return &ast.FuncDef{Name: name, Params: ps, Body: []ast.Stmt{&ast.ReturnStmt{Value: body}}}
}

func TestInline_SingleReturnCall(t *testing.T) {
	// def add1(x) { return x + 1 }
	// def main() { return add1(41) }   -> return (41 + 1)
	add1 := retFunc("add1", []string{"x"}, &ast.BinaryOp{Op: "+", Left: &ast.Name{Ident: "x"}, Right: &ast.Literal{Value: 1}})
	main := &ast.FuncDef{Name: "run", Body: []ast.Stmt{
		&ast.ReturnStmt{Value: &ast.Call{Func: &ast.Name{Ident: "add1"}, Args: []ast.Expr{&ast.Literal{Value: 41}}}},
	}}
	mod := &ast.Module{Body: []ast.Stmt{add1, main}}
	InlineModule(mod)

	ret := mod.Body[1].(*ast.FuncDef).Body[0].(*ast.ReturnStmt)
	bin, ok := ret.Value.(*ast.BinaryOp)
	if !ok {
		t.Fatalf("call not inlined to a binary op, got %T", ret.Value)
	}
	// After substitution: (41 + 1)
	if l, ok := bin.Left.(*ast.Literal); !ok || l.Value != 41 {
		t.Errorf("argument not substituted for parameter: %#v", bin.Left)
	}
}

func TestInline_ThenFoldProducesConstant(t *testing.T) {
	add1 := retFunc("add1", []string{"x"}, &ast.BinaryOp{Op: "+", Left: &ast.Name{Ident: "x"}, Right: &ast.Literal{Value: 1}})
	main := &ast.FuncDef{Name: "run", Body: []ast.Stmt{
		&ast.ReturnStmt{Value: &ast.Call{Func: &ast.Name{Ident: "add1"}, Args: []ast.Expr{&ast.Literal{Value: 41}}}},
	}}
	mod := &ast.Module{Body: []ast.Stmt{add1, main}}
	Optimize(mod) // fold -> inline -> fold -> dce

	ret := mod.Body[1].(*ast.FuncDef).Body[0].(*ast.ReturnStmt)
	if l, ok := ret.Value.(*ast.Literal); !ok || l.Value != 42 {
		t.Errorf("expected inlined+folded constant 42, got %#v", ret.Value)
	}
}

func TestInline_SkipsRecursive(t *testing.T) {
	// def f(x) { return f(x) }  must not be inlined.
	f := retFunc("f", []string{"x"}, &ast.Call{Func: &ast.Name{Ident: "f"}, Args: []ast.Expr{&ast.Name{Ident: "x"}}})
	main := &ast.FuncDef{Name: "run", Body: []ast.Stmt{
		&ast.ReturnStmt{Value: &ast.Call{Func: &ast.Name{Ident: "f"}, Args: []ast.Expr{&ast.Literal{Value: 1}}}},
	}}
	mod := &ast.Module{Body: []ast.Stmt{f, main}}
	InlineModule(mod)
	ret := mod.Body[1].(*ast.FuncDef).Body[0].(*ast.ReturnStmt)
	if _, ok := ret.Value.(*ast.Call); !ok {
		t.Errorf("recursive function should not be inlined, got %T", ret.Value)
	}
}

func TestInline_SkipsArityMismatch(t *testing.T) {
	add := retFunc("add", []string{"a", "b"}, &ast.BinaryOp{Op: "+", Left: &ast.Name{Ident: "a"}, Right: &ast.Name{Ident: "b"}})
	main := &ast.FuncDef{Name: "run", Body: []ast.Stmt{
		&ast.ReturnStmt{Value: &ast.Call{Func: &ast.Name{Ident: "add"}, Args: []ast.Expr{&ast.Literal{Value: 1}}}},
	}}
	mod := &ast.Module{Body: []ast.Stmt{add, main}}
	InlineModule(mod)
	ret := mod.Body[1].(*ast.FuncDef).Body[0].(*ast.ReturnStmt)
	if _, ok := ret.Value.(*ast.Call); !ok {
		t.Errorf("arity mismatch should not inline, got %T", ret.Value)
	}
}

func TestInline_SkipsDuplicatingSideEffectArg(t *testing.T) {
	// def twice(x) { return x + x }  called with a non-simple arg (a call) must
	// NOT inline, since it would duplicate the side-effecting argument.
	twice := retFunc("twice", []string{"x"}, &ast.BinaryOp{Op: "+", Left: &ast.Name{Ident: "x"}, Right: &ast.Name{Ident: "x"}})
	main := &ast.FuncDef{Name: "run", Body: []ast.Stmt{
		&ast.ReturnStmt{Value: &ast.Call{Func: &ast.Name{Ident: "twice"}, Args: []ast.Expr{
			&ast.Call{Func: &ast.Name{Ident: "sideEffect"}},
		}}},
	}}
	mod := &ast.Module{Body: []ast.Stmt{twice, main}}
	InlineModule(mod)
	ret := mod.Body[1].(*ast.FuncDef).Body[0].(*ast.ReturnStmt)
	if _, ok := ret.Value.(*ast.Call); !ok {
		t.Errorf("must not inline when it would duplicate a side-effecting arg, got %T", ret.Value)
	}
}

func TestInline_AllowsDuplicatingSimpleArg(t *testing.T) {
	// twice(5) is fine to inline: 5 is a literal, safe to duplicate -> (5 + 5).
	twice := retFunc("twice", []string{"x"}, &ast.BinaryOp{Op: "+", Left: &ast.Name{Ident: "x"}, Right: &ast.Name{Ident: "x"}})
	main := &ast.FuncDef{Name: "run", Body: []ast.Stmt{
		&ast.ReturnStmt{Value: &ast.Call{Func: &ast.Name{Ident: "twice"}, Args: []ast.Expr{&ast.Literal{Value: 5}}}},
	}}
	mod := &ast.Module{Body: []ast.Stmt{twice, main}}
	Optimize(mod)
	ret := mod.Body[1].(*ast.FuncDef).Body[0].(*ast.ReturnStmt)
	if l, ok := ret.Value.(*ast.Literal); !ok || l.Value != 10 {
		t.Errorf("expected (5+5) inlined and folded to 10, got %#v", ret.Value)
	}
}
