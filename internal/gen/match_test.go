package gen

import (
	"go/format"
	"testing"

	"rayo/internal/ast"
)

func TestEmitMatchIfElseChain(t *testing.T) {
	m := &ast.MatchStmt{
		Subject: &ast.Name{Ident: "n"},
		Cases: []*ast.Case{
			{Pattern: &ast.Literal{Value: 0}, Body: []ast.Stmt{&ast.ReturnStmt{Value: &ast.Literal{Value: "zero"}}}},
			{Pattern: &ast.Literal{Value: 1}, Body: []ast.Stmt{&ast.ReturnStmt{Value: &ast.Literal{Value: "one"}}}},
			{Binding: "other", Body: []ast.Stmt{&ast.ReturnStmt{Value: &ast.Name{Ident: "other"}}}},
		},
	}
	ctx := NewGenContext("main")
	EmitStmt(m, ctx)
	code := ctx.Code.String()

	if !contains(code, ":= n") {
		t.Errorf("subject not evaluated into a temp: %s", code)
	}
	if !contains(code, "== 0 {") {
		t.Errorf("first literal case not compared: %s", code)
	}
	if !contains(code, "} else if") {
		t.Errorf("second case not an else-if: %s", code)
	}
	if !contains(code, "} else {") {
		t.Errorf("capture default not lowered to else: %s", code)
	}
	if !contains(code, "other :=") {
		t.Errorf("capture binding not emitted: %s", code)
	}
}

func TestEmitMatchWildcardOnly(t *testing.T) {
	// A match whose only arm is a wildcard emits a bare block, not an if.
	m := &ast.MatchStmt{
		Subject: &ast.Name{Ident: "x"},
		Cases: []*ast.Case{
			{IsWildcard: true, Body: []ast.Stmt{&ast.ExprStmt{Expr: &ast.Call{Func: &ast.Name{Ident: "print"}, Args: []ast.Expr{&ast.Literal{Value: "hi"}}}}}},
		},
	}
	ctx := NewGenContext("main")
	EmitStmt(m, ctx)
	code := ctx.Code.String()
	if contains(code, "if ") {
		t.Errorf("wildcard-only match should not emit an if: %s", code)
	}
}

func TestEmitMatchIsValidGo(t *testing.T) {
	mod := &ast.Module{
		Body: []ast.Stmt{
			&ast.FuncDef{
				Name:   "classify",
				Params: []*ast.Param{{Name: "n"}},
				Body: []ast.Stmt{
					&ast.MatchStmt{
						Subject: &ast.Name{Ident: "n"},
						Cases: []*ast.Case{
							{Pattern: &ast.Literal{Value: 0}, Body: []ast.Stmt{&ast.ReturnStmt{Value: &ast.Literal{Value: "zero"}}}},
							{IsWildcard: true, Body: []ast.Stmt{&ast.ReturnStmt{Value: &ast.Literal{Value: "other"}}}},
						},
					},
				},
			},
		},
	}
	code := EmitModule(mod, NewGenContext("main"))
	if _, err := format.Source([]byte(code)); err != nil {
		t.Fatalf("generated match code is not valid Go: %v\n%s", err, code)
	}
}
