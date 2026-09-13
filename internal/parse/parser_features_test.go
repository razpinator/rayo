package parse

import (
	"testing"

	"rayo/internal/ast"
)

func mustParse(t *testing.T, src string) *ast.Module {
	t.Helper()
	p := NewParser(src)
	mod := p.ParseModule()
	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parse errors for %q: %v", src, p.Errors())
	}
	return mod
}

func TestParseFuncParamsAndTypes(t *testing.T) {
	mod := mustParse(t, `def add(a: int, b: int) -> int { return a + b }`)
	fn, ok := mod.Body[0].(*ast.FuncDef)
	if !ok {
		t.Fatalf("expected FuncDef, got %T", mod.Body[0])
	}
	if len(fn.Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(fn.Params))
	}
	if fn.Params[0].Name != "a" || fn.Params[1].Name != "b" {
		t.Errorf("param names wrong: %+v", fn.Params)
	}
	if n, ok := fn.Params[0].Type.(*ast.Named); !ok || n.Name != "int" {
		t.Errorf("param type not parsed as int: %#v", fn.Params[0].Type)
	}
	if n, ok := fn.RetType.(*ast.Named); !ok || n.Name != "int" {
		t.Errorf("return type not parsed as int: %#v", fn.RetType)
	}
}

func TestParseGenericTypeParams(t *testing.T) {
	mod := mustParse(t, `def identity[T](x: T) -> T { return x }`)
	fn := mod.Body[0].(*ast.FuncDef)
	if len(fn.TypeParams) != 1 || fn.TypeParams[0] != "T" {
		t.Errorf("type params not parsed: %#v", fn.TypeParams)
	}
	if n, ok := fn.Params[0].Type.(*ast.Named); !ok || n.Name != "T" {
		t.Errorf("param type should be T: %#v", fn.Params[0].Type)
	}
}

func TestParseParameterizedTypes(t *testing.T) {
	mod := mustParse(t, `def f(xs: list[int], m: dict[str, int]) { return xs }`)
	fn := mod.Body[0].(*ast.FuncDef)
	lt, ok := fn.Params[0].Type.(*ast.Named)
	if !ok || lt.Name != "list" || len(lt.Args) != 1 {
		t.Fatalf("list[int] not parsed: %#v", fn.Params[0].Type)
	}
	dt, ok := fn.Params[1].Type.(*ast.Named)
	if !ok || dt.Name != "dict" || len(dt.Args) != 2 {
		t.Errorf("dict[str,int] not parsed: %#v", fn.Params[1].Type)
	}
}

func TestParseElifChain(t *testing.T) {
	mod := mustParse(t, `def f() { if a { return 1 } elif b { return 2 } else { return 3 } }`)
	fn := mod.Body[0].(*ast.FuncDef)
	ifStmt, ok := fn.Body[0].(*ast.IfStmt)
	if !ok {
		t.Fatalf("expected IfStmt, got %T", fn.Body[0])
	}
	if len(ifStmt.Elifs) != 1 {
		t.Errorf("expected 1 elif, got %d", len(ifStmt.Elifs))
	}
	if len(ifStmt.Else) != 1 {
		t.Errorf("expected else branch")
	}
}

func TestParseWhileAndFor(t *testing.T) {
	mod := mustParse(t, `def f() { while cond { step() } for x in items { use(x) } }`)
	fn := mod.Body[0].(*ast.FuncDef)
	if _, ok := fn.Body[0].(*ast.WhileStmt); !ok {
		t.Errorf("expected WhileStmt, got %T", fn.Body[0])
	}
	forStmt, ok := fn.Body[1].(*ast.ForStmt)
	if !ok {
		t.Fatalf("expected ForStmt, got %T", fn.Body[1])
	}
	if forStmt.Var != "x" {
		t.Errorf("for loop var wrong: %q", forStmt.Var)
	}
}

func TestParseTryExceptFinally(t *testing.T) {
	mod := mustParse(t, `def f() { try { risky() } except ValueError as e { handle(e) } finally { done() } }`)
	fn := mod.Body[0].(*ast.FuncDef)
	try, ok := fn.Body[0].(*ast.TryStmt)
	if !ok {
		t.Fatalf("expected TryStmt, got %T", fn.Body[0])
	}
	if len(try.Excepts) != 1 || try.Excepts[0].Var != "e" {
		t.Errorf("except clause not parsed: %#v", try.Excepts)
	}
	if len(try.Finally) != 1 {
		t.Errorf("finally not parsed")
	}
}

func TestParseLiterals(t *testing.T) {
	mod := mustParse(t, `def f() { a = 3.14 b = True c = None d = [1, 2, 3] e = {"k": 1} }`)
	fn := mod.Body[0].(*ast.FuncDef)
	if a := fn.Body[0].(*ast.AssignStmt); a.Value.(*ast.Literal).Value != 3.14 {
		t.Errorf("float literal not parsed: %#v", a.Value)
	}
	if b := fn.Body[1].(*ast.AssignStmt); b.Value.(*ast.Literal).Value != true {
		t.Errorf("bool literal not parsed")
	}
	if c := fn.Body[2].(*ast.AssignStmt); c.Value.(*ast.Literal).Value != nil {
		t.Errorf("None literal not parsed")
	}
	if _, ok := fn.Body[3].(*ast.AssignStmt).Value.(*ast.ListLit); !ok {
		t.Errorf("list literal not parsed")
	}
	if _, ok := fn.Body[4].(*ast.AssignStmt).Value.(*ast.DictLit); !ok {
		t.Errorf("dict literal not parsed")
	}
}

func TestParseLogicalAndArithmeticPrecedence(t *testing.T) {
	mod := mustParse(t, `def f() { return a or b and c }`)
	fn := mod.Body[0].(*ast.FuncDef)
	ret := fn.Body[0].(*ast.ReturnStmt)
	top, ok := ret.Value.(*ast.BinaryOp)
	if !ok || top.Op != "or" {
		t.Fatalf("expected top-level 'or', got %#v", ret.Value)
	}
	// Right side should be `b and c`.
	if r, ok := top.Right.(*ast.BinaryOp); !ok || r.Op != "and" {
		t.Errorf("'and' should bind tighter than 'or': %#v", top.Right)
	}
}

func TestParseImportAlias(t *testing.T) {
	mod := mustParse(t, `import "rayo/stdlib/data" as data`)
	if len(mod.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(mod.Imports))
	}
	if mod.Imports[0].Path != "rayo/stdlib/data" || mod.Imports[0].Alias != "data" {
		t.Errorf("import alias not parsed: %+v", mod.Imports[0])
	}
}
