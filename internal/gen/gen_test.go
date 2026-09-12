package gen

import (
	"go/format"
	"rayo/internal/ast"
	"strings"
	"testing"
)

func TestEmitModule(t *testing.T) {
	mod := &ast.Module{
		Name:    "main",
		Imports: []*ast.Import{{Path: "fmt"}},
		Body: []ast.Stmt{
			&ast.VarStmt{Name: "x", Value: &ast.Literal{Value: 42}},
			&ast.ReturnStmt{Value: &ast.Name{Ident: "x"}},
		},
	}
	ctx := NewGenContext("main")
	code := EmitModule(mod, ctx)
	if !contains(code, "var x = 42") || !contains(code, "return x") {
		t.Errorf("codegen failed: %s", code)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestEmitLiteralNumberUnquoted(t *testing.T) {
	if got := emitLiteral(42); got != "42" {
		t.Errorf("int literal: got %q, want 42", got)
	}
	if got := emitLiteral("hi"); got != `"hi"` {
		t.Errorf("string literal: got %q, want quoted", got)
	}
	if got := emitLiteral(true); got != "true" {
		t.Errorf("bool literal: got %q, want true", got)
	}
	if got := emitLiteral(nil); got != "nil" {
		t.Errorf("nil literal: got %q, want nil", got)
	}
}

func TestNewTempVarNoOverflow(t *testing.T) {
	ctx := NewGenContext("main")
	seen := map[string]bool{}
	for i := 0; i < 15; i++ {
		v := ctx.NewTempVar()
		if seen[v] {
			t.Fatalf("duplicate temp var %q", v)
		}
		if !strings.HasPrefix(v, "_tmp") {
			t.Fatalf("bad temp var %q", v)
		}
		seen[v] = true
	}
	if !seen["_tmp10"] {
		t.Errorf("expected _tmp10 among generated temps (overflow regression)")
	}
}

func TestEmitElifChain(t *testing.T) {
	ctx := NewGenContext("main")
	ifStmt := &ast.IfStmt{
		Cond:  &ast.BinaryOp{Op: "==", Left: &ast.Name{Ident: "x"}, Right: &ast.Literal{Value: 1}},
		Then:  []ast.Stmt{&ast.ReturnStmt{Value: &ast.Literal{Value: 1}}},
		Elifs: []*ast.Elif{{Cond: &ast.BinaryOp{Op: "==", Left: &ast.Name{Ident: "x"}, Right: &ast.Literal{Value: 2}}, Body: []ast.Stmt{&ast.ReturnStmt{Value: &ast.Literal{Value: 2}}}}},
		Else:  []ast.Stmt{&ast.ReturnStmt{Value: &ast.Literal{Value: 3}}},
	}
	EmitStmt(ifStmt, ctx)
	code := ctx.Code.String()
	if !contains(code, "} else if (x == 2) {") {
		t.Errorf("elif not lowered to `else if`: %s", code)
	}
	if !contains(code, "} else {") {
		t.Errorf("else branch missing: %s", code)
	}
}

func TestEmitWhileAndFor(t *testing.T) {
	ctx := NewGenContext("main")
	EmitStmt(&ast.WhileStmt{Cond: &ast.Name{Ident: "cond"}, Body: []ast.Stmt{}}, ctx)
	EmitStmt(&ast.ForStmt{Var: "item", Iter: &ast.Name{Ident: "items"}, Body: []ast.Stmt{}}, ctx)
	code := ctx.Code.String()
	if !contains(code, "for cond {") {
		t.Errorf("while not lowered to Go for: %s", code)
	}
	if !contains(code, "for _, item := range items {") {
		t.Errorf("for-in not lowered to range: %s", code)
	}
}

func TestEmitDictAndList(t *testing.T) {
	ctx := NewGenContext("main")
	dict := &ast.DictLit{
		Keys: []ast.Expr{&ast.Literal{Value: "name"}},
		Vals: []ast.Expr{&ast.Literal{Value: "Alice"}},
	}
	if got := LowerDict(dict, ctx); got != `map[string]any{"name": "Alice"}` {
		t.Errorf("dict lowering: got %q", got)
	}
	list := &ast.ListLit{Elems: []ast.Expr{&ast.Literal{Value: 1}, &ast.Literal{Value: 2}}}
	if got := LowerList(list, ctx); got != "[]any{1, 2}" {
		t.Errorf("list lowering: got %q", got)
	}
}

func TestEmitTryRecovers(t *testing.T) {
	try := &ast.TryStmt{
		Body:    []ast.Stmt{&ast.ExprStmt{Expr: &ast.Call{Func: &ast.Name{Ident: "risky"}}}},
		Excepts: []*ast.Except{{Var: "e", Body: []ast.Stmt{&ast.ExprStmt{Expr: &ast.Call{Func: &ast.Name{Ident: "print"}, Args: []ast.Expr{&ast.Name{Ident: "e"}}}}}}},
		Finally: []ast.Stmt{&ast.ExprStmt{Expr: &ast.Call{Func: &ast.Name{Ident: "cleanup"}}}},
	}
	code := LowerTryExcept(try)
	if !contains(code, "recover()") {
		t.Errorf("try/except should use recover(): %s", code)
	}
	if !contains(code, "defer func()") {
		t.Errorf("finally should be emitted via defer: %s", code)
	}
}

func TestEmitLogicalOperators(t *testing.T) {
	ctx := NewGenContext("main")
	expr := &ast.BinaryOp{Op: "and", Left: &ast.Name{Ident: "a"}, Right: &ast.Name{Ident: "b"}}
	if got := emitExpr(expr, ctx); got != "(a && b)" {
		t.Errorf("`and` should lower to &&: got %q", got)
	}
	or := &ast.BinaryOp{Op: "or", Left: &ast.Name{Ident: "a"}, Right: &ast.Name{Ident: "b"}}
	if got := emitExpr(or, ctx); got != "(a || b)" {
		t.Errorf("`or` should lower to ||: got %q", got)
	}
}

func TestEmitFuncWithParamsAndFallthrough(t *testing.T) {
	ctx := NewGenContext("main")
	fn := &ast.FuncDef{
		Name:   "greet",
		Params: []*ast.Param{{Name: "who"}},
		Body:   []ast.Stmt{&ast.ExprStmt{Expr: &ast.Call{Func: &ast.Name{Ident: "print"}, Args: []ast.Expr{&ast.Name{Ident: "who"}}}}},
	}
	EmitStmt(fn, ctx)
	code := ctx.Code.String()
	if !contains(code, "func greet(who any) any {") {
		t.Errorf("params not emitted: %s", code)
	}
	if !contains(code, "return nil") {
		t.Errorf("expected synthetic return nil for fallthrough body: %s", code)
	}
}

func TestModuleFmtImportOnlyWhenPrinting(t *testing.T) {
	// No print -> no fmt import.
	mod := &ast.Module{
		Body: []ast.Stmt{&ast.AssignStmt{Target: &ast.Name{Ident: "x"}, Value: &ast.Literal{Value: 1}}},
	}
	code := EmitModule(mod, NewGenContext("main"))
	if contains(code, `import "fmt"`) {
		t.Errorf("fmt should not be imported when print() is unused: %s", code)
	}
}

func TestEmitModuleIsValidGo(t *testing.T) {
	// A module exercising funcs, control flow, dict/list, and print must emit
	// source that go/format accepts (a proxy for syntactic validity).
	mod := &ast.Module{
		Body: []ast.Stmt{
			&ast.FuncDef{
				Name: "build",
				Body: []ast.Stmt{
					&ast.AssignStmt{Target: &ast.Name{Ident: "d"}, Value: &ast.DictLit{Keys: []ast.Expr{&ast.Literal{Value: "k"}}, Vals: []ast.Expr{&ast.Literal{Value: 1}}}},
					&ast.ReturnStmt{Value: &ast.Name{Ident: "d"}},
				},
			},
			&ast.ExprStmt{Expr: &ast.Call{Func: &ast.Name{Ident: "print"}, Args: []ast.Expr{&ast.Call{Func: &ast.Name{Ident: "build"}}}}},
		},
	}
	code := EmitModule(mod, NewGenContext("main"))
	if _, err := format.Source([]byte(code)); err != nil {
		t.Fatalf("generated code is not valid Go: %v\n%s", err, code)
	}
}
