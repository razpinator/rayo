package sem

import (
	"rayo/internal/ast"
	"rayo/internal/diag"
	"testing"
)

type testReporter struct {
	errors []string
}

func (r *testReporter) Report(span diag.Span, msg string) {
	r.errors = append(r.errors, msg)
}

func TestUnusedVarWarning(t *testing.T) {
	mod := &ast.Module{
		Name:    "test",
		Imports: []*ast.Import{},
		Body: []ast.Stmt{
			&ast.VarStmt{Name: "x", Value: &ast.Literal{Value: 1}},
		},
	}
	rep := &testReporter{}
	CheckModule(mod, rep)
	found := false
	for _, err := range rep.errors {
		if err == "unused variable: x" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected unused variable warning")
	}
}

func TestNullSafetyDiagnostics(t *testing.T) {
	// Unsafe dereference of optional value
	expr := &ast.Attr{Target: &ast.Literal{Value: nil}}
	stmt := &ast.ExprStmt{Expr: expr}
	mod := &ast.Module{Body: []ast.Stmt{stmt}}
	rep := &testReporter{}
	CheckModule(mod, rep)
	found := false
	for _, err := range rep.errors {
		if err == "unsafe dereference of optional value" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected null safety diagnostic")
	}
}

func TestIndexNullSafetyDiagnostics(t *testing.T) {
	expr := &ast.Index{Target: &ast.Literal{Value: nil}, Index: &ast.Literal{Value: 0}}
	stmt := &ast.ExprStmt{Expr: expr}
	mod := &ast.Module{Body: []ast.Stmt{stmt}}
	rep := &testReporter{}
	CheckModule(mod, rep)
	found := false
	for _, err := range rep.errors {
		if err == "unsafe index of optional value" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected index null safety diagnostic")
	}
}

func hasMsg(msgs []string, want string) bool {
	for _, m := range msgs {
		if m == want {
			return true
		}
	}
	return false
}

func TestMustReturnMissing(t *testing.T) {
	// def f() { if true { return 1 } }  -- no else, so a path falls through.
	fn := &ast.FuncDef{
		Name: "f",
		Body: []ast.Stmt{
			&ast.IfStmt{
				Cond: &ast.Literal{Value: true},
				Then: []ast.Stmt{&ast.ReturnStmt{Value: &ast.Literal{Value: 1}}},
			},
		},
	}
	mod := &ast.Module{Body: []ast.Stmt{fn}}
	rep := &testReporter{}
	CheckModule(mod, rep)
	if !hasMsg(rep.errors, "missing return: not all paths in 'f' return a value") {
		t.Errorf("expected missing-return diagnostic, got %v", rep.errors)
	}
}

func TestMustReturnSatisfiedByIfElse(t *testing.T) {
	// def f() { if true { return 1 } else { return 2 } } -- all paths return.
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
	rep := &testReporter{}
	CheckModule(mod, rep)
	for _, m := range rep.errors {
		if m == "missing return: not all paths in 'f' return a value" {
			t.Errorf("did not expect missing-return diagnostic, got %v", rep.errors)
		}
	}
}

func TestMustReturnElifCovered(t *testing.T) {
	// All of then/elif/else return -> satisfied.
	fn := &ast.FuncDef{
		Name: "f",
		Body: []ast.Stmt{
			&ast.IfStmt{
				Cond:  &ast.Literal{Value: true},
				Then:  []ast.Stmt{&ast.ReturnStmt{Value: &ast.Literal{Value: 1}}},
				Elifs: []*ast.Elif{{Cond: &ast.Literal{Value: false}, Body: []ast.Stmt{&ast.ReturnStmt{Value: &ast.Literal{Value: 2}}}}},
				Else:  []ast.Stmt{&ast.ReturnStmt{Value: &ast.Literal{Value: 3}}},
			},
		},
	}
	rep := &testReporter{}
	CheckModule(&ast.Module{Body: []ast.Stmt{fn}}, rep)
	if hasMsg(rep.errors, "missing return: not all paths in 'f' return a value") {
		t.Errorf("elif branch should count toward must-return, got %v", rep.errors)
	}

	// Missing elif return -> flagged.
	fn2 := &ast.FuncDef{
		Name: "g",
		Body: []ast.Stmt{
			&ast.IfStmt{
				Cond:  &ast.Literal{Value: true},
				Then:  []ast.Stmt{&ast.ReturnStmt{Value: &ast.Literal{Value: 1}}},
				Elifs: []*ast.Elif{{Cond: &ast.Literal{Value: false}, Body: []ast.Stmt{&ast.ExprStmt{Expr: &ast.Literal{Value: 0}}}}},
				Else:  []ast.Stmt{&ast.ReturnStmt{Value: &ast.Literal{Value: 3}}},
			},
		},
	}
	rep2 := &testReporter{}
	CheckModule(&ast.Module{Body: []ast.Stmt{fn2}}, rep2)
	if !hasMsg(rep2.errors, "missing return: not all paths in 'g' return a value") {
		t.Errorf("expected missing-return when an elif branch falls through, got %v", rep2.errors)
	}
}

