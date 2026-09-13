package opt

import (
	"testing"

	"rayo/internal/ast"
)

func TestDCE_RemovesUnreachableAfterReturn(t *testing.T) {
	fn := &ast.FuncDef{
		Name: "f",
		Body: []ast.Stmt{
			&ast.ReturnStmt{Value: &ast.Literal{Value: 1}},
			&ast.ExprStmt{Expr: &ast.Call{Func: &ast.Name{Ident: "print"}, Args: []ast.Expr{&ast.Literal{Value: 2}}}},
		},
	}
	mod := &ast.Module{Body: []ast.Stmt{fn}}
	EliminateDeadCode(mod)
	got := mod.Body[0].(*ast.FuncDef)
	if len(got.Body) != 1 {
		t.Errorf("expected unreachable statement after return removed, body len=%d", len(got.Body))
	}
}

func TestDCE_PrunesConstantTrueIf(t *testing.T) {
	// if true { return 1 } else { return 2 }  ->  return 1
	fn := &ast.FuncDef{
		Name: "f",
		Body: []ast.Stmt{
			&ast.IfStmt{
				Cond: &ast.Literal{Value: true},
				Then: []ast.Stmt{&ast.ReturnStmt{Value: &ast.Literal{Value: 1}}},
				Else: []ast.Stmt{&ast.ReturnStmt{Value: &ast.Literal{Value: 2}}},
			},
		},
	}
	mod := &ast.Module{Body: []ast.Stmt{fn}}
	EliminateDeadCode(mod)
	body := mod.Body[0].(*ast.FuncDef).Body
	if len(body) != 1 {
		t.Fatalf("expected single statement after pruning, got %d", len(body))
	}
	ret, ok := body[0].(*ast.ReturnStmt)
	if !ok || ret.Value.(*ast.Literal).Value != 1 {
		t.Errorf("expected `return 1` from constant-true branch, got %#v", body[0])
	}
}

func TestDCE_PrunesConstantFalseIfToElse(t *testing.T) {
	fn := &ast.FuncDef{
		Name: "f",
		Body: []ast.Stmt{
			&ast.IfStmt{
				Cond: &ast.Literal{Value: false},
				Then: []ast.Stmt{&ast.ReturnStmt{Value: &ast.Literal{Value: 1}}},
				Else: []ast.Stmt{&ast.ReturnStmt{Value: &ast.Literal{Value: 2}}},
			},
		},
	}
	mod := &ast.Module{Body: []ast.Stmt{fn}}
	EliminateDeadCode(mod)
	body := mod.Body[0].(*ast.FuncDef).Body
	if len(body) != 1 {
		t.Fatalf("expected single statement, got %d", len(body))
	}
	ret := body[0].(*ast.ReturnStmt)
	if ret.Value.(*ast.Literal).Value != 2 {
		t.Errorf("expected `return 2` from else branch, got %#v", body[0])
	}
}

func TestDCE_DropsConstantFalseElif(t *testing.T) {
	// if x { a } elif false { b } else { c }  keeps x-branch, drops elif.
	fn := &ast.FuncDef{
		Name: "f",
		Body: []ast.Stmt{
			&ast.IfStmt{
				Cond:  &ast.Name{Ident: "x"},
				Then:  []ast.Stmt{&ast.ExprStmt{Expr: &ast.Name{Ident: "a"}}},
				Elifs: []*ast.Elif{{Cond: &ast.Literal{Value: false}, Body: []ast.Stmt{&ast.ExprStmt{Expr: &ast.Name{Ident: "b"}}}}},
				Else:  []ast.Stmt{&ast.ExprStmt{Expr: &ast.Name{Ident: "c"}}},
			},
		},
	}
	mod := &ast.Module{Body: []ast.Stmt{fn}}
	EliminateDeadCode(mod)
	ifStmt := mod.Body[0].(*ast.FuncDef).Body[0].(*ast.IfStmt)
	if len(ifStmt.Elifs) != 0 {
		t.Errorf("expected constantly-false elif dropped, got %d elifs", len(ifStmt.Elifs))
	}
}
