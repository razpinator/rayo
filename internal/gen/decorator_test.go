package gen

import (
	"go/format"
	"strings"
	"testing"

	"rayo/internal/ast"
)

func TestEmitSingleDecorator(t *testing.T) {
	fn := &ast.FuncDef{
		Name:       "unit",
		Params:     []*ast.Param{{Name: "x"}},
		Decorators: []ast.Expr{&ast.Name{Ident: "announce"}},
		Body:       []ast.Stmt{&ast.ReturnStmt{Value: &ast.Name{Ident: "x"}}},
	}
	ctx := NewGenContext("main")
	ctx.registerFunc("announce")
	EmitStmt(fn, ctx)
	code := ctx.Code.String()
	if !contains(code, "func unit(x any) any {") {
		t.Errorf("wrapper signature missing: %s", code)
	}
	if !contains(code, "announce(func(x any) any {") {
		t.Errorf("decorator not applied to inner literal: %s", code)
	}
	if !contains(code, "_dec.(func(any) any)(x)") {
		t.Errorf("decorated result not invoked via assertion: %s", code)
	}
}

func TestEmitDecoratorOrder(t *testing.T) {
	// @a @b def f -> a(b(<literal>)); nearest-def (b) applied first.
	fn := &ast.FuncDef{
		Name:       "f",
		Params:     []*ast.Param{{Name: "x"}},
		Decorators: []ast.Expr{&ast.Name{Ident: "a"}, &ast.Name{Ident: "b"}},
		Body:       []ast.Stmt{&ast.ReturnStmt{Value: &ast.Name{Ident: "x"}}},
	}
	ctx := NewGenContext("main")
	ctx.registerFunc("a")
	ctx.registerFunc("b")
	EmitStmt(fn, ctx)
	code := ctx.Code.String()
	// The outer call must be a(...) and the inner b(...).
	ai := strings.Index(code, "a(")
	bi := strings.Index(code, "b(")
	if ai < 0 || bi < 0 || ai > bi {
		t.Errorf("expected a( to wrap b( (a outermost): %s", code)
	}
}

func TestEmitCallDecorator(t *testing.T) {
	fn := &ast.FuncDef{
		Name:       "f",
		Params:     []*ast.Param{{Name: "x"}},
		Decorators: []ast.Expr{&ast.Call{Func: &ast.Name{Ident: "retry"}, Args: []ast.Expr{&ast.Literal{Value: 3}}}},
		Body:       []ast.Stmt{&ast.ReturnStmt{Value: &ast.Name{Ident: "x"}}},
	}
	ctx := NewGenContext("main")
	// `retry` is a top-level decorator factory; its call `retry(3)` returns the
	// actual decorator, which is applied via a callable assertion.
	ctx.registerFunc("retry")
	EmitStmt(fn, ctx)
	code := ctx.Code.String()
	if !contains(code, "retry(3).(func(any) any)(func(x any) any {") {
		t.Errorf("call (factory) decorator not applied via assertion: %s", code)
	}
}

func TestEmitDecoratedFuncIsValidGo(t *testing.T) {
	mod := &ast.Module{
		Body: []ast.Stmt{
			&ast.FuncDef{
				Name:   "announce",
				Params: []*ast.Param{{Name: "fn"}},
				Body:   []ast.Stmt{&ast.ReturnStmt{Value: &ast.Name{Ident: "fn"}}},
			},
			&ast.FuncDef{
				Name:       "unit",
				Params:     []*ast.Param{{Name: "x"}},
				Decorators: []ast.Expr{&ast.Name{Ident: "announce"}},
				Body:       []ast.Stmt{&ast.ReturnStmt{Value: &ast.Name{Ident: "x"}}},
			},
		},
	}
	code := EmitModule(mod, NewGenContext("main"))
	if _, err := format.Source([]byte(code)); err != nil {
		t.Fatalf("generated decorator code is not valid Go: %v\n%s", err, code)
	}
}