func TestRaiseTerminates(t *testing.T) {
	// def f() { if x { return 1 } else { raise ValueError() } } -- raise counts.
	raiseCall := &ast.Call{Func: &ast.Name{Ident: "raise"}, Args: []ast.Expr{&ast.Call{Func: &ast.Name{Ident: "ValueError"}}}}
	fn := &ast.FuncDef{
		Name: "f",
		Body: []ast.Stmt{
			&ast.IfStmt{
				Cond: &ast.Literal{Value: true},
				Then: []ast.Stmt{&ast.ReturnStmt{Value: &ast.Literal{Value: 1}}},
				Else: []ast.Stmt{&ast.ExprStmt{Expr: raiseCall}},
			},
		},
	}
	rep := &testReporter{}
	CheckModule(&ast.Module{Body: []ast.Stmt{fn}}, rep)
	if hasMsg(rep.errors, "missing return: not all paths in 'f' return a value") {
		t.Errorf("raise should satisfy must-return, got %v", rep.errors)
	}
}

func TestVoidFunctionNoMustReturn(t *testing.T) {
	// A function with only bare statements is void and must not be flagged.
	fn := &ast.FuncDef{
		Name: "f",
		Body: []ast.Stmt{&ast.ExprStmt{Expr: &ast.Call{Func: &ast.Name{Ident: "print"}, Args: []ast.Expr{&ast.Literal{Value: "hi"}}}}},
	}
	rep := &testReporter{}
	CheckModule(&ast.Module{Body: []ast.Stmt{fn}}, rep)
	for _, m := range rep.errors {
		if m == "missing return: not all paths in 'f' return a value" {
			t.Errorf("void function should not require a return, got %v", rep.errors)
		}
	}
}

func TestGetReturnsOptionalTriggersNullSafety(t *testing.T) {
	// x = d.get("k"); x.field  -> unsafe dereference of optional.
	get := &ast.Call{Func: &ast.Attr{Target: &ast.Name{Ident: "d"}, Attr: "get"}, Args: []ast.Expr{&ast.Literal{Value: "k"}}}
	mod := &ast.Module{Body: []ast.Stmt{
		&ast.VarStmt{Name: "d", Value: &ast.DictLit{}},
		&ast.AssignStmt{Target: &ast.Name{Ident: "x"}, Value: get},
		&ast.ExprStmt{Expr: &ast.Attr{Target: &ast.Name{Ident: "x"}, Attr: "field"}},
	}}
	rep := &testReporter{}
	CheckModule(mod, rep)
	if !hasMsg(rep.errors, "unsafe dereference of optional value") {
		t.Errorf("expected optional null-safety diagnostic for .get() result, got %v", rep.errors)
	}
}

func TestInferTypePublicAPI(t *testing.T) {
	if TypeString(InferType(&ast.Literal{Value: 5})) != "int" {
		t.Errorf("expected int inference")
	}
	if TypeString(InferType(&ast.Literal{Value: "s"})) != "str" {
		t.Errorf("expected str inference")
	}
	if !IsOptional(InferType(&ast.Literal{Value: nil})) {
		t.Errorf("expected None literal to infer optional")
	}
}
